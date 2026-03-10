package tools

import (
	"context"
	"fmt"
	"strings"
)

type SpawnTool struct {
	manager        *SubagentManager
	allowlistCheck func(targetAgentID string) bool
}

// Compile-time check: SpawnTool implements AsyncExecutor.
var _ AsyncExecutor = (*SpawnTool)(nil)

func NewSpawnTool(manager *SubagentManager) *SpawnTool {
	return &SpawnTool{
		manager: manager,
	}
}

func (t *SpawnTool) Name() string {
	return "spawn"
}

func (t *SpawnTool) Description() string {
	return "Spawn a subagent to handle a task in the background. Use this for complex or time-consuming tasks that can run independently. The subagent will complete the task and report back when done."
}

func (t *SpawnTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"task": map[string]any{
				"type":        "string",
				"description": "The task for subagent to complete",
			},
			"label": map[string]any{
				"type":        "string",
				"description": "Optional short label for the task (for display)",
			},
			"agent_id": map[string]any{
				"type":        "string",
				"description": "Optional target agent ID to delegate the task to (or the harness ID for ACP)",
			},
			"runtime": map[string]any{
				"type":        "string",
				"description": "Execution runtime. Can be 'subagent' (default) or 'acp'. Use 'acp' for external harnesses like codex, gemini, etc.",
			},
			"mode": map[string]any{
				"type":        "string",
				"description": "For ACP runtime: 'run' (one-shot) or 'session' (persistent)",
			},
			"cwd": map[string]any{
				"type":        "string",
				"description": "Requested working directory for the subagent or ACP process",
			},
		},
		"required": []string{"task"},
	}
}

func (t *SpawnTool) SetAllowlistChecker(check func(targetAgentID string) bool) {
	t.allowlistCheck = check
}

func (t *SpawnTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	return t.execute(ctx, args, nil)
}

// ExecuteAsync implements AsyncExecutor. The callback is passed through to the
// subagent manager as a call parameter — never stored on the SpawnTool instance.
func (t *SpawnTool) ExecuteAsync(ctx context.Context, args map[string]any, cb AsyncCallback) *ToolResult {
	return t.execute(ctx, args, cb)
}

func (t *SpawnTool) execute(ctx context.Context, args map[string]any, cb AsyncCallback) *ToolResult {
	task, ok := args["task"].(string)
	if !ok || strings.TrimSpace(task) == "" {
		return ErrorResult("task is required and must be a non-empty string")
	}

	label, _ := args["label"].(string)
	agentID, _ := args["agent_id"].(string)

	// Check allowlist if targeting a specific agent
	if agentID != "" && t.allowlistCheck != nil {
		if !t.allowlistCheck(agentID) {
			return ErrorResult(fmt.Sprintf("not allowed to spawn agent '%s'", agentID))
		}
	}

	// Extract new parameters
	runtime, _ := args["runtime"].(string)
	if runtime == "" {
		runtime = "subagent"
	}
	mode, _ := args["mode"].(string)
	if mode == "" {
		mode = "run"
	}
	cwd, _ := args["cwd"].(string)
	
	if runtime == "acp" {
		// Verify agent_id mapping. E.g., agent_id=gemini might map to `gemini` executable.
		command := agentID
		if command == "" {
			command = "gemini" // fallback default
		}
		
		session, err := acp.GetManager().Spawn(agentID, mode, command, cwd, label, []string{})
		if err != nil {
			return ErrorResult(fmt.Sprintf("Failed to spawn ACP session: %v", err))
		}
		
		// Send initial task
		if err := session.Write(task); err != nil {
			return ErrorResult(fmt.Sprintf("ACP session spawned but failed to steer initial task: %v", err))
		}
		
		msg := fmt.Sprintf("Spawned ACP session '%s' (key: %s)", command, session.Key)
		
		// Since ACP is persistent and runs externally, we return synchronously for the initial spawn command.
		// Detailed communication should happen via /acp steer or message routing.
		return &ToolResult{
			ForLLM:  msg,
			ForUser: msg,
			Silent:  false,
			IsError: false,
			Async:   false,
		}
	}

	// Normal Subagent Logic
	if t.manager == nil {
		return ErrorResult("Subagent manager not configured")
	}

	channel := ToolChannel(ctx)
	if channel == "" {
		channel = "cli"
	}
	chatID := ToolChatID(ctx)
	if chatID == "" {
		chatID = "direct"
	}

	// Pass callback to manager for async completion notification
	result, err := t.manager.Spawn(ctx, task, label, agentID, channel, chatID, cb)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to spawn subagent: %v", err))
	}

	// Return AsyncResult since the task runs in background
	return AsyncResult(result)
}
