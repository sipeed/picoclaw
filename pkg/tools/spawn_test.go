package tools

import (
	"context"
	"strings"
	"testing"
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

// mockTool implements Tool for testing tool filtering.
type mockTool struct {
	name string
}

func (m *mockTool) Name() string            { return m.name }
func (m *mockTool) Description() string     { return "mock tool: " + m.name }
func (m *mockTool) Parameters() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{}}
}
func (m *mockTool) Execute(_ context.Context, _ map[string]any) *ToolResult {
	return &ToolResult{ForLLM: "mock result"}
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

func TestSpawnTool_AllowDirectMessage_False_FiltersMessageTool(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, "test-model", "/tmp/test")

	// Register a mock message tool and a regular tool
	manager.RegisterTool(&mockTool{name: "message"})
	manager.RegisterTool(&mockTool{name: "web_search"})

	tool := NewSpawnTool(manager)
	spawner := &mockSpawner{done: make(chan struct{})}
	tool.SetSpawner(spawner)

	ctx := context.Background()
	args := map[string]any{
		"task":                 "Search for news",
		"allow_direct_message": false,
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

	// Verify message tool is filtered out
	hasMessage := false
	hasWebSearch := false
	for _, t := range spawner.lastConfig.Tools {
		if t.Name() == "message" {
			hasMessage = true
		}
		if t.Name() == "web_search" {
			hasWebSearch = true
		}
	}
	if hasMessage {
		t.Error("message tool should be filtered out when allow_direct_message=false")
	}
	if !hasWebSearch {
		t.Error("web_search tool should still be present when allow_direct_message=false")
	}
}

func TestSpawnTool_AllowDirectMessage_True_KeepsMessageTool(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, "test-model", "/tmp/test")

	manager.RegisterTool(&mockTool{name: "message"})
	manager.RegisterTool(&mockTool{name: "web_search"})

	tool := NewSpawnTool(manager)
	spawner := &mockSpawner{done: make(chan struct{})}
	tool.SetSpawner(spawner)

	ctx := context.Background()
	args := map[string]any{
		"task":                 "Search for news",
		"allow_direct_message": true,
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

	hasMessage := false
	hasWebSearch := false
	for _, t := range spawner.lastConfig.Tools {
		if t.Name() == "message" {
			hasMessage = true
		}
		if t.Name() == "web_search" {
			hasWebSearch = true
		}
	}
	if !hasMessage {
		t.Error("message tool should be present when allow_direct_message=true")
	}
	if !hasWebSearch {
		t.Error("web_search tool should be present when allow_direct_message=true")
	}
}

func TestSpawnTool_AllowDirectMessage_Default_True_KeepsMessageTool(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, "test-model", "/tmp/test")

	manager.RegisterTool(&mockTool{name: "message"})
	manager.RegisterTool(&mockTool{name: "web_search"})

	tool := NewSpawnTool(manager)
	spawner := &mockSpawner{done: make(chan struct{})}
	tool.SetSpawner(spawner)

	ctx := context.Background()
	args := map[string]any{
		"task": "Search for news",
		// allow_direct_message not specified, should default to true
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

	hasMessage := false
	for _, t := range spawner.lastConfig.Tools {
		if t.Name() == "message" {
			hasMessage = true
		}
	}
	if !hasMessage {
		t.Error("message tool should be present by default (allow_direct_message defaults to true)")
	}
}
