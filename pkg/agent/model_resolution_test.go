package agent

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

func TestModelNameFromIdentityKey_LegacyProviderModel(t *testing.T) {
	if got := modelNameFromIdentityKey("openai/gpt-5.4"); got != "gpt-5.4" {
		t.Fatalf("modelNameFromIdentityKey() = %q, want %q", got, "gpt-5.4")
	}
}

func TestModelNameFromIdentityKey_PreservesNonLegacyIdentity(t *testing.T) {
	if got := modelNameFromIdentityKey("model_name:primary"); got != "model_name:primary" {
		t.Fatalf("modelNameFromIdentityKey() = %q, want %q", got, "model_name:primary")
	}
}

func TestModelAliasFromCandidateIdentityKey(t *testing.T) {
	if got := modelAliasFromCandidateIdentityKey("model_name:primary"); got != "primary" {
		t.Fatalf("modelAliasFromCandidateIdentityKey() = %q, want %q", got, "primary")
	}
	if got := modelAliasFromCandidateIdentityKey("openai/gpt-5.4"); got != "" {
		t.Fatalf("modelAliasFromCandidateIdentityKey() = %q, want empty", got)
	}
}

func TestResolvedCandidateModelName_PrefersIdentityAlias(t *testing.T) {
	got := resolvedCandidateModelName([]providers.FallbackCandidate{
		{Provider: "openai", Model: "gpt-5.4", IdentityKey: "model_name:primary"},
	}, "fallback-model")
	if got != "primary" {
		t.Fatalf("resolvedCandidateModelName() = %q, want %q", got, "primary")
	}
}

func TestResolvedCandidateModelName_DoesNotScanFallbackAliases(t *testing.T) {
	got := resolvedCandidateModelName([]providers.FallbackCandidate{
		{Provider: "openai", Model: "gpt-5.4"},
		{Provider: "openai", Model: "gpt-5.4-mini", IdentityKey: "model_name:fallback"},
	}, "primary-model")
	if got != "primary-model" {
		t.Fatalf("resolvedCandidateModelName() = %q, want %q", got, "primary-model")
	}
}

func TestResolvedCandidateModelName_UsesCandidateDisplayName(t *testing.T) {
	got := resolvedCandidateModelName([]providers.FallbackCandidate{
		{Provider: "openai", Model: "gpt-5.4", DisplayName: "gpt-5.4-display"},
	}, "fallback-model")
	if got != "gpt-5.4-display" {
		t.Fatalf("resolvedCandidateModelName() = %q, want %q", got, "gpt-5.4-display")
	}
}

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
