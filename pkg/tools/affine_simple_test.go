package tools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAffineSimpleTool_Name(t *testing.T) {
	tool := NewAffineSimpleTool(AffineSimpleToolOptions{
		MCPEndpoint:    "https://app.affine.pro/api/workspaces/test/mcp",
		APIKey:         "test-key",
		WorkspaceID:    "test-workspace",
		TimeoutSeconds: 30,
	})

	assert.Equal(t, "affine", tool.Name())
}

func TestAffineSimpleTool_Description(t *testing.T) {
	tool := NewAffineSimpleTool(AffineSimpleToolOptions{
		MCPEndpoint:    "https://app.affine.pro/api/workspaces/test/mcp",
		APIKey:         "test-key",
		WorkspaceID:    "test-workspace",
		TimeoutSeconds: 30,
	})

	desc := tool.Description()
	assert.Contains(t, desc, "Affine")
	assert.Contains(t, desc, "workspace")
}

func TestAffineSimpleTool_Parameters(t *testing.T) {
	tool := NewAffineSimpleTool(AffineSimpleToolOptions{
		MCPEndpoint:    "https://app.affine.pro/api/workspaces/test/mcp",
		APIKey:         "test-key",
		WorkspaceID:    "test-workspace",
		TimeoutSeconds: 30,
	})

	params := tool.Parameters()
	assert.NotNil(t, params)

	// Check required fields
	assert.Equal(t, "object", params["type"])
	props, ok := params["properties"].(map[string]any)
	assert.True(t, ok)
	assert.NotNil(t, props["action"])

	// Check action enum
	actionProp, ok := props["action"].(map[string]any)
	assert.True(t, ok)
	enum, ok := actionProp["enum"].([]string)
	assert.True(t, ok)
	assert.Contains(t, enum, "search")
	assert.Contains(t, enum, "semantic_search")
	assert.Contains(t, enum, "read")
}

func TestAffineSimpleTool_Execute_MissingAction(t *testing.T) {
	tool := NewAffineSimpleTool(AffineSimpleToolOptions{
		MCPEndpoint:    "https://app.affine.pro/api/workspaces/test/mcp",
		APIKey:         "test-key",
		WorkspaceID:    "test-workspace",
		TimeoutSeconds: 30,
	})

	result := tool.Execute(context.Background(), map[string]any{})
	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "action is required")
}

func TestAffineSimpleTool_Execute_UnknownAction(t *testing.T) {
	tool := NewAffineSimpleTool(AffineSimpleToolOptions{
		MCPEndpoint:    "https://app.affine.pro/api/workspaces/test/mcp",
		APIKey:         "test-key",
		WorkspaceID:    "test-workspace",
		TimeoutSeconds: 30,
	})

	result := tool.Execute(context.Background(), map[string]any{
		"action": "invalid_action",
	})
	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "unknown action")
}

func TestAffineSimpleTool_Execute_SearchMissingQuery(t *testing.T) {
	tool := NewAffineSimpleTool(AffineSimpleToolOptions{
		MCPEndpoint:    "https://app.affine.pro/api/workspaces/test/mcp",
		APIKey:         "test-key",
		WorkspaceID:    "test-workspace",
		TimeoutSeconds: 30,
	})

	result := tool.Execute(context.Background(), map[string]any{
		"action": "search",
	})
	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "query is required")
}

func TestAffineSimpleTool_Execute_ReadMissingQuery(t *testing.T) {
	tool := NewAffineSimpleTool(AffineSimpleToolOptions{
		MCPEndpoint:    "https://app.affine.pro/api/workspaces/test/mcp",
		APIKey:         "test-key",
		WorkspaceID:    "test-workspace",
		TimeoutSeconds: 30,
	})

	result := tool.Execute(context.Background(), map[string]any{
		"action": "read",
	})
	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "query is required")
}

func TestAffineSimpleTool_Execute_SemanticSearchMissingQuery(t *testing.T) {
	tool := NewAffineSimpleTool(AffineSimpleToolOptions{
		MCPEndpoint:    "https://app.affine.pro/api/workspaces/test/mcp",
		APIKey:         "test-key",
		WorkspaceID:    "test-workspace",
		TimeoutSeconds: 30,
	})

	result := tool.Execute(context.Background(), map[string]any{
		"action": "semantic_search",
	})
	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "query is required")
}

// Note: Integration tests against a real Affine MCP endpoint would require:
// - Valid MCP endpoint URL
// - Valid API key
// - Valid workspace ID
// These tests focus on parameter validation and error handling
