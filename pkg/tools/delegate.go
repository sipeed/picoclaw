package tools

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/sipeed/picoclaw/pkg/routing"
	taskregistry "github.com/sipeed/picoclaw/pkg/tasks"
)

// DelegateTool delegates a task to a specific named agent and waits for
// the result. Unlike spawn (async, fire-and-forget) or subagent (sync but
// generic), delegate targets a named agent and runs the task using that
// agent's own workspace, model, and tools.
type DelegateTool struct {
	spawner        SubTurnSpawner
	allowlistCheck func(targetAgentID string) bool
	selfAgentID    string
	taskRegistry   *taskregistry.Registry
	taskSeq        atomic.Int64
}

func NewDelegateTool() *DelegateTool {
	return &DelegateTool{}
}

func (t *DelegateTool) SetSpawner(spawner SubTurnSpawner) {
	t.spawner = spawner
}

func (t *DelegateTool) SetAllowlistChecker(check func(targetAgentID string) bool) {
	t.allowlistCheck = check
}

func (t *DelegateTool) SetSelfAgentID(id string) {
	t.selfAgentID = id
}

func (t *DelegateTool) SetTaskRegistry(registry *taskregistry.Registry) {
	t.taskRegistry = registry
}

func (t *DelegateTool) Name() string {
	return "delegate"
}

func (t *DelegateTool) Description() string {
	return "Delegate a task to another agent and wait for the result. " +
		"Use this when another agent is better suited to handle a specific task " +
		"based on their capabilities. The target agent runs with its own workspace, " +
		"model, and tools."
}

func (t *DelegateTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"agent_id": map[string]any{
				"type":        "string",
				"description": "The ID of the target agent to delegate the task to",
			},
			"task": map[string]any{
				"type":        "string",
				"description": "Clear description of the task to delegate",
			},
			"delivery_mode": map[string]any{
				"type":        "string",
				"description": "Optional sync result routing policy: parent_only, user_only, or user_and_parent. Defaults to parent_only.",
				"enum": []string{
					string(AsyncDeliveryParentOnly),
					string(AsyncDeliveryUserOnly),
					string(AsyncDeliveryUserAndParent),
				},
			},
		},
		"required": []string{"agent_id", "task"},
	}
}

func (t *DelegateTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	rawAgentID, _ := args["agent_id"].(string)
	if strings.TrimSpace(rawAgentID) == "" {
		return ErrorResult("agent_id is required and must be a non-empty string")
	}
	agentID := routing.NormalizeAgentID(rawAgentID)

	task, _ := args["task"].(string)
	if strings.TrimSpace(task) == "" {
		return ErrorResult("task is required and must be a non-empty string")
	}
	deliveryMode, err := parseDelegateDeliveryMode(args["delivery_mode"])
	if err != nil {
		return ErrorResult(err.Error()).WithError(err)
	}

	if t.selfAgentID != "" && agentID == t.selfAgentID {
		return ErrorResult("cannot delegate to self")
	}

	if t.allowlistCheck != nil && !t.allowlistCheck(agentID) {
		return ErrorResult(fmt.Sprintf("not allowed to delegate to agent %q", agentID))
	}

	if t.spawner == nil {
		return ErrorResult("delegate tool not configured")
	}

	taskID := t.nextTaskID()
	t.recordDelegateTask(
		ctx, taskID, agentID, task, deliveryMode,
		taskregistry.StatusRunning,
		taskregistry.DeliveryPending,
		"",
		nil,
	)
	result, err := t.spawner.SpawnSubTurn(ctx, SubTurnConfig{
		TargetAgentID: agentID,
		SystemPrompt:  task,
		Async:         false,
		DeliveryMode:  deliveryMode,
	})
	if err != nil {
		msg := fmt.Sprintf("delegation to agent %q failed: %v", agentID, err)
		t.recordDelegateTask(
			ctx, taskID, agentID, task, deliveryMode,
			taskregistry.StatusFailed,
			taskregistry.DeliveryPending,
			msg,
			nil,
		)
		return ErrorResult(fmt.Sprintf("delegation to agent %q failed: %v", agentID, err)).WithError(err)
	}
	if result == nil {
		msg := fmt.Sprintf("delegation to agent %q returned no result", agentID)
		t.recordDelegateTask(
			ctx, taskID, agentID, task, deliveryMode,
			taskregistry.StatusFailed,
			taskregistry.DeliveryPending,
			msg,
			nil,
		)
		return ErrorResult(fmt.Sprintf("delegation to agent %q returned no result", agentID))
	}

	result.ForLLM = fmt.Sprintf("[Response from agent %q]\n%s", agentID, result.ForLLM)
	if deliveryMode == AsyncDeliveryUserOnly {
		result.Silent = true
		result.ResponseHandled = true
	}
	t.recordDelegateTask(
		ctx, taskID, agentID, task, deliveryMode,
		taskregistry.StatusSucceeded,
		delegateDeliveryStatus(result, deliveryMode),
		result.ContentForLLM(),
		completionPayloadForTaskRegistry(result),
	)

	return result
}

func (t *DelegateTool) nextTaskID() string {
	seq := t.taskSeq.Add(1)
	return fmt.Sprintf("delegate-%d-%d", time.Now().UnixMilli(), seq)
}

func (t *DelegateTool) recordDelegateTask(
	ctx context.Context,
	taskID, agentID, task string,
	deliveryMode AsyncDeliveryMode,
	status taskregistry.Status,
	delivery taskregistry.DeliveryStatus,
	summary string,
	completion *taskregistry.CompletionPayload,
) {
	if t == nil || t.taskRegistry == nil || taskID == "" {
		return
	}
	now := time.Now().UnixMilli()
	rec := taskregistry.Record{
		TaskID:              taskID,
		Runtime:             taskregistry.RuntimeDelegate,
		TaskKind:            "delegate",
		RequesterSessionKey: ToolSessionKey(ctx),
		OwnerKey:            ToolAgentID(ctx),
		Channel:             ToolChannel(ctx),
		ChatID:              ToolChatID(ctx),
		TopicID:             ToolTopicID(ctx),
		AgentID:             agentID,
		Label:               "delegate:" + agentID,
		Task:                task,
		Status:              status,
		DeliveryStatus:      delivery,
		NotifyPolicy:        taskregistry.NotifyDoneOnly,
		DeliveryMode:        string(deliveryMode),
		LastEventAt:         now,
		TerminalSummary:     summary,
		Completion:          completion,
	}
	if status == taskregistry.StatusRunning {
		rec.CreatedAt = now
		rec.StartedAt = now
	} else if existing, ok := t.taskRegistry.Get(taskID); ok {
		rec.CreatedAt = existing.CreatedAt
		rec.StartedAt = existing.StartedAt
	}
	if status == taskregistry.StatusFailed {
		rec.Error = summary
	}
	_ = t.taskRegistry.Upsert(rec)
}

func delegateDeliveryStatus(result *ToolResult, mode AsyncDeliveryMode) taskregistry.DeliveryStatus {
	if result == nil {
		return taskregistry.DeliveryFailed
	}
	switch mode {
	case AsyncDeliveryParentOnly:
		return taskregistry.DeliverySessionQueued
	case AsyncDeliveryUserOnly:
		if result.ResponseHandled || result.Silent {
			return taskregistry.DeliveryDelivered
		}
		return taskregistry.DeliveryPending
	case AsyncDeliveryUserAndParent:
		return taskregistry.DeliveryPending
	default:
		return taskregistry.DeliveryPending
	}
}

func parseDelegateDeliveryMode(raw any) (AsyncDeliveryMode, error) {
	if raw == nil {
		return AsyncDeliveryParentOnly, nil
	}
	value, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("delivery_mode must be a string")
	}
	switch AsyncDeliveryMode(strings.TrimSpace(value)) {
	case AsyncDeliveryParentOnly, AsyncDeliveryUserOnly, AsyncDeliveryUserAndParent:
		return AsyncDeliveryMode(strings.TrimSpace(value)), nil
	case "":
		return AsyncDeliveryParentOnly, nil
	default:
		return "", fmt.Errorf("delivery_mode must be one of: parent_only, user_only, user_and_parent")
	}
}
