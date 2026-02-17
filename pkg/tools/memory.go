package tools

import (
	"context"
	"fmt"
)

// MemoryWriter is the interface for memory operations.
// Implemented by agent.MemoryStore.
type MemoryWriter interface {
	WriteLongTerm(content string) error
	AppendToday(content string) error
	ReadLongTerm() string
	ReadToday() string
}

type MemoryTool struct {
	writer MemoryWriter
}

func NewMemoryTool(writer MemoryWriter) *MemoryTool {
	return &MemoryTool{writer: writer}
}

func (t *MemoryTool) Name() string {
	return "memory"
}

func (t *MemoryTool) Description() string {
	return "Store and retrieve persistent memory. Actions: write_long_term, append_daily, read_long_term, read_daily"
}

func (t *MemoryTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"description": "The memory action to perform",
				"enum":        []string{"write_long_term", "append_daily", "read_long_term", "read_daily"},
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "Content to write (required for write_long_term and append_daily)",
			},
		},
		"required": []string{"action"},
	}
}

func (t *MemoryTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	action, ok := args["action"].(string)
	if !ok {
		return ErrorResult("action is required")
	}

	switch action {
	case "write_long_term":
		content, ok := args["content"].(string)
		if !ok || content == "" {
			return ErrorResult("content is required for write_long_term")
		}
		if err := t.writer.WriteLongTerm(content); err != nil {
			return ErrorResult(fmt.Sprintf("failed to write long-term memory: %v", err))
		}
		return SilentResult("Long-term memory updated successfully")

	case "append_daily":
		content, ok := args["content"].(string)
		if !ok || content == "" {
			return ErrorResult("content is required for append_daily")
		}
		if err := t.writer.AppendToday(content); err != nil {
			return ErrorResult(fmt.Sprintf("failed to append daily note: %v", err))
		}
		return SilentResult("Daily note appended successfully")

	case "read_long_term":
		content := t.writer.ReadLongTerm()
		if content == "" {
			return SilentResult("No long-term memory found")
		}
		return SilentResult(content)

	case "read_daily":
		content := t.writer.ReadToday()
		if content == "" {
			return SilentResult("No daily notes for today")
		}
		return SilentResult(content)

	default:
		return ErrorResult(fmt.Sprintf("unknown action: %s", action))
	}
}
