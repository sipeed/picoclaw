package skills

import (
	"slices"
	"strings"
	"sync"
	"time"
)

// SearchCache provides lightweight caching for search results.
// It uses trigram-based similarity to match similar queries to cached results,
// avoiding redundant API calls. Thread-safe for concurrent access.
type SearchCache struct {
	mu         sync.RWMutex
	entries    map[string]*cacheEntry
	maxEntries int
	ttl        time.Duration
	// LRU linked list implementation
	head *cacheEntry // Oldest entry
	tail *cacheEntry // Newest entry
}

type cacheEntry struct {
	query     string
	trigrams  []uint32
	results   []SearchResult
	createdAt time.Time
	// LRU linked list pointers
	prev *cacheEntry
	next *cacheEntry
}

// similarityThreshold is the minimum trigram Jaccard similarity for a cache hit.
const similarityThreshold = 0.7

// NewSearchCache creates a new search cache.
// maxEntries is the maximum number of cached queries (excess evicts LRU).
// ttl is how long each entry lives before expiration.
func NewSearchCache(maxEntries int, ttl time.Duration) *SearchCache {
	if maxEntries <= 0 {
		maxEntries = 50
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &SearchCache{
		entries:    make(map[string]*cacheEntry),
		maxEntries: maxEntries,
		ttl:        ttl,
		head:        nil,
		tail:        nil,
	}
}

// Get looks up results for a query. Returns cached results and true if found
// (either exact or similar match above threshold). Returns nil, false on miss.
func (sc *SearchCache) Get(query string) ([]SearchResult, bool) {
	normalized := normalizeQuery(query)
	if normalized == "" {
		return nil, false
	}

	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Exact match first.
	if entry, ok := sc.entries[normalized]; ok {
		if time.Since(entry.createdAt) < sc.ttl {
			sc.moveToTailLocked(entry)
			return copyResults(entry.results), true
		}
		// Remove expired entry
		sc.removeEntryLocked(entry)
		delete(sc.entries, normalized)
	}

	// Similarity match.
	queryTrigrams := buildTrigrams(normalized)
	var bestEntry *cacheEntry
	var bestSim float64

	for key, entry := range sc.entries {
		if time.Since(entry.createdAt) >= sc.ttl {
			// Remove expired entry
			// Note: In Go, deleting from a map during range iteration is safe
			// The iteration continues over the remaining elements
			sc.removeEntryLocked(entry)
			delete(sc.entries, key)
			continue // Skip expired.
		}
		sim := jaccardSimilarity(queryTrigrams, entry.trigrams)
		if sim > bestSim {
			bestSim = sim
			bestEntry = entry
		}
	}

	if bestSim >= similarityThreshold && bestEntry != nil {
		sc.moveToTailLocked(bestEntry)
		return copyResults(bestEntry.results), true
	}

	return nil, false
}

// Put stores results for a query. Evicts the oldest entry if at capacity.
func (sc *SearchCache) Put(query string, results []SearchResult) {
	normalized := normalizeQuery(query)
	if normalized == "" {
		return
	}

	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Evict expired entries first.
	sc.evictExpiredLocked()

	// If already exists, update.
	if entry, ok := sc.entries[normalized]; ok {
		// Update entry
		entry.trigrams = buildTrigrams(normalized)
		entry.results = copyResults(results)
		entry.createdAt = time.Now()
		// Move to end of LRU order.
		sc.moveToTailLocked(entry)
		return
	}

	// Evict LRU if at capacity.
	for len(sc.entries) >= sc.maxEntries && sc.head != nil {
		oldest := sc.head
		sc.removeEntryLocked(oldest)
		delete(sc.entries, oldest.query)
	}

	// Insert new entry.
	newEntry := &cacheEntry{
		query:     normalized,
		trigrams:  buildTrigrams(normalized),
		results:   copyResults(results),
		createdAt: time.Now(),
		prev:      nil,
		next:      nil,
	}
	sc.entries[normalized] = newEntry
	sc.addToTailLocked(newEntry)
}

// Len returns the number of entries (for testing).
func (sc *SearchCache) Len() int {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return len(sc.entries)
}

// --- internal ---

func (sc *SearchCache) evictExpiredLocked() {
	now := time.Now()
	current := sc.head
	for current != nil {
		next := current.next
		if now.Sub(current.createdAt) >= sc.ttl {
			sc.removeEntryLocked(current)
			delete(sc.entries, current.query)
		}
		current = next
	}
}

// addToTailLocked adds an entry to the tail of the LRU list
func (sc *SearchCache) addToTailLocked(entry *cacheEntry) {
	if sc.tail == nil {
		// List is empty
		sc.head = entry
		sc.tail = entry
	} else {
		// Add to tail
		sc.tail.next = entry
		entry.prev = sc.tail
		sc.tail = entry
	}
}

// removeEntryLocked removes an entry from the LRU list
func (sc *SearchCache) removeEntryLocked(entry *cacheEntry) {
	if entry.prev != nil {
		entry.prev.next = entry.next
	} else {
		// Entry is head
		sc.head = entry.next
	}

	if entry.next != nil {
		entry.next.prev = entry.prev
	} else {
		// Entry is tail
		sc.tail = entry.prev
	}

	// Clear pointers
	entry.prev = nil
	entry.next = nil
}

// moveToTailLocked moves an entry to the tail of the LRU list
func (sc *SearchCache) moveToTailLocked(entry *cacheEntry) {
	// Remove from current position
	sc.removeEntryLocked(entry)
	// Add to tail
	sc.addToTailLocked(entry)
}

func normalizeQuery(q string) string {
	return strings.ToLower(strings.TrimSpace(q))
}

// buildTrigrams generates hash of trigrams from a string.
// Example: "hello" → {"hel", "ell", "llo"}
// "hel" -> 0x0068656c -> 4 bytes; compared to 16 bytes of a string
func buildTrigrams(s string) []uint32 {
	if len(s) < 3 {
		return nil
	}

	trigrams := make([]uint32, 0, len(s)-2)
	for i := 0; i <= len(s)-3; i++ {
		trigrams = append(trigrams, uint32(s[i])<<16|uint32(s[i+1])<<8|uint32(s[i+2]))
	}

	// Sort and Deduplication
	slices.Sort(trigrams)
	n := 1
	for i := 1; i < len(trigrams); i++ {
		if trigrams[i] != trigrams[i-1] {
			trigrams[n] = trigrams[i]
			n++
		}
	}

	return trigrams[:n]
}

// jaccardSimilarity computes |A ∩ B| / |A ∪ B|.
func jaccardSimilarity(a, b []uint32) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1
	}
	i, j := 0, 0
	intersection := 0

	for i < len(a) && j < len(b) {
		if a[i] == b[j] {
			intersection++
			i++
			j++
		} else if a[i] < b[j] {
			i++
		} else {
			j++
		}
	}

	union := len(a) + len(b) - intersection
	return float64(intersection) / float64(union)
}

func copyResults(results []SearchResult) []SearchResult {
	if results == nil {
		return nil
	}
	cp := make([]SearchResult, len(results))
	copy(cp, results)
	return cp
}
