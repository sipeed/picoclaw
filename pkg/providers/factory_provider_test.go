// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package providers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestExtractProtocol(t *testing.T) {
	tests := []struct {
		name         string
		config       *config.ModelConfig
		wantProtocol string
		wantModelID  string
	}{
		{
			name:         "openai with prefix",
			config:       &config.ModelConfig{Model: "openai/gpt-4o"},
			wantProtocol: "openai",
			wantModelID:  "gpt-4o",
		},
		{
			name:         "anthropic with prefix",
			config:       &config.ModelConfig{Model: "anthropic/claude-sonnet-4.6"},
			wantProtocol: "anthropic",
			wantModelID:  "claude-sonnet-4.6",
		},
		{
			name:         "no prefix - defaults to openai",
			config:       &config.ModelConfig{Model: "gpt-4o"},
			wantProtocol: "openai",
			wantModelID:  "gpt-4o",
		},
		{
			name:         "groq with prefix",
			config:       &config.ModelConfig{Model: "groq/llama-3.1-70b"},
			wantProtocol: "groq",
			wantModelID:  "llama-3.1-70b",
		},
		{
			name:         "empty string",
			config:       &config.ModelConfig{Model: ""},
			wantProtocol: "",
			wantModelID:  "",
		},
		{
			name:         "with whitespace",
			config:       &config.ModelConfig{Model: "  openai/gpt-4  "},
			wantProtocol: "openai",
			wantModelID:  "gpt-4",
		},
		{
			name:         "multiple slashes",
			config:       &config.ModelConfig{Model: "nvidia/meta/llama-3.1-8b"},
			wantProtocol: "nvidia",
			wantModelID:  "meta/llama-3.1-8b",
		},
		{
			name:         "normalizes provider",
			config:       &config.ModelConfig{Model: "z.ai/glm-5.1"},
			wantProtocol: "zai",
			wantModelID:  "glm-5.1",
		},
		{
			name:         "azure with prefix",
			config:       &config.ModelConfig{Model: "azure/my-gpt5-deployment"},
			wantProtocol: "azure",
			wantModelID:  "my-gpt5-deployment",
		},
		{
			name:         "explicit provider keeps model",
			config:       &config.ModelConfig{Provider: "nvidia", Model: "z-ai/glm-5.1"},
			wantProtocol: "nvidia",
			wantModelID:  "z-ai/glm-5.1",
		},
		{
			name:         "explicit provider preserves matching prefix",
			config:       &config.ModelConfig{Provider: "openai", Model: "openai/gpt-4o"},
			wantProtocol: "openai",
			wantModelID:  "openai/gpt-4o",
		},
		{
			name:         "explicit provider preserves aliased prefix",
			config:       &config.ModelConfig{Provider: "qwen", Model: "qwen/qwen-plus"},
			wantProtocol: "qwen-portal",
			wantModelID:  "qwen/qwen-plus",
		},
		{
			name:         "empty provider segment",
			config:       &config.ModelConfig{Model: "/gpt-4o"},
			wantProtocol: "",
			wantModelID:  "gpt-4o",
		},
		{
			name:         "nil config",
			wantProtocol: "",
			wantModelID:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			protocol, modelID := ExtractProtocol(tt.config)
			if protocol != tt.wantProtocol {
				t.Errorf("ExtractProtocol() protocol = %q, want %q", protocol, tt.wantProtocol)
			}
			if modelID != tt.wantModelID {
				t.Errorf("ExtractProtocol() modelID = %q, want %q", modelID, tt.wantModelID)
			}
		})
	}
}

func TestCreateProviderFromConfig_OpenAI(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "test-openai",
		Model:     "openai/gpt-4o",
		APIBase:   "https://api.example.com/v1",
	}
	cfg.SetAPIKey("test-key")

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

func TestCreateProviderFromConfig_UsesExplicitProvider(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "test-explicit-provider",
		Model:     "z-ai/glm-5.1",
		Provider:  "nvidia",
	}
	cfg.SetAPIKey("test-key")

	provider, modelID, err := CreateProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateProviderFromConfig() error = %v", err)
	}
	if provider == nil {
		t.Fatal("CreateProviderFromConfig() returned nil provider")
	}
	if modelID != "z-ai/glm-5.1" {
		t.Fatalf("modelID = %q, want z-ai/glm-5.1", modelID)
	}
	if got := ResolveAPIBase(cfg); got != "https://integrate.api.nvidia.com/v1" {
		t.Fatalf("ResolveAPIBase() = %q, want NVIDIA default API base", got)
	}
}

func TestCreateProviderFromConfig_PreservesExplicitProviderPrefixedModel(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "test-openai",
		Provider:  "openai",
		Model:     "openai/gpt-4o",
		APIBase:   "https://api.example.com/v1",
	}
	cfg.SetAPIKey("test-key")

	provider, modelID, err := CreateProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateProviderFromConfig() error = %v", err)
	}
	if provider == nil {
		t.Fatal("CreateProviderFromConfig() returned nil provider")
	}
	if modelID != "openai/gpt-4o" {
		t.Fatalf("modelID = %q, want %q", modelID, "openai/gpt-4o")
	}
}

func TestCreateProviderFromConfig_DefaultAPIBase(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
	}{
		{"openai", "openai"},
		{"venice", "venice"},
		{"groq", "groq"},
		{"novita", "novita"},
		{"openrouter", "openrouter"},
		{"cerebras", "cerebras"},
		{"vivgrid", "vivgrid"},
		{"qwen", "qwen"},
		{"vllm", "vllm"},
		{"ovms", "ovms"},
		{"deepseek", "deepseek"},
		{"ollama", "ollama"},
		{"lmstudio", "lmstudio"},
		{"longcat", "longcat"},
		{"modelscope", "modelscope"},
		{"mimo", "mimo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.ModelConfig{
				ModelName: "test-" + tt.protocol,
				Model:     tt.protocol + "/test-model",
			}
			cfg.SetAPIKey("test-key")

			provider, _, err := CreateProviderFromConfig(cfg)
			if err != nil {
				t.Fatalf("CreateProviderFromConfig() error = %v", err)
			}

			// Verify we got an HTTPProvider for all these protocols
			if _, ok := provider.(*HTTPProvider); !ok {
				t.Fatalf("expected *HTTPProvider, got %T", provider)
			}
		})
	}
}

func TestGetDefaultAPIBase_LiteLLM(t *testing.T) {
	if got := getDefaultAPIBase("litellm"); got != "http://localhost:4000/v1" {
		t.Fatalf("getDefaultAPIBase(%q) = %q, want %q", "litellm", got, "http://localhost:4000/v1")
	}
}

func TestGetDefaultAPIBase_LMStudio(t *testing.T) {
	if got := getDefaultAPIBase("lmstudio"); got != "http://localhost:1234/v1" {
		t.Fatalf("getDefaultAPIBase(%q) = %q, want %q", "lmstudio", got, "http://localhost:1234/v1")
	}
}

func TestGetDefaultAPIBase_Venice(t *testing.T) {
	if got := getDefaultAPIBase("venice"); got != "https://api.venice.ai/api/v1" {
		t.Fatalf("getDefaultAPIBase(%q) = %q, want %q", "venice", got, "https://api.venice.ai/api/v1")
	}
}

func TestCreateProviderFromConfig_LiteLLM(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "test-litellm",
		Model:     "litellm/my-proxy-alias",
		APIBase:   "http://localhost:4000/v1",
	}
	cfg.SetAPIKey("test-key")

	provider, modelID, err := CreateProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateProviderFromConfig() error = %v", err)
	}
	if provider == nil {
		t.Fatal("CreateProviderFromConfig() returned nil provider")
	}
	if modelID != "my-proxy-alias" {
		t.Errorf("modelID = %q, want %q", modelID, "my-proxy-alias")
	}
}

func TestCreateProviderFromConfig_LocalProviders(t *testing.T) {
	tests := []struct {
		name        string
		modelName   string
		model       string
		apiKey      string
		wantModelID string
	}{
		{
			name:        "LMStudio with API key",
			modelName:   "test-lmstudio",
			model:       "lmstudio/openai/gpt-oss-20b",
			apiKey:      "test-key",
			wantModelID: "openai/gpt-oss-20b",
		},
		{
			name:        "LMStudio without API key",
			modelName:   "test-lmstudio",
			model:       "lmstudio/openai/gpt-oss-20b",
			apiKey:      "",
			wantModelID: "openai/gpt-oss-20b",
		},
		{
			name:        "Ollama with API key",
			modelName:   "test-ollama",
			model:       "ollama/llama3.1:8b",
			apiKey:      "test-key",
			wantModelID: "llama3.1:8b",
		},
		{
			name:        "Ollama without API key",
			modelName:   "test-ollama",
			model:       "ollama/llama3.1:8b",
			apiKey:      "",
			wantModelID: "llama3.1:8b",
		},
		{
			name:        "VLLM with API key",
			modelName:   "test-vllm",
			model:       "vllm/Qwen/Qwen3-8B",
			apiKey:      "test-key",
			wantModelID: "Qwen/Qwen3-8B",
		},
		{
			name:        "VLLM without API key",
			modelName:   "test-vllm",
			model:       "vllm/Qwen/Qwen3-8B",
			apiKey:      "",
			wantModelID: "Qwen/Qwen3-8B",
		},
		{
			name:        "OVMS with API key",
			modelName:   "test-ovms",
			model:       "ovms/llama3",
			apiKey:      "test-key",
			wantModelID: "llama3",
		},
		{
			name:        "OVMS without API key",
			modelName:   "test-ovms",
			model:       "ovms/llama3",
			apiKey:      "",
			wantModelID: "llama3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.ModelConfig{
				ModelName: tt.modelName,
				Model:     tt.model,
			}
			if tt.apiKey != "" {
				cfg.SetAPIKey(tt.apiKey)
			}

			provider, modelID, err := CreateProviderFromConfig(cfg)
			if err != nil {
				t.Fatalf("CreateProviderFromConfig() error = %v", err)
			}
			if provider == nil {
				t.Fatal("CreateProviderFromConfig() returned nil provider")
			}
			if modelID != tt.wantModelID {
				t.Errorf("modelID = %q, want %q", modelID, tt.wantModelID)
			}
			if _, ok := provider.(*HTTPProvider); !ok {
				t.Fatalf("expected *HTTPProvider, got %T", provider)
			}
		})
	}
}

func TestCreateProviderFromConfig_LongCat(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "test-longcat",
		Model:     "longcat/LongCat-Flash-Thinking",
		APIBase:   "https://api.longcat.chat/openai",
	}
	cfg.SetAPIKey("test-key")

	provider, modelID, err := CreateProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateProviderFromConfig() error = %v", err)
	}
	if provider == nil {
		t.Fatal("CreateProviderFromConfig() returned nil provider")
	}
	if modelID != "LongCat-Flash-Thinking" {
		t.Errorf("modelID = %q, want %q", modelID, "LongCat-Flash-Thinking")
	}
	if _, ok := provider.(*HTTPProvider); !ok {
		t.Fatalf("expected *HTTPProvider, got %T", provider)
	}
}

func TestCreateProviderFromConfig_ModelScope(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "test-modelscope",
		Model:     "modelscope/Qwen/Qwen3-235B-A22B-Instruct-2507",
		APIBase:   "https://api-inference.modelscope.cn/v1",
	}
	cfg.SetAPIKey("test-key")

	provider, modelID, err := CreateProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateProviderFromConfig() error = %v", err)
	}
	if provider == nil {
		t.Fatal("CreateProviderFromConfig() returned nil provider")
	}
	if modelID != "Qwen/Qwen3-235B-A22B-Instruct-2507" {
		t.Errorf("modelID = %q, want %q", modelID, "Qwen/Qwen3-235B-A22B-Instruct-2507")
	}
	if _, ok := provider.(*HTTPProvider); !ok {
		t.Fatalf("expected *HTTPProvider, got %T", provider)
	}
}

func TestGetDefaultAPIBase_ModelScope(t *testing.T) {
	if got := getDefaultAPIBase("modelscope"); got != "https://api-inference.modelscope.cn/v1" {
		t.Fatalf("getDefaultAPIBase(%q) = %q, want %q", "modelscope", got, "https://api-inference.modelscope.cn/v1")
	}
}

func TestCreateProviderFromConfig_Novita(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "test-novita",
		Model:     "novita/deepseek/deepseek-v3.2",
	}
	cfg.SetAPIKey("test-key")

	provider, modelID, err := CreateProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateProviderFromConfig() error = %v", err)
	}
	if provider == nil {
		t.Fatal("CreateProviderFromConfig() returned nil provider")
	}
	if modelID != "deepseek/deepseek-v3.2" {
		t.Errorf("modelID = %q, want %q", modelID, "deepseek/deepseek-v3.2")
	}
	if _, ok := provider.(*HTTPProvider); !ok {
		t.Fatalf("expected *HTTPProvider, got %T", provider)
	}
}

func TestGetDefaultAPIBase_Novita(t *testing.T) {
	if got := getDefaultAPIBase("novita"); got != "https://api.novita.ai/openai" {
		t.Fatalf("getDefaultAPIBase(%q) = %q, want %q", "novita", got, "https://api.novita.ai/openai")
	}
}

func TestCreateProviderFromConfig_Mimo(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "test-mimo",
		Model:     "mimo/mimo-v2-pro",
		APIBase:   "https://api.xiaomimimo.com/v1",
	}
	cfg.SetAPIKey("test-key")

	provider, modelID, err := CreateProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateProviderFromConfig() error = %v", err)
	}
	if provider == nil {
		t.Fatal("CreateProviderFromConfig() returned nil provider")
	}
	if modelID != "mimo-v2-pro" {
		t.Errorf("modelID = %q, want %q", modelID, "mimo-v2-pro")
	}
	if _, ok := provider.(*HTTPProvider); !ok {
		t.Fatalf("expected *HTTPProvider, got %T", provider)
	}
}

func TestCreateProviderFromConfig_Venice(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "test-venice",
		Model:     "venice/venice-uncensored",
	}
	cfg.SetAPIKey("test-key")

	provider, modelID, err := CreateProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateProviderFromConfig() error = %v", err)
	}
	if provider == nil {
		t.Fatal("CreateProviderFromConfig() returned nil provider")
	}
	if modelID != "venice-uncensored" {
		t.Errorf("modelID = %q, want %q", modelID, "venice-uncensored")
	}
	if _, ok := provider.(*HTTPProvider); !ok {
		t.Fatalf("expected *HTTPProvider, got %T", provider)
	}
}

func TestGetDefaultAPIBase_Mimo(t *testing.T) {
	if got := getDefaultAPIBase("mimo"); got != "https://api.xiaomimimo.com/v1" {
		t.Fatalf("getDefaultAPIBase(%q) = %q, want %q", "mimo", got, "https://api.xiaomimimo.com/v1")
	}
}

func TestCreateProviderFromConfig_Anthropic(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "test-anthropic",
		Model:     "anthropic/claude-sonnet-4.6",
	}
	cfg.SetAPIKey("test-key")

	provider, modelID, err := CreateProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateProviderFromConfig() error = %v", err)
	}
	if provider == nil {
		t.Fatal("CreateProviderFromConfig() returned nil provider")
	}
	if modelID != "claude-sonnet-4.6" {
		t.Errorf("modelID = %q, want %q", modelID, "claude-sonnet-4.6")
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

func TestCreateProviderFromConfig_Gemini(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "test-gemini",
		Model:     "gemini/gemini-2.5-flash",
	}
	cfg.SetAPIKey("test-key")

	provider, modelID, err := CreateProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateProviderFromConfig() error = %v", err)
	}
	if provider == nil {
		t.Fatal("CreateProviderFromConfig() returned nil provider")
	}
	if modelID != "gemini-2.5-flash" {
		t.Errorf("modelID = %q, want %q", modelID, "gemini-2.5-flash")
	}
	if _, ok := provider.(*GeminiProvider); !ok {
		t.Fatalf("expected *GeminiProvider, got %T", provider)
	}
}

func TestCreateProviderFromConfig_GeminiMissingAPIKey(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "test-gemini-no-key",
		Model:     "gemini/gemini-2.5-flash",
	}

	_, _, err := CreateProviderFromConfig(cfg)
	if err == nil {
		t.Fatal("CreateProviderFromConfig() expected error for missing gemini API key")
	}
}

func TestCreateProviderFromConfig_GeminiCustomAPIBaseWithoutKey(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "test-gemini-custom-base",
		Model:     "gemini/gemini-2.5-flash",
		APIBase:   "https://proxy.example.com/v1beta",
	}

	provider, modelID, err := CreateProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateProviderFromConfig() error = %v", err)
	}
	if provider == nil {
		t.Fatal("CreateProviderFromConfig() returned nil provider")
	}
	if modelID != "gemini-2.5-flash" {
		t.Errorf("modelID = %q, want %q", modelID, "gemini-2.5-flash")
	}
	if _, ok := provider.(*GeminiProvider); !ok {
		t.Fatalf("expected *GeminiProvider, got %T", provider)
	}
}

func TestCreateProviderFromConfig_ClaudeCLI(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "test-claude-cli",
		Model:     "claude-cli/claude-sonnet-4.6",
	}

	provider, modelID, err := CreateProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateProviderFromConfig() error = %v", err)
	}
	if provider == nil {
		t.Fatal("CreateProviderFromConfig() returned nil provider")
	}
	if modelID != "claude-sonnet-4.6" {
		t.Errorf("modelID = %q, want %q", modelID, "claude-sonnet-4.6")
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
	}
	cfg.SetAPIKey("test-key")

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

func TestCreateProviderFromConfig_RequestTimeoutPropagation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1500 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"},"finish_reason":"stop"}]}`))
	}))
	defer server.Close()

	cfg := &config.ModelConfig{
		ModelName:      "test-timeout",
		Model:          "openai/gpt-4o",
		APIBase:        server.URL,
		RequestTimeout: 1,
	}
	cfg.SetAPIKey("test-key")

	provider, modelID, err := CreateProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateProviderFromConfig() error = %v", err)
	}
	if modelID != "gpt-4o" {
		t.Fatalf("modelID = %q, want %q", modelID, "gpt-4o")
	}

	_, err = provider.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hi"}},
		nil,
		modelID,
		nil,
	)
	if err == nil {
		t.Fatal("Chat() expected timeout error, got nil")
	}
	errMsg := err.Error()
	if !strings.Contains(errMsg, "context deadline exceeded") && !strings.Contains(errMsg, "Client.Timeout exceeded") {
		t.Fatalf("Chat() error = %q, want timeout-related error", errMsg)
	}
}

func TestCreateProviderFromConfig_Azure(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "azure-gpt5",
		Model:     "azure/my-gpt5-deployment",
		APIBase:   "https://my-resource.openai.azure.com",
	}
	cfg.SetAPIKey("test-azure-key")

	provider, modelID, err := CreateProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateProviderFromConfig() error = %v", err)
	}
	if provider == nil {
		t.Fatal("CreateProviderFromConfig() returned nil provider")
	}
	if modelID != "my-gpt5-deployment" {
		t.Errorf("modelID = %q, want %q", modelID, "my-gpt5-deployment")
	}
}

func TestCreateProviderFromConfig_AzureOpenAIAlias(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "azure-gpt4",
		Model:     "azure-openai/my-deployment",
		APIBase:   "https://my-resource.openai.azure.com",
	}
	cfg.SetAPIKey("test-azure-key")

	provider, modelID, err := CreateProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateProviderFromConfig() error = %v", err)
	}
	if provider == nil {
		t.Fatal("CreateProviderFromConfig() returned nil provider")
	}
	if modelID != "my-deployment" {
		t.Errorf("modelID = %q, want %q", modelID, "my-deployment")
	}
}

func TestCreateProviderFromConfig_AzureMissingAPIKey(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "azure-gpt5",
		Model:     "azure/my-gpt5-deployment",
		APIBase:   "https://my-resource.openai.azure.com",
	}

	_, _, err := CreateProviderFromConfig(cfg)
	if err == nil {
		t.Fatal("CreateProviderFromConfig() expected error for missing API key")
	}
}

func TestCreateProviderFromConfig_AzureMissingAPIBase(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "azure-gpt5",
		Model:     "azure/my-gpt5-deployment",
	}
	cfg.SetAPIKey("test-azure-key")

	_, _, err := CreateProviderFromConfig(cfg)
	if err == nil {
		t.Fatal("CreateProviderFromConfig() expected error for missing API base")
	}
}

func TestCreateProviderFromConfig_QwenInternationalAlias(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
	}{
		{"qwen-international", "qwen-international"},
		{"dashscope-intl", "dashscope-intl"},
		{"qwen-intl", "qwen-intl"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.ModelConfig{
				ModelName: "test-" + tt.protocol,
				Model:     tt.protocol + "/qwen-max",
			}
			cfg.SetAPIKey("test-key")

			provider, modelID, err := CreateProviderFromConfig(cfg)
			if err != nil {
				t.Fatalf("CreateProviderFromConfig() error = %v", err)
			}
			if provider == nil {
				t.Fatal("CreateProviderFromConfig() returned nil provider")
			}
			wantModelID := "qwen-max"
			if modelID != wantModelID {
				t.Errorf("modelID = %q, want %q", modelID, wantModelID)
			}
			if _, ok := provider.(*HTTPProvider); !ok {
				t.Fatalf("expected *HTTPProvider, got %T", provider)
			}
		})
	}
}

func TestCreateProviderFromConfig_QwenUSAlias(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
	}{
		{"qwen-us", "qwen-us"},
		{"dashscope-us", "dashscope-us"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.ModelConfig{
				ModelName: "test-" + tt.protocol,
				Model:     tt.protocol + "/qwen-max",
			}
			cfg.SetAPIKey("test-key")

			provider, modelID, err := CreateProviderFromConfig(cfg)
			if err != nil {
				t.Fatalf("CreateProviderFromConfig() error = %v", err)
			}
			if provider == nil {
				t.Fatal("CreateProviderFromConfig() returned nil provider")
			}
			wantModelID := "qwen-max"
			if modelID != wantModelID {
				t.Errorf("modelID = %q, want %q", modelID, wantModelID)
			}
			if _, ok := provider.(*HTTPProvider); !ok {
				t.Fatalf("expected *HTTPProvider, got %T", provider)
			}
		})
	}
}

func TestCreateProviderFromConfig_CodingPlanAnthropic(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
	}{
		{"coding-plan-anthropic", "coding-plan-anthropic"},
		{"alibaba-coding-anthropic", "alibaba-coding-anthropic"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.ModelConfig{
				ModelName: "test-" + tt.protocol,
				Model:     tt.protocol + "/claude-sonnet-4-20250514",
			}
			cfg.SetAPIKey("test-key")

			provider, modelID, err := CreateProviderFromConfig(cfg)
			if err != nil {
				t.Fatalf("CreateProviderFromConfig() error = %v", err)
			}
			if provider == nil {
				t.Fatal("CreateProviderFromConfig() returned nil provider")
			}
			wantModelID := "claude-sonnet-4-20250514"
			if modelID != wantModelID {
				t.Errorf("modelID = %q, want %q", modelID, wantModelID)
			}
			// coding-plan-anthropic uses Anthropic Messages provider
			// Verify it's the anthropic messages provider by checking interface
			var _ LLMProvider = provider
		})
	}
}

func TestGetDefaultAPIBase_CodingPlanAnthropic(t *testing.T) {
	expectedURL := "https://coding-intl.dashscope.aliyuncs.com/apps/anthropic"
	if got := getDefaultAPIBase("coding-plan-anthropic"); got != expectedURL {
		t.Fatalf("getDefaultAPIBase(%q) = %q, want %q", "coding-plan-anthropic", got, expectedURL)
	}
	if got := getDefaultAPIBase("alibaba-coding-anthropic"); got != expectedURL {
		t.Fatalf("getDefaultAPIBase(%q) = %q, want %q", "alibaba-coding-anthropic", got, expectedURL)
	}
}

func TestGetDefaultAPIBase_QwenIntlAliases(t *testing.T) {
	expectedURL := "https://dashscope-intl.aliyuncs.com/compatible-mode/v1"
	for _, protocol := range []string{"qwen-intl", "qwen-international", "dashscope-intl"} {
		if got := getDefaultAPIBase(protocol); got != expectedURL {
			t.Fatalf("getDefaultAPIBase(%q) = %q, want %q", protocol, got, expectedURL)
		}
	}
}

func TestGetDefaultAPIBase_QwenUSAliases(t *testing.T) {
	expectedURL := "https://dashscope-us.aliyuncs.com/compatible-mode/v1"
	for _, protocol := range []string{"qwen-us", "dashscope-us"} {
		if got := getDefaultAPIBase(protocol); got != expectedURL {
			t.Fatalf("getDefaultAPIBase(%q) = %q, want %q", protocol, got, expectedURL)
		}
	}
}

func TestCreateProviderFromConfig_MinimaxInjectsReasoningSplit(t *testing.T) {
	var requestBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"},"finish_reason":"stop"}]}`))
	}))
	defer server.Close()

	cfg := &config.ModelConfig{
		ModelName: "test-minimax",
		Model:     "minimax/MiniMax-M2.5",
		APIBase:   server.URL,
	}
	cfg.SetAPIKey("test-key")

	provider, modelID, err := CreateProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateProviderFromConfig() error = %v", err)
	}
	if provider == nil {
		t.Fatal("CreateProviderFromConfig() returned nil provider")
	}
	if modelID != "MiniMax-M2.5" {
		t.Errorf("modelID = %q, want %q", modelID, "MiniMax-M2.5")
	}

	_, err = provider.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hi"}},
		nil,
		modelID,
		nil,
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	// Verify reasoning_split is automatically injected
	if got, ok := requestBody["reasoning_split"]; !ok || got != true {
		t.Fatalf("reasoning_split = %v, want true", got)
	}
}

func TestCreateProviderFromConfig_MinimaxPreservesUserExtraBody(t *testing.T) {
	var requestBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"},"finish_reason":"stop"}]}`))
	}))
	defer server.Close()

	cfg := &config.ModelConfig{
		ModelName: "test-minimax-custom",
		Model:     "minimax/MiniMax-M2.5",
		APIBase:   server.URL,
		ExtraBody: map[string]any{"custom_field": "test"},
	}
	cfg.SetAPIKey("test-key")

	provider, modelID, err := CreateProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateProviderFromConfig() error = %v", err)
	}

	_, err = provider.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hi"}},
		nil,
		modelID,
		nil,
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	// Verify reasoning_split is automatically injected
	if got, ok := requestBody["reasoning_split"]; !ok || got != true {
		t.Fatalf("reasoning_split = %v, want true", got)
	}
	// Verify user's custom field is preserved
	if got, ok := requestBody["custom_field"]; !ok || got != "test" {
		t.Fatalf("custom_field = %v, want test", got)
	}
}

func TestCreateProviderFromConfig_CustomHeaders(t *testing.T) {
	var gotSource, gotAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSource = r.Header.Get("X-Source")
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"},"finish_reason":"stop"}]}`))
	}))
	defer server.Close()

	cfg := &config.ModelConfig{
		ModelName:     "test-headers",
		Model:         "openai/gpt-4o",
		APIBase:       server.URL,
		CustomHeaders: map[string]string{"X-Source": "coding-plan", "Authorization": "Token config-auth"},
	}
	cfg.SetAPIKey("test-key")

	provider, modelID, err := CreateProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateProviderFromConfig() error = %v", err)
	}

	_, err = provider.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hi"}},
		nil,
		modelID,
		nil,
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if gotSource != "coding-plan" {
		t.Fatalf("X-Source = %q, want %q", gotSource, "coding-plan")
	}
	if gotAuth != "Token config-auth" {
		t.Fatalf("Authorization = %q, want %q", gotAuth, "Token config-auth")
	}
}

// openaiCompatResponse is the JSON response used by OpenAI-compatible providers.
const openaiCompatResponse = `{"choices":[{"message":{"content":"ok"},"finish_reason":"stop"}]}`

// anthropicResponse is the JSON response used by Anthropic providers.
const anthropicResponse = `{"content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn","model":"claude-sonnet-4-20250514","usage":{"input_tokens":10,"output_tokens":5}}`

func TestCreateProviderFromConfig_UserAgent(t *testing.T) {
	defaultUA := "PicoClaw/" + config.Version

	tests := []struct {
		name      string
		model     string
		userAgent string
		apiKey    string
		response  string
		wantUA    string
		chatOpts  map[string]any
	}{
		{
			name:     "openai default user agent",
			model:    "openai/gpt-4o",
			apiKey:   "test-key",
			response: openaiCompatResponse,
			wantUA:   defaultUA,
		},
		{
			name:      "openai custom user agent",
			model:     "openai/gpt-4o",
			apiKey:    "test-key",
			userAgent: "MyAgent/1.2.3",
			response:  openaiCompatResponse,
			wantUA:    "MyAgent/1.2.3",
		},
		{
			name:     "anthropic default user agent",
			model:    "anthropic/claude-sonnet-4-20250514",
			apiKey:   "test-key",
			response: anthropicResponse,
			wantUA:   defaultUA,
		},
		{
			name:     "anthropic-messages default user agent",
			model:    "anthropic-messages/claude-sonnet-4-20250514",
			apiKey:   "test-key",
			response: anthropicResponse,
			wantUA:   defaultUA,
			chatOpts: map[string]any{"max_tokens": 1024},
		},
		{
			name:     "azure default user agent",
			model:    "azure/my-deployment",
			apiKey:   "test-azure-key",
			response: openaiCompatResponse,
			wantUA:   defaultUA,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedUA string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedUA = r.Header.Get("User-Agent")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.response))
			}))
			defer server.Close()

			cfg := &config.ModelConfig{
				ModelName: "test-ua-" + tt.name,
				Model:     tt.model,
				APIBase:   server.URL,
				UserAgent: tt.userAgent,
			}
			cfg.SetAPIKey(tt.apiKey)

			provider, modelID, err := CreateProviderFromConfig(cfg)
			if err != nil {
				t.Fatalf("CreateProviderFromConfig() error = %v", err)
			}
			if provider == nil {
				t.Fatal("CreateProviderFromConfig() returned nil provider")
			}

			_, err = provider.Chat(
				t.Context(),
				[]Message{{Role: "user", Content: "hi"}},
				nil,
				modelID,
				tt.chatOpts,
			)
			if err != nil {
				t.Fatalf("Chat() error = %v", err)
			}

			if receivedUA != tt.wantUA {
				t.Errorf("User-Agent = %q, want %q", receivedUA, tt.wantUA)
			}
		})
	}
}

func TestCreateProviderFromConfig_Bedrock(t *testing.T) {
	// Set dummy AWS env vars to make test deterministic
	t.Setenv("AWS_ACCESS_KEY_ID", "test-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	// Clear profile-related env vars to avoid loading shared config
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_DEFAULT_PROFILE", "")
	t.Setenv("AWS_SDK_LOAD_CONFIG", "")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "")

	cfg := &config.ModelConfig{
		ModelName: "bedrock-claude",
		Model:     "bedrock/us.anthropic.claude-sonnet-4-20250514-v1:0",
		APIBase:   "us-west-2", // Region (also sets AWS region)
	}

	provider, modelID, err := CreateProviderFromConfig(cfg)
	if err == nil {
		// Provider created successfully (built with -tags bedrock)
		if provider == nil {
			t.Error("provider is nil on success")
		}
		if modelID != "us.anthropic.claude-sonnet-4-20250514-v1:0" {
			t.Errorf("modelID = %q, want %q", modelID, "us.anthropic.claude-sonnet-4-20250514-v1:0")
		}
		return
	}
	errMsg := err.Error()
	// When built without -tags bedrock, expect stub error
	if strings.Contains(errMsg, "build with -tags bedrock") {
		return // Expected stub error
	}
	// Unexpected error - fail the test
	t.Errorf("unexpected error from bedrock provider: %v", err)
}

func TestCreateProviderFromConfig_BedrockWithEndpointURL(t *testing.T) {
	// Set dummy AWS env vars to make test deterministic
	t.Setenv("AWS_ACCESS_KEY_ID", "test-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")
	t.Setenv("AWS_REGION", "us-east-1") // Required when using endpoint URL
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	// Clear profile-related env vars to avoid loading shared config
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_DEFAULT_PROFILE", "")
	t.Setenv("AWS_SDK_LOAD_CONFIG", "")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "")

	cfg := &config.ModelConfig{
		ModelName: "bedrock-claude",
		Model:     "bedrock/us.anthropic.claude-sonnet-4-20250514-v1:0",
		APIBase:   "https://bedrock-runtime.us-east-1.amazonaws.com", // Full endpoint URL
	}

	provider, modelID, err := CreateProviderFromConfig(cfg)
	if err == nil {
		// Provider created successfully (built with -tags bedrock)
		if provider == nil {
			t.Error("provider is nil on success")
		}
		if modelID != "us.anthropic.claude-sonnet-4-20250514-v1:0" {
			t.Errorf("modelID = %q, want %q", modelID, "us.anthropic.claude-sonnet-4-20250514-v1:0")
		}
		return
	}
	errMsg := err.Error()
	// When built without -tags bedrock, expect stub error
	if strings.Contains(errMsg, "build with -tags bedrock") {
		return // Expected stub error
	}
	// Unexpected error - fail the test
	t.Errorf("unexpected error from bedrock provider: %v", err)
}

func TestCreateProviderFromConfig_ToolSchemaTransformWrapsProvider(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName:           "claude-cli-test",
		Provider:            "claude-cli",
		Model:               "claude-sonnet-4.6",
		Workspace:           t.TempDir(),
		ToolSchemaTransform: "simple",
	}

	provider, modelID, err := CreateProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateProviderFromConfig() error = %v", err)
	}
	if modelID != "claude-sonnet-4.6" {
		t.Fatalf("modelID = %q, want %q", modelID, "claude-sonnet-4.6")
	}
	if _, ok := provider.(*toolSchemaTransformProvider); !ok {
		t.Fatalf("provider = %T, want *toolSchemaTransformProvider", provider)
	}
}

func TestCreateProviderFromConfig_InvalidToolSchemaTransform(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName:           "claude-cli-test",
		Provider:            "claude-cli",
		Model:               "claude-sonnet-4.6",
		Workspace:           t.TempDir(),
		ToolSchemaTransform: "invalid",
	}

	_, _, err := CreateProviderFromConfig(cfg)
	if err == nil {
		t.Fatal("CreateProviderFromConfig() expected error for invalid tool_schema_transform")
	}
	if !strings.Contains(err.Error(), "tool_schema_transform") {
		t.Fatalf("error = %v, want mention tool_schema_transform", err)
	}
}
