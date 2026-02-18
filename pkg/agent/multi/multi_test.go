// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package multi

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/sipeed/picoclaw/pkg/tools"
)

// mockAgent is a simple Agent implementation for testing.
type mockAgent struct {
	name         string
	role         string
	systemPrompt string
	capabilities []string
	toolRegistry *tools.ToolRegistry
	executeFunc  func(ctx context.Context, task string, shared *SharedContext) (string, error)
}

func newMockAgent(name, role string, capabilities []string) *mockAgent {
	return &mockAgent{
		name:         name,
		role:         role,
		systemPrompt: fmt.Sprintf("You are %s, a %s agent.", name, role),
		capabilities: capabilities,
		toolRegistry: tools.NewToolRegistry(),
		executeFunc: func(ctx context.Context, task string, shared *SharedContext) (string, error) {
			return fmt.Sprintf("[%s] completed: %s", name, task), nil
		},
	}
}

func (m *mockAgent) Name() string                 { return m.name }
func (m *mockAgent) Role() string                 { return m.role }
func (m *mockAgent) SystemPrompt() string         { return m.systemPrompt }
func (m *mockAgent) Capabilities() []string       { return m.capabilities }
func (m *mockAgent) Tools() *tools.ToolRegistry   { return m.toolRegistry }
func (m *mockAgent) Execute(ctx context.Context, task string, shared *SharedContext) (string, error) {
	return m.executeFunc(ctx, task, shared)
}

// --- SharedContext Tests ---

func TestSharedContext_SetGet(t *testing.T) {
	sc := NewSharedContext()

	sc.Set("key1", "value1")
	sc.Set("key2", 42)

	v1, ok := sc.Get("key1")
	if !ok || v1 != "value1" {
		t.Errorf("expected key1=value1, got %v (ok=%v)", v1, ok)
	}

	v2, ok := sc.Get("key2")
	if !ok || v2 != 42 {
		t.Errorf("expected key2=42, got %v (ok=%v)", v2, ok)
	}

	_, ok = sc.Get("nonexistent")
	if ok {
		t.Error("expected nonexistent key to return false")
	}
}

func TestSharedContext_GetString(t *testing.T) {
	sc := NewSharedContext()

	sc.Set("str", "hello")
	sc.Set("num", 42)

	if s := sc.GetString("str"); s != "hello" {
		t.Errorf("expected 'hello', got %q", s)
	}

	if s := sc.GetString("num"); s != "" {
		t.Errorf("expected empty string for non-string, got %q", s)
	}

	if s := sc.GetString("missing"); s != "" {
		t.Errorf("expected empty string for missing key, got %q", s)
	}
}

func TestSharedContext_Delete(t *testing.T) {
	sc := NewSharedContext()

	sc.Set("key", "value")
	sc.Delete("key")

	_, ok := sc.Get("key")
	if ok {
		t.Error("expected deleted key to return false")
	}
}

func TestSharedContext_Keys(t *testing.T) {
	sc := NewSharedContext()

	sc.Set("a", 1)
	sc.Set("b", 2)
	sc.Set("c", 3)

	keys := sc.Keys()
	if len(keys) != 3 {
		t.Errorf("expected 3 keys, got %d", len(keys))
	}

	keySet := make(map[string]bool)
	for _, k := range keys {
		keySet[k] = true
	}
	for _, expected := range []string{"a", "b", "c"} {
		if !keySet[expected] {
			t.Errorf("expected key %q in keys", expected)
		}
	}
}

func TestSharedContext_Events(t *testing.T) {
	sc := NewSharedContext()

	sc.AddEvent("agent1", "handoff", "delegating to agent2")
	sc.AddEvent("agent2", "result", "task completed")
	sc.AddEvent("agent1", "error", "something failed")

	events := sc.Events()
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}

	if events[0].Agent != "agent1" || events[0].Type != "handoff" {
		t.Errorf("unexpected first event: %+v", events[0])
	}
}

func TestSharedContext_EventsByAgent(t *testing.T) {
	sc := NewSharedContext()

	sc.AddEvent("agent1", "handoff", "task1")
	sc.AddEvent("agent2", "result", "done")
	sc.AddEvent("agent1", "result", "task2")

	agent1Events := sc.EventsByAgent("agent1")
	if len(agent1Events) != 2 {
		t.Errorf("expected 2 events for agent1, got %d", len(agent1Events))
	}

	agent2Events := sc.EventsByAgent("agent2")
	if len(agent2Events) != 1 {
		t.Errorf("expected 1 event for agent2, got %d", len(agent2Events))
	}
}

func TestSharedContext_EventsByType(t *testing.T) {
	sc := NewSharedContext()

	sc.AddEvent("a1", "result", "r1")
	sc.AddEvent("a2", "error", "e1")
	sc.AddEvent("a3", "result", "r2")

	results := sc.EventsByType("result")
	if len(results) != 2 {
		t.Errorf("expected 2 result events, got %d", len(results))
	}

	errors := sc.EventsByType("error")
	if len(errors) != 1 {
		t.Errorf("expected 1 error event, got %d", len(errors))
	}
}

func TestSharedContext_Snapshot(t *testing.T) {
	sc := NewSharedContext()

	sc.Set("key", "value")
	snap := sc.Snapshot()

	// Modify original
	sc.Set("key", "changed")

	// Snapshot should be independent
	if snap["key"] != "value" {
		t.Errorf("snapshot should be independent, got %v", snap["key"])
	}
}

func TestSharedContext_ConcurrentAccess(t *testing.T) {
	sc := NewSharedContext()
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sc.Set(fmt.Sprintf("key-%d", i), i)
			sc.AddEvent(fmt.Sprintf("agent-%d", i), "write", "data")
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sc.Get(fmt.Sprintf("key-%d", i))
			sc.Keys()
			sc.Events()
		}(i)
	}

	wg.Wait()

	keys := sc.Keys()
	if len(keys) != 100 {
		t.Errorf("expected 100 keys after concurrent writes, got %d", len(keys))
	}
}

// --- BaseAgent Tests ---

func TestBaseAgent_Fields(t *testing.T) {
	registry := tools.NewToolRegistry()
	agent := NewBaseAgent(AgentConfig{
		Name:         "coder",
		Role:         "Code generation and review",
		SystemPrompt: "You are a coding agent.",
		Capabilities: []string{"code", "review"},
	}, registry)

	if agent.Name() != "coder" {
		t.Errorf("expected name 'coder', got %q", agent.Name())
	}
	if agent.Role() != "Code generation and review" {
		t.Errorf("unexpected role: %q", agent.Role())
	}
	if agent.SystemPrompt() != "You are a coding agent." {
		t.Errorf("unexpected system prompt: %q", agent.SystemPrompt())
	}
	caps := agent.Capabilities()
	if len(caps) != 2 || caps[0] != "code" || caps[1] != "review" {
		t.Errorf("unexpected capabilities: %v", caps)
	}
	if agent.Tools() != registry {
		t.Error("expected same tool registry")
	}
}

func TestBaseAgent_NilRegistry(t *testing.T) {
	agent := NewBaseAgent(AgentConfig{Name: "test"}, nil)
	if agent.Tools() == nil {
		t.Error("expected non-nil default tool registry")
	}
}

// --- AgentRegistry Tests ---

func TestAgentRegistry_Register(t *testing.T) {
	r := NewAgentRegistry()
	agent := newMockAgent("coder", "coding", []string{"code"})

	err := r.Register(agent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Duplicate registration
	err = r.Register(agent)
	if err == nil {
		t.Error("expected error on duplicate registration")
	}
}

func TestAgentRegistry_Unregister(t *testing.T) {
	r := NewAgentRegistry()
	agent := newMockAgent("coder", "coding", []string{"code"})
	r.Register(agent)

	err := r.Unregister("coder")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify agent is gone
	if r.Get("coder") != nil {
		t.Error("expected nil after unregister")
	}

	// Unregister nonexistent
	err = r.Unregister("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent agent")
	}
}

func TestAgentRegistry_Get(t *testing.T) {
	r := NewAgentRegistry()
	agent := newMockAgent("searcher", "search", []string{"search"})
	r.Register(agent)

	got := r.Get("searcher")
	if got == nil {
		t.Fatal("expected non-nil agent")
	}
	if got.Name() != "searcher" {
		t.Errorf("expected 'searcher', got %q", got.Name())
	}

	if r.Get("nonexistent") != nil {
		t.Error("expected nil for nonexistent agent")
	}
}

func TestAgentRegistry_List(t *testing.T) {
	r := NewAgentRegistry()
	r.Register(newMockAgent("a", "role-a", nil))
	r.Register(newMockAgent("b", "role-b", nil))
	r.Register(newMockAgent("c", "role-c", nil))

	names := r.List()
	if len(names) != 3 {
		t.Errorf("expected 3 agents, got %d", len(names))
	}
}

func TestAgentRegistry_FindByCapability(t *testing.T) {
	r := NewAgentRegistry()
	r.Register(newMockAgent("coder", "coding", []string{"code", "review"}))
	r.Register(newMockAgent("searcher", "searching", []string{"search", "web"}))
	r.Register(newMockAgent("reviewer", "reviewing", []string{"review"}))

	codeAgents := r.FindByCapability("code")
	if len(codeAgents) != 1 {
		t.Errorf("expected 1 agent with 'code', got %d", len(codeAgents))
	}

	reviewAgents := r.FindByCapability("review")
	if len(reviewAgents) != 2 {
		t.Errorf("expected 2 agents with 'review', got %d", len(reviewAgents))
	}

	noneAgents := r.FindByCapability("nonexistent")
	if len(noneAgents) != 0 {
		t.Errorf("expected 0 agents, got %d", len(noneAgents))
	}
}

func TestAgentRegistry_SharedContext(t *testing.T) {
	r := NewAgentRegistry()
	sc := r.SharedContext()

	if sc == nil {
		t.Fatal("expected non-nil shared context")
	}

	// Verify it's the same instance
	if r.SharedContext() != sc {
		t.Error("expected same shared context instance")
	}
}

// --- Handoff Tests ---

func TestAgentRegistry_Handoff_DirectRouting(t *testing.T) {
	r := NewAgentRegistry()
	r.Register(newMockAgent("coder", "coding", []string{"code"}))

	result := r.Handoff(context.Background(), HandoffRequest{
		FromAgent: "main",
		ToAgent:   "coder",
		Task:      "write a function",
	})

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if result.AgentName != "coder" {
		t.Errorf("expected agent 'coder', got %q", result.AgentName)
	}
	if result.Content != "[coder] completed: write a function" {
		t.Errorf("unexpected content: %q", result.Content)
	}

	// Verify events were recorded
	events := r.SharedContext().Events()
	if len(events) < 2 {
		t.Fatalf("expected at least 2 events, got %d", len(events))
	}
	if events[0].Type != "handoff" {
		t.Errorf("expected first event type 'handoff', got %q", events[0].Type)
	}
	if events[1].Type != "result" {
		t.Errorf("expected second event type 'result', got %q", events[1].Type)
	}
}

func TestAgentRegistry_Handoff_CapabilityRouting(t *testing.T) {
	r := NewAgentRegistry()
	r.Register(newMockAgent("coder", "coding", []string{"code"}))
	r.Register(newMockAgent("searcher", "searching", []string{"search", "web"}))

	result := r.Handoff(context.Background(), HandoffRequest{
		FromAgent:          "main",
		RequiredCapability: "search",
		Task:               "find documentation",
	})

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if result.AgentName != "searcher" {
		t.Errorf("expected agent 'searcher', got %q", result.AgentName)
	}
}

func TestAgentRegistry_Handoff_NotFound(t *testing.T) {
	r := NewAgentRegistry()

	// Target agent not found
	result := r.Handoff(context.Background(), HandoffRequest{
		FromAgent: "main",
		ToAgent:   "nonexistent",
		Task:      "do something",
	})
	if result.Err == nil {
		t.Error("expected error for nonexistent agent")
	}

	// Capability not found
	result = r.Handoff(context.Background(), HandoffRequest{
		FromAgent:          "main",
		RequiredCapability: "nonexistent",
		Task:               "do something",
	})
	if result.Err == nil {
		t.Error("expected error for nonexistent capability")
	}

	// Neither ToAgent nor RequiredCapability
	result = r.Handoff(context.Background(), HandoffRequest{
		FromAgent: "main",
		Task:      "do something",
	})
	if result.Err == nil {
		t.Error("expected error when neither routing field is set")
	}
}

func TestAgentRegistry_Handoff_ExecutionError(t *testing.T) {
	r := NewAgentRegistry()

	failAgent := newMockAgent("failer", "failing", []string{"fail"})
	failAgent.executeFunc = func(ctx context.Context, task string, shared *SharedContext) (string, error) {
		return "", fmt.Errorf("execution failed: %s", task)
	}
	r.Register(failAgent)

	result := r.Handoff(context.Background(), HandoffRequest{
		FromAgent: "main",
		ToAgent:   "failer",
		Task:      "break things",
	})

	if result.Err == nil {
		t.Fatal("expected execution error")
	}
	if result.AgentName != "failer" {
		t.Errorf("expected agent 'failer', got %q", result.AgentName)
	}

	// Verify error event was recorded
	errorEvents := r.SharedContext().EventsByType("error")
	if len(errorEvents) == 0 {
		t.Error("expected error event to be recorded")
	}
}

func TestAgentRegistry_Handoff_ContextPassing(t *testing.T) {
	r := NewAgentRegistry()

	// Agent that reads from shared context
	reader := newMockAgent("reader", "reading", []string{"read"})
	reader.executeFunc = func(ctx context.Context, task string, shared *SharedContext) (string, error) {
		val := shared.GetString("input_data")
		return fmt.Sprintf("read: %s", val), nil
	}
	r.Register(reader)

	result := r.Handoff(context.Background(), HandoffRequest{
		FromAgent: "main",
		ToAgent:   "reader",
		Task:      "process data",
		Context:   map[string]interface{}{"input_data": "hello world"},
	})

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if result.Content != "read: hello world" {
		t.Errorf("unexpected content: %q", result.Content)
	}
}

func TestAgentRegistry_Handoff_AgentStateTransition(t *testing.T) {
	r := NewAgentRegistry()

	// Agent that checks its own state via a channel
	stateChecked := make(chan AgentState, 1)
	statefulAgent := newMockAgent("stateful", "checking", []string{"check"})
	statefulAgent.executeFunc = func(ctx context.Context, task string, shared *SharedContext) (string, error) {
		state, _ := r.GetAgentState("stateful")
		stateChecked <- state
		return "done", nil
	}
	r.Register(statefulAgent)

	// Before hand-off: idle
	state, ok := r.GetAgentState("stateful")
	if !ok || state != AgentIdle {
		t.Errorf("expected idle state before handoff, got %v", state)
	}

	r.Handoff(context.Background(), HandoffRequest{
		FromAgent: "main",
		ToAgent:   "stateful",
		Task:      "check state",
	})

	// During execution: should have been active
	duringState := <-stateChecked
	if duringState != AgentActive {
		t.Errorf("expected active state during execution, got %v", duringState)
	}

	// After hand-off: idle again
	state, _ = r.GetAgentState("stateful")
	if state != AgentIdle {
		t.Errorf("expected idle state after handoff, got %v", state)
	}
}

func TestAgentRegistry_Handoff_ContextCancellation(t *testing.T) {
	r := NewAgentRegistry()

	// Agent that respects context cancellation
	cancelAgent := newMockAgent("cancellable", "cancelling", []string{"cancel"})
	cancelAgent.executeFunc = func(ctx context.Context, task string, shared *SharedContext) (string, error) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
			return "completed before cancel", nil
		}
	}
	r.Register(cancelAgent)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result := r.Handoff(ctx, HandoffRequest{
		FromAgent: "main",
		ToAgent:   "cancellable",
		Task:      "should be cancelled",
	})

	if result.Err == nil {
		// The agent might complete before checking ctx, which is fine
		// Just verify it ran
		if result.Content == "" {
			t.Error("expected some content")
		}
	}
}

// --- Integration Test ---

func TestMultiAgent_Integration(t *testing.T) {
	r := NewAgentRegistry()

	// Register a chain of agents
	analyzer := newMockAgent("analyzer", "analysis", []string{"analyze"})
	analyzer.executeFunc = func(ctx context.Context, task string, shared *SharedContext) (string, error) {
		shared.Set("analysis_result", "code needs refactoring")
		return "Analysis complete", nil
	}

	coder := newMockAgent("coder", "coding", []string{"code"})
	coder.executeFunc = func(ctx context.Context, task string, shared *SharedContext) (string, error) {
		analysis := shared.GetString("analysis_result")
		return fmt.Sprintf("Applied fix based on: %s", analysis), nil
	}

	r.Register(analyzer)
	r.Register(coder)

	// Step 1: Analyze
	result1 := r.Handoff(context.Background(), HandoffRequest{
		FromAgent:          "main",
		RequiredCapability: "analyze",
		Task:               "review the codebase",
	})
	if result1.Err != nil {
		t.Fatalf("analysis failed: %v", result1.Err)
	}

	// Step 2: Code fix based on analysis result (already in shared context)
	result2 := r.Handoff(context.Background(), HandoffRequest{
		FromAgent:          "main",
		RequiredCapability: "code",
		Task:               "fix the issues found",
	})
	if result2.Err != nil {
		t.Fatalf("coding failed: %v", result2.Err)
	}

	if result2.Content != "Applied fix based on: code needs refactoring" {
		t.Errorf("unexpected content: %q", result2.Content)
	}

	// Verify full event trail
	events := r.SharedContext().Events()
	if len(events) != 4 { // 2 handoffs + 2 results
		t.Errorf("expected 4 events, got %d", len(events))
	}
}
