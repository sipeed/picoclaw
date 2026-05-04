package seahorse

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sipeed/picoclaw/pkg/tools"
)

// GrepTool searches summaries and messages for matching content.
type GrepTool struct {
	engine *RetrievalEngine
}

func NewGrepTool(engine *RetrievalEngine) *GrepTool {
	return &GrepTool{engine: engine}
}

func (t *GrepTool) Name() string {
	return "short_grep"
}

func (t *GrepTool) Description() string {
	return `Search summaries and messages for matching content.

Pattern syntax:
- Words: "authentication" - matches content containing this word
- AND: "auth AND login" - matches content with both words
- OR: "auth OR signin" - matches content with either word
- NOT: "bug NOT fixed" - matches "bug" but excludes "fixed"
- Wildcard: "%auth%" - matches any text containing "auth" (e.g., "auth", "authentication")

Each summary has a "depth" field:
- depth 0: Created from messages, most detailed
- depth 1+: Created from other summaries, more compressed but covers longer time

Parameters:
- pattern (required): Search pattern
- scope: "both" (default), "summary", or "message" - what to search
- role: "user", "assistant", or omit for all - filter by message role
- last: Time shortcut like "6h", "7d", "2w", "1m" (hours/days/weeks/months)
- all_conversations: Search all conversations (default: current only)
- since: ISO8601 timestamp, content after this time
- before: ISO8601 timestamp, content before this time
- limit: Max results (default: 20)

Returns:
{
  "success": true,
  "summaries": [{"id": "sum_abc", "content": "...", "depth": 0, "kind": "leaf", "conversationId": 1, "rank": -0.5}],
  "messages": [{"id": "10", "snippet": "...matched...", "role": "user", "conversationId": 1, "rank": -1.2}],
  "totalSummaries": 5,
  "totalMessages": 10,
  "hint": "No matches. Try: %keyword% for fuzzy search"
}

Rank field (FTS5 mode only): bm25 relevance score, negative value where more negative = higher relevance.
Examples: -5=excellent, -2=good, -0.5=partial. LIKE mode (%pattern%) has no rank.

Examples:
  {"pattern": "authentication"}
  {"pattern": "bug AND login"}
  {"pattern": "%snake%"}
  {"pattern": "project", "scope": "summary"}
  {"pattern": "error", "role": "assistant", "last": "7d"}
  {"pattern": "error", "all_conversations": true}`
}

func (t *GrepTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "Search pattern. Supports: words, AND/OR/NOT operators, % wildcard",
			},
			"scope": map[string]any{
				"type":        "string",
				"enum":        []string{"both", "summary", "message"},
				"description": "What to search: 'both' (default), 'summary', or 'message'",
			},
			"role": map[string]any{
				"type":        "string",
				"enum":        []string{"user", "assistant"},
				"description": "Filter by message role (default: all roles)",
			},
			"last": map[string]any{
				"type":        "string",
				"description": "Time shortcut: '6h' (6 hours), '7d' (7 days), '2w' (2 weeks), '1m' (1 month)",
			},
			"all_conversations": map[string]any{
				"type":        "boolean",
				"description": "Search across all conversations (default: searches current conversation only)",
			},
			"since": map[string]any{
				"type":        "string",
				"description": "ISO8601 timestamp, only return content after this time",
			},
			"before": map[string]any{
				"type":        "string",
				"description": "ISO8601 timestamp, only return content before this time",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Maximum number of results (default: 20)",
			},
		},
		"required": []string{"pattern"},
	}
}

func (t *GrepTool) Execute(ctx context.Context, args map[string]any) *tools.ToolResult {
	pattern, ok := args["pattern"].(string)
	if !ok || pattern == "" {
		return tools.ErrorResult("Missing required 'pattern' argument. Example: {\"pattern\": \"authentication\"}")
	}

	input := GrepInput{Pattern: pattern}

	if scope, ok := args["scope"].(string); ok && scope != "" {
		input.Scope = scope
	}
	if role, ok := args["role"].(string); ok && role != "" {
		input.Role = role
	}
	if last, ok := args["last"].(string); ok && last != "" {
		input.Last = last
	}
	if allConv, ok := args["all_conversations"].(bool); ok {
		input.AllConversations = allConv
	}
	if !input.AllConversations {
		conversationID, err := t.engine.ConversationIDForSession(ctx, tools.ToolSessionKey(ctx))
		if err != nil {
			return tools.ErrorResult("Grep failed: resolve current conversation: " + err.Error())
		}
		input.ConversationID = conversationID
	}
	if limit, ok := args["limit"].(float64); ok {
		input.Limit = int(limit)
	}
	if sinceStr, ok := args["since"].(string); ok && sinceStr != "" {
		parsed, err := time.Parse(time.RFC3339, sinceStr)
		if err != nil {
			return tools.ErrorResult(fmt.Sprintf(
				"Invalid 'since' timestamp. Use RFC3339 format like '2024-01-15T10:00:00Z'. Error: %v", err))
		}
		input.Since = &parsed
	}
	if beforeStr, ok := args["before"].(string); ok && beforeStr != "" {
		parsed, err := time.Parse(time.RFC3339, beforeStr)
		if err != nil {
			return tools.ErrorResult(fmt.Sprintf("Invalid 'before' timestamp format: %v", err))
		}
		input.Before = &parsed
	}

	result, err := t.engine.Grep(ctx, input)
	if err != nil {
		return tools.ErrorResult("Grep failed: " + err.Error())
	}

	// Build response
	output := map[string]any{
		"success":   result.Success,
		"summaries": result.Summaries,
		"messages":  result.Messages,
	}

	// Add hint if provided
	if result.Hint != "" {
		output["hint"] = result.Hint
	}

	data, _ := json.Marshal(output)
	return tools.NewToolResult(string(data))
}
