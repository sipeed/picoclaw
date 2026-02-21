package agent

import (
	"testing"
	"time"
)

func TestTouch(t *testing.T) {
	st := NewSessionTracker()

	// Basic touch creates entry
	st.Touch("sess1", "telegram", "123", "projects/myapp")
	entries := st.ListActive()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].SessionKey != "sess1" {
		t.Errorf("expected session_key=sess1, got %s", entries[0].SessionKey)
	}
	if entries[0].Channel != "telegram" {
		t.Errorf("expected channel=telegram, got %s", entries[0].Channel)
	}
	if entries[0].TouchDir != "projects/myapp" {
		t.Errorf("expected touch_dir=projects/myapp, got %s", entries[0].TouchDir)
	}

	// Touch again with new dir overwrites TouchDir
	st.Touch("sess1", "", "", "projects/other")
	entries = st.ListActive()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].TouchDir != "projects/other" {
		t.Errorf("expected touch_dir=projects/other, got %s", entries[0].TouchDir)
	}
	// Channel should remain from first touch
	if entries[0].Channel != "telegram" {
		t.Errorf("expected channel=telegram (unchanged), got %s", entries[0].Channel)
	}

	// Touch with empty dir does not overwrite TouchDir
	st.Touch("sess1", "", "", "")
	entries = st.ListActive()
	if entries[0].TouchDir != "projects/other" {
		t.Errorf("expected touch_dir unchanged, got %s", entries[0].TouchDir)
	}
}

func TestIsActiveInDir(t *testing.T) {
	st := NewSessionTracker()

	// Setup: sess1 touches "projects/myapp"
	st.Touch("sess1", "telegram", "123", "projects/myapp")

	// Same dir, excluding sess1 → false
	if st.IsActiveInDir("projects/myapp", "sess1") {
		t.Error("expected false when excluding the only active session")
	}

	// Same dir, excluding different key → true
	if !st.IsActiveInDir("projects/myapp", "heartbeat") {
		t.Error("expected true for exact dir match")
	}

	// Parent dir match: "projects" is prefix of "projects/myapp"
	if !st.IsActiveInDir("projects", "heartbeat") {
		t.Error("expected true for parent dir match")
	}

	// Child dir match: "projects/myapp/src" has prefix "projects/myapp"
	if !st.IsActiveInDir("projects/myapp/src", "heartbeat") {
		t.Error("expected true for child dir match")
	}

	// Unrelated dir → false
	if st.IsActiveInDir("other/stuff", "heartbeat") {
		t.Error("expected false for unrelated dir")
	}

	// Stale entry (manually set LastSeenAt to past)
	val, _ := st.entries.Load("sess1")
	entry := val.(*SessionEntry)
	entry.LastSeenAt = time.Now().Add(-sessionActivityTimeout - time.Minute)

	if st.IsActiveInDir("projects/myapp", "heartbeat") {
		t.Error("expected false for stale session")
	}
}

func TestListActive(t *testing.T) {
	st := NewSessionTracker()

	// Add two sessions
	st.Touch("sess1", "telegram", "123", "projects/a")
	time.Sleep(5 * time.Millisecond) // ensure different timestamps
	st.Touch("sess2", "discord", "456", "projects/b")

	entries := st.ListActive()
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// Most recent first
	if entries[0].SessionKey != "sess2" {
		t.Errorf("expected sess2 first (most recent), got %s", entries[0].SessionKey)
	}
	if entries[1].SessionKey != "sess1" {
		t.Errorf("expected sess1 second, got %s", entries[1].SessionKey)
	}

	// Make sess1 stale
	val, _ := st.entries.Load("sess1")
	entry := val.(*SessionEntry)
	entry.LastSeenAt = time.Now().Add(-sessionActivityTimeout - time.Minute)

	entries = st.ListActive()
	if len(entries) != 1 {
		t.Fatalf("expected 1 active entry after stale, got %d", len(entries))
	}
	if entries[0].SessionKey != "sess2" {
		t.Errorf("expected only sess2, got %s", entries[0].SessionKey)
	}
}
