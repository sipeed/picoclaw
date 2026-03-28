// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package config

import (
	"strings"
	"testing"
)

func TestConvertProvidersToModelList_OpenAI(t *testing.T) {
	cfg := &configV0{
		Providers: providersConfigV0{
			OpenAI: openAIProviderConfigV0{
				providerConfigV0: providerConfigV0{
					APIKey:  "sk-test-key",
					APIBase: "https://custom.api.com/v1",
				},
			},
		},
	}

	result := v0ConvertProvidersToModelList(cfg)

	if len(result) != 1 {
		t.Fatalf("len(result) = %d, want 1", len(result))
	}

	if result[0].ModelName != "openai" {
		t.Errorf("ModelName = %q, want %q", result[0].ModelName, "openai")
	}
	if result[0].Model != "openai/gpt-5.4" {
		t.Errorf("Model = %q, want %q", result[0].Model, "openai/gpt-5.4")
	}
	if result[0].APIKey != "sk-test-key" {
		t.Errorf("APIKey = %q, want %q", result[0].APIKey, "sk-test-key")
	}
}

func TestConvertProvidersToModelList_Anthropic(t *testing.T) {
	cfg := &configV0{
		Providers: providersConfigV0{
			Anthropic: providerConfigV0{
				APIBase: "https://custom.anthropic.com",
			},
		},
	}

	result := v0ConvertProvidersToModelList(cfg)

	if len(result) != 1 {
		t.Fatalf("len(result) = %d, want 1", len(result))
	}

	if result[0].ModelName != "anthropic" {
		t.Errorf("ModelName = %q, want %q", result[0].ModelName, "anthropic")
	}
	if result[0].Model != "anthropic/claude-sonnet-4.6" {
		t.Errorf("Model = %q, want %q", result[0].Model, "anthropic/claude-sonnet-4.6")
	}
}

func TestConvertProvidersToModelList_LiteLLM(t *testing.T) {
	cfg := &configV0{
		Providers: providersConfigV0{
			LiteLLM: providerConfigV0{
				APIBase: "http://localhost:4000/v1",
			},
		},
	}

	result := v0ConvertProvidersToModelList(cfg)

	if len(result) != 1 {
		t.Fatalf("len(result) = %d, want 1", len(result))
	}

	if result[0].ModelName != "litellm" {
		t.Errorf("ModelName = %q, want %q", result[0].ModelName, "litellm")
	}
	if result[0].Model != "litellm/auto" {
		t.Errorf("Model = %q, want %q", result[0].Model, "litellm/auto")
	}
	if result[0].APIBase != "http://localhost:4000/v1" {
		t.Errorf("APIBase = %q, want %q", result[0].APIBase, "http://localhost:4000/v1")
	}
}

func TestConvertProvidersToModelList_Multiple(t *testing.T) {
	cfg := &configV0{
		Providers: providersConfigV0{
			OpenAI: openAIProviderConfigV0{providerConfigV0: providerConfigV0{APIKey: "openai-key"}},
			Groq:   providerConfigV0{APIKey: "groq-key"},
			Zhipu:  providerConfigV0{APIKey: "zhipu-key"},
		},
	}

	result := v0ConvertProvidersToModelList(cfg)

	if len(result) != 3 {
		t.Fatalf("len(result) = %d, want 3", len(result))
	}

	// Check that all providers are present
	found := make(map[string]bool)
	for _, mc := range result {
		found[mc.ModelName] = true
	}

	for _, name := range []string{"openai", "groq", "zhipu"} {
		if !found[name] {
			t.Errorf("Missing provider %q in result", name)
		}
	}
}

func TestConvertProvidersToModelList_Empty(t *testing.T) {
	cfg := &configV0{
		Providers: providersConfigV0{},
	}

	result := v0ConvertProvidersToModelList(cfg)

	if len(result) != 0 {
		t.Errorf("len(result) = %d, want 0", len(result))
	}
}

func TestConvertProvidersToModelList_Nil(t *testing.T) {
	result := v0ConvertProvidersToModelList(nil)

	if result != nil {
		t.Errorf("result = %v, want nil", result)
	}
}

func TestConvertProvidersToModelList_AllProviders(t *testing.T) {
	// This test verifies that when providers have at least one configured field,
	// they are converted. GitHubCopilot has ConnectMode set, Antigravity has AuthMethod.
	// Other providers have no configuration, so they won't be converted.
	cfg := &configV0{
		Providers: providersConfigV0{
			OpenAI:        openAIProviderConfigV0{providerConfigV0: providerConfigV0{APIKey: "key1"}},
			LiteLLM:       providerConfigV0{APIKey: "key-litellm", APIBase: "http://localhost:4000/v1"},
			Anthropic:     providerConfigV0{APIKey: "key2"},
			OpenRouter:    providerConfigV0{APIKey: "key3"},
			Groq:          providerConfigV0{APIKey: "key4"},
			Zhipu:         providerConfigV0{APIKey: "key5"},
			VLLM:          providerConfigV0{APIKey: "key6"},
			Gemini:        providerConfigV0{APIKey: "key7"},
			Nvidia:        providerConfigV0{APIKey: "key8"},
			Ollama:        providerConfigV0{APIKey: "key9"},
			Moonshot:      providerConfigV0{APIKey: "key10"},
			ShengSuanYun:  providerConfigV0{APIKey: "key11"},
			DeepSeek:      providerConfigV0{APIKey: "key12"},
			Cerebras:      providerConfigV0{APIKey: "key13"},
			Vivgrid:       providerConfigV0{APIKey: "key14"},
			VolcEngine:    providerConfigV0{APIKey: "key15"},
			GitHubCopilot: providerConfigV0{ConnectMode: "grpc"},
			Antigravity:   providerConfigV0{AuthMethod: "oauth"},
			Qwen:          providerConfigV0{APIKey: "key17"},
			Mistral:       providerConfigV0{APIKey: "key18"},
			Avian:         providerConfigV0{APIKey: "key19"},
			LongCat:       providerConfigV0{APIKey: "key-longcat"},
			ModelScope:    providerConfigV0{APIKey: "key-modelscope"},
		},
	}

	result := v0ConvertProvidersToModelList(cfg)

	// All 23 providers should be converted
	if len(result) != 23 {
		t.Errorf("len(result) = %d, want 23", len(result))
	}
}

func TestConvertProvidersToModelList_Proxy(t *testing.T) {
	cfg := &configV0{
		Providers: providersConfigV0{
			OpenAI: openAIProviderConfigV0{
				providerConfigV0: providerConfigV0{
					APIKey: "key",
					Proxy:  "http://proxy:8080",
				},
			},
		},
	}

	result := v0ConvertProvidersToModelList(cfg)

	if len(result) != 1 {
		t.Fatalf("len(result) = %d, want 1", len(result))
	}

	if result[0].Proxy != "http://proxy:8080" {
		t.Errorf("Proxy = %q, want %q", result[0].Proxy, "http://proxy:8080")
	}
}

func TestConvertProvidersToModelList_RequestTimeout(t *testing.T) {
	cfg := &configV0{
		Providers: providersConfigV0{
			Ollama: providerConfigV0{
				APIBase:        "http://localhost:11434",
				RequestTimeout: 300,
			},
		},
	}

	result := v0ConvertProvidersToModelList(cfg)

	if len(result) != 1 {
		t.Fatalf("len(result) = %d, want 1", len(result))
	}

	if result[0].RequestTimeout != 300 {
		t.Errorf("RequestTimeout = %d, want %d", result[0].RequestTimeout, 300)
	}
}

func TestConvertProvidersToModelList_AuthMethod(t *testing.T) {
	cfg := &configV0{
		Providers: providersConfigV0{
			OpenAI: openAIProviderConfigV0{
				providerConfigV0: providerConfigV0{
					AuthMethod: "oauth",
				},
			},
		},
	}

	result := v0ConvertProvidersToModelList(cfg)

	if len(result) != 0 {
		t.Errorf("len(result) = %d, want 0 (AuthMethod alone should not create entry)", len(result))
	}
}

// Tests for preserving user's configured model during migration

func TestConvertProvidersToModelList_PreservesUserModel_DeepSeek(t *testing.T) {
	cfg := &configV0{
		Agents: agentsConfigV0{
			Defaults: agentDefaultsV0{
				Provider: "deepseek",
				Model:    "deepseek-reasoner",
			},
		},
		Providers: providersConfigV0{
			DeepSeek: providerConfigV0{APIKey: "sk-deepseek"},
		},
	}

	result := v0ConvertProvidersToModelList(cfg)

	if len(result) != 1 {
		t.Fatalf("len(result) = %d, want 1", len(result))
	}

	// Should use user's model, not default
	if result[0].Model != "deepseek/deepseek-reasoner" {
		t.Errorf("Model = %q, want %q (user's configured model)", result[0].Model, "deepseek/deepseek-reasoner")
	}
}

func TestConvertProvidersToModelList_PreservesUserModel_OpenAI(t *testing.T) {
	cfg := &configV0{
		Agents: agentsConfigV0{
			Defaults: agentDefaultsV0{
				Provider: "openai",
				Model:    "gpt-4-turbo",
			},
		},
		Providers: providersConfigV0{
			OpenAI: openAIProviderConfigV0{providerConfigV0: providerConfigV0{APIKey: "sk-openai"}},
		},
	}

	result := v0ConvertProvidersToModelList(cfg)

	if len(result) != 1 {
		t.Fatalf("len(result) = %d, want 1", len(result))
	}

	if result[0].Model != "openai/gpt-4-turbo" {
		t.Errorf("Model = %q, want %q", result[0].Model, "openai/gpt-4-turbo")
	}
}

func TestConvertProvidersToModelList_PreservesUserModel_Anthropic(t *testing.T) {
	cfg := &configV0{
		Agents: agentsConfigV0{
			Defaults: agentDefaultsV0{
				Provider: "claude", // alternative name
				Model:    "claude-opus-4-20250514",
			},
		},
		Providers: providersConfigV0{
			Anthropic: providerConfigV0{APIKey: "sk-ant"},
		},
	}

	result := v0ConvertProvidersToModelList(cfg)

	if len(result) != 1 {
		t.Fatalf("len(result) = %d, want 1", len(result))
	}

	if result[0].Model != "anthropic/claude-opus-4-20250514" {
		t.Errorf("Model = %q, want %q", result[0].Model, "anthropic/claude-opus-4-20250514")
	}
}

func TestConvertProvidersToModelList_PreservesUserModel_Qwen(t *testing.T) {
	cfg := &configV0{
		Agents: agentsConfigV0{
			Defaults: agentDefaultsV0{
				Provider: "qwen",
				Model:    "qwen-plus",
			},
		},
		Providers: providersConfigV0{
			Qwen: providerConfigV0{APIKey: "sk-qwen"},
		},
	}

	result := v0ConvertProvidersToModelList(cfg)

	if len(result) != 1 {
		t.Fatalf("len(result) = %d, want 1", len(result))
	}

	if result[0].Model != "qwen/qwen-plus" {
		t.Errorf("Model = %q, want %q", result[0].Model, "qwen/qwen-plus")
	}
}

func TestConvertProvidersToModelList_UsesDefaultWhenNoUserModel(t *testing.T) {
	cfg := &configV0{
		Agents: agentsConfigV0{
			Defaults: agentDefaultsV0{
				Provider: "deepseek",
				Model:    "", // no model specified
			},
		},
		Providers: providersConfigV0{
			DeepSeek: providerConfigV0{APIKey: "sk-deepseek"},
		},
	}

	result := v0ConvertProvidersToModelList(cfg)

	if len(result) != 1 {
		t.Fatalf("len(result) = %d, want 1", len(result))
	}

	// Should use default model
	if result[0].Model != "deepseek/deepseek-chat" {
		t.Errorf("Model = %q, want %q (default)", result[0].Model, "deepseek/deepseek-chat")
	}
}

func TestConvertProvidersToModelList_MultipleProviders_PreservesUserModel(t *testing.T) {
	cfg := &configV0{
		Agents: agentsConfigV0{
			Defaults: agentDefaultsV0{
				Provider: "deepseek",
				Model:    "deepseek-reasoner",
			},
		},
		Providers: providersConfigV0{
			OpenAI:   openAIProviderConfigV0{providerConfigV0: providerConfigV0{APIKey: "sk-openai"}},
			DeepSeek: providerConfigV0{APIKey: "sk-deepseek"},
		},
	}

	result := v0ConvertProvidersToModelList(cfg)

	if len(result) != 2 {
		t.Fatalf("len(result) = %d, want 2", len(result))
	}

	// Find each provider and verify model
	for _, mc := range result {
		switch mc.ModelName {
		case "openai":
			if mc.Model != "openai/gpt-5.4" {
				t.Errorf("OpenAI Model = %q, want %q (default)", mc.Model, "openai/gpt-5.4")
			}
		case "deepseek":
			if mc.Model != "deepseek/deepseek-reasoner" {
				t.Errorf("DeepSeek Model = %q, want %q (user's)", mc.Model, "deepseek/deepseek-reasoner")
			}
		}
	}
}

func TestConvertProvidersToModelList_ProviderNameAliases(t *testing.T) {
	tests := []struct {
		providerAlias string
		expectedModel string
		provider      providerConfigV0
	}{
		{"gpt", "openai/gpt-4-custom", providerConfigV0{APIKey: "key"}},
		{"claude", "anthropic/claude-custom", providerConfigV0{APIKey: "key"}},
		{"doubao", "volcengine/doubao-custom", providerConfigV0{APIKey: "key"}},
		{"tongyi", "qwen/qwen-custom", providerConfigV0{APIKey: "key"}},
		{"kimi", "moonshot/kimi-custom", providerConfigV0{APIKey: "key"}},
	}

	for _, tt := range tests {
		t.Run(tt.providerAlias, func(t *testing.T) {
			cfg := &configV0{
				Agents: agentsConfigV0{
					Defaults: agentDefaultsV0{
						Provider: tt.providerAlias,
						Model: strings.TrimPrefix(
							tt.expectedModel,
							tt.expectedModel[:strings.Index(tt.expectedModel, "/")+1],
						),
					},
				},
				Providers: providersConfigV0{},
			}

			// Set the appropriate provider config
			switch tt.providerAlias {
			case "gpt":
				cfg.Providers.OpenAI = openAIProviderConfigV0{providerConfigV0: tt.provider}
			case "claude":
				cfg.Providers.Anthropic = tt.provider
			case "doubao":
				cfg.Providers.VolcEngine = tt.provider
			case "tongyi":
				cfg.Providers.Qwen = tt.provider
			case "kimi":
				cfg.Providers.Moonshot = tt.provider
			}

			// Need to fix the model name in config
			cfg.Agents.Defaults.Model = strings.TrimPrefix(
				tt.expectedModel,
				tt.expectedModel[:strings.Index(tt.expectedModel, "/")+1],
			)

			result := v0ConvertProvidersToModelList(cfg)
			if len(result) != 1 {
				t.Fatalf("len(result) = %d, want 1", len(result))
			}

			// Extract just the model ID part (after the first /)
			expectedModelID := tt.expectedModel
			if result[0].Model != expectedModelID {
				t.Errorf("Model = %q, want %q", result[0].Model, expectedModelID)
			}
		})
	}
}

// Test for backward compatibility: single provider without explicit provider field
// This matches the legacy config pattern where users only set model, not provider

func TestConvertProvidersToModelList_NoProviderField_SingleProvider(t *testing.T) {
	// This matches the user's actual config:
	// - No provider field set
	// - model = "glm-4.7"
	// - Only zhipu has API key configured
	cfg := &configV0{
		Agents: agentsConfigV0{
			Defaults: agentDefaultsV0{
				Provider: "", // Not set
				Model:    "glm-4.7",
			},
		},
		Providers: providersConfigV0{
			Zhipu: providerConfigV0{
				APIKey: "test-zhipu-key",
			},
		},
	}

	result := v0ConvertProvidersToModelList(cfg)

	if len(result) != 1 {
		t.Fatalf("len(result) = %d, want 1", len(result))
	}

	// ModelName should be the user's model value for backward compatibility
	if result[0].ModelName != "glm-4.7" {
		t.Errorf("ModelName = %q, want %q (user's model for backward compatibility)", result[0].ModelName, "glm-4.7")
	}

	// Model should use the user's model with protocol prefix
	if result[0].Model != "zhipu/glm-4.7" {
		t.Errorf("Model = %q, want %q", result[0].Model, "zhipu/glm-4.7")
	}
}

func TestConvertProvidersToModelList_NoProviderField_MultipleProviders(t *testing.T) {
	// When multiple providers are configured but no provider field is set,
	// the FIRST provider (in migration order) will use userModel as ModelName
	// for backward compatibility with legacy implicit provider selection
	cfg := &configV0{
		Agents: agentsConfigV0{
			Defaults: agentDefaultsV0{
				Provider: "", // Not set
				Model:    "some-model",
			},
		},
		Providers: providersConfigV0{
			OpenAI: openAIProviderConfigV0{providerConfigV0: providerConfigV0{APIKey: "openai-key"}},
			Zhipu:  providerConfigV0{APIKey: "zhipu-key"},
		},
	}

	result := v0ConvertProvidersToModelList(cfg)

	if len(result) != 2 {
		t.Fatalf("len(result) = %d, want 2", len(result))
	}

	// The first provider (OpenAI in migration order) should use userModel as ModelName
	// This ensures GetModelConfig("some-model") will find it
	if result[0].ModelName != "some-model" {
		t.Errorf("First provider ModelName = %q, want %q", result[0].ModelName, "some-model")
	}

	// Other providers should use provider name as ModelName
	if result[1].ModelName != "zhipu" {
		t.Errorf("Second provider ModelName = %q, want %q", result[1].ModelName, "zhipu")
	}
}

func TestConvertProvidersToModelList_NoProviderField_NoModel(t *testing.T) {
	// Edge case: no provider, no model
	cfg := &configV0{
		Agents: agentsConfigV0{
			Defaults: agentDefaultsV0{
				Provider: "",
				Model:    "",
			},
		},
		Providers: providersConfigV0{
			Zhipu: providerConfigV0{APIKey: "zhipu-key"},
		},
	}

	result := v0ConvertProvidersToModelList(cfg)

	if len(result) != 1 {
		t.Fatalf("len(result) = %d, want 1", len(result))
	}

	// Should use default provider name since no model is specified
	if result[0].ModelName != "zhipu" {
		t.Errorf("ModelName = %q, want %q", result[0].ModelName, "zhipu")
	}
}

// Tests for buildModelWithProtocol helper function

func TestBuildModelWithProtocol_NoPrefix(t *testing.T) {
	result := buildModelWithProtocol("openai", "gpt-5.4")
	if result != "openai/gpt-5.4" {
		t.Errorf("buildModelWithProtocol(openai, gpt-5.4) = %q, want %q", result, "openai/gpt-5.4")
	}
}

func TestBuildModelWithProtocol_AlreadyHasPrefix(t *testing.T) {
	result := buildModelWithProtocol("openrouter", "openrouter/auto")
	if result != "openrouter/auto" {
		t.Errorf("buildModelWithProtocol(openrouter, openrouter/auto) = %q, want %q", result, "openrouter/auto")
	}
}

func TestBuildModelWithProtocol_DifferentPrefix(t *testing.T) {
	result := buildModelWithProtocol("anthropic", "openrouter/claude-sonnet-4.6")
	if result != "openrouter/claude-sonnet-4.6" {
		t.Errorf(
			"buildModelWithProtocol(anthropic, openrouter/claude-sonnet-4.6) = %q, want %q",
			result,
			"openrouter/claude-sonnet-4.6",
		)
	}
}

// Test for legacy config with protocol prefix in model name
func TestConvertProvidersToModelList_LegacyModelWithProtocolPrefix(t *testing.T) {
	cfg := &configV0{
		Agents: agentsConfigV0{
			Defaults: agentDefaultsV0{
				Provider: "",                // No explicit provider
				Model:    "openrouter/auto", // Model already has protocol prefix
			},
		},
		Providers: providersConfigV0{
			OpenRouter: providerConfigV0{APIKey: "sk-or-test"},
		},
	}

	result := v0ConvertProvidersToModelList(cfg)

	if len(result) < 1 {
		t.Fatalf("len(result) = %d, want at least 1", len(result))
	}

	// First provider should use userModel as ModelName for backward compatibility
	if result[0].ModelName != "openrouter/auto" {
		t.Errorf("ModelName = %q, want %q", result[0].ModelName, "openrouter/auto")
	}

	// Model should NOT have duplicated prefix
	if result[0].Model != "openrouter/auto" {
		t.Errorf("Model = %q, want %q (should not duplicate prefix)", result[0].Model, "openrouter/auto")
	}
}
