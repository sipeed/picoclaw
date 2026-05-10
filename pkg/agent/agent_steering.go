// PicoClaw - Ultra-lightweight personal AI agent

package agent

import (
	"context"
	"strings"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/logger"
)

func (al *AgentLoop) processMessageSync(ctx context.Context, msg bus.InboundMessage) {
	if al.channelManager != nil {
		defer al.channelManager.InvokeTypingStop(msg.Channel, msg.ChatID)
	}

	response, err := al.processMessage(ctx, msg)
	if err != nil {
		if !al.maybePublishErrorWithPolicy(
			ctx,
			msg.Channel,
			msg.ChatID,
			msg.SessionKey,
			err,
			finalResponseAlwaysPublish,
		) {
			return
		}
		response = ""
	}
	al.publishResponseWithContextIfNeeded(
		ctx,
		msg.Channel,
		msg.ChatID,
		msg.SessionKey,
		response,
		&msg.Context,
		finalResponseAlwaysPublish,
	)
}

func (al *AgentLoop) runTurnWithSteering(ctx context.Context, initialMsg bus.InboundMessage) {
	// Process the initial message
	response, err := al.processMessage(ctx, initialMsg)
	if err != nil {
		if !al.maybePublishErrorWithPolicy(
			ctx,
			initialMsg.Channel,
			initialMsg.ChatID,
			initialMsg.SessionKey,
			err,
			finalResponseAlwaysPublish,
		) {
			return // context canceled
		}
		response = ""
	}
	responses := appendSteeringResponse(nil, response)

	// Build continuation target
	target, targetErr := al.buildContinuationTarget(initialMsg)
	if targetErr != nil {
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
		logger.WarnCF("agent", "Failed to continue queued steering",
			map[string]any{
				"channel": target.Channel,
				"chat_id": target.ChatID,
				"error":   continueErr.Error(),
			})
	} else {
		responses = appendSteeringResponse(responses, continued)
	}

	// Publish final response
	finalResponse := joinSteeringResponses(responses)
	if finalResponse != "" {
		al.publishResponseWithContextIfNeeded(
			ctx,
			target.Channel,
			target.ChatID,
			target.SessionKey,
			finalResponse,
			&bus.InboundContext{
				Channel: initialMsg.Context.Channel,
				ChatID:  initialMsg.Context.ChatID,
				TopicID: initialMsg.Context.TopicID,
				Raw: func() map[string]string {
					raw := make(map[string]string, len(initialMsg.Context.Raw)+1)
					for k, v := range initialMsg.Context.Raw {
						raw[k] = v
					}
					raw[metadataKeyMessageKind] = messageKindFinalReply
					return raw
				}(),
			},
			finalResponseAlwaysPublish,
		)
	}
}

func (al *AgentLoop) drainQueuedSteeringContinuations(
	ctx context.Context,
	target *continuationTarget,
) (string, error) {
	if target == nil {
		return "", nil
	}

	responses := make([]string, 0, 2)
	for al.pendingSteeringCountForScope(target.SessionKey) > 0 {
		if err := ctx.Err(); err != nil {
			return joinSteeringResponses(responses), err
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
			return joinSteeringResponses(responses), continueErr
		}
		if continued == "" {
			break
		}
		responses = appendSteeringResponse(responses, continued)
	}

	return joinSteeringResponses(responses), nil
}

func appendSteeringResponse(responses []string, response string) []string {
	response = strings.TrimSpace(response)
	if response == "" {
		return responses
	}
	if n := len(responses); n > 0 && responses[n-1] == response {
		return responses
	}
	return append(responses, response)
}

func joinSteeringResponses(responses []string) string {
	if len(responses) == 0 {
		return ""
	}
	return strings.Join(responses, "\n\n")
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

	return al.resolveEffectiveSessionKey(allocation.SessionKey, msg.SessionKey), agent.ID, true
}
