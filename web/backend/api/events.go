package api

import (
	"encoding/json"
	"sync"
)

// GatewayEvent represents a state change event for the gateway process.
type GatewayEvent struct {
	Status             string `json:"gateway_status"` // "running", "starting", "restarting", "stopped", "error"
	PID                int    `json:"pid,omitempty"`
	BootDefaultModel   string `json:"boot_default_model,omitempty"`
	ConfigDefaultModel string `json:"config_default_model,omitempty"`
	RestartRequired    bool   `json:"gateway_restart_required,omitempty"`
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
	defer b.mu.Unlock()

	// Check if the channel is still registered before closing
	if _, exists := b.clients[ch]; exists {
		delete(b.clients, ch)
		close(ch)
	}
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

// Shutdown closes all subscriber channels, notifying all SSE clients to disconnect.
// This should be called when the server is shutting down.
func (b *EventBroadcaster) Shutdown() {
	// Close all channels to notify listeners
	for ch := range b.clients {
		b.Unsubscribe(ch)
	}
	// Clear the map
	b.clients = make(map[chan string]struct{})
}
