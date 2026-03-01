package tools

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/agent/sandbox"
	"github.com/sipeed/picoclaw/pkg/config"
)

type ExecTool struct {
	workingDir          string
	timeout             time.Duration
	denyPatterns        []*regexp.Regexp
	allowPatterns       []*regexp.Regexp
	restrictToWorkspace bool
}

var (
	defaultDenyPatterns = []*regexp.Regexp{
		regexp.MustCompile(`\brm\s+-[rf]{1,2}\b`),
		regexp.MustCompile(`\bdel\s+/[fq]\b`),
		regexp.MustCompile(`\brmdir\s+/s\b`),
		// Match disk wiping commands (must be followed by space/args)
		regexp.MustCompile(
			`\b(format|mkfs|diskpart)\b\s`,
		),
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

	// absolutePathPattern matches absolute file paths in commands (Unix and Windows).
	absolutePathPattern = regexp.MustCompile(`[A-Za-z]:\\[^\\\"']+|/[^\s\"']+`)
)

func NewExecTool(workingDir string, restrict bool) (*ExecTool, error) {
	return NewExecToolWithConfig(workingDir, restrict, nil)
}

func NewExecToolWithConfig(workingDir string, restrict bool, config *config.Config) (*ExecTool, error) {
	denyPatterns := make([]*regexp.Regexp, 0)

	if config != nil {
		execConfig := config.Tools.Exec
		enableDenyPatterns := execConfig.EnableDenyPatterns
		if enableDenyPatterns {
			denyPatterns = append(denyPatterns, defaultDenyPatterns...)
			if len(execConfig.CustomDenyPatterns) > 0 {
				fmt.Printf("Using custom deny patterns: %v\n", execConfig.CustomDenyPatterns)
				for _, pattern := range execConfig.CustomDenyPatterns {
					re, err := regexp.Compile(pattern)
					if err != nil {
						return nil, fmt.Errorf("invalid custom deny pattern %q: %w", pattern, err)
					}
					denyPatterns = append(denyPatterns, re)
				}
			}
		} else {
			// If deny patterns are disabled, we won't add any patterns, allowing all commands.
			fmt.Println("Warning: deny patterns are disabled. All commands will be allowed.")
		}
	} else {
		denyPatterns = append(denyPatterns, defaultDenyPatterns...)
	}

	return &ExecTool{
		workingDir:          workingDir,
		timeout:             60 * time.Second,
		denyPatterns:        denyPatterns,
		allowPatterns:       nil,
		restrictToWorkspace: restrict,
	}, nil
}

func (t *ExecTool) Name() string {
	return "exec"
}

func (t *ExecTool) Description() string {
	return "Execute a shell command and return its output. Use with caution."
}

func (t *ExecTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "The shell command to execute",
			},
			"working_dir": map[string]any{
				"type":        "string",
				"description": "Optional working directory for the command",
			},
		},
		"required": []string{"command"},
	}
}

func (t *ExecTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	command, ok := args["command"].(string)
	if !ok {
		return ErrorResult("command is required")
	}

	wd, _ := args["working_dir"].(string)

	sb := sandbox.FromContext(ctx)
	if sb == nil {
		return ErrorResult("sandbox environment unavailable")
	}

	effectiveWorkspace := sb.GetWorkspace(ctx)

	// Resolve the working directory
	cwd := wd
	if cwd == "" {
		cwd = effectiveWorkspace
	}

	if cwd == "" {
		if dir, err := os.Getwd(); err == nil {
			cwd = dir
		}
	}

	if cwd == "" {
		cwd = "."
	}

	if wd != "" && t.restrictToWorkspace && effectiveWorkspace != "" {
		resolvedWD, err := sandbox.ValidatePath(wd, effectiveWorkspace, true)
		if err != nil {
			// If ValidatePath explicitly found the path is outside (e.g. symlink escape),
			// block it immediately and do NOT fall back to prefix matching.
			if errors.Is(err, sandbox.ErrOutsideWorkspace) {
				return ErrorResult("Command blocked by safety guard (" + err.Error() + ")")
			}

			// In sandbox mode, allow explicit container workspace paths when
			// restrict_to_workspace is enabled, but only for paths that don't exist on host.
			if filepath.IsAbs(wd) && isSandboxWorkspaceAbsolutePath(wd, effectiveWorkspace) {
				cwd = wd
			} else {
				return ErrorResult("Command blocked by safety guard (" + err.Error() + ")")
			}
		} else {
			cwd = resolvedWD
		}
	}

	if guardError := t.guardCommand(command, cwd); guardError != "" {
		return ErrorResult(guardError)
	}

	sandboxWD := t.resolveSandboxWorkingDir(cwd, effectiveWorkspace)
	res, err := sb.Exec(ctx, sandbox.ExecRequest{
		Command:    command,
		WorkingDir: sandboxWD,
		TimeoutMs:  t.timeout.Milliseconds(),
	})
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "context deadline exceeded") {
			msg := fmt.Sprintf("command timed out after %v", t.timeout)
			return &ToolResult{
				ForLLM:  msg,
				ForUser: msg,
				IsError: true,
			}
		}
		return ErrorResult(fmt.Sprintf("sandbox exec failed: %v", err))
	}
	output := res.Stdout
	if res.Stderr != "" {
		output += "\nSTDERR:\n" + res.Stderr
	}
	if output == "" {
		output = "(no output)"
	}

	maxLen := 10000
	if len(output) > maxLen {
		output = output[:maxLen] + fmt.Sprintf("\n... (truncated, %d more chars)", len(output)-maxLen)
	}

	if res.ExitCode != 0 {
		output += fmt.Sprintf("\nExit code: %d", res.ExitCode)
		return &ToolResult{
			ForLLM:  fmt.Sprintf("Command failed with exit code %d:\n%s", res.ExitCode, output),
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
	return guardCommandWithPolicy(
		command,
		cwd,
		t.restrictToWorkspace,
		t.denyPatterns,
		t.allowPatterns,
	)
}

func guardCommandWithPolicy(
	command, cwd string,
	restrictToWorkspace bool,
	denyPatterns, allowPatterns []*regexp.Regexp,
) string {
	cmd := strings.TrimSpace(command)
	lower := strings.ToLower(cmd)

	for _, pattern := range denyPatterns {
		if pattern.MatchString(lower) {
			return "Command blocked by safety guard (dangerous pattern detected)"
		}
	}

	if len(allowPatterns) > 0 {
		allowed := false
		for _, pattern := range allowPatterns {
			if pattern.MatchString(lower) {
				allowed = true
				break
			}
		}
		if !allowed {
			return "Command blocked by safety guard (not in allowlist)"
		}
	}

	if restrictToWorkspace {
		if strings.Contains(cmd, "..\\") || strings.Contains(cmd, "../") {
			return "Command blocked by safety guard (path traversal detected)"
		}

		cwdPath, err := filepath.Abs(cwd)
		if err != nil {
			return ""
		}

		matches := absolutePathPattern.FindAllString(cmd, -1)

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

func (t *ExecTool) resolveSandboxWorkingDir(cwd, workspace string) string {
	trimmed := strings.TrimSpace(cwd)
	if trimmed == "" {
		return "."
	}
	if !filepath.IsAbs(trimmed) {
		return trimmed
	}
	base := strings.TrimSpace(workspace)
	if base != "" {
		absBase, err := filepath.Abs(base)
		if err == nil {
			rel, err := filepath.Rel(absBase, trimmed)
			if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
				if rel == "." {
					return "."
				}
				return filepath.ToSlash(rel)
			}
		}
	}
	if isSandboxWorkspaceAbsolutePath(trimmed, workspace) {
		return filepath.ToSlash(trimmed)
	}
	// Preserve explicit absolute paths in sandbox mode (e.g. /tmp/logs),
	// instead of silently downgrading to ".".
	return filepath.ToSlash(trimmed)
}

func isSandboxWorkspaceAbsolutePath(wd, workspace string) bool {
	if workspace == "" {
		return false
	}
	cleanWD := path.Clean(filepath.ToSlash(strings.TrimSpace(wd)))
	cleanWS := path.Clean(filepath.ToSlash(strings.TrimSpace(workspace)))
	return cleanWD == cleanWS || strings.HasPrefix(cleanWD, cleanWS+"/")
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
