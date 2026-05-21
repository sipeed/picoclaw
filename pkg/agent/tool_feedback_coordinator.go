package agent

import (
	"context"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/utils"
)

// publishToolFeedbackForCall is the agent-side entry point for visible tool
// progress. Channel-specific code still owns editing/deleting the message, but
// this function owns whether a turn should emit a feedback update and how that
// update is formatted.
func (al *AgentLoop) publishToolFeedbackForCall(
	ctx context.Context,
	ts *turnState,
	response *providers.LLMResponse,
	toolCall providers.ToolCall,
	toolName string,
	toolArgs map[string]any,
	messages []providers.Message,
) {
	if !shouldPublishToolFeedback(al, ts) || ts.channel == "pico" {
		return
	}
	toolFeedbackMaxLen := al.cfg.Agents.Defaults.GetToolFeedbackMaxArgsLength()
	toolFeedbackExplanation := toolFeedbackExplanationForToolCall(
		response,
		toolCall,
		messages,
	)
	toolArgsPreview := toolFeedbackArgsPreview(toolArgs, toolFeedbackMaxLen)
	feedbackMsg := utils.FormatToolFeedbackMessageWithStyle(
		al.cfg.Agents.Defaults.GetToolFeedbackStyle(),
		toolName,
		toolFeedbackExplanation,
		toolArgsPreview,
	)
	if title := toolFeedbackTitleForTurn(ts); title != "" {
		feedbackMsg = utils.FormatToolFeedbackMessageWithStyleAndTitle(
			al.cfg.Agents.Defaults.GetToolFeedbackStyle(),
			title,
			toolName,
			toolFeedbackExplanation,
			toolArgsPreview,
		)
	}
	fbCtx, fbCancel := context.WithTimeout(ctx, 3*time.Second)
	_ = al.bus.PublishOutbound(fbCtx, outboundMessageForTurnWithOptions(
		ts,
		feedbackMsg,
		outboundTurnMessageOptions{kind: messageKindToolFeedback},
	))
	fbCancel()
}

func (al *AgentLoop) dismissToolFeedbackForTurn(ctx context.Context, ts *turnState) {
	if al == nil || al.channelManager == nil || ts == nil || ts.channel == "" {
		return
	}
	al.channelManager.DismissToolFeedback(ctx, ts.channel, ts.chatID, ts.opts.InboundContext)
}

func (al *AgentLoop) dismissToolFeedbackForSession(
	ctx context.Context,
	channel string,
	chatID string,
	inboundCtx *bus.InboundContext,
	sessionKey string,
) {
	if al == nil || al.channelManager == nil || channel == "" || chatID == "" {
		return
	}
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
