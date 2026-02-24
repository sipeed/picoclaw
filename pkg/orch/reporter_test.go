package orch

import "testing"

// Compile-time: Broadcaster must satisfy AgentReporter.
var _ AgentReporter = (*Broadcaster)(nil)

// TestNoop_AllMethods_NoPanic verifies that orch.Noop can be called for all
// four methods without panic. This is the nil-safe baseline for disabled
// orchestration mode.
func TestNoop_AllMethods_NoPanic(t *testing.T) {
	Noop.ReportSpawn("id", "label", "task")
	Noop.ReportStateChange("id", "waiting", "")
	Noop.ReportStateChange("id", "toolcall", "bash")
	Noop.ReportConversation("conductor", "sub-1", "do something")
	Noop.ReportGC("id", "completed")
}

// TestBroadcaster_ReportSpawn_MapsToAgentSpawnEvent verifies that ReportSpawn
// publishes an Event with Type="agent_spawn" and the correct ID/Label/Task
// fields, and that the agent appears in the Snapshot immediately.
func TestBroadcaster_ReportSpawn_MapsToAgentSpawnEvent(t *testing.T) {
	b := NewBroadcaster()
	sub := b.Subscribe()
	defer b.Unsubscribe(sub)

	b.ReportSpawn("agent-1", "scout", "find all TODOs")

	ev := <-sub.Ch
	if ev.Type != "agent_spawn" {
		t.Fatalf("want agent_spawn, got %q", ev.Type)
	}
	if ev.ID != "agent-1" || ev.Label != "scout" || ev.Task != "find all TODOs" {
		t.Fatalf("field mismatch: %+v", ev)
	}
	snap := b.Snapshot()
	if len(snap) != 1 || snap[0].ID != "agent-1" || snap[0].Label != "scout" {
		t.Fatalf("snapshot not updated correctly: %v", snap)
	}
}

// TestBroadcaster_ReportStateChange_MapsToAgentStateEvent verifies that
// ReportStateChange publishes agent_state and updates the live snapshot.
func TestBroadcaster_ReportStateChange_MapsToAgentStateEvent(t *testing.T) {
	b := NewBroadcaster()
	sub := b.Subscribe()
	defer b.Unsubscribe(sub)

	b.ReportSpawn("agent-1", "coder", "implement it")
	<-sub.Ch // consume spawn

	b.ReportStateChange("agent-1", "toolcall", "bash")
	ev := <-sub.Ch
	if ev.Type != "agent_state" || ev.State != "toolcall" || ev.Tool != "bash" {
		t.Fatalf("unexpected event: %+v", ev)
	}
	snap := b.Snapshot()
	if snap[0].State != "toolcall" || snap[0].Tool != "bash" {
		t.Fatalf("snapshot state not updated: %v", snap)
	}
}

// TestBroadcaster_ReportConversation_MapsToConversationEvent verifies that
// ReportConversation publishes a conversation event with correct From/To/Text
// fields and does NOT modify the agent snapshot (conversation is not a state
// change of any agent).
func TestBroadcaster_ReportConversation_MapsToConversationEvent(t *testing.T) {
	b := NewBroadcaster()
	sub := b.Subscribe()
	defer b.Unsubscribe(sub)

	b.ReportConversation("conductor", "sub-1", "please do the task")

	ev := <-sub.Ch
	if ev.Type != "conversation" || ev.From != "conductor" || ev.To != "sub-1" || ev.Text != "please do the task" {
		t.Fatalf("unexpected event: %+v", ev)
	}
	if len(b.Snapshot()) != 0 {
		t.Fatal("conversation event must not modify agent snapshot")
	}
}

// TestBroadcaster_ReportGC_RemovesAgentFromSnapshot verifies that ReportGC
// publishes agent_gc with the correct Reason and removes the agent from the
// live snapshot so new WS connections no longer see it.
func TestBroadcaster_ReportGC_RemovesAgentFromSnapshot(t *testing.T) {
	b := NewBroadcaster()
	sub := b.Subscribe()
	defer b.Unsubscribe(sub)

	b.ReportSpawn("agent-1", "scout", "task")
	<-sub.Ch // consume spawn

	b.ReportGC("agent-1", "completed")
	ev := <-sub.Ch
	if ev.Type != "agent_gc" || ev.ID != "agent-1" || ev.Reason != "completed" {
		t.Fatalf("unexpected event: %+v", ev)
	}
	if len(b.Snapshot()) != 0 {
		t.Fatal("agent must be removed from snapshot after ReportGC")
	}
}
