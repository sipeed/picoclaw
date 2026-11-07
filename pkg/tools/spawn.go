package tools

import (
	"context"
	"fmt"
	"strings"
)

type SpawnTool struct {
	spawner        SubTurnSpawner
	defaultModel   string
	maxTokens      int
	temperature    float64
	allowlistCheck func(targetAgentID string) bool
	tools          *ToolRegistry
}

// Compile-time check: SpawnTool implements AsyncExecutor.
var _ AsyncExecutor = (*SpawnTool)(nil)

func NewSpawnTool(manager *SubagentManager) *SpawnTool {
	if manager == nil {
		return &SpawnTool{}
	}
	return &SpawnTool{
		defaultModel: manager.defaultModel,
		maxTokens:    manager.maxTokens,
		temperature:  manager.temperature,
		tools:        manager.tools,
	}
}

// SetSpawner sets the SubTurnSpawner for direct sub-turn execution.
func (t *SpawnTool) SetSpawner(spawner SubTurnSpawner) {
	t.spawner = spawner
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
				"description": "Optional target agent ID to delegate the task to",
			},
			"direct_reply": map[string]any{
				"type":        "boolean",
				"description": "When true (default), the subagent's result is sent directly to the user and saved to conversation context — no follow-up LLM turn is triggered. When false, the result is returned to the main agent which will process and present it to the user.",
			},
			"allow_direct_message": map[string]any{
				"type":        "boolean",
				"description": "When true (default), the spawned subagent is allowed to use the message tool to send messages directly to the user during execution. When false, the message tool is removed from the subagent's toolset, so it cannot directly message the user; results must be returned to the main agent.",
			},
		},
		"required": []string{"task"},
	}
}

func (t *SpawnTool) SetAllowlistChecker(check func(targetAgentID string) bool) {
	t.allowlistCheck = check
}

// SetTools sets the tool registry for spawned subagents. When allow_direct_message
// is false, the message tool will be filtered out from the subagent's toolset.
func (t *SpawnTool) SetTools(tools *ToolRegistry) {
	t.tools = tools
}

func (t *SpawnTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	return t.execute(ctx, args, nil)
}

// ExecuteAsync implements AsyncExecutor. The callback is passed through to the
// subagent manager as a call parameter — never stored on the SpawnTool instance.
func (t *SpawnTool) ExecuteAsync(
	ctx context.Context,
	args map[string]any,
	cb AsyncCallback,
) *ToolResult {
	return t.execute(ctx, args, cb)
}

func (t *SpawnTool) execute(
	ctx context.Context,
	args map[string]any,
	cb AsyncCallback,
) *ToolResult {
	task, ok := args["task"].(string)
	if !ok || strings.TrimSpace(task) == "" {
		return ErrorResult("task is required and must be a non-empty string")
	}

	label, ok := args["label"].(string)
	if !ok {
		label = ""
	}
	agentID, ok := args["agent_id"].(string)
	if !ok {
		agentID = ""
	}
	targetAgentID := strings.TrimSpace(agentID)

	// direct_reply controls whether the subagent result is sent directly to
	// the user (true, default) or returned to the main agent for processing.
	// When true, SkipInboundTurn is set on the result so the async callback
	// does not publish an inbound message — instead it saves the result to
	// session history so it's available as context in the next user turn.
	// When false, ForUser is stripped so only the inbound message path
	// remains, and the main agent will present the result in its response.
	directReply := true
	if v, ok := args["direct_reply"].(bool); ok {
		directReply = v
	}

	// allow_direct_message controls whether the spawned subagent can use the
	// message tool to send messages directly to the user during execution.
	// Defaults to true (preserves existing behavior).
	allowDirectMessage := true
	if v, ok := args["allow_direct_message"].(bool); ok {
		allowDirectMessage = v
	}

	// Build the toolset for the subagent. When allow_direct_message is false,
	// filter out the message tool so the subagent cannot directly message users.
	var subTools []Tool
	if t.tools != nil {
		for _, name := range t.tools.List() {
			if !allowDirectMessage && name == "message" {
				continue
			}
			if tool, ok := t.tools.Get(name); ok {
				subTools = append(subTools, tool)
			}
		}
	}

	// Check allowlist if targeting a specific agent
	if targetAgentID != "" && t.allowlistCheck != nil {
		if !t.allowlistCheck(targetAgentID) {
			return ErrorResult(fmt.Sprintf("not allowed to spawn agent '%s'", targetAgentID))
		}
	}

	// Build system prompt for spawned subagent
	systemPrompt := fmt.Sprintf(
		`You are a spawned subagent running in the background. Complete the given task independently and report back when done.

Task: %s`,
		task,
	)

	if label != "" {
		systemPrompt = fmt.Sprintf(
			`You are a spawned subagent labeled "%s" running in the background. Complete the given task independently and report back when done.

Task: %s`,
			label,
			task,
		)
	}

	// Use spawner if available (direct SpawnSubTurn call)
	if t.spawner != nil {
		// Launch async sub-turn in goroutine
		go func() {
			result, err := t.spawner.SpawnSubTurn(ctx, SubTurnConfig{
				Model:         t.defaultModel,
				Tools:         subTools,
				SystemPrompt:  systemPrompt,
				MaxTokens:     t.maxTokens,
				Temperature:   t.temperature,
				Async:         true, // Async execution
				Critical:      true, // Background spawn should survive parent turn completion
				TargetAgentID: targetAgentID,
			})
			if err != nil {
				result = ErrorResult(fmt.Sprintf("Spawn failed: %v", err)).WithError(err)
			}

			// When direct_reply is false, strip ForUser to prevent the async
			// callback from pushing the result directly to the user.
			// Only ForLLM remains, which flows through the inbound message
			// path so the main agent can process and present the result.
			if !directReply && result != nil {
				result.ForUser = ""
			}

			// When direct_reply is true, set SkipInboundTurn so the async
			// callback skips the inbound publish (no follow-up LLM turn)
			// and instead saves the result to session history for context
			// continuity. The ForUser push above already delivered the
			// result directly to the user.
			if directReply && result != nil {
				result.SkipInboundTurn = true
			}

			// Call callback if provided
			if cb != nil {
				cb(ctx, result)
			}
		}()

		// Return immediate acknowledgment
		if label != "" {
			return AsyncResult(fmt.Sprintf("Spawned subagent '%s' for task: %s", label, task))
		}
		return AsyncResult(fmt.Sprintf("Spawned subagent for task: %s", task))
	}

	// Fallback: spawner not configured
	return ErrorResult("Subagent manager not configured")
}
