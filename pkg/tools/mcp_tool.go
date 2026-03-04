package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
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

// sanitizeIdentifierComponent normalizes a string so it can be safely used
// as part of a tool/function identifier for downstream providers.
// It:
//   - lowercases the string
//   - replaces any character not in [a-z0-9_-] with '_'
//   - collapses multiple consecutive '_' into a single '_'
//   - trims leading/trailing '_'
//   - falls back to "unnamed" if the result is empty
//   - truncates overly long components to a reasonable length
func sanitizeIdentifierComponent(s string) string {
	const maxLen = 64

	s = strings.ToLower(s)
	var b strings.Builder
	b.Grow(len(s))

	prevUnderscore := false
	for _, r := range s {
		isAllowed := (r >= 'a' && r <= 'z') ||
			(r >= '0' && r <= '9') ||
			r == '_' || r == '-'

		if !isAllowed {
			// Normalize any disallowed character to '_'
			if !prevUnderscore {
				b.WriteRune('_')
				prevUnderscore = true
			}
			continue
		}

		if r == '_' {
			if prevUnderscore {
				continue
			}
			prevUnderscore = true
		} else {
			prevUnderscore = false
		}

		b.WriteRune(r)
	}

	result := strings.Trim(b.String(), "_")
	if result == "" {
		result = "unnamed"
	}

	if len(result) > maxLen {
		result = result[:maxLen]
	}

	return result
}

// Name returns the tool name, prefixed with the server name.
// The total length is capped at 64 characters (OpenAI-compatible API limit).
// A short hash of the original (unsanitized) server and tool names is appended
// whenever sanitization is lossy or the name is truncated, ensuring that two
// names which differ only in disallowed characters remain distinct after sanitization.
func (t *MCPTool) Name() string {
	// Prefix with server name to avoid conflicts, and sanitize components
	sanitizedServer := sanitizeIdentifierComponent(t.serverName)
	sanitizedTool := sanitizeIdentifierComponent(t.tool.Name)
	full := fmt.Sprintf("mcp_%s_%s", sanitizedServer, sanitizedTool)

	// Check if sanitization was lossless (only lowercasing, no char replacement/truncation)
	lossless := strings.ToLower(t.serverName) == sanitizedServer &&
		strings.ToLower(t.tool.Name) == sanitizedTool

	const maxTotal = 64
	if lossless && len(full) <= maxTotal {
		return full
	}

	// Sanitization was lossy or name too long: append hash of the ORIGINAL names
	// (not the sanitized names) so different originals always yield different hashes.
	h := fnv.New32a()
	_, _ = h.Write([]byte(t.serverName + "\x00" + t.tool.Name))
	suffix := fmt.Sprintf("%08x", h.Sum32()) // 8 chars

	base := full
	if len(base) > maxTotal-9 {
		base = strings.TrimRight(full[:maxTotal-9], "_")
	}
	return base + "_" + suffix
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
