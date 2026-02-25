package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/tools"
)

// MemorySearchTool performs semantic lookup over persisted memory files.
type MemorySearchTool struct {
	memory          *MemoryStore
	defaultTopK     int
	defaultMinScore float64
}

func NewMemorySearchTool(memory *MemoryStore, defaultTopK int, defaultMinScore float64) *MemorySearchTool {
	if defaultTopK <= 0 {
		defaultTopK = defaultMemoryVectorTopK
	}
	if defaultMinScore < 0 || defaultMinScore >= 1 {
		defaultMinScore = defaultMemoryVectorMinScore
	}
	return &MemorySearchTool{
		memory:          memory,
		defaultTopK:     defaultTopK,
		defaultMinScore: defaultMinScore,
	}
}

func (t *MemorySearchTool) Name() string {
	return "memory_search"
}

func (t *MemorySearchTool) Description() string {
	return "Semantically search MEMORY.md and recent daily notes for relevant facts"
}

func (t *MemorySearchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Natural-language query to search semantic memory",
			},
			"top_k": map[string]any{
				"type":        "integer",
				"description": "Maximum number of hits to return (default from agent settings)",
			},
			"min_score": map[string]any{
				"type":        "number",
				"description": "Minimum cosine similarity in [0,1), lower means broader recall",
			},
		},
		"required": []string{"query"},
	}
}

func (t *MemorySearchTool) Execute(ctx context.Context, args map[string]any) *tools.ToolResult {
	_ = ctx

	if t.memory == nil {
		return tools.ErrorResult("memory store unavailable")
	}

	query, ok := args["query"].(string)
	if !ok || strings.TrimSpace(query) == "" {
		return tools.ErrorResult("query is required")
	}

	topK := t.defaultTopK
	if raw, ok := args["top_k"]; ok {
		switch v := raw.(type) {
		case int:
			if v > 0 {
				topK = v
			}
		case int64:
			if v > 0 {
				topK = int(v)
			}
		case float64:
			if int(v) > 0 {
				topK = int(v)
			}
		}
	}

	minScore := t.defaultMinScore
	if raw, ok := args["min_score"]; ok {
		if v, ok := raw.(float64); ok && v >= 0 && v < 1 {
			minScore = v
		}
	}

	hits, err := t.memory.SearchRelevant(query, topK, minScore)
	if err != nil {
		return tools.ErrorResult(fmt.Sprintf("memory search failed: %v", err)).WithError(err)
	}
	if len(hits) == 0 {
		return tools.SilentResult("No relevant memory hits found.")
	}

	var sb strings.Builder
	sb.WriteString("Memory search hits:\n")
	for _, hit := range hits {
		sb.WriteString(fmt.Sprintf("- (score=%.2f, source=%s) %s\n", hit.Score, hit.Source, hit.Text))
	}

	return tools.SilentResult(strings.TrimSpace(sb.String()))
}

// MemoryGetTool returns a specific memory item by its source citation.
type MemoryGetTool struct {
	memory *MemoryStore
}

func NewMemoryGetTool(memory *MemoryStore) *MemoryGetTool {
	return &MemoryGetTool{memory: memory}
}

func (t *MemoryGetTool) Name() string {
	return "memory_get"
}

func (t *MemoryGetTool) Description() string {
	return "Retrieve one memory entry by source citation returned from memory_search"
}

func (t *MemoryGetTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"source": map[string]any{
				"type":        "string",
				"description": "Citation source like MEMORY.md#Long-term Facts",
			},
		},
		"required": []string{"source"},
	}
}

func (t *MemoryGetTool) Execute(ctx context.Context, args map[string]any) *tools.ToolResult {
	_ = ctx

	if t.memory == nil {
		return tools.ErrorResult("memory store unavailable")
	}

	source, ok := args["source"].(string)
	if !ok || strings.TrimSpace(source) == "" {
		return tools.ErrorResult("source is required")
	}

	hit, found, err := t.memory.GetBySource(source)
	if err != nil {
		return tools.ErrorResult(fmt.Sprintf("memory get failed: %v", err)).WithError(err)
	}
	if !found {
		return tools.SilentResult("Memory source not found.")
	}

	return tools.SilentResult(fmt.Sprintf(
		"Memory entry:\n- source=%s\n- content=%s",
		hit.Source,
		hit.Text,
	))
}
