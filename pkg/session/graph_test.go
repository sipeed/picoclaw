package session

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
)

func TestSessionGraph_Messages(t *testing.T) {
	store := newTestStore(t)
	if err := store.Create("g1", nil); err != nil {
		t.Fatal(err)
	}
	if err := store.Append("g1", &Turn{
		Kind:     TurnNormal,
		Messages: []providers.Message{{Role: "user", Content: "hello"}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.Append("g1", &Turn{
		Kind: TurnNormal,
		Messages: []providers.Message{
			{Role: "assistant", Content: "hi"},
			{Role: "user", Content: "how are you"},
		},
	}); err != nil {
		t.Fatal(err)
	}

	g := NewSessionGraph(store)
	msgs, err := g.Messages("g1")
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}
	if msgs[0].Content != "hello" || msgs[1].Content != "hi" || msgs[2].Content != "how are you" {
		t.Errorf("unexpected messages: %+v", msgs)
	}
}

func TestSessionGraph_Messages_Empty(t *testing.T) {
	store := newTestStore(t)
	if err := store.Create("empty", nil); err != nil {
		t.Fatal(err)
	}
	g := NewSessionGraph(store)
	msgs, err := g.Messages("empty")
	if err != nil {
		t.Fatal(err)
	}
	if msgs == nil || len(msgs) != 0 {
		t.Errorf("expected empty slice, got %v", msgs)
	}
}

func TestTurnWriter_Commit(t *testing.T) {
	store := newTestStore(t)
	if err := store.Create("tw1", nil); err != nil {
		t.Fatal(err)
	}

	g := NewSessionGraph(store)
	tw := g.BeginTurn("tw1", TurnNormal)
	tw.Add(providers.Message{Role: "user", Content: "msg1"})
	tw.Add(providers.Message{Role: "assistant", Content: "msg2"})
	tw.SetOrigin("parent-key")
	tw.SetAuthor("agent-1")

	if err := tw.Commit(); err != nil {
		t.Fatal(err)
	}

	turns, err := store.Turns("tw1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(turns) != 1 {
		t.Fatalf("expected 1 turn, got %d", len(turns))
	}
	if len(turns[0].Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(turns[0].Messages))
	}
	if turns[0].OriginKey != "parent-key" {
		t.Errorf("expected origin 'parent-key', got %q", turns[0].OriginKey)
	}
	if turns[0].Author != "agent-1" {
		t.Errorf("expected author 'agent-1', got %q", turns[0].Author)
	}
}

func TestTurnWriter_Discard(t *testing.T) {
	store := newTestStore(t)
	if err := store.Create("tw2", nil); err != nil {
		t.Fatal(err)
	}

	g := NewSessionGraph(store)
	tw := g.BeginTurn("tw2", TurnNormal)
	tw.Add(providers.Message{Role: "user", Content: "should not persist"})
	tw.Discard()

	turns, err := store.Turns("tw2", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(turns) != 0 {
		t.Errorf("expected 0 turns after discard, got %d", len(turns))
	}
}

func TestTurnWriter_DoubleCommit(t *testing.T) {
	store := newTestStore(t)
	if err := store.Create("tw3", nil); err != nil {
		t.Fatal(err)
	}

	g := NewSessionGraph(store)
	tw := g.BeginTurn("tw3", TurnNormal)
	tw.Add(providers.Message{Role: "user", Content: "once"})

	if err := tw.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := tw.Commit(); err == nil {
		t.Error("expected error on double commit")
	}
}

func TestTurnWriter_CommitAfterDiscard(t *testing.T) {
	store := newTestStore(t)
	if err := store.Create("tw4", nil); err != nil {
		t.Fatal(err)
	}

	g := NewSessionGraph(store)
	tw := g.BeginTurn("tw4", TurnNormal)
	tw.Add(providers.Message{Role: "user", Content: "x"})
	tw.Discard()

	if err := tw.Commit(); err == nil {
		t.Error("expected error on commit after discard")
	}
}
