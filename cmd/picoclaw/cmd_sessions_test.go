package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestListSessionEntries_NoDirectory(t *testing.T) {
	entries, err := listSessionEntries("/nonexistent/path/sessions")
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
	if entries != nil {
		t.Fatalf("expected nil entries, got %d", len(entries))
	}
}

func TestListSessionEntries_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	entries, err := listSessionEntries(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}
}

func TestListSessionEntries_ValidSessions(t *testing.T) {
	tmpDir := t.TempDir()

	// Create two valid session files
	writeTestSession(t, tmpDir, "telegram_123456.json", sessionData{
		Key:      "telegram:123456",
		Messages: json.RawMessage(`[{"role":"user","content":"hello"},{"role":"assistant","content":"hi"}]`),
		Created:  time.Now().Add(-1 * time.Hour),
		Updated:  time.Now(),
	})

	writeTestSession(t, tmpDir, "discord_789.json", sessionData{
		Key:      "discord:789",
		Messages: json.RawMessage(`[{"role":"user","content":"test"}]`),
		Created:  time.Now().Add(-2 * time.Hour),
		Updated:  time.Now().Add(-30 * time.Minute),
	})

	entries, err := listSessionEntries(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// Find the telegram entry
	var telegramEntry *sessionEntry
	for i := range entries {
		if entries[i].id == "telegram:123456" {
			telegramEntry = &entries[i]
			break
		}
	}
	if telegramEntry == nil {
		t.Fatal("telegram:123456 entry not found")
	}
	if telegramEntry.messages != 2 {
		t.Errorf("expected 2 messages, got %d", telegramEntry.messages)
	}
	if telegramEntry.corrupt {
		t.Error("expected non-corrupt session")
	}
}

func TestListSessionEntries_CorruptSession(t *testing.T) {
	tmpDir := t.TempDir()

	// Write a corrupt JSON file
	err := os.WriteFile(filepath.Join(tmpDir, "bad_session.json"), []byte("{invalid json"), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	entries, err := listSessionEntries(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if !entries[0].corrupt {
		t.Error("expected corrupt flag to be set")
	}
	if entries[0].id != "bad_session" {
		t.Errorf("expected id 'bad_session', got %q", entries[0].id)
	}
}

func TestListSessionEntries_SkipsNonJSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Write a non-JSON file
	err := os.WriteFile(filepath.Join(tmpDir, "notes.txt"), []byte("not a session"), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Write a subdirectory
	err = os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)
	if err != nil {
		t.Fatalf("failed to create test dir: %v", err)
	}

	entries, err := listSessionEntries(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}
}

func TestFindSessionFile_ByKey(t *testing.T) {
	tmpDir := t.TempDir()

	writeTestSession(t, tmpDir, "telegram_123456.json", sessionData{
		Key:      "telegram:123456",
		Messages: json.RawMessage(`[]`),
	})

	path := findSessionFile(tmpDir, "telegram:123456")
	if path == "" {
		t.Fatal("expected to find session file by key")
	}
	expected := filepath.Join(tmpDir, "telegram_123456.json")
	if path != expected {
		t.Errorf("expected path %q, got %q", expected, path)
	}
}

func TestFindSessionFile_BySanitizedName(t *testing.T) {
	tmpDir := t.TempDir()

	// Write a session file where the key doesn't match the ID we're looking for,
	// but the sanitized filename does
	writeTestSession(t, tmpDir, "cli_default.json", sessionData{
		Key:      "cli:default",
		Messages: json.RawMessage(`[]`),
	})

	// Should find by key
	path := findSessionFile(tmpDir, "cli:default")
	if path == "" {
		t.Fatal("expected to find session file")
	}
}

func TestFindSessionFile_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	path := findSessionFile(tmpDir, "nonexistent")
	if path != "" {
		t.Fatalf("expected empty path for nonexistent session, got %q", path)
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{2560, "2.5 KB"},
		{1048576, "1.0 MB"},
		{1572864, "1.5 MB"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatSize(tt.bytes)
			if got != tt.want {
				t.Errorf("formatSize(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestListSessionEntries_FallbackToFilename(t *testing.T) {
	tmpDir := t.TempDir()

	// Session with empty key — should fall back to filename
	writeTestSession(t, tmpDir, "orphan.json", sessionData{
		Key:      "",
		Messages: json.RawMessage(`[{"role":"user","content":"hi"}]`),
	})

	entries, err := listSessionEntries(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].id != "orphan" {
		t.Errorf("expected id 'orphan', got %q", entries[0].id)
	}
}

func writeTestSession(t *testing.T, dir, filename string, sess sessionData) {
	t.Helper()
	data, err := json.Marshal(sess)
	if err != nil {
		t.Fatalf("failed to marshal session: %v", err)
	}
	err = os.WriteFile(filepath.Join(dir, filename), data, 0644)
	if err != nil {
		t.Fatalf("failed to write session file: %v", err)
	}
}
