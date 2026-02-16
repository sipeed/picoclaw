package skills

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSearchCacheExactHit(t *testing.T) {
	cache := NewSearchCache(10, 5*time.Minute)

	results := []SearchResult{
		{Slug: "github", Score: 0.9, RegistryName: "clawhub"},
		{Slug: "docker", Score: 0.7, RegistryName: "clawhub"},
	}
	cache.Put("github integration", results)

	got, hit := cache.Get("github integration")
	assert.True(t, hit)
	assert.Len(t, got, 2)
	assert.Equal(t, "github", got[0].Slug)
}

func TestSearchCacheExactHitCaseInsensitive(t *testing.T) {
	cache := NewSearchCache(10, 5*time.Minute)

	results := []SearchResult{{Slug: "github", Score: 0.9}}
	cache.Put("GitHub Integration", results)

	got, hit := cache.Get("github integration")
	assert.True(t, hit)
	assert.Len(t, got, 1)
}

func TestSearchCacheSimilarHit(t *testing.T) {
	cache := NewSearchCache(10, 5*time.Minute)

	results := []SearchResult{{Slug: "github", Score: 0.9}}
	cache.Put("github integration tool", results)

	// "github integration" is very similar to "github integration tool"
	got, hit := cache.Get("github integration")
	assert.True(t, hit)
	assert.Len(t, got, 1)
}

func TestSearchCacheDissimilarMiss(t *testing.T) {
	cache := NewSearchCache(10, 5*time.Minute)

	results := []SearchResult{{Slug: "github", Score: 0.9}}
	cache.Put("github integration", results)

	// Completely unrelated query
	_, hit := cache.Get("database management")
	assert.False(t, hit)
}

func TestSearchCacheTTLExpiration(t *testing.T) {
	cache := NewSearchCache(10, 50*time.Millisecond)

	results := []SearchResult{{Slug: "github", Score: 0.9}}
	cache.Put("github integration", results)

	// Immediately should hit
	_, hit := cache.Get("github integration")
	assert.True(t, hit)

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	_, hit = cache.Get("github integration")
	assert.False(t, hit)
}

func TestSearchCacheLRUEviction(t *testing.T) {
	cache := NewSearchCache(3, 5*time.Minute)

	cache.Put("query-1", []SearchResult{{Slug: "a"}})
	cache.Put("query-2", []SearchResult{{Slug: "b"}})
	cache.Put("query-3", []SearchResult{{Slug: "c"}})

	assert.Equal(t, 3, cache.Len())

	// Adding a 4th should evict query-1 (oldest)
	cache.Put("query-4", []SearchResult{{Slug: "d"}})
	assert.Equal(t, 3, cache.Len())

	_, hit := cache.Get("query-1")
	assert.False(t, hit, "oldest entry should be evicted")

	got, hit := cache.Get("query-4")
	assert.True(t, hit)
	assert.Equal(t, "d", got[0].Slug)
}

func TestSearchCacheEmptyQuery(t *testing.T) {
	cache := NewSearchCache(10, 5*time.Minute)

	_, hit := cache.Get("")
	assert.False(t, hit)

	_, hit = cache.Get("   ")
	assert.False(t, hit)
}

func TestSearchCacheResultsCopied(t *testing.T) {
	cache := NewSearchCache(10, 5*time.Minute)

	original := []SearchResult{{Slug: "github", Score: 0.9}}
	cache.Put("test", original)

	// Mutate original after putting
	original[0].Slug = "mutated"

	got, hit := cache.Get("test")
	assert.True(t, hit)
	assert.Equal(t, "github", got[0].Slug, "cache should hold a copy, not a reference")
}

func TestBuildTrigrams(t *testing.T) {
	trigrams := buildTrigrams("hello")
	assert.Contains(t, trigrams, "hel")
	assert.Contains(t, trigrams, "ell")
	assert.Contains(t, trigrams, "llo")
	assert.Len(t, trigrams, 3)
}

func TestJaccardSimilarity(t *testing.T) {
	a := buildTrigrams("github integration")
	b := buildTrigrams("github integration tool")

	sim := jaccardSimilarity(a, b)
	assert.Greater(t, sim, 0.5, "similar strings should have high sim")

	c := buildTrigrams("completely different query about databases")
	sim2 := jaccardSimilarity(a, c)
	assert.Less(t, sim2, 0.3, "dissimilar strings should have low sim")
}

func TestJaccardSimilarityEdgeCases(t *testing.T) {
	empty := buildTrigrams("")
	nonempty := buildTrigrams("hello")

	assert.Equal(t, 1.0, jaccardSimilarity(empty, empty))
	assert.Equal(t, 0.0, jaccardSimilarity(empty, nonempty))
	assert.Equal(t, 0.0, jaccardSimilarity(nonempty, empty))
}

func TestSearchCacheConcurrency(t *testing.T) {
	cache := NewSearchCache(50, 5*time.Minute)
	done := make(chan struct{})

	// Concurrent writes
	go func() {
		for i := 0; i < 100; i++ {
			cache.Put("query-write-"+string(rune('a'+i%26)), []SearchResult{{Slug: "x"}})
		}
		done <- struct{}{}
	}()

	// Concurrent reads
	go func() {
		for i := 0; i < 100; i++ {
			cache.Get("query-write-a")
		}
		done <- struct{}{}
	}()

	<-done
	<-done
}
