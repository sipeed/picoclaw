package session

import (
	"os"
	"path/filepath"
	"testing"
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

func TestDelete_RemovesSessionFromMemoryAndDisk(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSessionManager(tmpDir)

	key := "agent:main:test"
	sm.GetOrCreate(key)
	sm.AddMessage(key, "user", "hello")

	if err := sm.Save(key); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	expectedFile := filepath.Join(tmpDir, "agent_main_test.json")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Fatalf("expected session file to exist before delete")
	}

	if err := sm.Delete(key); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify removed from memory
	history := sm.GetHistory(key)
	if len(history) != 0 {
		t.Errorf("expected empty history after delete, got %d messages", len(history))
	}

	// Verify removed from disk
	if _, err := os.Stat(expectedFile); !os.IsNotExist(err) {
		t.Errorf("expected session file to be deleted from disk")
	}
}

func TestDelete_NonexistentKeyNoError(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSessionManager(tmpDir)

	// Deleting a key that doesn't exist should not error
	if err := sm.Delete("nonexistent:key"); err != nil {
		t.Errorf("Delete of nonexistent key should not error, got: %v", err)
	}
}

func TestClearSession_ResetsHistoryAndSummary(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSessionManager(tmpDir)

	key := "agent:main:clear-test"
	sm.GetOrCreate(key)
	sm.AddMessage(key, "user", "hello")
	sm.AddMessage(key, "assistant", "hi there")
	sm.SetSummary(key, "User said hello")

	// Verify pre-conditions
	history := sm.GetHistory(key)
	if len(history) != 2 {
		t.Fatalf("expected 2 messages before clear, got %d", len(history))
	}
	if sm.GetSummary(key) == "" {
		t.Fatalf("expected non-empty summary before clear")
	}

	sm.ClearSession(key)

	// Verify history is cleared
	history = sm.GetHistory(key)
	if len(history) != 0 {
		t.Errorf("expected 0 messages after clear, got %d", len(history))
	}

	// Verify summary is cleared
	if sm.GetSummary(key) != "" {
		t.Errorf("expected empty summary after clear, got %q", sm.GetSummary(key))
	}
}

func TestClearSession_NonexistentKeyNoPanic(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSessionManager(tmpDir)

	// Clearing a nonexistent session should not panic
	sm.ClearSession("nonexistent:key")
}
