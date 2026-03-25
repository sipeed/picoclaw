package agent

import (
	"os"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

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

				ModelName: "glm-5",
			},
		},

		ModelList: []*config.ModelConfig{
			{
				ModelName: "glm-5",

				Model: "glm-5",

				APIBase: "https://api.z.ai/api/coding/paas/v4",
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
