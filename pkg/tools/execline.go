package tools

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/tools/shell"
)

// ExeclineTool executes commands using execlineb instead of shell
// Security: execlineb does not support variable expansion ($VAR) or command
// substitution $(cmd) by default, reducing attack surface significantly.
type ExeclineTool struct {
	config *config.Config
}

// NewExeclineTool creates a new ExeclineTool instance
func NewExeclineTool(cfg *config.Config) *ExeclineTool {
	return &ExeclineTool{
		config: cfg,
	}
}

// Name returns the name of the tool
func (t *ExeclineTool) Name() string {
	return "execline"
}

// Description returns the tool description
func (t *ExeclineTool) Description() string {
	return `Execute commands using execlineb - a secure, minimal shell that does not expand variables or command substitution. Use for simple commands without shell features.`
}

// Parameters returns the tool parameters
func (t *ExeclineTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "The command to execute (passed as-is, no shell expansion)",
			},
		},
		"required": []string{"command"},
	}
}

// Execute runs a command using execlineb
// Commands are executed as-is without shell expansion
func (t *ExeclineTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	// Extract command from args
	cmdArg, ok := args["command"]
	if !ok {
		return &ToolResult{
			ForLLM:  "Missing 'command' argument",
			IsError: true,
			Err:     fmt.Errorf("missing 'command' argument"),
		}
	}

	// Get the command string
	command, ok := cmdArg.(string)
	if !ok {
		return &ToolResult{
			ForLLM:  "'command' must be a string",
			IsError: true,
			Err:     fmt.Errorf("'command' must be a string"),
		}
	}

	if command == "" {
		return &ToolResult{
			ForLLM:  "Empty command",
			IsError: true,
			Err:     fmt.Errorf("empty command"),
		}
	}

	// Check for dangerous patterns in the command
	if err := t.validateCommand(command); err != nil {
		return &ToolResult{
			ForLLM:  fmt.Sprintf("Validation error: %v", err),
			IsError: true,
			Err:     fmt.Errorf("validation error: %w", err),
		}
	}

	// Build safe environment
	baseEnv := shell.WithAllowedEnv(nil, nil)
	var extraEnv map[string]string
	if t.config != nil {
		extraEnv = t.config.Tools.Exec.EnvSet
	}
	execEnv := shell.MergeEnvVars(baseEnv, nil, extraEnv)

	// Use execlineb to execute
	// execlineb -c takes a command string and executes it
	// Unlike sh -c, it doesn't expand $VAR or $(cmd)
	cmd := exec.CommandContext(ctx, "/usr/bin/execlineb", "-c", command)
	cmd.Env = shell.MapToEnvSlice(execEnv)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return &ToolResult{
			ForLLM:  string(output),
			ForUser: string(output),
			IsError: true,
			Err:     fmt.Errorf("execlineb error: %w", err),
		}
	}

	return &ToolResult{
		ForLLM:  string(output),
		ForUser: string(output),
		IsError: false,
	}
}

// validateCommand checks for dangerous command patterns
// Since execline is secure by default, we only block obvious exploits
func (t *ExeclineTool) validateCommand(cmd string) error {
	// Block obvious shell escape attempts
	if strings.Contains(cmd, "&&") || strings.Contains(cmd, "||") {
		return fmt.Errorf("control operators (&&, ||) not supported in execline")
	}
	if strings.Contains(cmd, "|") && strings.Contains(cmd, "sh") {
		return fmt.Errorf("pipe to shell detected")
	}
	// Note: $VAR and $(cmd) are simply not expanded by execlineb
	// They are passed literally to the command, so this is safe
	return nil
}
