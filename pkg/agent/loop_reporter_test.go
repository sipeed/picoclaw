package agent

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/orch"
)

// makeOrchTestLoop creates a minimal AgentLoop with a temp workspace and

// a real Broadcaster wired as the reporter.

// Returns the loop, the broadcaster, and a cleanup function.

func makeOrchTestLoop(t *testing.T) (*AgentLoop, *orch.Broadcaster) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "agent-orch-test-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}

	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace: tmpDir,

				Model: "test-model",

				MaxTokens: 512,

				MaxToolIterations: 5,
			},
		},
	}

	al := NewAgentLoop(cfg, bus.NewMessageBus(), &mockProvider{})

	b := orch.NewBroadcaster()

	al.SetOrchReporter(b)

	return al, b
}

// collectOrchEvents drains the subscriber channel until an agent_gc event

// arrives or the deadline is exceeded.

func collectOrchEvents(t *testing.T, ch <-chan orch.Event, timeout time.Duration) []orch.Event {
	t.Helper()

	var events []orch.Event

	deadline := time.After(timeout)

	for {
		select {
		case ev := <-ch:

			events = append(events, ev)

			if ev.Type == "agent_gc" {
				return events
			}

		case <-deadline:

			t.Fatalf("timed out waiting for agent_gc; events so far: %+v", events)
		}
	}
}

// TestAgentLoop_ProcessDirect_EmitsSpawnWaitingGC verifies that a main

// session processed via ProcessDirect emits the full lifecycle:

//

//	agent_spawn(sessionKey) → agent_state(waiting) → agent_gc(completed)

//

// and that the Broadcaster snapshot is empty after the call returns.

func TestAgentLoop_ProcessDirect_EmitsSpawnWaitingGC(t *testing.T) {
	al, b := makeOrchTestLoop(t)

	sub := b.Subscribe()

	defer b.Unsubscribe(sub)

	const sessionKey = "orch-test-session"

	_, err := al.ProcessDirect(context.Background(), "hello", sessionKey)
	if err != nil {
		t.Fatalf("ProcessDirect: %v", err)
	}

	events := collectOrchEvents(t, sub.Ch, 5*time.Second)

	// First event: agent_spawn with correct ID.

	if events[0].Type != "agent_spawn" || events[0].ID != sessionKey {
		t.Errorf("first event must be agent_spawn(%s), got: %+v", sessionKey, events[0])
	}

	// At least one agent_state(waiting) for this session.

	var hasWaiting bool

	for _, ev := range events {
		if ev.Type == "agent_state" && ev.ID == sessionKey && ev.State == "waiting" {
			hasWaiting = true

			break
		}
	}

	if !hasWaiting {
		t.Errorf("missing agent_state(waiting) for %s; events: %+v", sessionKey, events)
	}

	// Last event: agent_gc(completed) for this session.

	last := events[len(events)-1]

	if last.Type != "agent_gc" || last.ID != sessionKey || last.Reason != "completed" {
		t.Errorf("last event must be agent_gc(completed,%s), got: %+v", sessionKey, last)
	}

	// Snapshot must be empty — session removed on GC.

	if snap := b.Snapshot(); len(snap) != 0 {
		t.Errorf("snapshot must be empty after GC, got: %v", snap)
	}
}

// TestAgentLoop_ProcessHeartbeat_EmitsSpawnAndGC verifies that heartbeat

// sessions appear on canvas with sessionKey = "heartbeat".

func TestAgentLoop_ProcessHeartbeat_EmitsSpawnAndGC(t *testing.T) {
	al, b := makeOrchTestLoop(t)

	sub := b.Subscribe()

	defer b.Unsubscribe(sub)

	_, err := al.ProcessHeartbeat(context.Background(), "check system", "heartbeat-chan", "none")
	if err != nil {
		t.Fatalf("ProcessHeartbeat: %v", err)
	}

	events := collectOrchEvents(t, sub.Ch, 5*time.Second)

	// ProcessHeartbeat always uses sessionKey = "heartbeat".

	const want = "heartbeat"

	if events[0].Type != "agent_spawn" || events[0].ID != want {
		t.Errorf("first event must be agent_spawn(%s), got: %+v", want, events[0])
	}

	last := events[len(events)-1]

	if last.Type != "agent_gc" || last.ID != want || last.Reason != "completed" {
		t.Errorf("last event must be agent_gc(completed,%s), got: %+v", want, last)
	}
}
