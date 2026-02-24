package tools

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/orch"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// blockingProvider blocks inside Chat until the context is cancelled.
// The ready channel is closed the moment Chat is entered, so callers can
// synchronise before cancelling the context.
type blockingProvider struct {
	ready chan struct{}
}

func newBlockingProvider() *blockingProvider {
	return &blockingProvider{ready: make(chan struct{})}
}

func (p *blockingProvider) Chat(ctx context.Context, _ []providers.Message, _ []providers.ToolDefinition, _ string, _ map[string]any) (*providers.LLMResponse, error) {
	close(p.ready) // signal: we are now blocking
	<-ctx.Done()
	return nil, ctx.Err()
}

func (p *blockingProvider) GetDefaultModel() string { return "test" }

// TestSubagentManager_Spawn_EmitsLifecycleEvents verifies that Spawn() fires
// the correct sequence of orchestration events through a real Broadcaster:
//
//	agent_spawn → conversation(conductor→sub) → agent_state(waiting) →
//	conversation(sub→conductor) → agent_gc(completed)
//
// It also verifies that the snapshot is empty after ReportGC and that the
// completion callback is invoked.
func TestSubagentManager_Spawn_EmitsLifecycleEvents(t *testing.T) {
	b := orch.NewBroadcaster()
	sub := b.Subscribe()
	defer b.Unsubscribe(sub)

	provider := &MockLLMProvider{}
	mgr := NewSubagentManager(provider, "test-model", "/tmp/test", nil, b)

	var callbackCalled int32
	cb := AsyncCallback(func(_ context.Context, _ *ToolResult) {
		atomic.StoreInt32(&callbackCalled, 1)
	})

	_, err := mgr.Spawn(
		context.Background(),
		"say hello", "hello-task", "", "cli", "direct",
		cb,
	)
	if err != nil {
		t.Fatalf("Spawn() error: %v", err)
	}

	// Collect events until agent_gc or timeout.
	var events []orch.Event
	deadline := time.After(3 * time.Second)
loop:
	for {
		select {
		case ev := <-sub.Ch:
			events = append(events, ev)
			if ev.Type == "agent_gc" {
				break loop
			}
		case <-deadline:
			t.Fatalf("timed out waiting for agent_gc; events so far: %+v", events)
		}
	}

	// 1. First event must be agent_spawn with the correct label.
	if len(events) == 0 || events[0].Type != "agent_spawn" {
		t.Fatalf("first event must be agent_spawn, got: %+v", events)
	}
	if events[0].Label != "hello-task" {
		t.Errorf("agent_spawn label = %q, want %q", events[0].Label, "hello-task")
	}
	spawnedID := events[0].ID

	// 2. There must be a conversation from conductor → subagent.
	var hasConvToSub bool
	for _, ev := range events {
		if ev.Type == "conversation" && ev.From == "conductor" && ev.To == spawnedID {
			hasConvToSub = true
			break
		}
	}
	if !hasConvToSub {
		t.Errorf("missing conversation(conductor → %s); events: %+v", spawnedID, events)
	}

	// 3. There must be at least one agent_state(waiting) for the subagent.
	var hasWaiting bool
	for _, ev := range events {
		if ev.Type == "agent_state" && ev.ID == spawnedID && ev.State == "waiting" {
			hasWaiting = true
			break
		}
	}
	if !hasWaiting {
		t.Errorf("missing agent_state(waiting) for %s; events: %+v", spawnedID, events)
	}

	// 4. Last event must be agent_gc with reason "completed".
	last := events[len(events)-1]
	if last.Type != "agent_gc" || last.ID != spawnedID || last.Reason != "completed" {
		t.Errorf("last event must be agent_gc(completed), got: %+v", last)
	}

	// 5. Snapshot must be empty after GC (agent removed from live map).
	if snap := b.Snapshot(); len(snap) != 0 {
		t.Errorf("snapshot must be empty after agent_gc, got: %v", snap)
	}

	// 6. Callback must be called. The callback fires in the same goroutine
	// as ReportGC (after the deferred unlock), so we poll briefly.
	for i := 0; i < 100; i++ {
		if atomic.LoadInt32(&callbackCalled) == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if atomic.LoadInt32(&callbackCalled) != 1 {
		t.Error("completion callback was not called after agent_gc")
	}
}

// TestSubagentManager_Spawn_SnapshotLiveDuringExecution verifies that the
// Broadcaster snapshot contains the agent between agent_spawn and agent_gc.
// Because Publish() updates the agent map before dispatching to subscribers,
// the snapshot is guaranteed to be non-empty as soon as agent_spawn is
// received on the channel.
func TestSubagentManager_Spawn_SnapshotLiveDuringExecution(t *testing.T) {
	b := orch.NewBroadcaster()
	sub := b.Subscribe()
	defer b.Unsubscribe(sub)

	provider := &MockLLMProvider{}
	mgr := NewSubagentManager(provider, "test-model", "/tmp/test", nil, b)

	_, err := mgr.Spawn(
		context.Background(),
		"any task", "live-test", "", "cli", "direct",
		nil,
	)
	if err != nil {
		t.Fatalf("Spawn() error: %v", err)
	}

	// Wait for agent_spawn, then immediately check snapshot.
	deadline := time.After(2 * time.Second)
	for {
		select {
		case ev := <-sub.Ch:
			if ev.Type == "agent_spawn" {
				snap := b.Snapshot()
				if len(snap) == 0 {
					t.Error("snapshot must contain the spawned agent after agent_spawn event")
				}
				return // test complete; background goroutine drains safely
			}
		case <-deadline:
			t.Fatal("timed out waiting for agent_spawn event")
		}
	}
}

// TestSubagentManager_Spawn_CancelledDuringExecution verifies that when the
// context is cancelled while a subagent's LLM call is in progress, the
// Broadcaster receives agent_gc with reason="cancelled" and the agent is
// removed from the snapshot.
//
// Synchronisation:
//  1. blockingProvider.ready is closed when Chat() is entered (goroutine is
//     now blocked inside the LLM call).
//  2. Only then is the context cancelled, so there is no race between spawn
//     and cancellation.
func TestSubagentManager_Spawn_CancelledDuringExecution(t *testing.T) {
	b := orch.NewBroadcaster()
	sub := b.Subscribe()
	defer b.Unsubscribe(sub)

	bp := newBlockingProvider()
	mgr := NewSubagentManager(bp, "test-model", "/tmp/test", nil, b)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := mgr.Spawn(ctx, "long task", "cancel-me", "", "cli", "direct", nil)
	if err != nil {
		t.Fatalf("Spawn() error: %v", err)
	}

	// Wait until the subagent goroutine is inside Chat (blocking on ctx).
	select {
	case <-bp.ready:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for blockingProvider to enter Chat")
	}

	// Now cancel — the LLM call unblocks with ctx.Err().
	cancel()

	// Collect events until agent_gc.
	var events []orch.Event
	deadline := time.After(3 * time.Second)
loop:
	for {
		select {
		case ev := <-sub.Ch:
			events = append(events, ev)
			if ev.Type == "agent_gc" {
				break loop
			}
		case <-deadline:
			t.Fatalf("timed out waiting for agent_gc; events so far: %+v", events)
		}
	}

	// Locate agent_gc and verify reason = "cancelled".
	var gcEv orch.Event
	for _, ev := range events {
		if ev.Type == "agent_gc" {
			gcEv = ev
			break
		}
	}
	if gcEv.Reason != "cancelled" {
		t.Errorf("agent_gc reason = %q, want %q; events: %+v", gcEv.Reason, "cancelled", events)
	}

	// Snapshot must be empty after the GC event.
	if snap := b.Snapshot(); len(snap) != 0 {
		t.Errorf("snapshot must be empty after agent_gc(cancelled), got: %v", snap)
	}
}
