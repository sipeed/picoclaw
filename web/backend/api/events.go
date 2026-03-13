package api

import (
	"encoding/json"
	"sync"
)

// GatewayEvent represents a state change event for the gateway process.
type GatewayEvent struct {
	Status string `json:"gateway_status"` // "running", "starting", "stopped", "error"
	PID    int    `json:"pid,omitempty"`
}

// EventBroadcaster manages SSE client subscriptions and broadcasts events.
type EventBroadcaster struct {
	mu      sync.RWMutex
	clients map[chan string]struct{}
}

// NewEventBroadcaster creates a new broadcaster.
func NewEventBroadcaster() *EventBroadcaster {
	return &EventBroadcaster{
		clients: make(map[chan string]struct{}),
	}
}

// Subscribe adds a new listener channel and returns it.
// The caller must call Unsubscribe when done.
func (b *EventBroadcaster) Subscribe() chan string {
	ch := make(chan string, 8)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

// Unsubscribe removes a listener channel and closes it.
func (b *EventBroadcaster) Unsubscribe(ch chan string) {
	b.mu.Lock()
	delete(b.clients, ch)
	b.mu.Unlock()
	close(ch)
}

// Broadcast sends a GatewayEvent to all connected SSE clients.
func (b *EventBroadcaster) Broadcast(event GatewayEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch := range b.clients {
		// Non-blocking send; drop event if client is slow
		select {
		case ch <- string(data):
		default:
		}
	}
}
