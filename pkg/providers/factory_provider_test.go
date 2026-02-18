// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package providers

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestExtractProtocol(t *testing.T) {
	tests := []struct {
		name          string
		model         string
		wantProtocol  string
		wantModelID   string
	}{
		{
			name:         "openai with prefix",
			model:        "openai/gpt-4o",
			wantProtocol: "openai",
			wantModelID:  "gpt-4o",
		},
		{
			name:         "anthropic with prefix",
			model:        "anthropic/claude-3-sonnet",
			wantProtocol: "anthropic",
			wantModelID:  "claude-3-sonnet",
		},
		{
			name:         "no prefix - defaults to openai",
			model:        "gpt-4o",
			wantProtocol: "openai",
			wantModelID:  "gpt-4o",
		},
		{
			name:         "groq with prefix",
			model:        "groq/llama-3.1-70b",
			wantProtocol: "groq",
			wantModelID:  "llama-3.1-70b",
		},
		{
			name:         "empty string",
			model:        "",
			wantProtocol: "openai",
			wantModelID:  "",
		},
		{
			name:         "with whitespace",
			model:        "  openai/gpt-4  ",
			wantProtocol: "openai",
			wantModelID:  "gpt-4",
		},
		{
			name:         "multiple slashes",
			model:        "nvidia/meta/llama-3.1-8b",
			wantProtocol: "nvidia",
			wantModelID:  "meta/llama-3.1-8b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			protocol, modelID := ExtractProtocol(tt.model)
			if protocol != tt.wantProtocol {
				t.Errorf("ExtractProtocol(%q) protocol = %q, want %q", tt.model, protocol, tt.wantProtocol)
			}
			if modelID != tt.wantModelID {
				t.Errorf("ExtractProtocol(%q) modelID = %q, want %q", tt.model, modelID, tt.wantModelID)
			}
		})
	}
}

func TestCreateProviderFromConfig_OpenAI(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "test-openai",
		Model:     "openai/gpt-4o",
		APIKey:    "test-key",
		APIBase:   "https://api.example.com/v1",
	}

	provider, modelID, err := CreateProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateProviderFromConfig() error = %v", err)
	}
	if provider == nil {
		t.Fatal("CreateProviderFromConfig() returned nil provider")
	}
	if modelID != "gpt-4o" {
		t.Errorf("modelID = %q, want %q", modelID, "gpt-4o")
	}
}

func TestCreateProviderFromConfig_DefaultAPIBase(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		wantBase string
	}{
		{"openai", "openai", "https://api.openai.com/v1"},
		{"groq", "groq", "https://api.groq.com/openai/v1"},
		{"openrouter", "openrouter", "https://openrouter.ai/api/v1"},
		{"cerebras", "cerebras", "https://api.cerebras.ai/v1"},
		{"qwen", "qwen", "https://dashscope.aliyuncs.com/compatible-mode/v1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.ModelConfig{
				ModelName: "test-" + tt.protocol,
				Model:     tt.protocol + "/test-model",
				APIKey:    "test-key",
			}

			provider, _, err := CreateProviderFromConfig(cfg)
			if err != nil {
				t.Fatalf("CreateProviderFromConfig() error = %v", err)
			}

			httpProvider, ok := provider.(*HTTPProvider)
			if !ok {
				t.Fatalf("expected *HTTPProvider, got %T", provider)
			}
			if httpProvider.apiBase != tt.wantBase {
				t.Errorf("apiBase = %q, want %q", httpProvider.apiBase, tt.wantBase)
			}
		})
	}
}

func TestCreateProviderFromConfig_Anthropic(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "test-anthropic",
		Model:     "anthropic/claude-3-sonnet",
		APIKey:    "test-key",
	}

	provider, modelID, err := CreateProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateProviderFromConfig() error = %v", err)
	}
	if provider == nil {
		t.Fatal("CreateProviderFromConfig() returned nil provider")
	}
	if modelID != "claude-3-sonnet" {
		t.Errorf("modelID = %q, want %q", modelID, "claude-3-sonnet")
	}
}

func TestCreateProviderFromConfig_Antigravity(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "test-antigravity",
		Model:     "antigravity/gemini-2.0-flash",
	}

	provider, modelID, err := CreateProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateProviderFromConfig() error = %v", err)
	}
	if provider == nil {
		t.Fatal("CreateProviderFromConfig() returned nil provider")
	}
	if modelID != "gemini-2.0-flash" {
		t.Errorf("modelID = %q, want %q", modelID, "gemini-2.0-flash")
	}
}

func TestCreateProviderFromConfig_ClaudeCLI(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "test-claude-cli",
		Model:     "claude-cli/claude-sonnet-4-20250514",
	}

	provider, modelID, err := CreateProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateProviderFromConfig() error = %v", err)
	}
	if provider == nil {
		t.Fatal("CreateProviderFromConfig() returned nil provider")
	}
	if modelID != "claude-sonnet-4-20250514" {
		t.Errorf("modelID = %q, want %q", modelID, "claude-sonnet-4-20250514")
	}
}

func TestCreateProviderFromConfig_CodexCLI(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "test-codex-cli",
		Model:     "codex-cli/codex",
	}

	provider, modelID, err := CreateProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateProviderFromConfig() error = %v", err)
	}
	if provider == nil {
		t.Fatal("CreateProviderFromConfig() returned nil provider")
	}
	if modelID != "codex" {
		t.Errorf("modelID = %q, want %q", modelID, "codex")
	}
}

func TestCreateProviderFromConfig_MissingAPIKey(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "test-no-key",
		Model:     "openai/gpt-4o",
	}

	_, _, err := CreateProviderFromConfig(cfg)
	if err == nil {
		t.Fatal("CreateProviderFromConfig() expected error for missing API key")
	}
}

func TestCreateProviderFromConfig_UnknownProtocol(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "test-unknown",
		Model:     "unknown-protocol/model",
		APIKey:    "test-key",
	}

	_, _, err := CreateProviderFromConfig(cfg)
	if err == nil {
		t.Fatal("CreateProviderFromConfig() expected error for unknown protocol")
	}
}

func TestCreateProviderFromConfig_NilConfig(t *testing.T) {
	_, _, err := CreateProviderFromConfig(nil)
	if err == nil {
		t.Fatal("CreateProviderFromConfig(nil) expected error")
	}
}

func TestCreateProviderFromConfig_EmptyModel(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "test-empty",
		Model:     "",
	}

	_, _, err := CreateProviderFromConfig(cfg)
	if err == nil {
		t.Fatal("CreateProviderFromConfig() expected error for empty model")
	}
}
