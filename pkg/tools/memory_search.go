package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/vecstore"
)

// MemorySearchTool searches memory using vector similarity.
type MemorySearchTool struct {
	embedder   vecstore.Embedder
	store      *vecstore.VectorStore
	maxResults int
}

// NewMemorySearchTool creates a memory search tool.
func NewMemorySearchTool(embedder vecstore.Embedder, store *vecstore.VectorStore, maxResults int) *MemorySearchTool {
	if maxResults <= 0 {
		maxResults = 5
	}
	return &MemorySearchTool{
		embedder:   embedder,
		store:      store,
		maxResults: maxResults,
	}
}

func (t *MemorySearchTool) Name() string { return "memory_search" }

func (t *MemorySearchTool) Description() string {
	return "Search long-term memory for relevant information using semantic similarity. Use this to find specific memories, notes, or facts."
}

func (t *MemorySearchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "The search query to find relevant memories",
			},
		},
		"required": []string{"query"},
	}
}

func (t *MemorySearchTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	query, _ := args["query"].(string)
	if query == "" {
		return "", fmt.Errorf("query is required")
	}

	// Embed the query
	embeddings, err := t.embedder.Embed(ctx, []string{query})
	if err != nil {
		return "", fmt.Errorf("embed query: %w", err)
	}
	if len(embeddings) == 0 || len(embeddings[0]) == 0 {
		return "No results found.", nil
	}

	// Search
	results := t.store.Search(embeddings[0], t.maxResults)
	if len(results) == 0 {
		return "No relevant memories found.", nil
	}

	// Format results
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d relevant memories:\n\n", len(results)))
	for i, r := range results {
		snippet := r.Text
		if len(snippet) > 700 {
			snippet = snippet[:700] + "..."
		}
		sb.WriteString(fmt.Sprintf("--- Result %d (score: %.2f, source: %s) ---\n%s\n\n", i+1, r.Score, r.Source, snippet))
	}
	return sb.String(), nil
}
