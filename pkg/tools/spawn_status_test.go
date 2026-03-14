package tools

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestSpawnStatusTool_Name(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, "test-model", "/tmp/test")
	tool := NewSpawnStatusTool(manager)

	if tool.Name() != "spawn_status" {
		t.Errorf("Expected name 'spawn_status', got '%s'", tool.Name())
	}
}

func TestSpawnStatusTool_Description(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, "test-model", "/tmp/test")
	tool := NewSpawnStatusTool(manager)

	desc := tool.Description()
	if desc == "" {
		t.Error("Description should not be empty")
	}
	if !strings.Contains(strings.ToLower(desc), "subagent") {
		t.Errorf("Description should mention 'subagent', got: %s", desc)
	}
}

func TestSpawnStatusTool_Parameters(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, "test-model", "/tmp/test")
	tool := NewSpawnStatusTool(manager)

	params := tool.Parameters()
	if params["type"] != "object" {
		t.Errorf("Expected type 'object', got: %v", params["type"])
	}
	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("Expected 'properties' to be a map")
	}
	if _, hasTaskID := props["task_id"]; !hasTaskID {
		t.Error("Expected 'task_id' parameter in properties")
	}
}

func TestSpawnStatusTool_NilManager(t *testing.T) {
	tool := &SpawnStatusTool{manager: nil}
	result := tool.Execute(context.Background(), map[string]any{})
	if !result.IsError {
		t.Error("Expected error result when manager is nil")
	}
}

func TestSpawnStatusTool_Empty(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, "test-model", "/tmp/test")
	tool := NewSpawnStatusTool(manager)

	result := tool.Execute(context.Background(), map[string]any{})
	if result.IsError {
		t.Fatalf("Expected success, got error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "No subagents") {
		t.Errorf("Expected 'No subagents' message, got: %s", result.ForLLM)
	}
}

func TestSpawnStatusTool_ListAll(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, "test-model", "/tmp/test")

	now := time.Now().UnixMilli()
	manager.mu.Lock()
	manager.tasks["subagent-1"] = &SubagentTask{
		ID:      "subagent-1",
		Task:    "Do task A",
		Label:   "task-a",
		Status:  "running",
		Created: now,
	}
	manager.tasks["subagent-2"] = &SubagentTask{
		ID:      "subagent-2",
		Task:    "Do task B",
		Label:   "task-b",
		Status:  "completed",
		Result:  "Done successfully",
		Created: now,
	}
	manager.tasks["subagent-3"] = &SubagentTask{
		ID:     "subagent-3",
		Task:   "Do task C",
		Status: "failed",
		Result: "Error: something went wrong",
	}
	manager.mu.Unlock()

	tool := NewSpawnStatusTool(manager)
	result := tool.Execute(context.Background(), map[string]any{})

	if result.IsError {
		t.Fatalf("Expected success, got error: %s", result.ForLLM)
	}

	// Summary header
	if !strings.Contains(result.ForLLM, "3 total") {
		t.Errorf("Expected total count in header, got: %s", result.ForLLM)
	}

	// Individual task IDs
	for _, id := range []string{"subagent-1", "subagent-2", "subagent-3"} {
		if !strings.Contains(result.ForLLM, id) {
			t.Errorf("Expected task %s in output, got:\n%s", id, result.ForLLM)
		}
	}

	// Status values
	for _, status := range []string{"running", "completed", "failed"} {
		if !strings.Contains(result.ForLLM, status) {
			t.Errorf("Expected status '%s' in output, got:\n%s", status, result.ForLLM)
		}
	}

	// Result content
	if !strings.Contains(result.ForLLM, "Done successfully") {
		t.Errorf("Expected result text in output, got:\n%s", result.ForLLM)
	}
}

func TestSpawnStatusTool_GetByID(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, "test-model", "/tmp/test")

	manager.mu.Lock()
	manager.tasks["subagent-42"] = &SubagentTask{
		ID:      "subagent-42",
		Task:    "Specific task",
		Label:   "my-task",
		Status:  "failed",
		Result:  "Something went wrong",
		Created: time.Now().UnixMilli(),
	}
	manager.mu.Unlock()

	tool := NewSpawnStatusTool(manager)
	result := tool.Execute(context.Background(), map[string]any{"task_id": "subagent-42"})

	if result.IsError {
		t.Fatalf("Expected success, got error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "subagent-42") {
		t.Errorf("Expected task ID in output, got: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "failed") {
		t.Errorf("Expected status 'failed' in output, got: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "Something went wrong") {
		t.Errorf("Expected result text in output, got: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "my-task") {
		t.Errorf("Expected label in output, got: %s", result.ForLLM)
	}
}

func TestSpawnStatusTool_GetByID_NotFound(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, "test-model", "/tmp/test")
	tool := NewSpawnStatusTool(manager)

	result := tool.Execute(context.Background(), map[string]any{"task_id": "nonexistent-999"})
	if !result.IsError {
		t.Errorf("Expected error for nonexistent task, got: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "nonexistent-999") {
		t.Errorf("Expected task ID in error message, got: %s", result.ForLLM)
	}
}

func TestSpawnStatusTool_ResultTruncation(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, "test-model", "/tmp/test")

	longResult := strings.Repeat("X", 500)
	manager.mu.Lock()
	manager.tasks["subagent-1"] = &SubagentTask{
		ID:     "subagent-1",
		Task:   "Long task",
		Status: "completed",
		Result: longResult,
	}
	manager.mu.Unlock()

	tool := NewSpawnStatusTool(manager)
	result := tool.Execute(context.Background(), map[string]any{"task_id": "subagent-1"})

	if result.IsError {
		t.Fatalf("Unexpected error: %s", result.ForLLM)
	}
	// Output should be shorter than the raw result due to truncation
	if len(result.ForLLM) >= len(longResult) {
		t.Errorf("Expected result to be truncated, but ForLLM is %d chars", len(result.ForLLM))
	}
	if !strings.Contains(result.ForLLM, "…") {
		t.Errorf("Expected truncation indicator '…' in output, got: %s", result.ForLLM)
	}
}

func TestSpawnStatusTool_StatusCounts(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, "test-model", "/tmp/test")

	manager.mu.Lock()
	for i, status := range []string{"running", "running", "completed", "failed", "canceled"} {
		id := fmt.Sprintf("subagent-%d", i+1)
		manager.tasks[id] = &SubagentTask{ID: id, Task: "t", Status: status}
	}
	manager.mu.Unlock()

	tool := NewSpawnStatusTool(manager)
	result := tool.Execute(context.Background(), map[string]any{})

	if result.IsError {
		t.Fatalf("Unexpected error: %s", result.ForLLM)
	}
	// The summary line should mention all statuses that have counts
	for _, want := range []string{"Running:", "Completed:", "Failed:", "Canceled:"} {
		if !strings.Contains(result.ForLLM, want) {
			t.Errorf("Expected %q in summary, got:\n%s", want, result.ForLLM)
		}
	}
}
