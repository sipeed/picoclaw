package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// pdfFollowUpWait is how long to wait for a follow-up message after
// receiving a PDF file without accompanying text. During this window
// the user can send OCR keywords like "figures" / "図版".
const pdfFollowUpWait = 5 * time.Second

// pdfFollowUpHint is sent when a PDF arrives without text.
const pdfFollowUpHint = "PDF received. You can send OCR options " +
	"(e.g. \"figures\" / \"\u56f3\u7248\") within a few seconds, or processing will start automatically."

// pdfCancelKeywords triggers OCR cancellation when found in a message
// received during Phase 2 (OCR in progress).
var pdfCancelKeywords = []string{
	"cancel", "abort", "stop",
	"\u4e2d\u6b62", "\u30ad\u30e3\u30f3\u30bb\u30eb", "\u3084\u3081",
}

// isCancelKeyword returns true if content contains a cancel keyword.
func isCancelKeyword(content string) bool {
	lower := strings.ToLower(content)
	for _, kw := range pdfCancelKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// ── Phase 1: Pre-OCR keyword wait ──────────────────────────────────────

// waitForPDFFollowUp checks whether msg contains a bare PDF file (no text).
// If so, it sends a hint and waits up to pdfFollowUpWait for a follow-up
// message that may contain OCR keywords (e.g. "figures"). The follow-up
// text is merged into msg.Content so that wantFigures() picks it up.
//
// If the message already has text (Caption) or is not a file, it returns
// the message unchanged.
func (al *AgentLoop) waitForPDFFollowUp(
	ctx context.Context, msg bus.InboundMessage, queue <-chan bus.InboundMessage,
) (bus.InboundMessage, []bus.InboundMessage) {
	if !messageHasBareFile(msg) {
		return msg, nil
	}

	// Send hint so the user knows they can add instructions
	if al.bus != nil && msg.Channel != "" && msg.ChatID != "" {
		_ = al.bus.PublishOutbound(ctx, bus.OutboundMessage{
			Channel:         msg.Channel,
			ChatID:          msg.ChatID,
			Content:         pdfFollowUpHint,
			SkipPlaceholder: true,
		})
	}

	logger.DebugCF("agent", "Phase 1: waiting for PDF follow-up keywords", map[string]any{
		"chat_id": msg.ChatID,
		"wait":    pdfFollowUpWait.String(),
	})

	timer := time.NewTimer(pdfFollowUpWait)
	defer timer.Stop()

	// Collect messages that arrived for other chats during the wait;
	// they must not be lost.
	var overflow []bus.InboundMessage

	for {
		select {
		case <-ctx.Done():
			return msg, overflow
		case <-timer.C:
			return msg, overflow
		case followUp, ok := <-queue:
			if !ok {
				return msg, overflow
			}

			// Different chat or slash command → don't absorb
			if followUp.ChatID != msg.ChatID ||
				strings.HasPrefix(strings.TrimSpace(followUp.Content), "/") {
				overflow = append(overflow, followUp)
				continue
			}

			followUpText := strings.TrimSpace(followUp.Content)
			if followUpText == "" {
				continue
			}

			// Merge follow-up text into the PDF message so wantFigures()
			// and the LLM both see it.
			msg.Content = msg.Content + "\n" + followUpText

			logger.InfoCF("agent", "Phase 1: merged PDF follow-up keywords", map[string]any{
				"chat_id":   msg.ChatID,
				"follow_up": followUpText,
			})

			return msg, overflow
		}
	}
}

// ── Phase 2: Buffer during OCR ─────────────────────────────────────────

// processResult holds the return values of processMessage.
type processResult struct {
	response string
	err      error
}

// processPDFWithBuffering runs processMessage (which triggers OCR) in a
// goroutine while draining the llmQueue for the same chat. Messages
// arriving during OCR are buffered and returned so the caller can
// merge them into a follow-up LLM turn.
//
// If a cancel keyword is detected, the context is canceled (killing the
// OCR process) and the cancel message is returned as the response.
func (al *AgentLoop) processPDFWithBuffering(
	ctx context.Context,
	msg bus.InboundMessage,
	queue <-chan bus.InboundMessage,
	overflow []bus.InboundMessage,
) (response string, err error, buffered []bus.InboundMessage) {
	// Run processMessage concurrently so we can drain the queue.
	ocrCtx, ocrCancel := context.WithCancel(ctx)
	defer ocrCancel()

	resultCh := make(chan processResult, 1)
	go func() {
		resp, e := al.processMessage(ocrCtx, msg)
		resultCh <- processResult{resp, e}
	}()

	// Notify the user that messages will be collected.
	if al.bus != nil && msg.Channel != "" && msg.ChatID != "" {
		_ = al.bus.PublishOutbound(ctx, bus.OutboundMessage{
			Channel:         msg.Channel,
			ChatID:          msg.ChatID,
			Content:         "OCR starting. You can send instructions — they will be included after processing. Send \"cancel\" to abort.",
			SkipPlaceholder: true,
			IsStatus:        true,
		})
	}

	// Check overflow messages from Phase 1 first (they may contain cancel
	// or instructions for same chat).
	for _, o := range overflow {
		if o.ChatID == msg.ChatID &&
			!strings.HasPrefix(strings.TrimSpace(o.Content), "/") {
			text := strings.TrimSpace(o.Content)
			if isCancelKeyword(text) {
				ocrCancel()
				<-resultCh // wait for processMessage to finish
				return "PDF processing canceled.", nil, nil
			}
			buffered = append(buffered, o)
		} else {
			buffered = append(buffered, o)
		}
	}

	// Drain queue while OCR is running.
	for {
		select {
		case result := <-resultCh:
			// processMessage finished (OCR done + LLM responded)
			return result.response, result.err, buffered

		case followUp, ok := <-queue:
			if !ok {
				result := <-resultCh
				return result.response, result.err, buffered
			}

			// Same chat, not a command → buffer or cancel
			if followUp.ChatID == msg.ChatID &&
				!strings.HasPrefix(strings.TrimSpace(followUp.Content), "/") {
				text := strings.TrimSpace(followUp.Content)
				if isCancelKeyword(text) {
					ocrCancel()
					<-resultCh
					return "PDF processing canceled.", nil, nil
				}
				if text != "" {
					buffered = append(buffered, followUp)
					logger.InfoCF("agent", "Phase 2: buffered message during OCR", map[string]any{
						"chat_id": msg.ChatID,
						"content": text,
					})
				}
			} else {
				// Different chat or command → keep for later processing
				buffered = append(buffered, followUp)
			}
		}
	}
}

// ── Helpers ────────────────────────────────────────────────────────────

// messageHasBareFile returns true if the message contains a [file] tag
// (indicating a document attachment) but has no meaningful user text
// alongside it. A message like "[file]" or "\n[file]" is considered bare.
func messageHasBareFile(msg bus.InboundMessage) bool {
	content := msg.Content
	if !strings.Contains(content, "[file]") {
		return false
	}
	// Strip all media tags to see if there's any remaining user text
	stripped := content
	for _, tag := range []string{"[file]", "[image: photo]", "[voice]", "[audio]"} {
		stripped = strings.ReplaceAll(stripped, tag, "")
	}
	stripped = strings.TrimSpace(stripped)
	return stripped == ""
}

// mergeBufferedMessages creates a combined content string from buffered
// messages for use as a follow-up user message after OCR.
func mergeBufferedMessages(buffered []bus.InboundMessage, chatID string) string {
	var parts []string
	for _, b := range buffered {
		if b.ChatID == chatID {
			text := strings.TrimSpace(b.Content)
			if text != "" {
				parts = append(parts, text)
			}
		}
	}
	return strings.Join(parts, "\n")
}

// extractNonChatMessages returns messages not belonging to the given chatID.
func extractNonChatMessages(buffered []bus.InboundMessage, chatID string) []bus.InboundMessage {
	var result []bus.InboundMessage
	for _, b := range buffered {
		if b.ChatID != chatID {
			result = append(result, b)
		}
	}
	return result
}

// formatBufferedNotice creates a status message listing how many messages
// were collected during OCR processing.
func formatBufferedNotice(count int) string {
	if count == 0 {
		return ""
	}
	if count == 1 {
		return "(1 message received during processing — included below)"
	}
	return fmt.Sprintf("(%d messages received during processing — included below)", count)
}
