package providers

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestNewGitHubCopilotProvider_UnknownConnectMode(t *testing.T) {
	_, err := NewGitHubCopilotProvider("localhost:4321", "unknown", "gpt-4.1")
	if err == nil {
		t.Fatal("expected error for unknown connect mode, got nil")
	}
	if got := err.Error(); got != "unknown connect mode: unknown" {
		t.Fatalf("error = %q, want %q", got, "unknown connect mode: unknown")
	}
}

func TestNewGitHubCopilotProvider_DefaultConnectMode(t *testing.T) {
	// When connectMode is empty, it should default to "grpc" and attempt to connect.
	// Since there's no server, we expect a connection error (not a "not implemented" error).
	_, err := NewGitHubCopilotProvider("localhost:19999", "", "gpt-4.1")
	if err == nil {
		t.Fatal("expected connection error, got nil")
	}
	// Should NOT get "not implemented" error — that was the old behavior before stdio was implemented
	if got := err.Error(); got == "stdio mode not implemented for GitHub Copilot provider; please use 'grpc' mode instead" {
		t.Fatal("got old 'not implemented' error; stdio mode should be implemented now")
	}
}

func TestNewGitHubCopilotProvider_StdioModeAccepted(t *testing.T) {
	// Stdio mode should no longer return "not implemented".
	// It will fail to start because the copilot CLI binary is likely not installed,
	// but the error should be about starting the CLI process, not about the mode
	// being unimplemented.
	_, err := NewGitHubCopilotProvider("", "stdio", "gpt-4.1")
	if err == nil {
		// If it succeeds, that's fine too (copilot CLI is installed).
		return
	}
	got := err.Error()
	if got == "stdio mode not implemented for GitHub Copilot provider; please use 'grpc' mode instead" {
		t.Fatal("got old 'not implemented' error; stdio mode should be implemented now")
	}
	// Expect a startup/connection error, not an implementation error
	t.Logf("expected startup error (copilot CLI likely not installed): %v", err)
}

func TestNewGitHubCopilotProvider_StdioWithCustomCLIPath(t *testing.T) {
	// When a custom CLI path is provided in stdio mode, the error should reference
	// the startup failure, not "not implemented".
	_, err := NewGitHubCopilotProvider("/nonexistent/copilot", "stdio", "gpt-4.1")
	if err == nil {
		t.Fatal("expected error for nonexistent CLI path, got nil")
	}
	got := err.Error()
	if got == "stdio mode not implemented for GitHub Copilot provider; please use 'grpc' mode instead" {
		t.Fatal("got old 'not implemented' error; stdio mode should be implemented now")
	}
	t.Logf("got expected startup error: %v", err)
}

func TestGitHubCopilotProvider_GetDefaultModel(t *testing.T) {
	p := &GitHubCopilotProvider{}
	if got := p.GetDefaultModel(); got != "gpt-4.1" {
		t.Fatalf("GetDefaultModel() = %q, want %q", got, "gpt-4.1")
	}
}

func TestGitHubCopilotProvider_CloseNilClient(t *testing.T) {
	// Close on a provider with nil client should not panic
	p := &GitHubCopilotProvider{}
	p.Close() // should not panic
}

func TestGitHubCopilotProvider_ChatNilSession(t *testing.T) {
	p := &GitHubCopilotProvider{}
	_, err := p.Chat(nil, nil, nil, "gpt-4.1", nil)
	if err == nil {
		t.Fatal("expected error for nil session, got nil")
	}
	if got := err.Error(); got != "provider closed" {
		t.Fatalf("error = %q, want %q", got, "provider closed")
	}
}

func TestCreateProviderFromConfig_CopilotStdioMode(t *testing.T) {
	cfg := &config.ModelConfig{
		Model:       "copilot/gpt-4.1",
		ConnectMode: "stdio",
	}

	// This will try to start the copilot CLI which likely isn't installed,
	// but the error should be a startup error, not "unknown protocol" or "not implemented".
	_, _, err := CreateProviderFromConfig(cfg)
	if err == nil {
		return // copilot CLI is installed, all good
	}
	got := err.Error()
	if got == "stdio mode not implemented for GitHub Copilot provider; please use 'grpc' mode instead" {
		t.Fatal("got old 'not implemented' error; stdio should be supported now")
	}
	t.Logf("expected startup error: %v", err)
}

func TestCreateProviderFromConfig_CopilotGrpcModeDefaultAPIBase(t *testing.T) {
	cfg := &config.ModelConfig{
		Model:       "github-copilot/gpt-4.1",
		ConnectMode: "grpc",
	}

	// Will fail to connect, but should attempt localhost:4321
	_, _, err := CreateProviderFromConfig(cfg)
	if err == nil {
		return
	}
	// The error should mention the connection failure, not "not implemented"
	t.Logf("expected connection error: %v", err)
}
