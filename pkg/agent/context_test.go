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
			{Role: "assistant", Content: "hi"},
			{Role: "user", Content: "first"},
		}

		msgs := cb.BuildMessages(history, "", "second", nil, "cli", "chat1")
		if len(msgs) < 2 {
			t.Fatalf("expected at least 2 messages, got %d", len(msgs))
		}
		// BuildMessages should inject a single fresh system message at the start
		if msgs[0].Role != "system" {
			t.Fatalf("expected first message to be system, got %s", msgs[0].Role)
		}
		for i := 1; i < len(msgs); i++ {
			if msgs[i].Role == "system" {
				t.Fatalf("unexpected system message at index %d; history should not include system messages", i)
			}
		}
		// ensure no two consecutive messages are from user anywhere in the list
		for i := 1; i < len(msgs); i++ {
			if msgs[i-1].Role == "user" && msgs[i].Role == "user" {
				t.Fatalf("found consecutive user messages at indices %d and %d", i-1, i)
			}
		}
		// and the last is a single coalesced user message
		last := msgs[len(msgs)-1]
		if last.Role != "user" {
			t.Fatalf("expected last message role=user, got %s", last.Role)
		}
		if got, want := last.Content, "first\n\nsecond"; got != want {
			t.Fatalf("expected coalesced content %q, got %q", want, got)
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

func TestBuildMessages_CoalescesWithEmptyLastUserMessage(t *testing.T) {
	withTempWorkspace(t, func(ws string) {
		cb := NewContextBuilder(ws)
		history := []providers.Message{
			{Role: "assistant", Content: "hi"},
			{Role: "user", Content: ""},
		}
		msgs := cb.BuildMessages(history, "", "second", nil, "cli", "chat1")
		if len(msgs) < 2 {
			t.Fatalf("expected at least 2 messages, got %d", len(msgs))
		}
		// Only one injected system message should be at index 0
		if msgs[0].Role != "system" {
			t.Fatalf("expected first message to be system, got %s", msgs[0].Role)
		}
		for i := 1; i < len(msgs); i++ {
			if msgs[i].Role == "system" {
				t.Fatalf("unexpected system message at index %d; history should not include system messages", i)
			}
		}
		// ensure no consecutive user messages anywhere
		for i := 1; i < len(msgs); i++ {
			if msgs[i-1].Role == "user" && msgs[i].Role == "user" {
				t.Fatalf("found consecutive user messages at indices %d and %d", i-1, i)
			}
		}
		last := msgs[len(msgs)-1]
		if last.Role != "user" {
			t.Fatalf("expected last message role=user, got %s", last.Role)
		}
		if last.Content != "second" {
			t.Fatalf("expected coalesced content 'second' when last history content is empty, got %q", last.Content)
		}
	})
}
