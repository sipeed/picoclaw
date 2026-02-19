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

func (d *dummyTool) Name() string                       { return d.name }
func (d *dummyTool) Description() string                { return "test tool" }
func (d *dummyTool) Parameters() map[string]interface{} { return nil }
func (d *dummyTool) Execute(_ context.Context, _ map[string]interface{}) *ToolResult {
	return NewToolResult("ok")
}

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

// TestToolHook_AfterExecuteRunsForAllHooksOnBlock verifies that when a BeforeExecute
// hook blocks execution, AfterExecute is still invoked on ALL registered hooks
// (not just the blocking one) for observability purposes.
func TestToolHook_AfterExecuteRunsForAllHooksOnBlock(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(&dummyTool{name: "observed_tool"})

	hook1 := &testHook{blockTool: "observed_tool"}
	hook2 := &testHook{} // does not block, but should still get AfterExecute
	reg.AddHook(hook1)
	reg.AddHook(hook2)

	result := reg.Execute(context.Background(), "observed_tool", nil)

	if !result.IsError {
		t.Error("expected error result when hook1 blocks")
	}
	// BeforeExecute: hook1 called, hook2 NOT called (chain stopped)
	if len(hook1.beforeCalls) != 1 {
		t.Errorf("hook1.beforeCalls = %d, want 1", len(hook1.beforeCalls))
	}
	if len(hook2.beforeCalls) != 0 {
		t.Errorf("hook2.beforeCalls = %d, want 0 (chain stopped)", len(hook2.beforeCalls))
	}
	// AfterExecute: BOTH hooks called (inner loop over all hooks for observability)
	if len(hook1.afterCalls) != 1 {
		t.Errorf("hook1.afterCalls = %d, want 1", len(hook1.afterCalls))
	}
	if len(hook2.afterCalls) != 1 {
		t.Errorf("hook2.afterCalls = %d, want 1 (AfterExecute runs for all hooks even on block)", len(hook2.afterCalls))
	}
}

// TestToolHook_NotFoundToolSkipsHooks verifies that hooks are not called when
// a tool does not exist in the registry.
func TestToolHook_NotFoundToolSkipsHooks(t *testing.T) {
	reg := NewToolRegistry()
	// Do NOT register the tool

	hook := &testHook{}
	reg.AddHook(hook)

	result := reg.Execute(context.Background(), "ghost_tool", nil)

	if !result.IsError {
		t.Error("expected error for unknown tool")
	}
	// Hooks should not be called when the tool doesn't exist (early return before hook loop)
	if len(hook.beforeCalls) != 0 {
		t.Errorf("hook.beforeCalls = %d, want 0 for missing tool", len(hook.beforeCalls))
	}
	if len(hook.afterCalls) != 0 {
		t.Errorf("hook.afterCalls = %d, want 0 for missing tool", len(hook.afterCalls))
	}
}

// TestToolHook_NoHooksSucceeds verifies that a tool executes normally with no hooks registered.
func TestToolHook_NoHooksSucceeds(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(&dummyTool{name: "plain_tool"})
	// No hooks added

	result := reg.Execute(context.Background(), "plain_tool", nil)

	if result.IsError {
		t.Errorf("expected success with no hooks, got error: %s", result.ForLLM)
	}
}
