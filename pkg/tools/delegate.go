package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/routing"
)

// DelegateTool delegates a task to a specific named agent and waits for
// the result. Unlike spawn (async, fire-and-forget) or subagent (sync but
// generic), delegate targets a named agent and runs the task using that
// agent's own workspace, model, and tools.
type DelegateTool struct {
	spawner        SubTurnSpawner
	allowlistCheck func(targetAgentID string) bool
	selfAgentID    string
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

	if t.selfAgentID != "" && agentID == t.selfAgentID {
		return ErrorResult("cannot delegate to self")
	}

	if t.allowlistCheck != nil && !t.allowlistCheck(agentID) {
		return ErrorResult(fmt.Sprintf("not allowed to delegate to agent %q", agentID))
	}

	if t.spawner == nil {
		return ErrorResult("delegate tool not configured")
	}

	result, err := t.spawner.SpawnSubTurn(ctx, SubTurnConfig{
		TargetAgentID: agentID,
		SystemPrompt:  task,
		Async:         false,
	})
	if err != nil {
		return ErrorResult(fmt.Sprintf("delegation to agent %q failed: %v", agentID, err)).WithError(err)
	}
	if result == nil {
		return ErrorResult(fmt.Sprintf("delegation to agent %q returned no result", agentID))
	}

	result.ForLLM = fmt.Sprintf("[Response from agent %q]\n%s", agentID, result.ForLLM)

	return result
}
