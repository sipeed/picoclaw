package multiagent

import (
	"context"
	"fmt"

	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/tools"
)

// AgentResolver looks up an agent by ID.
// Typically backed by agent.AgentRegistry.GetAgent.
type AgentResolver interface {
	GetAgentInfo(agentID string) *AgentInfo
	ListAgents() []AgentInfo
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
}

// HandoffRequest describes a delegation from one agent to another.
type HandoffRequest struct {
	FromAgentID string
	ToAgentID   string
	Task        string
	Context     map[string]string // k-v to write to blackboard before handoff
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
func ExecuteHandoff(ctx context.Context, resolver AgentResolver, board *Blackboard, req HandoffRequest, channel, chatID string) *HandoffResult {
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
		LLMOptions: map[string]interface{}{
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
