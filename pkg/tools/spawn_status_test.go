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
	workspace := t.TempDir()
	manager := NewSubagentManager(provider, "test-model", workspace)
	tool := NewSpawnStatusTool(manager)

	if tool.Name() != "spawn_status" {
		t.Errorf("Expected name 'spawn_status', got '%s'", tool.Name())
	}
}

func TestSpawnStatusTool_Description(t *testing.T) {
	provider := &MockLLMProvider{}
	workspace := t.TempDir()
	manager := NewSubagentManager(provider, "test-model", workspace)
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
	workspace := t.TempDir()
	manager := NewSubagentManager(provider, "test-model", workspace)
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
	workspace := t.TempDir()
	manager := NewSubagentManager(provider, "test-model", workspace)
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
	workspace := t.TempDir()
	manager := NewSubagentManager(provider, "test-model", workspace)

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

func TestSpawnStatusTool_TaskID_NonString(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, "test-model", "/tmp/test")
	tool := NewSpawnStatusTool(manager)

	for _, badVal := range []any{42, 3.14, true, map[string]any{"x": 1}, []string{"a"}} {
		result := tool.Execute(context.Background(), map[string]any{"task_id": badVal})
		if !result.IsError {
			t.Errorf("Expected error for task_id=%T(%v), got success: %s", badVal, badVal, result.ForLLM)
		}
		if !strings.Contains(result.ForLLM, "task_id must be a string") {
			t.Errorf("Expected type-error message, got: %s", result.ForLLM)
		}
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

func TestSpawnStatusTool_ResultTruncation_Unicode(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, "test-model", "/tmp/test")

	// Each CJK rune is 3 bytes; 400 runes = 1200 bytes — well over the 300-rune limit.
	cjkChar := string(rune(0x5b57))
	longResult := strings.Repeat(cjkChar, 400)
	manager.mu.Lock()
	manager.tasks["subagent-1"] = &SubagentTask{
		ID:     "subagent-1",
		Task:   "Unicode task",
		Status: "completed",
		Result: longResult,
	}
	manager.mu.Unlock()

	tool := NewSpawnStatusTool(manager)
	result := tool.Execute(context.Background(), map[string]any{"task_id": "subagent-1"})

	if result.IsError {
		t.Fatalf("Unexpected error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "…") {
		t.Errorf("Expected truncation indicator in output")
	}
	// The truncated result must be valid UTF-8 (no split rune boundaries).
	if !strings.Contains(result.ForLLM, cjkChar) {
		t.Errorf("Expected CJK runes to appear intact in output")
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

func TestSpawnStatusTool_SortByCreatedTimestamp(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, "test-model", "/tmp/test")

	now := time.Now().UnixMilli()
	manager.mu.Lock()
	// Intentionally insert with out-of-order IDs and timestamps that reflect
	// true spawn order: subagent-2 was spawned first, subagent-10 second.
	manager.tasks["subagent-10"] = &SubagentTask{
		ID: "subagent-10", Task: "second", Status: "running",
		Created: now + 1,
	}
	manager.tasks["subagent-2"] = &SubagentTask{
		ID: "subagent-2", Task: "first", Status: "running",
		Created: now,
	}
	manager.mu.Unlock()

	tool := NewSpawnStatusTool(manager)
	result := tool.Execute(context.Background(), map[string]any{})

	if result.IsError {
		t.Fatalf("Unexpected error: %s", result.ForLLM)
	}

	pos2 := strings.Index(result.ForLLM, "subagent-2")
	pos10 := strings.Index(result.ForLLM, "subagent-10")
	if pos2 < 0 || pos10 < 0 {
		t.Fatalf("Both task IDs should appear in output:\n%s", result.ForLLM)
	}
	if pos2 > pos10 {
		t.Errorf("Expected subagent-2 (created first) to appear before subagent-10, but got:\n%s", result.ForLLM)
	}
}

func TestSpawnStatusTool_ChannelFiltering_ListAll(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, "test-model", "/tmp/test")

	manager.mu.Lock()
	manager.tasks["subagent-1"] = &SubagentTask{
		ID: "subagent-1", Task: "mine", Status: "running",
		OriginChannel: "telegram", OriginChatID: "chat-A",
	}
	manager.tasks["subagent-2"] = &SubagentTask{
		ID: "subagent-2", Task: "other user", Status: "running",
		OriginChannel: "telegram", OriginChatID: "chat-B",
	}
	manager.mu.Unlock()

	tool := NewSpawnStatusTool(manager)

	// Caller is chat-A — should only see subagent-1.
	ctx := WithToolContext(context.Background(), "telegram", "chat-A")
	result := tool.Execute(ctx, map[string]any{})

	if result.IsError {
		t.Fatalf("Unexpected error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "subagent-1") {
		t.Errorf("Expected own task in output, got:\n%s", result.ForLLM)
	}
	if strings.Contains(result.ForLLM, "subagent-2") {
		t.Errorf("Should NOT see other chat's task, got:\n%s", result.ForLLM)
	}
}

func TestSpawnStatusTool_ChannelFiltering_GetByID(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, "test-model", "/tmp/test")

	manager.mu.Lock()
	manager.tasks["subagent-99"] = &SubagentTask{
		ID: "subagent-99", Task: "secret", Status: "completed", Result: "private data",
		OriginChannel: "slack", OriginChatID: "room-Z",
	}
	manager.mu.Unlock()

	tool := NewSpawnStatusTool(manager)

	// Different chat trying to look up subagent-99 by ID.
	ctx := WithToolContext(context.Background(), "slack", "room-OTHER")
	result := tool.Execute(ctx, map[string]any{"task_id": "subagent-99"})

	if !result.IsError {
		t.Errorf("Expected error (cross-chat lookup blocked), got: %s", result.ForLLM)
	}
}

func TestSpawnStatusTool_ChannelFiltering_NoContext(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, "test-model", "/tmp/test")

	manager.mu.Lock()
	manager.tasks["subagent-1"] = &SubagentTask{
		ID: "subagent-1", Task: "t", Status: "completed",
		OriginChannel: "telegram", OriginChatID: "chat-A",
	}
	manager.mu.Unlock()

	tool := NewSpawnStatusTool(manager)

	// No ToolContext injected (e.g. a direct programmatic call that bypasses
	// WithToolContext entirely) — callerChannel and callerChatID are both "".
	// Note: the normal CLI path uses ProcessDirectWithChannel("cli", "direct"),
	// which *does* inject a non-empty context; this test covers the case where
	// no context injection happens at all.
	// The filter conditions require a non-empty caller value, so all tasks pass through.
	result := tool.Execute(context.Background(), map[string]any{})
	if result.IsError {
		t.Fatalf("Unexpected error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "subagent-1") {
		t.Errorf("Expected task visible from no-context caller, got:\n%s", result.ForLLM)
	}
}
