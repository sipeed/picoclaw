package providers

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/auth"
	"github.com/sipeed/picoclaw/pkg/config"
)

func TestCreateProviderReturnsHTTPProviderForOpenRouter(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = "test-openrouter"
	modelCfg := &config.ModelConfig{
		ModelName: "test-openrouter",
		Model:     "openrouter/auto",
		APIBase:   "https://openrouter.ai/api/v1",
	}
	modelCfg.SetAPIKey("sk-or-test")
	cfg.ModelList = []*config.ModelConfig{modelCfg}

	provider, _, err := CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider() error = %v", err)
	}

	if _, ok := provider.(*HTTPProvider); !ok {
		t.Fatalf("provider type = %T, want *HTTPProvider", provider)
	}
}

func TestCreateProviderResolvesRawDefaultFromProviderTemplate(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = "openrouter/deepseek/deepseek-v3.2"
	modelCfg := &config.ModelConfig{
		ModelName:     "openrouter-template",
		Model:         "openrouter/auto",
		APIBase:       "https://openrouter.ai/api/v1",
		ThinkingLevel: "off",
	}
	modelCfg.SetAPIKey("sk-or-test")
	cfg.ModelList = []*config.ModelConfig{modelCfg}

	provider, modelID, err := CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider() error = %v", err)
	}
	if _, ok := provider.(*HTTPProvider); !ok {
		t.Fatalf("provider type = %T, want *HTTPProvider", provider)
	}
	if modelID != "deepseek/deepseek-v3.2" {
		t.Fatalf("model ID = %q, want %q", modelID, "deepseek/deepseek-v3.2")
	}
	if modelCfg.Model != "openrouter/auto" {
		t.Fatalf("template model mutated to %q", modelCfg.Model)
	}
	resolved, err := ResolveModelConfig(cfg, cfg.Agents.Defaults.ModelName)
	if err != nil {
		t.Fatalf("ResolveModelConfig() error = %v", err)
	}
	if resolved.ThinkingLevel != "" {
		t.Fatalf("raw model inherited template thinking level %q", resolved.ThinkingLevel)
	}
}

func TestResolveModelConfigPrefersConfiguredProviderTemplate(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ModelList = []*config.ModelConfig{
		{ModelName: "unconfigured", Model: "openrouter/auto"},
		{
			ModelName: "configured",
			Model:     "openrouter/other",
			APIBase:   "https://openrouter.ai/api/v1",
			APIKeys:   config.SimpleSecureStrings("sk-or-test"),
		},
	}

	resolved, err := ResolveModelConfig(cfg, "openrouter/new-model")
	if err != nil {
		t.Fatalf("ResolveModelConfig() error = %v", err)
	}
	if resolved.APIBase != "https://openrouter.ai/api/v1" || resolved.APIKey() != "sk-or-test" {
		t.Fatalf("resolved template = %#v, want configured OpenRouter entry", resolved)
	}
}

func TestResolveModelConfigPrefersConfiguredExactRawMatch(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ModelList = []*config.ModelConfig{
		{ModelName: "unconfigured", Provider: "openrouter", Model: "openrouter/target"},
		{
			ModelName: "configured",
			Provider:  "openrouter",
			Model:     "openrouter/target",
			APIKeys:   config.SimpleSecureStrings("sk-or-test"),
		},
	}

	resolved, err := ResolveModelConfig(cfg, "openrouter/target")
	if err != nil {
		t.Fatalf("ResolveModelConfig() error = %v", err)
	}
	if resolved.ModelName != "configured" {
		t.Fatalf("ModelName = %q, want configured", resolved.ModelName)
	}
}

func TestResolveModelConfigUsesDefaultProviderForBareTemplate(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Provider = "openrouter"
	cfg.ModelList = []*config.ModelConfig{{
		ModelName: "bare-template",
		Model:     "auto",
		APIKeys:   config.SimpleSecureStrings("sk-or-test"),
	}}

	resolved, err := ResolveModelConfig(cfg, "openrouter/new-model")
	if err != nil {
		t.Fatalf("ResolveModelConfig() error = %v", err)
	}
	if resolved.APIKey() != "sk-or-test" {
		t.Fatalf("APIKey() = %q, want provider template key", resolved.APIKey())
	}
}

func TestResolveModelConfigPreservesLegacyExactModelProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Provider = "openrouter"
	cfg.ModelList = []*config.ModelConfig{{
		ModelName: "legacy-template",
		Provider:  "openrouter",
		Model:     "openai/gpt-4o",
		APIKeys:   config.SimpleSecureStrings("sk-or-test"),
	}}

	resolved, err := ResolveModelConfig(cfg, "openai/gpt-4o")
	if err != nil {
		t.Fatalf("ResolveModelConfig() error = %v", err)
	}
	if resolved.Provider != "openrouter" {
		t.Fatalf("Provider = %q, want openrouter", resolved.Provider)
	}
}

func TestResolveModelConfigSetsDefaultProviderOnBareEntry(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Provider = "openrouter"
	cfg.ModelList = []*config.ModelConfig{{
		ModelName: "bare",
		Model:     "gpt-4o",
		APIKeys:   config.SimpleSecureStrings("sk-or-test"),
	}}

	resolved, err := ResolveModelConfig(cfg, "bare")
	if err != nil {
		t.Fatalf("ResolveModelConfig() error = %v", err)
	}
	if resolved.Provider != "openrouter" || resolved.Model != "gpt-4o" {
		t.Fatalf("resolved provider/model = %q/%q, want openrouter/gpt-4o", resolved.Provider, resolved.Model)
	}
}

func TestResolveModelConfigPrefersCustomAPIBaseTemplate(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ModelList = []*config.ModelConfig{
		{
			ModelName: "default-base",
			Provider:  "openai",
			Model:     "gpt-4o",
			APIBase:   DefaultAPIBaseForProtocol("openai"),
		},
		{
			ModelName: "custom-base",
			Provider:  "openai",
			Model:     "gpt-4.1",
			APIBase:   "https://proxy.example.test/v1",
		},
	}

	resolved, err := ResolveModelConfig(cfg, "openai/new-model")
	if err != nil {
		t.Fatalf("ResolveModelConfig() error = %v", err)
	}
	if resolved.APIBase != "https://proxy.example.test/v1" {
		t.Fatalf("APIBase = %q, want custom provider template", resolved.APIBase)
	}
}

func TestCreateProviderReturnsCodexCliProviderForCodexCode(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = "test-codex"
	cfg.ModelList = []*config.ModelConfig{
		{
			ModelName: "test-codex",
			Model:     "codex-cli/codex-model",
			Workspace: "/tmp/workspace",
		},
	}

	provider, _, err := CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider() error = %v", err)
	}

	if _, ok := provider.(*CodexCliProvider); !ok {
		t.Fatalf("provider type = %T, want *CodexCliProvider", provider)
	}
}

func TestCreateProviderReturnsClaudeCliProviderForClaudeCli(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = "test-claude-cli"
	cfg.ModelList = []*config.ModelConfig{
		{
			ModelName: "test-claude-cli",
			Model:     "claude-cli/claude-sonnet",
			Workspace: "/tmp/workspace",
		},
	}

	provider, _, err := CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider() error = %v", err)
	}

	if _, ok := provider.(*ClaudeCliProvider); !ok {
		t.Fatalf("provider type = %T, want *ClaudeCliProvider", provider)
	}
}

func TestCreateProviderReturnsClaudeProviderForAnthropicOAuth(t *testing.T) {
	originalGetCredential := getCredential
	t.Cleanup(func() { getCredential = originalGetCredential })

	getCredential = func(provider string) (*auth.AuthCredential, error) {
		if provider != "anthropic" {
			t.Fatalf("provider = %q, want anthropic", provider)
		}
		return &auth.AuthCredential{
			AccessToken: "anthropic-token",
		}, nil
	}

	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = "test-claude-oauth"
	cfg.ModelList = []*config.ModelConfig{
		{
			ModelName:  "test-claude-oauth",
			Model:      "anthropic/claude-sonnet-4.6",
			AuthMethod: "oauth",
		},
	}

	provider, _, err := CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider() error = %v", err)
	}

	if _, ok := provider.(*ClaudeProvider); !ok {
		t.Fatalf("provider type = %T, want *ClaudeProvider", provider)
	}
	// TODO: Test custom APIBase when createClaudeAuthProvider supports it
}

func TestCreateProviderReturnsCodexProviderForOpenAIOAuth(t *testing.T) {
	// TODO: This test requires openai protocol to support auth_method: "oauth"
	// which is not yet implemented in the new factory_provider.go
	t.Skip("OpenAI OAuth via model_list not yet implemented")
}
