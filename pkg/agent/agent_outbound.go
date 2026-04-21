// PicoClaw - Ultra-lightweight personal AI agent

package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/logger"
)

func (al *AgentLoop) maybePublishError(ctx context.Context, channel, chatID, sessionKey string, err error) bool {
	if errors.Is(err, context.Canceled) {
		return false
	}
	al.PublishResponseIfNeeded(ctx, channel, chatID, sessionKey, fmt.Sprintf("Error processing message: %v", err))
	return true
}

func (al *AgentLoop) publishResponseOrError(
	ctx context.Context,
	channel, chatID, sessionKey string,
	response string,
	err error,
) {
	if err != nil {
		if !al.maybePublishError(ctx, channel, chatID, sessionKey, err) {
			return
		}
		response = ""
	}
	al.PublishResponseIfNeeded(ctx, channel, chatID, sessionKey, response)
}

func (al *AgentLoop) PublishResponseIfNeeded(ctx context.Context, channel, chatID, sessionKey, response string) {
	al.publishResponseWithContextIfNeeded(ctx, nil, channel, chatID, sessionKey, response)
}

func (al *AgentLoop) publishResponseWithContextIfNeeded(
	ctx context.Context,
	inboundCtx *bus.InboundContext,
	channel, chatID, sessionKey, response string,
) {
	if response == "" {
		return
	}

	alreadySentToSameChat := false
	defaultAgent := al.GetRegistry().GetDefaultAgent()
	if defaultAgent != nil {
		alreadySentToSameChat = anySentTrackingToolSentTo(defaultAgent, sessionKey, channel, chatID)
	}

	if alreadySentToSameChat {
		logger.DebugCF(
			"agent",
			"Skipped outbound (message tool already sent to same chat)",
			map[string]any{"channel": channel, "chat_id": chatID},
		)
		return
	}

	outboundCtx := bus.NewOutboundContext(channel, chatID, "")
	if inboundCtx != nil {
		outboundCtx = outboundContextFromInbound(inboundCtx, channel, chatID, "")
	}
	attachPlanTodoFallback(&outboundCtx, response)

	msg := bus.OutboundMessage{
		Context: outboundCtx,
		Content: response,
	}
	if sessionKey != "" {
		msg.ContextUsage = computeContextUsage(al.agentForSession(sessionKey), sessionKey)
	}
	al.bus.PublishOutbound(ctx, msg)
	logger.InfoCF("agent", "Published outbound response",
		map[string]any{
			"channel":        channel,
			"chat_id":        chatID,
			"content_len":    len(response),
			"has_structured": strings.TrimSpace(outboundCtx.Raw[metadataKeyStructuredData]) != "",
		})
}

func (al *AgentLoop) targetReasoningChannelID(channelName string) (chatID string) {
	if al.channelManager == nil {
		return ""
	}
	if ch, ok := al.channelManager.GetChannel(channelName); ok {
		return ch.ReasoningChannelID()
	}
	return ""
}

func (al *AgentLoop) publishPicoReasoning(ctx context.Context, reasoningContent, chatID string) {
	if reasoningContent == "" || chatID == "" {
		return
	}

	if ctx.Err() != nil {
		return
	}

	pubCtx, pubCancel := context.WithTimeout(ctx, 5*time.Second)
	defer pubCancel()

	if err := al.bus.PublishOutbound(pubCtx, bus.OutboundMessage{
		Context: bus.InboundContext{
			Channel: "pico",
			ChatID:  chatID,
			Raw: map[string]string{
				metadataKeyMessageKind: messageKindThought,
			},
		},
		Content: reasoningContent,
	}); err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) ||
			errors.Is(err, bus.ErrBusClosed) {
			logger.DebugCF("agent", "Pico reasoning publish skipped (timeout/cancel)", map[string]any{
				"channel": "pico",
				"error":   err.Error(),
			})
		} else {
			logger.WarnCF("agent", "Failed to publish pico reasoning (best-effort)", map[string]any{
				"channel": "pico",
				"error":   err.Error(),
			})
		}
	}
}

func (al *AgentLoop) publishPicoStructured(ctx context.Context, chatID, content string, structured any) {
	if chatID == "" || structured == nil {
		return
	}
	if ctx.Err() != nil {
		return
	}

	rawStructured, err := json.Marshal(structured)
	if err != nil {
		logger.WarnCF("agent", "Failed to encode pico structured payload", map[string]any{
			"channel": "pico",
			"error":   err.Error(),
		})
		return
	}

	pubCtx, pubCancel := context.WithTimeout(ctx, 5*time.Second)
	defer pubCancel()

	if err := al.bus.PublishOutbound(pubCtx, bus.OutboundMessage{
		Context: bus.InboundContext{
			Channel: "pico",
			ChatID:  chatID,
			Raw: map[string]string{
				metadataKeyStructuredData: string(rawStructured),
			},
		},
		Content: content,
	}); err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) ||
			errors.Is(err, bus.ErrBusClosed) {
			logger.DebugCF("agent", "Pico structured publish skipped (timeout/cancel)", map[string]any{
				"channel": "pico",
				"error":   err.Error(),
			})
		} else {
			logger.WarnCF("agent", "Failed to publish pico structured payload", map[string]any{
				"channel": "pico",
				"error":   err.Error(),
			})
		}
	}
}

func (al *AgentLoop) publishPicoToolProgress(ctx context.Context, chatID, toolName, status, detail string) {
	if toolName == "" || chatID == "" {
		return
	}
	content := fmt.Sprintf("%s: %s", toolName, status)
	if detail != "" {
		content = fmt.Sprintf("%s\n%s", content, detail)
	}
	al.publishPicoStructured(ctx, chatID, content, map[string]any{
		"type":    "progress",
		"kind":    "agent/tool-exec",
		"title":   toolName,
		"status":  status,
		"content": detail,
	})
}

func (al *AgentLoop) handleReasoning(
	ctx context.Context,
	reasoningContent, channelName, channelID string,
) {
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
		Context: bus.NewOutboundContext(channelName, channelID, ""),
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
				"error":   err.Error(),
			})
		} else {
			logger.WarnCF("agent", "Failed to publish reasoning (best-effort)", map[string]any{
				"channel": channelName,
				"error":   err.Error(),
			})
		}
	}
}
