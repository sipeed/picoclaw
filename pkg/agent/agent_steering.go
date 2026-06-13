// PicoClaw - Ultra-lightweight personal AI agent

package agent

import (
	"context"
	"errors"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/logger"
)

const (
	turnDoneStatusOK       = "ok"
	turnDoneStatusError    = "error"
	turnDoneStatusCanceled = "canceled"
)

func turnDoneStatusFromError(err error) string {
	if err == nil {
		return turnDoneStatusOK
	}
	if errors.Is(err, context.Canceled) {
		return turnDoneStatusCanceled
	}
	return turnDoneStatusError
}

func turnDoneStatusFromTurnResult(turnStatus TurnEndStatus, err error) string {
	if err != nil {
		return turnDoneStatusFromError(err)
	}
	if turnStatus == TurnEndStatusAborted {
		return turnDoneStatusCanceled
	}
	return turnDoneStatusOK
}

func (al *AgentLoop) notifyTurnDone(ctx context.Context, msg bus.InboundMessage, status string) {
	if al == nil || al.channelManager == nil {
		return
	}
	msg = bus.NormalizeInboundMessage(msg)
	al.channelManager.NotifyTurnDone(ctx, msg.Channel, msg.ChatID, msg.MessageID, status)
}

func (al *AgentLoop) invokeTypingStop(msg bus.InboundMessage) {
	if al == nil || al.channelManager == nil {
		return
	}
	msg = bus.NormalizeInboundMessage(msg)
	al.channelManager.InvokeTypingStop(msg.Channel, msg.ChatID)
}

func (al *AgentLoop) invokeTypingStopForTarget(channel, chatID string) {
	if al == nil || al.channelManager == nil {
		return
	}
	al.channelManager.InvokeTypingStop(channel, chatID)
}

func (al *AgentLoop) notifyTurnDoneForSteeringEntries(
	ctx context.Context,
	entries []steeringQueueEntry,
	status string,
) {
	if al == nil || al.channelManager == nil {
		return
	}
	for _, entry := range entries {
		if entry.inboundCtx == nil {
			continue
		}
		msg := bus.NormalizeInboundMessage(bus.InboundMessage{
			Context: *cloneInboundContext(entry.inboundCtx),
		})
		al.notifyTurnDone(ctx, msg, status)
	}
}

func (al *AgentLoop) processMessageSync(ctx context.Context, msg bus.InboundMessage) {
	status := turnDoneStatusOK
	defer func() {
		al.invokeTypingStop(msg)
		al.notifyTurnDone(ctx, msg, status)
	}()

	response, turnStatus, err := al.processMessageWithStatus(ctx, msg)
	status = turnDoneStatusFromTurnResult(turnStatus, err)
	if turnStatus == TurnEndStatusAborted {
		response = ""
	}
	al.publishResponseOrError(ctx, msg.Channel, msg.ChatID, msg.SessionKey, response, err)
}

func (al *AgentLoop) runTurnWithSteering(ctx context.Context, initialMsg bus.InboundMessage) {
	status := turnDoneStatusOK
	defer func() {
		al.invokeTypingStop(initialMsg)
		al.notifyTurnDone(ctx, initialMsg, status)
	}()

	// Process the initial message
	response, turnStatus, err := al.processMessageWithStatus(ctx, initialMsg)
	status = turnDoneStatusFromTurnResult(turnStatus, err)
	if err != nil {
		if !al.maybePublishError(ctx, initialMsg.Channel, initialMsg.ChatID, initialMsg.SessionKey, err) {
			return // context canceled
		}
		response = ""
	}
	if turnStatus == TurnEndStatusAborted {
		return
	}
	finalResponse := response

	// Build continuation target
	target, targetErr := al.buildContinuationTarget(initialMsg)
	if targetErr != nil {
		status = turnDoneStatusError
		logger.WarnCF("agent", "Failed to build steering continuation target",
			map[string]any{
				"channel": initialMsg.Channel,
				"error":   targetErr.Error(),
			})
		return
	}
	if target == nil {
		// System message or non-routable, response already published
		return
	}

	continued, continueErr := al.drainQueuedSteeringContinuations(ctx, target)
	if continueErr != nil {
		status = turnDoneStatusFromError(continueErr)
		logger.WarnCF("agent", "Failed to continue queued steering",
			map[string]any{
				"channel": target.Channel,
				"chat_id": target.ChatID,
				"error":   continueErr.Error(),
			})
	} else if continued != "" {
		finalResponse = continued
	}

	// Publish final response
	if finalResponse != "" {
		al.PublishResponseIfNeeded(ctx, target.Channel, target.ChatID, target.SessionKey, finalResponse)
	}
}

func (al *AgentLoop) drainQueuedSteeringContinuations(
	ctx context.Context,
	target *continuationTarget,
) (string, error) {
	if target == nil {
		return "", nil
	}

	finalResponse := ""
	for al.pendingSteeringCountForScope(target.SessionKey) > 0 {
		if err := ctx.Err(); err != nil {
			return finalResponse, err
		}

		logger.InfoCF("agent", "Continuing queued steering after turn end",
			map[string]any{
				"channel":     target.Channel,
				"chat_id":     target.ChatID,
				"session_key": target.SessionKey,
				"queue_depth": al.pendingSteeringCountForScope(target.SessionKey),
			})

		continued, continueErr := al.Continue(ctx, target.SessionKey, target.Channel, target.ChatID)
		if continueErr != nil {
			return finalResponse, continueErr
		}
		if continued == "" {
			break
		}
		finalResponse = continued
	}

	return finalResponse, nil
}

func (al *AgentLoop) resolveSteeringTarget(msg bus.InboundMessage) (string, string, bool) {
	if msg.Channel == "system" {
		return "", "", false
	}

	route, agent, err := al.resolveMessageRoute(msg)
	if err != nil || agent == nil {
		return "", "", false
	}
	allocation := al.allocateRouteSession(route, msg)

	return resolveScopeKey(allocation.SessionKey, msg.SessionKey), agent.ID, true
}
