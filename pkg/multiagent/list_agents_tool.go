package multiagent

import (
	"context"
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/tools"
)

// ListAgentsTool allows the LLM to discover all available agents.
type ListAgentsTool struct {
	resolver AgentResolver
}

// NewListAgentsTool creates a discovery tool backed by an AgentResolver.
func NewListAgentsTool(resolver AgentResolver) *ListAgentsTool {
	return &ListAgentsTool{resolver: resolver}
}

func (t *ListAgentsTool) Name() string { return "list_agents" }

func (t *ListAgentsTool) Description() string {
	return "List all available agents with their IDs, names, and roles."
}

func (t *ListAgentsTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

func (t *ListAgentsTool) Execute(_ context.Context, _ map[string]interface{}) *tools.ToolResult {
	agents := t.resolver.ListAgents()
	if len(agents) == 0 {
		return tools.NewToolResult("No agents registered.")
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Available agents (%d):\n", len(agents)))
	for _, a := range agents {
		sb.WriteString(fmt.Sprintf("- ID: %s", a.ID))
		if a.Name != "" {
			sb.WriteString(fmt.Sprintf(", Name: %s", a.Name))
		}
		if a.Role != "" {
			sb.WriteString(fmt.Sprintf(", Role: %s", a.Role))
		}
		sb.WriteString("\n")
	}
	return tools.NewToolResult(sb.String())
}
