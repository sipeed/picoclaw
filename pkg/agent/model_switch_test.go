package agent

import (
	"context"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

type statefulMockProvider struct {
	closed bool
}

func (m *statefulMockProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	options map[string]any,
) (*providers.LLMResponse, error) {
	return &providers.LLMResponse{Content: "ok"}, nil
}

func (m *statefulMockProvider) GetDefaultModel() string { return "mock" }
func (m *statefulMockProvider) Close()                  { m.closed = true }

func testSwitchConfig() *config.Config {
	return &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         "/tmp/picoclaw-switch-test",
				ModelName:         "model-a",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
			List: []config.AgentConfig{
				{ID: "main", Default: true},
				{ID: "worker", Model: &config.AgentModelConfig{Primary: "fixed-model"}},
			},
		},
		ModelList: []config.ModelConfig{
			{ModelName: "model-a", Model: "codex-cli/codex"},
			{ModelName: "model-b", Model: "claude-cli/claude"},
			{ModelName: "fixed-model", Model: "codex-cli/fixed"},
			{ModelName: "broken-model", Model: "openai/gpt-bad"},
		},
	}
}

func TestModelSwitchManager_SwitchModelSuccess(t *testing.T) {
	cfg := testSwitchConfig()
	initialProvider := &statefulMockProvider{}
	registry := NewAgentRegistry(cfg, initialProvider)
	manager := NewModelSwitchManager(cfg, registry)

	if err := manager.SwitchModel("", "model-b"); err != nil {
		t.Fatalf("SwitchModel() error = %v", err)
	}

	if got := cfg.Agents.Defaults.GetModelName(); got != "model-b" {
		t.Fatalf("default model = %q, want %q", got, "model-b")
	}

	mainAgent, ok := registry.GetAgent("main")
	if !ok {
		t.Fatal("main agent not found")
	}
	if mainAgent.Model != "model-b" {
		t.Fatalf("main agent model = %q, want %q", mainAgent.Model, "model-b")
	}
	if len(mainAgent.Candidates) == 0 || mainAgent.Candidates[0].Provider != "claude-cli" {
		t.Fatalf("main candidates not refreshed: %+v", mainAgent.Candidates)
	}

	worker, ok := registry.GetAgent("worker")
	if !ok {
		t.Fatal("worker agent not found")
	}
	if worker.Model != "fixed-model" {
		t.Fatalf("worker model should remain fixed-model, got %q", worker.Model)
	}
	if worker.Provider == initialProvider {
		t.Fatal("worker provider should be hot-swapped to new provider")
	}

	if !initialProvider.closed {
		t.Fatal("old stateful provider should be closed after switch")
	}
}

func TestModelSwitchManager_SwitchModelRollbackOnProviderCreateFailure(t *testing.T) {
	cfg := testSwitchConfig()
	initialProvider := &statefulMockProvider{}
	registry := NewAgentRegistry(cfg, initialProvider)
	manager := NewModelSwitchManager(cfg, registry)

	err := manager.SwitchModel("", "broken-model")
	if err == nil {
		t.Fatal("expected error for broken-model provider creation")
	}

	if got := cfg.Agents.Defaults.GetModelName(); got != "model-a" {
		t.Fatalf("default model should roll back to model-a, got %q", got)
	}

	mainAgent, ok := registry.GetAgent("main")
	if !ok {
		t.Fatal("main agent not found")
	}
	if mainAgent.Model != "model-a" {
		t.Fatalf("main model should remain model-a, got %q", mainAgent.Model)
	}
	if mainAgent.Provider != initialProvider {
		t.Fatal("provider should remain unchanged on failure")
	}
	if initialProvider.closed {
		t.Fatal("old provider should not be closed when switch fails")
	}
}
