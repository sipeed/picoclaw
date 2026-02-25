package tools

import (
	"path/filepath"
	"testing"
	"time"
)

func TestTaskLedger_CreateUpdateReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks", "ledger.json")
	ledger := NewTaskLedger(path)

	err := ledger.CreateTask(TaskLedgerEntry{
		ID:            "task-1",
		AgentID:       "main",
		Source:        "spawn",
		Intent:        "fetch weather",
		OriginChannel: "telegram",
		OriginChatID:  "chat-1",
	})
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	if err := ledger.SetStatus("task-1", TaskStatusRunning, "", ""); err != nil {
		t.Fatalf("SetStatus running failed: %v", err)
	}
	if err := ledger.AddEvidence("task-1", TaskEvidence{
		ToolName:      "web_search",
		Arguments:     map[string]any{"query": "weather"},
		ResultPreview: "sunny",
	}); err != nil {
		t.Fatalf("AddEvidence failed: %v", err)
	}
	if err := ledger.SetStatus("task-1", TaskStatusCompleted, "done", ""); err != nil {
		t.Fatalf("SetStatus completed failed: %v", err)
	}

	entry, ok := ledger.Get("task-1")
	if !ok {
		t.Fatal("expected task entry")
	}
	if entry.Status != TaskStatusCompleted {
		t.Fatalf("status = %q, want %q", entry.Status, TaskStatusCompleted)
	}
	if len(entry.Evidence) != 1 {
		t.Fatalf("evidence count = %d, want 1", len(entry.Evidence))
	}

	// Reload from disk and verify persistence.
	ledger2 := NewTaskLedger(path)
	entry2, ok := ledger2.Get("task-1")
	if !ok {
		t.Fatal("expected reloaded task entry")
	}
	if entry2.Result != "done" {
		t.Fatalf("result = %q, want %q", entry2.Result, "done")
	}
	if len(entry2.Evidence) != 1 {
		t.Fatalf("reloaded evidence count = %d, want 1", len(entry2.Evidence))
	}
}

func TestTaskLedger_ListSince(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks", "ledger.json")
	ledger := NewTaskLedger(path)

	now := time.Now().UnixMilli()
	old := now - int64((2 * time.Hour).Milliseconds())
	recent := now - int64((20 * time.Minute).Milliseconds())

	if err := ledger.UpsertTask(TaskLedgerEntry{
		ID:          "task-old",
		Status:      TaskStatusCompleted,
		CreatedAtMS: old,
	}); err != nil {
		t.Fatalf("Upsert old task failed: %v", err)
	}
	if err := ledger.UpsertTask(TaskLedgerEntry{
		ID:          "task-recent",
		Status:      TaskStatusCompleted,
		CreatedAtMS: recent,
	}); err != nil {
		t.Fatalf("Upsert recent task failed: %v", err)
	}

	items := ledger.ListSince(time.Now().Add(-1 * time.Hour))
	if len(items) != 1 {
		t.Fatalf("ListSince returned %d items, want 1", len(items))
	}
	if items[0].ID != "task-recent" {
		t.Fatalf("ListSince first id = %q, want %q", items[0].ID, "task-recent")
	}
}
