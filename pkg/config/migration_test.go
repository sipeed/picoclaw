// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package config

import (
	"testing"
)

func TestConvertProvidersToModelList_OpenAI(t *testing.T) {
	cfg := &Config{
		Providers: ProvidersConfig{
			OpenAI: ProviderConfig{
				APIKey: "sk-test-key",
				APIBase: "https://custom.api.com/v1",
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
	if result[0].Model != "openai/gpt-4o" {
		t.Errorf("Model = %q, want %q", result[0].Model, "openai/gpt-4o")
	}
	if result[0].APIKey != "sk-test-key" {
		t.Errorf("APIKey = %q, want %q", result[0].APIKey, "sk-test-key")
	}
}

func TestConvertProvidersToModelList_Anthropic(t *testing.T) {
	cfg := &Config{
		Providers: ProvidersConfig{
			Anthropic: ProviderConfig{
				APIKey: "ant-key",
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
	if result[0].Model != "anthropic/claude-3-sonnet" {
		t.Errorf("Model = %q, want %q", result[0].Model, "anthropic/claude-3-sonnet")
	}
}

func TestConvertProvidersToModelList_Multiple(t *testing.T) {
	cfg := &Config{
		Providers: ProvidersConfig{
			OpenAI: ProviderConfig{APIKey: "openai-key"},
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
			OpenAI:       ProviderConfig{APIKey: "key1"},
			Anthropic:    ProviderConfig{APIKey: "key2"},
			OpenRouter:   ProviderConfig{APIKey: "key3"},
			Groq:         ProviderConfig{APIKey: "key4"},
			Zhipu:        ProviderConfig{APIKey: "key5"},
			VLLM:         ProviderConfig{APIKey: "key6"},
			Gemini:       ProviderConfig{APIKey: "key7"},
			Nvidia:       ProviderConfig{APIKey: "key8"},
			Ollama:       ProviderConfig{APIKey: "key9"},
			Moonshot:     ProviderConfig{APIKey: "key10"},
			ShengSuanYun: ProviderConfig{APIKey: "key11"},
			DeepSeek:     ProviderConfig{APIKey: "key12"},
			Cerebras:     ProviderConfig{APIKey: "key13"},
			VolcEngine:   ProviderConfig{APIKey: "key14"},
			GitHubCopilot: ProviderConfig{ConnectMode: "grpc"},
			Antigravity:  ProviderConfig{AuthMethod: "oauth"},
			Qwen:         ProviderConfig{APIKey: "key17"},
		},
	}

	result := ConvertProvidersToModelList(cfg)

	// All 17 providers should be converted
	if len(result) != 17 {
		t.Errorf("len(result) = %d, want 17", len(result))
	}
}

func TestConvertProvidersToModelList_Proxy(t *testing.T) {
	cfg := &Config{
		Providers: ProvidersConfig{
			OpenAI: ProviderConfig{
				APIKey: "key",
				Proxy:  "http://proxy:8080",
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
			OpenAI: ProviderConfig{
				AuthMethod: "oauth",
			},
		},
	}

	result := ConvertProvidersToModelList(cfg)

	if len(result) != 0 {
		t.Errorf("len(result) = %d, want 0 (AuthMethod alone should not create entry)", len(result))
	}
}
