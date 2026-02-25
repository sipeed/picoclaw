package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"
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

func TestGetSessionSnapshot_IsDeepCopy(t *testing.T) {
	sm := NewSessionManager(t.TempDir())
	key := "agent:main:main"
	sm.AddMessage(key, "user", "hello")

	snapshot, ok := sm.GetSessionSnapshot(key)
	if !ok {
		t.Fatalf("expected snapshot for key %q", key)
	}
	if len(snapshot.Messages) != 1 {
		t.Fatalf("expected 1 message in snapshot, got %d", len(snapshot.Messages))
	}

	// Mutate returned snapshot; internal state should remain unchanged.
	snapshot.Messages[0].Content = "mutated"
	history := sm.GetHistory(key)
	if history[0].Content != "hello" {
		t.Fatalf("snapshot mutation leaked into manager state, got %q", history[0].Content)
	}
}

func TestListSessionSnapshots_SortedByUpdatedDesc(t *testing.T) {
	sm := NewSessionManager(t.TempDir())
	sm.AddMessage("session-a", "user", "old")
	time.Sleep(10 * time.Millisecond)
	sm.AddMessage("session-b", "user", "new")

	snapshots := sm.ListSessionSnapshots()
	if len(snapshots) != 2 {
		t.Fatalf("expected 2 snapshots, got %d", len(snapshots))
	}
	if snapshots[0].Key != "session-b" {
		t.Fatalf("expected newest session first, got %q", snapshots[0].Key)
	}

	// Ensure returned slices are copies.
	snapshots[0].Messages[0].Content = "changed"
	history := sm.GetHistory("session-b")
	if history[0].Content != "new" {
		t.Fatalf("snapshot mutation leaked into manager state, got %q", history[0].Content)
	}
}

func TestCompactionStateLifecycle(t *testing.T) {
	sm := NewSessionManager(t.TempDir())
	key := "agent:main:main"
	sm.AddMessage(key, "user", "hello")

	count, flushedCount, flushAt := sm.GetCompactionState(key)
	if count != 0 || flushedCount != 0 || !flushAt.IsZero() {
		t.Fatalf("initial compaction state unexpected: count=%d flushed=%d flushAt=%v", count, flushedCount, flushAt)
	}

	next := sm.IncrementCompactionCount(key)
	if next != 1 {
		t.Fatalf("IncrementCompactionCount = %d, want 1", next)
	}

	sm.MarkMemoryFlush(key, next)
	count, flushedCount, flushAt = sm.GetCompactionState(key)
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
	if flushedCount != 1 {
		t.Fatalf("flushedCount = %d, want 1", flushedCount)
	}
	if flushAt.IsZero() {
		t.Fatal("flushAt should be set")
	}
}
