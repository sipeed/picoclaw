package tools

import (
	"context"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
)

type CheckMailTool struct {
	config config.EmailConfig
}

func NewCheckMailTool(cfg config.EmailConfig) *CheckMailTool {
	return &CheckMailTool{
		config: cfg,
	}
}

func (t *CheckMailTool) Name() string {
	return "check_email"
}

func (t *CheckMailTool) Description() string {
	return "Check for new emails and return them. This tool fetches recent unread emails from all accounts."
}

func (t *CheckMailTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"count": map[string]interface{}{
				"type":        "integer",
				"description": "Number of emails to fetch (default: 5)",
			},
		},
	}
}

func (t *CheckMailTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	// Delegate to ReadEmailTool
	readTool := NewReadEmailTool(t.config)

	// Default to unread only for "check"
	readArgs := map[string]interface{}{
		"unread_only": true,
		"count":       5.0,
	}

	if c, ok := args["count"]; ok {
		readArgs["count"] = c
	}

	result := readTool.Execute(ctx, readArgs)

	// If no unread, try fetching latest 3 just to show something (optional, but "check" usually implies new)
	if result.ForLLM == "" || strings.Contains(result.ForLLM, "No unread emails") {
		// Fallback to fetch recent 3 if no unread, to confirm connection?
		// No, user said "no mails came through", implies they expected new ones or just wanted to see *something*.
		// But "check mail" usually means "sync and show new".
		// Let's stick to unread.
		return result
	}

	return result
}
