package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/providers"
)

// SpawnSubAgentTool executes a customized subagent task synchronously using Anthology-style single worker delegation.
type SpawnSubAgentTool struct {
	manager       *SubagentManager
	originChannel string
	originChatID  string
}

func NewSpawnSubAgentTool(manager *SubagentManager) *SpawnSubAgentTool {
	return &SpawnSubAgentTool{
		manager:       manager,
		originChannel: "cli",
		originChatID:  "direct",
	}
}

func (t *SpawnSubAgentTool) Name() string {
	return "spawn_sub_agent"
}

func (t *SpawnSubAgentTool) Description() string {
	return "Directly delegate a specific task to a new, isolated sub-agent. You (the main agent) should autonomously determine the appropriate expert role and specific task based on the user's high-level request. It will execute independently and return the final result."
}

func (t *SpawnSubAgentTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"task": map[string]any{
				"type":        "string",
				"description": "The specific task the sub-agent needs to accomplish.",
			},
			"role": map[string]any{
				"type":        "string",
				"description": "The system prompt/role assignment for the sub-agent (e.g., 'You are an expert code reviewer').",
			},
		},
		"required": []string{"task", "role"},
	}
}

func (t *SpawnSubAgentTool) SetContext(channel, chatID string) {
	t.originChannel = channel
	t.originChatID = chatID
}

func (t *SpawnSubAgentTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	task, ok := args["task"].(string)
	if !ok || strings.TrimSpace(task) == "" {
		return ErrorResult("task is required").WithError(fmt.Errorf("task parameter is required"))
	}

	role, ok := args["role"].(string)
	if !ok || strings.TrimSpace(role) == "" {
		return ErrorResult("role is required").WithError(fmt.Errorf("role parameter is required"))
	}

	if t.manager == nil {
		return ErrorResult("Subagent manager not configured").WithError(fmt.Errorf("manager is nil"))
	}

	// 1. Isolation: Each SubAgent gets a completely fresh message set
	messages := []providers.Message{
		{
			Role:    "system",
			Content: role,
		},
		{
			Role:    "user",
			Content: task,
		},
	}

	// 2. Base Configuration (Timeout & LLM constraints)
	config := t.manager.BuildBaseWorkerConfig(ctx)

	// Note: For MVP, we pass the current ToolRegistry unmodified.
	// To enforce strict sandboxing later, we can construct a new ToolRegistry here based on args['allowed_tools'].

	loopResult, err := RunToolLoop(ctx, config, messages, t.originChannel, t.originChatID)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Subagent execution failed: %v", err)).WithError(err)
	}

	// Return full details to LLM
	llmContent := fmt.Sprintf("Subagent (Role: %s) task completed:\nIterations: %d\nResult: %s",
		role, loopResult.Iterations, loopResult.Content)

	return &ToolResult{
		ForLLM:  llmContent,
		ForUser: "Sub-agent finished task.",
		Silent:  false,
		IsError: false,
		Async:   false,
	}
}
