package tools

import (
	"context"
	"fmt"
	"time"

	toolshared "github.com/sipeed/picoclaw/pkg/tools/shared"
)

// GetCurrentTimeTool returns the current time and/or date information
type GetCurrentTimeTool struct {
	timezone string
}

// NewGetCurrentTimeTool creates a new GetCurrentTimeTool
func NewGetCurrentTimeTool(timezone string) *GetCurrentTimeTool {
	if timezone == "" {
		timezone = "Local"
	}
	return &GetCurrentTimeTool{
		timezone: timezone,
	}
}

// Name returns the tool name
func (t *GetCurrentTimeTool) Name() string {
	return "get_current_time"
}

// Description returns the tool description
func (t *GetCurrentTimeTool) Description() string {
	return "Get the current time, date, or both. Returns ISO 8601 format by default, or can return formatted strings suitable for display."
}

// Parameters returns the tool parameters schema
func (t *GetCurrentTimeTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"format": map[string]any{
				"type":        "string",
				"enum":        []string{"iso", "time", "date", "datetime", "unix"},
				"description": "Output format: 'iso' (ISO 8601, default), 'time' (HH:MM:SS), 'date' (YYYY-MM-DD), 'datetime' (YYYY-MM-DD HH:MM:SS), 'unix' (Unix timestamp)",
				"default":     "iso",
			},
			"timezone": map[string]any{
				"type":        "string",
				"description": "Timezone name (e.g., 'America/New_York', 'Europe/London', 'Asia/Shanghai'). Uses system local time if not specified.",
			},
		},
	}
}

// Execute runs the tool
func (t *GetCurrentTimeTool) Execute(ctx context.Context, args map[string]any) *toolshared.ToolResult {
	// Get timezone
	tzName := t.timezone
	if tzArg, ok := args["timezone"].(string); ok && tzArg != "" {
		tzName = tzArg
	}

	// Load timezone
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		// Fallback to local timezone if specified one is invalid
		loc = time.Local
		tzName = "Local"
	}

	now := time.Now().In(loc)

	// Get format
	format := "iso"
	if fmtArg, ok := args["format"].(string); ok && fmtArg != "" {
		format = fmtArg
	}

	var result string
	switch format {
	case "time":
		result = now.Format("15:04:05")
	case "date":
		result = now.Format("2006-01-02")
	case "datetime":
		result = now.Format("2006-01-02 15:04:05")
	case "unix":
		result = fmt.Sprintf("%d", now.Unix())
	case "iso":
		fallthrough
	default:
		result = now.Format(time.RFC3339)
	}

	// Build response
	response := fmt.Sprintf("Current time (%s): %s", tzName, result)

	return &toolshared.ToolResult{
		ForLLM:  response,
		ForUser: response,
	}
}
