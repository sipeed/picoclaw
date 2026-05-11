package tools

import (
	"context"
	"fmt"
	"strings"
)

type SpawnTool struct {
	manager        *SubagentManager
	spawner        SubTurnSpawner
	defaultModel   string
	maxTokens      int
	temperature    float64
	allowlistCheck func(targetAgentID string) bool
}

// Compile-time check: SpawnTool implements AsyncExecutor.
var _ AsyncExecutor = (*SpawnTool)(nil)

func NewSpawnTool(manager *SubagentManager) *SpawnTool {
	if manager == nil {
		return &SpawnTool{}
	}
	return &SpawnTool{
		manager:      manager,
		defaultModel: manager.defaultModel,
		maxTokens:    manager.maxTokens,
		temperature:  manager.temperature,
	}
}

// SetSpawner sets the SubTurnSpawner for direct sub-turn execution.
func (t *SpawnTool) SetSpawner(spawner SubTurnSpawner) {
	t.spawner = spawner
	if t.manager != nil && spawner != nil {
		t.manager.SetSpawner(func(
			ctx context.Context,
			task, label, agentID string,
			tools *ToolRegistry,
			maxTokens int,
			temperature float64,
			hasMaxTokens, hasTemperature bool,
		) (*ToolResult, error) {
			return spawner.SpawnSubTurn(ctx, SubTurnConfig{
				TargetAgentID: strings.TrimSpace(agentID),
				Model:         t.defaultModel,
				Tools:         nil,
				SystemPrompt:  buildSpawnSystemPrompt(task, label),
				MaxTokens:     maxTokens,
				Temperature:   temperature,
				Async:         false,
				Critical:      true,
			})
		})
	}
}

func (t *SpawnTool) Name() string {
	return "spawn"
}

func (t *SpawnTool) Description() string {
	return "Spawn a subagent to handle a task in the background. Use this for complex or time-consuming tasks that can run independently. The subagent will complete the task and report back when done. Optional delivery_mode controls whether the final async result goes to the user, the parent agent, or both."
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
			"delivery_mode": map[string]any{
				"type":        "string",
				"description": "Optional async result routing policy: user_only, parent_only, or user_and_parent. Defaults to user_only.",
				"enum": []string{
					string(AsyncDeliveryUserOnly),
					string(AsyncDeliveryParentOnly),
					string(AsyncDeliveryUserAndParent),
				},
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

	label, _ := args["label"].(string)
	agentID, _ := args["agent_id"].(string)
	targetAgentID := strings.TrimSpace(agentID)
	deliveryMode, err := parseSpawnDeliveryMode(args["delivery_mode"])
	if err != nil {
		return ErrorResult(err.Error()).WithError(err)
	}

	// Check allowlist if targeting a specific agent
	if targetAgentID != "" && t.allowlistCheck != nil {
		if !t.allowlistCheck(targetAgentID) {
			return ErrorResult(fmt.Sprintf("not allowed to spawn agent '%s'", targetAgentID))
		}
	}

	// Preferred path: route through SubagentManager so spawn_status and
	// background execution share the same task registry.
	if t.manager != nil {
		wrappedCallback := cb
		if cb != nil {
			wrappedCallback = func(cbCtx context.Context, res *ToolResult) {
				if res != nil {
					res.WithAsyncDelivery(deliveryMode)
				}
				cb(cbCtx, res)
			}
		}
		ack, err := t.manager.Spawn(
			ctx,
			task,
			label,
			strings.TrimSpace(agentID),
			ToolChannel(ctx),
			ToolChatID(ctx),
			wrappedCallback,
		)
		if err != nil {
			return ErrorResult(fmt.Sprintf("Spawn failed: %v", err)).WithError(err)
		}
		return AsyncResult(ack)
	}

	// Fallback: manager not configured
	return ErrorResult("Subagent manager not configured")
}

func parseSpawnDeliveryMode(raw any) (AsyncDeliveryMode, error) {
	if raw == nil {
		return AsyncDeliveryUserOnly, nil
	}
	value, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("delivery_mode must be a string")
	}
	switch AsyncDeliveryMode(strings.TrimSpace(value)) {
	case AsyncDeliveryUserOnly, AsyncDeliveryParentOnly, AsyncDeliveryUserAndParent:
		return AsyncDeliveryMode(strings.TrimSpace(value)), nil
	case "":
		return AsyncDeliveryUserOnly, nil
	default:
		return "", fmt.Errorf("delivery_mode must be one of: user_only, parent_only, user_and_parent")
	}
}

func buildSpawnSystemPrompt(task, label string) string {
	if label != "" {
		return fmt.Sprintf(
			`You are a spawned subagent labeled "%s" running in the background. Complete the given task independently and report back when done.

Task: %s`,
			label,
			task,
		)
	}
	return fmt.Sprintf(
		`You are a spawned subagent running in the background. Complete the given task independently and report back when done.
Task: %s`,
		task,
	)
}
