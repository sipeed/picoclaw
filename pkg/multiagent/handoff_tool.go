package multiagent

import (
	"context"
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/tools"
)

// HandoffTool allows an LLM agent to delegate a task to another agent.
type HandoffTool struct {
	resolver      AgentResolver
	board         *Blackboard
	fromAgentID   string
	originChannel string
	originChatID  string
}

// NewHandoffTool creates a handoff tool bound to a specific source agent.
func NewHandoffTool(resolver AgentResolver, board *Blackboard, fromAgentID string) *HandoffTool {
	return &HandoffTool{
		resolver:      resolver,
		board:         board,
		fromAgentID:   fromAgentID,
		originChannel: "cli",
		originChatID:  "direct",
	}
}

func (t *HandoffTool) Name() string { return "handoff" }

func (t *HandoffTool) Description() string {
	agents := t.resolver.ListAgents()
	if len(agents) <= 1 {
		return "Delegate a task to another agent. No other agents are currently available."
	}

	var sb strings.Builder
	sb.WriteString("Delegate a task to another agent. Available agents:\n")
	for _, a := range agents {
		if a.ID == t.fromAgentID {
			continue
		}
		sb.WriteString(fmt.Sprintf("- %s", a.ID))
		if a.Name != "" {
			sb.WriteString(fmt.Sprintf(" (%s)", a.Name))
		}
		if a.Role != "" {
			sb.WriteString(fmt.Sprintf(": %s", a.Role))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func (t *HandoffTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"agent_id": map[string]interface{}{
				"type":        "string",
				"description": "The ID of the target agent to hand off to",
			},
			"task": map[string]interface{}{
				"type":        "string",
				"description": "The task description for the target agent",
			},
			"context": map[string]interface{}{
				"type":        "object",
				"description": "Optional key-value context to share via blackboard before handoff",
			},
		},
		"required": []string{"agent_id", "task"},
	}
}

func (t *HandoffTool) SetContext(channel, chatID string) {
	t.originChannel = channel
	t.originChatID = chatID
}

func (t *HandoffTool) Execute(ctx context.Context, args map[string]interface{}) *tools.ToolResult {
	agentID, _ := args["agent_id"].(string)
	task, _ := args["task"].(string)

	if agentID == "" {
		return tools.ErrorResult("agent_id is required")
	}
	if task == "" {
		return tools.ErrorResult("task is required")
	}

	// Parse optional context map
	var contextMap map[string]string
	if ctxRaw, ok := args["context"].(map[string]interface{}); ok {
		contextMap = make(map[string]string, len(ctxRaw))
		for k, v := range ctxRaw {
			contextMap[k] = fmt.Sprintf("%v", v)
		}
	}

	result := ExecuteHandoff(ctx, t.resolver, t.board, HandoffRequest{
		FromAgentID: t.fromAgentID,
		ToAgentID:   agentID,
		Task:        task,
		Context:     contextMap,
	}, t.originChannel, t.originChatID)

	if !result.Success {
		return tools.ErrorResult(fmt.Sprintf("Handoff to %q failed: %s", agentID, result.Error))
	}

	return &tools.ToolResult{
		ForLLM:  fmt.Sprintf("Agent %q completed task (iterations: %d):\n%s", agentID, result.Iterations, result.Content),
		ForUser: result.Content,
	}
}
