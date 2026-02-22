package tools

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/sipeed/picoclaw/pkg/miniapp"
)

// DevPreviewTool allows the agent to control the Mini App dev reverse proxy.
type DevPreviewTool struct {
	manager miniapp.DevTargetManager
}

// NewDevPreviewTool creates a new DevPreviewTool.
func NewDevPreviewTool(manager miniapp.DevTargetManager) *DevPreviewTool {
	return &DevPreviewTool{manager: manager}
}

func (t *DevPreviewTool) Name() string { return "dev_preview" }

func (t *DevPreviewTool) Description() string {
	return "Control the Mini App dev preview proxy. Use 'start' to register and activate a local dev server (localhost only), 'stop' to deactivate the proxy, 'unregister' to remove a registered target, or 'status' to check all registered targets and active state."
}

func (t *DevPreviewTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"start", "stop", "unregister", "status"},
				"description": "Action to perform: start (register + activate target), stop (deactivate proxy), unregister (remove a registered target), status (list all targets).",
			},
			"target": map[string]any{
				"type":        "string",
				"description": "Target URL for the dev server (e.g. http://localhost:3000). Required for 'start' action. Must be a localhost URL.",
			},
			"name": map[string]any{
				"type":        "string",
				"description": "Display name for the target (e.g. 'frontend'). Optional for 'start' action; auto-generated from host:port if omitted.",
			},
			"id": map[string]any{
				"type":        "string",
				"description": "Target ID. Required for 'unregister' action.",
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
		name, _ := args["name"].(string)
		if name == "" {
			name = inferName(target)
		}
		id, err := t.manager.RegisterDevTarget(name, target)
		if err != nil {
			return ErrorResult(fmt.Sprintf("failed to register dev target: %v", err))
		}
		if err := t.manager.ActivateDevTarget(id); err != nil {
			return ErrorResult(fmt.Sprintf("failed to activate dev target: %v", err))
		}
		return SilentResult(fmt.Sprintf("Dev preview started (id=%s, name=%s). Target: %s\nUsers can view it in the Mini App Dev tab.", id, name, target))

	case "stop":
		if err := t.manager.DeactivateDevTarget(); err != nil {
			return ErrorResult(fmt.Sprintf("failed to stop dev preview: %v", err))
		}
		return SilentResult("Dev preview stopped.")

	case "unregister":
		id, _ := args["id"].(string)
		if id == "" {
			return ErrorResult("id is required for unregister action")
		}
		if err := t.manager.UnregisterDevTarget(id); err != nil {
			return ErrorResult(fmt.Sprintf("failed to unregister target: %v", err))
		}
		return SilentResult(fmt.Sprintf("Dev target %s unregistered.", id))

	case "status":
		targets := t.manager.ListDevTargets()
		active := t.manager.GetDevTarget()
		if len(targets) == 0 {
			if active == "" {
				return SilentResult("Dev preview is not active. No targets registered.")
			}
			return SilentResult(fmt.Sprintf("Dev preview is active. Target: %s\nNo registered targets.", active))
		}
		var sb strings.Builder
		if active != "" {
			sb.WriteString(fmt.Sprintf("Dev preview is active. Target: %s\n", active))
		} else {
			sb.WriteString("Dev preview is not active.\n")
		}
		sb.WriteString("Registered targets:\n")
		for _, dt := range targets {
			sb.WriteString(fmt.Sprintf("  [%s] %s → %s\n", dt.ID, dt.Name, dt.Target))
		}
		return SilentResult(sb.String())

	default:
		return ErrorResult(fmt.Sprintf("unknown action: %s", action))
	}
}

// inferName generates a display name from a target URL (e.g. "localhost:3000").
func inferName(target string) string {
	u, err := url.Parse(target)
	if err != nil {
		return target
	}
	host := u.Hostname()
	port := u.Port()
	if port != "" {
		return host + ":" + port
	}
	return host
}
