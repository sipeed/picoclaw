package multiagent

import (
	"context"
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
