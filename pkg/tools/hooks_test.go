package tools

import (
	"context"
	"errors"
	"testing"
)

// testHook records calls and optionally blocks execution.
type testHook struct {
	beforeCalls []string
	afterCalls  []string
	blockTool   string // if non-empty, block this tool name
}

func (h *testHook) BeforeExecute(_ context.Context, toolName string, _ map[string]interface{}) error {
	h.beforeCalls = append(h.beforeCalls, toolName)
	if h.blockTool != "" && toolName == h.blockTool {
		return errors.New("blocked by test hook")
	}
	return nil
}

func (h *testHook) AfterExecute(_ context.Context, toolName string, _ map[string]interface{}, _ *ToolResult) {
	h.afterCalls = append(h.afterCalls, toolName)
}

// dummyTool is a minimal tool for hook testing.
type dummyTool struct {
	name string
}

func (d *dummyTool) Name() string                                                  { return d.name }
func (d *dummyTool) Description() string                                            { return "test tool" }
func (d *dummyTool) Parameters() map[string]interface{}                             { return nil }
func (d *dummyTool) Execute(_ context.Context, _ map[string]interface{}) *ToolResult { return NewToolResult("ok") }

func TestToolHook_BeforeAndAfterCalled(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(&dummyTool{name: "test_tool"})

	hook := &testHook{}
	reg.AddHook(hook)

	reg.Execute(context.Background(), "test_tool", nil)

	if len(hook.beforeCalls) != 1 || hook.beforeCalls[0] != "test_tool" {
		t.Errorf("beforeCalls = %v, want [test_tool]", hook.beforeCalls)
	}
	if len(hook.afterCalls) != 1 || hook.afterCalls[0] != "test_tool" {
		t.Errorf("afterCalls = %v, want [test_tool]", hook.afterCalls)
	}
}

func TestToolHook_BlocksExecution(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(&dummyTool{name: "blocked_tool"})

	hook := &testHook{blockTool: "blocked_tool"}
	reg.AddHook(hook)

	result := reg.Execute(context.Background(), "blocked_tool", nil)

	if !result.IsError {
		t.Error("expected error result when hook blocks")
	}
	if len(hook.beforeCalls) != 1 {
		t.Errorf("beforeCalls count = %d, want 1", len(hook.beforeCalls))
	}
	// AfterExecute should still be called (for observability)
	if len(hook.afterCalls) != 1 {
		t.Errorf("afterCalls count = %d, want 1 (observability)", len(hook.afterCalls))
	}
}

func TestToolHook_MultipleHooks(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(&dummyTool{name: "multi"})

	hook1 := &testHook{}
	hook2 := &testHook{}
	reg.AddHook(hook1)
	reg.AddHook(hook2)

	reg.Execute(context.Background(), "multi", nil)

	if len(hook1.beforeCalls) != 1 || len(hook2.beforeCalls) != 1 {
		t.Error("expected both hooks to be called")
	}
}

func TestToolHook_FirstBlockStopsChain(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(&dummyTool{name: "chain_test"})

	hook1 := &testHook{blockTool: "chain_test"}
	hook2 := &testHook{}
	reg.AddHook(hook1)
	reg.AddHook(hook2)

	result := reg.Execute(context.Background(), "chain_test", nil)

	if !result.IsError {
		t.Error("expected error when first hook blocks")
	}
	// hook1 should have been called, hook2's Before should NOT
	if len(hook1.beforeCalls) != 1 {
		t.Error("hook1 before should have been called")
	}
	if len(hook2.beforeCalls) != 0 {
		t.Error("hook2 before should NOT have been called (chain stopped)")
	}
}
