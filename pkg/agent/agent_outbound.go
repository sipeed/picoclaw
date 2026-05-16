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
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/tools"
	"github.com/sipeed/picoclaw/pkg/utils"
)

type finalResponseDeliveryPolicy uint8

const (
	finalResponseSuppressIfMessageToolSent finalResponseDeliveryPolicy = iota
	finalResponseAlwaysPublish
)

func (al *AgentLoop) maybePublishError(ctx context.Context, channel, chatID, sessionKey string, err error) bool {
	return al.maybePublishErrorWithPolicy(
		ctx,
		channel,
		chatID,
		sessionKey,
		err,
		finalResponseSuppressIfMessageToolSent,
	)
}

func (al *AgentLoop) maybePublishErrorWithPolicy(
	ctx context.Context,
	channel, chatID, sessionKey string,
	err error,
	policy finalResponseDeliveryPolicy,
) bool {
	if errors.Is(err, context.Canceled) {
		return false
	}
	al.publishResponseIfNeededWithPolicy(
		ctx,
		channel,
		chatID,
		sessionKey,
		fmt.Sprintf("Error processing message: %v", err),
		policy,
	)
	return true
}

func (al *AgentLoop) publishResponseOrErrorWithPolicy(
	ctx context.Context,
	channel, chatID, sessionKey string,
	response string,
	err error,
	policy finalResponseDeliveryPolicy,
) {
	if err != nil {
		if !al.maybePublishErrorWithPolicy(ctx, channel, chatID, sessionKey, err, policy) {
			return
		}
		response = ""
	}
	al.publishResponseIfNeededWithPolicy(ctx, channel, chatID, sessionKey, response, policy)
}

func (al *AgentLoop) PublishResponseIfNeeded(ctx context.Context, channel, chatID, sessionKey, response string) {
	al.publishResponseIfNeededWithPolicy(
		ctx,
		channel,
		chatID,
		sessionKey,
		response,
		finalResponseSuppressIfMessageToolSent,
	)
}

func (al *AgentLoop) publishResponseIfNeededWithPolicy(
	ctx context.Context,
	channel, chatID, sessionKey, response string,
	policy finalResponseDeliveryPolicy,
) {
	if response == "" {
		return
	}

	alreadySentToSameChat := false
	if policy == finalResponseSuppressIfMessageToolSent {
		defaultAgent := al.GetRegistry().GetDefaultAgent()
		if defaultAgent != nil {
			if tool, ok := defaultAgent.Tools.Get("message"); ok {
				if mt, ok := tool.(*tools.MessageTool); ok {
					alreadySentToSameChat = mt.HasSentTo(sessionKey, channel, chatID)
				}
			}
		}
	}

	if alreadySentToSameChat {
		if al.channelManager != nil && channel != "" && chatID != "" {
			dismissCtx, dismissCancel := context.WithTimeout(ctx, 5*time.Second)
			al.channelManager.DismissToolFeedback(
				dismissCtx,
				channel,
				chatID,
				nil,
			)
			dismissCancel()
		}
		logger.DebugCF(
			"agent",
			"Skipped outbound (message tool already sent to same chat)",
			map[string]any{"channel": channel, "chat_id": chatID},
		)
		return
	}

	msg := bus.OutboundMessage{
		Context: bus.NewOutboundContext(channel, chatID, ""),
		Content: response,
	}
	if sessionKey != "" {
		msg.ContextUsage = computeContextUsage(al.agentForSession(sessionKey), sessionKey)
	}
	al.bus.PublishOutbound(ctx, msg)
	logger.InfoCF("agent", "Published outbound response",
		map[string]any{
			"channel":     channel,
			"chat_id":     chatID,
			"content_len": len(response),
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

func (al *AgentLoop) publishPicoToolCallInterim(
	ctx context.Context,
	ts *turnState,
	reasoningContent string,
	content string,
	toolCalls []providers.ToolCall,
) {
	if ts == nil || ts.chatID == "" || al == nil || al.bus == nil {
		return
	}

	if strings.TrimSpace(reasoningContent) != "" {
		pubCtx, pubCancel := context.WithTimeout(ctx, 3*time.Second)
		err := al.bus.PublishOutbound(
			pubCtx,
			outboundMessageForTurnWithKind(ts, reasoningContent, messageKindThought),
		)
		pubCancel()
		if err != nil && !errors.Is(err, context.DeadlineExceeded) &&
			!errors.Is(err, context.Canceled) &&
			!errors.Is(err, bus.ErrBusClosed) {
			logger.WarnCF("agent", "Failed to publish pico reasoning", map[string]any{
				"channel": ts.channel,
				"chat_id": ts.chatID,
				"error":   err.Error(),
			})
		}
	}

	if !ts.opts.AllowInterimPicoPublish {
		return
	}

	visibleToolCalls := utils.BuildVisibleToolCalls(
		toolCalls,
		al.cfg.Agents.Defaults.GetToolFeedbackMaxArgsLength(),
	)
	duplicateToolCallContent := len(visibleToolCalls) > 0 &&
		utils.ToolCallExplanationDuplicatesContent(content, toolCalls)

	if strings.TrimSpace(content) != "" && !duplicateToolCallContent {
		pubCtx, pubCancel := context.WithTimeout(ctx, 3*time.Second)
		err := al.bus.PublishOutbound(pubCtx, outboundMessageForTurn(ts, content))
		pubCancel()
		if err != nil && !errors.Is(err, context.DeadlineExceeded) &&
			!errors.Is(err, context.Canceled) &&
			!errors.Is(err, bus.ErrBusClosed) {
			logger.WarnCF("agent", "Failed to publish pico interim assistant content", map[string]any{
				"channel": ts.channel,
				"chat_id": ts.chatID,
				"error":   err.Error(),
			})
		}
	}

	if len(visibleToolCalls) == 0 {
		return
	}

	rawToolCalls, err := json.Marshal(visibleToolCalls)
	if err != nil {
		logger.WarnCF("agent", "Failed to serialize pico tool calls", map[string]any{
			"channel": ts.channel,
			"chat_id": ts.chatID,
			"error":   err.Error(),
		})
		return
	}

	msg := outboundMessageForTurnWithKind(ts, "", messageKindToolCalls)
	if msg.Context.Raw == nil {
		msg.Context.Raw = map[string]string{}
	}
	msg.Context.Raw[metadataKeyToolCalls] = string(rawToolCalls)

	pubCtx, pubCancel := context.WithTimeout(ctx, 3*time.Second)
	err = al.bus.PublishOutbound(pubCtx, msg)
	pubCancel()
	if err != nil && !errors.Is(err, context.DeadlineExceeded) &&
		!errors.Is(err, context.Canceled) &&
		!errors.Is(err, bus.ErrBusClosed) {
		logger.WarnCF("agent", "Failed to publish pico tool calls", map[string]any{
			"channel": ts.channel,
			"chat_id": ts.chatID,
			"error":   err.Error(),
		})
	}
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
