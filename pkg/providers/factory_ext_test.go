package providers

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/auth"
	"github.com/sipeed/picoclaw/pkg/config"
)

func TestCreateProviderByName_OpenAI_OAuth(t *testing.T) {
	originalGetCredential := getCredential
	t.Cleanup(func() { getCredential = originalGetCredential })

	getCredential = func(provider string) (*auth.AuthCredential, error) {
		if provider != "openai" {
			t.Fatalf("provider = %q, want openai", provider)
		}
		return &auth.AuthCredential{
			AccessToken: "openai-token",
			AccountID:   "acct_test",
		}, nil
	}

	cfg := config.DefaultConfig()
	cfg.ModelList = []*config.ModelConfig{
		{
			ModelName:  "openai-oauth",
			Model:      "openai/gpt-4o",
			AuthMethod: "oauth",
		},
	}

	provider, err := CreateProviderByName(cfg, "openai-oauth")
	if err != nil {
		t.Fatalf("CreateProviderByName() error = %v", err)
	}

	if _, ok := provider.(*CodexProvider); !ok {
		t.Fatalf("provider type = %T, want *CodexProvider", provider)
	}
}

func TestCreateProviderByName_VLLM(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ModelList = append(cfg.ModelList, &config.ModelConfig{
		ModelName: "vllm",
		Model:     "vllm/test-model",
		APIBase:   "https://api.example.com/v1",
	})

	provider, err := CreateProviderByName(cfg, "vllm")
	if err != nil {
		t.Fatalf("CreateProviderByName() error = %v", err)
	}

	if _, ok := provider.(*HTTPProvider); !ok {
		t.Fatalf("provider type = %T, want *HTTPProvider", provider)
	}
}

func TestCreateProviderByName_Unknown(t *testing.T) {
	cfg := config.DefaultConfig()

	_, err := CreateProviderByName(cfg, "nonexistent-provider")
	if err == nil {
		t.Fatal("expected error for unknown provider, got nil")
	}
}

func TestCreateProviderByName_CaseInsensitive(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ModelList = append(cfg.ModelList, &config.ModelConfig{
		ModelName: "vllm",
		Model:     "vllm/test-model",
		APIBase:   "https://example.com/v1",
	})

	provider, err := CreateProviderByName(cfg, "VLLM")
	if err != nil {
		t.Fatalf("CreateProviderByName() error = %v", err)
	}

	if _, ok := provider.(*HTTPProvider); !ok {
		t.Fatalf("provider type = %T, want *HTTPProvider", provider)
	}
}
