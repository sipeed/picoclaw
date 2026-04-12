//go:build integration

package providers

import (
	"context"
	exec "os/exec"
	"strings"
	"testing"
	"time"
)

// TestIntegration_RealCodexCLI tests the CodexCliProvider with a real codex CLI.
// Run with: go test -tags=integration ./pkg/providers/...
func TestIntegration_RealCodexCLI(t *testing.T) {
	path, err := exec.LookPath("codex")
	if err != nil {
		t.Skip("codex CLI not found in PATH, skipping integration test")
	}
	t.Logf("Using codex CLI at: %s", path)

	p := NewCodexCliProvider(t.TempDir())

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	resp, err := p.Chat(ctx, []Message{
		{Role: "user", Content: "Respond with only the word 'pong'. Nothing else."},
	}, nil, "", nil)
	if err != nil {
		t.Fatalf("Chat() with real CLI error = %v", err)
	}

	if resp.Content == "" {
		t.Error("Content is empty")
	}
	if resp.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want %q", resp.FinishReason, "stop")
	}
	if resp.Usage != nil {
		t.Logf("Usage: prompt=%d, completion=%d, total=%d",
			resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
	}

	t.Logf("Response content: %q", resp.Content)

	if !strings.Contains(strings.ToLower(resp.Content), "pong") {
		t.Errorf("Content = %q, expected to contain 'pong'", resp.Content)
	}
}

func TestIntegration_RealCodexCLI_WithSystemPrompt(t *testing.T) {
	if _, err := exec.LookPath("codex"); err != nil {
		t.Skip("codex CLI not found in PATH")
	}

	p := NewCodexCliProvider(t.TempDir())

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
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

func TestIntegration_RealCodexCLI_ParsesRealJSONL(t *testing.T) {
	if _, err := exec.LookPath("codex"); err != nil {
		t.Skip("codex CLI not found in PATH")
	}

	// Run codex directly and verify our parser handles real output
	cmd := exec.Command("codex", "exec",
		"--json",
		"--dangerously-bypass-approvals-and-sandbox",
		"--skip-git-repo-check",
		"--color", "never",
		"-C", t.TempDir(),
		"-")
	cmd.Stdin = strings.NewReader("Say hi")

	output, err := cmd.Output()
	if err != nil {
		// codex may write diagnostic noise to stderr but still produce valid output
		if len(output) == 0 {
			t.Fatalf("codex CLI failed: %v", err)
		}
	}

	t.Logf("Raw CLI output (first 500 chars): %s", string(output[:min(len(output), 500)]))

	// Verify our parser can handle real output
	p := NewCodexCliProvider("")
	resp, err := p.parseJSONLEvents(string(output))
	if err != nil {
		t.Fatalf("parseJSONLEvents() failed on real CLI output: %v", err)
	}

	if resp.Content == "" {
		t.Error("parsed Content is empty")
	}
	if resp.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want stop", resp.FinishReason)
	}

	t.Logf("Parsed: content=%q, finish=%s, usage=%+v", resp.Content, resp.FinishReason, resp.Usage)
}
