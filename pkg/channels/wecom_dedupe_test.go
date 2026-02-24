package channels

import (
	"sync"
	"testing"
)

func TestMarkMessageProcessed_DuplicateDetection(t *testing.T) {
	var mu sync.RWMutex
	processed := make(map[string]bool)

	if ok := markMessageProcessed(&mu, &processed, "msg-1", 1000); !ok {
		t.Fatalf("first message should be accepted")
	}

	if ok := markMessageProcessed(&mu, &processed, "msg-1", 1000); ok {
		t.Fatalf("duplicate message should be rejected")
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

	// Inserting second unique message exceeds maxEntries and should reset map.
	if ok := markMessageProcessed(&mu, &processed, "msg-2", 1); !ok {
		t.Fatalf("second unique message should be accepted")
	}
	if len(processed) != 0 {
		t.Fatalf("expected map to be reset after rotation, got size %d", len(processed))
	}
	if processed["msg-2"] {
		t.Fatalf("expected current message marker to be cleared after rotation")
	}
}
