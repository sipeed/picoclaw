package tools

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/vecstore"
)

// mockEmbedder returns a fixed embedding for any input.
type mockEmbedder struct {
	embedding []float32
	err       error
}

func (m *mockEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	if m.err != nil {
		return nil, m.err
	}
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i] = m.embedding
	}
	return result, nil
}

func TestMemorySearchExecute(t *testing.T) {
	store := vecstore.NewVectorStore("")
	now := time.Now()
	store.Upsert([]vecstore.Chunk{
		{ID: "a", Text: "The user prefers dark mode", Source: "memory/MEMORY.md", Embedding: []float32{1, 0, 0}, UpdatedAt: now},
		{ID: "b", Text: "Meeting notes from Monday", Source: "memory/202601/20260112.md", Embedding: []float32{0, 1, 0}, UpdatedAt: now},
		{ID: "c", Text: "User timezone is PST", Source: "memory/MEMORY.md", Embedding: []float32{0.9, 0.1, 0}, UpdatedAt: now},
	})

	embedder := &mockEmbedder{embedding: []float32{1, 0, 0}}
	tool := NewMemorySearchTool(embedder, store, 2)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"query": "user preferences",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "dark mode") {
		t.Error("expected top result to contain 'dark mode'")
	}
	if !strings.Contains(result, "timezone") {
		t.Error("expected second result to contain 'timezone'")
	}
	if strings.Contains(result, "Meeting notes") {
		t.Error("should not contain third result (maxResults=2)")
	}
	if !strings.Contains(result, "score:") {
		t.Error("result should include scores")
	}
	if !strings.Contains(result, "memory/MEMORY.md") {
		t.Error("result should include source path")
	}
}

func TestMemorySearchEmptyQuery(t *testing.T) {
	store := vecstore.NewVectorStore("")
	embedder := &mockEmbedder{embedding: []float32{1, 0}}
	tool := NewMemorySearchTool(embedder, store, 5)

	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"query": "",
	})
	if err == nil {
		t.Error("expected error for empty query")
	}
}

func TestMemorySearchEmbedError(t *testing.T) {
	store := vecstore.NewVectorStore("")
	embedder := &mockEmbedder{err: fmt.Errorf("API unavailable")}
	tool := NewMemorySearchTool(embedder, store, 5)

	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"query": "test",
	})
	if err == nil {
		t.Error("expected error when embedder fails")
	}
	if !strings.Contains(err.Error(), "embed query") {
		t.Errorf("error should wrap embed failure, got: %v", err)
	}
}

func TestMemorySearchNoResults(t *testing.T) {
	store := vecstore.NewVectorStore("") // empty store
	embedder := &mockEmbedder{embedding: []float32{1, 0}}
	tool := NewMemorySearchTool(embedder, store, 5)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"query": "anything",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "No relevant memories") {
		t.Errorf("expected no-results message, got: %s", result)
	}
}
