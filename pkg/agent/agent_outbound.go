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
	"github.com/sipeed/picoclaw/pkg/constants"
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
	al.publishResponseWithContextIfNeeded(
		ctx,
		channel,
		chatID,
		sessionKey,
		fmt.Sprintf("Error processing message: %v", err),
		nil,
		policy,
	)
	return true
}

func (al *AgentLoop) publishResponseOrError(
	ctx context.Context,
	channel, chatID, sessionKey string,
	response string,
	err error,
) {
	al.publishResponseOrErrorWithPolicy(
		ctx,
		channel,
		chatID,
		sessionKey,
		response,
		err,
		finalResponseSuppressIfMessageToolSent,
	)
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
	al.publishResponseWithContextIfNeeded(ctx, channel, chatID, sessionKey, response, nil, policy)
}

func (al *AgentLoop) PublishResponseIfNeeded(ctx context.Context, channel, chatID, sessionKey, response string) {
	al.publishResponseWithContextIfNeeded(
		ctx,
		channel,
		chatID,
		sessionKey,
		response,
		nil,
		finalResponseSuppressIfMessageToolSent,
	)
}

func (al *AgentLoop) publishResponseWithContextIfNeeded(
	ctx context.Context,
	channel, chatID, sessionKey, response string,
	inboundCtx *bus.InboundContext,
	policy finalResponseDeliveryPolicy,
) {
	if response == "" {
		return
	}

	messageToolSentToSameChat := al.messageToolSentToSameChat(sessionKey, channel, chatID)

	if policy == finalResponseSuppressIfMessageToolSent && messageToolSentToSameChat {
		if al.channelManager != nil && channel != "" && chatID != "" {
			dismissCtx, dismissCancel := context.WithTimeout(ctx, 5*time.Second)
			al.channelManager.DismissToolFeedbackForSession(
				dismissCtx,
				channel,
				chatID,
				inboundCtx,
				sessionKey,
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

	agent := al.agentForSession(sessionKey)
	agentID := ""
	if agent != nil {
		agentID = agent.ID
	}
	msg := bus.OutboundMessage{
		Channel:    channel,
		ChatID:     chatID,
		Context:    outboundContextFromInbound(inboundCtx, channel, chatID, ""),
		AgentID:    agentID,
		SessionKey: sessionKey,
		Content:    response,
	}
	if policy == finalResponseAlwaysPublish && messageToolSentToSameChat {
		if msg.Context.Raw == nil {
			msg.Context.Raw = make(map[string]string, 1)
		}
		msg.Context.Raw[metadataKeyMessageKind] = messageKindFinalReply
	}
	if sessionKey != "" {
		msg.ContextUsage = computeContextUsage(agent, sessionKey)
	}
	al.bus.PublishOutbound(ctx, msg)
}

func (al *AgentLoop) deliverToolResultToUser(
	ctx context.Context,
	ts *turnState,
	result *tools.ToolResult,
	toolName string,
) ([]providers.Attachment, bool, error) {
	if al == nil || ts == nil || result == nil {
		return nil, false, nil
	}

	mediaRefs := toolResultMediaRefs(result)
	text := toolResultUserText(result)
	if len(mediaRefs) > 0 {
		parts := al.mediaPartsFromRefs(mediaRefs, result.Completion, text)
		outboundMedia := bus.OutboundMediaMessage{
			Channel: ts.channel,
			ChatID:  ts.chatID,
			Context: outboundContextFromInbound(
				ts.opts.Dispatch.InboundContext,
				ts.channel,
				ts.chatID,
				ts.opts.Dispatch.ReplyToMessageID(),
			),
			AgentID:    ts.agent.ID,
			SessionKey: ts.sessionKey,
			Scope:      outboundScopeFromSessionScope(ts.opts.Dispatch.SessionScope),
			Parts:      parts,
		}
		if al.channelManager != nil && ts.channel != "" && !constants.IsInternalChannel(ts.channel) {
			if err := al.channelManager.SendMedia(ctx, outboundMedia); err != nil {
				logger.WarnCF("agent", "Failed to deliver tool result media",
					map[string]any{
						"agent_id": ts.agent.ID,
						"tool":     toolName,
						"channel":  ts.channel,
						"chat_id":  ts.chatID,
						"error":    err.Error(),
					})
				return nil, false, err
			}
			return buildProviderAttachments(al.mediaStore, mediaRefs), true, nil
		}
		if al.bus != nil {
			al.bus.PublishOutboundMedia(ctx, outboundMedia)
		}
		return nil, false, nil
	}

	if strings.TrimSpace(text) == "" {
		return nil, false, nil
	}
	if result.Silent && result.Completion == nil {
		return nil, false, nil
	}
	if al.bus == nil {
		return nil, false, nil
	}
	al.bus.PublishOutbound(ctx, outboundMessageForTurn(ts, text))
	logger.DebugCF("agent", "Sent tool result to user",
		map[string]any{
			"tool":        toolName,
			"content_len": len(text),
		})
	return nil, true, nil
}

func toolResultUserText(result *tools.ToolResult) string {
	if result == nil {
		return ""
	}
	if text := strings.TrimSpace(result.ForUser); text != "" {
		return result.ForUser
	}
	if result.Completion != nil {
		return result.Completion.Text
	}
	return ""
}

func toolResultMediaRefs(result *tools.ToolResult) []string {
	if result == nil {
		return nil
	}
	seen := make(map[string]struct{}, len(result.Media))
	refs := make([]string, 0, len(result.Media))
	appendRef := func(ref string) {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			return
		}
		if _, ok := seen[ref]; ok {
			return
		}
		seen[ref] = struct{}{}
		refs = append(refs, ref)
	}
	for _, ref := range result.Media {
		appendRef(ref)
	}
	if result.Completion != nil {
		for _, item := range result.Completion.Media {
			appendRef(item.Ref)
		}
	}
	return refs
}

func (al *AgentLoop) mediaPartsFromRefs(
	refs []string,
	completion *tools.CompletionResult,
	caption string,
) []bus.MediaPart {
	hints := make(map[string]tools.CompletionMedia)
	if completion != nil {
		for _, item := range completion.Media {
			ref := strings.TrimSpace(item.Ref)
			if ref != "" {
				hints[ref] = item
			}
		}
	}

	parts := make([]bus.MediaPart, 0, len(refs))
	for i, ref := range refs {
		part := bus.MediaPart{Ref: ref}
		if item, ok := hints[ref]; ok {
			part.Type = item.Type
			part.Filename = item.Filename
			part.ContentType = item.ContentType
		}
		if al != nil && al.mediaStore != nil {
			if _, meta, err := al.mediaStore.ResolveWithMeta(ref); err == nil {
				if part.Filename == "" {
					part.Filename = meta.Filename
				}
				if part.ContentType == "" {
					part.ContentType = meta.ContentType
				}
				if part.Type == "" {
					part.Type = inferMediaType(meta.Filename, meta.ContentType)
				}
			}
		}
		if i == 0 {
			part.Caption = caption
		}
		parts = append(parts, part)
	}
	return parts
}

func (al *AgentLoop) messageToolSentToSameChat(sessionKey, channel, chatID string) bool {
	if strings.TrimSpace(sessionKey) == "" {
		return false
	}
	agents := make([]*AgentInstance, 0, 2)
	if agent := al.agentForSession(sessionKey); agent != nil {
		agents = append(agents, agent)
	}
	if defaultAgent := al.GetRegistry().GetDefaultAgent(); defaultAgent != nil {
		agents = append(agents, defaultAgent)
	}
	for _, agent := range agents {
		if agent == nil || agent.Tools == nil {
			continue
		}
		tool, ok := agent.Tools.Get("message")
		if !ok {
			continue
		}
		mt, ok := tool.(*tools.MessageTool)
		if ok && mt.HasSentTo(sessionKey, channel, chatID) {
			return true
		}
	}
	return false
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
