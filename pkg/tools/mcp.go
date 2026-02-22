package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/mcp"
)

// MCPBridgeTool exposes MCP servers to the LLM as a single tool with actions.
type MCPBridgeTool struct {
	manager *mcp.Manager
}

// NewMCPBridgeTool creates a new MCP bridge tool.
func NewMCPBridgeTool(manager *mcp.Manager) *MCPBridgeTool {
	return &MCPBridgeTool{manager: manager}
}

func (t *MCPBridgeTool) Name() string {
	return "mcp"
}

func (t *MCPBridgeTool) Description() string {
	return "Interact with MCP (Model Context Protocol) servers. Actions: mcp_list (list available servers), mcp_tools (get server's tool list), mcp_call (call a server tool), mcp_read_resource (read a resource by URI)"
}

func (t *MCPBridgeTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"description": "The MCP action to perform",
				"enum":        []string{"mcp_list", "mcp_tools", "mcp_call", "mcp_read_resource"},
			},
			"server": map[string]interface{}{
				"type":        "string",
				"description": "MCP server name (required for mcp_tools and mcp_call)",
			},
			"tool": map[string]interface{}{
				"type":        "string",
				"description": "Tool name to call (required for mcp_call)",
			},
			"arguments": map[string]interface{}{
				"type":        "object",
				"description": "Arguments to pass to the tool (for mcp_call)",
			},
			"uri": map[string]interface{}{
				"type":        "string",
				"description": "Resource URI to read (required for mcp_read_resource)",
			},
		},
		"required": []string{"action"},
	}
}

func (t *MCPBridgeTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	action, ok := args["action"].(string)
	if !ok {
		return ErrorResult("action is required")
	}

	switch action {
	case "mcp_list":
		return t.listServers()
	case "mcp_tools":
		return t.getTools(ctx, args)
	case "mcp_call":
		return t.callTool(ctx, args)
	case "mcp_read_resource":
		return t.readResource(ctx, args)
	default:
		return ErrorResult(fmt.Sprintf("unknown action: %s", action))
	}
}

func (t *MCPBridgeTool) listServers() *ToolResult {
	servers := t.manager.ListServers()
	if len(servers) == 0 {
		return SilentResult("No MCP servers configured")
	}

	var sb strings.Builder
	sb.WriteString("Available MCP servers:\n")
	for _, s := range servers {
		sb.WriteString(fmt.Sprintf("- %s: %s [%s]\n", s.Name, s.Description, s.Status))
	}
	return SilentResult(sb.String())
}

func (t *MCPBridgeTool) getTools(ctx context.Context, args map[string]interface{}) *ToolResult {
	server, ok := args["server"].(string)
	if !ok || server == "" {
		return ErrorResult("server is required for mcp_tools")
	}

	tools, err := t.manager.GetTools(ctx, server)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to get tools from %q: %v", server, err))
	}

	if len(tools) == 0 {
		return SilentResult(fmt.Sprintf("Server %q has no tools", server))
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Tools from server %q:\n\n", server))
	for _, tool := range tools {
		sb.WriteString(fmt.Sprintf("## %s\n", tool.Name))
		if tool.Description != "" {
			sb.WriteString(fmt.Sprintf("%s\n", tool.Description))
		}
		if tool.InputSchema != nil {
			schema, _ := json.MarshalIndent(tool.InputSchema, "", "  ")
			sb.WriteString(fmt.Sprintf("Input schema:\n```json\n%s\n```\n", string(schema)))
		}
		sb.WriteString("\n")
	}
	return SilentResult(sb.String())
}

func (t *MCPBridgeTool) callTool(ctx context.Context, args map[string]interface{}) *ToolResult {
	server, ok := args["server"].(string)
	if !ok || server == "" {
		return ErrorResult("server is required for mcp_call")
	}

	toolName, ok := args["tool"].(string)
	if !ok || toolName == "" {
		return ErrorResult("tool is required for mcp_call")
	}

	// Extract arguments (optional)
	var toolArgs map[string]interface{}
	if a, ok := args["arguments"]; ok && a != nil {
		if m, ok := a.(map[string]interface{}); ok {
			toolArgs = m
		}
	}

	result, err := t.manager.CallTool(ctx, server, toolName, toolArgs)
	if err != nil {
		return ErrorResult(fmt.Sprintf("mcp_call %s/%s failed: %v", server, toolName, err))
	}

	return SilentResult(result)
}

func (t *MCPBridgeTool) readResource(ctx context.Context, args map[string]interface{}) *ToolResult {
	server, ok := args["server"].(string)
	if !ok || server == "" {
		return ErrorResult("server is required for mcp_read_resource")
	}

	uri, ok := args["uri"].(string)
	if !ok || uri == "" {
		return ErrorResult("uri is required for mcp_read_resource")
	}

	result, err := t.manager.ReadResource(ctx, server, uri)
	if err != nil {
		return ErrorResult(fmt.Sprintf("mcp_read_resource %s/%s failed: %v", server, uri, err))
	}

	return SilentResult(result)
}
