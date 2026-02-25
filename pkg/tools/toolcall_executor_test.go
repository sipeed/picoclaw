package tools

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/providers"
)

type executorMockTool struct {
	name   string
	policy ToolParallelPolicy

	delay time.Duration

	result *ToolResult
	errMsg string

	running    *atomic.Int32
	maxRunning *atomic.Int32
	onExecute  func()
	onComplete func()
}

func (t *executorMockTool) Name() string {
	return t.name
}

func (t *executorMockTool) Description() string {
	return "executor mock tool"
}

func (t *executorMockTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
	}
}

func (t *executorMockTool) ParallelPolicy() ToolParallelPolicy {
	return t.policy
}

func (t *executorMockTool) Execute(_ context.Context, _ map[string]any) *ToolResult {
	if t.onExecute != nil {
		t.onExecute()
	}

	if t.running != nil && t.maxRunning != nil {
		cur := t.running.Add(1)
		for {
			prev := t.maxRunning.Load()
			if cur <= prev || t.maxRunning.CompareAndSwap(prev, cur) {
				break
			}
		}
		defer t.running.Add(-1)
	}

	if t.delay > 0 {
		time.Sleep(t.delay)
	}
	if t.onComplete != nil {
		t.onComplete()
	}

	if t.errMsg != "" {
		return ErrorResult(t.errMsg).WithError(fmt.Errorf("%s", t.errMsg))
	}
	if t.result != nil {
		return t.result
	}
	return SilentResult("ok")
}

type executorContextualTool struct {
	executorMockTool
	channel string
	chatID  string
}

func (t *executorContextualTool) SetContext(channel, chatID string) {
	t.channel = channel
	t.chatID = chatID
}

func TestExecuteToolCalls_PreservesOrderWithParallelBatch(t *testing.T) {
	registry := NewToolRegistry()
	registry.Register(&executorMockTool{
		name:   "slow",
		policy: ToolParallelReadOnly,
		delay:  60 * time.Millisecond,
		result: SilentResult("slow-result"),
	})
	registry.Register(&executorMockTool{
		name:   "fast",
		policy: ToolParallelReadOnly,
		delay:  5 * time.Millisecond,
		result: SilentResult("fast-result"),
	})

	calls := []providers.ToolCall{
		{ID: "tc-1", Name: "slow", Arguments: map[string]any{}},
		{ID: "tc-2", Name: "fast", Arguments: map[string]any{}},
	}

	results := ExecuteToolCalls(context.Background(), registry, calls, ToolCallExecutionOptions{
		Iteration: 1,
		LogScope:  "test",
		Parallel: ToolCallParallelConfig{
			Enabled:        true,
			MaxConcurrency: 8,
			Mode:           ParallelToolsModeReadOnlyOnly,
		},
	})

	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	if results[0].ToolCall.ID != "tc-1" || results[0].Result.ForLLM != "slow-result" {
		t.Fatalf("results[0] = %+v, want tc-1/slow-result", results[0])
	}
	if results[1].ToolCall.ID != "tc-2" || results[1].Result.ForLLM != "fast-result" {
		t.Fatalf("results[1] = %+v, want tc-2/fast-result", results[1])
	}
}

func TestExecuteToolCalls_RespectsConcurrencyLimit(t *testing.T) {
	registry := NewToolRegistry()
	var running atomic.Int32
	var maxRunning atomic.Int32
	registry.Register(&executorMockTool{
		name:       "io",
		policy:     ToolParallelReadOnly,
		delay:      25 * time.Millisecond,
		result:     SilentResult("ok"),
		running:    &running,
		maxRunning: &maxRunning,
	})

	calls := make([]providers.ToolCall, 0, 20)
	for i := 0; i < 20; i++ {
		calls = append(calls, providers.ToolCall{
			ID:        fmt.Sprintf("tc-%d", i),
			Name:      "io",
			Arguments: map[string]any{"index": i},
		})
	}

	results := ExecuteToolCalls(context.Background(), registry, calls, ToolCallExecutionOptions{
		Iteration: 1,
		LogScope:  "test",
		Parallel: ToolCallParallelConfig{
			Enabled:        true,
			MaxConcurrency: 3,
			Mode:           ParallelToolsModeReadOnlyOnly,
		},
	})

	if len(results) != len(calls) {
		t.Fatalf("len(results) = %d, want %d", len(results), len(calls))
	}
	if maxRunning.Load() > 3 {
		t.Fatalf("maxRunning = %d, want <= 3", maxRunning.Load())
	}
	if maxRunning.Load() < 2 {
		t.Fatalf("maxRunning = %d, want >= 2 to confirm parallel execution", maxRunning.Load())
	}
}

func TestExecuteToolCalls_SerialBoundaryBeforeParallelBatch(t *testing.T) {
	registry := NewToolRegistry()
	writeDone := make(chan struct{})
	var readStartedBeforeWriteDone atomic.Bool

	registry.Register(&executorMockTool{
		name:       "write_file",
		policy:     ToolParallelSerialOnly,
		delay:      60 * time.Millisecond,
		result:     SilentResult("write-ok"),
		onComplete: func() { close(writeDone) },
	})

	readTool := &executorMockTool{
		name:   "read_file",
		policy: ToolParallelReadOnly,
		delay:  20 * time.Millisecond,
		result: SilentResult("read-ok"),
		onExecute: func() {
			select {
			case <-writeDone:
			default:
				readStartedBeforeWriteDone.Store(true)
			}
		},
	}
	registry.Register(readTool)

	calls := []providers.ToolCall{
		{ID: "tc-1", Name: "write_file", Arguments: map[string]any{"path": "x"}},
		{ID: "tc-2", Name: "read_file", Arguments: map[string]any{"path": "x"}},
		{ID: "tc-3", Name: "read_file", Arguments: map[string]any{"path": "y"}},
	}

	results := ExecuteToolCalls(context.Background(), registry, calls, ToolCallExecutionOptions{
		Iteration: 1,
		LogScope:  "test",
		Parallel: ToolCallParallelConfig{
			Enabled:        true,
			MaxConcurrency: 8,
			Mode:           ParallelToolsModeReadOnlyOnly,
		},
	})

	if len(results) != 3 {
		t.Fatalf("len(results) = %d, want 3", len(results))
	}
	if readStartedBeforeWriteDone.Load() {
		t.Fatal("read tool started before preceding serial write tool finished")
	}
}

func TestExecuteToolCalls_CollectsFailuresWithoutShortCircuit(t *testing.T) {
	registry := NewToolRegistry()
	registry.Register(&executorMockTool{
		name:   "ok",
		policy: ToolParallelReadOnly,
		delay:  20 * time.Millisecond,
		result: SilentResult("ok-result"),
	})
	registry.Register(&executorMockTool{
		name:   "fail",
		policy: ToolParallelReadOnly,
		delay:  5 * time.Millisecond,
		errMsg: "boom",
	})

	calls := []providers.ToolCall{
		{ID: "tc-1", Name: "ok", Arguments: map[string]any{}},
		{ID: "tc-2", Name: "fail", Arguments: map[string]any{}},
	}

	results := ExecuteToolCalls(context.Background(), registry, calls, ToolCallExecutionOptions{
		Iteration: 1,
		LogScope:  "test",
		Parallel: ToolCallParallelConfig{
			Enabled:        true,
			MaxConcurrency: 8,
			Mode:           ParallelToolsModeReadOnlyOnly,
		},
	})

	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	if results[0].Result.IsError {
		t.Fatalf("results[0].Result.IsError = true, want false")
	}
	if !results[1].Result.IsError {
		t.Fatalf("results[1].Result.IsError = false, want true")
	}
}

func TestExecuteToolCalls_OverrideForcesParallelInReadOnlyMode(t *testing.T) {
	registry := NewToolRegistry()
	var running atomic.Int32
	var maxRunning atomic.Int32

	// Default policy is serial_only for tools without ParallelPolicyProvider.
	registry.Register(&executorMockTool{
		name:       "custom_tool",
		delay:      20 * time.Millisecond,
		result:     SilentResult("ok"),
		running:    &running,
		maxRunning: &maxRunning,
	})

	calls := []providers.ToolCall{
		{ID: "tc-1", Name: "custom_tool", Arguments: map[string]any{}},
		{ID: "tc-2", Name: "custom_tool", Arguments: map[string]any{}},
	}

	results := ExecuteToolCalls(context.Background(), registry, calls, ToolCallExecutionOptions{
		Iteration: 1,
		LogScope:  "test",
		Parallel: ToolCallParallelConfig{
			Enabled:        true,
			MaxConcurrency: 8,
			Mode:           ParallelToolsModeReadOnlyOnly,
			ToolPolicyOverrides: map[string]string{
				"custom_tool": string(ToolParallelReadOnly),
			},
		},
	})

	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	if maxRunning.Load() < 2 {
		t.Fatalf("maxRunning = %d, want >= 2 when override forces parallel", maxRunning.Load())
	}
}

func TestExecuteToolCalls_OverrideForcesSerialInAllMode(t *testing.T) {
	registry := NewToolRegistry()
	var running atomic.Int32
	var maxRunning atomic.Int32
	registry.Register(&executorMockTool{
		name:       "any_tool",
		delay:      20 * time.Millisecond,
		result:     SilentResult("ok"),
		running:    &running,
		maxRunning: &maxRunning,
	})

	calls := []providers.ToolCall{
		{ID: "tc-1", Name: "any_tool", Arguments: map[string]any{}},
		{ID: "tc-2", Name: "any_tool", Arguments: map[string]any{}},
	}

	results := ExecuteToolCalls(context.Background(), registry, calls, ToolCallExecutionOptions{
		Iteration: 1,
		LogScope:  "test",
		Parallel: ToolCallParallelConfig{
			Enabled:        true,
			MaxConcurrency: 8,
			Mode:           ParallelToolsModeAll,
			ToolPolicyOverrides: map[string]string{
				"any_tool": string(ToolParallelSerialOnly),
			},
		},
	})

	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	if maxRunning.Load() > 1 {
		t.Fatalf("maxRunning = %d, want <= 1 when override forces serial", maxRunning.Load())
	}
}

func TestExecuteToolCalls_OverrideCannotBypassInstanceSafety(t *testing.T) {
	registry := NewToolRegistry()
	var running atomic.Int32
	var maxRunning atomic.Int32
	registry.Register(&executorContextualTool{
		executorMockTool: executorMockTool{
			name:       "ctx_tool",
			delay:      20 * time.Millisecond,
			result:     SilentResult("ok"),
			running:    &running,
			maxRunning: &maxRunning,
		},
	})

	calls := []providers.ToolCall{
		{ID: "tc-1", Name: "ctx_tool", Arguments: map[string]any{}},
		{ID: "tc-2", Name: "ctx_tool", Arguments: map[string]any{}},
	}

	results := ExecuteToolCalls(context.Background(), registry, calls, ToolCallExecutionOptions{
		Channel:   "telegram",
		ChatID:    "chat-1",
		Iteration: 1,
		LogScope:  "test",
		Parallel: ToolCallParallelConfig{
			Enabled:        true,
			MaxConcurrency: 8,
			Mode:           ParallelToolsModeAll,
			ToolPolicyOverrides: map[string]string{
				"ctx_tool": string(ToolParallelReadOnly),
			},
		},
	})

	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	if maxRunning.Load() > 1 {
		t.Fatalf("maxRunning = %d, want <= 1 because instance safety should block parallel", maxRunning.Load())
	}
}

func TestPercentileInt64_NearestRank(t *testing.T) {
	values := []int64{10, 100}
	if got := percentileInt64(values, 0.95); got != 100 {
		t.Fatalf("p95 with n=2 = %d, want 100 (nearest-rank)", got)
	}
	if got := percentileInt64(values, 0.50); got != 10 {
		t.Fatalf("p50 with n=2 = %d, want 10 (nearest-rank)", got)
	}

	values3 := []int64{10, 20, 30}
	if got := percentileInt64(values3, 0.95); got != 30 {
		t.Fatalf("p95 with n=3 = %d, want 30 (nearest-rank)", got)
	}
}
