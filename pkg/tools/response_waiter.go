package tools

import "sync"

// DeviceResponseWaiter is a package-level shared instance used to synchronize
// tool_request/tool_response pairs between the Android tool and WebSocket channel.
var DeviceResponseWaiter = NewResponseWaiter()

// ResponseWaiter manages pending request/response synchronization.
// Each request registers a channel by ID; the response is delivered when it arrives.
type ResponseWaiter struct {
	pending map[string]chan string
	mu      sync.Mutex
}

func NewResponseWaiter() *ResponseWaiter {
	return &ResponseWaiter{
		pending: make(map[string]chan string),
	}
}

// Register creates a buffered channel for the given request ID and returns it.
// The caller should select on this channel with a timeout.
func (w *ResponseWaiter) Register(id string) chan string {
	w.mu.Lock()
	defer w.mu.Unlock()
	ch := make(chan string, 1)
	w.pending[id] = ch
	return ch
}

// Deliver sends the response content to the waiting channel for the given ID.
// If no waiter is registered for the ID, the delivery is silently dropped.
func (w *ResponseWaiter) Deliver(id, content string) {
	w.mu.Lock()
	ch, ok := w.pending[id]
	if ok {
		delete(w.pending, id)
	}
	w.mu.Unlock()
	if ok {
		ch <- content
	}
}

// Cleanup removes the pending channel for the given ID without delivering.
// Used on timeout to prevent memory leaks.
func (w *ResponseWaiter) Cleanup(id string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.pending, id)
}
