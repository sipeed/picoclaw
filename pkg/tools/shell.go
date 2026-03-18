package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/constants"
)

var globalSessionManager = NewSessionManager()
var sessionManagerMu sync.RWMutex

func getSessionManager() *SessionManager {
	sessionManagerMu.RLock()
	defer sessionManagerMu.RUnlock()
	return globalSessionManager
}

type ExecTool struct {
	workingDir          string
	timeout             time.Duration
	denyPatterns        []*regexp.Regexp
	allowPatterns       []*regexp.Regexp
	customAllowPatterns []*regexp.Regexp
	allowedPathPatterns []*regexp.Regexp
	restrictToWorkspace bool
	allowRemote         bool
	sessionManager      *SessionManager
}

var ptyForbiddenPrograms = []string{
	"bash", "sh", "zsh", "fish", "dash", "ksh", "csh", "tcsh",
	"python", "python3", "python2",
	"node", "nodejs",
	"ruby", "perl", "php", "lua", "lua5",
	"powershell", "pwsh",
	"adb", "telnet",
}

func isForbiddenInterpreter(cmd string) bool {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return false
	}
	base := filepath.Base(parts[0])
	for _, p := range ptyForbiddenPrograms {
		if base == p {
			return true
		}
	}
	return false
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
		regexp.MustCompile(`\bssh\b.*@`),
		regexp.MustCompile(`\beval\b`),
		regexp.MustCompile(`\bsource\s+.*\.sh\b`),
	}

	// absolutePathPattern matches absolute file paths in commands (Unix and Windows).
	absolutePathPattern = regexp.MustCompile(`[A-Za-z]:\\[^\\\"']+|/[^\s\"']+`)

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

func NewExecTool(workingDir string, restrict bool, allowPaths ...[]*regexp.Regexp) (*ExecTool, error) {
	return NewExecToolWithConfig(workingDir, restrict, nil, allowPaths...)
}

func NewExecToolWithConfig(
	workingDir string,
	restrict bool,
	config *config.Config,
	allowPaths ...[]*regexp.Regexp,
) (*ExecTool, error) {
	denyPatterns := make([]*regexp.Regexp, 0)
	customAllowPatterns := make([]*regexp.Regexp, 0)
	var allowedPathPatterns []*regexp.Regexp
	allowRemote := true
	if len(allowPaths) > 0 {
		allowedPathPatterns = allowPaths[0]
	}

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

	timeout := 60 * time.Second
	if config != nil && config.Tools.Exec.TimeoutSeconds > 0 {
		timeout = time.Duration(config.Tools.Exec.TimeoutSeconds) * time.Second
	}

	return &ExecTool{
		workingDir:          workingDir,
		timeout:             timeout,
		denyPatterns:        denyPatterns,
		allowPatterns:       nil,
		customAllowPatterns: customAllowPatterns,
		allowedPathPatterns: allowedPathPatterns,
		restrictToWorkspace: restrict,
		allowRemote:         allowRemote,
		sessionManager:      getSessionManager(),
	}, nil
}

func (t *ExecTool) Name() string {
	return "exec"
}

func (t *ExecTool) Description() string {
	return `Execute a shell command and return its output.

Actions:
- run: Execute a command (supports background mode)
- list: List all active sessions
- poll: Check session status
- read: Read output from a session
- write: Send input to a session
- kill: Terminate a session

Use background=true for long-running commands. Use pty=true for interactive commands (not supported for shell interpreters).`
}

func (t *ExecTool) Parameters() map[string]any {
	return map[string]any{
		"oneOf": []map[string]any{
			{
				"type": "object",
				"properties": map[string]any{
					"action":     map[string]any{"const": "run"},
					"command":    map[string]any{"type": "string", "description": "The shell command to execute"},
					"pty":        map[string]any{"type": "boolean", "description": "Use PTY for interactive mode"},
					"background": map[string]any{"type": "boolean", "description": "Run in background mode"},
					"timeout":    map[string]any{"type": "integer", "description": "Timeout in seconds (0 = no timeout)"},
					"cwd":        map[string]any{"type": "string", "description": "Working directory"},
				},
				"required": []string{"action", "command"},
			},
			{
				"type": "object",
				"properties": map[string]any{
					"action": map[string]any{"const": "list"},
				},
				"required": []string{"action"},
			},
			{
				"type": "object",
				"properties": map[string]any{
					"action":    map[string]any{"const": "poll"},
					"sessionId": map[string]any{"type": "string", "description": "Session ID to poll"},
				},
				"required": []string{"action", "sessionId"},
			},
			{
				"type": "object",
				"properties": map[string]any{
					"action":    map[string]any{"const": "read"},
					"sessionId": map[string]any{"type": "string", "description": "Session ID to read from"},
				},
				"required": []string{"action", "sessionId"},
			},
			{
				"type": "object",
				"properties": map[string]any{
					"action":    map[string]any{"const": "write"},
					"sessionId": map[string]any{"type": "string", "description": "Session ID to write to"},
					"data":      map[string]any{"type": "string", "description": "Data to write to the session"},
				},
				"required": []string{"action", "sessionId", "data"},
			},
			{
				"type": "object",
				"properties": map[string]any{
					"action":    map[string]any{"const": "kill"},
					"sessionId": map[string]any{"type": "string", "description": "Session ID to kill"},
				},
				"required": []string{"action", "sessionId"},
			},
		},
	}
}

func (t *ExecTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	action, _ := args["action"].(string)
	if action == "" {
		return ErrorResult("action is required")
	}

	switch action {
	case "run":
		return t.executeRun(ctx, args)
	case "list":
		return t.executeList()
	case "poll":
		return t.executePoll(args)
	case "read":
		return t.executeRead(args)
	case "write":
		return t.executeWrite(args)
	case "kill":
		return t.executeKill(args)
	default:
		return ErrorResult(fmt.Sprintf("unknown action: %s", action))
	}
}

func (t *ExecTool) executeRun(ctx context.Context, args map[string]any) *ToolResult {
	command, ok := args["command"].(string)
	if !ok {
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

	pty, _ := args["pty"].(bool)
	background, _ := args["background"].(bool)

	if pty {
		if runtime.GOOS == "windows" {
			return ErrorResult("PTY is not supported on Windows. Use background=true without pty.")
		}
		if isForbiddenInterpreter(command) {
			return ErrorResult("PTY is forbidden for interpreter programs (bash, python, node, etc.) for security reasons")
		}
	}

	cwd := t.workingDir
	if wd, ok := args["cwd"].(string); ok && wd != "" {
		if t.restrictToWorkspace && t.workingDir != "" {
			resolvedWD, err := validatePathWithAllowPaths(wd, t.workingDir, true, t.allowedPathPatterns)
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
		if isAllowedPath(resolved, t.allowedPathPatterns) {
			cwd = resolved
		} else {
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
	}

	if background {
		return t.runBackground(ctx, command, cwd, pty)
	}

	return t.runSync(ctx, command, cwd)
}

func (t *ExecTool) runSync(ctx context.Context, command, cwd string) *ToolResult {
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

	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\nSTDERR:\n" + stderr.String()
	}

	if err != nil {
		if errors.Is(cmdCtx.Err(), context.DeadlineExceeded) {
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

func (t *ExecTool) runBackground(ctx context.Context, command, cwd string, ptyEnabled bool) *ToolResult {
	sessionID := generateSessionID()
	session := &ProcessSession{
		ID:         sessionID,
		Command:    command,
		PTY:        ptyEnabled,
		Background: true,
		StartTime:  time.Now().Unix(),
		Status:     "running",
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}
	if cwd != "" {
		cmd.Dir = cwd
	}

	prepareCommandForTermination(cmd)

	if ptyEnabled {
		// Create PTY pair
		ptmx, tty, err := pty.Open()
		if err != nil {
			return ErrorResult(fmt.Sprintf("failed to create PTY: %v", err))
		}

		// Set up the command to use the PTY slave as stdin/stdout/stderr
		cmd.Stdin = tty
		cmd.Stdout = tty
		cmd.Stderr = tty

		// For PTY, we need Setsid to create a new session.
		// Note: Setsid and Setpgid conflict, so we must replace SysProcAttr entirely.
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

		session.ptyMaster = ptmx
	} else {
		// Non-PTY mode: use pipes
		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			return ErrorResult(fmt.Sprintf("failed to create stdout pipe: %v", err))
		}
		stderrPipe, err := cmd.StderrPipe()
		if err != nil {
			return ErrorResult(fmt.Sprintf("failed to create stderr pipe: %v", err))
		}
		stdinPipe, err := cmd.StdinPipe()
		if err != nil {
			return ErrorResult(fmt.Sprintf("failed to create stdin pipe: %v", err))
		}

		session.stdoutPipe = io.MultiReader(stdoutPipe, stderrPipe)
		session.stdinWriter = stdinPipe
	}

	if err := cmd.Start(); err != nil {
		if session.ptyMaster != nil {
			session.ptyMaster.Close()
		}
		return ErrorResult(fmt.Sprintf("failed to start command: %v", err))
	}

	session.PID = cmd.Process.Pid
	t.sessionManager.Add(session)

	go func() {
		err := cmd.Wait()

		// Close PTY master when process exits
		if session.ptyMaster != nil {
			session.ptyMaster.Close()
			session.ptyMaster = nil
		}

		session.mu.Lock()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				session.ExitCode = exitErr.ExitCode()
			} else {
				session.ExitCode = 1
			}
		} else {
			session.ExitCode = 0
		}
		session.Status = "done"
		session.mu.Unlock()
	}()

	resp := ExecResponse{
		SessionID: sessionID,
		Status:    "running",
	}
	data, _ := json.Marshal(resp)
	return &ToolResult{
		ForLLM:  string(data),
		ForUser: fmt.Sprintf("Session %s started", sessionID),
		IsError: false,
	}
}

func (t *ExecTool) executeList() *ToolResult {
	sessions := t.sessionManager.List()
	resp := ExecResponse{
		Sessions: sessions,
	}
	data, _ := json.Marshal(resp)
	return &ToolResult{
		ForLLM:  string(data),
		ForUser: fmt.Sprintf("%d active sessions", len(sessions)),
		IsError: false,
	}
}

func (t *ExecTool) executePoll(args map[string]any) *ToolResult {
	sessionID, ok := args["sessionId"].(string)
	if !ok {
		return ErrorResult("sessionId is required")
	}

	session, err := t.sessionManager.Get(sessionID)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return ErrorResult(fmt.Sprintf("session not found: %s", sessionID))
		}
		return ErrorResult(err.Error())
	}

	resp := ExecResponse{
		SessionID: sessionID,
		Status:    session.GetStatus(),
		ExitCode:  session.GetExitCode(),
	}
	data, _ := json.Marshal(resp)
	return &ToolResult{
		ForLLM:  string(data),
		IsError: false,
	}
}

func (t *ExecTool) executeRead(args map[string]any) *ToolResult {
	sessionID, ok := args["sessionId"].(string)
	if !ok {
		return ErrorResult("sessionId is required")
	}

	session, err := t.sessionManager.Get(sessionID)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return ErrorResult(fmt.Sprintf("session not found: %s", sessionID))
		}
		return ErrorResult(err.Error())
	}

	if session.IsDone() {
		return ErrorResult(fmt.Sprintf("process already exited with code %d", session.GetExitCode()))
	}

	output, err := session.Read()
	if err != nil && !errors.Is(err, ErrSessionDone) {
		return ErrorResult(fmt.Sprintf("failed to read from session: %v", err))
	}

	resp := ExecResponse{
		SessionID: sessionID,
		Output:    output,
		Status:    session.GetStatus(),
	}
	data, _ := json.Marshal(resp)
	return &ToolResult{
		ForLLM:  string(data),
		IsError: false,
	}
}

func (t *ExecTool) executeWrite(args map[string]any) *ToolResult {
	sessionID, ok := args["sessionId"].(string)
	if !ok {
		return ErrorResult("sessionId is required")
	}

	data, ok := args["data"].(string)
	if !ok {
		return ErrorResult("data is required")
	}

	session, err := t.sessionManager.Get(sessionID)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return ErrorResult(fmt.Sprintf("session not found: %s", sessionID))
		}
		return ErrorResult(err.Error())
	}

	if session.IsDone() {
		return ErrorResult(fmt.Sprintf("process already exited with code %d", session.GetExitCode()))
	}

	if err := session.Write(data); err != nil {
		if errors.Is(err, ErrSessionDone) {
			return ErrorResult(fmt.Sprintf("process already exited with code %d", session.GetExitCode()))
		}
		return ErrorResult(fmt.Sprintf("failed to write to session: %v", err))
	}

	resp := ExecResponse{
		SessionID: sessionID,
		Status:    session.GetStatus(),
	}
	respData, _ := json.Marshal(resp)
	return &ToolResult{
		ForLLM:  string(respData),
		IsError: false,
	}
}

func (t *ExecTool) executeKill(args map[string]any) *ToolResult {
	sessionID, ok := args["sessionId"].(string)
	if !ok {
		return ErrorResult("sessionId is required")
	}

	session, err := t.sessionManager.Get(sessionID)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return ErrorResult(fmt.Sprintf("session not found: %s", sessionID))
		}
		return ErrorResult(err.Error())
	}

	if session.IsDone() {
		return ErrorResult(fmt.Sprintf("process already exited with code %d", session.GetExitCode()))
	}

	if err := session.Kill(); err != nil {
		return ErrorResult(fmt.Sprintf("failed to kill session: %v", err))
	}

	t.sessionManager.Remove(sessionID)

	resp := ExecResponse{
		SessionID: sessionID,
		Status:    "done",
	}
	data, _ := json.Marshal(resp)
	return &ToolResult{
		ForLLM:  string(data),
		ForUser: fmt.Sprintf("Session %s killed", sessionID),
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

	if len(t.allowPatterns) > 0 {
		allowed := false
		for _, pattern := range t.allowPatterns {
			if pattern.MatchString(lower) {
				allowed = true
				break
			}
		}
		if !allowed {
			return "Command blocked by safety guard (not in allowlist)"
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

		// Web URL schemes whose path components (starting with //) should be exempt
		// from workspace sandbox checks. file: is intentionally excluded so that
		// file:// URIs are still validated against the workspace boundary.
		webSchemes := []string{"http:", "https:", "ftp:", "ftps:", "sftp:", "ssh:", "git:"}

		matchIndices := absolutePathPattern.FindAllStringIndex(cmd, -1)

		for _, loc := range matchIndices {
			raw := cmd[loc[0]:loc[1]]

			// Skip URL path components that look like they're from web URLs.
			// When a URL like "https://github.com" is parsed, the regex captures
			// "//github.com" as a match (the path portion after "https:").
			// Use the exact match position (loc[0]) so that duplicate //path substrings
			// in the same command are each evaluated at their own position.
			if strings.HasPrefix(raw, "//") && loc[0] > 0 {
				before := cmd[:loc[0]]
				isWebURL := false

				for _, scheme := range webSchemes {
					if strings.HasSuffix(before, scheme) {
						isWebURL = true
						break
					}
				}

				if isWebURL {
					continue
				}
			}

			p, err := filepath.Abs(raw)
			if err != nil {
				continue
			}

			if safePaths[p] {
				continue
			}
			if isAllowedPath(p, t.allowedPathPatterns) {
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
