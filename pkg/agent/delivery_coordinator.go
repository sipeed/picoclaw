package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	runtimeevents "github.com/sipeed/picoclaw/pkg/events"
	"github.com/sipeed/picoclaw/pkg/logger"
	taskregistry "github.com/sipeed/picoclaw/pkg/tasks"
	"github.com/sipeed/picoclaw/pkg/tools"
)

// AsyncDeliveryDecision is the routing plan for a completed async tool result.
//
// This is intentionally decision-only for now. The current runtime still
// performs delivery in pipeline_execute.go, but all routing policy should flow
// through this type so media, duplicate, timeout, and restart handling can move
// behind the same coordinator boundary later.
type AsyncDeliveryDecision struct {
	TaskID        string
	DeliveryMode  tools.AsyncDeliveryMode
	PublishToUser bool
	QueueParent   bool
	ParentHandled bool
	ContentLen    int
	ForUserLen    int
	MediaCount    int
	IsError       bool
}

type AsyncDeliveryRequest struct {
	TurnState    *turnState
	ToolName     string
	CompletionID string
	Result       *tools.ToolResult
	Decision     AsyncDeliveryDecision
}

func (al *AgentLoop) deliverAsyncToolCompletion(req AsyncDeliveryRequest) {
	ts := req.TurnState
	result := req.Result
	asyncToolName := strings.TrimSpace(req.ToolName)
	if ts == nil || result == nil {
		return
	}
	if asyncToolName == "" {
		asyncToolName = "async_tool"
	}
	delivery := req.Decision
	if delivery.DeliveryMode == "" {
		delivery = decideAsyncToolResultDelivery(result)
	}
	completionID := strings.TrimSpace(req.CompletionID)
	if al.asyncTaskDeliveryAlreadyHandled(ts.workspace, delivery.TaskID, completionID) {
		logger.InfoCF("agent", "Skipping duplicate async delivery",
			map[string]any{
				"tool":          asyncToolName,
				"completion_id": completionID,
				"task_id":       delivery.TaskID,
			})
		return
	}
	if result.IsError {
		content := strings.TrimSpace(result.ForUser)
		if content == "" {
			content = strings.TrimSpace(result.ContentForLLM())
		}
		delivered := false
		deliveryErr := ""
		if content != "" && !result.Silent {
			outCtx, outCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer outCancel()
			if err := al.bus.PublishOutbound(outCtx, outboundMessageForTurn(ts, content)); err != nil {
				deliveryErr = err.Error()
			} else {
				delivered = true
			}
		}
		switch {
		case delivered:
			al.updateAsyncTaskDeliveryStatus(
				ts.workspace,
				delivery.TaskID,
				taskregistry.DeliveryDelivered,
				completionID,
				"",
			)
		case deliveryErr != "":
			al.updateAsyncTaskDeliveryStatus(
				ts.workspace,
				delivery.TaskID,
				taskregistry.DeliveryFailed,
				completionID,
				deliveryErr,
			)
		default:
			al.updateAsyncTaskDeliveryStatus(
				ts.workspace,
				delivery.TaskID,
				taskregistry.DeliveryNotApplicable,
				completionID,
				"",
			)
		}
		return
	}
	if delivery.PublishToUser {
		outCtx, outCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer outCancel()
		userDelivered := false
		userDeliveryErr := ""
		if _, delivered, err := al.deliverToolResultToUser(outCtx, ts, result, asyncToolName); err != nil {
			userDeliveryErr = err.Error()
			logger.WarnCF("agent", "Failed to deliver async tool result to user",
				map[string]any{
					"tool":    asyncToolName,
					"channel": ts.channel,
					"chat_id": ts.chatID,
					"error":   err.Error(),
				})
		} else if !delivered && delivery.MediaCount > 0 {
			userDelivered = true
		} else if !delivered && strings.TrimSpace(result.ForUser) != "" && !result.Silent {
			if err := al.bus.PublishOutbound(outCtx, outboundMessageForTurn(ts, result.ForUser)); err != nil {
				userDeliveryErr = err.Error()
			} else {
				userDelivered = true
			}
		} else if delivered {
			userDelivered = true
		}
		if !delivery.QueueParent {
			if userDelivered {
				al.updateAsyncTaskDeliveryStatus(
					ts.workspace,
					delivery.TaskID,
					taskregistry.DeliveryDelivered,
					completionID,
					"",
				)
			} else if userDeliveryErr != "" {
				al.updateAsyncTaskDeliveryStatus(
					ts.workspace,
					delivery.TaskID,
					taskregistry.DeliveryFailed,
					completionID,
					userDeliveryErr,
				)
			} else {
				al.updateAsyncTaskDeliveryStatus(
					ts.workspace,
					delivery.TaskID,
					taskregistry.DeliveryNotApplicable,
					completionID,
					"",
				)
			}
			return
		}
	}

	if !delivery.QueueParent {
		al.updateAsyncTaskDeliveryStatus(
			ts.workspace,
			delivery.TaskID,
			taskregistry.DeliveryNotApplicable,
			completionID,
			"",
		)
		return
	}

	content := result.ContentForLLM()
	content = al.cfg.FilterSensitiveData(content)

	logger.InfoCF("agent", "Async tool completed, publishing result",
		map[string]any{
			"tool":        asyncToolName,
			"content_len": len(content),
			"channel":     ts.channel,
		})
	al.emitEvent(
		runtimeevents.KindAgentFollowUpQueued,
		ts.scope.meta(0, "delivery_coordinator", "turn.follow_up.queued"),
		FollowUpQueuedPayload{
			SourceTool: asyncToolName,
			ContentLen: len(content),
		},
	)
	origin := bus.InboundContext{
		Channel:  ts.channel,
		ChatID:   ts.chatID,
		ChatType: "direct",
		SenderID: fmt.Sprintf("async:%s", asyncToolName),
		TopicID:  originTopicID(ts.opts.Dispatch.InboundContext),
	}
	if ts.opts.Dispatch.InboundContext != nil {
		origin = *cloneInboundContext(ts.opts.Dispatch.InboundContext)
		if strings.TrimSpace(origin.Channel) == "" {
			origin.Channel = ts.channel
		}
		if strings.TrimSpace(origin.ChatID) == "" {
			origin.ChatID = ts.chatID
		}
		if strings.TrimSpace(origin.ChatType) == "" {
			origin.ChatType = "direct"
		}
		origin.SenderID = fmt.Sprintf("async:%s", asyncToolName)
	}
	completionCtx, completionCancel := context.WithTimeout(context.Background(), asyncCompletionSynthesisTimeout)
	defer completionCancel()
	if _, err := al.processAsyncCompletion(completionCtx, AsyncCompletionInput{
		SourceTool:   asyncToolName,
		CompletionID: completionID,
		Content:      asyncCompletionPrompt(asyncToolName, content),
		Origin:       origin,
		SenderID:     fmt.Sprintf("async:%s", asyncToolName),
	}); err != nil {
		al.updateAsyncTaskDeliveryStatus(
			ts.workspace,
			delivery.TaskID,
			taskregistry.DeliveryFailed,
			completionID,
			err.Error(),
		)
		logger.WarnCF("agent", "Failed to process async completion",
			map[string]any{
				"tool":          asyncToolName,
				"completion_id": completionID,
				"channel":       ts.channel,
				"chat_id":       ts.chatID,
				"error":         err.Error(),
			})
	} else if delivery.DeliveryMode == tools.AsyncDeliveryParentOnly {
		al.updateAsyncTaskDeliveryStatus(
			ts.workspace,
			delivery.TaskID,
			taskregistry.DeliverySessionQueued,
			completionID,
			"",
		)
	} else {
		al.updateAsyncTaskDeliveryStatus(
			ts.workspace,
			delivery.TaskID,
			taskregistry.DeliveryDelivered,
			completionID,
			"",
		)
	}
}

func decideAsyncToolResultDelivery(result *tools.ToolResult) AsyncDeliveryDecision {
	decision := AsyncDeliveryDecision{
		DeliveryMode: effectiveAsyncToolResultDelivery(result),
	}
	if result == nil {
		return decision
	}

	content := result.ContentForLLM()
	decision.TaskID = result.AsyncTaskID
	decision.ContentLen = len(content)
	decision.ForUserLen = len(result.ForUser)
	decision.MediaCount = len(result.Media)
	if result.Completion != nil {
		decision.MediaCount += len(result.Completion.Media)
	}
	decision.IsError = result.IsError

	if decision.DeliveryMode != tools.AsyncDeliveryParentOnly {
		decision.PublishToUser = !result.Silent && (result.ForUser != "" || decision.MediaCount > 0)
	}
	if decision.DeliveryMode != tools.AsyncDeliveryUserOnly {
		decision.QueueParent = content != ""
	}
	decision.ParentHandled = !decision.QueueParent && !result.IsError &&
		decision.DeliveryMode == tools.AsyncDeliveryUserOnly
	return decision
}

func effectiveAsyncToolResultDelivery(result *tools.ToolResult) tools.AsyncDeliveryMode {
	if result == nil || result.AsyncDelivery == "" {
		return tools.AsyncDeliveryUserAndParent
	}
	return result.AsyncDelivery
}

func asyncDeliveryModeFromToolArgs(toolName string, args map[string]any) (tools.AsyncDeliveryMode, error) {
	if toolName != "spawn" && toolName != "delegate" {
		return tools.AsyncDeliveryUserAndParent, nil
	}
	raw, ok := args["delivery_mode"]
	if !ok || raw == nil {
		if toolName == "spawn" {
			return tools.AsyncDeliveryUserOnly, nil
		}
		return tools.AsyncDeliveryParentOnly, nil
	}
	value, ok := raw.(string)
	if !ok {
		return "", nil
	}
	switch mode := tools.AsyncDeliveryMode(strings.TrimSpace(value)); mode {
	case tools.AsyncDeliveryUserOnly, tools.AsyncDeliveryParentOnly, tools.AsyncDeliveryUserAndParent:
		return mode, nil
	default:
		return "", nil
	}
}
