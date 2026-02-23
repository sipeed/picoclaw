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

func TestNewAgentInstance_PlanModel_SetFromDefaults(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-instance-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:          tmpDir,
				Provider:           "zhipu",
				Model:              "glm-4.7",
				PlanModel:          "anthropic/claude-sonnet-4-6",
				PlanModelFallbacks: []string{"openai/gpt-4o"},
				MaxTokens:          8192,
				MaxToolIterations:  20,
			},
		},
	}

	provider := &mockProvider{}
	agent := NewAgentInstance(nil, &cfg.Agents.Defaults, cfg, provider)

	if agent.PlanModel != "anthropic/claude-sonnet-4-6" {
		t.Errorf("PlanModel = %q, want 'anthropic/claude-sonnet-4-6'", agent.PlanModel)
	}
	if len(agent.PlanFallbacks) != 1 || agent.PlanFallbacks[0] != "openai/gpt-4o" {
		t.Errorf("PlanFallbacks = %v, want [openai/gpt-4o]", agent.PlanFallbacks)
	}
	if len(agent.PlanCandidates) == 0 {
		t.Fatal("PlanCandidates should not be empty when plan_model is set")
	}
	if agent.PlanCandidates[0].Model != "claude-sonnet-4-6" {
		t.Errorf("PlanCandidates[0].Model = %q, want 'claude-sonnet-4-6'", agent.PlanCandidates[0].Model)
	}
}

func TestNewAgentInstance_PlanModel_NilWhenUnset(t *testing.T) {
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

	if agent.PlanModel != "" {
		t.Errorf("PlanModel = %q, want empty", agent.PlanModel)
	}
	if agent.PlanCandidates != nil {
		t.Errorf("PlanCandidates = %v, want nil", agent.PlanCandidates)
	}
}

func TestNewAgentInstance_PlanModel_AgentOverridesDefaults(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-instance-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:          tmpDir,
				Provider:           "zhipu",
				Model:              "glm-4.7",
				PlanModel:          "default-plan-model",
				PlanModelFallbacks: []string{"default-fallback"},
				MaxTokens:          8192,
				MaxToolIterations:  20,
			},
		},
	}

	agentCfg := &config.AgentConfig{
		ID: "custom",
		PlanModel: &config.AgentModelConfig{
			Primary:   "agent-plan-model",
			Fallbacks: []string{"agent-fallback"},
		},
	}

	provider := &mockProvider{}
	agent := NewAgentInstance(agentCfg, &cfg.Agents.Defaults, cfg, provider)

	if agent.PlanModel != "agent-plan-model" {
		t.Errorf("PlanModel = %q, want 'agent-plan-model'", agent.PlanModel)
	}
	if len(agent.PlanFallbacks) != 1 || agent.PlanFallbacks[0] != "agent-fallback" {
		t.Errorf("PlanFallbacks = %v, want [agent-fallback]", agent.PlanFallbacks)
	}
}
