package tools

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// LogsTool provides on-demand access to application logs from the in-memory ring buffer.

// Designed for token-efficient log analysis: defaults to WARN level to exclude noise.

type LogsTool struct{}

func NewLogsTool() *LogsTool {
	return &LogsTool{}
}

func (t *LogsTool) Name() string { return "logs" }

func (t *LogsTool) Description() string {
	return "Retrieve recent application logs from the in-memory ring buffer. " +

		"Use level filter to minimize token usage (default: WARN). " +

		"Call this when the user asks about errors, issues, or system health."
}

func (t *LogsTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",

		"properties": map[string]any{
			"level": map[string]any{
				"type": "string",

				"description": "Minimum log level: DEBUG, INFO, WARN, ERROR. Default: WARN",

				"enum": []string{"DEBUG", "INFO", "WARN", "ERROR"},
			},

			"component": map[string]any{
				"type": "string",

				"description": "Filter by component name (e.g. telegram, discord, slack, agent)",
			},

			"limit": map[string]any{
				"type": "integer",

				"description": "Maximum number of log entries to return. Default: 50",
			},

			"query": map[string]any{
				"type": "string",

				"description": "Filter by substring match in log message",
			},
		},
	}
}

func (t *LogsTool) Execute(_ context.Context, args map[string]any) *ToolResult {
	// Parse level (default: WARN)

	level := logger.WARN

	if lvlStr, ok := args["level"].(string); ok && lvlStr != "" {
		if parsed, ok := logger.ParseLevel(lvlStr); ok {
			level = parsed
		}
	}

	// Parse component

	component, _ := args["component"].(string)

	// Parse limit (default: 50, max: 300)

	limit := 50

	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	if limit > 300 {
		limit = 300
	}

	// Parse query

	query, _ := args["query"].(string)

	// Fetch from ring buffer (already sanitized by RecentLogs)

	entries := logger.RecentLogs(level, component, limit)

	// Apply query filter if specified

	if query != "" {
		filtered := make([]logger.LogEntry, 0, len(entries))

		queryLower := strings.ToLower(query)

		for _, e := range entries {
			if strings.Contains(strings.ToLower(e.Message), queryLower) {
				filtered = append(filtered, e)
			}
		}

		entries = filtered
	}

	if len(entries) == 0 {
		return SilentResult("No log entries found matching the criteria.")
	}

	data, err := json.Marshal(entries)
	if err != nil {
		return ErrorResult("Failed to marshal log entries: " + err.Error())
	}

	return SilentResult(string(data))
}
