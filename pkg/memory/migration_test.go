package memory

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/providers"
)

func writeJSONSession(
	t *testing.T, dir string, filename string, sess jsonSession,
) {
	t.Helper()
	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		t.Fatalf("marshal session: %v", err)
	}
	err = os.WriteFile(filepath.Join(dir, filename), data, 0o644)
	if err != nil {
		t.Fatalf("write session file: %v", err)
	}
}

func TestMigrateFromJSON_Basic(t *testing.T) {
	sessionsDir := t.TempDir()
	store := newTestStore(t)
	ctx := context.Background()

	writeJSONSession(t, sessionsDir, "test.json", jsonSession{
		Key: "test",
		Messages: []providers.Message{
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "hi"},
		},
		Summary: "A greeting.",
		Created: time.Now(),
		Updated: time.Now(),
	})

	count, err := MigrateFromJSON(ctx, sessionsDir, store)
	if err != nil {
		t.Fatalf("MigrateFromJSON: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 migrated, got %d", count)
	}

	history, err := store.GetHistory(ctx, "test")
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(history))
	}
	if history[0].Content != "hello" || history[1].Content != "hi" {
		t.Errorf("unexpected messages: %+v", history)
	}

	summary, err := store.GetSummary(ctx, "test")
	if err != nil {
		t.Fatalf("GetSummary: %v", err)
	}
	if summary != "A greeting." {
		t.Errorf("summary = %q", summary)
	}
}

func TestMigrateFromJSON_WithToolCalls(t *testing.T) {
	sessionsDir := t.TempDir()
	store := newTestStore(t)
	ctx := context.Background()

	writeJSONSession(t, sessionsDir, "tools.json", jsonSession{
		Key: "tools",
		Messages: []providers.Message{
			{
				Role:    "assistant",
				Content: "Searching...",
				ToolCalls: []providers.ToolCall{
					{
						ID:   "call_1",
						Type: "function",
						Function: &providers.FunctionCall{
							Name:      "web_search",
							Arguments: `{"q":"test"}`,
						},
					},
				},
			},
			{
				Role:       "tool",
				Content:    "result",
				ToolCallID: "call_1",
			},
		},
		Created: time.Now(),
		Updated: time.Now(),
	})

	count, err := MigrateFromJSON(ctx, sessionsDir, store)
	if err != nil {
		t.Fatalf("MigrateFromJSON: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1, got %d", count)
	}

	history, err := store.GetHistory(ctx, "tools")
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(history))
	}
	if len(history[0].ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(history[0].ToolCalls))
	}
	if history[0].ToolCalls[0].Function.Name != "web_search" {
		t.Errorf("function = %q", history[0].ToolCalls[0].Function.Name)
	}
	if history[1].ToolCallID != "call_1" {
		t.Errorf("ToolCallID = %q", history[1].ToolCallID)
	}
}

func TestMigrateFromJSON_MultipleFiles(t *testing.T) {
	sessionsDir := t.TempDir()
	store := newTestStore(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		key := string(rune('a' + i))
		writeJSONSession(t, sessionsDir, key+".json", jsonSession{
			Key:      key,
			Messages: []providers.Message{{Role: "user", Content: "msg " + key}},
			Created:  time.Now(),
			Updated:  time.Now(),
		})
	}

	count, err := MigrateFromJSON(ctx, sessionsDir, store)
	if err != nil {
		t.Fatalf("MigrateFromJSON: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3, got %d", count)
	}

	for i := 0; i < 3; i++ {
		key := string(rune('a' + i))
		history, histErr := store.GetHistory(ctx, key)
		if histErr != nil {
			t.Fatalf("GetHistory(%q): %v", key, histErr)
		}
		if len(history) != 1 {
			t.Errorf("session %q: expected 1 msg, got %d", key, len(history))
		}
	}
}

func TestMigrateFromJSON_InvalidJSON(t *testing.T) {
	sessionsDir := t.TempDir()
	store := newTestStore(t)
	ctx := context.Background()

	// One valid, one invalid.
	writeJSONSession(t, sessionsDir, "good.json", jsonSession{
		Key:      "good",
		Messages: []providers.Message{{Role: "user", Content: "ok"}},
		Created:  time.Now(),
		Updated:  time.Now(),
	})
	err := os.WriteFile(
		filepath.Join(sessionsDir, "bad.json"),
		[]byte("{invalid json"),
		0o644,
	)
	if err != nil {
		t.Fatalf("write bad file: %v", err)
	}

	count, err := MigrateFromJSON(ctx, sessionsDir, store)
	if err != nil {
		t.Fatalf("MigrateFromJSON: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 (bad file skipped), got %d", count)
	}

	history, err := store.GetHistory(ctx, "good")
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) != 1 {
		t.Errorf("expected 1 message, got %d", len(history))
	}
}

func TestMigrateFromJSON_RenamesFiles(t *testing.T) {
	sessionsDir := t.TempDir()
	store := newTestStore(t)
	ctx := context.Background()

	writeJSONSession(t, sessionsDir, "rename.json", jsonSession{
		Key:      "rename",
		Messages: []providers.Message{{Role: "user", Content: "hi"}},
		Created:  time.Now(),
		Updated:  time.Now(),
	})

	_, err := MigrateFromJSON(ctx, sessionsDir, store)
	if err != nil {
		t.Fatalf("MigrateFromJSON: %v", err)
	}

	// Original .json should not exist.
	_, statErr := os.Stat(filepath.Join(sessionsDir, "rename.json"))
	if !os.IsNotExist(statErr) {
		t.Error("rename.json should have been renamed")
	}
	// .json.migrated should exist.
	_, statErr = os.Stat(
		filepath.Join(sessionsDir, "rename.json.migrated"),
	)
	if statErr != nil {
		t.Errorf("rename.json.migrated should exist: %v", statErr)
	}
}

func TestMigrateFromJSON_Idempotent(t *testing.T) {
	sessionsDir := t.TempDir()
	store := newTestStore(t)
	ctx := context.Background()

	writeJSONSession(t, sessionsDir, "idem.json", jsonSession{
		Key:      "idem",
		Messages: []providers.Message{{Role: "user", Content: "once"}},
		Created:  time.Now(),
		Updated:  time.Now(),
	})

	count1, err := MigrateFromJSON(ctx, sessionsDir, store)
	if err != nil {
		t.Fatalf("first migration: %v", err)
	}
	if count1 != 1 {
		t.Errorf("first run: expected 1, got %d", count1)
	}

	// Second run should find only .migrated files, skip them.
	count2, err := MigrateFromJSON(ctx, sessionsDir, store)
	if err != nil {
		t.Fatalf("second migration: %v", err)
	}
	if count2 != 0 {
		t.Errorf("second run: expected 0, got %d", count2)
	}

	history, err := store.GetHistory(ctx, "idem")
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) != 1 {
		t.Errorf("expected 1 message, got %d", len(history))
	}
}

func TestMigrateFromJSON_ColonInKey(t *testing.T) {
	sessionsDir := t.TempDir()
	store := newTestStore(t)
	ctx := context.Background()

	// File is named telegram_123 (sanitized), but the key inside is telegram:123.
	writeJSONSession(t, sessionsDir, "telegram_123.json", jsonSession{
		Key:      "telegram:123",
		Messages: []providers.Message{{Role: "user", Content: "from telegram"}},
		Created:  time.Now(),
		Updated:  time.Now(),
	})

	count, err := MigrateFromJSON(ctx, sessionsDir, store)
	if err != nil {
		t.Fatalf("MigrateFromJSON: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1, got %d", count)
	}

	// Accessible via the original key "telegram:123".
	history, err := store.GetHistory(ctx, "telegram:123")
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 message, got %d", len(history))
	}
	if history[0].Content != "from telegram" {
		t.Errorf("content = %q", history[0].Content)
	}

	// In the file-based store, "telegram:123" and "telegram_123" both
	// sanitize to the same filename, so they share storage. This is
	// expected — the colon-to-underscore mapping is a one-way function.
	history2, err := store.GetHistory(ctx, "telegram_123")
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history2) != 1 {
		t.Errorf("expected 1 (same file), got %d", len(history2))
	}
}

func TestMigrateFromJSON_NonexistentDir(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	count, err := MigrateFromJSON(ctx, "/nonexistent/path", store)
	if err != nil {
		t.Fatalf("MigrateFromJSON: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
}
