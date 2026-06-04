package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	taskregistry "github.com/sipeed/picoclaw/pkg/tasks"
)

func TestTaskBoardExecuteNextTool_ExecutesDelegateBackedStep(t *testing.T) {
	registry := taskregistry.NewRegistry(taskregistry.WorkspaceStorePath(t.TempDir()))
	toolRegistry := NewToolRegistry()
	toolRegistry.Register(NewTaskBoardTool(registry))

	spawner := &delegateMockSpawner{}
	delegateTool := NewDelegateTool()
	delegateTool.SetSpawner(spawner)
	delegateTool.SetTaskRegistry(registry)
	toolRegistry.Register(delegateTool)

	executor := NewTaskBoardExecuteNextTool(registry, toolRegistry)
	toolRegistry.Register(executor)

	ctx := WithToolContext(context.Background(), "telegram", "chat-1")
	add := toolRegistry.ExecuteWithContext(ctx, "task_board", map[string]any{
		"action":         "add_step",
		"board_id":       "workflow-1",
		"step_id":        "research",
		"step_title":     "Research recipe",
		"owner":          "research",
		"task":           "Research the recipe context.",
		"execution_tool": "delegate",
	}, "telegram", "chat-1", nil)
	if add.IsError {
		t.Fatalf("add_step failed: %s", add.ForLLM)
	}

	result := toolRegistry.ExecuteWithContext(ctx, "task_board_execute_next", map[string]any{
		"board_id": "workflow-1",
	}, "telegram", "chat-1", nil)
	if result.IsError {
		t.Fatalf("execute_next failed: %s", result.ForLLM)
	}

	var payload struct {
		Action          string         `json:"action"`
		StepID          string         `json:"step_id"`
		Executed        bool           `json:"executed"`
		RecommendedTool string         `json:"recommended_tool"`
		DelegateArgs    map[string]any `json:"delegate_args"`
		Result          string         `json:"result"`
	}
	if err := json.Unmarshal([]byte(result.ForLLM), &payload); err != nil {
		t.Fatalf("execute_next JSON error = %v\n%s", err, result.ForLLM)
	}
	if payload.Action != "execute_next" ||
		payload.StepID != "research" ||
		!payload.Executed ||
		payload.RecommendedTool != "delegate" ||
		payload.DelegateArgs["agent_id"] != "research" ||
		!strings.Contains(payload.Result, `[Response from agent "research"]`) {
		t.Fatalf("unexpected execute payload: %+v\n%s", payload, result.ForLLM)
	}
	if spawner.lastCfg.TargetAgentID != "research" {
		t.Fatalf("delegate target = %q, want research", spawner.lastCfg.TargetAgentID)
	}
}

func TestTaskBoardExecuteNextTool_DoesNotAutoRunSpawnStep(t *testing.T) {
	registry := taskregistry.NewRegistry(taskregistry.WorkspaceStorePath(t.TempDir()))
	toolRegistry := NewToolRegistry()
	toolRegistry.Register(NewTaskBoardTool(registry))
	toolRegistry.Register(NewTaskBoardExecuteNextTool(registry, toolRegistry))

	ctx := WithToolContext(context.Background(), "telegram", "chat-1")
	add := toolRegistry.ExecuteWithContext(ctx, "task_board", map[string]any{
		"action":         "add_step",
		"board_id":       "workflow-1",
		"step_id":        "background",
		"step_title":     "Background work",
		"owner":          "research",
		"task":           "Do background work.",
		"execution_tool": "spawn",
	}, "telegram", "chat-1", nil)
	if add.IsError {
		t.Fatalf("add_step failed: %s", add.ForLLM)
	}

	result := toolRegistry.ExecuteWithContext(ctx, "task_board_execute_next", map[string]any{
		"board_id": "workflow-1",
		"step_id":  "background",
	}, "telegram", "chat-1", nil)
	if result.IsError {
		t.Fatalf("execute_next should return a non-executed plan, got error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, `"executed": false`) ||
		!strings.Contains(result.ForLLM, `"recommended_tool": "spawn"`) ||
		!strings.Contains(result.ForLLM, "not delegate-backed") {
		t.Fatalf("unexpected spawn execute response:\n%s", result.ForLLM)
	}
}
