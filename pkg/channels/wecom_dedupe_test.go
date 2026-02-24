package channels

import (
	"sync"
	"testing"
)

func TestMarkMessageProcessed_DuplicateDetection(t *testing.T) {
	var mu sync.RWMutex
	processed := make(map[string]bool)

	if ok := markMessageProcessed(&mu, &processed, "msg-1", wecomMaxProcessedMessages); !ok {
		t.Fatalf("first message should be accepted")
	}

	if ok := markMessageProcessed(&mu, &processed, "msg-1", wecomMaxProcessedMessages); ok {
		t.Fatalf("duplicate message should be rejected")
	}
}

func TestMarkMessageProcessed_ConcurrentSameMessage(t *testing.T) {
	var mu sync.RWMutex
	processed := make(map[string]bool)

	const goroutines = 64
	var wg sync.WaitGroup
	wg.Add(goroutines)

	results := make(chan bool, goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			results <- markMessageProcessed(&mu, &processed, "msg-concurrent", wecomMaxProcessedMessages)
		}()
	}

	wg.Wait()
	close(results)

	successes := 0
	for ok := range results {
		if ok {
			successes++
		}
	}

	if successes != 1 {
		t.Fatalf("expected exactly 1 successful mark, got %d", successes)
	}
}

func TestMarkMessageProcessed_RotationClearsMapAtBoundary(t *testing.T) {
	var mu sync.RWMutex
	processed := make(map[string]bool)

	if ok := markMessageProcessed(&mu, &processed, "msg-1", 1); !ok {
		t.Fatalf("first message should be accepted")
	}
	if len(processed) != 1 {
		t.Fatalf("expected map size 1 after first insert, got %d", len(processed))
	}

	// Inserting second unique message exceeds maxEntries and should reset map, but keep the new message.
	if ok := markMessageProcessed(&mu, &processed, "msg-2", 1); !ok {
		t.Fatalf("second unique message should be accepted")
	}
	if len(processed) != 1 {
		t.Fatalf("expected map to retain current message after rotation, got size %d", len(processed))
	}
	if !processed["msg-2"] {
		t.Fatalf("expected current message marker to be retained after rotation")
	}

	// Because msg-2 was retained, an immediate duplicate should be rejected.
	if ok := markMessageProcessed(&mu, &processed, "msg-2", 1); ok {
		t.Fatalf("duplicate message immediately after rotation should be rejected")
	}
}
