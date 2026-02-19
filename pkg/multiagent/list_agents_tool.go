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

// Name returns the tool name.
func (t *ListAgentsTool) Name() string { return "list_agents" }

// Description returns a human-readable description of the tool.
func (t *ListAgentsTool) Description() string {
	return "List all available agents with their IDs, names, and roles."
}

// Parameters returns the JSON Schema for the tool's input.
func (t *ListAgentsTool) Parameters() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

// Execute lists all registered agents with their metadata.
func (t *ListAgentsTool) Execute(_ context.Context, _ map[string]any) *tools.ToolResult {
	agents := t.resolver.ListAgents()
	if len(agents) == 0 {
		return tools.NewToolResult("No agents registered.")
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Available agents (%d):\n", len(agents))
	for _, a := range agents {
		fmt.Fprintf(&sb, "- ID: %s", a.ID)
		if a.Name != "" {
			fmt.Fprintf(&sb, ", Name: %s", a.Name)
		}
		if a.Role != "" {
			fmt.Fprintf(&sb, ", Role: %s", a.Role)
		}
		sb.WriteString("\n")
	}
	return tools.NewToolResult(sb.String())
}
