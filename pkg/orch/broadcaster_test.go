package orch

import (
	"testing"
	"time"
)

func TestBroadcasterSpawnAndGC(t *testing.T) {
	b := NewBroadcaster()
	sub := b.Subscribe()
	defer b.Unsubscribe(sub)

	b.Publish(Event{Type: "agent_spawn", ID: "t1", Label: "scout", Task: "do something"})

	select {
	case ev := <-sub.Ch:
		if ev.Type != "agent_spawn" || ev.ID != "t1" {
			t.Fatalf("expected agent_spawn for t1, got %+v", ev)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for agent_spawn event")
	}

	snap := b.Snapshot()
	if len(snap) != 1 || snap[0].ID != "t1" {
		t.Fatalf("expected 1 agent in snapshot, got %v", snap)
	}

	b.Publish(Event{Type: "agent_gc", ID: "t1", Reason: "completed"})

	select {
	case ev := <-sub.Ch:
		if ev.Type != "agent_gc" || ev.Reason != "completed" {
			t.Fatalf("expected agent_gc/completed, got %+v", ev)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for agent_gc event")
	}

	if len(b.Snapshot()) != 0 {
		t.Fatal("snapshot should be empty after agent_gc")
	}
}

func TestBroadcasterAgentState(t *testing.T) {
	b := NewBroadcaster()
	sub := b.Subscribe()
	defer b.Unsubscribe(sub)

	b.Publish(Event{Type: "agent_spawn", ID: "t1"})
	<-sub.Ch // consume spawn

	b.Publish(Event{Type: "agent_state", ID: "t1", State: "toolcall", Tool: "bash"})

	select {
	case ev := <-sub.Ch:
		if ev.State != "toolcall" || ev.Tool != "bash" {
			t.Fatalf("unexpected state event: %+v", ev)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for agent_state event")
	}

	snap := b.Snapshot()
	if len(snap) == 0 || snap[0].State != "toolcall" || snap[0].Tool != "bash" {
		t.Fatalf("snapshot state not updated: %v", snap)
	}
}

func TestBroadcasterNonBlocking(t *testing.T) {
	b := NewBroadcaster()
	sub := b.Subscribe() // do NOT read from sub.Ch
	defer b.Unsubscribe(sub)

	// Fill buffer beyond capacity (cap=32) — must not block or deadlock
	done := make(chan struct{})
	go func() {
		for i := 0; i < 50; i++ {
			b.Publish(Event{Type: "agent_state", ID: "t1", State: "waiting"})
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Publish blocked on slow subscriber")
	}
}

func TestBroadcasterMultipleSubscribers(t *testing.T) {
	b := NewBroadcaster()
	s1 := b.Subscribe()
	s2 := b.Subscribe()
	defer b.Unsubscribe(s1)
	defer b.Unsubscribe(s2)

	b.Publish(Event{Type: "agent_spawn", ID: "t1", Label: "worker"})

	for _, sub := range []*Subscriber{s1, s2} {
		select {
		case ev := <-sub.Ch:
			if ev.Type != "agent_spawn" {
				t.Fatalf("expected agent_spawn, got %s", ev.Type)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timeout: not all subscribers received event")
		}
	}
}

func TestBroadcasterUnsubscribe(t *testing.T) {
	b := NewBroadcaster()
	sub := b.Subscribe()
	b.Unsubscribe(sub)

	b.Publish(Event{Type: "agent_spawn", ID: "t1"})

	select {
	case ev := <-sub.Ch:
		t.Fatalf("received event after unsubscribe: %+v", ev)
	case <-time.After(50 * time.Millisecond):
		// correct: nothing delivered after unsubscribe
	}
}

func TestBroadcasterTimestampAutoSet(t *testing.T) {
	b := NewBroadcaster()
	sub := b.Subscribe()
	defer b.Unsubscribe(sub)

	before := time.Now().UnixMilli()
	b.Publish(Event{Type: "agent_spawn", ID: "t1"}) // Created == 0
	after := time.Now().UnixMilli()

	ev := <-sub.Ch
	if ev.Created < before || ev.Created > after {
		t.Fatalf("Created timestamp %d not in [%d, %d]", ev.Created, before, after)
	}
}
