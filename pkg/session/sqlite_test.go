package session

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/providers"
)

func newTestStore(t *testing.T) *SQLiteStore {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")

	store, err := OpenSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("OpenSQLiteStore: %v", err)
	}

	t.Cleanup(func() { store.Close() })

	return store
}

func TestSQLite_CreateGetDelete(t *testing.T) {
	store := newTestStore(t)

	// Create

	if err := store.Create("s1", nil); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Get

	info, err := store.Get("s1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if info == nil {
		t.Fatal("expected session, got nil")
	}

	if info.Key != "s1" || info.Status != "active" {
		t.Errorf("unexpected session: %+v", info)
	}

	// Get non-existent

	info, err = store.Get("nope")
	if err != nil {
		t.Fatalf("Get non-existent: %v", err)
	}

	if info != nil {
		t.Errorf("expected nil for non-existent session")
	}

	// Delete

	if delErr := store.Delete("s1"); delErr != nil {
		t.Fatalf("Delete: %v", delErr)
	}

	info, err = store.Get("s1")
	if err != nil {
		t.Fatalf("Get after delete: %v", err)
	}

	if info != nil {
		t.Errorf("expected nil after delete")
	}
}

func TestSQLite_CreateWithOpts(t *testing.T) {
	store := newTestStore(t)

	if err := store.Create("parent", nil); err != nil {
		t.Fatalf("Create parent: %v", err)
	}

	if err := store.Create("child", &CreateOpts{
		ParentKey: "parent",

		ForkTurnID: "turn-1",

		Label: "test child",
	}); err != nil {
		t.Fatalf("Create child: %v", err)
	}

	info, _ := store.Get("child")

	if info.ParentKey != "parent" || info.ForkTurnID != "turn-1" || info.Label != "test child" {
		t.Errorf("unexpected opts: %+v", info)
	}
}

func TestSQLite_List(t *testing.T) {
	store := newTestStore(t)

	store.Create("a", nil)

	store.Create("b", &CreateOpts{ParentKey: "a"})

	store.Create("c", nil)

	all, _ := store.List(nil)

	if len(all) != 3 {
		t.Fatalf("expected 3 sessions, got %d", len(all))
	}

	children, _ := store.List(&ListFilter{ParentKey: "a"})

	if len(children) != 1 || children[0].Key != "b" {
		t.Errorf("unexpected children: %+v", children)
	}

	store.SetStatus("c", "archived")

	active, _ := store.List(&ListFilter{Status: "active"})

	if len(active) != 2 {
		t.Errorf("expected 2 active, got %d", len(active))
	}
}

func TestSQLite_SetStatusSummary(t *testing.T) {
	store := newTestStore(t)

	store.Create("s1", nil)

	if err := store.SetStatus("s1", "archived"); err != nil {
		t.Fatalf("SetStatus: %v", err)
	}

	info, _ := store.Get("s1")

	if info.Status != "archived" {
		t.Errorf("expected archived, got %s", info.Status)
	}

	if err := store.SetSummary("s1", "test summary"); err != nil {
		t.Fatalf("SetSummary: %v", err)
	}

	info, _ = store.Get("s1")

	if info.Summary != "test summary" {
		t.Errorf("expected 'test summary', got %q", info.Summary)
	}

	// Non-existent session

	if err := store.SetStatus("nope", "active"); err == nil {
		t.Error("expected error for non-existent session")
	}
}

func TestSQLite_Children(t *testing.T) {
	store := newTestStore(t)

	store.Create("p", nil)

	store.Create("c1", &CreateOpts{ParentKey: "p"})

	store.Create("c2", &CreateOpts{ParentKey: "p"})

	store.Create("other", nil)

	children, err := store.Children("p")
	if err != nil {
		t.Fatalf("Children: %v", err)
	}

	if len(children) != 2 {
		t.Errorf("expected 2 children, got %d", len(children))
	}
}

func TestSQLite_AppendAndTurns(t *testing.T) {
	store := newTestStore(t)

	store.Create("s1", nil)

	turn1 := &Turn{
		SessionKey: "s1",

		Kind: TurnNormal,

		Messages: []providers.Message{{Role: "user", Content: "hello"}},

		Author: "user",
	}

	if err := store.Append("s1", turn1); err != nil {
		t.Fatalf("Append: %v", err)
	}

	if turn1.ID == "" {
		t.Error("expected ID to be assigned")
	}

	if turn1.Seq != 1 {
		t.Errorf("expected seq 1, got %d", turn1.Seq)
	}

	turn2 := &Turn{
		SessionKey: "s1",

		Kind: TurnNormal,

		Messages: []providers.Message{{Role: "assistant", Content: "hi"}},

		Author: "assistant",
	}

	store.Append("s1", turn2)

	if turn2.Seq != 2 {
		t.Errorf("expected seq 2, got %d", turn2.Seq)
	}

	// Get all turns

	turns, err := store.Turns("s1", 0)
	if err != nil {
		t.Fatalf("Turns: %v", err)
	}

	if len(turns) != 2 {
		t.Fatalf("expected 2 turns, got %d", len(turns))
	}

	// sinceSeq filter

	turns, _ = store.Turns("s1", 1)

	if len(turns) != 1 || turns[0].Seq != 2 {
		t.Errorf("expected 1 turn with seq 2, got %+v", turns)
	}
}

func TestSQLite_LastTurn(t *testing.T) {
	store := newTestStore(t)

	store.Create("s1", nil)

	// No turns

	last, err := store.LastTurn("s1")
	if err != nil {
		t.Fatalf("LastTurn empty: %v", err)
	}

	if last != nil {
		t.Error("expected nil for empty session")
	}

	store.Append("s1", &Turn{Messages: []providers.Message{{Role: "user", Content: "a"}}})

	store.Append("s1", &Turn{Messages: []providers.Message{{Role: "user", Content: "b"}}})

	last, _ = store.LastTurn("s1")

	if last == nil || last.Messages[0].Content != "b" {
		t.Errorf("expected last message 'b', got %+v", last)
	}
}

func TestSQLite_TurnCount(t *testing.T) {
	store := newTestStore(t)

	store.Create("s1", nil)

	count, _ := store.TurnCount("s1")

	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}

	store.Append("s1", &Turn{Messages: []providers.Message{{Role: "user", Content: "a"}}})

	store.Append("s1", &Turn{Messages: []providers.Message{{Role: "user", Content: "b"}}})

	count, _ = store.TurnCount("s1")

	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}
}

func TestSQLite_Compact(t *testing.T) {
	store := newTestStore(t)

	store.Create("s1", nil)

	for i := range 5 {
		store.Append("s1", &Turn{
			Messages: []providers.Message{{Role: "user", Content: string(rune('a' + i))}},
		})
	}

	// Compact up to seq 3

	if err := store.Compact("s1", 3, "summary of first 3 turns"); err != nil {
		t.Fatalf("Compact: %v", err)
	}

	turns, _ := store.Turns("s1", 0)

	if len(turns) != 2 {
		t.Errorf("expected 2 remaining turns, got %d", len(turns))
	}

	if turns[0].Seq != 4 {
		t.Errorf("expected first remaining seq 4, got %d", turns[0].Seq)
	}

	info, _ := store.Get("s1")

	if info.Summary != "summary of first 3 turns" {
		t.Errorf("expected compacted summary, got %q", info.Summary)
	}
}

func TestSQLite_Fork(t *testing.T) {
	store := newTestStore(t)

	store.Create("parent", nil)

	store.Append("parent", &Turn{
		Messages: []providers.Message{{Role: "user", Content: "hello"}},
	})

	last, _ := store.LastTurn("parent")

	if err := store.Fork("parent", "child", &CreateOpts{ForkTurnID: last.ID}); err != nil {
		t.Fatalf("Fork: %v", err)
	}

	child, _ := store.Get("child")

	if child.ParentKey != "parent" || child.ForkTurnID != last.ID {
		t.Errorf("unexpected fork result: %+v", child)
	}

	children, _ := store.Children("parent")

	if len(children) != 1 || children[0].Key != "child" {
		t.Errorf("expected 1 child, got %+v", children)
	}
}

func TestSQLite_Prune(t *testing.T) {
	store := newTestStore(t)

	// Create an old session by manipulating updated_at directly

	store.Create("old", nil)

	store.Create("new", nil)

	old := time.Now().UTC().Add(-48 * time.Hour).Format(time.RFC3339Nano)

	store.db.Exec(`UPDATE sessions SET updated_at = ? WHERE key = ?`, old, "old")

	pruned, err := store.Prune(24 * time.Hour)
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}

	if pruned != 1 {
		t.Errorf("expected 1 pruned, got %d", pruned)
	}

	info, _ := store.Get("old")

	if info != nil {
		t.Error("expected old session to be pruned")
	}

	info, _ = store.Get("new")

	if info == nil {
		t.Error("expected new session to survive prune")
	}
}

func TestSQLite_MessagesRoundTrip(t *testing.T) {
	store := newTestStore(t)

	store.Create("s1", nil)

	msgs := []providers.Message{
		{Role: "user", Content: "hello"},

		{
			Role: "assistant",

			Content: "sure",

			ToolCalls: []providers.ToolCall{
				{
					ID: "call_1",

					Type: "function",

					Function: &providers.FunctionCall{
						Name: "exec",

						Arguments: map[string]any{"cmd": "ls"},
					},
				},
			},
		},

		{Role: "tool", Content: "file1\nfile2", ToolCallID: "call_1"},

		{Role: "assistant", Content: "done"},
	}

	store.Append("s1", &Turn{Messages: msgs})

	turns, _ := store.Turns("s1", 0)

	if len(turns) != 1 {
		t.Fatalf("expected 1 turn, got %d", len(turns))
	}

	got := turns[0].Messages

	if len(got) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(got))
	}

	// Check tool call round-trip

	if got[1].ToolCalls[0].ID != "call_1" {
		t.Errorf("tool call ID mismatch: %s", got[1].ToolCalls[0].ID)
	}

	if got[1].ToolCalls[0].Function.Name != "exec" {
		t.Errorf("tool call function name mismatch: %s", got[1].ToolCalls[0].Function.Name)
	}

	if got[1].ToolCalls[0].Function.Arguments["cmd"] != "ls" {
		t.Errorf("tool call arguments mismatch: %v", got[1].ToolCalls[0].Function.Arguments)
	}

	if got[2].ToolCallID != "call_1" {
		t.Errorf("tool call ID mismatch on result: %s", got[2].ToolCallID)
	}
}

func TestSQLite_CascadeDelete(t *testing.T) {
	store := newTestStore(t)

	store.Create("s1", nil)

	store.Append("s1", &Turn{Messages: []providers.Message{{Role: "user", Content: "a"}}})

	store.Append("s1", &Turn{Messages: []providers.Message{{Role: "user", Content: "b"}}})

	count, _ := store.TurnCount("s1")

	if count != 2 {
		t.Fatalf("expected 2 turns before delete, got %d", count)
	}

	store.Delete("s1")

	count, _ = store.TurnCount("s1")

	if count != 0 {
		t.Errorf("expected 0 turns after cascade delete, got %d", count)
	}
}
