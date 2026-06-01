package tools

import (
	"context"
	"errors"
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
		"model, and tools. For multi-step workflows, create/inspect the workflow with task_board, " +
		"then pass board_id and step metadata so related delegate/spawn runs appear together in task_board and task_status."
}

func (t *DelegateTool) Parameters() map[string]any {
	props := map[string]any{
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
		"timeout_seconds": map[string]any{
			"type":        "number",
			"description": "Optional maximum time to wait for this delegated child step. If omitted, the runtime subturn default is used.",
		},
	}
	addTaskBoardMetadataParameters(props)
	return map[string]any{
		"type":       "object",
		"properties": props,
		"required":   []string{"agent_id", "task"},
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
	boardMeta, err := parseTaskBoardMetadata(args)
	if err != nil {
		return ErrorResult(err.Error()).WithError(err)
	}
	timeout, err := parseOptionalTimeoutSeconds(args["timeout_seconds"])
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
		ctx, taskID, agentID, task, deliveryMode, boardMeta,
		taskregistry.StatusRunning,
		taskregistry.DeliveryPending,
		"",
		nil,
		nil,
	)
	result, err := t.spawner.SpawnSubTurn(ctx, SubTurnConfig{
		TargetAgentID: agentID,
		SystemPrompt:  task,
		Async:         false,
		DeliveryMode:  deliveryMode,
		Timeout:       timeout,
	})
	if err != nil {
		msg := fmt.Sprintf("delegation to agent %q failed: %v", agentID, err)
		status := taskregistry.StatusFailed
		if errors.Is(err, context.DeadlineExceeded) {
			status = taskregistry.StatusTimedOut
		}
		t.recordDelegateTask(
			ctx, taskID, agentID, task, deliveryMode, boardMeta,
			status,
			taskregistry.DeliveryPending,
			msg,
			nil,
			nil,
		)
		return ErrorResult(fmt.Sprintf("delegation to agent %q failed: %v", agentID, err)).WithError(err)
	}
	if result == nil {
		msg := fmt.Sprintf("delegation to agent %q returned no result", agentID)
		t.recordDelegateTask(
			ctx, taskID, agentID, task, deliveryMode, boardMeta,
			taskregistry.StatusFailed,
			taskregistry.DeliveryPending,
			msg,
			nil,
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
		ctx, taskID, agentID, task, deliveryMode, boardMeta,
		taskregistry.StatusSucceeded,
		delegateDeliveryStatus(result, deliveryMode),
		result.ContentForLLM(),
		completionPayloadForTaskRegistry(result),
		deliverablePayloadForTaskRegistry(result),
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
	boardMeta TaskBoardMetadata,
	status taskregistry.Status,
	delivery taskregistry.DeliveryStatus,
	summary string,
	completion *taskregistry.CompletionPayload,
	deliverable *taskregistry.DeliverablePayload,
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
		Deliverable:         deliverable,
	}
	applyTaskBoardMetadata(&rec, boardMeta)
	if status == taskregistry.StatusRunning {
		rec.CreatedAt = now
		rec.StartedAt = now
	} else if existing, ok := t.taskRegistry.Get(taskID); ok {
		rec.CreatedAt = existing.CreatedAt
		rec.StartedAt = existing.StartedAt
	}
	if status == taskregistry.StatusFailed || status == taskregistry.StatusTimedOut {
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

func parseOptionalTimeoutSeconds(raw any) (time.Duration, error) {
	if raw == nil {
		return 0, nil
	}
	var seconds float64
	switch v := raw.(type) {
	case int:
		seconds = float64(v)
	case int64:
		seconds = float64(v)
	case float64:
		seconds = v
	case float32:
		seconds = float64(v)
	default:
		return 0, fmt.Errorf("timeout_seconds must be a positive number")
	}
	if seconds <= 0 {
		return 0, fmt.Errorf("timeout_seconds must be a positive number")
	}
	return time.Duration(seconds * float64(time.Second)), nil
}
