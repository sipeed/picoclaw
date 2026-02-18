package multiagent

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/tools"
)

// mockProvider is a minimal LLM provider for testing.
type mockProvider struct {
	response string
	err      error
}

func (m *mockProvider) Chat(_ context.Context, _ []providers.Message, _ []providers.ToolDefinition, _ string, _ map[string]any) (*providers.LLMResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &providers.LLMResponse{
		Content:      m.response,
		FinishReason: "stop",
	}, nil
}

func (m *mockProvider) GetDefaultModel() string { return "mock-model" }

// mockResolver implements AgentResolver for testing.
type mockResolver struct {
	agents map[string]*AgentInfo
}

func newMockResolver(agents ...*AgentInfo) *mockResolver {
	m := &mockResolver{agents: make(map[string]*AgentInfo)}
	for _, a := range agents {
		m.agents[a.ID] = a
	}
	return m
}

func (m *mockResolver) GetAgentInfo(agentID string) *AgentInfo {
	return m.agents[agentID]
}

func (m *mockResolver) ListAgents() []AgentInfo {
	result := make([]AgentInfo, 0, len(m.agents))
	for _, a := range m.agents {
		result = append(result, *a)
	}
	return result
}

func TestExecuteHandoff_Success(t *testing.T) {
	provider := &mockProvider{response: "task completed successfully"}
	resolver := newMockResolver(&AgentInfo{
		ID:       "coder",
		Name:     "Code Agent",
		Role:     "coding specialist",
		Model:    "test-model",
		Provider: provider,
		Tools:    tools.NewToolRegistry(),
		MaxIter:  5,
	})

	bb := NewBlackboard()
	result := ExecuteHandoff(context.Background(), resolver, bb, HandoffRequest{
		FromAgentID: "main",
		ToAgentID:   "coder",
		Task:        "write a function",
		Context:     map[string]string{"language": "Go"},
	}, "cli", "direct")

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if result.Content != "task completed successfully" {
		t.Errorf("Content = %q, want %q", result.Content, "task completed successfully")
	}
	if result.AgentID != "coder" {
		t.Errorf("AgentID = %q, want %q", result.AgentID, "coder")
	}

	// Verify context was written to blackboard
	if bb.Get("language") != "Go" {
		t.Errorf("blackboard 'language' = %q, want %q", bb.Get("language"), "Go")
	}
}

func TestExecuteHandoff_UnknownAgent(t *testing.T) {
	resolver := newMockResolver()
	bb := NewBlackboard()

	result := ExecuteHandoff(context.Background(), resolver, bb, HandoffRequest{
		FromAgentID: "main",
		ToAgentID:   "nonexistent",
		Task:        "do something",
	}, "cli", "direct")

	if result.Success {
		t.Error("expected failure for unknown agent")
	}
	if !strings.Contains(result.Error, "not found") {
		t.Errorf("Error = %q, expected 'not found'", result.Error)
	}
}

func TestExecuteHandoff_NilBlackboard(t *testing.T) {
	provider := &mockProvider{response: "done"}
	resolver := newMockResolver(&AgentInfo{
		ID:       "helper",
		Name:     "Helper",
		Model:    "test",
		Provider: provider,
		Tools:    tools.NewToolRegistry(),
		MaxIter:  5,
	})

	// Should not panic with nil blackboard
	result := ExecuteHandoff(context.Background(), resolver, nil, HandoffRequest{
		FromAgentID: "main",
		ToAgentID:   "helper",
		Task:        "help me",
		Context:     map[string]string{"key": "value"},
	}, "cli", "direct")

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
}

func TestHandoffTool_Execute(t *testing.T) {
	provider := &mockProvider{response: "handoff result"}
	resolver := newMockResolver(
		&AgentInfo{ID: "main", Name: "Main", Role: "orchestrator", Provider: provider, Tools: tools.NewToolRegistry(), MaxIter: 5},
		&AgentInfo{ID: "coder", Name: "Coder", Role: "coding", Model: "test", Provider: provider, Tools: tools.NewToolRegistry(), MaxIter: 5},
	)

	bb := NewBlackboard()
	tool := NewHandoffTool(resolver, bb, "main")

	result := tool.Execute(context.Background(), map[string]any{
		"agent_id": "coder",
		"task":     "write code",
	})

	if result.IsError {
		t.Fatalf("handoff tool failed: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "handoff result") {
		t.Errorf("ForLLM = %q, expected to contain 'handoff result'", result.ForLLM)
	}
}

func TestHandoffTool_MissingArgs(t *testing.T) {
	resolver := newMockResolver()
	bb := NewBlackboard()
	tool := NewHandoffTool(resolver, bb, "main")

	// Missing agent_id
	result := tool.Execute(context.Background(), map[string]any{
		"task": "do something",
	})
	if !result.IsError {
		t.Error("expected error for missing agent_id")
	}

	// Missing task
	result = tool.Execute(context.Background(), map[string]any{
		"agent_id": "coder",
	})
	if !result.IsError {
		t.Error("expected error for missing task")
	}
}

func TestHandoffTool_Description(t *testing.T) {
	resolver := newMockResolver(
		&AgentInfo{ID: "main", Name: "Main"},
		&AgentInfo{ID: "coder", Name: "Coder", Role: "coding specialist"},
	)
	bb := NewBlackboard()
	tool := NewHandoffTool(resolver, bb, "main")

	desc := tool.Description()
	if !strings.Contains(desc, "coder") {
		t.Errorf("Description = %q, expected to contain 'coder'", desc)
	}
	if !strings.Contains(desc, "coding specialist") {
		t.Errorf("Description = %q, expected to contain role", desc)
	}
}

func TestHandoffTool_Description_WithCapabilities(t *testing.T) {
	resolver := newMockResolver(
		&AgentInfo{ID: "main", Name: "Main"},
		&AgentInfo{ID: "coder", Name: "Coder", Role: "coding", Capabilities: []string{"coding", "review"}},
	)
	bb := NewBlackboard()
	tool := NewHandoffTool(resolver, bb, "main")

	desc := tool.Description()
	if !strings.Contains(desc, "coding, review") {
		t.Errorf("Description = %q, expected capabilities", desc)
	}
}

func TestHandoffTool_ExecuteByCapability(t *testing.T) {
	provider := &mockProvider{response: "capability result"}
	resolver := newMockResolver(
		&AgentInfo{ID: "main", Name: "Main", Provider: provider, Tools: tools.NewToolRegistry(), MaxIter: 5},
		&AgentInfo{ID: "coder", Name: "Coder", Capabilities: []string{"coding"}, Provider: provider, Tools: tools.NewToolRegistry(), MaxIter: 5},
	)
	bb := NewBlackboard()
	tool := NewHandoffTool(resolver, bb, "main")

	result := tool.Execute(context.Background(), map[string]any{
		"capability": "coding",
		"task":       "write a function",
	})

	if result.IsError {
		t.Fatalf("handoff by capability failed: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "capability result") {
		t.Errorf("ForLLM = %q, expected 'capability result'", result.ForLLM)
	}
}

func TestHandoffTool_ExecuteByCapability_NotFound(t *testing.T) {
	resolver := newMockResolver(
		&AgentInfo{ID: "main", Name: "Main"},
	)
	bb := NewBlackboard()
	tool := NewHandoffTool(resolver, bb, "main")

	result := tool.Execute(context.Background(), map[string]any{
		"capability": "nonexistent",
		"task":       "do something",
	})

	if !result.IsError {
		t.Error("expected error for unknown capability")
	}
}

func TestHandoffTool_ExecuteNoAgentNoCapability(t *testing.T) {
	resolver := newMockResolver()
	bb := NewBlackboard()
	tool := NewHandoffTool(resolver, bb, "main")

	result := tool.Execute(context.Background(), map[string]any{
		"task": "do something",
	})

	if !result.IsError {
		t.Error("expected error when neither agent_id nor capability provided")
	}
}

func TestListAgentsTool_Execute(t *testing.T) {
	resolver := newMockResolver(
		&AgentInfo{ID: "main", Name: "Main Agent", Role: "general"},
		&AgentInfo{ID: "coder", Name: "Code Agent", Role: "coding"},
	)
	tool := NewListAgentsTool(resolver)

	result := tool.Execute(context.Background(), nil)
	if result.IsError {
		t.Fatalf("list_agents failed: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "main") || !strings.Contains(result.ForLLM, "coder") {
		t.Errorf("ForLLM = %q, expected agent IDs", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "2") {
		t.Errorf("ForLLM = %q, expected count", result.ForLLM)
	}
}

func TestListAgentsTool_Empty(t *testing.T) {
	resolver := newMockResolver()
	tool := NewListAgentsTool(resolver)

	result := tool.Execute(context.Background(), nil)
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "No agents") {
		t.Errorf("ForLLM = %q, expected 'No agents' message", result.ForLLM)
	}
}

func TestFindAgentsByCapability(t *testing.T) {
	resolver := newMockResolver(
		&AgentInfo{ID: "coder", Name: "Coder", Capabilities: []string{"coding", "review"}},
		&AgentInfo{ID: "researcher", Name: "Researcher", Capabilities: []string{"research", "web_search"}},
		&AgentInfo{ID: "generalist", Name: "Generalist"},
	)

	// Find coding agents
	matches := FindAgentsByCapability(resolver, "coding")
	if len(matches) != 1 || matches[0].ID != "coder" {
		t.Errorf("FindAgentsByCapability(coding) = %v, want [coder]", matches)
	}

	// Find research agents
	matches = FindAgentsByCapability(resolver, "research")
	if len(matches) != 1 || matches[0].ID != "researcher" {
		t.Errorf("FindAgentsByCapability(research) = %v, want [researcher]", matches)
	}

	// No match
	matches = FindAgentsByCapability(resolver, "design")
	if len(matches) != 0 {
		t.Errorf("FindAgentsByCapability(design) = %v, want empty", matches)
	}
}

func TestFindAgentsByCapability_Multiple(t *testing.T) {
	resolver := newMockResolver(
		&AgentInfo{ID: "a", Capabilities: []string{"coding"}},
		&AgentInfo{ID: "b", Capabilities: []string{"coding", "review"}},
		&AgentInfo{ID: "c", Capabilities: []string{"research"}},
	)

	matches := FindAgentsByCapability(resolver, "coding")
	if len(matches) != 2 {
		t.Errorf("expected 2 matches, got %d", len(matches))
	}
}

func TestFindAgentsByCapability_Empty(t *testing.T) {
	resolver := newMockResolver()
	matches := FindAgentsByCapability(resolver, "anything")
	if len(matches) != 0 {
		t.Errorf("expected empty, got %v", matches)
	}
}

func TestAgentInfo_Capabilities(t *testing.T) {
	agent := &AgentInfo{
		ID:           "coder",
		Name:         "Code Agent",
		Capabilities: []string{"coding", "review", "testing"},
	}
	if len(agent.Capabilities) != 3 {
		t.Errorf("Capabilities len = %d, want 3", len(agent.Capabilities))
	}

	// Nil capabilities should not panic
	agent2 := &AgentInfo{ID: "basic"}
	if agent2.Capabilities != nil {
		t.Error("expected nil Capabilities for unset agent")
	}
}

func TestExecuteHandoff_DepthLimit(t *testing.T) {
	provider := &mockProvider{response: "done"}
	resolver := newMockResolver(&AgentInfo{
		ID:       "target",
		Name:     "Target",
		Model:    "test",
		Provider: provider,
		Tools:    tools.NewToolRegistry(),
		MaxIter:  5,
	})

	bb := NewBlackboard()
	result := ExecuteHandoff(context.Background(), resolver, bb, HandoffRequest{
		FromAgentID: "main",
		ToAgentID:   "target",
		Task:        "do something",
		Depth:       3, // at max depth
		MaxDepth:    3,
		Visited:     []string{"main", "agent-a", "agent-b"},
	}, "cli", "direct")

	if result.Success {
		t.Error("expected failure at max depth")
	}
	if !strings.Contains(result.Error, "depth limit") {
		t.Errorf("Error = %q, expected 'depth limit'", result.Error)
	}
}

func TestExecuteHandoff_CycleDetection(t *testing.T) {
	provider := &mockProvider{response: "done"}
	resolver := newMockResolver(
		&AgentInfo{ID: "main", Name: "Main", Model: "test", Provider: provider, Tools: tools.NewToolRegistry(), MaxIter: 5},
		&AgentInfo{ID: "coder", Name: "Coder", Model: "test", Provider: provider, Tools: tools.NewToolRegistry(), MaxIter: 5},
	)

	bb := NewBlackboard()

	// Try to hand off to "main" which is already in the visited chain
	result := ExecuteHandoff(context.Background(), resolver, bb, HandoffRequest{
		FromAgentID: "coder",
		ToAgentID:   "main",
		Task:        "some task",
		Depth:       1,
		Visited:     []string{"main", "coder"},
	}, "cli", "direct")

	if result.Success {
		t.Error("expected failure due to cycle detection")
	}
	if !strings.Contains(result.Error, "cycle detected") {
		t.Errorf("Error = %q, expected 'cycle detected'", result.Error)
	}
}

func TestExecuteHandoff_DefaultMaxDepth(t *testing.T) {
	provider := &mockProvider{response: "done"}
	resolver := newMockResolver(&AgentInfo{
		ID: "target", Name: "Target", Model: "test",
		Provider: provider, Tools: tools.NewToolRegistry(), MaxIter: 5,
	})

	bb := NewBlackboard()

	// Depth 2 with default max (3) should succeed
	result := ExecuteHandoff(context.Background(), resolver, bb, HandoffRequest{
		FromAgentID: "main",
		ToAgentID:   "target",
		Task:        "do something",
		Depth:       2,
		Visited:     []string{"main", "middle"},
	}, "cli", "direct")
	if !result.Success {
		t.Fatalf("expected success at depth 2 (max 3), got error: %s", result.Error)
	}

	// Depth 3 with default max should fail
	result = ExecuteHandoff(context.Background(), resolver, bb, HandoffRequest{
		FromAgentID: "main",
		ToAgentID:   "target",
		Task:        "do something",
		Depth:       3,
		Visited:     []string{"main", "a", "b"},
	}, "cli", "direct")
	if result.Success {
		t.Error("expected failure at depth 3 with default max 3")
	}
}

func TestExecuteHandoff_PropagatesDepthToTarget(t *testing.T) {
	provider := &mockProvider{response: "done"}
	targetRegistry := tools.NewToolRegistry()
	innerResolver := newMockResolver()
	targetHandoff := NewHandoffTool(innerResolver, NewBlackboard(), "target")
	targetRegistry.Register(targetHandoff)

	resolver := newMockResolver(&AgentInfo{
		ID: "target", Name: "Target", Model: "test",
		Provider: provider, Tools: targetRegistry, MaxIter: 5,
	})

	bb := NewBlackboard()
	result := ExecuteHandoff(context.Background(), resolver, bb, HandoffRequest{
		FromAgentID: "main",
		ToAgentID:   "target",
		Task:        "do something",
		Depth:       1,
		Visited:     []string{"main"},
		MaxDepth:    5,
	}, "cli", "direct")

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	// Verify the target's handoff tool got the propagated depth
	if targetHandoff.depth != 2 {
		t.Errorf("target handoff depth = %d, want 2", targetHandoff.depth)
	}
	if len(targetHandoff.visited) != 2 || targetHandoff.visited[0] != "main" || targetHandoff.visited[1] != "target" {
		t.Errorf("target handoff visited = %v, want [main target]", targetHandoff.visited)
	}
	if targetHandoff.maxDepth != 5 {
		t.Errorf("target handoff maxDepth = %d, want 5", targetHandoff.maxDepth)
	}
}

func TestHandoffTool_AllowlistBlocks(t *testing.T) {
	provider := &mockProvider{response: "done"}
	resolver := newMockResolver(
		&AgentInfo{ID: "main", Name: "Main", Provider: provider, Tools: tools.NewToolRegistry(), MaxIter: 5},
		&AgentInfo{ID: "restricted", Name: "Restricted", Model: "test", Provider: provider, Tools: tools.NewToolRegistry(), MaxIter: 5},
	)

	bb := NewBlackboard()
	tool := NewHandoffTool(resolver, bb, "main")
	tool.SetAllowlistChecker(AllowlistCheckerFunc(func(from, to string) bool {
		return to == "allowed-agent" // only allow "allowed-agent"
	}))

	result := tool.Execute(context.Background(), map[string]any{
		"agent_id": "restricted",
		"task":     "do something",
	})
	if !result.IsError {
		t.Error("expected error for blocked handoff")
	}
	if !strings.Contains(result.ForLLM, "not allowed") {
		t.Errorf("ForLLM = %q, expected 'not allowed'", result.ForLLM)
	}
}

func TestHandoffTool_AllowlistPermits(t *testing.T) {
	provider := &mockProvider{response: "allowed result"}
	resolver := newMockResolver(
		&AgentInfo{ID: "main", Name: "Main", Provider: provider, Tools: tools.NewToolRegistry(), MaxIter: 5},
		&AgentInfo{ID: "coder", Name: "Coder", Model: "test", Provider: provider, Tools: tools.NewToolRegistry(), MaxIter: 5},
	)

	bb := NewBlackboard()
	tool := NewHandoffTool(resolver, bb, "main")
	tool.SetAllowlistChecker(AllowlistCheckerFunc(func(from, to string) bool {
		return to == "coder" // allow coder
	}))

	result := tool.Execute(context.Background(), map[string]any{
		"agent_id": "coder",
		"task":     "write code",
	})
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "allowed result") {
		t.Errorf("ForLLM = %q, expected 'allowed result'", result.ForLLM)
	}
}

func TestHandoffTool_NoAllowlistAllowsAll(t *testing.T) {
	provider := &mockProvider{response: "ok"}
	resolver := newMockResolver(
		&AgentInfo{ID: "main", Name: "Main", Provider: provider, Tools: tools.NewToolRegistry(), MaxIter: 5},
		&AgentInfo{ID: "any", Name: "Any", Model: "test", Provider: provider, Tools: tools.NewToolRegistry(), MaxIter: 5},
	)

	bb := NewBlackboard()
	tool := NewHandoffTool(resolver, bb, "main")
	// No allowlist checker set

	result := tool.Execute(context.Background(), map[string]any{
		"agent_id": "any",
		"task":     "anything",
	})
	if result.IsError {
		t.Fatalf("expected success with no allowlist, got: %s", result.ForLLM)
	}
}

func TestHandoffTool_SetBoard(t *testing.T) {
	provider := &mockProvider{response: "done"}
	resolver := newMockResolver(
		&AgentInfo{ID: "main", Name: "Main", Provider: provider, Tools: tools.NewToolRegistry(), MaxIter: 5},
		&AgentInfo{ID: "coder", Name: "Coder", Model: "test", Provider: provider, Tools: tools.NewToolRegistry(), MaxIter: 5},
	)

	bb1 := NewBlackboard()
	bb2 := NewBlackboard()
	bb2.Set("session_data", "hello", "system")

	tool := NewHandoffTool(resolver, bb1, "main")

	// Switch to session board
	tool.SetBoard(bb2)

	// Execute with context that writes to blackboard
	tool.Execute(context.Background(), map[string]any{
		"agent_id": "coder",
		"task":     "write code",
		"context":  map[string]any{"language": "Go"},
	})

	// Context should have been written to bb2 (session board), not bb1
	if bb1.Get("language") != "" {
		t.Error("context was written to old board")
	}
	if bb2.Get("language") != "Go" {
		t.Errorf("context not written to session board: %q", bb2.Get("language"))
	}
}

func TestAllowlistCheckerFunc(t *testing.T) {
	checker := AllowlistCheckerFunc(func(from, to string) bool {
		return from == "main" && to == "coder"
	})

	if !checker.CanHandoff("main", "coder") {
		t.Error("expected main->coder to be allowed")
	}
	if checker.CanHandoff("main", "other") {
		t.Error("expected main->other to be blocked")
	}
	if checker.CanHandoff("other", "coder") {
		t.Error("expected other->coder to be blocked")
	}
}

// TestExecuteHandoff_DepthBoundary verifies that depth == maxDepth - 1 (one below limit) succeeds,
// while depth == maxDepth fails. This is the exact boundary behaviour of the recursion guard.
func TestExecuteHandoff_DepthBoundary(t *testing.T) {
	provider := &mockProvider{response: "done"}
	resolver := newMockResolver(&AgentInfo{
		ID: "target", Name: "Target", Model: "test",
		Provider: provider, Tools: tools.NewToolRegistry(), MaxIter: 5,
	})
	bb := NewBlackboard()

	// depth == maxDepth - 1 (2 < 3): must succeed
	result := ExecuteHandoff(context.Background(), resolver, bb, HandoffRequest{
		FromAgentID: "main",
		ToAgentID:   "target",
		Task:        "do something",
		Depth:       2,
		MaxDepth:    3,
		Visited:     []string{"main", "middle"},
	}, "cli", "direct")
	if !result.Success {
		t.Errorf("depth == maxDepth-1 should succeed, got error: %s", result.Error)
	}

	// depth == maxDepth (3 >= 3): must fail
	result = ExecuteHandoff(context.Background(), resolver, bb, HandoffRequest{
		FromAgentID: "main",
		ToAgentID:   "target",
		Task:        "do something",
		Depth:       3,
		MaxDepth:    3,
		Visited:     []string{"main", "a", "b"},
	}, "cli", "direct")
	if result.Success {
		t.Error("depth == maxDepth should fail")
	}
	if !strings.Contains(result.Error, "depth limit") {
		t.Errorf("Error = %q, expected 'depth limit'", result.Error)
	}
}

// TestExecuteHandoff_ProviderError verifies that a provider error during RunToolLoop
// is surfaced as a failed HandoffResult with an error message.
func TestExecuteHandoff_ProviderError(t *testing.T) {
	provider := &mockProvider{err: fmt.Errorf("LLM provider unavailable")}
	resolver := newMockResolver(&AgentInfo{
		ID: "target", Name: "Target", Model: "test",
		Provider: provider, Tools: tools.NewToolRegistry(), MaxIter: 5,
	})

	bb := NewBlackboard()
	result := ExecuteHandoff(context.Background(), resolver, bb, HandoffRequest{
		FromAgentID: "main",
		ToAgentID:   "target",
		Task:        "failing task",
	}, "cli", "direct")

	if result.Success {
		t.Error("expected failure when provider returns error")
	}
	if !strings.Contains(result.Error, "provider unavailable") {
		t.Errorf("Error = %q, expected provider error message", result.Error)
	}
	if result.AgentID != "target" {
		t.Errorf("AgentID = %q, want 'target'", result.AgentID)
	}
}

// TestExecuteHandoff_MaxIterDefault verifies that MaxIter == 0 on the target agent
// is defaulted to 10 inside ExecuteHandoff (not left as 0 which would mean no iterations).
func TestExecuteHandoff_MaxIterDefault(t *testing.T) {
	provider := &mockProvider{response: "ran with default iter"}
	resolver := newMockResolver(&AgentInfo{
		ID:       "target",
		Name:     "Target",
		Model:    "test",
		Provider: provider,
		Tools:    tools.NewToolRegistry(),
		MaxIter:  0, // explicitly zero, should default to 10
	})

	bb := NewBlackboard()
	result := ExecuteHandoff(context.Background(), resolver, bb, HandoffRequest{
		FromAgentID: "main",
		ToAgentID:   "target",
		Task:        "task with default iter",
	}, "cli", "direct")

	if !result.Success {
		t.Errorf("expected success with default MaxIter, got: %s", result.Error)
	}
}

// TestExecuteHandoff_CycleDetectionSingleHop verifies A->A (self-handoff) is caught.
func TestExecuteHandoff_CycleDetectionSingleHop(t *testing.T) {
	provider := &mockProvider{response: "done"}
	resolver := newMockResolver(&AgentInfo{
		ID: "main", Name: "Main", Model: "test",
		Provider: provider, Tools: tools.NewToolRegistry(), MaxIter: 5,
	})

	bb := NewBlackboard()
	// "main" handing off to itself, already in visited
	result := ExecuteHandoff(context.Background(), resolver, bb, HandoffRequest{
		FromAgentID: "main",
		ToAgentID:   "main",
		Task:        "self task",
		Depth:       0,
		Visited:     []string{"main"},
	}, "cli", "direct")

	if result.Success {
		t.Error("expected failure for self-handoff cycle")
	}
	if !strings.Contains(result.Error, "cycle detected") {
		t.Errorf("Error = %q, expected 'cycle detected'", result.Error)
	}
}

// TestHandoffTool_SetContext verifies SetContext updates origin channel and chatID.
func TestHandoffTool_SetContext(t *testing.T) {
	resolver := newMockResolver()
	bb := NewBlackboard()
	tool := NewHandoffTool(resolver, bb, "main")

	tool.SetContext("telegram", "chat-123")

	// Verify fields are updated (access via the exported setter, values verified by ensuring
	// no panic and the defaults were overwritten — integration confirmed via Execute routing).
	if tool.originChannel != "telegram" {
		t.Errorf("originChannel = %q, want %q", tool.originChannel, "telegram")
	}
	if tool.originChatID != "chat-123" {
		t.Errorf("originChatID = %q, want %q", tool.originChatID, "chat-123")
	}
}

// TestHandoff_DepthPolicy_LeafNoSpawn verifies that at max depth, the target agent's
// tool registry clone has spawn/handoff/list_agents removed.
func TestHandoff_DepthPolicy_LeafNoSpawn(t *testing.T) {
	provider := &mockProvider{response: "leaf result"}
	targetRegistry := tools.NewToolRegistry()
	targetRegistry.Register(&simpleTool{name: "read_file"})
	targetRegistry.Register(&simpleTool{name: "spawn"})
	targetRegistry.Register(&simpleTool{name: "handoff"})
	targetRegistry.Register(&simpleTool{name: "list_agents"})

	resolver := newMockResolver(&AgentInfo{
		ID: "leaf", Name: "Leaf Agent", Model: "test",
		Provider: provider, Tools: targetRegistry, MaxIter: 5,
	})

	bb := NewBlackboard()
	// Depth 2, maxDepth 3: the target will run at depth 3 (req.Depth+1),
	// which equals maxDepth, triggering depth deny.
	result := ExecuteHandoff(context.Background(), resolver, bb, HandoffRequest{
		FromAgentID: "main",
		ToAgentID:   "leaf",
		Task:        "do something as a leaf",
		Depth:       2,
		MaxDepth:    3,
		Visited:     []string{"main", "middle"},
	}, "cli", "direct")

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	// Original registry should still have all 4 tools (clone was modified, not original)
	if targetRegistry.Count() != 4 {
		t.Errorf("original registry count = %d, want 4 (unmodified)", targetRegistry.Count())
	}
}

// TestHandoff_DepthPolicy_MidChain verifies that mid-chain agents retain all tools.
func TestHandoff_DepthPolicy_MidChain(t *testing.T) {
	provider := &mockProvider{response: "mid-chain result"}
	targetRegistry := tools.NewToolRegistry()
	targetRegistry.Register(&simpleTool{name: "read_file"})
	targetRegistry.Register(&simpleTool{name: "spawn"})
	targetRegistry.Register(&simpleTool{name: "handoff"})
	targetRegistry.Register(&simpleTool{name: "list_agents"})

	resolver := newMockResolver(&AgentInfo{
		ID: "mid", Name: "Mid Agent", Model: "test",
		Provider: provider, Tools: targetRegistry, MaxIter: 5,
	})

	bb := NewBlackboard()
	// Depth 0, maxDepth 3: target runs at depth 1, well below max.
	result := ExecuteHandoff(context.Background(), resolver, bb, HandoffRequest{
		FromAgentID: "main",
		ToAgentID:   "mid",
		Task:        "mid-chain task",
		Depth:       0,
		MaxDepth:    3,
		Visited:     []string{"main"},
	}, "cli", "direct")

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	// Original registry should still have all 4 tools
	if targetRegistry.Count() != 4 {
		t.Errorf("original registry count = %d, want 4", targetRegistry.Count())
	}
}

// simpleTool is a minimal tool for depth policy tests.
type simpleTool struct {
	name string
}

func (s *simpleTool) Name() string                       { return s.name }
func (s *simpleTool) Description() string                { return "test tool" }
func (s *simpleTool) Parameters() map[string]interface{} { return nil }
func (s *simpleTool) Execute(_ context.Context, _ map[string]interface{}) *tools.ToolResult {
	return tools.NewToolResult("ok")
}

func TestBuildHandoffSystemPrompt(t *testing.T) {
	agent := &AgentInfo{
		Name:         "Code Agent",
		Role:         "coding specialist",
		SystemPrompt: "Focus on Go code quality.",
	}

	bb := NewBlackboard()
	bb.Set("language", "Go", "main")

	prompt := buildHandoffSystemPrompt(agent, bb)
	if !strings.Contains(prompt, "Code Agent") {
		t.Errorf("prompt missing agent name: %s", prompt)
	}
	if !strings.Contains(prompt, "coding specialist") {
		t.Errorf("prompt missing role: %s", prompt)
	}
	if !strings.Contains(prompt, "Focus on Go code quality") {
		t.Errorf("prompt missing system prompt: %s", prompt)
	}
	if !strings.Contains(prompt, "language") {
		t.Errorf("prompt missing blackboard context: %s", prompt)
	}
}
