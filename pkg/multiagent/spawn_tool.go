package multiagent

import (
	"context"
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/tools"
)

// SpawnTool allows an LLM agent to asynchronously spawn a child agent.
// Unlike HandoffTool (synchronous, blocking), SpawnTool returns immediately
// with a run ID. Results are auto-announced back to the parent session.
//
// Pattern: Anthropic's orchestrator-workers + OpenAI Swarm's lightweight handoffs.
type SpawnTool struct {
	resolver         AgentResolver
	board            *Blackboard
	spawnManager     *SpawnManager
	fromAgentID      string
	originChannel    string
	originChatID     string
	depth            int
	visited          []string
	maxDepth         int
	parentSessionKey string
	allowlistChecker AllowlistChecker
}

// NewSpawnTool creates a spawn tool bound to a source agent.
func NewSpawnTool(resolver AgentResolver, board *Blackboard, spawnManager *SpawnManager, fromAgentID string) *SpawnTool {
	return &SpawnTool{
		resolver:      resolver,
		board:         board,
		spawnManager:  spawnManager,
		fromAgentID:   fromAgentID,
		originChannel: "cli",
		originChatID:  "direct",
	}
}

func (t *SpawnTool) Name() string { return "spawn_agent" }

func (t *SpawnTool) Description() string {
	agents := t.resolver.ListAgents()
	if len(agents) <= 1 {
		return "Spawn a child agent asynchronously. Returns immediately — result auto-announces back. No other agents currently available."
	}

	var sb strings.Builder
	sb.WriteString("Spawn a child agent asynchronously. Returns immediately with a run ID — the result will auto-announce back when complete. Available agents:\n")
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
		sb.WriteString("\n")
	}
	sb.WriteString("\nUse 'list_spawns' to check status. Results are auto-delivered — no need to poll.")
	return sb.String()
}

func (t *SpawnTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"agent_id": map[string]any{
				"type":        "string",
				"description": "The ID of the agent to spawn",
			},
			"capability": map[string]any{
				"type":        "string",
				"description": "Route to an agent with this capability instead of by ID",
			},
			"task": map[string]any{
				"type":        "string",
				"description": "The task for the spawned agent",
			},
			"context": map[string]any{
				"type":        "object",
				"description": "Optional key-value context to share via blackboard",
			},
		},
		"required": []string{"task"},
	}
}

// SetBoard implements BoardAware.
func (t *SpawnTool) SetBoard(board *Blackboard) {
	t.board = board
}

// SetContext implements ContextualTool.
func (t *SpawnTool) SetContext(channel, chatID string) {
	t.originChannel = channel
	t.originChatID = chatID
}

// SetAllowlistChecker sets the allowlist checker for spawn permissions.
func (t *SpawnTool) SetAllowlistChecker(checker AllowlistChecker) {
	t.allowlistChecker = checker
}

// SetRunRegistry sets registry and parent key for cascade tracking.
func (t *SpawnTool) SetRunRegistry(registry *RunRegistry, parentSessionKey string) {
	t.parentSessionKey = parentSessionKey
}

func (t *SpawnTool) Execute(ctx context.Context, args map[string]any) *tools.ToolResult {
	agentID, _ := args["agent_id"].(string)
	capability, _ := args["capability"].(string)
	task, _ := args["task"].(string)

	if task == "" {
		return tools.ErrorResult("task is required")
	}

	// Resolve agent
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

	// Allowlist check
	if t.allowlistChecker != nil && !t.allowlistChecker.CanHandoff(t.fromAgentID, agentID) {
		return tools.ErrorResult(fmt.Sprintf("spawn from %q to %q not allowed by policy", t.fromAgentID, agentID))
	}

	// Parse context
	var contextMap map[string]string
	if ctxRaw, ok := args["context"].(map[string]any); ok {
		contextMap = make(map[string]string, len(ctxRaw))
		for k, v := range ctxRaw {
			contextMap[k] = fmt.Sprintf("%v", v)
		}
	}

	result := t.spawnManager.AsyncSpawn(ctx, t.resolver, t.board, SpawnRequest{
		FromAgentID:  t.fromAgentID,
		ToAgentID:    agentID,
		Task:         task,
		Context:      contextMap,
		Depth:        t.depth,
		Visited:      t.visited,
		MaxDepth:     t.maxDepth,
		ParentRunKey: t.parentSessionKey,
	}, t.originChannel, t.originChatID)

	if result.Status != "accepted" {
		return tools.ErrorResult(fmt.Sprintf("Spawn rejected: %s", result.Error))
	}

	return &tools.ToolResult{
		ForLLM: fmt.Sprintf("Agent %q spawned (run_id: %s). It runs asynchronously — the result will auto-announce back when complete. Continue with other work.", agentID, result.RunID),
		ForUser: fmt.Sprintf("Spawned agent %q (run: %s)", agentID, result.RunID),
	}
}
