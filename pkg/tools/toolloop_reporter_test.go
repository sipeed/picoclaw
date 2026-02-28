package tools

import (
	"context"
	"sync"
	"testing"

	"github.com/sipeed/picoclaw/pkg/orch"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// reporterSpy records every ReportStateChange call in order.
// Spawn/Conversation/GC are not needed for toolloop tests.
type reporterSpy struct {
	mu    sync.Mutex
	calls []spyCall
}

type spyCall struct {
	state orch.AgentState
	tool  string
}

func (r *reporterSpy) ReportSpawn(id, label, task string)       {}
func (r *reporterSpy) ReportConversation(from, to, text string) {}
func (r *reporterSpy) ReportGC(id, reason string)               {}
func (r *reporterSpy) ReportStateChange(id string, state orch.AgentState, tool string) {
	r.mu.Lock()
	r.calls = append(r.calls, spyCall{state, tool})
	r.mu.Unlock()
}

func (r *reporterSpy) snapshot() []spyCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]spyCall, len(r.calls))
	copy(out, r.calls)
	return out
}

// sequenceMockProvider returns a tool call on the first Chat() call and a
// plain text response on all subsequent calls. Used to exercise the
// waiting → toolcall → waiting event sequence in RunToolLoop.
type sequenceMockProvider struct {
	mu        sync.Mutex
	callCount int
}

func (m *sequenceMockProvider) Chat(
	_ context.Context,
	_ []providers.Message,
	_ []providers.ToolDefinition,
	_ string,
	_ map[string]any,
) (*providers.LLMResponse, error) {
	m.mu.Lock()
	m.callCount++
	n := m.callCount
	m.mu.Unlock()
	if n == 1 {
		return &providers.LLMResponse{
			ToolCalls: []providers.ToolCall{
				{ID: "tc-1", Name: "echo_tool", Arguments: map[string]any{"msg": "hi"}},
			},
		}, nil
	}
	return &providers.LLMResponse{Content: "done"}, nil
}
func (m *sequenceMockProvider) GetDefaultModel() string { return "test" }
func (m *sequenceMockProvider) SupportsTools() bool     { return true }
func (m *sequenceMockProvider) GetContextWindow() int   { return 4096 }

// echoTool is a minimal Tool stub registered as "echo_tool".
type echoTool struct{}

func (t *echoTool) Name() string        { return "echo_tool" }
func (t *echoTool) Description() string { return "echo" }
func (t *echoTool) Parameters() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{}}
}

func (t *echoTool) Execute(_ context.Context, _ map[string]any) *ToolResult {
	return &ToolResult{ForLLM: "echoed"}
}

// TestToolLoop_NilReporter_FallsBackToNoop ensures that passing nil as
// Reporter does not panic — the loop must substitute orch.Noop internally.
func TestToolLoop_NilReporter_FallsBackToNoop(t *testing.T) {
	_, err := RunToolLoop(context.Background(), ToolLoopConfig{
		Provider:      &MockLLMProvider{},
		Model:         "test",
		MaxIterations: 1,
		Reporter:      nil, // must not panic
	}, []providers.Message{{Role: "user", Content: "hi"}}, "cli", "direct")
	if err != nil {
		t.Fatalf("unexpected error with nil reporter: %v", err)
	}
}

// TestToolLoop_Reporter_WaitingBeforeLLM verifies that ReportStateChange is
// called with state="waiting" before the first LLM call. The mock provider
// returns a direct text answer (no tool calls), so exactly one waiting event
// is expected.
func TestToolLoop_Reporter_WaitingBeforeLLM(t *testing.T) {
	rep := &reporterSpy{}
	_, err := RunToolLoop(context.Background(), ToolLoopConfig{
		Provider:      &MockLLMProvider{},
		Model:         "test",
		MaxIterations: 1,
		Reporter:      rep,
		AgentID:       "sess-1",
	}, []providers.Message{{Role: "user", Content: "hi"}}, "cli", "direct")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	calls := rep.snapshot()
	if len(calls) == 0 {
		t.Fatal("expected at least one ReportStateChange call")
	}
	if calls[0].state != orch.AgentStateWaiting {
		t.Fatalf("first call must be state=waiting, got %+v", calls[0])
	}
}

// TestToolLoop_Reporter_ToolcallOrderedAfterWaiting verifies the canonical
// two-iteration sequence:
//
//	waiting  (before 1st LLM call)
//	toolcall(echo_tool)  (before tool execution)
//	waiting  (before 2nd LLM call)
//
// The sequenceMockProvider returns a tool call on iteration 1 and a text
// response on iteration 2, driving exactly this path.
func TestToolLoop_Reporter_ToolcallOrderedAfterWaiting(t *testing.T) {
	rep := &reporterSpy{}
	reg := NewToolRegistry()
	reg.Register(&echoTool{})

	_, err := RunToolLoop(context.Background(), ToolLoopConfig{
		Provider:      &sequenceMockProvider{},
		Model:         "test",
		Tools:         reg,
		MaxIterations: 5,
		Reporter:      rep,
		AgentID:       "sess-1",
	}, []providers.Message{{Role: "user", Content: "do it"}}, "cli", "direct")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	calls := rep.snapshot()
	if len(calls) < 3 {
		t.Fatalf("expected at least 3 calls, got %d: %+v", len(calls), calls)
	}
	if calls[0].state != orch.AgentStateWaiting {
		t.Fatalf("calls[0] must be waiting, got %+v", calls[0])
	}
	if calls[1].state != orch.AgentStateToolCall || calls[1].tool != "echo_tool" {
		t.Fatalf("calls[1] must be toolcall(echo_tool), got %+v", calls[1])
	}
	if calls[2].state != orch.AgentStateWaiting {
		t.Fatalf("calls[2] must be waiting (2nd LLM iteration), got %+v", calls[2])
	}
}

// TestToolLoop_ToolCallStats verifies that ToolLoopResult.ToolCalls and
// ToolStats are populated correctly after a tool call iteration.
func TestToolLoop_ToolCallStats(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(&echoTool{})

	result, err := RunToolLoop(context.Background(), ToolLoopConfig{
		Provider:      &sequenceMockProvider{},
		Model:         "test",
		Tools:         reg,
		MaxIterations: 5,
	}, []providers.Message{{Role: "user", Content: "do it"}}, "cli", "direct")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ToolCalls != 1 {
		t.Errorf("ToolCalls = %d, want 1", result.ToolCalls)
	}
	if result.ToolStats["echo_tool"] != 1 {
		t.Errorf("ToolStats[echo_tool] = %d, want 1", result.ToolStats["echo_tool"])
	}
	if result.Iterations != 2 {
		t.Errorf("Iterations = %d, want 2", result.Iterations)
	}
}

// TestToolLoop_NoToolCalls_ZeroStats verifies that a direct answer (no tool
// calls) produces zero ToolCalls and an empty ToolStats map.
func TestToolLoop_NoToolCalls_ZeroStats(t *testing.T) {
	result, err := RunToolLoop(context.Background(), ToolLoopConfig{
		Provider:      &MockLLMProvider{},
		Model:         "test",
		MaxIterations: 1,
	}, []providers.Message{{Role: "user", Content: "hi"}}, "cli", "direct")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ToolCalls != 0 {
		t.Errorf("ToolCalls = %d, want 0", result.ToolCalls)
	}
	if len(result.ToolStats) != 0 {
		t.Errorf("ToolStats = %v, want empty", result.ToolStats)
	}
}

// TestToolLoop_Reporter_NoopImplementsInterface is a compile-time check that
// orch.Noop satisfies the orch.AgentReporter interface accepted by
// ToolLoopConfig.Reporter. If Noop ever stops implementing the interface the
// build will fail here before any test runs.
func TestToolLoop_Reporter_NoopImplementsInterface(t *testing.T) {
	var _ orch.AgentReporter = orch.Noop
}
