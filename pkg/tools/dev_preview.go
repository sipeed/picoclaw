package tools

import (
	"context"
	"fmt"

	"github.com/sipeed/picoclaw/pkg/miniapp"
)

// DevPreviewTool allows the agent to control the Mini App dev reverse proxy.
type DevPreviewTool struct {
	setter miniapp.DevTargetSetter
}

// NewDevPreviewTool creates a new DevPreviewTool.
func NewDevPreviewTool(setter miniapp.DevTargetSetter) *DevPreviewTool {
	return &DevPreviewTool{setter: setter}
}

func (t *DevPreviewTool) Name() string { return "dev_preview" }

func (t *DevPreviewTool) Description() string {
	return "Control the Mini App dev preview proxy. Use 'start' to expose a local dev server (localhost only) through the Mini App, 'stop' to disable it, or 'status' to check the current state."
}

func (t *DevPreviewTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"start", "stop", "status"},
				"description": "Action to perform: start (set proxy target), stop (disable proxy), status (check current target).",
			},
			"target": map[string]any{
				"type":        "string",
				"description": "Target URL for the dev server (e.g. http://localhost:3000). Required for 'start' action. Must be a localhost URL.",
			},
		},
		"required": []string{"action"},
	}
}

func (t *DevPreviewTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	action, ok := args["action"].(string)
	if !ok {
		return ErrorResult("action is required")
	}

	switch action {
	case "start":
		target, _ := args["target"].(string)
		if target == "" {
			return ErrorResult("target is required for start action")
		}
		if err := t.setter.SetDevTarget(target); err != nil {
			return ErrorResult(fmt.Sprintf("failed to set dev target: %v", err))
		}
		return SilentResult(fmt.Sprintf("Dev preview started. Target: %s\nUsers can view it in the Mini App Dev tab.", target))

	case "stop":
		if err := t.setter.SetDevTarget(""); err != nil {
			return ErrorResult(fmt.Sprintf("failed to stop dev preview: %v", err))
		}
		return SilentResult("Dev preview stopped.")

	case "status":
		target := t.setter.GetDevTarget()
		if target == "" {
			return SilentResult("Dev preview is not active.")
		}
		return SilentResult(fmt.Sprintf("Dev preview is active. Target: %s", target))

	default:
		return ErrorResult(fmt.Sprintf("unknown action: %s", action))
	}
}
