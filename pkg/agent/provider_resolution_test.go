package agent

import (
	"os"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestNewAgentInstance_UsesDedicatedProviderForFallbackAlias(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-provider-resolution-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:      tmpDir,
				Provider:       "openai",
				Model:          "gpt-4.1-mini",
				ModelFallbacks: []string{"gemini-fallback"},
			},
		},
		ModelList: []config.ModelConfig{
			{
				ModelName: "gemini-fallback",
				Model:     "gemini-3.1-flash-lite-preview",
				APIKey:    "test-key",
				APIBase:   "https://customproxy.example/v1",
			},
		},
	}

	primaryProvider := &mockProvider{}
	agent := NewAgentInstance(nil, &cfg.Agents.Defaults, cfg, primaryProvider)

	if len(agent.Candidates) != 2 {
		t.Fatalf("len(Candidates) = %d, want 2", len(agent.Candidates))
	}

	primary := agent.providerForCandidate(agent.Candidates[0].Provider, agent.Candidates[0].Model)
	if primary != primaryProvider {
		t.Fatalf("primary candidate should use the primary provider")
	}

	fallback := agent.providerForCandidate(agent.Candidates[1].Provider, agent.Candidates[1].Model)
	if fallback == nil {
		t.Fatal("fallback provider is nil")
	}
	if fallback == primaryProvider {
		t.Fatal("fallback alias reused the primary provider")
	}
}

func TestNewAgentInstance_UsesConsistentProviderForLoadBalancedFallbackAlias(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-provider-resolution-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:      tmpDir,
				Provider:       "openai",
				Model:          "gpt-4.1-mini",
				ModelFallbacks: []string{"shared-fallback"},
			},
		},
		ModelList: []config.ModelConfig{
			{
				ModelName: "shared-fallback",
				Model:     "gemini/gemini-2.5-flash-lite-preview-06-17",
				APIKey:    "gemini-key",
			},
			{
				ModelName: "shared-fallback",
				Model:     "openrouter/google/gemini-2.5-flash-preview",
				APIKey:    "openrouter-key",
				APIBase:   "https://openrouter.ai/api/v1",
			},
		},
	}

	primaryProvider := &mockProvider{}
	agent := NewAgentInstance(nil, &cfg.Agents.Defaults, cfg, primaryProvider)

	if len(agent.Candidates) != 2 {
		t.Fatalf("len(Candidates) = %d, want 2", len(agent.Candidates))
	}

	fallback := agent.providerForCandidate(agent.Candidates[1].Provider, agent.Candidates[1].Model)
	if fallback == nil {
		t.Fatal("fallback provider is nil")
	}
	if fallback == primaryProvider {
		t.Fatal("load-balanced fallback alias reused the primary provider")
	}
}

func TestNewAgentInstance_UsesDedicatedProviderForLightModelAlias(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-provider-resolution-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace: tmpDir,
				Provider:  "openai",
				Model:     "gpt-4.1-mini",
				Routing: &config.RoutingConfig{
					Enabled:    true,
					LightModel: "gemini-light",
					Threshold:  0.5,
				},
			},
		},
		ModelList: []config.ModelConfig{
			{
				ModelName: "gemini-light",
				Model:     "gemini/gemini-2.5-flash-lite-preview-06-17",
				APIKey:    "test-key",
				APIBase:   "https://customproxy.example/v1",
			},
		},
	}

	primaryProvider := &mockProvider{}
	agent := NewAgentInstance(nil, &cfg.Agents.Defaults, cfg, primaryProvider)

	if len(agent.LightCandidates) != 1 {
		t.Fatalf("len(LightCandidates) = %d, want 1", len(agent.LightCandidates))
	}

	lightProvider := agent.providerForCandidate(agent.LightCandidates[0].Provider, agent.LightCandidates[0].Model)
	if lightProvider == nil {
		t.Fatal("light model provider is nil")
	}
	if lightProvider == primaryProvider {
		t.Fatal("light model alias reused the primary provider")
	}
}
