package tools

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/security"
)

// ExecToolConfig holds configurable options for ExecTool.
type ExecToolConfig struct {
	DenyPatterns  []string // Additional regex deny patterns from config
	AllowPatterns []string // If set, only matching commands are allowed
	MaxTimeout    int      // Seconds, default 60
	PolicyEngine  *security.PolicyEngine
	ExecGuardMode security.PolicyMode
}

type ExecTool struct {
	workingDir          string
	timeout             time.Duration
	denyPatterns        []*regexp.Regexp
	allowPatterns       []*regexp.Regexp
	restrictToWorkspace bool
	policyEngine        *security.PolicyEngine
	execGuardMode       security.PolicyMode
	channel             string
	chatID              string
}

var defaultDenyPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\brm\s+-[rf]{1,2}\b`),
	regexp.MustCompile(`\bdel\s+/[fq]\b`),
	regexp.MustCompile(`\brmdir\s+/s\b`),
	regexp.MustCompile(`\b(format|mkfs|diskpart)\b\s`), // Match disk wiping commands (must be followed by space/args)
	regexp.MustCompile(`\bdd\s+if=`),
	regexp.MustCompile(`>\s*/dev/sd[a-z]\b`), // Block writes to disk devices (but allow /dev/null)
	regexp.MustCompile(`\b(shutdown|reboot|poweroff)\b`),
	regexp.MustCompile(`:\(\)\s*\{.*\};\s*:`),
	regexp.MustCompile(`\$\([^)]+\)`),
	regexp.MustCompile(`\$\{[^}]+\}`),
	regexp.MustCompile("`[^`]+`"),
	regexp.MustCompile(`\|\s*sh\b`),
	regexp.MustCompile(`\|\s*bash\b`),
	regexp.MustCompile(`;\s*rm\s+-[rf]`),
	regexp.MustCompile(`&&\s*rm\s+-[rf]`),
	regexp.MustCompile(`\|\|\s*rm\s+-[rf]`),
	regexp.MustCompile(`>\s*/dev/null\s*>&?\s*\d?`),
	regexp.MustCompile(`<<\s*EOF`),
	regexp.MustCompile(`\$\(\s*cat\s+`),
	regexp.MustCompile(`\$\(\s*curl\s+`),
	regexp.MustCompile(`\$\(\s*wget\s+`),
	regexp.MustCompile(`\$\(\s*which\s+`),
	regexp.MustCompile(`\bsudo\b`),
	regexp.MustCompile(`\bchmod\s+[0-7]{3,4}\b`),
	regexp.MustCompile(`\bchown\b`),
	regexp.MustCompile(`\bpkill\b`),
	regexp.MustCompile(`\bkillall\b`),
	regexp.MustCompile(`\bkill\s+-[9]\b`),
	regexp.MustCompile(`\bcurl\b.*\s+(-d|--data|--data-raw|--data-binary|-F|--form|-T|--upload-file)\b`),
	regexp.MustCompile(`\bwget\b.*\s+(--post-data|--post-file)\b`),
	regexp.MustCompile(`\bnc\b\s+\S+\s+\d+`),
	regexp.MustCompile(`\bncat\b\s+\S+\s+\d+`),
	regexp.MustCompile(`base64\b.*\|\s*(sh|bash|zsh)\b`),
	regexp.MustCompile(`\b(bash|sh|zsh)\s+-i\s+[>&]`),
	regexp.MustCompile(`/dev/tcp/`),
	regexp.MustCompile(`\bcurl\b.*\|\s*(sh|bash)`),
	regexp.MustCompile(`\bwget\b.*\|\s*(sh|bash)`),
	regexp.MustCompile(`\bnpm\s+install\s+-g\b`),
	regexp.MustCompile(`\bpip\s+install\s+--user\b`),
	regexp.MustCompile(`\bapt\s+(install|remove|purge)\b`),
	regexp.MustCompile(`\byum\s+(install|remove)\b`),
	regexp.MustCompile(`\bdnf\s+(install|remove)\b`),
	regexp.MustCompile(`\bdocker\s+run\b`),
	regexp.MustCompile(`\bdocker\s+exec\b`),
	regexp.MustCompile(`\bgit\s+push\b`),
	regexp.MustCompile(`\bgit\s+force\b`),
	regexp.MustCompile(`\bssh\b.*@`),
	regexp.MustCompile(`\beval\b`),
	regexp.MustCompile(`\bsource\s+.*\.sh\b`),
}

func NewExecTool(workingDir string, restrict bool) *ExecTool {
	return NewExecToolWithConfig(workingDir, restrict, ExecToolConfig{})
}

func NewExecToolWithConfig(workingDir string, restrict bool, cfg ExecToolConfig) *ExecTool {
	denyPatterns := make([]*regexp.Regexp, len(defaultDenyPatterns))
	copy(denyPatterns, defaultDenyPatterns)

	for _, p := range cfg.DenyPatterns {
		re, err := regexp.Compile(p)
		if err == nil {
			denyPatterns = append(denyPatterns, re)
		}
	}

	var allowPatterns []*regexp.Regexp
	for _, p := range cfg.AllowPatterns {
		re, err := regexp.Compile(p)
		if err == nil {
			allowPatterns = append(allowPatterns, re)
		}
	}

	timeout := 60 * time.Second
	if cfg.MaxTimeout > 0 {
		timeout = time.Duration(cfg.MaxTimeout) * time.Second
	}

	return &ExecTool{
		workingDir:          workingDir,
		timeout:             timeout,
		denyPatterns:        denyPatterns,
		allowPatterns:       allowPatterns,
		restrictToWorkspace: restrict,
		policyEngine:        cfg.PolicyEngine,
		execGuardMode:       cfg.ExecGuardMode,
	}
}

// SetContext implements ContextualTool so the ExecTool receives the current
// IM channel and chatID for approval requests.
func (t *ExecTool) SetContext(channel, chatID string) {
	t.channel = channel
	t.chatID = chatID
}

func (t *ExecTool) Name() string {
	return "exec"
}

func (t *ExecTool) Description() string {
	return "Execute a shell command and return its output. Use with caution."
}

func (t *ExecTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "The shell command to execute",
			},
			"working_dir": map[string]interface{}{
				"type":        "string",
				"description": "Optional working directory for the command",
			},
		},
		"required": []string{"command"},
	}
}

func (t *ExecTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	command, ok := args["command"].(string)
	if !ok {
		return ErrorResult("command is required")
	}

	cwd := t.workingDir
	if wd, ok := args["working_dir"].(string); ok && wd != "" {
		cwd = wd
	}

	if cwd == "" {
		wd, err := os.Getwd()
		if err == nil {
			cwd = wd
		}
	}

	if guardError := t.guardCommand(ctx, command, cwd); guardError != "" {
		return ErrorResult(guardError)
	}

	// timeout == 0 means no timeout
	var cmdCtx context.Context
	var cancel context.CancelFunc
	if t.timeout > 0 {
		cmdCtx, cancel = context.WithTimeout(ctx, t.timeout)
	} else {
		cmdCtx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(cmdCtx, "powershell", "-NoProfile", "-NonInteractive", "-Command", command)
	} else {
		cmd = exec.CommandContext(cmdCtx, "sh", "-c", command)
	}
	if cwd != "" {
		cmd.Dir = cwd
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\nSTDERR:\n" + stderr.String()
	}

	if err != nil {
		if cmdCtx.Err() == context.DeadlineExceeded {
			msg := fmt.Sprintf("Command timed out after %v", t.timeout)
			return &ToolResult{
				ForLLM:  msg,
				ForUser: msg,
				IsError: true,
			}
		}
		output += fmt.Sprintf("\nExit code: %v", err)
	}

	if output == "" {
		output = "(no output)"
	}

	maxLen := 10000
	if len(output) > maxLen {
		output = output[:maxLen] + fmt.Sprintf("\n... (truncated, %d more chars)", len(output)-maxLen)
	}

	if err != nil {
		return &ToolResult{
			ForLLM:  output,
			ForUser: output,
			IsError: true,
		}
	}

	return &ToolResult{
		ForLLM:  output,
		ForUser: output,
		IsError: false,
	}
}

func (t *ExecTool) guardCommand(ctx context.Context, command, cwd string) string {
	mode := t.execGuardMode
	cmd := strings.TrimSpace(command)
	lower := strings.ToLower(cmd)

	// Deny-pattern check (mode-aware)
	if !mode.IsOff() {
		for _, pattern := range t.denyPatterns {
			if pattern.MatchString(lower) {
				reason := "dangerous pattern detected: " + pattern.String()
				if err := t.evaluatePolicy(ctx, mode, command, reason, pattern.String()); err != nil {
					return err.Error()
				}
				break // approved by user, continue
			}
		}

		// Allow-pattern check
		if len(t.allowPatterns) > 0 {
			allowed := false
			for _, pattern := range t.allowPatterns {
				if pattern.MatchString(lower) {
					allowed = true
					break
				}
			}
			if !allowed {
				reason := "command not in allowlist"
				if err := t.evaluatePolicy(ctx, mode, command, reason, "allowlist"); err != nil {
					return err.Error()
				}
			}
		}
	}

	// Workspace restriction checks (always active when restrictToWorkspace is true)
	if t.restrictToWorkspace {
		if strings.Contains(cmd, "..\\") || strings.Contains(cmd, "../") {
			return "Command blocked by safety guard (path traversal detected)"
		}

		sensitivePathPatterns := []*regexp.Regexp{
			regexp.MustCompile(`\b/etc/`),
			regexp.MustCompile(`\b/var/`),
			regexp.MustCompile(`\b/root\b`),
			regexp.MustCompile(`\b/home/`),
			regexp.MustCompile(`\b/proc/`),
			regexp.MustCompile(`\b/sys/`),
			regexp.MustCompile(`\b/boot/`),
		}
		for _, pattern := range sensitivePathPatterns {
			if pattern.MatchString(lower) {
				return "Command blocked by safety guard (access to sensitive path)"
			}
		}

		cwdPath, err := filepath.Abs(cwd)
		if err != nil {
			return ""
		}

		pathPattern := regexp.MustCompile(`[A-Za-z]:\\[^\\\"']+|/[^\s\"']+`)
		matches := pathPattern.FindAllString(cmd, -1)

		for _, raw := range matches {
			p, err := filepath.Abs(raw)
			if err != nil {
				continue
			}

			rel, err := filepath.Rel(cwdPath, p)
			if err != nil {
				continue
			}

			if strings.HasPrefix(rel, "..") {
				return "Command blocked by safety guard (path outside working dir)"
			}
		}
	}

	return ""
}

// evaluatePolicy delegates to the PolicyEngine when available.
func (t *ExecTool) evaluatePolicy(ctx context.Context, mode security.PolicyMode, action, reason, ruleName string) error {
	if t.policyEngine == nil {
		if mode == security.ModeOff {
			return nil
		}
		return fmt.Errorf("blocked by safety guard: %s", reason)
	}
	return t.policyEngine.Evaluate(ctx, mode, security.Violation{
		Category: "exec_guard",
		Tool:     "exec",
		Action:   action,
		Reason:   reason,
		RuleName: ruleName,
	}, t.channel, t.chatID)
}

func (t *ExecTool) SetTimeout(timeout time.Duration) {
	t.timeout = timeout
}

func (t *ExecTool) SetRestrictToWorkspace(restrict bool) {
	t.restrictToWorkspace = restrict
}

func (t *ExecTool) SetAllowPatterns(patterns []string) error {
	t.allowPatterns = make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			return fmt.Errorf("invalid allow pattern %q: %w", p, err)
		}
		t.allowPatterns = append(t.allowPatterns, re)
	}
	return nil
}
