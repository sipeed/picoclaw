package channels

import "sync"

const wecomMaxProcessedMessages = 1000

// MessageDeduplicator provides thread-safe message deduplication using a circular queue (ring buffer)
// combined with a hash map. This ensures fast O(1) lookups while naturally evicting the oldest
// messages without causing "amnesia cliffs" when the limit is reached.
type MessageDeduplicator struct {
	mu   sync.Mutex
	msgs map[string]bool
	ring []string
	idx  int
	max  int
}

// NewMessageDeduplicator creates a new deduplicator with the specified capacity.
func NewMessageDeduplicator(maxEntries int) *MessageDeduplicator {
	if maxEntries <= 0 {
		maxEntries = wecomMaxProcessedMessages
	}
	return &MessageDeduplicator{
		msgs: make(map[string]bool, maxEntries),
		ring: make([]string, maxEntries),
		max:  maxEntries,
	}
}

// MarkMessageProcessed marks msgID as processed and returns false for duplicates.
func (d *MessageDeduplicator) MarkMessageProcessed(msgID string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	// 1. Check for duplicate
	if d.msgs[msgID] {
		return false
	}

	// 2. Evict the oldest message at our current ring position (if any)
	oldestID := d.ring[d.idx]
	if oldestID != "" {
		delete(d.msgs, oldestID)
	}

	// 3. Store the new message
	d.msgs[msgID] = true
	d.ring[d.idx] = msgID

	// 4. Advance the circle queue index
	d.idx = (d.idx + 1) % d.max

	return true
}
