package tools

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSplitQuoted(t *testing.T) {
	cmdStr := `--mcp-stdio "npx alpaca-mcp-server" --list`
	parts := splitQuoted(cmdStr)
	assert.Equal(t, []string{"--mcp-stdio", "npx alpaca-mcp-server", "--list"}, parts)

	cmdStr = `--mcp-stdio "npx alpaca-mcp-server" my-tool --param1 "value 1" --param2 value2`
	parts = splitQuoted(cmdStr)
	assert.Equal(t, []string{"--mcp-stdio", "npx alpaca-mcp-server", "my-tool", "--param1", "value 1", "--param2", "value2"}, parts)
}

func TestMCP2CliToolExecuteValidation(t *testing.T) {
	tool := NewMCP2CliTool(nil)

	// Test missing command
	result := tool.Execute(context.Background(), map[string]any{})
	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "command parameter is required")

	// Test invalid source
	result = tool.Execute(context.Background(), map[string]any{
		"command": "--list",
	})
	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "source is required")

	// Test error connecting
	result = tool.Execute(context.Background(), map[string]any{
		"command": "--mcp-stdio non_existent_cmd",
	})
	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "failed to connect")
}
