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
	depth            int              // current handoff depth (0 = top-level)
	visited          []string         // agent IDs already in the call chain
	maxDepth         int              // max allowed depth (0 = use DefaultMaxHandoffDepth)
	allowlistChecker AllowlistChecker // optional; nil = allow all
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

// Name returns the tool name.
func (t *HandoffTool) Name() string { return "handoff" }

// Description returns a dynamic description listing available target agents.
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
		fmt.Fprintf(&sb, "- %s", a.ID)
		if a.Name != "" {
			fmt.Fprintf(&sb, " (%s)", a.Name)
		}
		if a.Role != "" {
			fmt.Fprintf(&sb, ": %s", a.Role)
		}
		if len(a.Capabilities) > 0 {
			fmt.Fprintf(&sb, " [%s]", strings.Join(a.Capabilities, ", "))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// Parameters returns the JSON Schema for the tool's input.
func (t *HandoffTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"agent_id": map[string]any{
				"type":        "string",
				"description": "The ID of the target agent to hand off to (required if capability is not set)",
			},
			"capability": map[string]any{
				"type":        "string",
				"description": "Route to an agent with this capability instead of by ID (e.g. \"coding\", \"research\")",
			},
			"task": map[string]any{
				"type":        "string",
				"description": "The task description for the target agent",
			},
			"context": map[string]any{
				"type":        "object",
				"description": "Optional key-value context to share via blackboard before handoff",
			},
		},
		"required": []string{"task"},
	}
}

// SetBoard replaces the blackboard reference, allowing the tool to be wired
// to the correct per-session board before each execution.
func (t *HandoffTool) SetBoard(board *Blackboard) {
	t.board = board
}

// SetAllowlistChecker sets an optional checker that controls which agents
// can be handed off to. If nil, all handoffs are allowed.
func (t *HandoffTool) SetAllowlistChecker(checker AllowlistChecker) {
	t.allowlistChecker = checker
}

// SetContext updates the origin channel and chat ID for handoff routing.
func (t *HandoffTool) SetContext(channel, chatID string) {
	t.originChannel = channel
	t.originChatID = chatID
}

// Execute delegates a task to the specified target agent.
func (t *HandoffTool) Execute(ctx context.Context, args map[string]any) *tools.ToolResult {
	agentID, _ := args["agent_id"].(string)
	capability, _ := args["capability"].(string)
	task, _ := args["task"].(string)

	if task == "" {
		return tools.ErrorResult("task is required")
	}

	// Resolve agent: by ID or by capability
	if agentID == "" && capability != "" {
		matches := FindAgentsByCapability(t.resolver, capability)
		if len(matches) == 0 {
			return tools.ErrorResult(fmt.Sprintf("no agent found with capability %q", capability))
		}
		agentID = matches[0].ID
	}
	if agentID == "" {
		return tools.ErrorResult("agent_id or capability is required")
	}

	// Allowlist check: if a checker is set and it denies the handoff, block it.
	if t.allowlistChecker != nil && !t.allowlistChecker.CanHandoff(t.fromAgentID, agentID) {
		return tools.ErrorResult(fmt.Sprintf("handoff from %q to %q not allowed by policy", t.fromAgentID, agentID))
	}

	// Parse optional context map
	var contextMap map[string]string
	if ctxRaw, ok := args["context"].(map[string]any); ok {
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
		Depth:       t.depth,
		Visited:     t.visited,
		MaxDepth:    t.maxDepth,
	}, t.originChannel, t.originChatID)

	if !result.Success {
		return tools.ErrorResult(fmt.Sprintf("Handoff to %q failed: %s", agentID, result.Error))
	}

	return &tools.ToolResult{
		ForLLM:  fmt.Sprintf("Agent %q completed task (iterations: %d):\n%s", agentID, result.Iterations, result.Content),
		ForUser: result.Content,
	}
}
