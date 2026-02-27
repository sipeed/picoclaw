package agent

import (
	"os"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
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
	agent := NewAgentInstance(nil, &cfg.Agents.Defaults, cfg, provider)

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
	agent := NewAgentInstance(nil, &cfg.Agents.Defaults, cfg, provider)

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
	agent := NewAgentInstance(nil, &cfg.Agents.Defaults, cfg, provider)

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

	provider := &mockProvider{}
	agent := NewAgentInstance(nil, &cfg.Agents.Defaults, cfg, provider)

	if len(agent.Candidates) != 1 {
		t.Fatalf("len(Candidates) = %d, want 1", len(agent.Candidates))
	}
	if agent.Candidates[0].Provider != "openrouter" {
		t.Fatalf("candidate provider = %q, want %q", agent.Candidates[0].Provider, "openrouter")
	}
	if agent.Candidates[0].Model != "stepfun/step-3.5-flash:free" {
		t.Fatalf("candidate model = %q, want %q", agent.Candidates[0].Model, "stepfun/step-3.5-flash:free")
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

	provider := &mockProvider{}
	agent := NewAgentInstance(nil, &cfg.Agents.Defaults, cfg, provider)

	if len(agent.Candidates) != 1 {
		t.Fatalf("len(Candidates) = %d, want 1", len(agent.Candidates))
	}
	if agent.Candidates[0].Provider != "openai" {
		t.Fatalf("candidate provider = %q, want %q", agent.Candidates[0].Provider, "openai")
	}
	if agent.Candidates[0].Model != "glm-5" {
		t.Fatalf("candidate model = %q, want %q", agent.Candidates[0].Model, "glm-5")
	}
}
