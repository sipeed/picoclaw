package agent

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

func TestResolveActiveModelConfig_PrefersCandidateIdentityKey(t *testing.T) {
	cfg := &config.Config{
		ModelList: []*config.ModelConfig{
			{
				ModelName: "glm-4.7",
				Provider:  "zhipu",
				Model:     "glm-4.7",
				Streaming: config.ModelStreamingConfig{Enabled: false},
			},
			{
				ModelName: "suanneng-glm-4.7",
				Provider:  "zhipu",
				Model:     "glm-4.7",
				Streaming: config.ModelStreamingConfig{Enabled: true},
			},
		},
	}

	got := resolveActiveModelConfig(
		cfg,
		"/workspace",
		[]providers.FallbackCandidate{{
			Provider:    "zhipu",
			Model:       "glm-4.7",
			IdentityKey: "model_name:suanneng-glm-4.7",
		}},
		"glm-4.7",
		"openai",
	)

	if got == nil {
		t.Fatal("resolveActiveModelConfig() = nil, want model config")
	}
	if got.ModelName != "suanneng-glm-4.7" {
		t.Fatalf("model_name = %q, want %q", got.ModelName, "suanneng-glm-4.7")
	}
	if !got.Streaming.Enabled {
		t.Fatal("streaming.enabled = false, want true from identity-matched model config")
	}
}

func TestResolveActiveModelConfig_LoadBalancedAliasUsesSelectedCandidate(t *testing.T) {
	cfg := &config.Config{
		ModelList: []*config.ModelConfig{
			{
				ModelName: "lb-model",
				Model:     "openai/primary",
				Streaming: config.ModelStreamingConfig{Enabled: false},
			},
			{
				ModelName: "lb-model",
				Model:     "openai/secondary",
				Streaming: config.ModelStreamingConfig{Enabled: true},
			},
		},
	}

	got := resolveActiveModelConfig(
		cfg,
		"/workspace",
		[]providers.FallbackCandidate{{
			Provider:    "openai",
			Model:       "secondary",
			IdentityKey: "model_name:lb-model",
		}},
		"lb-model",
		"openai",
	)

	if got == nil {
		t.Fatal("resolveActiveModelConfig() = nil, want model config")
	}
	if got.Model != "openai/secondary" {
		t.Fatalf("model = %q, want openai/secondary", got.Model)
	}
	if !got.Streaming.Enabled {
		t.Fatal("streaming.enabled = false, want true from selected load-balanced entry")
	}
}

func TestResolveActiveModelConfig_DoesNotFallbackToOpenAIForDefaultProviderCandidate(t *testing.T) {
	cfg := &config.Config{
		ModelList: []*config.ModelConfig{
			{
				ModelName: "openai-gpt",
				Provider:  "openai",
				Model:     "gpt-4o",
				Streaming: config.ModelStreamingConfig{Enabled: true},
			},
		},
	}

	got := resolveActiveModelConfig(
		cfg,
		"/workspace",
		[]providers.FallbackCandidate{{
			Provider: "nvidia",
			Model:    "gpt-4o",
		}},
		"gpt-4o",
		"nvidia",
	)

	if got != nil {
		t.Fatalf("resolveActiveModelConfig() = %#v, want nil for non-active provider config", got)
	}
}
