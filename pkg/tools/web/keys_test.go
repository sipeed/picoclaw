package web

import (
	"testing"
)

func TestAPIKeyPool(t *testing.T) {
	pool := NewAPIKeyPool([]string{"key1", "key2", "key3"})
	if len(pool.keys) != 3 {
		t.Fatalf("expected 3 keys, got %d", len(pool.keys))
	}
	if pool.keys[0] != "key1" || pool.keys[1] != "key2" || pool.keys[2] != "key3" {
		t.Fatalf("unexpected keys: %v", pool.keys)
	}

	// Test Iterator: each iterator should cover all keys exactly once
	iter := pool.NewIterator()
	expected := []string{"key1", "key2", "key3"}
	for i, want := range expected {
		k, ok := iter.Next()
		if !ok {
			t.Fatalf("iter.Next() returned false at step %d", i)
		}
		if k != want {
			t.Errorf("step %d: expected %s, got %s", i, want, k)
		}
	}
	// Should be exhausted
	if _, ok := iter.Next(); ok {
		t.Errorf("expected iterator exhausted after all keys")
	}

	// Second iterator starts at next position (load balancing)
	iter2 := pool.NewIterator()
	k, ok := iter2.Next()
	if !ok {
		t.Fatal("iter2.Next() returned false")
	}
	if k != "key2" {
		t.Errorf("expected key2 (round-robin), got %s", k)
	}

	// Empty pool
	emptyPool := NewAPIKeyPool([]string{})
	emptyIter := emptyPool.NewIterator()
	if _, ok := emptyIter.Next(); ok {
		t.Errorf("expected false for empty pool")
	}

	// Single key pool
	singlePool := NewAPIKeyPool([]string{"single"})
	singleIter := singlePool.NewIterator()
	if k, ok := singleIter.Next(); !ok || k != "single" {
		t.Errorf("expected single, got %s (ok=%v)", k, ok)
	}
	if _, ok := singleIter.Next(); ok {
		t.Errorf("expected exhausted after single key")
	}
}
