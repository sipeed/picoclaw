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

func TestSanitizeHistory_OrphanedToolCall(t *testing.T) {
	history := []providers.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "sure", ToolCalls: []providers.ToolCall{
			{ID: "call_1", Name: "exec"},
			{ID: "call_2", Name: "list_dir"},
		}},
		{Role: "tool", Content: "ok", ToolCallID: "call_1"},
		// Missing tool result for call_2 → orphaned
	}

	sanitized, removed := SanitizeHistory(history)
	// The orphaned assistant msg (with call_2 missing) and the trailing tool result
	// should both be removed, leaving just the user message
	if removed == 0 {
		t.Fatal("expected orphaned messages to be removed")
	}
	// After sanitization, only the user message should remain
	if len(sanitized) != 1 || sanitized[0].Role != "user" {
		t.Errorf("expected [user], got %d messages: %v", len(sanitized), sanitized)
	}
}

func TestSanitizeHistory_CleanHistory(t *testing.T) {
	history := []providers.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "sure", ToolCalls: []providers.ToolCall{
			{ID: "call_1", Name: "exec"},
		}},
		{Role: "tool", Content: "ok", ToolCallID: "call_1"},
		{Role: "assistant", Content: "done"},
	}

	sanitized, removed := SanitizeHistory(history)
	if removed != 0 {
		t.Errorf("expected 0 removed, got %d", removed)
	}
	if len(sanitized) != 4 {
		t.Errorf("expected 4 messages, got %d", len(sanitized))
	}
}

func TestSanitizeHistory_Empty(t *testing.T) {
	sanitized, removed := SanitizeHistory(nil)
	if removed != 0 || sanitized != nil {
		t.Errorf("expected nil/0, got %v/%d", sanitized, removed)
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
