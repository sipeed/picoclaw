package providers

import (
	"context"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestNewGitHubCopilotProviderStdioIsLazy(t *testing.T) {
	provider, err := NewGitHubCopilotProvider("", "stdio", "gpt-4.1")
	if err != nil {
		t.Fatalf("NewGitHubCopilotProvider() error = %v", err)
	}

	if provider == nil {
		t.Fatal("NewGitHubCopilotProvider() returned nil")
	}
	if provider.connectMode != "stdio" {
		t.Fatalf("connectMode = %q, want stdio", provider.connectMode)
	}
	if provider.client != nil {
		t.Fatalf("client = %#v, want nil before first Chat", provider.client)
	}
	if provider.session != nil {
		t.Fatalf("session = %#v, want nil before first Chat", provider.session)
	}
}

func TestCreateProviderUsesStdioWithoutGrpcDefaultApiBase(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = "test-copilot"
	cfg.ModelList = []*config.ModelConfig{{
		ModelName:   "test-copilot",
		Model:       "github-copilot/gpt-4.1",
		ConnectMode: "stdio",
	}}

	provider, _, err := CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider() error = %v", err)
	}

	githubCopilotProvider, ok := provider.(*GitHubCopilotProvider)
	if !ok {
		t.Fatalf("provider type = %T, want *GitHubCopilotProvider", provider)
	}
	if githubCopilotProvider.uri != "" {
		t.Fatalf("uri = %q, want empty for stdio mode", githubCopilotProvider.uri)
	}
	if githubCopilotProvider.connectMode != "stdio" {
		t.Fatalf("connectMode = %q, want stdio", githubCopilotProvider.connectMode)
	}
}

func TestGitHubCopilotProviderClosePreventsReopen(t *testing.T) {
	provider, err := NewGitHubCopilotProvider("", "stdio", "gpt-4.1")
	if err != nil {
		t.Fatalf("NewGitHubCopilotProvider() error = %v", err)
	}

	provider.Close()

	_, err = provider.Chat(context.Background(), nil, nil, "gpt-4.1", nil)
	if err == nil {
		t.Fatal("Chat() error = nil, want provider closed")
	}
	if !strings.Contains(err.Error(), "provider closed") {
		t.Fatalf("Chat() error = %v, want provider closed", err)
	}
}
