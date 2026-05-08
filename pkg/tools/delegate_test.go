package tools

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// delegateMockSpawner records the config and returns a canned result.
type delegateMockSpawner struct {
	lastCfg SubTurnConfig
	result  *ToolResult
	err     error
}

func (m *delegateMockSpawner) SpawnSubTurn(_ context.Context, cfg SubTurnConfig) (*ToolResult, error) {
	m.lastCfg = cfg
	if m.err != nil {
		return nil, m.err
	}
	if m.result != nil {
		return m.result, nil
	}
	return &ToolResult{
		ForLLM:  "completed: " + cfg.SystemPrompt,
		ForUser: "completed",
	}, nil
}

func TestDelegateTool_Name(t *testing.T) {
	tool := NewDelegateTool()
	if tool.Name() != "delegate" {
		t.Errorf("Name() = %q, want %q", tool.Name(), "delegate")
	}
}

func TestDelegateTool_Parameters(t *testing.T) {
	tool := NewDelegateTool()
	params := tool.Parameters()

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("properties should be a map")
	}
	_, hasAgentID := props["agent_id"]
	if !hasAgentID {
		t.Error("agent_id parameter should exist")
	}
	_, hasTask := props["task"]
	if !hasTask {
		t.Error("task parameter should exist")
	}

	required, ok := params["required"].([]string)
	if !ok {
		t.Fatal("required should be a string array")
	}
	if len(required) != 2 {
		t.Fatalf("required should have 2 entries, got %d", len(required))
	}
}

func TestDelegateTool_Execute_Success(t *testing.T) {
	spawner := &delegateMockSpawner{}
	tool := NewDelegateTool()
	tool.SetSpawner(spawner)

	result := tool.Execute(context.Background(), map[string]any{
		"agent_id": "researcher",
		"task":     "summarize the logs",
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, `[Response from agent "researcher"]`) {
		t.Errorf("result should contain attribution, got: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "summarize the logs") {
		t.Errorf("result should contain task output, got: %s", result.ForLLM)
	}

	// Verify spawner received correct config
	if spawner.lastCfg.TargetAgentID != "researcher" {
		t.Errorf("TargetAgentID = %q, want %q", spawner.lastCfg.TargetAgentID, "researcher")
	}
	if spawner.lastCfg.Async {
		t.Error("delegate should be synchronous (Async=false)")
	}
	if spawner.lastCfg.SystemPrompt != "summarize the logs" {
		t.Errorf("SystemPrompt = %q, want %q", spawner.lastCfg.SystemPrompt, "summarize the logs")
	}
}

func TestDelegateTool_Execute_EmptyAgentID(t *testing.T) {
	tests := []struct {
		name string
		args map[string]any
	}{
		{"missing", map[string]any{"task": "test"}},
		{"empty string", map[string]any{"agent_id": "", "task": "test"}},
		{"whitespace only", map[string]any{"agent_id": "  ", "task": "test"}},
		{"wrong type", map[string]any{"agent_id": 123, "task": "test"}},
	}

	tool := NewDelegateTool()
	tool.SetSpawner(&delegateMockSpawner{})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tool.Execute(context.Background(), tt.args)
			if !result.IsError {
				t.Error("expected error for invalid agent_id")
			}
			if !strings.Contains(result.ForLLM, "agent_id is required") {
				t.Errorf("error should mention agent_id, got: %s", result.ForLLM)
			}
		})
	}
}

func TestDelegateTool_Execute_EmptyTask(t *testing.T) {
	tests := []struct {
		name string
		args map[string]any
	}{
		{"missing", map[string]any{"agent_id": "a"}},
		{"empty string", map[string]any{"agent_id": "a", "task": ""}},
		{"whitespace only", map[string]any{"agent_id": "a", "task": "\t\n"}},
	}

	tool := NewDelegateTool()
	tool.SetSpawner(&delegateMockSpawner{})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tool.Execute(context.Background(), tt.args)
			if !result.IsError {
				t.Error("expected error for invalid task")
			}
			if !strings.Contains(result.ForLLM, "task is required") {
				t.Errorf("error should mention task, got: %s", result.ForLLM)
			}
		})
	}
}

func TestDelegateTool_Execute_PermissionDenied(t *testing.T) {
	tool := NewDelegateTool()
	tool.SetSpawner(&delegateMockSpawner{})
	tool.SetAllowlistChecker(func(targetAgentID string) bool {
		return targetAgentID == "allowed-agent"
	})

	result := tool.Execute(context.Background(), map[string]any{
		"agent_id": "forbidden-agent",
		"task":     "test",
	})

	if !result.IsError {
		t.Error("expected error for denied agent")
	}
	if !strings.Contains(result.ForLLM, "not allowed to delegate") {
		t.Errorf("error should mention permission, got: %s", result.ForLLM)
	}
}

func TestDelegateTool_Execute_PermissionAllowed(t *testing.T) {
	tool := NewDelegateTool()
	tool.SetSpawner(&delegateMockSpawner{})
	tool.SetAllowlistChecker(func(targetAgentID string) bool {
		return targetAgentID == "allowed-agent"
	})

	result := tool.Execute(context.Background(), map[string]any{
		"agent_id": "allowed-agent",
		"task":     "test",
	})

	if result.IsError {
		t.Errorf("expected success for allowed agent, got error: %s", result.ForLLM)
	}
}

func TestDelegateTool_Execute_NoSpawner(t *testing.T) {
	tool := NewDelegateTool()

	result := tool.Execute(context.Background(), map[string]any{
		"agent_id": "a",
		"task":     "test",
	})

	if !result.IsError {
		t.Error("expected error when spawner is nil")
	}
	if !strings.Contains(result.ForLLM, "not configured") {
		t.Errorf("error should mention not configured, got: %s", result.ForLLM)
	}
}

func TestDelegateTool_Execute_SpawnerError(t *testing.T) {
	spawner := &delegateMockSpawner{
		err: fmt.Errorf("context deadline exceeded"),
	}
	tool := NewDelegateTool()
	tool.SetSpawner(spawner)

	result := tool.Execute(context.Background(), map[string]any{
		"agent_id": "researcher",
		"task":     "test",
	})

	if !result.IsError {
		t.Error("expected error when spawner fails")
	}
	if !strings.Contains(result.ForLLM, "delegation to agent") {
		t.Errorf("error should mention delegation failure, got: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "context deadline exceeded") {
		t.Errorf("error should propagate cause, got: %s", result.ForLLM)
	}
}

func TestDelegateTool_Execute_NoAllowlistCheck(t *testing.T) {
	// When no allowlist checker is set, all agents are allowed
	tool := NewDelegateTool()
	tool.SetSpawner(&delegateMockSpawner{})

	result := tool.Execute(context.Background(), map[string]any{
		"agent_id": "any-agent",
		"task":     "test",
	})

	if result.IsError {
		t.Errorf("expected success without allowlist, got error: %s", result.ForLLM)
	}
}

func TestDelegateTool_Execute_NilResult(t *testing.T) {
	tool := NewDelegateTool()
	tool.SetSpawner(&nilResultSpawner{})

	result := tool.Execute(context.Background(), map[string]any{
		"agent_id": "researcher",
		"task":     "test",
	})

	if !result.IsError {
		t.Error("expected error for nil result")
	}
	if !strings.Contains(result.ForLLM, "returned no result") {
		t.Errorf("error should mention no result, got: %s", result.ForLLM)
	}
}

func TestDelegateTool_Execute_SelfDelegation(t *testing.T) {
	tool := NewDelegateTool()
	tool.SetSpawner(&delegateMockSpawner{})
	tool.SetSelfAgentID("alpha")

	result := tool.Execute(context.Background(), map[string]any{
		"agent_id": "alpha",
		"task":     "test",
	})

	if !result.IsError {
		t.Error("expected error for self-delegation")
	}
	if !strings.Contains(result.ForLLM, "cannot delegate to self") {
		t.Errorf("error should mention self-delegation, got: %s", result.ForLLM)
	}
}

func TestDelegateTool_Execute_SelfDelegation_Normalized(t *testing.T) {
	tool := NewDelegateTool()
	tool.SetSpawner(&delegateMockSpawner{})
	tool.SetSelfAgentID("alpha") // stored normalized

	// Case-insensitive and whitespace variants should still be caught
	variants := []string{"ALPHA", " Alpha ", "  alpha  "}
	for _, v := range variants {
		t.Run(v, func(t *testing.T) {
			result := tool.Execute(context.Background(), map[string]any{
				"agent_id": v,
				"task":     "test",
			})
			if !result.IsError {
				t.Errorf("agent_id=%q should be caught as self-delegation", v)
			}
		})
	}
}

// nilResultSpawner always returns (nil, nil).
type nilResultSpawner struct{}

func (m *nilResultSpawner) SpawnSubTurn(_ context.Context, _ SubTurnConfig) (*ToolResult, error) {
	return nil, nil
}
