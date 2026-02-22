// ABOUTME: Tests for the SemanticStore vector memory implementation.
// ABOUTME: Uses a deterministic mock embedding function to test without Ollama.
package memory

import (
	"context"
	"os"
	"testing"
)

// mockEmbeddingFunc returns a deterministic embedding based on text length.
// This gives different texts different (but reproducible) embeddings,
// allowing cosine similarity to work for testing.
func mockEmbeddingFunc(text string) []float32 {
	const dims = 64
	embedding := make([]float32, dims)
	for i, ch := range text {
		embedding[i%dims] += float32(ch) / 1000.0
	}
	// Normalize to unit vector for cosine similarity
	var norm float32
	for _, v := range embedding {
		norm += v * v
	}
	if norm > 0 {
		norm = sqrt32(norm)
		for i := range embedding {
			embedding[i] /= norm
		}
	}
	return embedding
}

func sqrt32(x float32) float32 {
	// Newton's method for float32
	z := x / 2
	for i := 0; i < 10; i++ {
		z = z - (z*z-x)/(2*z)
	}
	return z
}

func newTestStore(t *testing.T) *SemanticStore {
	t.Helper()
	dir := t.TempDir()

	embedFn := func(_ context.Context, text string) ([]float32, error) {
		return mockEmbeddingFunc(text), nil
	}

	store, err := NewSemanticStoreWithEmbedding(dir, embedFn)
	if err != nil {
		t.Fatalf("NewSemanticStoreWithEmbedding: %v", err)
	}
	return store
}

func TestSemanticStore_IsAvailable(t *testing.T) {
	store := newTestStore(t)
	if !store.IsAvailable() {
		t.Error("expected store to be available")
	}
}

func TestSemanticStore_RememberAndRecall(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()

	// Store some memories
	entries := []MemoryEntry{
		{Content: "User prefers dark mode in all applications", Category: "preference", Source: "agent"},
		{Content: "The database password is rotated every 90 days", Category: "fact", Source: "agent"},
		{Content: "We decided to use PostgreSQL instead of MySQL", Category: "decision", Source: "agent"},
	}

	for _, entry := range entries {
		if err := store.Remember(ctx, entry); err != nil {
			t.Fatalf("Remember(%q): %v", entry.Content, err)
		}
	}

	// Recall with a query related to preferences
	results, err := store.Recall(ctx, "What does the user prefer for themes?", 3)
	if err != nil {
		t.Fatalf("Recall: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// All results should have content
	for _, r := range results {
		if r.Content == "" {
			t.Error("result has empty content")
		}
		if r.Similarity <= 0 {
			t.Errorf("expected positive similarity, got %f", r.Similarity)
		}
	}
}

func TestSemanticStore_RecallEmpty(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()

	results, err := store.Recall(ctx, "anything", 5)
	if err != nil {
		t.Fatalf("Recall on empty store: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results from empty store, got %d", len(results))
	}
}

func TestSemanticStore_RememberAutoID(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()

	entry := MemoryEntry{Content: "test content", Category: "fact"}
	if err := store.Remember(ctx, entry); err != nil {
		t.Fatalf("Remember: %v", err)
	}

	results, err := store.Recall(ctx, "test", 1)
	if err != nil {
		t.Fatalf("Recall: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ID == "" {
		t.Error("expected auto-generated ID, got empty")
	}
}

func TestSemanticStore_MetadataPreserved(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()

	entry := MemoryEntry{
		Content:  "Important fact about the project",
		Category: "fact",
		Tags:     []string{"project", "architecture"},
		Source:   "auto-extract",
	}
	if err := store.Remember(ctx, entry); err != nil {
		t.Fatalf("Remember: %v", err)
	}

	results, err := store.Recall(ctx, "project fact", 1)
	if err != nil {
		t.Fatalf("Recall: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.Category != "fact" {
		t.Errorf("category = %q, want %q", r.Category, "fact")
	}
	if r.Source != "auto-extract" {
		t.Errorf("source = %q, want %q", r.Source, "auto-extract")
	}
	if len(r.Tags) != 2 || r.Tags[0] != "project" || r.Tags[1] != "architecture" {
		t.Errorf("tags = %v, want [project architecture]", r.Tags)
	}
	if r.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

func TestSemanticStore_RecallTopKCapped(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()

	// Store 2 entries, ask for 10
	store.Remember(ctx, MemoryEntry{Content: "first memory"})
	store.Remember(ctx, MemoryEntry{Content: "second memory"})

	results, err := store.Recall(ctx, "memory", 10)
	if err != nil {
		t.Fatalf("Recall: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results (capped to collection size), got %d", len(results))
	}
}

func TestSemanticStore_UnavailableStore(t *testing.T) {
	store := &SemanticStore{available: false}

	if store.IsAvailable() {
		t.Error("expected unavailable store")
	}

	err := store.Remember(t.Context(), MemoryEntry{Content: "test"})
	if err == nil {
		t.Error("expected error from unavailable store Remember")
	}

	_, err = store.Recall(t.Context(), "test", 5)
	if err == nil {
		t.Error("expected error from unavailable store Recall")
	}
}

func TestSemanticStore_Persistence(t *testing.T) {
	dir := t.TempDir()
	ctx := t.Context()

	embedFn := func(_ context.Context, text string) ([]float32, error) {
		return mockEmbeddingFunc(text), nil
	}

	// Create store and add memory
	store1, err := NewSemanticStoreWithEmbedding(dir, embedFn)
	if err != nil {
		t.Fatalf("first store: %v", err)
	}
	store1.Remember(ctx, MemoryEntry{Content: "persistent memory test"})

	// Create a second store from the same directory
	store2, err := NewSemanticStoreWithEmbedding(dir, embedFn)
	if err != nil {
		t.Fatalf("second store: %v", err)
	}

	results, err := store2.Recall(ctx, "persistent", 1)
	if err != nil {
		t.Fatalf("Recall from second store: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 persisted result, got %d", len(results))
	}
	if results[0].Content != "persistent memory test" {
		t.Errorf("content = %q, want %q", results[0].Content, "persistent memory test")
	}
}

func TestNewSemanticStore_BadDir(t *testing.T) {
	// Try to create a store in a path that can't exist
	_, err := NewSemanticStoreWithEmbedding("/dev/null/impossible", nil)
	if err == nil {
		t.Error("expected error for impossible path")
	}
}

func TestNewSemanticStore_WithOllamaDefaults(t *testing.T) {
	// This tests the constructor with empty strings (uses defaults).
	// It will fail to connect to Ollama in test env, but should return
	// a store (possibly in degraded state) without panicking.
	dir := t.TempDir()
	defer os.RemoveAll(dir)

	// We can't actually test with Ollama here, just verify no panic
	store, _ := NewSemanticStore(dir, "http://127.0.0.1:1", "fake-model")
	if store == nil {
		t.Error("expected non-nil store even on connection failure")
	}
}
