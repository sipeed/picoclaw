package session

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"telegram:123456", "telegram_123456"},
		{"discord:987654321", "discord_987654321"},
		{"slack:C01234", "slack_C01234"},
		{"no-colons-here", "no-colons-here"},
		{"multiple:colons:here", "multiple_colons_here"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeFilename(tt.input)
			if got != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSave_WithColonInKey(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSessionManager(tmpDir)

	// Create a session with a key containing colon (typical channel session key).
	key := "telegram:123456"
	sm.GetOrCreate(key)
	sm.AddMessage(key, "user", "hello")

	// Save should succeed even though the key contains ':'
	if err := sm.Save(key); err != nil {
		t.Fatalf("Save(%q) failed: %v", key, err)
	}

	// The file on disk should use sanitized name.
	expectedFile := filepath.Join(tmpDir, "telegram_123456.json")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Fatalf("expected session file %s to exist", expectedFile)
	}

	// Load into a fresh manager and verify the session round-trips.
	sm2 := NewSessionManager(tmpDir)
	history := sm2.GetHistory(key)
	if len(history) != 1 {
		t.Fatalf("expected 1 message after reload, got %d", len(history))
	}
	if history[0].Content != "hello" {
		t.Errorf("expected message content %q, got %q", "hello", history[0].Content)
	}
}

func TestSave_RejectsPathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSessionManager(tmpDir)

	badKeys := []string{"", ".", "..", "foo/bar", "foo\\bar"}
	for _, key := range badKeys {
		sm.GetOrCreate(key)
		if err := sm.Save(key); err == nil {
			t.Errorf("Save(%q) should have failed but didn't", key)
		}
	}
}

func TestTruncateHistory_StartsAtUserBoundary(t *testing.T) {
	sm := NewSessionManager("")
	key := "test-session"

	// Build a session with: user, assistant(tc), tool, user, assistant
	sm.AddMessage(key, "user", "first")
	sm.AddFullMessage(key, providers.Message{
		Role: "assistant",
		ToolCalls: []providers.ToolCall{
			{ID: "call_1", Type: "function", Function: &providers.FunctionCall{Name: "test"}},
		},
	})
	sm.AddFullMessage(key, providers.Message{
		Role:       "tool",
		Content:    "result",
		ToolCallID: "call_1",
	})
	sm.AddMessage(key, "user", "second")
	sm.AddMessage(key, "assistant", "response")

	// keepLast=3 would normally start at index 2 (tool message)
	// Smart truncation should advance to index 3 (user message)
	sm.TruncateHistory(key, 3)

	history := sm.GetHistory(key)
	if len(history) != 2 {
		t.Fatalf("expected 2 messages after truncation, got %d: %+v", len(history), history)
	}
	if history[0].Role != "user" {
		t.Errorf("expected first message to be user, got %s", history[0].Role)
	}
	if history[0].Content != "second" {
		t.Errorf("expected first message content 'second', got %q", history[0].Content)
	}
}

func TestTruncateHistory_AlreadyAtUserBoundary(t *testing.T) {
	sm := NewSessionManager("")
	key := "test-session"

	sm.AddMessage(key, "user", "hello")
	sm.AddMessage(key, "assistant", "hi")
	sm.AddMessage(key, "user", "bye")
	sm.AddMessage(key, "assistant", "goodbye")

	sm.TruncateHistory(key, 2)

	history := sm.GetHistory(key)
	if len(history) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(history))
	}
	if history[0].Role != "user" {
		t.Errorf("expected first message to be user, got %s", history[0].Role)
	}
}

func TestTruncateHistory_NoUserMessage(t *testing.T) {
	sm := NewSessionManager("")
	key := "test-session"

	// Only non-user messages
	sm.AddMessage(key, "assistant", "hello")
	sm.AddFullMessage(key, providers.Message{
		Role: "assistant",
		ToolCalls: []providers.ToolCall{
			{ID: "call_1", Type: "function", Function: &providers.FunctionCall{Name: "test"}},
		},
	})
	sm.AddFullMessage(key, providers.Message{
		Role:       "tool",
		Content:    "result",
		ToolCallID: "call_1",
	})

	original := sm.GetHistory(key)
	origLen := len(original)

	// Should keep all messages since no user boundary exists
	sm.TruncateHistory(key, 1)

	history := sm.GetHistory(key)
	if len(history) != origLen {
		t.Fatalf("expected %d messages (unchanged), got %d", origLen, len(history))
	}
}
