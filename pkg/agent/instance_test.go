package agent

import (
	"os"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

func TestNewAgentInstance_UsesDefaultsTemperatureAndMaxTokens(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-instance-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				Model:             "test-model",
				MaxTokens:         1234,
				MaxToolIterations: 5,
			},
		},
	}

	configuredTemp := 1.0
	cfg.Agents.Defaults.Temperature = &configuredTemp

	provider := &mockProvider{}
	agent := NewAgentInstance(nil, &cfg.Agents.Defaults, cfg, mockRegistry(provider))

	if agent.MaxTokens != 1234 {
		t.Fatalf("MaxTokens = %d, want %d", agent.MaxTokens, 1234)
	}
	if agent.Temperature != 1.0 {
		t.Fatalf("Temperature = %f, want %f", agent.Temperature, 1.0)
	}
}

func TestNewAgentInstance_DefaultsTemperatureWhenZero(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-instance-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				Model:             "test-model",
				MaxTokens:         1234,
				MaxToolIterations: 5,
			},
		},
	}

	configuredTemp := 0.0
	cfg.Agents.Defaults.Temperature = &configuredTemp

	provider := &mockProvider{}
	agent := NewAgentInstance(nil, &cfg.Agents.Defaults, cfg, mockRegistry(provider))

	if agent.Temperature != 0.0 {
		t.Fatalf("Temperature = %f, want %f", agent.Temperature, 0.0)
	}
}

func TestNewAgentInstance_DefaultsTemperatureWhenUnset(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-instance-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				Model:             "test-model",
				MaxTokens:         1234,
				MaxToolIterations: 5,
			},
		},
	}

	provider := &mockProvider{}
	agent := NewAgentInstance(nil, &cfg.Agents.Defaults, cfg, mockRegistry(provider))

	if agent.Temperature != 0.7 {
		t.Fatalf("Temperature = %f, want %f", agent.Temperature, 0.7)
	}
}

func TestNewAgentInstance_ResolveCandidatesFromModelListAlias(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-instance-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace: tmpDir,
				Model:     "step-3.5-flash",
			},
		},
		ModelList: []config.ModelConfig{
			{
				ModelName: "step-3.5-flash",
				Model:     "openrouter/stepfun/step-3.5-flash:free",
				APIBase:   "https://openrouter.ai/api/v1",
			},
		},
	}

	reg, err := providers.NewModelRegistry(cfg)
	if err != nil {
		t.Fatalf("NewModelRegistry: %v", err)
	}
	agent := NewAgentInstance(nil, &cfg.Agents.Defaults, cfg, reg)

	if len(agent.Candidates) != 1 {
		t.Fatalf("len(Candidates) = %d, want 1", len(agent.Candidates))
	}
	if agent.Candidates[0].Model != "step-3.5-flash" {
		t.Fatalf("candidate model = %q, want %q", agent.Candidates[0].Model, "step-3.5-flash")
	}
}

func TestNewAgentInstance_ResolveCandidatesFromModelListAliasWithoutProtocol(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-instance-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace: tmpDir,
				Model:     "glm-5",
			},
		},
		ModelList: []config.ModelConfig{
			{
				ModelName: "glm-5",
				Model:     "glm-5",
				APIBase:   "https://api.z.ai/api/coding/paas/v4",
			},
		},
	}

	reg, err := providers.NewModelRegistry(cfg)
	if err != nil {
		t.Fatalf("NewModelRegistry: %v", err)
	}
	agent := NewAgentInstance(nil, &cfg.Agents.Defaults, cfg, reg)

	if len(agent.Candidates) != 1 {
		t.Fatalf("len(Candidates) = %d, want 1", len(agent.Candidates))
	}
	if agent.Candidates[0].Model != "glm-5" {
		t.Fatalf("candidate model = %q, want %q", agent.Candidates[0].Model, "glm-5")
	}
}
