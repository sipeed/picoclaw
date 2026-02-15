package tools

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MockMCPManager is a mock implementation of MCPManager interface for testing
type MockMCPManager struct {
	callToolFunc func(ctx context.Context, serverName, toolName string, arguments map[string]interface{}) (*mcp.CallToolResult, error)
}

func (m *MockMCPManager) CallTool(ctx context.Context, serverName, toolName string, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	if m.callToolFunc != nil {
		return m.callToolFunc(ctx, serverName, toolName, arguments)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "mock result"},
		},
		IsError: false,
	}, nil
}

// newMCPToolForTest creates an MCP tool for testing with mock manager
func newMCPToolForTest(manager MCPManager, serverName string, tool *mcp.Tool) *MCPTool {
	return &MCPTool{
		manager:    manager,
		serverName: serverName,
		tool:       tool,
	}
}

// TestNewMCPTool verifies MCP tool creation
func TestNewMCPTool(t *testing.T) {
	manager := &MockMCPManager{}
	tool := &mcp.Tool{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"input": map[string]interface{}{
					"type":        "string",
					"description": "Test input",
				},
			},
		},
	}

	mcpTool := newMCPToolForTest(manager, "test_server", tool)

	if mcpTool == nil {
		t.Fatal("NewMCPTool should not return nil")
	}
	// Verify tool properties we can access
	if mcpTool.Name() != "mcp_test_server_test_tool" {
		t.Errorf("Expected tool name with prefix, got '%s'", mcpTool.Name())
	}
}

// TestMCPTool_Name verifies tool name with server prefix
func TestMCPTool_Name(t *testing.T) {
	tests := []struct {
		name       string
		serverName string
		toolName   string
		expected   string
	}{
		{
			name:       "simple name",
			serverName: "github",
			toolName:   "create_issue",
			expected:   "mcp_github_create_issue",
		},
		{
			name:       "filesystem server",
			serverName: "filesystem",
			toolName:   "read_file",
			expected:   "mcp_filesystem_read_file",
		},
		{
			name:       "remote server",
			serverName: "remote-api",
			toolName:   "fetch_data",
			expected:   "mcp_remote-api_fetch_data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &MockMCPManager{}
			tool := &mcp.Tool{Name: tt.toolName}
			mcpTool := newMCPToolForTest(manager, tt.serverName, tool)

			result := mcpTool.Name()
			if result != tt.expected {
				t.Errorf("Expected name '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestMCPTool_Description verifies tool description generation
func TestMCPTool_Description(t *testing.T) {
	tests := []struct {
		name            string
		serverName      string
		toolDescription string
		expectContains  []string
	}{
		{
			name:            "with description",
			serverName:      "github",
			toolDescription: "Create a GitHub issue",
			expectContains:  []string{"[MCP:github]", "Create a GitHub issue"},
		},
		{
			name:            "empty description",
			serverName:      "filesystem",
			toolDescription: "",
			expectContains:  []string{"[MCP:filesystem]", "MCP tool from filesystem server"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &MockMCPManager{}
			tool := &mcp.Tool{
				Name:        "test_tool",
				Description: tt.toolDescription,
			}
			mcpTool := newMCPToolForTest(manager, tt.serverName, tool)

			result := mcpTool.Description()

			for _, expected := range tt.expectContains {
				if !strings.Contains(result, expected) {
					t.Errorf("Description should contain '%s', got: %s", expected, result)
				}
			}
		})
	}
}

// TestMCPTool_Parameters verifies parameter schema conversion
func TestMCPTool_Parameters(t *testing.T) {
	tests := []struct {
		name        string
		inputSchema interface{}
		expectType  string
	}{
		{
			name: "map schema",
			inputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search query",
					},
				},
				"required": []string{"query"},
			},
			expectType: "object",
		},
		{
			name:        "nil schema",
			inputSchema: nil,
			expectType:  "object",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &MockMCPManager{}
			tool := &mcp.Tool{
				Name:        "test_tool",
				InputSchema: tt.inputSchema,
			}
			mcpTool := newMCPToolForTest(manager, "test_server", tool)

			params := mcpTool.Parameters()

			if params == nil {
				t.Fatal("Parameters should not be nil")
			}

			if params["type"] != tt.expectType {
				t.Errorf("Expected type '%s', got '%v'", tt.expectType, params["type"])
			}
		})
	}
}

// TestMCPTool_Execute_Success tests successful tool execution
func TestMCPTool_Execute_Success(t *testing.T) {
	manager := &MockMCPManager{
		callToolFunc: func(ctx context.Context, serverName, toolName string, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
			// Verify correct parameters passed
			if serverName != "github" {
				t.Errorf("Expected serverName 'github', got '%s'", serverName)
			}
			if toolName != "search_repos" {
				t.Errorf("Expected toolName 'search_repos', got '%s'", toolName)
			}

			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: "Found 3 repositories"},
				},
				IsError: false,
			}, nil
		},
	}

	tool := &mcp.Tool{
		Name:        "search_repos",
		Description: "Search GitHub repositories",
	}
	mcpTool := newMCPToolForTest(manager, "github", tool)

	ctx := context.Background()
	args := map[string]interface{}{
		"query": "golang mcp",
	}

	result := mcpTool.Execute(ctx, args)

	if result == nil {
		t.Fatal("Result should not be nil")
	}
	if result.IsError {
		t.Errorf("Expected no error, got error: %s", result.ForLLM)
	}
	if result.ForLLM != "Found 3 repositories" {
		t.Errorf("Expected 'Found 3 repositories', got '%s'", result.ForLLM)
	}
}

// TestMCPTool_Execute_ManagerError tests execution when manager returns error
func TestMCPTool_Execute_ManagerError(t *testing.T) {
	manager := &MockMCPManager{
		callToolFunc: func(ctx context.Context, serverName, toolName string, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
			return nil, fmt.Errorf("connection failed")
		},
	}

	tool := &mcp.Tool{Name: "test_tool"}
	mcpTool := newMCPToolForTest(manager, "test_server", tool)

	ctx := context.Background()
	result := mcpTool.Execute(ctx, map[string]interface{}{})

	if result == nil {
		t.Fatal("Result should not be nil")
	}
	if !result.IsError {
		t.Error("Expected IsError to be true")
	}
	if !strings.Contains(result.ForLLM, "MCP tool execution failed") {
		t.Errorf("Error message should mention execution failure, got: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "connection failed") {
		t.Errorf("Error message should include original error, got: %s", result.ForLLM)
	}
}

// TestMCPTool_Execute_ServerError tests execution when server returns error
func TestMCPTool_Execute_ServerError(t *testing.T) {
	manager := &MockMCPManager{
		callToolFunc: func(ctx context.Context, serverName, toolName string, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: "Invalid API key"},
				},
				IsError: true,
			}, nil
		},
	}

	tool := &mcp.Tool{Name: "test_tool"}
	mcpTool := newMCPToolForTest(manager, "test_server", tool)

	ctx := context.Background()
	result := mcpTool.Execute(ctx, map[string]interface{}{})

	if result == nil {
		t.Fatal("Result should not be nil")
	}
	if !result.IsError {
		t.Error("Expected IsError to be true")
	}
	if !strings.Contains(result.ForLLM, "MCP tool returned error") {
		t.Errorf("Error message should mention server error, got: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "Invalid API key") {
		t.Errorf("Error message should include server message, got: %s", result.ForLLM)
	}
}

// TestMCPTool_Execute_MultipleContent tests execution with multiple content items
func TestMCPTool_Execute_MultipleContent(t *testing.T) {
	manager := &MockMCPManager{
		callToolFunc: func(ctx context.Context, serverName, toolName string, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: "First line"},
					&mcp.TextContent{Text: "Second line"},
					&mcp.TextContent{Text: "Third line"},
				},
				IsError: false,
			}, nil
		},
	}

	tool := &mcp.Tool{Name: "multi_output"}
	mcpTool := newMCPToolForTest(manager, "test_server", tool)

	ctx := context.Background()
	result := mcpTool.Execute(ctx, map[string]interface{}{})

	if result.IsError {
		t.Errorf("Expected no error, got: %s", result.ForLLM)
	}

	expected := "First line\nSecond line\nThird line"
	if result.ForLLM != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result.ForLLM)
	}
}

// TestExtractContentText_TextContent tests text content extraction
func TestExtractContentText_TextContent(t *testing.T) {
	content := []mcp.Content{
		&mcp.TextContent{Text: "Hello World"},
		&mcp.TextContent{Text: "Second message"},
	}

	result := extractContentText(content)
	expected := "Hello World\nSecond message"

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestExtractContentText_ImageContent tests image content extraction
func TestExtractContentText_ImageContent(t *testing.T) {
	content := []mcp.Content{
		&mcp.ImageContent{
			Data:     []byte("base64data"),
			MIMEType: "image/png",
		},
	}

	result := extractContentText(content)

	if !strings.Contains(result, "[Image:") {
		t.Errorf("Expected image indicator, got: %s", result)
	}
	if !strings.Contains(result, "image/png") {
		t.Errorf("Expected MIME type in output, got: %s", result)
	}
}

// TestExtractContentText_MixedContent tests mixed content types
func TestExtractContentText_MixedContent(t *testing.T) {
	content := []mcp.Content{
		&mcp.TextContent{Text: "Description"},
		&mcp.ImageContent{
			Data:     []byte("data"),
			MIMEType: "image/jpeg",
		},
		&mcp.TextContent{Text: "More text"},
	}

	result := extractContentText(content)

	if !strings.Contains(result, "Description") {
		t.Errorf("Should contain text content, got: %s", result)
	}
	if !strings.Contains(result, "[Image:") {
		t.Errorf("Should contain image indicator, got: %s", result)
	}
	if !strings.Contains(result, "More text") {
		t.Errorf("Should contain second text, got: %s", result)
	}
}

// TestExtractContentText_EmptyContent tests empty content array
func TestExtractContentText_EmptyContent(t *testing.T) {
	content := []mcp.Content{}

	result := extractContentText(content)

	if result != "" {
		t.Errorf("Expected empty string for empty content, got: %s", result)
	}
}

// TestMCPTool_InterfaceCompliance verifies MCPTool implements Tool interface
func TestMCPTool_InterfaceCompliance(t *testing.T) {
	manager := &MockMCPManager{}
	tool := &mcp.Tool{Name: "test"}
	mcpTool := newMCPToolForTest(manager, "test_server", tool)

	// Verify it implements Tool interface
	var _ Tool = mcpTool
}

// TestMCPTool_Parameters_MapSchema tests schema that's already a map
func TestMCPTool_Parameters_MapSchema(t *testing.T) {
	manager := &MockMCPManager{}
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type":        "string",
				"description": "The name parameter",
			},
		},
		"required": []string{"name"},
	}

	tool := &mcp.Tool{
		Name:        "test_tool",
		InputSchema: schema,
	}
	mcpTool := newMCPToolForTest(manager, "test_server", tool)

	params := mcpTool.Parameters()

	// Should return the schema as-is when it's already a map
	if params["type"] != "object" {
		t.Errorf("Expected type 'object', got '%v'", params["type"])
	}

	props, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Error("Properties should be a map")
	}

	nameParam, ok := props["name"].(map[string]interface{})
	if !ok {
		t.Error("Name parameter should exist")
	}

	if nameParam["type"] != "string" {
		t.Errorf("Name type should be 'string', got '%v'", nameParam["type"])
	}
}
