package tools

import (
	"context"
	"strings"
	"testing"
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

func TestSubagentManager_ModelResolver(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, "default-model", "/tmp/test")

	// Set up model resolver
	resolvedAgentID := ""
	manager.SetModelResolver(func(agentID string) string {
		resolvedAgentID = agentID
		if agentID == "premium-agent" {
			return "gpt-4"
		}
		return ""
	})

	// Verify resolver is set
	if manager.modelResolver == nil {
		t.Fatal("Model resolver should be set")
	}

	// Test resolver is called with correct agent ID
	result := manager.modelResolver("premium-agent")
	if resolvedAgentID != "premium-agent" {
		t.Errorf("Expected resolver to be called with 'premium-agent', got '%s'", resolvedAgentID)
	}
	if result != "gpt-4" {
		t.Errorf("Expected 'gpt-4', got '%s'", result)
	}

	// Test fallback for unknown agent
	result = manager.modelResolver("unknown-agent")
	if result != "" {
		t.Errorf("Expected empty string for unknown agent, got '%s'", result)
	}
}
