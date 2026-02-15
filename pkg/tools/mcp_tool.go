package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpPkg "github.com/sipeed/picoclaw/pkg/mcp"
)

// MCPManager defines the interface for MCP manager operations
// This allows for easier testing with mock implementations
type MCPManager interface {
	CallTool(ctx context.Context, serverName, toolName string, arguments map[string]interface{}) (*mcp.CallToolResult, error)
}

// MCPTool wraps an MCP tool to implement the Tool interface
type MCPTool struct {
	manager    MCPManager
	serverName string
	tool       *mcp.Tool
}

// NewMCPTool creates a new MCP tool wrapper
func NewMCPTool(manager *mcpPkg.Manager, serverName string, tool *mcp.Tool) *MCPTool {
	return &MCPTool{
		manager:    manager,
		serverName: serverName,
		tool:       tool,
	}
}

// Name returns the tool name, prefixed with the server name
func (t *MCPTool) Name() string {
	// Prefix with server name to avoid conflicts
	return fmt.Sprintf("mcp_%s_%s", t.serverName, t.tool.Name)
}

// Description returns the tool description
func (t *MCPTool) Description() string {
	desc := t.tool.Description
	if desc == "" {
		desc = fmt.Sprintf("MCP tool from %s server", t.serverName)
	}
	// Add server info to description
	return fmt.Sprintf("[MCP:%s] %s", t.serverName, desc)
}

// Parameters returns the tool parameters schema
func (t *MCPTool) Parameters() map[string]interface{} {
	// The InputSchema is already a JSON Schema object
	schema := t.tool.InputSchema

	// Convert to map[string]interface{} for compatibility
	result := make(map[string]interface{})

	// Use reflection to convert the schema
	// The schema should already be in the correct format
	if schema != nil {
		// Attempt to convert directly
		if schemaMap, ok := schema.(map[string]interface{}); ok {
			return schemaMap
		}

		// Otherwise, build it manually
		result["type"] = "object"
		result["properties"] = map[string]interface{}{}
		result["required"] = []string{}
	} else {
		// Default schema when nil
		result["type"] = "object"
		result["properties"] = map[string]interface{}{}
		result["required"] = []string{}
	}

	return result
}

// Execute executes the MCP tool
func (t *MCPTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	result, err := t.manager.CallTool(ctx, t.serverName, t.tool.Name, args)
	if err != nil {
		return ErrorResult(fmt.Sprintf("MCP tool execution failed: %v", err)).WithError(err)
	}

	// Handle error result from server
	if result.IsError {
		errMsg := extractContentText(result.Content)
		return ErrorResult(fmt.Sprintf("MCP tool returned error: %s", errMsg)).
			WithError(fmt.Errorf("MCP tool error: %s", errMsg))
	}

	// Extract text content from result
	output := extractContentText(result.Content)

	return &ToolResult{
		ForLLM:  output,
		IsError: false,
	}
}

// extractContentText extracts text from MCP content array
func extractContentText(content []mcp.Content) string {
	var parts []string
	for _, c := range content {
		switch v := c.(type) {
		case *mcp.TextContent:
			parts = append(parts, v.Text)
		case *mcp.ImageContent:
			// For images, just indicate that an image was returned
			parts = append(parts, fmt.Sprintf("[Image: %s]", v.MIMEType))
		default:
			// For other content types, use string representation
			parts = append(parts, fmt.Sprintf("[Content: %T]", v))
		}
	}
	return strings.Join(parts, "\n")
}
