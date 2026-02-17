package tools

import (
	"context"
	"errors"
	"fmt"

	"github.com/sipeed/picoclaw/pkg/mcp"
)

type MCPTool struct {
	manager     *mcp.Manager
	name        string
	description string
	parameters  map[string]any
}

func NewMCPTool(manager *mcp.Manager, tool mcp.RegisteredTool) *MCPTool {
	description := tool.Description
	if description == "" {
		description = fmt.Sprintf("MCP tool %s from server %s", tool.ToolName, tool.ServerName)
	}

	return &MCPTool{
		manager:     manager,
		name:        tool.QualifiedName,
		description: description,
		parameters:  tool.Parameters,
	}
}

func (t *MCPTool) Name() string {
	return t.name
}

func (t *MCPTool) Description() string {
	return t.description
}

func (t *MCPTool) Parameters() map[string]interface{} {
	return t.parameters
}

func (t *MCPTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	if t.manager == nil {
		return ErrorResult("MCP manager is not configured")
	}

	result, err := t.manager.CallTool(ctx, t.name, args)
	if err != nil {
		return ErrorResult(fmt.Sprintf("MCP tool %s failed: %v", t.name, err)).WithError(err)
	}
	if result.IsError {
		err := errors.New(result.Content)
		return ErrorResult(result.Content).WithError(err)
	}
	return SilentResult(result.Content)
}

// RegisterMCPTools discovers tools from MCP servers and registers them into the registry.
func RegisterMCPTools(ctx context.Context, registry *ToolRegistry, manager *mcp.Manager) (int, error) {
	if registry == nil || manager == nil {
		return 0, nil
	}

	discoveredTools, err := manager.DiscoverTools(ctx)
	if err != nil {
		return 0, err
	}

	return RegisterKnownMCPTools(registry, manager, discoveredTools), nil
}

// RegisterKnownMCPTools registers already-discovered MCP tools.
// This avoids repeated discovery work when multiple registries share one manager.
func RegisterKnownMCPTools(registry *ToolRegistry, manager *mcp.Manager, discoveredTools []mcp.RegisteredTool) int {
	if registry == nil || manager == nil || len(discoveredTools) == 0 {
		return 0
	}

	for _, tool := range discoveredTools {
		registry.Register(NewMCPTool(manager, tool))
	}
	return len(discoveredTools)
}
