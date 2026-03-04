package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
)

func writeJSONSession(t *testing.T, dir string, sess Session) {
	t.Helper()

	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	filename := sanitizeFilename(sess.Key) + ".json"

	if err := os.WriteFile(filepath.Join(dir, filename), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestMigrate_Basic(t *testing.T) {
	jsonDir := t.TempDir()

	dbPath := filepath.Join(t.TempDir(), "test.db")

	store, err := OpenSQLiteStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}

	defer store.Close()

	writeJSONSession(t, jsonDir, Session{
		Key: "telegram:123",

		Messages: []providers.Message{
			{Role: "user", Content: "hello"},

			{Role: "assistant", Content: "hi"},
		},

		Summary: "greeting",
	})

	writeJSONSession(t, jsonDir, Session{
		Key: "discord:456",

		Messages: []providers.Message{
			{Role: "user", Content: "test"},
		},
	})

	migrated, err := MigrateJSONSessions(jsonDir, store)
	if err != nil {
		t.Fatalf("MigrateJSONSessions: %v", err)
	}

	if migrated != 2 {
		t.Errorf("expected 2 migrated, got %d", migrated)
	}

	// Verify sessions exist

	info, _ := store.Get("telegram:123")

	if info == nil || info.Summary != "greeting" {
		t.Errorf("telegram:123 not found or wrong summary")
	}

	turns, _ := store.Turns("telegram:123", 0)

	if len(turns) != 1 || len(turns[0].Messages) != 2 {
		t.Errorf("expected 1 turn with 2 messages, got %+v", turns)
	}

	info2, _ := store.Get("discord:456")

	if info2 == nil {
		t.Error("discord:456 not found")
	}

	// Verify .json files were renamed

	entries, _ := os.ReadDir(jsonDir)

	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".json" {
			t.Errorf("expected .json.migrated, found %s", e.Name())
		}
	}
}

func TestMigrate_EmptyMessages(t *testing.T) {
	jsonDir := t.TempDir()

	dbPath := filepath.Join(t.TempDir(), "test.db")

	store, _ := OpenSQLiteStore(dbPath)

	defer store.Close()

	writeJSONSession(t, jsonDir, Session{
		Key: "empty:1",

		Messages: []providers.Message{},
	})

	migrated, err := MigrateJSONSessions(jsonDir, store)
	if err != nil {
		t.Fatal(err)
	}

	if migrated != 1 {
		t.Errorf("expected 1, got %d", migrated)
	}

	// Should exist but have no turns

	count, _ := store.TurnCount("empty:1")

	if count != 0 {
		t.Errorf("expected 0 turns, got %d", count)
	}
}

func TestMigrate_EmptySummary(t *testing.T) {
	jsonDir := t.TempDir()

	dbPath := filepath.Join(t.TempDir(), "test.db")

	store, _ := OpenSQLiteStore(dbPath)

	defer store.Close()

	writeJSONSession(t, jsonDir, Session{
		Key: "nosummary:1",

		Messages: []providers.Message{{Role: "user", Content: "hi"}},
	})

	MigrateJSONSessions(jsonDir, store)

	info, _ := store.Get("nosummary:1")

	if info.Summary != "" {
		t.Errorf("expected empty summary, got %q", info.Summary)
	}
}

func TestMigrate_InvalidJSON(t *testing.T) {
	jsonDir := t.TempDir()

	dbPath := filepath.Join(t.TempDir(), "test.db")

	store, _ := OpenSQLiteStore(dbPath)

	defer store.Close()

	// Write invalid JSON

	os.WriteFile(filepath.Join(jsonDir, "bad.json"), []byte("{invalid"), 0o644)

	// Write a valid one too

	writeJSONSession(t, jsonDir, Session{
		Key: "good:1",

		Messages: []providers.Message{{Role: "user", Content: "hi"}},
	})

	migrated, err := MigrateJSONSessions(jsonDir, store)
	if err != nil {
		t.Fatal(err)
	}

	if migrated != 1 {
		t.Errorf("expected 1 (skipped bad), got %d", migrated)
	}

	// Bad file should still be .json (not renamed)

	if _, err := os.Stat(filepath.Join(jsonDir, "bad.json")); os.IsNotExist(err) {
		t.Error("bad.json should still exist")
	}
}

func TestMigrate_Idempotent(t *testing.T) {
	jsonDir := t.TempDir()

	dbPath := filepath.Join(t.TempDir(), "test.db")

	store, _ := OpenSQLiteStore(dbPath)

	defer store.Close()

	writeJSONSession(t, jsonDir, Session{
		Key: "k1",

		Messages: []providers.Message{{Role: "user", Content: "hello"}},
	})

	n1, _ := MigrateJSONSessions(jsonDir, store)

	if n1 != 1 {
		t.Fatalf("first run: expected 1, got %d", n1)
	}

	// Second run should find no .json files (all renamed)

	n2, _ := MigrateJSONSessions(jsonDir, store)

	if n2 != 0 {
		t.Errorf("second run: expected 0, got %d", n2)
	}
}

func TestMigrate_NonExistentDir(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")

	store, _ := OpenSQLiteStore(dbPath)

	defer store.Close()

	n, err := MigrateJSONSessions("/nonexistent/path", store)
	if err != nil {
		t.Fatalf("expected nil error for non-existent dir, got %v", err)
	}

	if n != 0 {
		t.Errorf("expected 0, got %d", n)
	}
}

func TestMigrate_AlreadyExistsInStore(t *testing.T) {
	jsonDir := t.TempDir()

	dbPath := filepath.Join(t.TempDir(), "test.db")

	store, _ := OpenSQLiteStore(dbPath)

	defer store.Close()

	// Pre-create session in store

	store.Create("k1", nil)

	// Write JSON for same key

	writeJSONSession(t, jsonDir, Session{
		Key: "k1",

		Messages: []providers.Message{{Role: "user", Content: "hello"}},
	})

	n, _ := MigrateJSONSessions(jsonDir, store)

	if n != 1 {
		t.Errorf("expected 1, got %d", n)
	}

	// File should still be renamed

	entries, _ := os.ReadDir(jsonDir)

	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".json" {
			t.Errorf("expected .json.migrated, found %s", e.Name())
		}
	}
}
