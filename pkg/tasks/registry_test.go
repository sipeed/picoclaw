package tasks

import (
	"path/filepath"
	"testing"
	"time"
)

func TestRegistryPersistsAndReloadsRecords(t *testing.T) {
	store := filepath.Join(t.TempDir(), "state", "task_registry.json")

	registry := NewRegistry(store)
	if err := registry.Upsert(Record{
		TaskID:         "subagent-7",
		Runtime:        RuntimeSubagent,
		Task:           "download media",
		Status:         StatusRunning,
		DeliveryStatus: DeliveryPending,
	}); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	if err := registry.Update("subagent-7", func(rec *Record) {
		rec.Status = StatusSucceeded
		rec.DeliveryStatus = DeliveryDelivered
		rec.TerminalSummary = "done"
	}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	reloaded := NewRegistry(store)
	rec, ok := reloaded.Get("subagent-7")
	if !ok {
		t.Fatal("expected persisted task after reload")
	}
	if rec.Status != StatusSucceeded {
		t.Fatalf("Status = %q, want %q", rec.Status, StatusSucceeded)
	}
	if rec.DeliveryStatus != DeliveryDelivered {
		t.Fatalf("DeliveryStatus = %q, want %q", rec.DeliveryStatus, DeliveryDelivered)
	}
	if rec.TerminalSummary != "done" {
		t.Fatalf("TerminalSummary = %q, want done", rec.TerminalSummary)
	}
}

func TestRegistryMaxNumericSuffix(t *testing.T) {
	registry := NewRegistry("")
	for _, id := range []string{"subagent-2", "subagent-10", "other-99"} {
		if err := registry.Upsert(Record{TaskID: id, Runtime: RuntimeSubagent, Task: "t"}); err != nil {
			t.Fatalf("Upsert(%s) error = %v", id, err)
		}
	}

	if got := registry.MaxNumericSuffix("subagent-"); got != 10 {
		t.Fatalf("MaxNumericSuffix() = %d, want 10", got)
	}
}

func TestRegistryStampsCleanupAfterForTerminalTasks(t *testing.T) {
	registry := NewRegistryWithOptions("", Options{TerminalRetention: time.Hour})
	endedAt := time.Now().UnixMilli()
	if err := registry.Upsert(Record{
		TaskID:         "task-1",
		Runtime:        RuntimeSubagent,
		Task:           "done",
		Status:         StatusSucceeded,
		DeliveryStatus: DeliveryDelivered,
		EndedAt:        endedAt,
	}); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}

	rec, ok := registry.Get("task-1")
	if !ok {
		t.Fatal("expected task")
	}
	if rec.CleanupAfter != endedAt+int64(time.Hour/time.Millisecond) {
		t.Fatalf("CleanupAfter = %d, want %d", rec.CleanupAfter, endedAt+int64(time.Hour/time.Millisecond))
	}
}

func TestRegistryPrunesExpiredTerminalTasks(t *testing.T) {
	store := filepath.Join(t.TempDir(), "state", "task_registry.json")
	registry := NewRegistryWithOptions(store, Options{TerminalRetention: time.Millisecond})

	if err := registry.Upsert(Record{
		TaskID:         "old-done",
		Runtime:        RuntimeSubagent,
		Task:           "old",
		Status:         StatusSucceeded,
		DeliveryStatus: DeliveryDelivered,
		EndedAt:        time.Now().Add(-time.Hour).UnixMilli(),
	}); err != nil {
		t.Fatalf("Upsert(old) error = %v", err)
	}
	if err := registry.Upsert(Record{
		TaskID:         "active",
		Runtime:        RuntimeSubagent,
		Task:           "active",
		Status:         StatusRunning,
		DeliveryStatus: DeliveryPending,
		CreatedAt:      time.Now().Add(-time.Hour).UnixMilli(),
	}); err != nil {
		t.Fatalf("Upsert(active) error = %v", err)
	}

	if _, ok := registry.Get("old-done"); ok {
		t.Fatal("expected expired terminal task to be pruned")
	}
	if _, ok := registry.Get("active"); !ok {
		t.Fatal("expected active task to be preserved")
	}
}

func TestRegistryPrunesOldestTerminalTasksAboveMaxRecords(t *testing.T) {
	registry := NewRegistryWithOptions("", Options{
		TerminalRetention: 24 * time.Hour,
		MaxRecords:        3,
	})

	records := []Record{
		{TaskID: "active-1", Status: StatusRunning, CreatedAt: time.Now().Add(-4 * time.Minute).UnixMilli()},
		{TaskID: "active-2", Status: StatusRunning, CreatedAt: time.Now().Add(-3 * time.Minute).UnixMilli()},
		{TaskID: "old-terminal", Status: StatusSucceeded, CreatedAt: time.Now().Add(-2 * time.Minute).UnixMilli(), EndedAt: time.Now().Add(-2 * time.Minute).UnixMilli()},
		{TaskID: "new-terminal", Status: StatusSucceeded, CreatedAt: time.Now().Add(-time.Minute).UnixMilli(), EndedAt: time.Now().Add(-time.Minute).UnixMilli()},
	}
	for _, rec := range records {
		rec.Runtime = RuntimeSubagent
		rec.Task = rec.TaskID
		rec.DeliveryStatus = DeliveryPending
		if err := registry.Upsert(rec); err != nil {
			t.Fatalf("Upsert(%s) error = %v", rec.TaskID, err)
		}
	}

	if _, ok := registry.Get("old-terminal"); ok {
		t.Fatal("expected oldest terminal task to be pruned")
	}
	for _, id := range []string{"active-1", "active-2", "new-terminal"} {
		if _, ok := registry.Get(id); !ok {
			t.Fatalf("expected %s to be preserved", id)
		}
	}
}

func TestRegistryListPendingTerminalDelivery(t *testing.T) {
	registry := NewRegistry("")
	records := []Record{
		{TaskID: "pending-done", Status: StatusSucceeded, DeliveryStatus: DeliveryPending},
		{TaskID: "pending-failed", Status: StatusFailed, DeliveryStatus: DeliveryPending},
		{TaskID: "pending-running", Status: StatusRunning, DeliveryStatus: DeliveryPending},
		{TaskID: "delivered-done", Status: StatusSucceeded, DeliveryStatus: DeliveryDelivered},
	}
	for _, rec := range records {
		rec.Runtime = RuntimeSubagent
		rec.Task = rec.TaskID
		if err := registry.Upsert(rec); err != nil {
			t.Fatalf("Upsert(%s) error = %v", rec.TaskID, err)
		}
	}

	got := registry.ListPendingTerminalDelivery()
	if len(got) != 2 {
		t.Fatalf("pending terminal count = %d, want 2: %+v", len(got), got)
	}
	if got[0].TaskID != "pending-done" || got[1].TaskID != "pending-failed" {
		t.Fatalf("pending terminal tasks = %v, want pending-done,pending-failed", []string{got[0].TaskID, got[1].TaskID})
	}
}
