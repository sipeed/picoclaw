// Package orch provides the orchestration event broadcaster used by the
// subagent system and the Mini App WebSocket UI.
package orch

import (
	"sync"
	"time"
)

// Event is a single orchestration event pushed over WebSocket to the UI.
// type values: "agent_spawn" | "agent_state" | "conversation" | "agent_gc"
type Event struct {
	Type    string `json:"type"`
	ID      string `json:"id,omitempty"`
	Label   string `json:"label,omitempty"`
	Task    string `json:"task,omitempty"`
	State   string `json:"state,omitempty"`  // waiting | toolcall | idle
	Tool    string `json:"tool,omitempty"`   // tool name during toolcall
	From    string `json:"from,omitempty"`
	To      string `json:"to,omitempty"`
	Text    string `json:"text,omitempty"`
	Reason  string `json:"reason,omitempty"` // agent_gc: completed | failed | cancelled
	Created int64  `json:"created,omitempty"`
}

// AgentInfo is the live snapshot of one active agent.
// Kept inside Broadcaster so new WS connections can get current state.
type AgentInfo struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Task    string `json:"task"`
	State   string `json:"state"`
	Tool    string `json:"tool,omitempty"`
	Created int64  `json:"created"`
}

// Subscriber is a single WebSocket client subscription.
type Subscriber struct {
	Ch chan Event
}

// Broadcaster distributes orchestration events to all connected WS clients.
// It also maintains a live agent snapshot for initial-state delivery on connect.
//
// Publish is non-blocking: events are dropped if a subscriber's buffer is full
// (same pattern as pkg/logger).
type Broadcaster struct {
	mu     sync.Mutex
	subs   map[*Subscriber]struct{}
	agents map[string]*AgentInfo // live agents, keyed by task ID
}

func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		subs:   make(map[*Subscriber]struct{}),
		agents: make(map[string]*AgentInfo),
	}
}

func (b *Broadcaster) Subscribe() *Subscriber {
	sub := &Subscriber{Ch: make(chan Event, 32)}
	b.mu.Lock()
	b.subs[sub] = struct{}{}
	b.mu.Unlock()
	return sub
}

func (b *Broadcaster) Unsubscribe(sub *Subscriber) {
	b.mu.Lock()
	delete(b.subs, sub)
	b.mu.Unlock()
}

// Snapshot returns the current set of active agents.
// Called once on new WS connection to send initial state.
func (b *Broadcaster) Snapshot() []AgentInfo {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]AgentInfo, 0, len(b.agents))
	for _, a := range b.agents {
		out = append(out, *a)
	}
	return out
}

// ReportSpawn implements AgentReporter.
func (b *Broadcaster) ReportSpawn(id, label, task string) {
	b.Publish(Event{Type: "agent_spawn", ID: id, Label: label, Task: task})
}

// ReportStateChange implements AgentReporter.
func (b *Broadcaster) ReportStateChange(id, state, tool string) {
	b.Publish(Event{Type: "agent_state", ID: id, State: state, Tool: tool})
}

// ReportConversation implements AgentReporter.
func (b *Broadcaster) ReportConversation(from, to, text string) {
	b.Publish(Event{Type: "conversation", From: from, To: to, Text: text})
}

// ReportGC implements AgentReporter.
func (b *Broadcaster) ReportGC(id, reason string) {
	b.Publish(Event{Type: "agent_gc", ID: id, Reason: reason})
}

// Publish updates internal agent state and fans out to all subscribers.
func (b *Broadcaster) Publish(ev Event) {
	if ev.Created == 0 {
		ev.Created = time.Now().UnixMilli()
	}

	b.mu.Lock()
	switch ev.Type {
	case "agent_spawn":
		b.agents[ev.ID] = &AgentInfo{
			ID:      ev.ID,
			Label:   ev.Label,
			Task:    ev.Task,
			State:   "idle",
			Created: ev.Created,
		}
	case "agent_state":
		if a, ok := b.agents[ev.ID]; ok {
			a.State = ev.State
			a.Tool = ev.Tool
		}
	case "agent_gc":
		delete(b.agents, ev.ID)
	}
	// snapshot subs while holding lock, then release before sending
	subs := make([]*Subscriber, 0, len(b.subs))
	for sub := range b.subs {
		subs = append(subs, sub)
	}
	b.mu.Unlock()

	for _, sub := range subs {
		select {
		case sub.Ch <- ev:
		default: // subscriber slow — drop (non-blocking)
		}
	}
}
