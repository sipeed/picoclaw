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

	"github.com/sipeed/picoclaw/pkg/logger"
)

// Precompiled regexes for workspace-escape checks (used when restrictToWorkspace=true)
var (
	shellMetaRe    = regexp.MustCompile("`|\\$\\(|\\$\\{")
	varReferenceRe = regexp.MustCompile(`\$[A-Za-z_][A-Za-z0-9_]*`)
	cdAbsoluteRe   = regexp.MustCompile(`(?i)\bcd\s+/`)
	ansiCQuoteRe   = regexp.MustCompile(`\$'`)
	ansiDQuoteRe   = regexp.MustCompile(`\$"`)
	hexEscapeRe    = regexp.MustCompile(`\\x[0-9a-fA-F]`)
	octalEscapeRe  = regexp.MustCompile(`\\[0-7]{1,3}`)
	escapedMetaRe  = regexp.MustCompile(`\\[` + "`" + `$]`)
)

type ExecTool struct {
	workingDir          string
	timeout             time.Duration
	denyPatterns        []*regexp.Regexp
	allowPatterns       []*regexp.Regexp
	restrictToWorkspace bool
}

func NewExecTool(workingDir string, restrict bool) *ExecTool {
	denyPatterns := []*regexp.Regexp{
		// rm with short flags
		regexp.MustCompile(`\brm\s+-[rf]{1,2}\b`),
		// rm with long flags
		regexp.MustCompile(`\brm\s+--recursive\b`),
		regexp.MustCompile(`\brm\s+--force\b`),
		// Windows delete commands
		regexp.MustCompile(`\bdel\s+/[fq]\b`),
		regexp.MustCompile(`\brmdir\s+/s\b`),
		// Disk wiping commands
		regexp.MustCompile(`\b(format|mkfs|diskpart|fdisk|parted|wipefs)\b\s`),
		regexp.MustCompile(`\bdd\s+if=`),
		// Block writes to disk devices (but allow /dev/null)
		regexp.MustCompile(`>\s*/dev/sd[a-z]\b`),
		// System shutdown/reboot
		regexp.MustCompile(`\b(shutdown|reboot|poweroff)\b`),
		// Fork bomb
		regexp.MustCompile(`:\(\)\s*\{.*\};\s*:`),
		// base64 decode piped to shell execution
		regexp.MustCompile(`base64\s+(-d|--decode).*\|\s*(sh|bash|ash|dash)\b`),
		// Scripting languages with inline execution flags
		regexp.MustCompile(`\b(python3?|perl|ruby)\s+-(c|e)\b`),
		// eval with dynamic content
		regexp.MustCompile(`\beval\s+["'` + "`" + `$]`),
		// curl/wget piped to shell
		regexp.MustCompile(`\b(curl|wget)\b.*\|\s*(sh|bash|ash|dash)\b`),
		// find -exec rm
		regexp.MustCompile(`\bfind\b.*-exec\s+rm\b`),
		// xargs rm
		regexp.MustCompile(`\bxargs\b.*\brm\b`),
	}

	return &ExecTool{
		workingDir:          workingDir,
		timeout:             60 * time.Second,
		denyPatterns:        denyPatterns,
		allowPatterns:       nil,
		restrictToWorkspace: restrict,
	}
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

	if t.restrictToWorkspace && cwd != t.workingDir {
		absCwd, err := filepath.Abs(cwd)
		if err != nil {
			return ErrorResult("invalid working directory")
		}
		absWs, err := filepath.Abs(t.workingDir)
		if err != nil {
			return ErrorResult("invalid workspace directory")
		}
		if !isWithinWorkspace(absCwd, absWs) {
			return ErrorResult("Command blocked by safety guard (working directory outside workspace)")
		}
	}

	if cwd == "" {
		wd, err := os.Getwd()
		if err == nil {
			cwd = wd
		}
	}

	if guardError := t.guardCommand(command, cwd); guardError != "" {
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

func (t *ExecTool) guardCommand(command, cwd string) string {
	cmd := strings.TrimSpace(command)
	lower := strings.ToLower(cmd)

	// Check denylist patterns
	for _, pattern := range t.denyPatterns {
		if pattern.MatchString(lower) {
			logger.WarnCF("shell", "Command blocked (dangerous pattern)", map[string]interface{}{
				"command_preview": truncateForLog(cmd),
				"pattern":         pattern.String(),
			})
			return "Command blocked by safety guard (dangerous pattern detected)"
		}
	}

	// Check allowlist if configured
	if len(t.allowPatterns) > 0 {
		allowed := false
		for _, pattern := range t.allowPatterns {
			if pattern.MatchString(lower) {
				allowed = true
				break
			}
		}
		if !allowed {
			logger.WarnCF("shell", "Command blocked (not in allowlist)", map[string]interface{}{
				"command_preview": truncateForLog(cmd),
			})
			return "Command blocked by safety guard (not in allowlist)"
		}
	}

	if t.restrictToWorkspace {
		// Block shell metacharacters that enable workspace escape (backticks, $(), ${})
		if shellMetaRe.MatchString(cmd) {
			logger.WarnCF("shell", "Command blocked (shell metacharacter in restricted mode)", map[string]interface{}{
				"command_preview": truncateForLog(cmd),
			})
			return "Command blocked by safety guard (shell metacharacter in restricted mode)"
		}

		// Block escape sequences that can bypass shell metacharacter detection
		escapePatterns := []*regexp.Regexp{ansiCQuoteRe, ansiDQuoteRe, hexEscapeRe, octalEscapeRe, escapedMetaRe}
		for _, re := range escapePatterns {
			if re.MatchString(cmd) {
				logger.WarnCF("shell", "Command blocked (escape sequence in restricted mode)", map[string]interface{}{
					"command_preview": truncateForLog(cmd),
					"pattern":         re.String(),
				})
				return "Command blocked by safety guard (escape sequence in restricted mode)"
			}
		}

		// Block variable expansion ($VAR) which can reference paths outside workspace
		if varReferenceRe.MatchString(cmd) {
			logger.WarnCF("shell", "Command blocked (variable expansion in restricted mode)", map[string]interface{}{
				"command_preview": truncateForLog(cmd),
			})
			return "Command blocked by safety guard (variable expansion in restricted mode)"
		}

		// Block cd to absolute path
		if cdAbsoluteRe.MatchString(cmd) {
			logger.WarnCF("shell", "Command blocked (cd to absolute path in restricted mode)", map[string]interface{}{
				"command_preview": truncateForLog(cmd),
			})
			return "Command blocked by safety guard (cd to absolute path in restricted mode)"
		}

		// Block relative path traversal
		if strings.Contains(cmd, "..\\") || strings.Contains(cmd, "../") {
			logger.WarnCF("shell", "Command blocked (path traversal)", map[string]interface{}{
				"command_preview": truncateForLog(cmd),
			})
			return "Command blocked by safety guard (path traversal detected)"
		}

		// Block absolute paths outside workspace
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
				logger.WarnCF("shell", "Command blocked (path outside working dir)", map[string]interface{}{
					"command_preview": truncateForLog(cmd),
					"path":            raw,
				})
				return "Command blocked by safety guard (path outside working dir)"
			}
		}
	}

	return ""
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

// truncateForLog truncates a string for safe logging, avoiding exposure of full commands.
func truncateForLog(s string) string {
	const maxLen = 120
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}
