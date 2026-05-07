package main

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/sipeed/picoclaw/pkg/seahorse"
)

func TestIngestSeahorseIdempotent(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Minimal test data
	samples := []LocomoSample{
		{
			SampleID: "test-1",
			Conversation: map[string]json.RawMessage{
				"session_1": json.RawMessage(`[
					{"speaker":"A","dia_id":"D1:1","text":"hello world this is a test message"},
					{"speaker":"B","dia_id":"D1:2","text":"another message for testing purposes"}
				]`),
			},
		},
	}

	// First ingestion
	result1, err := IngestSeahorse(ctx, samples, dbPath)
	if err != nil {
		t.Fatalf("first ingest failed: %v", err)
	}
	convCount1 := len(result1.ConvMap)
	result1.Engine.Close()

	// Second ingestion on same DB — should reuse existing data
	result2, err := IngestSeahorse(ctx, samples, dbPath)
	if err != nil {
		t.Fatalf("second ingest failed: %v", err)
	}
	defer result2.Engine.Close()

	// ConvMap should have same number of entries (no duplicates)
	if len(result2.ConvMap) != convCount1 {
		t.Errorf("second ingest convMap has %d entries, want %d (same as first)",
			len(result2.ConvMap), convCount1)
	}

	// Verify conversation IDs are the same (reused, not new ones)
	for id, cid1 := range result1.ConvMap {
		cid2, ok := result2.ConvMap[id]
		if !ok {
			t.Errorf("sample %s missing from second ConvMap", id)
			continue
		}
		if cid2 != cid1 {
			t.Errorf("sample %s: second ingest got convID %d, want %d (reused)", id, cid2, cid1)
		}
	}

	// Verify no duplicate messages by counting
	store := result2.Engine.GetRetrieval().Store()
	for _, convID := range result2.ConvMap {
		msgs, err := store.SearchMessages(ctx, seahorse.SearchInput{
			Pattern:        "test",
			ConversationID: convID,
			Limit:          100,
		})
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		// Should find exactly 1 message containing "test" (the first turn)
		if len(msgs) > 2 {
			t.Errorf("found %d messages for 'test' in conv %d, expected ≤2 (no duplicates)", len(msgs), convID)
		}
	}
}
