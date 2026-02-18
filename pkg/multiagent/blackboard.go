package multiagent

import (
	"encoding/json"
	"sort"
	"sync"
	"time"
)

// BlackboardEntry represents a single entry in the shared context pool.
type BlackboardEntry struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	Author    string    `json:"author"`
	Scope     string    `json:"scope"`
	Timestamp time.Time `json:"timestamp"`
}

// Blackboard is a thread-safe shared context pool for multi-agent collaboration.
// Agents read and write string key-value entries, each tagged with authorship
// and scope metadata.
type Blackboard struct {
	entries map[string]*BlackboardEntry
	mu      sync.RWMutex
}

// NewBlackboard creates an empty Blackboard.
func NewBlackboard() *Blackboard {
	return &Blackboard{
		entries: make(map[string]*BlackboardEntry),
	}
}

// Set writes or overwrites an entry on the blackboard.
func (b *Blackboard) Set(key, value, author string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.entries[key] = &BlackboardEntry{
		Key:       key,
		Value:     value,
		Author:    author,
		Scope:     "shared",
		Timestamp: time.Now(),
	}
}

// Get returns the value for a key, or empty string if not found.
func (b *Blackboard) Get(key string) string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if e, ok := b.entries[key]; ok {
		return e.Value
	}
	return ""
}

// GetEntry returns the full entry for a key, or nil if not found.
func (b *Blackboard) GetEntry(key string) *BlackboardEntry {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if e, ok := b.entries[key]; ok {
		cp := *e
		return &cp
	}
	return nil
}

// Delete removes an entry by key. Returns true if it existed.
func (b *Blackboard) Delete(key string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	_, ok := b.entries[key]
	if ok {
		delete(b.entries, key)
	}
	return ok
}

// List returns all keys sorted alphabetically.
func (b *Blackboard) List() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	keys := make([]string, 0, len(b.entries))
	for k := range b.entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Snapshot returns a string summary of all entries suitable for injection
// into an LLM system prompt.
func (b *Blackboard) Snapshot() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if len(b.entries) == 0 {
		return ""
	}

	keys := make([]string, 0, len(b.entries))
	for k := range b.entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	result := "## Shared Context (Blackboard)\n\n"
	for _, k := range keys {
		e := b.entries[k]
		result += "- **" + k + "** (by " + e.Author + "): " + e.Value + "\n"
	}
	return result
}

// Size returns the number of entries.
func (b *Blackboard) Size() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.entries)
}

// MarshalJSON serializes the blackboard entries to JSON.
func (b *Blackboard) MarshalJSON() ([]byte, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	entries := make([]*BlackboardEntry, 0, len(b.entries))
	for _, e := range b.entries {
		entries = append(entries, e)
	}
	return json.Marshal(entries)
}

// UnmarshalJSON deserializes blackboard entries from JSON.
func (b *Blackboard) UnmarshalJSON(data []byte) error {
	var entries []*BlackboardEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return err
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	b.entries = make(map[string]*BlackboardEntry, len(entries))
	for _, e := range entries {
		b.entries[e.Key] = e
	}
	return nil
}
