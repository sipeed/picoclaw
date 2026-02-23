package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/memory"
)

// MemorySaveTool saves a structured note to the memory vault with frontmatter.
// It auto-generates YAML frontmatter (title, created, updated, tags, aliases),
// preserves the original created date when updating an existing note, and
// rebuilds the vault index after every write.
type MemorySaveTool struct {
	vault *memory.Vault
}

// NewMemorySaveTool creates a new MemorySaveTool backed by the given vault.
func NewMemorySaveTool(vault *memory.Vault) *MemorySaveTool {
	return &MemorySaveTool{vault: vault}
}

func (t *MemorySaveTool) Name() string        { return "memory_save" }
func (t *MemorySaveTool) Description() string {
	return "Save a structured note to the memory vault with frontmatter metadata. " +
		"Notes are stored as markdown files with YAML frontmatter for tags, aliases, and dates."
}

func (t *MemorySaveTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Relative path within memory/ (e.g. 'topics/go-errors.md')",
			},
			"title": map[string]any{
				"type":        "string",
				"description": "Note title",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "Note body content (markdown)",
			},
			"tags": map[string]any{
				"type":        "string",
				"description": "Comma-separated tags (e.g. 'go, patterns, errors')",
			},
			"aliases": map[string]any{
				"type":        "string",
				"description": "Comma-separated aliases for wikilink resolution (optional)",
			},
		},
		"required": []string{"path", "title", "content"},
	}
}

func (t *MemorySaveTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	path, _ := args["path"].(string)
	title, _ := args["title"].(string)
	content, _ := args["content"].(string)

	if path == "" {
		return ErrorResult("path is required")
	}
	if title == "" {
		return ErrorResult("title is required")
	}
	if content == "" {
		return ErrorResult("content is required")
	}

	meta := memory.NoteMeta{
		Title: title,
	}

	if tagsStr, ok := args["tags"].(string); ok && tagsStr != "" {
		for _, tag := range strings.Split(tagsStr, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				meta.Tags = append(meta.Tags, tag)
			}
		}
	}

	if aliasStr, ok := args["aliases"].(string); ok && aliasStr != "" {
		for _, alias := range strings.Split(aliasStr, ",") {
			alias = strings.TrimSpace(alias)
			if alias != "" {
				meta.Aliases = append(meta.Aliases, alias)
			}
		}
	}

	if err := t.vault.SaveNote(path, meta, content); err != nil {
		return ErrorResult(fmt.Sprintf("failed to save note: %v", err))
	}

	tagInfo := ""
	if len(meta.Tags) > 0 {
		tagInfo = fmt.Sprintf(" [tags: %s]", strings.Join(meta.Tags, ", "))
	}
	return SilentResult(fmt.Sprintf("Saved: %s%s", path, tagInfo))
}

// MemorySearchTool searches the memory vault by tags, title, or text content.
// Tags use AND logic (a note must match all specified tags). The text query
// matches case-insensitively against title, tags, and aliases. Results are
// capped at 20 entries and include metadata only — use MemoryRecallTool to
// read full note content.
type MemorySearchTool struct {
	vault *memory.Vault
}

// NewMemorySearchTool creates a new MemorySearchTool backed by the given vault.
func NewMemorySearchTool(vault *memory.Vault) *MemorySearchTool {
	return &MemorySearchTool{vault: vault}
}

func (t *MemorySearchTool) Name() string        { return "memory_search" }
func (t *MemorySearchTool) Description() string {
	return "Search the memory vault by tags, title, or text content. " +
		"Returns a list of matching notes with metadata. Use memory_recall to read full content."
}

func (t *MemorySearchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Search query (matches title, tags, and aliases)",
			},
			"tags": map[string]any{
				"type":        "string",
				"description": "Filter by tags (comma-separated, AND logic)",
			},
		},
	}
}

func (t *MemorySearchTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	query, _ := args["query"].(string)
	tagsStr, _ := args["tags"].(string)

	var tags []string
	if tagsStr != "" {
		for _, tag := range strings.Split(tagsStr, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tags = append(tags, tag)
			}
		}
	}

	results, err := t.vault.Search(query, tags)
	if err != nil {
		return ErrorResult(fmt.Sprintf("search failed: %v", err))
	}

	if len(results) == 0 {
		return NewToolResult("No matching notes found.")
	}

	// Cap results
	if len(results) > 20 {
		results = results[:20]
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d note(s):\n\n", len(results)))
	for _, n := range results {
		tagsDisplay := ""
		if len(n.Tags) > 0 {
			tagsDisplay = fmt.Sprintf(" [%s]", strings.Join(n.Tags, ", "))
		}
		sb.WriteString(fmt.Sprintf("- **%s** (%s)%s", n.Title, n.RelPath, tagsDisplay))
		if n.Updated != "" {
			sb.WriteString(fmt.Sprintf(" — updated %s", n.Updated))
		}
		sb.WriteString("\n")
	}

	return NewToolResult(sb.String())
}

// MemoryRecallTool recalls specific notes from the memory vault by path or topic.
// Unlike MemorySearchTool which returns metadata only, MemoryRecallTool returns
// the full note body with frontmatter stripped. When using topic-based recall,
// it searches for matching notes and reads the top N (default 3).
type MemoryRecallTool struct {
	vault *memory.Vault
}

// NewMemoryRecallTool creates a new MemoryRecallTool backed by the given vault.
func NewMemoryRecallTool(vault *memory.Vault) *MemoryRecallTool {
	return &MemoryRecallTool{vault: vault}
}

func (t *MemoryRecallTool) Name() string        { return "memory_recall" }
func (t *MemoryRecallTool) Description() string {
	return "Recall specific notes from memory vault by path or topic. " +
		"Returns full note content. Use memory_search first to find relevant paths."
}

func (t *MemoryRecallTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Exact path to recall (e.g. 'topics/go-errors.md')",
			},
			"topic": map[string]any{
				"type":        "string",
				"description": "Topic to find relevant notes for (uses search + read)",
			},
			"max_notes": map[string]any{
				"type":        "number",
				"description": "Maximum number of notes to return when using topic (default: 3)",
			},
		},
	}
}

func (t *MemoryRecallTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	path, _ := args["path"].(string)
	topic, _ := args["topic"].(string)

	if path == "" && topic == "" {
		return ErrorResult("either 'path' or 'topic' is required")
	}

	// Direct path recall
	if path != "" {
		content, err := t.vault.ReadNote(path)
		if err != nil {
			return ErrorResult(fmt.Sprintf("failed to read note: %v", err))
		}
		return NewToolResult(content)
	}

	// Topic-based recall: search then read top matches
	maxNotes := 3
	if mn, ok := args["max_notes"].(float64); ok && mn > 0 {
		maxNotes = int(mn)
	}

	results, err := t.vault.Search(topic, nil)
	if err != nil {
		return ErrorResult(fmt.Sprintf("search failed: %v", err))
	}

	if len(results) == 0 {
		return NewToolResult("No matching notes found for topic: " + topic)
	}

	if len(results) > maxNotes {
		results = results[:maxNotes]
	}

	var sb strings.Builder
	for i, n := range results {
		if i > 0 {
			sb.WriteString("\n\n---\n\n")
		}
		sb.WriteString(fmt.Sprintf("## %s (%s)\n\n", n.Title, n.RelPath))
		content, err := t.vault.ReadNote(n.RelPath)
		if err != nil {
			sb.WriteString(fmt.Sprintf("(error reading note: %v)\n", err))
			continue
		}
		// Strip frontmatter from recalled content to avoid duplication
		_, body := memory.ParseFrontmatter(content)
		sb.WriteString(strings.TrimSpace(body))
	}

	return NewToolResult(sb.String())
}
