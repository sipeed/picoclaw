// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package multi

import (
	"sync"
)

// SharedContext implements a thread-safe blackboard pattern where multiple
// agents can read from and write to a common session context.
//
// It provides:
//   - Key-value storage for arbitrary data sharing between agents
//   - An append-only event log for agent activity tracking
//   - Thread-safe access via read-write mutex
//
// This is intentionally simple and in-memory. Future iterations may add
// persistence, TTL, or namespace isolation.
type SharedContext struct {
	mu     sync.RWMutex
	data   map[string]interface{}
	events []Event
}

// Event records an action taken by an agent within the shared context.
// Events are append-only and provide an audit trail of agent activity.
type Event struct {
	// Agent is the name of the agent that produced this event.
	Agent string

	// Type categorizes the event (e.g., "handoff", "result", "error").
	Type string

	// Content is the event payload.
	Content string
}

// NewSharedContext creates a new empty SharedContext.
func NewSharedContext() *SharedContext {
	return &SharedContext{
		data:   make(map[string]interface{}),
		events: make([]Event, 0),
	}
}

// Set stores a value in the shared context under the given key.
// Overwrites any existing value for the same key.
func (sc *SharedContext) Set(key string, value interface{}) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.data[key] = value
}

// Get retrieves a value from the shared context.
// Returns the value and true if found, nil and false otherwise.
func (sc *SharedContext) Get(key string) (interface{}, bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	v, ok := sc.data[key]
	return v, ok
}

// GetString retrieves a string value from the shared context.
// Returns empty string if the key doesn't exist or isn't a string.
func (sc *SharedContext) GetString(key string) string {
	v, ok := sc.Get(key)
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// Delete removes a key from the shared context.
func (sc *SharedContext) Delete(key string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	delete(sc.data, key)
}

// Keys returns all keys currently stored in the shared context.
func (sc *SharedContext) Keys() []string {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	keys := make([]string, 0, len(sc.data))
	for k := range sc.data {
		keys = append(keys, k)
	}
	return keys
}

// AddEvent appends an event to the shared context's event log.
func (sc *SharedContext) AddEvent(agent, eventType, content string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.events = append(sc.events, Event{
		Agent:   agent,
		Type:    eventType,
		Content: content,
	})
}

// Events returns a copy of all events in the shared context.
func (sc *SharedContext) Events() []Event {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	cp := make([]Event, len(sc.events))
	copy(cp, sc.events)
	return cp
}

// EventsByAgent returns all events produced by the given agent.
func (sc *SharedContext) EventsByAgent(agent string) []Event {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	var filtered []Event
	for _, e := range sc.events {
		if e.Agent == agent {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

// EventsByType returns all events of the given type.
func (sc *SharedContext) EventsByType(eventType string) []Event {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	var filtered []Event
	for _, e := range sc.events {
		if e.Type == eventType {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

// Snapshot returns a shallow copy of the entire data map.
// Useful for debugging or serialization.
func (sc *SharedContext) Snapshot() map[string]interface{} {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	snap := make(map[string]interface{}, len(sc.data))
	for k, v := range sc.data {
		snap[k] = v
	}
	return snap
}
