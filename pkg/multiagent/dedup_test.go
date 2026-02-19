package multiagent

import (
	"sync"
	"testing"
	"time"
)

func TestDedupCache_FirstCallNotDuplicate(t *testing.T) {
	dc := NewDedupCache(5 * time.Minute)
	defer dc.Stop()

	if dc.Check("key-1") {
		t.Error("first call should not be a duplicate")
	}
}

func TestDedupCache_SecondCallIsDuplicate(t *testing.T) {
	dc := NewDedupCache(5 * time.Minute)
	defer dc.Stop()

	dc.Check("key-1")
	if !dc.Check("key-1") {
		t.Error("second call with same key should be a duplicate")
	}
}

func TestDedupCache_DifferentKeysNotDuplicate(t *testing.T) {
	dc := NewDedupCache(5 * time.Minute)
	defer dc.Stop()

	dc.Check("key-1")
	if dc.Check("key-2") {
		t.Error("different key should not be a duplicate")
	}
}

func TestDedupCache_ExpiredEntryNotDuplicate(t *testing.T) {
	dc := NewDedupCache(50 * time.Millisecond) // very short TTL
	defer dc.Stop()

	dc.Check("key-1")
	time.Sleep(100 * time.Millisecond) // wait for expiry

	if dc.Check("key-1") {
		t.Error("expired entry should not be treated as duplicate")
	}
}

func TestDedupCache_CheckWithResult(t *testing.T) {
	dc := NewDedupCache(5 * time.Minute)
	defer dc.Stop()

	// First call: not a duplicate
	result, isDup := dc.CheckWithResult("key-1")
	if isDup || result != "" {
		t.Error("first call should not be a duplicate")
	}

	// Set result
	dc.SetResult("key-1", "cached-result")

	// Second call: duplicate with cached result
	result, isDup = dc.CheckWithResult("key-1")
	if !isDup {
		t.Error("second call should be a duplicate")
	}
	if result != "cached-result" {
		t.Errorf("expected cached-result, got %q", result)
	}
}

func TestDedupCache_Size(t *testing.T) {
	dc := NewDedupCache(5 * time.Minute)
	defer dc.Stop()

	if dc.Size() != 0 {
		t.Error("expected size 0")
	}

	dc.Check("key-1")
	dc.Check("key-2")
	dc.Check("key-3")

	if dc.Size() != 3 {
		t.Errorf("expected size 3, got %d", dc.Size())
	}
}

func TestDedupCache_ConcurrentAccess(t *testing.T) {
	dc := NewDedupCache(5 * time.Minute)
	defer dc.Stop()

	var wg sync.WaitGroup
	duplicates := 0
	var mu sync.Mutex

	// 100 goroutines all trying the same key
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if dc.Check("same-key") {
				mu.Lock()
				duplicates++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	// Exactly 99 should be duplicates (first one registers)
	if duplicates != 99 {
		t.Errorf("expected 99 duplicates, got %d", duplicates)
	}
}

func TestDedupCache_Sweep(t *testing.T) {
	dc := NewDedupCache(50 * time.Millisecond)
	defer dc.Stop()

	dc.Check("key-1")
	dc.Check("key-2")
	dc.Check("key-3")

	if dc.Size() != 3 {
		t.Fatalf("expected 3, got %d", dc.Size())
	}

	// Wait for entries to expire
	time.Sleep(100 * time.Millisecond)

	// Manually trigger sweep
	dc.sweep()

	if dc.Size() != 0 {
		t.Errorf("expected 0 after sweep, got %d", dc.Size())
	}
}

func TestBuildSpawnKey_Deterministic(t *testing.T) {
	k1 := BuildSpawnKey("main", "worker", "do X")
	k2 := BuildSpawnKey("main", "worker", "do X")
	k3 := BuildSpawnKey("main", "worker", "do Y")

	if k1 != k2 {
		t.Error("same inputs should produce same key")
	}
	if k1 == k3 {
		t.Error("different task should produce different key")
	}
}

func TestBuildAnnounceKey_Format(t *testing.T) {
	key := BuildAnnounceKey("child-session", "run-123")
	expected := "announce:v1:child-session:run-123"
	if key != expected {
		t.Errorf("expected %q, got %q", expected, key)
	}
}
