package tools

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/tools/shell"
)

// Default deny patterns for execline (Linux only - no variable expansion blocks needed)
var defaultExeclineDenyPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\brm\s+-[rf]{1,2}\b`),
	regexp.MustCompile(`\b(format|mkfs|diskpart)\b\s`),
	regexp.MustCompile(`\bdd\s+if=`),
	// Block device writes
	regexp.MustCompile(
		`>\s*/dev/(sd[a-z]|hd[a-z]|vd[a-z]|xvd[a-z]|nvme\d|mmcblk\d|loop\d|dm-\d|md\d|sr\d|nbd\d)`,
	),
	regexp.MustCompile(`\b(shutdown|reboot|poweroff)\b`),
	regexp.MustCompile(`\bsudo\b`),
	regexp.MustCompile(`\bdocker\s+run\b`),
	regexp.MustCompile(`\bdocker\s+exec\b`),
	regexp.MustCompile(`\bgit\s+push\b`),
	regexp.MustCompile(`\bgit\s+force\b`),
}

// ExeclineTool executes commands using execlineb instead of shell
// Security: execlineb does not support variable expansion ($VAR) or command
// substitution $(cmd) by default, reducing attack surface significantly.
type ExeclineTool struct {
	config        *config.Config
	denyPatterns  []*regexp.Regexp
	allowPatterns []*regexp.Regexp
	timeout       time.Duration
}

// NewExeclineTool creates a new ExeclineTool instance
func NewExeclineTool(cfg *config.Config) *ExeclineTool {
	// Start with default deny patterns only if enabled
	var denyPatterns []*regexp.Regexp
	if cfg.Tools.Execline.DenyDefaultsEnable {
		denyPatterns = append([]*regexp.Regexp{}, defaultExeclineDenyPatterns...)
	}

	// Add custom deny patterns from config
	for _, p := range cfg.Tools.Execline.Deny {
		if r, err := regexp.Compile(p); err == nil {
			denyPatterns = append(denyPatterns, r)
		}
	}

	// Compile allow patterns
	var allowPatterns []*regexp.Regexp
	for _, p := range cfg.Tools.Execline.Allow {
		if r, err := regexp.Compile(p); err == nil {
			allowPatterns = append(allowPatterns, r)
		}
	}

	// Default timeout 60s
	timeout := 60 * time.Second
	if cfg.Tools.Execline.TimeoutSeconds > 0 {
		timeout = time.Duration(cfg.Tools.Execline.TimeoutSeconds) * time.Second
	}

	return &ExeclineTool{
		config:        cfg,
		denyPatterns:  denyPatterns,
		allowPatterns: allowPatterns,
		timeout:       timeout,
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
			"working_dir": map[string]any{
				"type":        "string",
				"description": "Optional working directory for the command",
			},
			"env": map[string]any{
				"type":        "object",
				"description": "Additional environment variables to set for this command. Do not try to set PICOCLAW*, PATH, HOME, USER, LOGNAME, SHELL, LD_PRELOAD, or LD_LIBRARY_PATH",
				"additionalProperties": map[string]any{
					"type": "string",
				},
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
		extraEnv = t.config.Tools.Execline.EnvSet
	}

	// Parse env param from LLM (if provided)
	if envArg, ok := args["env"].(map[string]any); ok && envArg != nil {
		if extraEnv == nil {
			extraEnv = make(map[string]string)
		}
		for k, v := range envArg {
			if strVal, ok := v.(string); ok {
				extraEnv[k] = strVal
			}
		}
	}

	// Add PICOCLAW_EXEC_TIME - timestamp when command is executed
	execTimeEnv := map[string]string{
		"PICOCLAW_EXEC_TIME":    time.Now().Format(time.RFC3339),
		"PICOCLAW_EXEC_TIMEOUT": t.timeout.String(),
	}
	execEnv := shell.MergeEnvVars(baseEnv, execTimeEnv, extraEnv)

	// Use execlineb to execute
	// execlineb -c takes a command string and executes it
	// Unlike sh -c, it doesn't expand $VAR or $(cmd)
	// Add timeout to context
	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/usr/bin/execlineb", "-c", command)
	cmd.Env = shell.MapToEnvSlice(execEnv)

	// Set working directory if provided
	if wd, ok := args["working_dir"].(string); ok && wd != "" {
		cmd.Dir = wd
	}

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
	// Note: $VAR and $(cmd) are simply not expanded by execlineb
	// They are passed literally to the command, so this is safe

	// Check custom allow patterns first (can override deny)
	explicitlyAllowed := false
	for _, pattern := range t.allowPatterns {
		if pattern.MatchString(cmd) {
			explicitlyAllowed = true
			break
		}
	}

	if !explicitlyAllowed {
		// Check custom deny patterns
		for _, pattern := range t.denyPatterns {
			if pattern.MatchString(cmd) {
				return fmt.Errorf("command matches blocked pattern")
			}
		}
	}

	return nil
}
