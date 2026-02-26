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
	cfg := &Config{
		Providers: ProvidersConfig{
			OpenAI: OpenAIProviderConfig{
				ProviderConfig: ProviderConfig{
					APIKey:  "sk-test-key",
					APIBase: "https://custom.api.com/v1",
				},
			},
		},
	}

	result := ConvertProvidersToModelList(cfg)

	if len(result) != 1 {
		t.Fatalf("len(result) = %d, want 1", len(result))
	}

	if result[0].ModelName != "openai" {
		t.Errorf("ModelName = %q, want %q", result[0].ModelName, "openai")
	}
	if result[0].Model != "openai/gpt-5.2" {
		t.Errorf("Model = %q, want %q", result[0].Model, "openai/gpt-5.2")
	}
	if result[0].APIKey != "sk-test-key" {
		t.Errorf("APIKey = %q, want %q", result[0].APIKey, "sk-test-key")
	}
}

func TestConvertProvidersToModelList_Anthropic(t *testing.T) {
	cfg := &Config{
		Providers: ProvidersConfig{
			Anthropic: ProviderConfig{
				APIKey:  "ant-key",
				APIBase: "https://custom.anthropic.com",
			},
		},
	}

	result := ConvertProvidersToModelList(cfg)

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

func TestConvertProvidersToModelList_Multiple(t *testing.T) {
	cfg := &Config{
		Providers: ProvidersConfig{
			OpenAI: OpenAIProviderConfig{ProviderConfig: ProviderConfig{APIKey: "openai-key"}},
			Groq:   ProviderConfig{APIKey: "groq-key"},
			Zhipu:  ProviderConfig{APIKey: "zhipu-key"},
		},
	}

	result := ConvertProvidersToModelList(cfg)

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
	cfg := &Config{
		Providers: ProvidersConfig{},
	}

	result := ConvertProvidersToModelList(cfg)

	if len(result) != 0 {
		t.Errorf("len(result) = %d, want 0", len(result))
	}
}

func TestConvertProvidersToModelList_Nil(t *testing.T) {
	result := ConvertProvidersToModelList(nil)

	if result != nil {
		t.Errorf("result = %v, want nil", result)
	}
}

func TestConvertProvidersToModelList_AllProviders(t *testing.T) {
	cfg := &Config{
		Providers: ProvidersConfig{
			OpenAI:        OpenAIProviderConfig{ProviderConfig: ProviderConfig{APIKey: "key1"}},
			Anthropic:     ProviderConfig{APIKey: "key2"},
			OpenRouter:    ProviderConfig{APIKey: "key3"},
			Groq:          ProviderConfig{APIKey: "key4"},
			Zhipu:         ProviderConfig{APIKey: "key5"},
			VLLM:          ProviderConfig{APIKey: "key6"},
			Gemini:        ProviderConfig{APIKey: "key7"},
			Nvidia:        ProviderConfig{APIKey: "key8"},
			Ollama:        ProviderConfig{APIKey: "key9"},
			Moonshot:      ProviderConfig{APIKey: "key10"},
			ShengSuanYun:  ProviderConfig{APIKey: "key11"},
			DeepSeek:      ProviderConfig{APIKey: "key12"},
			Cerebras:      ProviderConfig{APIKey: "key13"},
			VolcEngine:    ProviderConfig{APIKey: "key14"},
			GitHubCopilot: ProviderConfig{ConnectMode: "grpc"},
			Antigravity:   ProviderConfig{AuthMethod: "oauth"},
			Qwen:          ProviderConfig{APIKey: "key17"},
			Mistral:       ProviderConfig{APIKey: "key18"},
		},
	}

	result := ConvertProvidersToModelList(cfg)

	// All 18 providers should be converted
	if len(result) != 18 {
		t.Errorf("len(result) = %d, want 18", len(result))
	}
}

func TestConvertProvidersToModelList_Proxy(t *testing.T) {
	cfg := &Config{
		Providers: ProvidersConfig{
			OpenAI: OpenAIProviderConfig{
				ProviderConfig: ProviderConfig{
					APIKey: "key",
					Proxy:  "http://proxy:8080",
				},
			},
		},
	}

	result := ConvertProvidersToModelList(cfg)

	if len(result) != 1 {
		t.Fatalf("len(result) = %d, want 1", len(result))
	}

	if result[0].Proxy != "http://proxy:8080" {
		t.Errorf("Proxy = %q, want %q", result[0].Proxy, "http://proxy:8080")
	}
}

func TestConvertProvidersToModelList_AuthMethod(t *testing.T) {
	cfg := &Config{
		Providers: ProvidersConfig{
			OpenAI: OpenAIProviderConfig{
				ProviderConfig: ProviderConfig{
					AuthMethod: "oauth",
				},
			},
		},
	}

	result := ConvertProvidersToModelList(cfg)

	if len(result) != 0 {
		t.Errorf("len(result) = %d, want 0 (AuthMethod alone should not create entry)", len(result))
	}
}

// Tests for preserving user's configured model during migration

func TestConvertProvidersToModelList_PreservesUserModel_DeepSeek(t *testing.T) {
	cfg := &Config{
		Agents: AgentsConfig{
			Defaults: AgentDefaults{
				Provider: "deepseek",
				Model:    "deepseek-reasoner",
			},
		},
		Providers: ProvidersConfig{
			DeepSeek: ProviderConfig{APIKey: "sk-deepseek"},
		},
	}

	result := ConvertProvidersToModelList(cfg)

	if len(result) != 1 {
		t.Fatalf("len(result) = %d, want 1", len(result))
	}

	// Should use user's model, not default
	if result[0].Model != "deepseek/deepseek-reasoner" {
		t.Errorf("Model = %q, want %q (user's configured model)", result[0].Model, "deepseek/deepseek-reasoner")
	}
}

func TestConvertProvidersToModelList_PreservesUserModel_OpenAI(t *testing.T) {
	cfg := &Config{
		Agents: AgentsConfig{
			Defaults: AgentDefaults{
				Provider: "openai",
				Model:    "gpt-4-turbo",
			},
		},
		Providers: ProvidersConfig{
			OpenAI: OpenAIProviderConfig{ProviderConfig: ProviderConfig{APIKey: "sk-openai"}},
		},
	}

	result := ConvertProvidersToModelList(cfg)

	if len(result) != 1 {
		t.Fatalf("len(result) = %d, want 1", len(result))
	}

	if result[0].Model != "openai/gpt-4-turbo" {
		t.Errorf("Model = %q, want %q", result[0].Model, "openai/gpt-4-turbo")
	}
}

func TestConvertProvidersToModelList_PreservesUserModel_Anthropic(t *testing.T) {
	cfg := &Config{
		Agents: AgentsConfig{
			Defaults: AgentDefaults{
				Provider: "claude", // alternative name
				Model:    "claude-opus-4-20250514",
			},
		},
		Providers: ProvidersConfig{
			Anthropic: ProviderConfig{APIKey: "sk-ant"},
		},
	}

	result := ConvertProvidersToModelList(cfg)

	if len(result) != 1 {
		t.Fatalf("len(result) = %d, want 1", len(result))
	}

	if result[0].Model != "anthropic/claude-opus-4-20250514" {
		t.Errorf("Model = %q, want %q", result[0].Model, "anthropic/claude-opus-4-20250514")
	}
}

func TestConvertProvidersToModelList_PreservesUserModel_Qwen(t *testing.T) {
	cfg := &Config{
		Agents: AgentsConfig{
			Defaults: AgentDefaults{
				Provider: "qwen",
				Model:    "qwen-plus",
			},
		},
		Providers: ProvidersConfig{
			Qwen: ProviderConfig{APIKey: "sk-qwen"},
		},
	}

	result := ConvertProvidersToModelList(cfg)

	if len(result) != 1 {
		t.Fatalf("len(result) = %d, want 1", len(result))
	}

	if result[0].Model != "qwen/qwen-plus" {
		t.Errorf("Model = %q, want %q", result[0].Model, "qwen/qwen-plus")
	}
}

func TestConvertProvidersToModelList_UsesDefaultWhenNoUserModel(t *testing.T) {
	cfg := &Config{
		Agents: AgentsConfig{
			Defaults: AgentDefaults{
				Provider: "deepseek",
				Model:    "", // no model specified
			},
		},
		Providers: ProvidersConfig{
			DeepSeek: ProviderConfig{APIKey: "sk-deepseek"},
		},
	}

	result := ConvertProvidersToModelList(cfg)

	if len(result) != 1 {
		t.Fatalf("len(result) = %d, want 1", len(result))
	}

	// Should use default model
	if result[0].Model != "deepseek/deepseek-chat" {
		t.Errorf("Model = %q, want %q (default)", result[0].Model, "deepseek/deepseek-chat")
	}
}

func TestConvertProvidersToModelList_MultipleProviders_PreservesUserModel(t *testing.T) {
	cfg := &Config{
		Agents: AgentsConfig{
			Defaults: AgentDefaults{
				Provider: "deepseek",
				Model:    "deepseek-reasoner",
			},
		},
		Providers: ProvidersConfig{
			OpenAI:   OpenAIProviderConfig{ProviderConfig: ProviderConfig{APIKey: "sk-openai"}},
			DeepSeek: ProviderConfig{APIKey: "sk-deepseek"},
		},
	}

	result := ConvertProvidersToModelList(cfg)

	if len(result) != 2 {
		t.Fatalf("len(result) = %d, want 2", len(result))
	}

	// Find each provider and verify model
	for _, mc := range result {
		switch mc.ModelName {
		case "openai":
			if mc.Model != "openai/gpt-5.2" {
				t.Errorf("OpenAI Model = %q, want %q (default)", mc.Model, "openai/gpt-5.2")
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
		provider      ProviderConfig
	}{
		{"gpt", "openai/gpt-4-custom", ProviderConfig{APIKey: "key"}},
		{"claude", "anthropic/claude-custom", ProviderConfig{APIKey: "key"}},
		{"doubao", "volcengine/doubao-custom", ProviderConfig{APIKey: "key"}},
		{"tongyi", "qwen/qwen-custom", ProviderConfig{APIKey: "key"}},
		{"kimi", "moonshot/kimi-custom", ProviderConfig{APIKey: "key"}},
	}

	for _, tt := range tests {
		t.Run(tt.providerAlias, func(t *testing.T) {
			cfg := &Config{
				Agents: AgentsConfig{
					Defaults: AgentDefaults{
						Provider: tt.providerAlias,
						Model: strings.TrimPrefix(
							tt.expectedModel,
							tt.expectedModel[:strings.Index(tt.expectedModel, "/")+1],
						),
					},
				},
				Providers: ProvidersConfig{},
			}

			// Set the appropriate provider config
			switch tt.providerAlias {
			case "gpt":
				cfg.Providers.OpenAI = OpenAIProviderConfig{ProviderConfig: tt.provider}
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

			result := ConvertProvidersToModelList(cfg)
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
	cfg := &Config{
		Agents: AgentsConfig{
			Defaults: AgentDefaults{
				Provider: "", // Not set
				Model:    "glm-4.7",
			},
		},
		Providers: ProvidersConfig{
			Zhipu: ProviderConfig{APIKey: "test-zhipu-key"},
		},
	}

	result := ConvertProvidersToModelList(cfg)

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
	cfg := &Config{
		Agents: AgentsConfig{
			Defaults: AgentDefaults{
				Provider: "", // Not set
				Model:    "some-model",
			},
		},
		Providers: ProvidersConfig{
			OpenAI: OpenAIProviderConfig{ProviderConfig: ProviderConfig{APIKey: "openai-key"}},
			Zhipu:  ProviderConfig{APIKey: "zhipu-key"},
		},
	}

	result := ConvertProvidersToModelList(cfg)

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
	cfg := &Config{
		Agents: AgentsConfig{
			Defaults: AgentDefaults{
				Provider: "",
				Model:    "",
			},
		},
		Providers: ProvidersConfig{
			Zhipu: ProviderConfig{APIKey: "zhipu-key"},
		},
	}

	result := ConvertProvidersToModelList(cfg)

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
	result := buildModelWithProtocol("openai", "gpt-5.2")
	if result != "openai/gpt-5.2" {
		t.Errorf("buildModelWithProtocol(openai, gpt-5.2) = %q, want %q", result, "openai/gpt-5.2")
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
	cfg := &Config{
		Agents: AgentsConfig{
			Defaults: AgentDefaults{
				Provider: "",                // No explicit provider
				Model:    "openrouter/auto", // Model already has protocol prefix
			},
		},
		Providers: ProvidersConfig{
			OpenRouter: ProviderConfig{APIKey: "sk-or-test"},
		},
	}

	result := ConvertProvidersToModelList(cfg)

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
