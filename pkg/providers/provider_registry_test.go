package providers

import (
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestCreateProvider_ZenExplicit(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Provider = "zen"
	cfg.Agents.Defaults.Model = "claude-sonnet-4.5"
	cfg.Providers.Zen.APIKey = "zen-key"

	provider, err := CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider(zen) error = %v", err)
	}

	httpProvider, ok := provider.(*HTTPProvider)
	if !ok {
		t.Fatalf("CreateProvider(zen) returned %T, want *HTTPProvider", provider)
	}
	if httpProvider.apiBase != "https://opencode.ai/zen/v1" {
		t.Errorf("apiBase = %q, want %q", httpProvider.apiBase, "https://opencode.ai/zen/v1")
	}
	if httpProvider.apiKey != "zen-key" {
		t.Errorf("apiKey = %q, want %q", httpProvider.apiKey, "zen-key")
	}
}

func TestCreateProvider_ZenByModelPrefix(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Model = "zen/kimi-k2.5-free"
	cfg.Providers.Zen.APIKey = "zen-key"

	provider, err := CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider(zen/*) error = %v", err)
	}

	httpProvider, ok := provider.(*HTTPProvider)
	if !ok {
		t.Fatalf("CreateProvider(zen/*) returned %T, want *HTTPProvider", provider)
	}
	if httpProvider.apiBase != "https://opencode.ai/zen/v1" {
		t.Errorf("apiBase = %q, want %q", httpProvider.apiBase, "https://opencode.ai/zen/v1")
	}
}

func TestCreateProvider_OpencodePrefixDoesNotAutoSelectZen(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Model = "opencode/kimi-k2.5-free"
	cfg.Providers.Zen.APIKey = "zen-key"

	_, err := CreateProvider(cfg)
	if err == nil {
		t.Fatal("CreateProvider(opencode/*) expected error, got nil")
	}
}

func TestCreateProvider_ExplicitProviderNotConfiguredFails(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Provider = "zen"
	cfg.Agents.Defaults.Model = "kimi-k2.5-free"

	_, err := CreateProvider(cfg)
	if err == nil {
		t.Fatal("CreateProvider(zen explicit without api key) expected error, got nil")
	}
	if !strings.Contains(err.Error(), "provider 'zen' is not configured") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateProvider_ExplicitUnknownProviderFails(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Provider = "unknown-provider"
	cfg.Agents.Defaults.Model = "gpt-4o"
	cfg.Providers.OpenRouter.APIKey = "or-key"

	_, err := CreateProvider(cfg)
	if err == nil {
		t.Fatal("CreateProvider(unknown provider) expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unknown provider: unknown-provider") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateProvider_MyGeminiProxyDoesNotAutoSelectGemini(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Model = "mygeminiproxy-v1"
	cfg.Providers.Gemini.APIKey = "gem-key"
	cfg.Providers.OpenRouter.APIKey = "or-key"

	provider, err := CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider(ambiguous gemini model) error = %v", err)
	}

	httpProvider, ok := provider.(*HTTPProvider)
	if !ok {
		t.Fatalf("CreateProvider(ambiguous gemini model) returned %T, want *HTTPProvider", provider)
	}
	if httpProvider.apiBase != "https://openrouter.ai/api/v1" {
		t.Errorf("apiBase = %q, want %q", httpProvider.apiBase, "https://openrouter.ai/api/v1")
	}
}

func TestCreateProvider_FallbackOpenRouter(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Model = "custom/non-standard-model"
	cfg.Providers.OpenRouter.APIKey = "or-key"

	provider, err := CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider(default fallback) error = %v", err)
	}

	httpProvider, ok := provider.(*HTTPProvider)
	if !ok {
		t.Fatalf("CreateProvider(default fallback) returned %T, want *HTTPProvider", provider)
	}
	if httpProvider.apiBase != "https://openrouter.ai/api/v1" {
		t.Errorf("apiBase = %q, want %q", httpProvider.apiBase, "https://openrouter.ai/api/v1")
	}
}

func TestCreateProvider_ModelPrefixAvoidsAmbiguousContains(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Model = "myclaudeproxy-v1"
	cfg.Providers.Anthropic.APIKey = "anthropic-key"
	cfg.Providers.OpenRouter.APIKey = "or-key"

	provider, err := CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider(ambiguous-model) error = %v", err)
	}

	httpProvider, ok := provider.(*HTTPProvider)
	if !ok {
		t.Fatalf("CreateProvider(ambiguous-model) returned %T, want *HTTPProvider", provider)
	}
	if httpProvider.apiBase != "https://openrouter.ai/api/v1" {
		t.Errorf("apiBase = %q, want %q", httpProvider.apiBase, "https://openrouter.ai/api/v1")
	}
}
