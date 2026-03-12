package agent

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
	"github.com/sipeed/picoclaw/pkg/utils"
)

func (al *AgentLoop) handleReasoning(ctx context.Context, reasoningContent, channelName, channelID string) {
	if reasoningContent == "" || channelName == "" || channelID == "" {
		return
	}

	// Check context cancellation before attempting to publish,

	// since PublishOutbound's select may race between send and ctx.Done().

	if ctx.Err() != nil {
		return
	}

	// Use a short timeout so the goroutine does not block indefinitely when

	// the outbound bus is full.  Reasoning output is best-effort; dropping it

	// is acceptable to avoid goroutine accumulation.

	pubCtx, pubCancel := context.WithTimeout(ctx, 5*time.Second)

	defer pubCancel()

	if err := al.bus.PublishOutbound(pubCtx, bus.OutboundMessage{
		Channel: channelName,

		ChatID: channelID,

		Content: reasoningContent,
	}); err != nil {
		// Treat context.DeadlineExceeded / context.Canceled as expected

		// (bus full under load, or parent canceled).  Check the error

		// itself rather than ctx.Err(), because pubCtx may time out

		// (5 s) while the parent ctx is still active.

		// Also treat ErrBusClosed as expected — it occurs during normal

		// shutdown when the bus is closed before all goroutines finish.

		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) ||

			errors.Is(err, bus.ErrBusClosed) {
			logger.DebugCF("agent", "Reasoning publish skipped (timeout/cancel)", map[string]any{
				"channel": channelName,

				"error": err.Error(),
			})
		} else {
			logger.WarnCF("agent", "Failed to publish reasoning (best-effort)", map[string]any{
				"channel": channelName,

				"error": err.Error(),
			})
		}
	}
}

// streamingReasoningLines is the number of lines reserved for reasoning

// in the streaming display.  The remaining lines go to content.

const streamingReasoningLines = 6

// buildStreamingDisplay builds a fixed-height status bubble for streaming.

//

// Layout when reasoning is active (reasoning only or both):

//

//	🧠 Thinking...

//	━━━━━━━━━━

//	<reasoning tail — streamingReasoningLines lines>

//	━━━━━━━━━━

//	<content tail — remaining lines>  (or blank if content is empty)

//	█

//

// Layout when no reasoning (content only):

//

//	<content tail — streamingDisplayLines lines>

//	█

func buildStreamingDisplay(content, reasoning string) string {
	if reasoning == "" {
		// No reasoning — full window for content.

		return utils.TailPad(content, streamingDisplayLines, maxEntryLineWidth) + " \u2589"
	}

	var sb strings.Builder

	// Header

	if content == "" {
		sb.WriteString("\U0001f9e0 Thinking...\n")
	} else {
		sb.WriteString("\U0001f9e0 Thought, now responding...\n")
	}

	sb.WriteString(statusSeparator)

	// Reasoning window

	headerLines := 2 // header + separator

	footerLines := 1 // separator before content

	contentLines := streamingDisplayLines - headerLines - footerLines - streamingReasoningLines

	if contentLines < 3 {
		contentLines = 3
	}

	rLines := streamingDisplayLines - headerLines - footerLines - contentLines

	sb.WriteString(utils.TailPad(reasoning, rLines, maxEntryLineWidth))

	sb.WriteByte('\n')

	sb.WriteString(statusSeparator)

	// Content window (may be blank padding if content hasn't started)

	sb.WriteString(utils.TailPad(content, contentLines, maxEntryLineWidth))

	sb.WriteString(" \u2589")

	return sb.String()
}

// consumeStreamWithRepetitionDetection reads StreamEvents from ch, accumulates

// content and tool calls, and runs repetition detection every checkInterval runes.

// If repetition is detected, cancelFn is called to abort the HTTP request and

// the function returns the partial response with detected=true.

func consumeStreamWithRepetitionDetection(
	ch <-chan protocoltypes.StreamEvent,

	cancelFn context.CancelFunc,

	checkInterval int,

	onChunk func(content, reasoning string),
) (*providers.LLMResponse, bool, error) {
	var content strings.Builder

	var reasoning strings.Builder

	var toolCalls []streamToolCallAcc

	var finishReason string

	var usage *providers.UsageInfo

	runesSinceLastCheck := 0

	for ev := range ch {
		if ev.Err != nil {
			return nil, false, ev.Err
		}

		updated := false

		if ev.ContentDelta != "" {
			content.WriteString(ev.ContentDelta)

			runesSinceLastCheck += utf8.RuneCountInString(ev.ContentDelta)

			updated = true
		}

		if ev.ReasoningDelta != "" {
			reasoning.WriteString(ev.ReasoningDelta)

			updated = true
		}

		if updated && onChunk != nil {
			onChunk(content.String(), reasoning.String())
		}

		if ev.FinishReason != "" {
			finishReason = ev.FinishReason
		}

		if ev.Usage != nil {
			usage = ev.Usage
		}

		for _, tc := range ev.ToolCallDeltas {
			for len(toolCalls) <= tc.Index {
				toolCalls = append(toolCalls, streamToolCallAcc{})
			}

			if tc.ID != "" {
				toolCalls[tc.Index].id = tc.ID
			}

			if tc.Name != "" {
				toolCalls[tc.Index].name = tc.Name
			}

			toolCalls[tc.Index].args.WriteString(tc.ArgumentsDelta)
		}

		// Run repetition detection periodically on accumulated content.

		if runesSinceLastCheck >= checkInterval && content.Len() > 2000 {
			runesSinceLastCheck = 0

			if utils.DetectRepetitionLoop(content.String()) {
				cancelFn()

				// Drain remaining events so the producer goroutine can exit.

				for range ch {
				}

				resp := buildAccumulatedResponse(content.String(), reasoning.String(), toolCalls, finishReason, usage)

				return resp, true, nil
			}
		}
	}

	resp := buildAccumulatedResponse(content.String(), reasoning.String(), toolCalls, finishReason, usage)

	return resp, false, nil
}

// streamToolCallAcc accumulates streamed tool call fragments.

type streamToolCallAcc struct {
	id string

	name string

	args strings.Builder
}

// buildAccumulatedResponse constructs an LLMResponse from accumulated stream data.

func buildAccumulatedResponse(
	content, reasoning string,

	toolCalls []streamToolCallAcc,

	finishReason string,

	usage *providers.UsageInfo,
) *providers.LLMResponse {
	resp := &providers.LLMResponse{
		Content: content,

		Reasoning: reasoning,

		FinishReason: finishReason,

		Usage: usage,
	}

	for _, tc := range toolCalls {
		arguments := make(map[string]any)

		argStr := tc.args.String()

		if argStr != "" {
			if err := json.Unmarshal([]byte(argStr), &arguments); err != nil {
				arguments["raw"] = argStr
			}
		}

		resp.ToolCalls = append(resp.ToolCalls, providers.ToolCall{
			ID: tc.id,

			Name: tc.name,

			Arguments: arguments,
		})
	}

	return resp
}
