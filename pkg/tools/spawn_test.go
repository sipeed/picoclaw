package tools

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/providers"
)

type blockingProvider struct {
	block <-chan struct{}
}

func (p *blockingProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	options map[string]any,
) (*providers.LLMResponse, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-p.block:
		return &providers.LLMResponse{Content: "done"}, nil
	}
}

func (p *blockingProvider) GetDefaultModel() string {
	return "blocking-model"
}

func extractSpawnTaskID(msg string) (string, error) {
	marker := "(id: "
	idx := strings.LastIndex(msg, marker)
	if idx < 0 {
		return "", fmt.Errorf("task id marker not found")
	}
	rest := msg[idx+len(marker):]
	end := strings.Index(rest, ")")
	if end < 0 {
		return "", fmt.Errorf("task id closing bracket not found")
	}
	id := strings.TrimSpace(rest[:end])
	if id == "" {
		return "", fmt.Errorf("empty task id")
	}
	return id, nil
}

func TestSpawnTool_Execute_EmptyTask(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, "test-model", "/tmp/test", nil)
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
	manager := NewSubagentManager(provider, "test-model", "/tmp/test", nil)
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

func TestSpawnTool_Execute_RespectsMaxTaskLimit(t *testing.T) {
	block := make(chan struct{})
	provider := &blockingProvider{block: block}
	manager := NewSubagentManager(provider, "test-model", "/tmp/test", nil)
	manager.SetLimits(0, 1, 0)
	tool := NewSpawnTool(manager)

	ctx := context.Background()

	first := tool.Execute(ctx, map[string]any{"task": "task-1"})
	if first == nil || first.IsError {
		t.Fatalf("first spawn should succeed, got %+v", first)
	}

	second := tool.Execute(ctx, map[string]any{"task": "task-2"})
	if second == nil {
		t.Fatal("second spawn result should not be nil")
	}
	if !second.IsError {
		t.Fatalf("second spawn should fail due to max task limit, got %+v", second)
	}
	if !strings.Contains(second.ForLLM, "max task limit reached") {
		t.Fatalf("unexpected error message: %s", second.ForLLM)
	}

	// Unblock the first task to avoid goroutine leak.
	close(block)
	time.Sleep(10 * time.Millisecond)
}

func TestSpawnTool_Execute_RespectsMaxDepth(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, "test-model", "/tmp/test", nil)
	manager.SetLimits(0, 0, 1)
	tool := NewSpawnTool(manager)

	registry := NewToolRegistry()
	registry.Register(tool)

	result := registry.ExecuteWithContext(
		context.Background(),
		"spawn",
		map[string]any{"task": "nested task"},
		"cli",
		"direct",
		"subagent:parent-task",
		nil,
	)
	if result == nil {
		t.Fatal("result should not be nil")
	}
	if !result.IsError {
		t.Fatalf("expected depth-limit error, got %+v", result)
	}
	if !strings.Contains(result.ForLLM, "max spawn depth reached") {
		t.Fatalf("unexpected error message: %s", result.ForLLM)
	}
}

func TestSpawnTool_Execute_SetsParentTaskIDAndDepth(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, "test-model", "/tmp/test", nil)
	tool := NewSpawnTool(manager)

	registry := NewToolRegistry()
	registry.Register(tool)

	parent := registry.ExecuteWithContext(
		context.Background(),
		"spawn",
		map[string]any{"task": "parent task"},
		"cli",
		"direct",
		"",
		nil,
	)
	if parent == nil || parent.IsError {
		t.Fatalf("parent spawn failed: %+v", parent)
	}
	parentID, err := extractSpawnTaskID(parent.ForLLM)
	if err != nil {
		t.Fatalf("extract parent id failed: %v", err)
	}

	child := registry.ExecuteWithContext(
		context.Background(),
		"spawn",
		map[string]any{"task": "child task"},
		"cli",
		"direct",
		"subagent:"+parentID,
		nil,
	)
	if child == nil || child.IsError {
		t.Fatalf("child spawn failed: %+v", child)
	}
	childID, err := extractSpawnTaskID(child.ForLLM)
	if err != nil {
		t.Fatalf("extract child id failed: %v", err)
	}

	childTask, ok := manager.GetTask(childID)
	if !ok {
		t.Fatalf("child task %q not found", childID)
	}
	if childTask.ParentTaskID != parentID {
		t.Fatalf("child ParentTaskID = %q, want %q", childTask.ParentTaskID, parentID)
	}
	if childTask.Depth != 2 {
		t.Fatalf("child depth = %d, want 2", childTask.Depth)
	}
}

func TestSubagentManager_CancelTaskTree(t *testing.T) {
	manager := NewSubagentManager(&MockLLMProvider{}, "test-model", "/tmp/test", nil)

	parentCtx, parentCancel := context.WithCancel(context.Background())
	defer parentCancel()
	childCtx, childCancel := context.WithCancel(context.Background())
	defer childCancel()
	grandCtx, grandCancel := context.WithCancel(context.Background())
	defer grandCancel()
	otherCtx, otherCancel := context.WithCancel(context.Background())
	defer otherCancel()

	manager.mu.Lock()
	manager.tasks["parent"] = &SubagentTask{ID: "parent", Status: "running"}
	manager.tasks["child"] = &SubagentTask{ID: "child", ParentTaskID: "parent", Status: "running"}
	manager.tasks["grand"] = &SubagentTask{ID: "grand", ParentTaskID: "child", Status: "running"}
	manager.tasks["other"] = &SubagentTask{ID: "other", ParentTaskID: "", Status: "running"}
	manager.taskCancels["parent"] = parentCancel
	manager.taskCancels["child"] = childCancel
	manager.taskCancels["grand"] = grandCancel
	manager.taskCancels["other"] = otherCancel
	manager.mu.Unlock()

	manager.cancelTaskTree("parent")

	select {
	case <-childCtx.Done():
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected child context to be cancelled")
	}
	select {
	case <-grandCtx.Done():
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected grandchild context to be cancelled")
	}
	select {
	case <-otherCtx.Done():
		t.Fatal("unrelated task should not be cancelled")
	default:
	}
	select {
	case <-parentCtx.Done():
		t.Fatal("parent task is not cancelled by cancelTaskTree")
	default:
	}
}
