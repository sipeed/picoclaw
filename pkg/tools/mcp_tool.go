package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPManager defines the interface for MCP manager operations
// This allows for easier testing with mock implementations
type MCPManager interface {
	CallTool(
		ctx context.Context,
		serverName, toolName string,
		arguments map[string]any,
	) (*mcp.CallToolResult, error)
}

// MCPTool wraps an MCP tool to implement the Tool interface
type MCPTool struct {
	manager    MCPManager
	serverName string
	tool       *mcp.Tool
}

// NewMCPTool creates a new MCP tool wrapper
func NewMCPTool(manager MCPManager, serverName string, tool *mcp.Tool) *MCPTool {
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
func (t *MCPTool) Parameters() map[string]any {
	// The InputSchema is already a JSON Schema object
	schema := t.tool.InputSchema

	// Handle nil schema
	if schema == nil {
		return map[string]any{
			"type":       "object",
			"properties": map[string]any{},
			"required":   []string{},
		}
	}

	// Try direct conversion first (fast path)
	if schemaMap, ok := schema.(map[string]any); ok {
		return schemaMap
	}

	// Handle json.RawMessage and []byte - unmarshal directly
	var jsonData []byte
	if rawMsg, ok := schema.(json.RawMessage); ok {
		jsonData = rawMsg
	} else if bytes, ok := schema.([]byte); ok {
		jsonData = bytes
	}

	if jsonData != nil {
		var result map[string]any
		if err := json.Unmarshal(jsonData, &result); err == nil {
			return result
		}
		// Fallback on error
		return map[string]any{
			"type":       "object",
			"properties": map[string]any{},
			"required":   []string{},
		}
	}

	// For other types (structs, etc.), convert via JSON marshal/unmarshal
	var err error
	jsonData, err = json.Marshal(schema)
	if err != nil {
		// Fallback to empty schema if marshaling fails
		return map[string]any{
			"type":       "object",
			"properties": map[string]any{},
			"required":   []string{},
		}
	}

	var result map[string]any
	if err := json.Unmarshal(jsonData, &result); err != nil {
		// Fallback to empty schema if unmarshaling fails
		return map[string]any{
			"type":       "object",
			"properties": map[string]any{},
			"required":   []string{},
		}
	}

	return result
}

// Execute executes the MCP tool
func (t *MCPTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	result, err := t.manager.CallTool(ctx, t.serverName, t.tool.Name, args)
	if err != nil {
		return ErrorResult(fmt.Sprintf("MCP tool execution failed: %v", err)).WithError(err)
	}

	if result == nil {
		nilErr := fmt.Errorf("MCP tool returned nil result without error")
		return ErrorResult("MCP tool execution failed: nil result").WithError(nilErr)
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
