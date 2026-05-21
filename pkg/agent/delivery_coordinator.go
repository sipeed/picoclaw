package agent

import (
	"strings"

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
		decision.PublishToUser = !result.Silent && result.ForUser != ""
	}
	if decision.DeliveryMode != tools.AsyncDeliveryUserOnly {
		decision.QueueParent = content != ""
	}
	decision.ParentHandled = !decision.QueueParent && !result.IsError && decision.DeliveryMode == tools.AsyncDeliveryUserOnly
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
