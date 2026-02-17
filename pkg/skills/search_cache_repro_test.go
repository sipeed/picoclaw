package skills

import (
	"testing"
	"time"
)

func TestSearchCache_LRU_Behavior(t *testing.T) {
	// Capacity 3
	// Capacity 3
	cache := NewSearchCache(3, time.Hour)

	// Fill cache: query-A, query-B, query-C
	// Use longer strings to ensure trigrams are generated and avoid false positive similarity
	cache.Put("query-A", []SearchResult{{Slug: "A"}})
	cache.Put("query-B", []SearchResult{{Slug: "B"}})
	cache.Put("query-C", []SearchResult{{Slug: "C"}})

	// Access query-A (should make it most recently used)
	// Current behavior: query-A remains at front (oldest) if Get doesn't update order
	if _, found := cache.Get("query-A"); !found {
		t.Fatal("query-A should be in cache")
	}

	// Add query-D. Should evict query-A if FIFO (current bug), or query-B if LRU (desired).
	cache.Put("query-D", []SearchResult{{Slug: "D"}})

	// Check if query-A is still there
	if _, found := cache.Get("query-A"); !found {
		t.Fatalf("query-A was evicted! valid LRU should have kept query-A and evicted query-B.")
	}

	// Check if query-B is evicted (if A was kept, B should be gone)
	if _, found := cache.Get("query-B"); found {
		t.Fatal("query-B should have been evicted")
	}
}
