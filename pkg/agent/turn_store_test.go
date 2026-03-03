package agent

import (
	"testing"
	"time"
)

func TestTurnStore_InsertAndQueryRecent(t *testing.T) {
	dir := t.TempDir()
	store, err := NewTurnStore(dir)
	if err != nil {
		t.Fatalf("NewTurnStore: %v", err)
	}
	defer store.Close()

	r := TurnRecord{
		Ts:         time.Now().Unix(),
		ChannelKey: "cli:direct",
		Score:      5,
		Intent:     "task",
		Tags:       []string{"deploy", "ci"},
		UserMsg:    "deploy now",
		Reply:      "done",
		ToolCalls:  []ToolCallRecord{{Name: "exec", Error: ""}},
		Status:     "pending",
	}

	if err := store.Insert(r); err != nil {
		t.Fatalf("Insert: %v", err)
	}

	rows, err := store.QueryRecent("cli:direct", 10)
	if err != nil {
		t.Fatalf("QueryRecent: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].Intent != "task" {
		t.Errorf("unexpected intent: %s", rows[0].Intent)
	}
	if len(rows[0].Tags) != 2 {
		t.Errorf("expected 2 tags, got %v", rows[0].Tags)
	}
}

func TestTurnStore_QueryByScore(t *testing.T) {
	dir := t.TempDir()
	store, err := NewTurnStore(dir)
	if err != nil {
		t.Fatalf("NewTurnStore: %v", err)
	}
	defer store.Close()

	now := time.Now().Unix()
	store.Insert(TurnRecord{ID: "s-1", Ts: now, Score: 3, UserMsg: "a", Reply: "b", Status: "pending"})
	store.Insert(TurnRecord{ID: "s-2", Ts: now + 1, Score: 8, UserMsg: "c", Reply: "d", Status: "pending"})
	store.Insert(TurnRecord{ID: "s-3", Ts: now + 2, Score: 9, UserMsg: "e", Reply: "f", Status: "pending"})

	high, err := store.QueryByScore(7)
	if err != nil {
		t.Fatalf("QueryByScore: %v", err)
	}
	if len(high) != 2 {
		t.Errorf("expected 2 always_keep turns, got %d", len(high))
	}
}

func TestTurnStore_SetStatus(t *testing.T) {
	dir := t.TempDir()
	store, err := NewTurnStore(dir)
	if err != nil {
		t.Fatalf("NewTurnStore: %v", err)
	}
	defer store.Close()

	r := TurnRecord{ID: "test-id-1", Ts: time.Now().Unix(), UserMsg: "x", Reply: "y", Status: "pending"}
	store.Insert(r)

	if err := store.SetStatus("test-id-1", "processed"); err != nil {
		t.Fatalf("SetStatus: %v", err)
	}

	pending, err := store.QueryPending(10)
	if err != nil {
		t.Fatalf("QueryPending: %v", err)
	}
	if len(pending) != 0 {
		t.Errorf("expected 0 pending, got %d", len(pending))
	}
}

func TestTurnStore_ArchiveOldProcessed(t *testing.T) {
	dir := t.TempDir()
	store, err := NewTurnStore(dir)
	if err != nil {
		t.Fatalf("NewTurnStore: %v", err)
	}
	defer store.Close()

	// Insert old processed turns (timestamp in the past).
	old := time.Now().AddDate(0, 0, -10).Unix()
	for i := 0; i < 3; i++ {
		r := TurnRecord{Ts: old, Score: 2, UserMsg: "old", Reply: "msg", Status: "processed"}
		store.Insert(r)
	}

	// Recent processed turn — should NOT be archived.
	recent := TurnRecord{Ts: time.Now().Unix(), Score: 2, UserMsg: "new", Reply: "msg", Status: "processed"}
	store.Insert(recent)

	if err := store.ArchiveOldProcessed(7); err != nil {
		t.Fatalf("ArchiveOldProcessed: %v", err)
	}

	// Query pending (should still be 0).
	pending, _ := store.QueryPending(100)
	if len(pending) != 0 {
		t.Errorf("expected 0 pending after archive, got %d", len(pending))
	}
}

func TestTurnStore_QueryByTags(t *testing.T) {
	dir := t.TempDir()
	store, err := NewTurnStore(dir)
	if err != nil {
		t.Fatalf("NewTurnStore: %v", err)
	}
	defer store.Close()

	now := time.Now().Unix()
	store.Insert(TurnRecord{ID: "tag-1", Ts: now, Score: 5, Tags: []string{"deploy", "ci"}, UserMsg: "a", Reply: "b"})
	store.Insert(TurnRecord{ID: "tag-2", Ts: now + 1, Score: 4, Tags: []string{"file", "read"}, UserMsg: "c", Reply: "d"})
	store.Insert(TurnRecord{ID: "tag-3", Ts: now + 2, Score: 3, Tags: []string{"deploy", "log"}, UserMsg: "e", Reply: "f"})

	rows, err := store.QueryByTags([]string{"deploy"})
	if err != nil {
		t.Fatalf("QueryByTags: %v", err)
	}
	if len(rows) < 2 {
		t.Errorf("expected at least 2 deploy turns, got %d", len(rows))
	}
}
