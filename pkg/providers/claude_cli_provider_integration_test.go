//go:build integration

package providers

import (
	"context"
	exec "os/exec"
	"strings"
	"testing"
	"time"
)

// TestIntegration_RealClaudeCLI tests the ClaudeCliProvider with a real claude CLI.
// Run with: go test -tags=integration ./pkg/providers/...
func TestIntegration_RealClaudeCLI(t *testing.T) {
	// Check if claude CLI is available
	path, err := exec.LookPath("claude")
	if err != nil {
		t.Skip("claude CLI not found in PATH, skipping integration test")
	}
	t.Logf("Using claude CLI at: %s", path)

	p := NewClaudeCliProvider(t.TempDir())

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := p.Chat(ctx, []Message{
		{Role: "user", Content: "Respond with only the word 'pong'. Nothing else."},
	}, nil, "", nil)

	if err != nil {
		t.Fatalf("Chat() with real CLI error = %v", err)
	}

	// Verify response structure
	if resp.Content == "" {
		t.Error("Content is empty")
	}
	if resp.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want %q", resp.FinishReason, "stop")
	}
	if resp.Usage == nil {
		t.Error("Usage should not be nil from real CLI")
	} else {
		if resp.Usage.PromptTokens == 0 {
			t.Error("PromptTokens should be > 0")
		}
		if resp.Usage.CompletionTokens == 0 {
			t.Error("CompletionTokens should be > 0")
		}
		t.Logf("Usage: prompt=%d, completion=%d, total=%d",
			resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
	}

	t.Logf("Response content: %q", resp.Content)

	// Loose check - should contain "pong" somewhere (model might capitalize or add punctuation)
	if !strings.Contains(strings.ToLower(resp.Content), "pong") {
		t.Errorf("Content = %q, expected to contain 'pong'", resp.Content)
	}
}

func TestIntegration_RealClaudeCLI_WithSystemPrompt(t *testing.T) {
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude CLI not found in PATH")
	}

	p := NewClaudeCliProvider(t.TempDir())

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := p.Chat(ctx, []Message{
		{Role: "system", Content: "You are a calculator. Only respond with numbers. No text."},
		{Role: "user", Content: "What is 2+2?"},
	}, nil, "", nil)

	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	t.Logf("Response: %q", resp.Content)

	if !strings.Contains(resp.Content, "4") {
		t.Errorf("Content = %q, expected to contain '4'", resp.Content)
	}
}

func TestIntegration_RealClaudeCLI_ParsesRealJSON(t *testing.T) {
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude CLI not found in PATH")
	}

	// Run claude directly and verify our parser handles real output
	cmd := exec.Command("claude", "-p", "--output-format", "json",
		"--dangerously-skip-permissions", "--no-chrome", "--no-session-persistence", "-")
	cmd.Stdin = strings.NewReader("Say hi")
	cmd.Dir = t.TempDir()

	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("claude CLI failed: %v", err)
	}

	t.Logf("Raw CLI output: %s", string(output))

	// Verify our parser can handle real output
	p := NewClaudeCliProvider("")
	resp, err := p.parseClaudeCliResponse(string(output))
	if err != nil {
		t.Fatalf("parseClaudeCliResponse() failed on real CLI output: %v", err)
	}

	if resp.Content == "" {
		t.Error("parsed Content is empty")
	}
	if resp.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want stop", resp.FinishReason)
	}
	if resp.Usage == nil {
		t.Error("Usage should not be nil")
	}

	t.Logf("Parsed: content=%q, finish=%s, usage=%+v", resp.Content, resp.FinishReason, resp.Usage)
}
