// ABOUTME: Recall tool for semantic search over stored memories.
// ABOUTME: Queries the vector store with natural language and returns ranked results.
package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/memory"
)

// RecallTool performs semantic search over stored memories.
type RecallTool struct {
	store memory.Store
}

func NewRecallTool(store memory.Store) *RecallTool {
	return &RecallTool{store: store}
}

func (t *RecallTool) Name() string { return "recall" }

func (t *RecallTool) Description() string {
	return "Search stored memories using natural language. Returns the most relevant memories ranked by similarity. Use this to recall facts, preferences, decisions, or context from previous conversations."
}

func (t *RecallTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Natural language search query describing what you want to recall.",
			},
			"top_k": map[string]any{
				"type":        "integer",
				"description": "Number of results to return (default 5, max 20).",
			},
		},
		"required": []any{"query"},
	}
}

func (t *RecallTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	if !t.store.IsAvailable() {
		return ErrorResult("Semantic memory is not available. Ollama may not be running.")
	}

	query, _ := args["query"].(string)
	if strings.TrimSpace(query) == "" {
		return ErrorResult("query is required")
	}

	topK := 5
	if k, ok := args["top_k"].(float64); ok && k > 0 {
		topK = int(k)
	}

	results, err := t.store.Recall(ctx, query, topK)
	if err != nil {
		return ErrorResult("Failed to search memories: " + err.Error())
	}

	if len(results) == 0 {
		return SilentResult("No memories found matching: " + query)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Found %d memories:\n\n", len(results))
	for i, r := range results {
		fmt.Fprintf(&sb, "%d. [%.0f%% match] [%s] %s\n",
			i+1, r.Similarity*100, r.Category, r.Content)
		if len(r.Tags) > 0 {
			fmt.Fprintf(&sb, "   Tags: %s\n", strings.Join(r.Tags, ", "))
		}
		if !r.Timestamp.IsZero() {
			fmt.Fprintf(&sb, "   Stored: %s\n", r.Timestamp.Format("2006-01-02 15:04"))
		}
	}

	return SilentResult(sb.String())
}
