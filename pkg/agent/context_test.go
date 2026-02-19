package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
)

// helper to create a temporary workspace for ContextBuilder
func withTempWorkspace(t *testing.T, fn func(string)) {
	t.Helper()
	dir := t.TempDir()
	// ensure required subdirs exist if any future logic reads them
	_ = os.MkdirAll(filepath.Join(dir, "skills"), 0755)
	fn(dir)
}

func TestBuildMessages_CoalescesConsecutiveUserMessages(t *testing.T) {
	withTempWorkspace(t, func(ws string) {
		cb := NewContextBuilder(ws)

		history := []providers.Message{
			{Role: "system", Content: "sys"},
			{Role: "assistant", Content: "hi"},
			{Role: "user", Content: "first"},
		}

		msgs := cb.BuildMessages(history, "", "second", nil, "cli", "chat1")
		if len(msgs) < 2 {
			t.Fatalf("expected at least 2 messages, got %d", len(msgs))
		}
		if msgs[0].Role != "system" {
			t.Fatalf("expected first message to be system, got %s", msgs[0].Role)
		}
		// ensure no consecutive user messages and last is a single coalesced user
		last := msgs[len(msgs)-1]
		if last.Role != "user" {
			t.Fatalf("expected last message role=user, got %s", last.Role)
		}
		if last.Content != "first\n\nsecond" {
			t.Fatalf("expected coalesced content 'first\\n\\nsecond', got %q", last.Content)
		}
	})
}

func TestBuildMessages_AppendsUserIfNoConsecutive(t *testing.T) {
	withTempWorkspace(t, func(ws string) {
		cb := NewContextBuilder(ws)
		history := []providers.Message{
			{Role: "assistant", Content: "hi"},
		}
		msgs := cb.BuildMessages(history, "", "second", nil, "cli", "chat1")
		if len(msgs) < 3 {
			t.Fatalf("expected at least 3 messages (system + history + user), got %d", len(msgs))
		}
		if msgs[len(msgs)-1].Role != "user" {
			t.Fatalf("expected last message role=user, got %s", msgs[len(msgs)-1].Role)
		}
		if msgs[len(msgs)-2].Role == "user" {
			t.Fatalf("did not expect consecutive user messages")
		}
	})
}