package tools

import (
	"context"
	"strings"
	"testing"
	"time"
)

// mockSpawner implements SubTurnSpawner for testing.
type mockSpawner struct {
	lastConfig SubTurnConfig
	done       chan struct{}
}

func (m *mockSpawner) SpawnSubTurn(ctx context.Context, cfg SubTurnConfig) (*ToolResult, error) {
	m.lastConfig = cfg
	if m.done != nil {
		close(m.done)
	}

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
	spawner := &mockSpawner{done: make(chan struct{})}
	tool.SetSpawner(spawner)

	ctx := context.Background()
	args := map[string]any{
		"task":     "Write a haiku about coding",
		"label":    "haiku-task",
		"agent_id": "research",
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
	<-spawner.done
	if spawner.lastConfig.TargetAgentID != "research" {
		t.Errorf("TargetAgentID = %q, want research", spawner.lastConfig.TargetAgentID)
	}
	if !spawner.lastConfig.Critical {
		t.Error("SpawnTool should mark background subturns as critical")
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

func TestSpawnTool_SpawnStatusSeesSpawnedTask(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, "test-model", "/tmp/test")
	spawnTool := NewSpawnTool(manager)
	spawner := &mockSpawner{done: make(chan struct{})}
	spawnTool.SetSpawner(spawner)
	statusTool := NewSpawnStatusTool(manager)

	ctx := WithToolContext(context.Background(), "telegram", "chat-1")
	args := map[string]any{
		"task":     "Write a haiku about coding",
		"label":    "haiku-task",
		"agent_id": "deep-research",
	}

	result := spawnTool.Execute(ctx, args)
	if result == nil {
		t.Fatal("Result should not be nil")
	}
	if result.IsError {
		t.Fatalf("Expected success for valid task, got error: %s", result.ForLLM)
	}
	if !result.Async {
		t.Fatal("SpawnTool should return async result")
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		status := statusTool.Execute(ctx, map[string]any{})
		if status == nil {
			t.Fatal("status result should not be nil")
		}
		if status.IsError {
			t.Fatalf("spawn_status returned error: %s", status.ForLLM)
		}
		if strings.Contains(status.ForLLM, "subagent-1") {
			if !strings.Contains(status.ForLLM, "haiku-task") {
				t.Fatalf("expected label in status output, got: %s", status.ForLLM)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("spawn_status never observed spawned task; last output: %s", status.ForLLM)
		}
		time.Sleep(10 * time.Millisecond)
	}

	<-spawner.done
}

func TestSpawnTool_ExecuteAsync_MarksCallbackResultUserOnly(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, "test-model", "/tmp/test")
	tool := NewSpawnTool(manager)
	spawner := &mockSpawner{}
	tool.SetSpawner(spawner)

	done := make(chan *ToolResult, 1)
	result := tool.ExecuteAsync(context.Background(), map[string]any{
		"task": "Write a haiku about coding",
	}, func(_ context.Context, res *ToolResult) {
		done <- res
	})

	if result == nil || !result.Async {
		t.Fatal("expected async acknowledgment result")
	}

	select {
	case cbResult := <-done:
		if cbResult == nil {
			t.Fatal("expected callback result")
		}
		if cbResult.AsyncDelivery != AsyncDeliveryUserOnly {
			t.Fatalf("AsyncDelivery = %q, want %q", cbResult.AsyncDelivery, AsyncDeliveryUserOnly)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for spawn callback result")
	}
}

func TestSpawnTool_ExecuteAsync_RespectsExplicitDeliveryMode(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, "test-model", "/tmp/test")
	tool := NewSpawnTool(manager)
	spawner := &mockSpawner{}
	tool.SetSpawner(spawner)

	done := make(chan *ToolResult, 1)
	result := tool.ExecuteAsync(context.Background(), map[string]any{
		"task":          "Write a haiku about coding",
		"delivery_mode": string(AsyncDeliveryUserAndParent),
	}, func(_ context.Context, res *ToolResult) {
		done <- res
	})

	if result == nil || !result.Async {
		t.Fatal("expected async acknowledgment result")
	}

	select {
	case cbResult := <-done:
		if cbResult == nil {
			t.Fatal("expected callback result")
		}
		if cbResult.AsyncDelivery != AsyncDeliveryUserAndParent {
			t.Fatalf("AsyncDelivery = %q, want %q", cbResult.AsyncDelivery, AsyncDeliveryUserAndParent)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for spawn callback result")
	}
}

func TestSpawnTool_Execute_InvalidDeliveryMode(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, "test-model", "/tmp/test")
	tool := NewSpawnTool(manager)

	tests := []map[string]any{
		{"task": "test", "delivery_mode": 123},
		{"task": "test", "delivery_mode": "wrong"},
	}

	for _, args := range tests {
		result := tool.Execute(context.Background(), args)
		if result == nil {
			t.Fatal("expected result")
		}
		if !result.IsError {
			t.Fatalf("expected error for args=%v", args)
		}
		if !strings.Contains(result.ForLLM, "delivery_mode") {
			t.Fatalf("expected delivery_mode error, got: %s", result.ForLLM)
		}
	}
}
