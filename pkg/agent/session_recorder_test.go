package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/session"
)

func newTestRecorder(t *testing.T) (*sessionRecorderImpl, *session.LegacyAdapter, session.SessionStore) {
	t.Helper()

	tmpDir := t.TempDir()

	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := session.OpenSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	adapter := session.NewLegacyAdapter(store)

	t.Cleanup(func() { adapter.Close() })

	recorder := newSessionRecorder(adapter)

	return recorder, adapter, store
}

func TestRecordFork(t *testing.T) {
	rec, _, store := newTestRecorder(t)

	// Create conductor session first.

	if err := store.Create("conductor:main", nil); err != nil {
		t.Fatalf("create conductor session: %v", err)
	}

	err := rec.RecordFork("conductor:main", "subagent:subagent-1", "subagent-1", "scout")
	if err != nil {
		t.Fatalf("RecordFork: %v", err)
	}

	// Verify child session exists with correct parent.

	info, err := store.Get("subagent:subagent-1")
	if err != nil {
		t.Fatalf("Get child: %v", err)
	}

	if info == nil {
		t.Fatal("child session not found")
	}

	if info.ParentKey != "conductor:main" {
		t.Errorf("ParentKey = %q, want %q", info.ParentKey, "conductor:main")
	}

	if info.ForkTurnID != "subagent-1" {
		t.Errorf("ForkTurnID = %q, want %q", info.ForkTurnID, "subagent-1")
	}

	if info.Label != "scout" {
		t.Errorf("Label = %q, want %q", info.Label, "scout")
	}

	// Verify parent lists child.

	children, err := store.Children("conductor:main")
	if err != nil {
		t.Fatalf("Children: %v", err)
	}

	if len(children) != 1 {
		t.Fatalf("children count = %d, want 1", len(children))
	}

	if children[0].Key != "subagent:subagent-1" {
		t.Errorf("child key = %q, want %q", children[0].Key, "subagent:subagent-1")
	}
}

func TestRecordSubagentTurn(t *testing.T) {
	rec, _, store := newTestRecorder(t)

	// Create subagent session.

	if err := store.Create("subagent:subagent-1", nil); err != nil {
		t.Fatalf("create: %v", err)
	}

	msgs := []providers.Message{
		{Role: "system", Content: "You are a scout."},

		{Role: "user", Content: "Investigate X."},

		{Role: "assistant", Content: "Found Y."},
	}

	if err := rec.RecordSubagentTurn("subagent:subagent-1", msgs); err != nil {
		t.Fatalf("RecordSubagentTurn: %v", err)
	}

	turns, err := store.Turns("subagent:subagent-1", 0)
	if err != nil {
		t.Fatalf("Turns: %v", err)
	}

	if len(turns) != 1 {
		t.Fatalf("turns count = %d, want 1", len(turns))
	}

	if turns[0].Kind != session.TurnNormal {
		t.Errorf("Kind = %d, want TurnNormal", turns[0].Kind)
	}

	if len(turns[0].Messages) != 3 {
		t.Errorf("messages count = %d, want 3", len(turns[0].Messages))
	}

	if turns[0].Messages[2].Content != "Found Y." {
		t.Errorf("last message = %q, want %q", turns[0].Messages[2].Content, "Found Y.")
	}
}

func TestRecordCompletion(t *testing.T) {
	rec, _, store := newTestRecorder(t)

	if err := store.Create("subagent:subagent-1", nil); err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := rec.RecordCompletion("subagent:subagent-1", "completed", "done"); err != nil {
		t.Fatalf("RecordCompletion: %v", err)
	}

	info, err := store.Get("subagent:subagent-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if info.Status != "completed" {
		t.Errorf("Status = %q, want %q", info.Status, "completed")
	}

	// Test failed status.

	if err := store.Create("subagent:subagent-2", nil); err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := rec.RecordCompletion("subagent:subagent-2", "failed", "error"); err != nil {
		t.Fatalf("RecordCompletion failed: %v", err)
	}

	info2, _ := store.Get("subagent:subagent-2")

	if info2.Status != "failed" {
		t.Errorf("Status = %q, want %q", info2.Status, "failed")
	}
}

func TestRecordReport(t *testing.T) {
	rec, adapter, store := newTestRecorder(t)

	// Create conductor session via adapter so it's in cache.

	_ = adapter.GetOrCreate("conductor:main")

	if err := store.Create("subagent:subagent-1", nil); err != nil {
		t.Fatalf("create subagent: %v", err)
	}

	content := "[System: subagent:subagent-1] Task 'scout' completed.\n\nResult:\nFound Y."

	if err := rec.RecordReport("conductor:main", "subagent:subagent-1", "subagent:subagent-1", content); err != nil {
		t.Fatalf("RecordReport: %v", err)
	}

	// Verify TurnReport in store.

	turns, err := store.Turns("conductor:main", 0)
	if err != nil {
		t.Fatalf("Turns: %v", err)
	}

	if len(turns) != 1 {
		t.Fatalf("turns count = %d, want 1", len(turns))
	}

	if turns[0].Kind != session.TurnReport {
		t.Errorf("Kind = %d, want TurnReport(%d)", turns[0].Kind, session.TurnReport)
	}

	if turns[0].OriginKey != "subagent:subagent-1" {
		t.Errorf("OriginKey = %q, want %q", turns[0].OriginKey, "subagent:subagent-1")
	}

	if turns[0].Author != "subagent:subagent-1" {
		t.Errorf("Author = %q, want %q", turns[0].Author, "subagent:subagent-1")
	}

	if len(turns[0].Messages) != 1 || turns[0].Messages[0].Role != "user" {
		t.Errorf("unexpected messages: %v", turns[0].Messages)
	}
}

func TestAdvanceStoredPreventsDoubleWrite(t *testing.T) {
	rec, adapter, store := newTestRecorder(t)

	// Create conductor session via adapter.

	_ = adapter.GetOrCreate("conductor:main")

	if err := store.Create("subagent:subagent-1", nil); err != nil {
		t.Fatalf("create subagent: %v", err)
	}

	// Simulate: conductor has 2 messages already flushed.

	adapter.AddMessage("conductor:main", "user", "hello")

	adapter.AddMessage("conductor:main", "assistant", "hi")

	if err := adapter.Save("conductor:main"); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// RecordReport writes directly to store and advances stored counter.

	content := "[System: subagent:subagent-1] result"

	if err := rec.RecordReport("conductor:main", "subagent:subagent-1", "subagent:subagent-1", content); err != nil {
		t.Fatalf("RecordReport: %v", err)
	}

	// The in-memory cache should also be updated (by loop.go calling AddFullMessage).

	// Simulate what loop.go does after RecordReport succeeds.

	adapter.AddFullMessage("conductor:main", providers.Message{Role: "user", Content: content})

	// AdvanceStored was already called by RecordReport, so stored = 3 + 1 = 4

	// but we added 1 message to cache making it len=4 as well. No double write.

	// Save should NOT re-write the report turn.

	if err := adapter.Save("conductor:main"); err != nil {
		t.Fatalf("Save after report: %v", err)
	}

	// Count all turns in store for conductor session.

	turns, err := store.Turns("conductor:main", 0)
	if err != nil {
		t.Fatalf("Turns: %v", err)
	}

	// Expected: turn 1 (initial 2 msgs), turn 2 (TurnReport from RecordReport)

	// NOT turn 3 (duplicate from flush).

	if len(turns) != 2 {
		t.Errorf("turns count = %d, want 2 (no double-write)", len(turns))

		for i, turn := range turns {
			t.Logf("  turn[%d]: seq=%d kind=%d msgs=%d", i, turn.Seq, turn.Kind, len(turn.Messages))
		}
	}
}

func TestRecordQuestion(t *testing.T) {
	rec, adapter, store := newTestRecorder(t)

	// Create conductor session via adapter so it's in cache.

	_ = adapter.GetOrCreate("conductor:main")

	if err := store.Create("subagent:subagent-1", nil); err != nil {
		t.Fatalf("create subagent: %v", err)
	}

	question := "What database schema should I use for the users table?"

	if err := rec.RecordQuestion("conductor:main", "subagent:subagent-1", "subagent-1", question); err != nil {
		t.Fatalf("RecordQuestion: %v", err)
	}

	turns, err := store.Turns("conductor:main", 0)
	if err != nil {
		t.Fatalf("Turns: %v", err)
	}

	if len(turns) != 1 {
		t.Fatalf("turns count = %d, want 1", len(turns))
	}

	if turns[0].Kind != session.TurnQuestion {
		t.Errorf("Kind = %d, want TurnQuestion(%d)", turns[0].Kind, session.TurnQuestion)
	}

	if turns[0].OriginKey != "subagent:subagent-1" {
		t.Errorf("OriginKey = %q, want %q", turns[0].OriginKey, "subagent:subagent-1")
	}

	if turns[0].Author != "subagent-1" {
		t.Errorf("Author = %q, want %q", turns[0].Author, "subagent-1")
	}

	if len(turns[0].Messages) != 1 || turns[0].Messages[0].Content != question {
		t.Errorf("unexpected messages: %v", turns[0].Messages)
	}
}

func TestRecordPlanSubmit(t *testing.T) {
	rec, adapter, store := newTestRecorder(t)

	_ = adapter.GetOrCreate("conductor:main")

	if err := store.Create("subagent:subagent-1", nil); err != nil {
		t.Fatalf("create subagent: %v", err)
	}

	planText := "Goal: Implement auth\nSteps:\n1. Add middleware\n2. Add JWT validation"

	if err := rec.RecordPlanSubmit("conductor:main", "subagent:subagent-1", "subagent-1", planText); err != nil {
		t.Fatalf("RecordPlanSubmit: %v", err)
	}

	turns, err := store.Turns("conductor:main", 0)
	if err != nil {
		t.Fatalf("Turns: %v", err)
	}

	if len(turns) != 1 {
		t.Fatalf("turns count = %d, want 1", len(turns))
	}

	if turns[0].Kind != session.TurnPlanSubmit {
		t.Errorf("Kind = %d, want TurnPlanSubmit(%d)", turns[0].Kind, session.TurnPlanSubmit)
	}

	if turns[0].OriginKey != "subagent:subagent-1" {
		t.Errorf("OriginKey = %q, want %q", turns[0].OriginKey, "subagent:subagent-1")
	}

	if len(turns[0].Messages) != 1 || turns[0].Messages[0].Content != planText {
		t.Errorf("unexpected messages: %v", turns[0].Messages)
	}
}

func TestExtractTaskID(t *testing.T) {
	tests := []struct {
		input string

		want string
	}{
		{"subagent:subagent-1", "subagent-1"},

		{"subagent:subagent-42", "subagent-42"},

		{"plain-id", "plain-id"},

		{"a:b:c", "c"},
	}

	for _, tt := range tests {
		got := extractTaskID(tt.input)

		if got != tt.want {
			t.Errorf("extractTaskID(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func init() {
	// Suppress log output in tests.

	os.Setenv("PICOCLAW_LOG_LEVEL", "error")
}
