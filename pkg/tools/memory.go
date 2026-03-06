// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/memory"
)

// MemoryStoreTool stores a new memory entry in the brain database.
type MemoryStoreTool struct {
	db *memory.DB
}

func NewMemoryStoreTool(db *memory.DB) *MemoryStoreTool {
	return &MemoryStoreTool{db: db}
}

func (t *MemoryStoreTool) Name() string { return "memory_store" }

func (t *MemoryStoreTool) Description() string {
	return "Store a memory entry in the second brain database. Use this to remember facts, notes, preferences, or anything worth keeping for future sessions."
}

func (t *MemoryStoreTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{
				"type":        "string",
				"description": "The content to remember. Be specific and detailed.",
			},
			"category": map[string]any{
				"type":        "string",
				"description": "Category for organization (e.g. 'preference', 'fact', 'task', 'note', 'contact'). Defaults to 'general'.",
			},
			"tags": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Optional tags for easier retrieval (e.g. ['work', 'important']).",
			},
		},
		"required": []string{"content"},
	}
}

func (t *MemoryStoreTool) Execute(_ context.Context, args map[string]any) *ToolResult {
	content, _ := args["content"].(string)
	category, _ := args["category"].(string)

	var tags []string
	if raw, ok := args["tags"].([]any); ok {
		for _, v := range raw {
			if s, ok := v.(string); ok && s != "" {
				tags = append(tags, s)
			}
		}
	}

	e, err := t.db.Store(content, category, tags)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Failed to store memory: %s", err)).WithError(err)
	}

	return SilentResult(fmt.Sprintf("Memory stored (id=%s, category=%s)", e.ID, e.Category))
}

// MemorySearchTool searches stored memories using full-text search.
type MemorySearchTool struct {
	db *memory.DB
}

func NewMemorySearchTool(db *memory.DB) *MemorySearchTool {
	return &MemorySearchTool{db: db}
}

func (t *MemorySearchTool) Name() string { return "memory_search" }

func (t *MemorySearchTool) Description() string {
	return "Search stored memories using full-text search across content and tags. Use this to recall relevant information."
}

func (t *MemorySearchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Search query. Matches against memory content and tags.",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Maximum number of results to return (default: 10).",
			},
		},
		"required": []string{"query"},
	}
}

func (t *MemorySearchTool) Execute(_ context.Context, args map[string]any) *ToolResult {
	query, _ := args["query"].(string)
	limit := 10
	if v, ok := args["limit"].(float64); ok && v > 0 {
		limit = int(v)
	}

	results := t.db.Search(query, limit)
	if len(results) == 0 {
		return SilentResult(fmt.Sprintf("No memories found for query: %q", query))
	}

	return SilentResult(formatEntries(results))
}

// MemoryListTool lists recent memories, optionally filtered by category.
type MemoryListTool struct {
	db *memory.DB
}

func NewMemoryListTool(db *memory.DB) *MemoryListTool {
	return &MemoryListTool{db: db}
}

func (t *MemoryListTool) Name() string { return "memory_list" }

func (t *MemoryListTool) Description() string {
	return "List recent memories from the second brain database, optionally filtered by category."
}

func (t *MemoryListTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"category": map[string]any{
				"type":        "string",
				"description": "Filter by category (e.g. 'preference', 'fact', 'task'). Omit to list all.",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Maximum number of results to return (default: 20).",
			},
		},
	}
}

func (t *MemoryListTool) Execute(_ context.Context, args map[string]any) *ToolResult {
	category, _ := args["category"].(string)
	limit := 20
	if v, ok := args["limit"].(float64); ok && v > 0 {
		limit = int(v)
	}

	entries := t.db.List(category, limit)
	total := t.db.Count()

	if len(entries) == 0 {
		msg := fmt.Sprintf("No memories found (total in brain: %d).", total)
		if category != "" {
			msg = fmt.Sprintf("No memories in category %q (total in brain: %d).", category, total)
		}
		return SilentResult(msg)
	}

	header := fmt.Sprintf("Showing %d of %d memories", len(entries), total)
	if category != "" {
		header = fmt.Sprintf("Showing %d memories in category %q", len(entries), category)
	}
	return SilentResult(header + ":\n\n" + formatEntries(entries))
}

// MemoryDeleteTool deletes a memory entry by ID.
type MemoryDeleteTool struct {
	db *memory.DB
}

func NewMemoryDeleteTool(db *memory.DB) *MemoryDeleteTool {
	return &MemoryDeleteTool{db: db}
}

func (t *MemoryDeleteTool) Name() string { return "memory_delete" }

func (t *MemoryDeleteTool) Description() string {
	return "Delete a specific memory entry by its ID. Use memory_list or memory_search first to find the ID."
}

func (t *MemoryDeleteTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{
				"type":        "string",
				"description": "The ID of the memory entry to delete.",
			},
		},
		"required": []string{"id"},
	}
}

func (t *MemoryDeleteTool) Execute(_ context.Context, args map[string]any) *ToolResult {
	id, _ := args["id"].(string)
	if id == "" {
		return ErrorResult("id is required")
	}

	if err := t.db.Delete(id); err != nil {
		return ErrorResult(fmt.Sprintf("Failed to delete memory: %s", err)).WithError(err)
	}
	return SilentResult(fmt.Sprintf("Memory %s deleted.", id))
}

// formatEntries formats a slice of memory entries for display.
func formatEntries(entries []*memory.Entry) string {
	var sb strings.Builder
	for _, e := range entries {
		sb.WriteString(fmt.Sprintf("[%s] (%s)", e.ID, e.Category))
		if len(e.Tags) > 0 {
			sb.WriteString(fmt.Sprintf(" #%s", strings.Join(e.Tags, " #")))
		}
		sb.WriteString(fmt.Sprintf(" — %s\n", e.CreatedAt.Format("2006-01-02")))
		sb.WriteString(fmt.Sprintf("  %s\n\n", e.Content))
	}
	return strings.TrimRight(sb.String(), "\n")
}
