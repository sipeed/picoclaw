package tools

import (
	"context"
	"strings"
	"testing"
	"time"
)

// mockSpawner implements SubTurnSpawner for testing
type mockSpawner struct{}

func (m *mockSpawner) SpawnSubTurn(ctx context.Context, cfg SubTurnConfig) (*ToolResult, error) {
	// Extract task from system prompt for response
	task := cfg.SystemPrompt
	if strings.Contains(task, "Task: ") {
		parts := strings.Split(task, "Task: ")
		if len(parts) > 1 {
			task = parts[1]
		}
	}
	return &ToolResult{
		ForLLM:  "Task completed: " + task,
		ForUser: "Task completed",
	}, nil
}

type managerSnapshotTool struct {
	name string
}

func (t *managerSnapshotTool) Name() string {
	return t.name
}

func (t *managerSnapshotTool) Description() string {
	return "test tool"
}

func (t *managerSnapshotTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
	}
}

func (t *managerSnapshotTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	return SilentResult("ok")
}

type recordingSpawner struct {
	toolNames []string
	done      chan struct{}
}

func (s *recordingSpawner) SpawnSubTurn(ctx context.Context, cfg SubTurnConfig) (*ToolResult, error) {
	for _, tool := range cfg.Tools {
		if tool != nil {
			s.toolNames = append(s.toolNames, tool.Name())
		}
	}
	if s.done != nil {
		close(s.done)
	}
	return &ToolResult{
		ForLLM:  "Task completed",
		ForUser: "Task completed",
	}, nil
}

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
	tool.SetSpawner(&mockSpawner{})

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

func TestSpawnTool_Execute_PassesManagerToolsToSubTurn(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, "test-model", "/tmp/test")
	manager.RegisterTool(&managerSnapshotTool{name: "snapshot_tool"})

	tool := NewSpawnTool(manager)
	spawner := &recordingSpawner{done: make(chan struct{})}
	tool.SetSpawner(spawner)

	result := tool.Execute(context.Background(), map[string]any{"task": "inspect tools"})
	if result == nil {
		t.Fatal("Result should not be nil")
	}
	if result.IsError {
		t.Fatalf("Expected success, got error: %s", result.ForLLM)
	}
	if !result.Async {
		t.Fatal("spawn result should be async")
	}

	select {
	case <-spawner.done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for async spawn execution")
	}

	if len(spawner.toolNames) != 1 || spawner.toolNames[0] != "snapshot_tool" {
		t.Fatalf("expected tool snapshot [snapshot_tool], got %v", spawner.toolNames)
	}
}
