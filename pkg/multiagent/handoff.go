package multiagent

import (
	"context"
	"fmt"
	"slices"

	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/tools"
)

// AgentResolver looks up an agent by ID.
// Typically backed by agent.AgentRegistry.GetAgent.
type AgentResolver interface {
	GetAgentInfo(agentID string) *AgentInfo
	ListAgents() []AgentInfo
}

// AllowlistChecker determines whether a handoff from one agent to another is allowed.
type AllowlistChecker interface {
	CanHandoff(fromAgentID, toAgentID string) bool
}

// AllowlistCheckerFunc adapts a function to the AllowlistChecker interface.
type AllowlistCheckerFunc func(fromAgentID, toAgentID string) bool

func (f AllowlistCheckerFunc) CanHandoff(fromAgentID, toAgentID string) bool {
	return f(fromAgentID, toAgentID)
}

// AgentInfo is a minimal view of an agent for handoff purposes,
// decoupled from the full AgentInstance to avoid circular imports.
type AgentInfo struct {
	ID           string
	Name         string
	Role         string
	SystemPrompt string
	Model        string
	Provider     providers.LLMProvider
	Tools        *tools.ToolRegistry
	MaxIter      int
	Capabilities []string // optional tags for capability-based routing (e.g. "coding", "research")
}

// FindAgentsByCapability returns agents that advertise the given capability.
func FindAgentsByCapability(resolver AgentResolver, capability string) []AgentInfo {
	var matches []AgentInfo
	for _, a := range resolver.ListAgents() {
		if slices.Contains(a.Capabilities, capability) {
			matches = append(matches, a)
		}
	}
	return matches
}

// DefaultMaxHandoffDepth is the maximum handoff chain depth when not configured.
const DefaultMaxHandoffDepth = 3

// HandoffRequest describes a delegation from one agent to another.
type HandoffRequest struct {
	FromAgentID string
	ToAgentID   string
	Task        string
	Context     map[string]string // k-v to write to blackboard before handoff
	Depth       int               // current depth level (0 = top-level)
	Visited     []string          // agent IDs already in the call chain
	MaxDepth    int               // max allowed depth (0 = use DefaultMaxHandoffDepth)
}

// HandoffResult contains the outcome of a handoff execution.
type HandoffResult struct {
	AgentID    string
	Content    string
	Iterations int
	Success    bool
	Error      string
}

// ExecuteHandoff delegates a task to a target agent, injecting blackboard context.
// It enforces recursion guards: depth limit and cycle detection.
func ExecuteHandoff(ctx context.Context, resolver AgentResolver, board *Blackboard, req HandoffRequest, channel, chatID string) *HandoffResult {
	// Recursion guard: depth limit
	maxDepth := req.MaxDepth
	if maxDepth == 0 {
		maxDepth = DefaultMaxHandoffDepth
	}
	if req.Depth >= maxDepth {
		return &HandoffResult{
			AgentID: req.ToAgentID,
			Success: false,
			Error:   fmt.Sprintf("handoff depth limit reached (%d/%d): %v -> %s", req.Depth, maxDepth, req.Visited, req.ToAgentID),
		}
	}

	// Recursion guard: cycle detection
	for _, v := range req.Visited {
		if v == req.ToAgentID {
			return &HandoffResult{
				AgentID: req.ToAgentID,
				Success: false,
				Error:   fmt.Sprintf("handoff cycle detected: %q already in chain %v", req.ToAgentID, req.Visited),
			}
		}
	}

	target := resolver.GetAgentInfo(req.ToAgentID)
	if target == nil {
		return &HandoffResult{
			AgentID: req.ToAgentID,
			Success: false,
			Error:   fmt.Sprintf("agent %q not found", req.ToAgentID),
		}
	}

	// Write request context to blackboard
	if board != nil && req.Context != nil {
		for k, v := range req.Context {
			board.Set(k, v, req.FromAgentID)
		}
	}

	// Propagate depth and visited to target agent's handoff tool
	newVisited := make([]string, len(req.Visited)+1)
	copy(newVisited, req.Visited)
	newVisited[len(req.Visited)] = req.ToAgentID

	if target.Tools != nil {
		// Wire session blackboard to target's tools
		if tool, ok := target.Tools.Get("blackboard"); ok {
			if ba, ok := tool.(BoardAware); ok {
				ba.SetBoard(board)
			}
		}
		if tool, ok := target.Tools.Get("handoff"); ok {
			if ba, ok := tool.(BoardAware); ok {
				ba.SetBoard(board)
			}
			if ht, ok := tool.(*HandoffTool); ok {
				ht.depth = req.Depth + 1
				ht.visited = newVisited
				ht.maxDepth = maxDepth
			}
		}
	}

	// Build system prompt incorporating agent role, system prompt, and blackboard
	systemPrompt := buildHandoffSystemPrompt(target, board)

	messages := []providers.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: req.Task},
	}

	maxIter := target.MaxIter
	if maxIter == 0 {
		maxIter = 10
	}

	loopResult, err := tools.RunToolLoop(ctx, tools.ToolLoopConfig{
		Provider:      target.Provider,
		Model:         target.Model,
		Tools:         target.Tools,
		MaxIterations: maxIter,
		LLMOptions: map[string]any{
			"max_tokens":  4096,
			"temperature": 0.7,
		},
	}, messages, channel, chatID)

	if err != nil {
		return &HandoffResult{
			AgentID: req.ToAgentID,
			Success: false,
			Error:   err.Error(),
		}
	}

	return &HandoffResult{
		AgentID:    req.ToAgentID,
		Content:    loopResult.Content,
		Iterations: loopResult.Iterations,
		Success:    true,
	}
}

func buildHandoffSystemPrompt(agent *AgentInfo, board *Blackboard) string {
	prompt := "You are " + agent.Name
	if agent.Role != "" {
		prompt += ", " + agent.Role
	}
	prompt += ".\n"

	if agent.SystemPrompt != "" {
		prompt += "\n" + agent.SystemPrompt + "\n"
	}

	prompt += "\nComplete the delegated task and provide a clear result."

	if board != nil {
		snapshot := board.Snapshot()
		if snapshot != "" {
			prompt += "\n\n" + snapshot
		}
	}

	return prompt
}
