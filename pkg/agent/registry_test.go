package agent

import (
	"context"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

type mockRegistryProvider struct{}

func (m *mockRegistryProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	options map[string]any,
) (*providers.LLMResponse, error) {
	return &providers.LLMResponse{Content: "mock", FinishReason: "stop"}, nil
}

func (m *mockRegistryProvider) GetDefaultModel() string {
	return "mock-model"
}

func testCfg(agents []config.AgentConfig) *config.Config {
	return &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         "/tmp/picoclaw-test-registry",
				Model:             "gpt-4",
				MaxTokens:         8192,
				MaxToolIterations: 10,
			},
			List: agents,
		},
	}
}

func TestNewAgentRegistry_ImplicitMain(t *testing.T) {
	cfg := testCfg(nil)
	registry := NewAgentRegistry(cfg, &mockRegistryProvider{})

	ids := registry.ListAgentIDs()
	if len(ids) != 1 || ids[0] != "main" {
		t.Errorf("expected implicit main agent, got %v", ids)
	}

	agent, ok := registry.GetAgent("main")
	if !ok || agent == nil {
		t.Fatal("expected to find 'main' agent")
	}
	if agent.ID != "main" {
		t.Errorf("agent.ID = %q, want 'main'", agent.ID)
	}
}

func TestNewAgentRegistry_ExplicitAgents(t *testing.T) {
	cfg := testCfg([]config.AgentConfig{
		{ID: "sales", Default: true, Name: "Sales Bot"},
		{ID: "support", Name: "Support Bot"},
	})
	registry := NewAgentRegistry(cfg, &mockRegistryProvider{})

	ids := registry.ListAgentIDs()
	if len(ids) != 2 {
		t.Fatalf("expected 2 agents, got %d: %v", len(ids), ids)
	}

	sales, ok := registry.GetAgent("sales")
	if !ok || sales == nil {
		t.Fatal("expected to find 'sales' agent")
	}
	if sales.Name != "Sales Bot" {
		t.Errorf("sales.Name = %q, want 'Sales Bot'", sales.Name)
	}

	support, ok := registry.GetAgent("support")
	if !ok || support == nil {
		t.Fatal("expected to find 'support' agent")
	}
}

func TestAgentRegistry_GetAgent_Normalize(t *testing.T) {
	cfg := testCfg([]config.AgentConfig{
		{ID: "my-agent", Default: true},
	})
	registry := NewAgentRegistry(cfg, &mockRegistryProvider{})

	agent, ok := registry.GetAgent("My-Agent")
	if !ok || agent == nil {
		t.Fatal("expected to find agent with normalized ID")
	}
	if agent.ID != "my-agent" {
		t.Errorf("agent.ID = %q, want 'my-agent'", agent.ID)
	}
}

func TestAgentRegistry_GetDefaultAgent(t *testing.T) {
	cfg := testCfg([]config.AgentConfig{
		{ID: "alpha"},
		{ID: "beta", Default: true},
	})
	registry := NewAgentRegistry(cfg, &mockRegistryProvider{})

	// GetDefaultAgent first checks for "main", then returns any
	agent := registry.GetDefaultAgent()
	if agent == nil {
		t.Fatal("expected a default agent")
	}
}

func TestAgentRegistry_CanSpawnSubagent(t *testing.T) {
	cfg := testCfg([]config.AgentConfig{
		{
			ID:      "parent",
			Default: true,
			Subagents: &config.SubagentsConfig{
				AllowAgents: []string{"child1", "child2"},
			},
		},
		{ID: "child1"},
		{ID: "child2"},
		{ID: "restricted"},
	})
	registry := NewAgentRegistry(cfg, &mockRegistryProvider{})

	if !registry.CanSpawnSubagent("parent", "child1") {
		t.Error("expected parent to be allowed to spawn child1")
	}
	if !registry.CanSpawnSubagent("parent", "child2") {
		t.Error("expected parent to be allowed to spawn child2")
	}
	if registry.CanSpawnSubagent("parent", "restricted") {
		t.Error("expected parent to NOT be allowed to spawn restricted")
	}
	if registry.CanSpawnSubagent("child1", "child2") {
		t.Error("expected child1 to NOT be allowed to spawn (no subagents config)")
	}
}

func TestAgentRegistry_CanSpawnSubagent_Wildcard(t *testing.T) {
	cfg := testCfg([]config.AgentConfig{
		{
			ID:      "admin",
			Default: true,
			Subagents: &config.SubagentsConfig{
				AllowAgents: []string{"*"},
			},
		},
		{ID: "any-agent"},
	})
	registry := NewAgentRegistry(cfg, &mockRegistryProvider{})

	if !registry.CanSpawnSubagent("admin", "any-agent") {
		t.Error("expected wildcard to allow spawning any agent")
	}
	if !registry.CanSpawnSubagent("admin", "nonexistent") {
		t.Error("expected wildcard to allow spawning even nonexistent agents")
	}
}

func TestAgentInstance_Model(t *testing.T) {
	model := &config.AgentModelConfig{Primary: "claude-opus"}
	cfg := testCfg([]config.AgentConfig{
		{ID: "custom", Default: true, Model: model},
	})
	registry := NewAgentRegistry(cfg, &mockRegistryProvider{})

	agent, _ := registry.GetAgent("custom")
	if agent.Model != "claude-opus" {
		t.Errorf("agent.Model = %q, want 'claude-opus'", agent.Model)
	}
}

func TestAgentInstance_FallbackInheritance(t *testing.T) {
	cfg := testCfg([]config.AgentConfig{
		{ID: "inherit", Default: true},
	})
	cfg.Agents.Defaults.ModelFallbacks = []string{"openai/gpt-4o-mini", "anthropic/haiku"}
	registry := NewAgentRegistry(cfg, &mockRegistryProvider{})

	agent, _ := registry.GetAgent("inherit")
	if len(agent.Fallbacks) != 2 {
		t.Errorf("expected 2 fallbacks inherited from defaults, got %d", len(agent.Fallbacks))
	}
}

func TestAgentInstance_FallbackExplicitEmpty(t *testing.T) {
	model := &config.AgentModelConfig{
		Primary:   "gpt-4",
		Fallbacks: []string{}, // explicitly empty = disable
	}
	cfg := testCfg([]config.AgentConfig{
		{ID: "no-fallback", Default: true, Model: model},
	})
	cfg.Agents.Defaults.ModelFallbacks = []string{"should-not-inherit"}
	registry := NewAgentRegistry(cfg, &mockRegistryProvider{})

	agent, _ := registry.GetAgent("no-fallback")
	if len(agent.Fallbacks) != 0 {
		t.Errorf("expected 0 fallbacks (explicit empty), got %d: %v", len(agent.Fallbacks), agent.Fallbacks)
	}
}
