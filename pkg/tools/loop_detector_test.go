package tools

import (
	"context"
	"fmt"
	"testing"
)

// loopTestTool is a minimal tool that returns configurable results.
type loopTestTool struct {
	name   string
	result string
}

func (t *loopTestTool) Name() string                       { return t.name }
func (t *loopTestTool) Description() string                { return "loop test tool" }
func (t *loopTestTool) Parameters() map[string]interface{} { return nil }
func (t *loopTestTool) Execute(_ context.Context, _ map[string]interface{}) *ToolResult {
	return NewToolResult(t.result)
}

// --- Context key tests ---

func TestWithSessionKey(t *testing.T) {
	ctx := context.Background()
	if got := sessionKeyFromContext(ctx); got != "_default" {
		t.Errorf("expected _default, got %q", got)
	}

	ctx = WithSessionKey(ctx, "session-123")
	if got := sessionKeyFromContext(ctx); got != "session-123" {
		t.Errorf("expected session-123, got %q", got)
	}
}

// --- Hash tests ---

func TestHashArgs_Deterministic(t *testing.T) {
	args := map[string]interface{}{"path": "/tmp/file.txt", "content": "hello"}
	h1 := hashArgs(args)
	h2 := hashArgs(args)
	if h1 != h2 {
		t.Errorf("hashArgs not deterministic: %s != %s", h1, h2)
	}
}

func TestHashArgs_Empty(t *testing.T) {
	if got := hashArgs(nil); got != "empty" {
		t.Errorf("expected 'empty' for nil args, got %q", got)
	}
	if got := hashArgs(map[string]interface{}{}); got != "empty" {
		t.Errorf("expected 'empty' for empty args, got %q", got)
	}
}

func TestHashArgs_DifferentArgs(t *testing.T) {
	h1 := hashArgs(map[string]interface{}{"a": "1"})
	h2 := hashArgs(map[string]interface{}{"a": "2"})
	if h1 == h2 {
		t.Error("expected different hashes for different args")
	}
}

func TestHashResult_NilResult(t *testing.T) {
	if got := hashResult(nil); got != "nil" {
		t.Errorf("expected 'nil', got %q", got)
	}
}

func TestHashResult_Deterministic(t *testing.T) {
	r := &ToolResult{ForLLM: "output data"}
	h1 := hashResult(r)
	h2 := hashResult(r)
	if h1 != h2 {
		t.Errorf("hashResult not deterministic: %s != %s", h1, h2)
	}
}

// --- Config validation tests ---

func TestNewLoopDetector_DefaultConfig(t *testing.T) {
	d := NewLoopDetector(DefaultLoopDetectorConfig())
	if d.config.HistorySize != 30 {
		t.Errorf("HistorySize = %d, want 30", d.config.HistorySize)
	}
	if d.config.WarningThreshold != 10 {
		t.Errorf("WarningThreshold = %d, want 10", d.config.WarningThreshold)
	}
	if d.config.CriticalThreshold != 20 {
		t.Errorf("CriticalThreshold = %d, want 20", d.config.CriticalThreshold)
	}
	if d.config.CircuitBreakerThreshold != 30 {
		t.Errorf("CircuitBreakerThreshold = %d, want 30", d.config.CircuitBreakerThreshold)
	}
}

func TestNewLoopDetector_FixesZeroThresholds(t *testing.T) {
	d := NewLoopDetector(LoopDetectorConfig{
		WarningThreshold:        0,  // zero → default 10
		CriticalThreshold:       -1, // negative → default 20
		CircuitBreakerThreshold: 0,  // zero → default 30
	})
	if d.config.WarningThreshold != DefaultWarningThreshold {
		t.Errorf("WarningThreshold = %d, want %d", d.config.WarningThreshold, DefaultWarningThreshold)
	}
	if d.config.CriticalThreshold != DefaultCriticalThreshold {
		t.Errorf("CriticalThreshold = %d, want %d", d.config.CriticalThreshold, DefaultCriticalThreshold)
	}
	if d.config.CircuitBreakerThreshold != DefaultCircuitBreakerThreshold {
		t.Errorf("CircuitBreakerThreshold = %d, want %d", d.config.CircuitBreakerThreshold, DefaultCircuitBreakerThreshold)
	}
}

func TestNewLoopDetector_RespectsPositiveThresholds(t *testing.T) {
	d := NewLoopDetector(LoopDetectorConfig{
		WarningThreshold:        100,
		CriticalThreshold:       3,
		CircuitBreakerThreshold: 5,
	})
	// All positive values should be kept as-is
	if d.config.WarningThreshold != 100 {
		t.Errorf("WarningThreshold = %d, want 100", d.config.WarningThreshold)
	}
	if d.config.CriticalThreshold != 3 {
		t.Errorf("CriticalThreshold = %d, want 3", d.config.CriticalThreshold)
	}
	if d.config.CircuitBreakerThreshold != 5 {
		t.Errorf("CircuitBreakerThreshold = %d, want 5", d.config.CircuitBreakerThreshold)
	}
}

// --- Generic repeat detection ---

func TestLoopDetector_BelowWarning_NoBlock(t *testing.T) {
	d := NewLoopDetector(LoopDetectorConfig{
		WarningThreshold:        5,
		CriticalThreshold:       10,
		CircuitBreakerThreshold: 20,
		EnableGenericRepeat:     true,
	})
	ctx := WithSessionKey(context.Background(), "test")
	args := map[string]interface{}{"key": "val"}

	// Call 4 times (below warning=5): all should pass
	for i := 0; i < 4; i++ {
		if err := d.BeforeExecute(ctx, "read_file", args); err != nil {
			t.Fatalf("call %d: unexpected block: %v", i, err)
		}
	}
}

func TestLoopDetector_AtWarning_NoBlock(t *testing.T) {
	d := NewLoopDetector(LoopDetectorConfig{
		WarningThreshold:        3,
		CriticalThreshold:       6,
		CircuitBreakerThreshold: 12,
		EnableGenericRepeat:     true,
	})
	ctx := WithSessionKey(context.Background(), "test")
	args := map[string]interface{}{"key": "val"}

	// Warning is informational — should NOT block
	for i := 0; i < 5; i++ {
		if err := d.BeforeExecute(ctx, "read_file", args); err != nil {
			t.Fatalf("call %d: unexpected block at warning level: %v", i, err)
		}
	}
}

func TestLoopDetector_AtCritical_Blocks(t *testing.T) {
	d := NewLoopDetector(LoopDetectorConfig{
		WarningThreshold:        3,
		CriticalThreshold:       6,
		CircuitBreakerThreshold: 15,
		EnableGenericRepeat:     true,
	})
	ctx := WithSessionKey(context.Background(), "test")
	args := map[string]interface{}{"key": "val"}

	// First 6 calls should pass (history counts 0..5 before each check)
	for i := 0; i < 6; i++ {
		if err := d.BeforeExecute(ctx, "read_file", args); err != nil {
			t.Fatalf("call %d: unexpected block: %v", i, err)
		}
	}

	// 7th call: history has 6 entries, check sees count=6 >= critical=6 → block
	if err := d.BeforeExecute(ctx, "read_file", args); err == nil {
		t.Fatal("expected block at critical threshold, got nil")
	}
}

func TestLoopDetector_DifferentTools_NoConflict(t *testing.T) {
	d := NewLoopDetector(LoopDetectorConfig{
		WarningThreshold:        3,
		CriticalThreshold:       6,
		CircuitBreakerThreshold: 15,
		EnableGenericRepeat:     true,
	})
	ctx := WithSessionKey(context.Background(), "test")

	// Alternate between two tools: neither should hit threshold
	for i := 0; i < 10; i++ {
		tool := "read_file"
		if i%2 == 1 {
			tool = "write_file"
		}
		if err := d.BeforeExecute(ctx, tool, nil); err != nil {
			t.Fatalf("call %d (%s): unexpected block: %v", i, tool, err)
		}
	}
}

func TestLoopDetector_GenericRepeatDisabled(t *testing.T) {
	d := NewLoopDetector(LoopDetectorConfig{
		WarningThreshold:        3,
		CriticalThreshold:       6,
		CircuitBreakerThreshold: 100, // high so circuit breaker doesn't fire
		EnableGenericRepeat:     false,
	})
	ctx := WithSessionKey(context.Background(), "test")

	// Should never block from generic repeat when disabled
	for i := 0; i < 20; i++ {
		if err := d.BeforeExecute(ctx, "read_file", nil); err != nil {
			t.Fatalf("call %d: block with generic repeat disabled: %v", i, err)
		}
	}
}

// --- Ping-pong detection ---

func TestLoopDetector_PingPong_Detected(t *testing.T) {
	d := NewLoopDetector(LoopDetectorConfig{
		WarningThreshold:        4,
		CriticalThreshold:       8,
		CircuitBreakerThreshold: 100,
		EnableGenericRepeat:     false, // isolate ping-pong
		EnablePingPong:          true,
	})
	ctx := WithSessionKey(context.Background(), "test")
	argsA := map[string]interface{}{"file": "a.txt"}
	argsB := map[string]interface{}{"file": "b.txt"}

	// Build alternation pattern: A, B, A, B, ...
	// With result tracking for no-progress evidence
	for i := 0; i < 20; i++ {
		var tool string
		var args map[string]interface{}
		if i%2 == 0 {
			tool = "read_file"
			args = argsA
		} else {
			tool = "write_file"
			args = argsB
		}
		err := d.BeforeExecute(ctx, tool, args)
		// Record identical result to establish no-progress
		d.AfterExecute(ctx, tool, args, &ToolResult{ForLLM: fmt.Sprintf("result_%s", tool)})

		if err != nil {
			// Should eventually block
			if i < 8 {
				t.Fatalf("blocked too early at call %d: %v", i, err)
			}
			return // success: blocked at or after critical threshold
		}
	}
	t.Fatal("ping-pong was never blocked after 20 alternating calls")
}

func TestLoopDetector_PingPong_WithProgress_NoBlock(t *testing.T) {
	d := NewLoopDetector(LoopDetectorConfig{
		WarningThreshold:        4,
		CriticalThreshold:       8,
		CircuitBreakerThreshold: 100,
		EnableGenericRepeat:     false,
		EnablePingPong:          true,
	})
	ctx := WithSessionKey(context.Background(), "test")
	argsA := map[string]interface{}{"file": "a.txt"}
	argsB := map[string]interface{}{"file": "b.txt"}

	// Build alternation but with CHANGING results (progress)
	for i := 0; i < 20; i++ {
		var tool string
		var args map[string]interface{}
		if i%2 == 0 {
			tool = "read_file"
			args = argsA
		} else {
			tool = "write_file"
			args = argsB
		}
		if err := d.BeforeExecute(ctx, tool, args); err != nil {
			t.Fatalf("blocked at call %d despite progress: %v", i, err)
		}
		// Different result each time = progress
		d.AfterExecute(ctx, tool, args, &ToolResult{ForLLM: fmt.Sprintf("result_%d", i)})
	}
}

func TestLoopDetector_PingPongDisabled(t *testing.T) {
	d := NewLoopDetector(LoopDetectorConfig{
		WarningThreshold:        3,
		CriticalThreshold:       6,
		CircuitBreakerThreshold: 100,
		EnableGenericRepeat:     false,
		EnablePingPong:          false,
	})
	ctx := WithSessionKey(context.Background(), "test")

	for i := 0; i < 20; i++ {
		tool := "read_file"
		if i%2 == 1 {
			tool = "write_file"
		}
		if err := d.BeforeExecute(ctx, tool, nil); err != nil {
			t.Fatalf("call %d: block with ping-pong disabled: %v", i, err)
		}
		d.AfterExecute(ctx, tool, nil, &ToolResult{ForLLM: "same"})
	}
}

// --- No-progress / circuit breaker ---

func TestLoopDetector_CircuitBreaker_NoProgress(t *testing.T) {
	threshold := 8
	d := NewLoopDetector(LoopDetectorConfig{
		WarningThreshold:        100, // high so generic repeat doesn't fire
		CriticalThreshold:       100,
		CircuitBreakerThreshold: threshold,
		EnableGenericRepeat:     false,
		EnablePingPong:          false,
	})
	ctx := WithSessionKey(context.Background(), "test")
	args := map[string]interface{}{"file": "/tmp/stuck"}

	for i := 0; i < threshold+5; i++ {
		err := d.BeforeExecute(ctx, "read_file", args)
		// Record identical result each time
		d.AfterExecute(ctx, "read_file", args, &ToolResult{ForLLM: "same output"})

		if err != nil {
			if i < threshold {
				t.Fatalf("circuit breaker fired too early at call %d", i)
			}
			return // success
		}
	}
	t.Fatal("circuit breaker never fired")
}

func TestLoopDetector_CircuitBreaker_WithProgress_NoBlock(t *testing.T) {
	d := NewLoopDetector(LoopDetectorConfig{
		WarningThreshold:        100,
		CriticalThreshold:       100,
		CircuitBreakerThreshold: 5,
		EnableGenericRepeat:     false,
	})
	ctx := WithSessionKey(context.Background(), "test")

	for i := 0; i < 20; i++ {
		if err := d.BeforeExecute(ctx, "exec", nil); err != nil {
			t.Fatalf("call %d: blocked despite progress: %v", i, err)
		}
		// Different result each time
		d.AfterExecute(ctx, "exec", nil, &ToolResult{ForLLM: fmt.Sprintf("output_%d", i)})
	}
}

// --- Session isolation ---

func TestLoopDetector_SessionIsolation(t *testing.T) {
	d := NewLoopDetector(LoopDetectorConfig{
		WarningThreshold:        3,
		CriticalThreshold:       5,
		CircuitBreakerThreshold: 15,
		EnableGenericRepeat:     true,
	})

	ctxA := WithSessionKey(context.Background(), "session-A")
	ctxB := WithSessionKey(context.Background(), "session-B")

	// Fill session A to near-critical
	for i := 0; i < 4; i++ {
		if err := d.BeforeExecute(ctxA, "read_file", nil); err != nil {
			t.Fatalf("session A call %d: unexpected block: %v", i, err)
		}
	}

	// Session B should be unaffected
	for i := 0; i < 4; i++ {
		if err := d.BeforeExecute(ctxB, "read_file", nil); err != nil {
			t.Fatalf("session B call %d: blocked by session A state: %v", i, err)
		}
	}
}

// --- ResetSession ---

func TestLoopDetector_ResetSession(t *testing.T) {
	d := NewLoopDetector(LoopDetectorConfig{
		WarningThreshold:        3,
		CriticalThreshold:       5,
		CircuitBreakerThreshold: 15,
		EnableGenericRepeat:     true,
	})
	ctx := WithSessionKey(context.Background(), "reset-test")

	// Fill to near-critical
	for i := 0; i < 4; i++ {
		d.BeforeExecute(ctx, "read_file", nil)
	}

	// Reset session
	d.ResetSession("reset-test")

	// Should be able to call again without hitting threshold
	for i := 0; i < 4; i++ {
		if err := d.BeforeExecute(ctx, "read_file", nil); err != nil {
			t.Fatalf("call %d after reset: unexpected block: %v", i, err)
		}
	}
}

// --- Sliding window ---

func TestLoopDetector_SlidingWindow_EvictsOld(t *testing.T) {
	d := NewLoopDetector(LoopDetectorConfig{
		HistorySize:             5, // tiny window
		WarningThreshold:        3,
		CriticalThreshold:       5,
		CircuitBreakerThreshold: 10,
		EnableGenericRepeat:     true,
	})
	ctx := WithSessionKey(context.Background(), "window-test")

	// Fill 5 entries with read_file
	for i := 0; i < 5; i++ {
		d.BeforeExecute(ctx, "read_file", nil)
	}

	// Now call different tools to push old entries out of window
	for i := 0; i < 5; i++ {
		d.BeforeExecute(ctx, fmt.Sprintf("tool_%d", i), nil)
	}

	// read_file should no longer be in window — calling it should not trigger anything
	if err := d.BeforeExecute(ctx, "read_file", nil); err != nil {
		t.Fatalf("expected no block after window eviction, got: %v", err)
	}
}

// --- Integration with ToolRegistry ---

func TestLoopDetector_IntegrationWithRegistry(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(&loopTestTool{name: "stuck_tool", result: "same"})

	d := NewLoopDetector(LoopDetectorConfig{
		WarningThreshold:        3,
		CriticalThreshold:       5,
		CircuitBreakerThreshold: 15,
		EnableGenericRepeat:     true,
	})
	reg.AddHook(d)

	ctx := WithSessionKey(context.Background(), "integration")

	// Should succeed initially
	for i := 0; i < 5; i++ {
		result := reg.Execute(ctx, "stuck_tool", nil)
		if result.IsError {
			t.Fatalf("call %d: unexpected error: %s", i, result.ForLLM)
		}
	}

	// 6th call should be blocked
	result := reg.Execute(ctx, "stuck_tool", nil)
	if !result.IsError {
		t.Fatal("expected block at critical threshold via registry integration")
	}
}

// --- AfterExecute records result ---

func TestLoopDetector_AfterExecute_RecordsResult(t *testing.T) {
	d := NewLoopDetector(DefaultLoopDetectorConfig())
	ctx := WithSessionKey(context.Background(), "after-test")
	args := map[string]interface{}{"x": "1"}

	d.BeforeExecute(ctx, "test_tool", args)
	d.AfterExecute(ctx, "test_tool", args, &ToolResult{ForLLM: "result"})

	// Verify result was recorded
	state := d.getSession("after-test")
	state.mu.Lock()
	defer state.mu.Unlock()

	if len(state.history) != 1 {
		t.Fatalf("history len = %d, want 1", len(state.history))
	}
	if state.history[0].ResultHash == "" {
		t.Error("result hash not recorded by AfterExecute")
	}
}

// --- Default session key ---

func TestLoopDetector_DefaultSessionKey(t *testing.T) {
	d := NewLoopDetector(LoopDetectorConfig{
		WarningThreshold:        3,
		CriticalThreshold:       5,
		CircuitBreakerThreshold: 15,
		EnableGenericRepeat:     true,
	})

	// No session key in context — should use "_default"
	ctx := context.Background()
	for i := 0; i < 4; i++ {
		if err := d.BeforeExecute(ctx, "test", nil); err != nil {
			t.Fatalf("call %d: unexpected block: %v", i, err)
		}
	}
}
