// ABOUTME: Remember tool for storing semantic memories via vector embeddings.
// ABOUTME: Accepts content, category, and tags, and persists them for later recall.
package tools

import (
	"context"
	"strings"

	"github.com/sipeed/picoclaw/pkg/memory"
)

// RememberTool stores a memory entry in the semantic memory store.
type RememberTool struct {
	store memory.Store
}

func NewRememberTool(store memory.Store) *RememberTool {
	return &RememberTool{store: store}
}

func (t *RememberTool) Name() string { return "remember" }

func (t *RememberTool) Description() string {
	return "Store a memory for later recall. Use this to remember important facts, user preferences, decisions, or any information worth persisting across conversations."
}

func (t *RememberTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{
				"type":        "string",
				"description": "The information to remember. Be specific and self-contained.",
			},
			"category": map[string]any{
				"type":        "string",
				"description": "Category: preference, fact, decision, context, or other.",
				"enum":        []string{"preference", "fact", "decision", "context", "other"},
			},
			"tags": map[string]any{
				"type":        "string",
				"description": "Comma-separated tags for organization (e.g. 'project,database,architecture').",
			},
		},
		"required": []any{"content"},
	}
}

func (t *RememberTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	if !t.store.IsAvailable() {
		return ErrorResult("Semantic memory is not available. Ollama may not be running.")
	}

	content, _ := args["content"].(string)
	if strings.TrimSpace(content) == "" {
		return ErrorResult("content is required")
	}

	category, _ := args["category"].(string)
	if category == "" {
		category = "other"
	}

	var tags []string
	if tagsStr, ok := args["tags"].(string); ok && tagsStr != "" {
		for _, tag := range strings.Split(tagsStr, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tags = append(tags, tag)
			}
		}
	}

	entry := memory.MemoryEntry{
		Content:  content,
		Category: category,
		Tags:     tags,
		Source:   "agent",
	}

	if err := t.store.Remember(ctx, entry); err != nil {
		return ErrorResult("Failed to store memory: " + err.Error())
	}

	return SilentResult("Memory stored successfully: " + content)
}
