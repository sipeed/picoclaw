package tools

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/constants"
)

type ExecTool struct {
	execToolExt // fork-specific fields (see shell_ext.go)

	workingDir string

	timeout time.Duration

	denyPatterns []*regexp.Regexp

	customAllowPatterns []*regexp.Regexp

	restrictToWorkspace bool

	allowRemote bool
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

		// Block writes to block devices (all common naming schemes).
		regexp.MustCompile(
			`>\s*/dev/(sd[a-z]|hd[a-z]|vd[a-z]|xvd[a-z]|nvme\d|mmcblk\d|loop\d|dm-\d|md\d|sr\d|nbd\d)`,
		),

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

		regexp.MustCompile(`\bkill\b`),

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

		regexp.MustCompile(`\bgit\s+checkout\b`),

		regexp.MustCompile(`\bgit\s+switch\b`),

		regexp.MustCompile(`\bssh\b.*@`),

		regexp.MustCompile(`\beval\b`),

		regexp.MustCompile(`\bsource\s+.*\.sh\b`),
	}

	// safePaths are kernel pseudo-devices that are always safe to reference in
	// commands, regardless of workspace restriction. They contain no user data
	// and cannot cause destructive writes.
	safePaths = map[string]bool{
		"/dev/null":    true,
		"/dev/zero":    true,
		"/dev/random":  true,
		"/dev/urandom": true,
		"/dev/stdin":   true,
		"/dev/stdout":  true,
		"/dev/stderr":  true,
	}
)

func NewExecTool(workingDir string, restrict bool) (*ExecTool, error) {
	return NewExecToolWithConfig(workingDir, restrict, nil)
}

func NewExecToolWithConfig(workingDir string, restrict bool, config *config.Config) (*ExecTool, error) {
	denyPatterns := make([]*regexp.Regexp, 0)
	customAllowPatterns := make([]*regexp.Regexp, 0)
	allowRemote := true

	if config != nil {
		execConfig := config.Tools.Exec
		enableDenyPatterns := execConfig.EnableDenyPatterns
		allowRemote = execConfig.AllowRemote

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
		for _, pattern := range execConfig.CustomAllowPatterns {
			re, err := regexp.Compile(pattern)
			if err != nil {
				return nil, fmt.Errorf("invalid custom allow pattern %q: %w", pattern, err)
			}
			customAllowPatterns = append(customAllowPatterns, re)
		}
	} else {
		denyPatterns = append(denyPatterns, defaultDenyPatterns...)
	}

	timeout := 5 * time.Minute
	if config != nil && config.Tools.Exec.TimeoutSeconds > 0 {
		timeout = time.Duration(config.Tools.Exec.TimeoutSeconds) * time.Second
	}

	bgCtx, bgCancel := context.WithCancel(context.Background())

	return &ExecTool{
		execToolExt: execToolExt{
			bgProcesses: make(map[string]*bgProcess),
			bgCtx:       bgCtx,
			bgShutdown:  bgCancel,
		},

		workingDir: workingDir,

		timeout: timeout,

		denyPatterns: denyPatterns,

		customAllowPatterns: customAllowPatterns,

		restrictToWorkspace: restrict,

		allowRemote: allowRemote,
	}, nil
}

func (t *ExecTool) Name() string {
	return "exec"
}

func (t *ExecTool) Description() string {
	return "Execute a shell command and return its output. Supports background execution with background=true, and managing background processes with bg_action."
}

func (t *ExecTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type": "string",

				"description": "The shell command to execute",
			},
			"working_dir": map[string]any{
				"type": "string",

				"description": "Optional working directory for the command",
			},

			"background": map[string]any{
				"type": "boolean",

				"description": "Run the command in the background. Returns immediately with a process ID.",
			},

			"bg_action": map[string]any{
				"type": "string",

				"enum": []string{"output", "kill"},

				"description": "Action on a background process: 'output' to get latest output, 'kill' to stop it.",
			},

			"bg_id": map[string]any{
				"type": "string",

				"description": "Background process ID (e.g. 'bg-1'). Required with bg_action.",
			},
		},

		"required": []string{},
	}
}

func (t *ExecTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	// Handle bg_action first (output/kill)

	if bgAction, ok := args["bg_action"].(string); ok && bgAction != "" {
		bgID, _ := args["bg_id"].(string)

		return t.handleBgAction(bgAction, bgID)
	}

	// Check for background execution

	bg, _ := args["background"].(bool)

	command, ok := args["command"].(string)

	if !ok || command == "" {
		return ErrorResult("command is required")
	}

	// GHSA-pv8c-p6jf-3fpp: block exec from remote channels (e.g. Telegram webhooks)
	// unless explicitly opted-in via config. Fail-closed: empty channel = blocked.
	if !t.allowRemote {
		channel := ToolChannel(ctx)
		if channel == "" {
			channel, _ = args["__channel"].(string)
		}
		channel = strings.TrimSpace(channel)
		if channel == "" || !constants.IsInternalChannel(channel) {
			return ErrorResult("exec is restricted to internal channels")
		}
	}

	cwd := t.workingDir

	if override := WorkspaceOverrideFromCtx(ctx); override != "" {
		cwd = override
	}

	if wd, ok := args["working_dir"].(string); ok && wd != "" {
		if t.restrictToWorkspace && cwd != "" {
			resolvedWD, err := validatePath(wd, cwd, true)
			if err != nil {
				return ErrorResult("Command blocked by safety guard (" + err.Error() + ")")
			}
			cwd = resolvedWD
		} else {
			cwd = wd
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

	// Re-resolve symlinks immediately before execution to shrink the TOCTOU window
	// between validation and cmd.Dir assignment.
	if t.restrictToWorkspace && t.workingDir != "" && cwd != t.workingDir {
		resolved, err := filepath.EvalSymlinks(cwd)
		if err != nil {
			return ErrorResult(fmt.Sprintf("Command blocked by safety guard (path resolution failed: %v)", err))
		}
		absWorkspace, _ := filepath.Abs(t.workingDir)
		wsResolved, _ := filepath.EvalSymlinks(absWorkspace)
		if wsResolved == "" {
			wsResolved = absWorkspace
		}
		rel, err := filepath.Rel(wsResolved, resolved)
		if err != nil || !filepath.IsLocal(rel) {
			return ErrorResult("Command blocked by safety guard (working directory escaped workspace)")
		}
		cwd = resolved
	}

	if bg {
		return t.executeBg(command, cwd)
	}

	return t.executeSync(ctx, command, cwd)
}

// executeSync runs a command synchronously (existing behavior).

func (t *ExecTool) executeSync(ctx context.Context, command, cwd string) *ToolResult {
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

	prepareCommandForTermination(cmd)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return ErrorResult(fmt.Sprintf("failed to start command: %v", err))
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	var err error
	select {
	case err = <-done:
	case <-cmdCtx.Done():
		_ = terminateProcessTree(cmd)
		select {
		case err = <-done:
		case <-time.After(2 * time.Second):
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			err = <-done
		}
	}

	var ob strings.Builder

	ob.WriteString(stdout.String())

	if stderr.Len() > 0 {
		ob.WriteString("\nSTDERR:\n")

		ob.WriteString(stderr.String())
	}

	if err != nil {
		if errors.Is(cmdCtx.Err(), context.DeadlineExceeded) {
			msg := fmt.Sprintf("Command timed out after %v", t.timeout)
			return &ToolResult{
				ForLLM: msg,

				ForUser: msg,
				IsError: true,
			}
		}

		fmt.Fprintf(&ob, "\nExit code: %v", err)
	}

	output := ob.String()

	if output == "" {
		output = "(no output)"
	}

	maxLen := 10000
	if len(output) > maxLen {
		output = output[:maxLen] + fmt.Sprintf("\n... (truncated, %d more chars)", len(output)-maxLen)
	}

	if err != nil {
		return &ToolResult{
			ForLLM: output,

			ForUser: output,
			IsError: true,
		}
	}

	return &ToolResult{
		ForLLM: output,

		ForUser: output,
		IsError: false,
	}
}

func (t *ExecTool) guardCommand(command, cwd string) string {
	cmd := strings.TrimSpace(command)
	lower := strings.ToLower(cmd)

	// Custom allow patterns exempt a command from deny checks.
	explicitlyAllowed := false
	for _, pattern := range t.customAllowPatterns {
		if pattern.MatchString(lower) {
			explicitlyAllowed = true
			break
		}
	}

	if !explicitlyAllowed {
		for _, pattern := range t.denyPatterns {
			if pattern.MatchString(lower) {
				return "Command blocked by safety guard (dangerous pattern detected)"
			}
		}
	}

	if len(t.allowRules) > 0 {
		if !matchAllowRules(lower, t.allowRules) {
			var b strings.Builder

			b.WriteString("Command blocked: not in allowlist [")

			for i, rule := range t.allowRules {
				if i > 0 {
					b.WriteByte(',')
				}

				b.WriteString(strings.Join(rule, " "))
			}

			b.WriteByte(']')

			return b.String()
		}
	}

	// Restrict curl/wget to localhost and RFC 1918 private addresses.

	// External HTTP access is available via the web_fetch tool.

	if t.localNetOnly && isCurlOrWget(cmd) {
		if errMsg := checkCurlLocalNet(cmd); errMsg != "" {
			return errMsg
		}
	}

	if t.restrictToWorkspace {
		if strings.Contains(cmd, "..\\") || strings.Contains(cmd, "../") {
			return "Command blocked by safety guard (path traversal detected)"
		}

		cwdPath, err := filepath.Abs(cwd)
		if err != nil {
			return ""
		}

		// Token-based absolute path detection.

		// Uses strings.Fields so relative paths (e.g., "tests/cold/file.py")

		// are not falsely flagged.

		// Flags like -I/usr/local/include are naturally skipped because

		// filepath.IsAbs returns false for tokens starting with "-".

		//

		// Agent CLI tools (claude, codex, gemini) accept slash commands

		// (e.g., "/review") that look like absolute paths but are not.

		// For these tools we check whether the token is an existing path

		// before blocking.

		// Web URL schemes whose path components (starting with //) should be exempt
		// from workspace sandbox checks. file: is intentionally excluded so that
		// file:// URIs are still validated against the workspace boundary.
		webSchemes := []string{"http:", "https:", "ftp:", "ftps:", "sftp:", "ssh:", "git:"}

		agentCLI := isAgentCLICommand(cmd)

		fields := strings.Fields(cmd)
		for i, token := range fields {
			token = strings.Trim(token, "\"'")

			// Handle file:// URIs: extract the path and validate it.
			if strings.HasPrefix(token, "file://") {
				token = strings.TrimPrefix(token, "file://")
				if !filepath.IsAbs(token) {
					continue
				}
			} else if !filepath.IsAbs(token) {
				continue
			}

			// Skip URL path components that look like they're from web URLs.
			// When a URL like "https://github.com/repo" is tokenised, the path
			// portion may appear absolute. Check whether the previous token is
			// a complete web URL containing this path component.
			if strings.HasPrefix(token, "//") {
				// Check if the previous non-operator token ends with a web scheme
				// that makes this token part of a URL rather than a real path.
				isPartOfURL := false
				if i > 0 {
					prev := strings.Trim(fields[i-1], "\"'")
					for _, scheme := range webSchemes {
						if strings.HasSuffix(prev, scheme) || strings.Contains(prev, scheme+"//") {
							isPartOfURL = true
							break
						}
					}
				}
				if isPartOfURL {
					continue
				}
			}

			p := filepath.Clean(token)

			if safePaths[p] {
				continue
			}

			rel, err := filepath.Rel(cwdPath, p)
			if err != nil {
				continue
			}

			if strings.HasPrefix(rel, "..") {
				// Path is outside workspace — allow if it's an executable binary

				if isExecutable(p) {
					continue
				}

				// Allow /dev/* paths (e.g. /dev/null, /dev/urandom).

				// Device files are not regular filesystem paths and pose

				// no workspace-escape risk.

				if strings.HasPrefix(p, "/dev/") {
					continue
				}

				// Agent CLI slash commands: skip non-existent paths

				// (e.g., "/review" is a command, not a file).

				if agentCLI {
					if _, statErr := os.Stat(p); os.IsNotExist(statErr) {
						continue
					}
				}

				return fmt.Sprintf("Command blocked: path outside working dir %s", p)
			}
		}
	}

	return ""
}

// agentCLINames lists agent CLI tools that use slash commands

// (e.g., "/review", "/help") which look like absolute paths.

var agentCLINames = []string{"claude", "codex", "gemini"}

// isAgentCLICommand returns true if the command invokes an agent CLI tool.

func isAgentCLICommand(cmd string) bool {
	fields := strings.Fields(cmd)

	if len(fields) == 0 {
		return false
	}

	base := filepath.Base(fields[0])

	for _, name := range agentCLINames {
		if base == name {
			return true
		}
	}

	return false
}

// isExecutable checks if a path points to an executable file.

// On Unix, checks the execute permission bits.

// On Windows, checks for known executable extensions.

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	if info.IsDir() {
		return false
	}

	if runtime.GOOS == "windows" {
		ext := strings.ToLower(filepath.Ext(path))

		switch ext {
		case ".exe", ".cmd", ".bat", ".ps1", ".com":

			return true
		}

		return false
	}

	return info.Mode()&0o111 != 0
}

func (t *ExecTool) SetTimeout(timeout time.Duration) {
	t.timeout = timeout
}

func (t *ExecTool) SetRestrictToWorkspace(restrict bool) {
	t.restrictToWorkspace = restrict
}

// SetAllowRules sets the command prefix allowlist.

// Each rule is a space-separated command prefix (e.g. "go test", "pnpm run lint").

// A command is allowed if its first N words match any rule's N words exactly.

func (t *ExecTool) SetAllowRules(rules []string) {
	t.allowRules = make([][]string, 0, len(rules))

	for _, r := range rules {
		words := strings.Fields(strings.ToLower(r))

		if len(words) > 0 {
			t.allowRules = append(t.allowRules, words)
		}
	}
}

// matchAllowRules checks if cmd matches any prefix in the allowlist.

func matchAllowRules(cmd string, rules [][]string) bool {
	cmdWords := strings.Fields(cmd)

	for _, ruleWords := range rules {
		if len(cmdWords) < len(ruleWords) {
			continue
		}

		match := true

		for i, rw := range ruleWords {
			if cmdWords[i] != rw {
				match = false

				break
			}
		}

		if match {
			return true
		}
	}

	return false
}
