package multiagent

import (
	"crypto/sha256"
	"fmt"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// Idempotency cache defaults.
// Follows Stripe's idempotency key pattern: deterministic keys with TTL-based expiry.
const (
	DefaultDedupTTL     = 5 * time.Minute
	DefaultDedupSweepInterval = 60 * time.Second
)

// DedupEntry tracks a single idempotent operation.
type DedupEntry struct {
	Key       string
	CreatedAt time.Time
	ExpiresAt time.Time
	Result    string // cached result for idempotent replay
}

// DedupCache provides idempotent execution guarantees for spawn and announce
// operations. Uses deterministic keys (like Stripe) with TTL-based expiry
// and periodic sweep (like Google Cloud Tasks dedup).
type DedupCache struct {
	mu      sync.RWMutex
	entries map[string]*DedupEntry
	ttl     time.Duration
	stop    chan struct{}
}

// NewDedupCache creates a dedup cache with the given TTL and starts
// the background sweep goroutine.
func NewDedupCache(ttl time.Duration) *DedupCache {
	if ttl <= 0 {
		ttl = DefaultDedupTTL
	}
	dc := &DedupCache{
		entries: make(map[string]*DedupEntry),
		ttl:     ttl,
		stop:    make(chan struct{}),
	}
	go dc.sweepLoop()
	return dc
}

// Check returns true if the key has already been processed (duplicate).
// If not a duplicate, registers the key and returns false.
// Thread-safe via mutex (simpler than CAS for this throughput level).
func (dc *DedupCache) Check(key string) bool {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	now := time.Now()

	// Check if key exists and hasn't expired.
	if entry, ok := dc.entries[key]; ok {
		if now.Before(entry.ExpiresAt) {
			return true // duplicate
		}
		// Expired — remove and treat as new.
		delete(dc.entries, key)
	}

	// Register new key.
	dc.entries[key] = &DedupEntry{
		Key:       key,
		CreatedAt: now,
		ExpiresAt: now.Add(dc.ttl),
	}
	return false // not a duplicate
}

// CheckWithResult returns the cached result if the key is a duplicate.
// If not a duplicate, registers the key and returns ("", false).
func (dc *DedupCache) CheckWithResult(key string) (string, bool) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	now := time.Now()
	if entry, ok := dc.entries[key]; ok {
		if now.Before(entry.ExpiresAt) {
			return entry.Result, true
		}
		delete(dc.entries, key)
	}

	dc.entries[key] = &DedupEntry{
		Key:       key,
		CreatedAt: now,
		ExpiresAt: now.Add(dc.ttl),
	}
	return "", false
}

// SetResult stores the result for an already-registered key.
func (dc *DedupCache) SetResult(key, result string) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	if entry, ok := dc.entries[key]; ok {
		entry.Result = result
	}
}

// Size returns the current number of entries.
func (dc *DedupCache) Size() int {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	return len(dc.entries)
}

// Stop stops the background sweep goroutine.
func (dc *DedupCache) Stop() {
	close(dc.stop)
}

// sweepLoop periodically removes expired entries (Google Cloud Tasks pattern).
func (dc *DedupCache) sweepLoop() {
	ticker := time.NewTicker(DefaultDedupSweepInterval)
	defer ticker.Stop()

	for {
		select {
		case <-dc.stop:
			return
		case <-ticker.C:
			dc.sweep()
		}
	}
}

func (dc *DedupCache) sweep() {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	now := time.Now()
	expired := 0
	for key, entry := range dc.entries {
		if now.After(entry.ExpiresAt) {
			delete(dc.entries, key)
			expired++
		}
	}
	if expired > 0 {
		logger.DebugCF("dedup", "Sweep completed", map[string]interface{}{
			"expired":   expired,
			"remaining": len(dc.entries),
		})
	}
}

// BuildSpawnKey creates a deterministic dedup key for a spawn request.
// Format: "spawn:v1:{from}:{to}:{task_hash}"
// Same task from the same agent to the same target within TTL = idempotent.
func BuildSpawnKey(fromAgentID, toAgentID, task string) string {
	h := sha256.Sum256([]byte(task))
	return fmt.Sprintf("spawn:v1:%s:%s:%x", fromAgentID, toAgentID, h[:8])
}

// BuildAnnounceKey creates a deterministic dedup key for an announcement.
// Format: "announce:v1:{childSessionKey}:{runID}"
// Prevents duplicate announcements for the same spawn completion.
func BuildAnnounceKey(childSessionKey, runID string) string {
	return fmt.Sprintf("announce:v1:%s:%s", childSessionKey, runID)
}
