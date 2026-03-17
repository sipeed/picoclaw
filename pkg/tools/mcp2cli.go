package tools

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"jane/pkg/config"
	janemcp "jane/pkg/mcp"
)

// MCP2CliTool provides a single CLI interface for MCP servers,
// saving tokens by exposing APIs dynamically instead of large JSON schemas upfront.
type MCP2CliTool struct {
	manager *janemcp.Manager
	mu      sync.Mutex
}

func NewMCP2CliTool(manager *janemcp.Manager) *MCP2CliTool {
	if manager == nil {
		manager = janemcp.NewManager()
	}
	return &MCP2CliTool{
		manager: manager,
	}
}

func (t *MCP2CliTool) Name() string {
	return "mcp2cli"
}

func (t *MCP2CliTool) Description() string {
	return `Turn any MCP server into a CLI — at runtime, with zero codegen.
Usage:
  mcp2cli --mcp-stdio "npx my-mcp-server" --list
  mcp2cli --mcp-stdio "npx my-mcp-server" --env API_KEY=abc my-tool --param1 "value"
  mcp2cli --mcp "http://localhost:8080/sse" --list
`
}

func (t *MCP2CliTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "The mcp2cli command to run, e.g., '--mcp-stdio \"uvx alpaca-mcp-server\" --list'",
			},
		},
		"required": []string{"command"},
	}
}

// mcp2cliArgs parses the string sent by the agent.
// Instead of a full `kong` CLI struct, we'll implement a custom parser
// to handle dynamic tool names and arguments after the global flags.
func (t *MCP2CliTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	cmdStr, ok := args["command"].(string)
	if !ok || cmdStr == "" {
		return ErrorResult("command parameter is required")
	}

	return t.executeCmd(ctx, cmdStr)
}

func (t *MCP2CliTool) executeCmd(ctx context.Context, cmdStr string) *ToolResult {
	// Simple shell-like splitting
	parts := splitQuoted(cmdStr)

	var (
		mcpURL     string
		mcpStdio   string
		envVars    []string
		authHeader string
		isList     bool
		toolName   string
		toolArgs   []string
	)

	// Parse arguments manually to handle dynamic tools
	i := 0
	for i < len(parts) {
		arg := parts[i]
		if arg == "--mcp" && i+1 < len(parts) {
			mcpURL = parts[i+1]
			i += 2
		} else if arg == "--mcp-stdio" && i+1 < len(parts) {
			mcpStdio = parts[i+1]
			i += 2
		} else if arg == "--env" && i+1 < len(parts) {
			envVars = append(envVars, parts[i+1])
			i += 2
		} else if arg == "--env-file" && i+1 < len(parts) {
			i += 2 // handled below
		} else if arg == "--auth-header" && i+1 < len(parts) {
			authHeader = parts[i+1]
			i += 2
		} else if arg == "--list" {
			isList = true
			i++
		} else if strings.HasPrefix(arg, "--") {
			// Unrecognized flag before tool name, assume it's part of tool args if toolName is set
			if toolName != "" {
				toolArgs = append(toolArgs, arg)
			}
			i++
		} else {
			if toolName == "" {
				toolName = arg
			} else {
				toolArgs = append(toolArgs, arg)
			}
			i++
		}
	}

	if mcpURL == "" && mcpStdio == "" {
		return ErrorResult("source is required: --mcp URL or --mcp-stdio CMD")
	}

	// Build server config
	serverCfg := config.MCPServerConfig{
		Enabled: true,
	}

	serverKey := ""
	if mcpStdio != "" {
		// e.g. "uvx alpaca-mcp-server"
		cmdParts := splitQuoted(mcpStdio)
		if len(cmdParts) == 0 {
			return ErrorResult("invalid --mcp-stdio command")
		}
		serverCfg.Command = cmdParts[0]
		if len(cmdParts) > 1 {
			serverCfg.Args = cmdParts[1:]
		}
		serverCfg.Type = "stdio"
		serverKey = mcpStdio

		serverCfg.Env = make(map[string]string)
		for _, e := range envVars {
			idx := strings.Index(e, "=")
			if idx > 0 {
				serverCfg.Env[e[:idx]] = e[idx+1:]
			}
		}

		// Also look for --env-file in parts and set it if present
		// This wasn't fully parsed in the loop, let's extract it now if possible.
		for i, arg := range parts {
			if arg == "--env-file" && i+1 < len(parts) {
				serverCfg.EnvFile = parts[i+1]
			}
		}
	} else {
		serverCfg.URL = mcpURL
		serverCfg.Type = "sse"
		serverKey = mcpURL
		if authHeader != "" {
			serverCfg.Headers = map[string]string{
				"Authorization": authHeader,
			}
		}
	}

	t.mu.Lock()
	_, exists := t.manager.GetServer(serverKey)
	if !exists {
		// Initialize the connection
		err := t.manager.ConnectServer(ctx, serverKey, serverCfg)
		if err != nil {
			t.mu.Unlock()
			return ErrorResult(fmt.Sprintf("failed to connect to MCP server: %v", err))
		}
	}
	t.mu.Unlock()

	server, _ := t.manager.GetServer(serverKey)

	// Action: List tools
	if isList {
		var b strings.Builder
		b.WriteString("Available commands:\n")
		for _, tool := range server.Tools {
			b.WriteString(fmt.Sprintf("  %s - %s\n", tool.Name, tool.Description))
		}
		return &ToolResult{
			ForLLM:  b.String(),
			IsError: false,
		}
	}

	// Action: Execute tool
	if toolName == "" {
		return ErrorResult("no tool or --list specified")
	}

	// We need to parse toolArgs (which look like --param1 val1 --param2 val2) into a map
	var mcpArgs = make(map[string]any)
	j := 0
	for j < len(toolArgs) {
		arg := toolArgs[j]
		if strings.HasPrefix(arg, "--") {
			key := strings.TrimPrefix(arg, "--")
			if j+1 < len(toolArgs) && !strings.HasPrefix(toolArgs[j+1], "--") {
				mcpArgs[key] = toolArgs[j+1]
				j += 2
			} else {
				mcpArgs[key] = true // boolean flag
				j++
			}
		} else {
			j++
		}
	}

	// For more complex nested JSON arguments, one might pass --json '{"nested": ...}'
	// As a fallback for raw JSON input, standard to mcp2cli if needed.

	result, err := t.manager.CallTool(ctx, serverKey, toolName, mcpArgs)
	if err != nil {
		return ErrorResult(fmt.Sprintf("tool execution failed: %v", err))
	}

	if result.IsError {
		return ErrorResult(fmt.Sprintf("tool returned error: %s", extractContentTextLocal(result.Content)))
	}

	return &ToolResult{
		ForLLM:  extractContentTextLocal(result.Content),
		IsError: false,
	}
}

// extractContentTextLocal extracts text from MCP content array.
// Redefined locally in case extractContentText isn't exported from mcp_tool.go
func extractContentTextLocal(content []mcp.Content) string {
	var parts []string
	for _, c := range content {
		switch v := c.(type) {
		case *mcp.TextContent:
			parts = append(parts, v.Text)
		case *mcp.ImageContent:
			parts = append(parts, fmt.Sprintf("[Image: %s]", v.MIMEType))
		default:
			parts = append(parts, fmt.Sprintf("[Content: %T]", v))
		}
	}
	return strings.Join(parts, "\n")
}

// splitQuoted splits a string by space but keeps quoted strings together
func splitQuoted(s string) []string {
	var parts []string
	var current strings.Builder
	var inQuotes bool
	var quoteChar rune

	for _, r := range s {
		if r == '"' || r == '\'' {
			if inQuotes && quoteChar == r {
				inQuotes = false
			} else if !inQuotes {
				inQuotes = true
				quoteChar = r
			} else {
				current.WriteRune(r)
			}
		} else if r == ' ' && !inQuotes {
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		} else {
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}
