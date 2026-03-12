package tools

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestSpawnTool_Execute_EmptyTask(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, "test-model", "/tmp/test")
	tool := NewSpawnTool(manager)

	ctx := context.Background()

	tests := []struct {
		name string
		args map[string]any
	}{
		{"empty string", map[string]any{"task": ""}},
		{"whitespace only", map[string]any{"task": "   "}},
		{"tabs and newlines", map[string]any{"task": "\t\n  "}},
		{"missing task key", map[string]any{"label": "test"}},
		{"wrong type", map[string]any{"task": 123}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tool.Execute(ctx, tt.args)
			if result == nil {
				t.Fatal("Result should not be nil")
			}
			if !result.IsError {
				t.Error("Expected error for invalid task parameter")
			}
			if !strings.Contains(result.ForLLM, "task is required") {
				t.Errorf("Error message should mention 'task is required', got: %s", result.ForLLM)
			}
		})
	}
}

func TestSpawnTool_Execute_ValidTask(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, "test-model", "/tmp/test")
	tool := NewSpawnTool(manager)

	ctx := context.Background()
	args := map[string]any{
		"task":  "Write a haiku about coding",
		"label": "haiku-task",
	}

	result := tool.Execute(ctx, args)
	if result == nil {
		t.Fatal("Result should not be nil")
	}
	if result.IsError {
		t.Errorf("Expected success for valid task, got error: %s", result.ForLLM)
	}
	if !result.Async {
		t.Error("SpawnTool should return async result")
	}
}

func TestSpawnTool_Execute_NilManager(t *testing.T) {
	tool := NewSpawnTool(nil)

	ctx := context.Background()
	args := map[string]any{"task": "test task"}

	result := tool.Execute(ctx, args)
	if !result.IsError {
		t.Error("Expected error for nil manager")
	}
	if !strings.Contains(result.ForLLM, "Subagent manager not configured") {
		t.Errorf("Error message should mention manager not configured, got: %s", result.ForLLM)
	}
}

func TestSpawnTool_ExecuteAsync_UsesTargetAgentModel(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, "caller-model", "/tmp/test")
	manager.SetAgentModelResolver(func(agentID string) (string, bool) {
		if agentID == "analyst" {
			return "target-model", true
		}
		return "", false
	})
	tool := NewSpawnTool(manager)

	done := make(chan struct{})
	ctx := WithToolContext(context.Background(), "cli", "direct")
	args := map[string]any{
		"task":     "Write a haiku about coding",
		"agent_id": "analyst",
	}

	result := tool.ExecuteAsync(ctx, args, func(context.Context, *ToolResult) {
		close(done)
	})
	if result == nil {
		t.Fatal("Result should not be nil")
	}
	if result.IsError {
		t.Fatalf("Expected success for valid task, got error: %s", result.ForLLM)
	}
	if !result.Async {
		t.Fatal("SpawnTool should return async result")
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("spawn callback was not invoked")
	}

	if provider.lastModel != "target-model" {
		t.Fatalf("lastModel = %q, want %q", provider.lastModel, "target-model")
	}
}
