package tools

import (
	"context"
	"fmt"
	"strings"
)

type SpawnTool struct {
	manager *SubagentManager

	originChannel string

	originChatID string

	allowlistCheck func(targetAgentID string) bool

	callback AsyncCallback // For async completion notification
}

// Compile-time check: SpawnTool implements AsyncExecutor.
var _ AsyncExecutor = (*SpawnTool)(nil)

func NewSpawnTool(manager *SubagentManager) *SpawnTool {
	return &SpawnTool{
		manager: manager,

		originChannel: "cli",

		originChatID: "direct",
	}
}

// SetCallback implements AsyncTool interface for async completion notification

func (t *SpawnTool) SetCallback(cb AsyncCallback) {
	t.callback = cb
}

func (t *SpawnTool) Name() string {
	return "spawn"
}

func (t *SpawnTool) Description() string {
	return "Spawn a subagent that runs NON-BLOCKING in the background and returns immediately. Prefer this over subagent for any task that can run independently. Use preset to control capabilities (scout, analyst, coder, worker, coordinator)."
}

func (t *SpawnTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"task": map[string]any{
				"type": "string",

				"description": "The task for subagent to complete",
			},
			"label": map[string]any{
				"type": "string",

				"description": "Optional short label for the task (for display)",
			},
			"agent_id": map[string]any{
				"type": "string",

				"description": "Optional target agent ID to delegate the task to",
			},

			"preset": map[string]any{
				"type": "string",

				"enum": []string{"scout", "analyst", "coder", "worker", "coordinator"},

				"description": "Optional capability tier: scout (explore), analyst (analyze), coder (code), worker (build), coordinator (orchestrate)",
			},
		},
		"required": []string{"task"},
	}
}

func (t *SpawnTool) SetContext(channel, chatID string) {
	t.originChannel = channel

	t.originChatID = chatID
}

func (t *SpawnTool) SetAllowlistChecker(check func(targetAgentID string) bool) {
	t.allowlistCheck = check
}

func (t *SpawnTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	return t.execute(ctx, args, t.callback)
}

// ExecuteAsync implements AsyncExecutor. The callback is passed through to the
// subagent manager as a call parameter — never stored on the SpawnTool instance.
func (t *SpawnTool) ExecuteAsync(ctx context.Context, args map[string]any, cb AsyncCallback) *ToolResult {
	return t.execute(ctx, args, cb)
}

func (t *SpawnTool) execute(ctx context.Context, args map[string]any, cb AsyncCallback) *ToolResult {
	task, ok := args["task"].(string)
	if !ok || strings.TrimSpace(task) == "" {
		return ErrorResult(

			`Required parameter "task" (string) is missing. ` +

				`Example: {"task": "describe what you need done", "preset": "scout"}`,
		)
	}

	label, _ := args["label"].(string)
	agentID, _ := args["agent_id"].(string)

	preset, _ := args["preset"].(string)

	// Check allowlist if targeting a specific agent ID.

	// Presets (scout, analyst, etc.) are NOT agent IDs — they are validated

	// separately by IsValidPreset() in the subagent manager.

	if agentID != "" && t.allowlistCheck != nil {
		if !t.allowlistCheck(agentID) {
			return ErrorResult(fmt.Sprintf("agent %q is not in the allowed agents list", agentID))
		}
	}

	// Validate preset name if provided

	if preset != "" && !IsValidPreset(Preset(preset)) {
		return ErrorResult(fmt.Sprintf(

			"preset %q is not valid. Available presets: scout, analyst, coder, worker, coordinator",

			preset,
		))
	}

	if t.manager == nil {
		return ErrorResult("spawn tool is not available in this session (orchestration may be disabled)")
	}

	// Pass callback to manager for async completion notification

	result, err := t.manager.Spawn(ctx, task, label, agentID, t.originChannel, t.originChatID, preset, cb)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to spawn subagent: %v", err))
	}

	// Return AsyncResult since the task runs in background
	return AsyncResult(result)
}
