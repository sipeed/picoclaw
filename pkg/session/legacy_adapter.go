package session

import (
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/providers"
)

// LegacyAdapter wraps a SessionStore and exposes the same public API as

// SessionManager so that all existing call sites (loop.go, etc.) work

// without modification.

type LegacyAdapter struct {
	store SessionStore

	mu sync.RWMutex

	cache map[string]*sessionCache

	dirtyMu sync.Mutex

	dirtyKeys map[string]bool

	done chan struct{}
}

type sessionCache struct {
	messages []providers.Message

	summary string

	created time.Time

	updated time.Time

	dirty bool

	replaced bool // SetHistory/TruncateHistory set this; Save does full rewrite

	stored int // number of messages already persisted in the store
}

// NewLegacyAdapter creates a LegacyAdapter backed by the given store.

func NewLegacyAdapter(store SessionStore) *LegacyAdapter {
	la := &LegacyAdapter{
		store: store,

		cache: make(map[string]*sessionCache),

		dirtyKeys: make(map[string]bool),

		done: make(chan struct{}),
	}

	go la.flushLoop()

	return la
}

// getOrLoad returns the cache entry for key, loading from the store if needed.

// Caller must hold la.mu (write lock).

func (la *LegacyAdapter) getOrLoad(key string) *sessionCache {
	if c, ok := la.cache[key]; ok {
		return c
	}

	// Try loading from store

	info, err := la.store.Get(key)

	if err != nil || info == nil {
		// Create in store

		_ = la.store.Create(key, nil)

		now := time.Now()

		c := &sessionCache{
			messages: []providers.Message{},

			created: now,

			updated: now,
		}

		la.cache[key] = c

		return c
	}

	// Load all turns and reconstruct messages

	turns, _ := la.store.Turns(key, 0)

	var msgs []providers.Message

	for _, t := range turns {
		msgs = append(msgs, t.Messages...)
	}

	if msgs == nil {
		msgs = []providers.Message{}
	}

	c := &sessionCache{
		messages: msgs,

		summary: info.Summary,

		created: info.CreatedAt,

		updated: info.UpdatedAt,

		stored: len(msgs),
	}

	la.cache[key] = c

	return c
}

// GetOrCreate returns a Session-compatible object for the given key.

// Creates the session if it doesn't exist.

func (la *LegacyAdapter) GetOrCreate(key string) *Session {
	la.mu.Lock()

	defer la.mu.Unlock()

	c := la.getOrLoad(key)

	return &Session{
		Key: key,

		Messages: c.messages,

		Summary: c.summary,

		Created: c.created,

		Updated: c.updated,
	}
}

// AddMessage adds a simple message to the session.

func (la *LegacyAdapter) AddMessage(sessionKey, role, content string) {
	la.AddFullMessage(sessionKey, providers.Message{
		Role: role,

		Content: content,
	})
}

// AddFullMessage adds a complete message with tool calls to the session.

func (la *LegacyAdapter) AddFullMessage(sessionKey string, msg providers.Message) {
	la.mu.Lock()

	defer la.mu.Unlock()

	c := la.getOrLoad(sessionKey)

	c.messages = append(c.messages, msg)

	c.updated = time.Now()

	c.dirty = true
}

// GetHistory returns a defensive copy of the session messages.

func (la *LegacyAdapter) GetHistory(key string) []providers.Message {
	la.mu.RLock()

	c, ok := la.cache[key]

	la.mu.RUnlock()

	if !ok {
		// Try lazy load

		la.mu.Lock()

		c, ok = la.cache[key]

		if !ok {
			// Check if it exists in the store

			info, _ := la.store.Get(key)

			if info == nil {
				la.mu.Unlock()

				return []providers.Message{}
			}

			c = la.getOrLoad(key)
		}

		la.mu.Unlock()
	}

	la.mu.RLock()

	defer la.mu.RUnlock()

	history := make([]providers.Message, len(c.messages))

	copy(history, c.messages)

	return history
}

// SetHistory replaces the session's message history entirely.

func (la *LegacyAdapter) SetHistory(key string, history []providers.Message) {
	la.mu.Lock()

	defer la.mu.Unlock()

	c, ok := la.cache[key]

	if !ok {
		return
	}

	msgs := make([]providers.Message, len(history))

	copy(msgs, history)

	c.messages = msgs

	c.updated = time.Now()

	c.replaced = true

	c.dirty = true
}

// GetSummary returns the session summary.

func (la *LegacyAdapter) GetSummary(key string) string {
	la.mu.RLock()

	c, ok := la.cache[key]

	la.mu.RUnlock()

	if !ok {
		la.mu.Lock()

		c, ok = la.cache[key]

		if !ok {
			info, _ := la.store.Get(key)

			if info == nil {
				la.mu.Unlock()

				return ""
			}

			c = la.getOrLoad(key)
		}

		la.mu.Unlock()
	}

	la.mu.RLock()

	defer la.mu.RUnlock()

	return c.summary
}

// SetSummary updates the session summary in cache and store.

func (la *LegacyAdapter) SetSummary(key string, summary string) {
	la.mu.Lock()

	defer la.mu.Unlock()

	c, ok := la.cache[key]

	if !ok {
		return
	}

	c.summary = summary

	c.updated = time.Now()

	_ = la.store.SetSummary(key, summary)
}

// TruncateHistory keeps only the last n messages.

func (la *LegacyAdapter) TruncateHistory(key string, keepLast int) {
	la.mu.Lock()

	defer la.mu.Unlock()

	c, ok := la.cache[key]

	if !ok {
		return
	}

	if keepLast <= 0 {
		c.messages = []providers.Message{}

		c.updated = time.Now()

		c.replaced = true

		c.dirty = true

		return
	}

	if len(c.messages) <= keepLast {
		return
	}

	c.messages = c.messages[len(c.messages)-keepLast:]

	c.updated = time.Now()

	c.replaced = true

	c.dirty = true
}

// MarkDirty marks a session key for deferred persistence.

func (la *LegacyAdapter) MarkDirty(key string) {
	la.dirtyMu.Lock()

	la.dirtyKeys[key] = true

	la.dirtyMu.Unlock()
}

// FlushDirty writes all dirty sessions to the store.

func (la *LegacyAdapter) FlushDirty() {
	la.dirtyMu.Lock()

	keys := make([]string, 0, len(la.dirtyKeys))

	for k := range la.dirtyKeys {
		keys = append(keys, k)
	}

	la.dirtyKeys = make(map[string]bool)

	la.dirtyMu.Unlock()

	for _, k := range keys {
		la.Save(k)
	}
}

// Save persists the session to the store.

func (la *LegacyAdapter) Save(key string) error {
	la.mu.RLock()

	c, ok := la.cache[key]

	if !ok {
		la.mu.RUnlock()

		return nil
	}

	// Snapshot under read lock

	replaced := c.replaced

	stored := c.stored

	msgs := make([]providers.Message, len(c.messages))

	copy(msgs, c.messages)

	la.mu.RUnlock()

	if replaced {
		// Full rewrite: compact all existing turns then write the whole history

		if err := la.store.Compact(key, 1<<31, ""); err != nil {
			return err
		}

		if len(msgs) > 0 {
			turn := &Turn{
				SessionKey: key,

				Kind: TurnNormal,

				Messages: msgs,
			}

			if err := la.store.Append(key, turn); err != nil {
				return err
			}
		}

		la.mu.Lock()

		if cc, ok := la.cache[key]; ok {
			cc.replaced = false

			cc.stored = len(msgs)

			cc.dirty = false
		}

		la.mu.Unlock()
	} else {
		// Incremental: only append new messages

		newMsgs := msgs[stored:]

		if len(newMsgs) > 0 {
			turn := &Turn{
				SessionKey: key,

				Kind: TurnNormal,

				Messages: newMsgs,
			}

			if err := la.store.Append(key, turn); err != nil {
				return err
			}
		}

		la.mu.Lock()

		if cc, ok := la.cache[key]; ok {
			cc.stored = len(msgs)

			cc.dirty = false
		}

		la.mu.Unlock()
	}

	return nil
}

// DefaultPruneTTL is the default time-to-live for session pruning.

const DefaultPruneTTL = 7 * 24 * time.Hour

// CompactOldTurns flushes pending writes, then compacts SQLite turns

// keeping only the last keepLast messages. Sets session summary to the given value.

func (la *LegacyAdapter) CompactOldTurns(key string, keepLast int, summary string) error {
	// 1. Flush pending messages to SQLite

	if err := la.Save(key); err != nil {
		return err
	}

	// 2. Query all turns

	turns, err := la.store.Turns(key, 0)
	if err != nil {
		return err
	}

	// 3. Count total messages, find cut point

	totalMsgs := 0

	for _, t := range turns {
		totalMsgs += len(t.Messages)
	}

	if keepLast >= totalMsgs {
		// Nothing to compact, just update summary

		if err := la.store.SetSummary(key, summary); err != nil {
			return err
		}

		la.mu.Lock()

		if c, ok := la.cache[key]; ok {
			c.summary = summary
		}

		la.mu.Unlock()

		return nil
	}

	dropCount := totalMsgs - keepLast

	accumulated := 0

	cutSeq := 0

	for _, t := range turns {
		accumulated += len(t.Messages)

		if accumulated <= dropCount {
			cutSeq = t.Seq
		} else {
			break
		}
	}

	if cutSeq == 0 {
		if err := la.store.SetSummary(key, summary); err != nil {
			return err
		}

		la.mu.Lock()

		if c, ok := la.cache[key]; ok {
			c.summary = summary
		}

		la.mu.Unlock()

		return nil
	}

	// 4. Compact in SQLite

	if err := la.store.Compact(key, cutSeq, summary); err != nil {
		return err
	}

	// 5. Update in-memory cache

	la.mu.Lock()

	defer la.mu.Unlock()

	if c, ok := la.cache[key]; ok {
		if keepLast < len(c.messages) {
			c.messages = c.messages[len(c.messages)-keepLast:]
		}

		c.stored = len(c.messages)

		c.replaced = false

		c.dirty = false

		c.summary = summary
	}

	return nil
}

// Store returns the underlying SessionStore for direct DAG operations.

func (la *LegacyAdapter) Store() SessionStore {
	return la.store
}

// Graph returns a SessionGraph backed by the underlying store.

func (la *LegacyAdapter) Graph() *SessionGraph {
	return NewSessionGraph(la.store)
}

// AdvanceStored increments the stored counter for a session by delta,

// preventing the flush loop from re-persisting messages already written

// directly to the store (e.g. TurnReport).

func (la *LegacyAdapter) AdvanceStored(key string, delta int) {
	la.mu.Lock()

	defer la.mu.Unlock()

	if c, ok := la.cache[key]; ok {
		c.stored += delta
	}
}

// Close stops the background flush loop and persists all dirty sessions.

func (la *LegacyAdapter) Close() {
	select {
	case <-la.done:

		return // already closed

	default:
	}

	close(la.done)

	la.FlushDirty()

	la.store.Close()
}

func (la *LegacyAdapter) flushLoop() {
	flushTicker := time.NewTicker(5 * time.Minute)

	pruneTicker := time.NewTicker(6 * time.Hour)

	defer flushTicker.Stop()

	defer pruneTicker.Stop()

	for {
		select {
		case <-flushTicker.C:

			la.FlushDirty()

		case <-pruneTicker.C:

			_, _ = la.store.Prune(DefaultPruneTTL)

		case <-la.done:

			return
		}
	}
}
