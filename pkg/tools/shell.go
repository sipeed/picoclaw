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
	"time"

	"github.com/creack/pty"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/constants"
	"github.com/sipeed/picoclaw/pkg/isolation"
)

var (
	globalSessionManager = NewSessionManager()
	sessionManagerMu     sync.RWMutex
)

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
	filterEnv           bool
	allowRemote         bool
	sessionManager      *SessionManager
}

var (
	defaultDenyPatterns = []*regexp.Regexp{
		regexp.MustCompile(`\brm\s+-[rf]{1,2}\b`),
		regexp.MustCompile(`\bdel\s+/[fq]\b`),
		regexp.MustCompile(`\brmdir\s+/s\b`),
		// Match disk wiping commands (must be followed by space/args)
		regexp.MustCompile(
			`(^|[^-\w])\b(format|mkfs|diskpart)\b\s`,
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
	cfg *config.Config,
	allowPaths ...[]*regexp.Regexp,
) (*ExecTool, error) {
	denyPatterns := make([]*regexp.Regexp, 0)
	customAllowPatterns := make([]*regexp.Regexp, 0)
	var allowedPathPatterns []*regexp.Regexp
	allowRemote := true
	if len(allowPaths) > 0 {
		allowedPathPatterns = allowPaths[0]
	}

	if cfg != nil {
		execConfig := cfg.Tools.Exec
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

	var timeout time.Duration
	if cfg != nil && cfg.Tools.Exec.TimeoutSeconds > 0 {
		timeout = time.Duration(cfg.Tools.Exec.TimeoutSeconds) * time.Second
	}

	filterEnv := false
	if cfg != nil {
		filterEnv = cfg.Tools.Exec.FilterEnv
	}

	return &ExecTool{
		workingDir:          workingDir,
		timeout:             timeout,
		denyPatterns:        denyPatterns,
		allowPatterns:       nil,
		customAllowPatterns: customAllowPatterns,
		allowedPathPatterns: allowedPathPatterns,
		restrictToWorkspace: restrict,
		filterEnv:           filterEnv,
		allowRemote:         allowRemote,
		sessionManager:      getSessionManager(),
	}, nil
}

func (t *ExecTool) Name() string {
	return "exec"
}

func (t *ExecTool) Description() string {
	return `Execute shell commands. Use background=true for long-running commands (returns sessionId). Use pty=true for interactive commands (can combine with background=true). Use poll/read/write/send-keys/kill with sessionId to manage background sessions. Sessions auto-cleanup 30 minutes after process exits; use kill to terminate early. Output buffer limit: 1MB.`
}

func (t *ExecTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"run", "list", "poll", "read", "write", "kill", "send-keys"},
				"description": "Action: run (execute command), list (show sessions), poll (check status), read (get output), write (send input), kill (terminate), send-keys (send keys to PTY)",
			},
			"command": map[string]any{
				"type":        "string",
				"description": "Shell command to execute (required for run)",
			},
			"sessionId": map[string]any{
				"type":        "string",
				"description": "Session ID (required for poll/read/write/kill/send-keys)",
			},
			"keys": map[string]any{
				"type":        "string",
				"description": "Key names for send-keys: up, down, left, right, enter, tab, escape, backspace, ctrl-c, ctrl-d, home, end, pageup, pagedown, f1-f12",
			},
			"data": map[string]any{
				"type":        "string",
				"description": "Data to write to stdin (required for write)",
			},
			"background": map[string]any{
				"type":        "string",
				"description": "Run in background immediately",
			},
			"pty": map[string]any{
				"type":        "string",
				"description": "Run in a pseudo-terminal (PTY) when available",
			},
			"cwd": map[string]any{
				"type":        "string",
				"description": "Working directory for the command",
			},
			"timeout": map[string]any{
				"type":        "integer",
				"description": "Timeout in seconds (0 = no timeout)",
			},
		},
		"required": []string{"action"},
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
	case "send-keys":
		return t.executeSendKeys(args)
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

	getBoolArg := func(key string) bool {
		switch v := args[key].(type) {
		case bool:
			return v
		case string:
			return v == "true"
		}
		return false
	}
	isPty := getBoolArg("pty")
	isBackground := getBoolArg("background")

	if isPty {
		if runtime.GOOS == "windows" {
			return ErrorResult("PTY is not supported on Windows. Use background=true without pty.")
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

	if isBackground {
		return t.runBackground(ctx, command, cwd, isPty)
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

	if t.filterEnv {
		cmd.Env = filterEnvironment(os.Environ())
	}

	prepareCommandForTermination(cmd)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Route shell execution through the shared isolation entry point so exec tool
	// subprocesses receive the same isolation policy as other integrations.
	if err := isolation.Start(cmd); err != nil {
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
			if output != "" {
				msg += "\n\nPartial output before timeout:\n" + output
			}
			return &ToolResult{
				ForLLM:  msg,
				ForUser: msg,
				IsError: true,
				Err:     fmt.Errorf("command timeout: %w", err),
			}
		}

		// Extract detailed exit information
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode := exitErr.ExitCode()
			output += fmt.Sprintf("\n\n[Command exited with code %d]", exitCode)

			// Add signal information if killed by signal (Unix)
			if exitCode == -1 {
				output += " (killed by signal)"
			}
		} else {
			output += fmt.Sprintf("\n\n[Command failed: %v]", err)
		}
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
		ptyKeyMode: PtyKeyModeCSI,
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

	var stdoutReader io.ReadCloser
	var stderrReader io.ReadCloser
	var stdinWriter io.WriteCloser

	if ptyEnabled {
		ptmx, tty, err := pty.Open()
		if err != nil {
			return ErrorResult(fmt.Sprintf("failed to create PTY: %v", err))
		}

		cmd.Stdin = tty
		cmd.Stdout = tty
		cmd.Stderr = tty

		// For PTY, we need Setsid to create a new session.
		// Note: Setsid and Setpgid conflict, so we must replace SysProcAttr entirely.
		setSysProcAttrForPty(cmd)

		session.ptyMaster = ptmx
	} else {
		var err error
		stdoutReader, err = cmd.StdoutPipe()
		if err != nil {
			return ErrorResult(fmt.Sprintf("failed to create stdout pipe: %v", err))
		}
		stderrReader, err = cmd.StderrPipe()
		if err != nil {
			return ErrorResult(fmt.Sprintf("failed to create stderr pipe: %v", err))
		}
		stdinWriter, err = cmd.StdinPipe()
		if err != nil {
			return ErrorResult(fmt.Sprintf("failed to create stdin pipe: %v", err))
		}
		session.stdoutPipe = io.MultiReader(stdoutReader, stderrReader)
		session.stdinWriter = stdinWriter
	}

	// Background sessions use the same startup path so isolation stays consistent
	// with synchronous exec runs.
	if err := isolation.Start(cmd); err != nil {
		if session.ptyMaster != nil {
			session.ptyMaster.Close()
		}
		return ErrorResult(fmt.Sprintf("failed to start command: %v", err))
	}

	session.PID = cmd.Process.Pid
	t.sessionManager.Add(session)

	session.outputBuffer = &bytes.Buffer{}

	// PTY mode: read from ptyMaster and wait for process
	// Note: On Linux, closing ptyMaster doesn't interrupt blocking Read() calls,
	// so we need cmd.Wait() in a separate goroutine to detect process exit.
	if session.PTY && session.ptyMaster != nil {
		go func() {
			cmd.Wait() // Wait for process to exit
			session.mu.Lock()
			if cmd.ProcessState != nil {
				session.ExitCode = cmd.ProcessState.ExitCode()
			}
			session.Status = "done"
			session.mu.Unlock()
		}()

		go func() {
			buf := make([]byte, 4096)
			for {
				n, err := session.ptyMaster.Read(buf)
				if n > 0 {
					raw := string(buf[:n])
					if mode := detectPtyKeyMode(raw); mode != PtyKeyModeNotFound && mode != session.GetPtyKeyMode() {
						session.SetPtyKeyMode(mode)
					}

					session.mu.Lock()
					if session.outputBuffer.Len() >= maxOutputBufferSize {
						if !session.outputTruncated {
							session.outputBuffer.WriteString(outputTruncateMarker)
							session.outputTruncated = true
						}
					} else {
						session.outputBuffer.Write(buf[:n])
					}
					session.mu.Unlock()
				}
				if err != nil {
					break
				}
			}
		}()
	} else {
		// Non-PTY mode: single goroutine reads pipes.
		// When Read() returns EOF (pipe closed), we break.
		// When process exits, OS closes pipe write end → Read() returns EOF → we exit.
		go func() {
			buf := make([]byte, 4096)

			// Read stdout
			for {
				n, err := stdoutReader.Read(buf)
				if n > 0 {
					session.mu.Lock()
					if session.outputBuffer.Len() >= maxOutputBufferSize {
						if !session.outputTruncated {
							session.outputBuffer.WriteString(outputTruncateMarker)
							session.outputTruncated = true
						}
					} else {
						session.outputBuffer.Write(buf[:n])
					}
					session.mu.Unlock()
				}
				if err != nil {
					break
				}
			}

			// Read stderr
			for {
				n, err := stderrReader.Read(buf)
				if n > 0 {
					session.mu.Lock()
					if session.outputBuffer.Len() >= maxOutputBufferSize {
						if !session.outputTruncated {
							session.outputBuffer.WriteString(outputTruncateMarker)
							session.outputTruncated = true
						}
					} else {
						session.outputBuffer.Write(buf[:n])
					}
					session.mu.Unlock()
				}
				if err != nil {
					break
				}
			}

			// All pipes closed, get exit status
			if stdinWriter != nil {
				stdinWriter.Close()
			}
			cmd.Wait()

			session.mu.Lock()
			if cmd.ProcessState != nil {
				session.ExitCode = cmd.ProcessState.ExitCode()
			}
			session.Status = "done"
			session.mu.Unlock()
		}()
	}

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

	output := session.Read()

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

// keyMap maps key names to their escape sequences.
var keyMap = map[string]string{
	"enter":     "\r",
	"return":    "\r",
	"tab":       "\t",
	"escape":    "\x1b",
	"esc":       "\x1b",
	"space":     " ",
	"backspace": "\x7f",
	"bspace":    "\x7f",
	"up":        "\x1b[A",
	"down":      "\x1b[B",
	"right":     "\x1b[C",
	"left":      "\x1b[D",
	"home":      "\x1b[1~",
	"end":       "\x1b[4~",
	"pageup":    "\x1b[5~",
	"pagedown":  "\x1b[6~",
	"pgup":      "\x1b[5~",
	"pgdn":      "\x1b[6~",
	"insert":    "\x1b[2~",
	"ic":        "\x1b[2~",
	"delete":    "\x1b[3~",
	"del":       "\x1b[3~",
	"dc":        "\x1b[3~",
	"btab":      "\x1b[Z",
	"f1":        "\x1bOP",
	"f2":        "\x1bOQ",
	"f3":        "\x1bOR",
	"f4":        "\x1bOS",
	"f5":        "\x1b[15~",
	"f6":        "\x1b[17~",
	"f7":        "\x1b[18~",
	"f8":        "\x1b[19~",
	"f9":        "\x1b[20~",
	"f10":       "\x1b[21~",
	"f11":       "\x1b[23~",
	"f12":       "\x1b[24~",
}

// ss3KeysMap maps key names to SS3 escape sequences
var ss3KeysMap = map[string]string{
	"up":    "\x1bOA",
	"down":  "\x1bOB",
	"right": "\x1bOC",
	"left":  "\x1bOD",
	"home":  "\x1bOH",
	"end":   "\x1bOF",
}

func detectPtyKeyMode(raw string) PtyKeyMode {
	const SMKX = "\x1b[?1h"
	const RMKX = "\x1b[?1l"

	lastSmkx := strings.LastIndex(raw, SMKX)
	lastRmkx := strings.LastIndex(raw, RMKX)

	if lastSmkx == -1 && lastRmkx == -1 {
		return PtyKeyModeNotFound
	}

	if lastSmkx > lastRmkx {
		return PtyKeyModeSS3
	}
	return PtyKeyModeCSI
}

// encodeKeyToken encodes a single key token into its escape sequence.
// Supports:
//   - Named keys: "enter", "tab", "up", "ctrl-c", "alt-x", etc.
//   - Ctrl modifier: "ctrl-c" or "c-c" (sends Ctrl+char)
//   - Alt modifier: "alt-x" or "m-x" (sends ESC+char)
func encodeKeyToken(token string, ptyKeyMode PtyKeyMode) (string, error) {
	token = strings.ToLower(strings.TrimSpace(token))
	if token == "" {
		return "", nil
	}

	// Handle ctrl-X format (c-x)
	if strings.HasPrefix(token, "c-") {
		char := token[2]
		if char >= 'a' && char <= 'z' {
			return string(rune(char) & 0x1f), nil // ctrl-a through ctrl-z
		}
		return "", fmt.Errorf("invalid ctrl key: %s", token)
	}

	// Handle ctrl-X format (ctrl-x)
	if strings.HasPrefix(token, "ctrl-") {
		char := token[5]
		if char >= 'a' && char <= 'z' {
			return string(rune(char) & 0x1f), nil
		}
		return "", fmt.Errorf("invalid ctrl key: %s", token)
	}

	// Handle alt-X format (m-x or alt-x)
	if strings.HasPrefix(token, "m-") || strings.HasPrefix(token, "alt-") {
		var char string
		if strings.HasPrefix(token, "m-") {
			char = token[2:]
		} else {
			char = token[4:]
		}
		if len(char) == 1 {
			return "\x1b" + char, nil
		}
		return "", fmt.Errorf("invalid alt key: %s", token)
	}

	// Handle shift modifier for special keys (shift-up, shift-down, etc.)
	if strings.HasPrefix(token, "s-") || strings.HasPrefix(token, "shift-") {
		var key string
		if strings.HasPrefix(token, "s-") {
			key = token[2:]
		} else {
			key = token[6:]
		}
		// Apply shift modifier: for single-char keys, return uppercase
		if seq, ok := keyMap[key]; ok {
			// For escape sequences, we can't easily add shift
			// For single-char keys (letters), return uppercase
			if len(seq) == 1 {
				return strings.ToUpper(seq), nil
			}
			return seq, nil
		}
		return "", fmt.Errorf("unknown key with shift: %s", key)
	}

	if ptyKeyMode == PtyKeyModeSS3 {
		if seq, ok := ss3KeysMap[token]; ok {
			return seq, nil
		}
	}

	if seq, ok := keyMap[token]; ok {
		return seq, nil
	}

	return "", fmt.Errorf("unknown key: %s (use write action for text input)", token)
}

// encodeKeySequence encodes a slice of key tokens into a single string.
func encodeKeySequence(tokens []string, ptyKeyMode PtyKeyMode) (string, error) {
	var result string
	for _, token := range tokens {
		seq, err := encodeKeyToken(token, ptyKeyMode)
		if err != nil {
			return "", err
		}
		result += seq
	}
	return result, nil
}

func (t *ExecTool) executeSendKeys(args map[string]any) *ToolResult {
	sessionID, ok := args["sessionId"].(string)
	if !ok {
		return ErrorResult("sessionId is required")
	}

	keysStr, ok := args["keys"].(string)
	if !ok {
		return ErrorResult("keys must be a string")
	}

	if keysStr == "" {
		return ErrorResult("keys cannot be empty")
	}

	// Parse comma-separated key names
	keyNames := strings.Split(keysStr, ",")
	var keys []string
	for _, k := range keyNames {
		k = strings.TrimSpace(k)
		if k != "" {
			keys = append(keys, k)
		}
	}

	if len(keys) == 0 {
		return ErrorResult("keys cannot be empty")
	}

	session, err := t.sessionManager.Get(sessionID)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return ErrorResult(fmt.Sprintf("session not found: %s", sessionID))
		}
		return ErrorResult(err.Error())
	}

	ptyKeyMode := session.GetPtyKeyMode()

	data, err := encodeKeySequence(keys, ptyKeyMode)
	if err != nil {
		return ErrorResult(fmt.Sprintf("invalid key: %v", err))
	}

	if session.IsDone() {
		return ErrorResult(fmt.Sprintf("process already exited with code %d", session.GetExitCode()))
	}

	if err := session.Write(data); err != nil {
		if errors.Is(err, ErrSessionDone) {
			return ErrorResult(fmt.Sprintf("process already exited with code %d", session.GetExitCode()))
		}
		return ErrorResult(fmt.Sprintf("failed to send keys: %v", err))
	}

	resp := ExecResponse{
		SessionID: sessionID,
		Status:    "running",
		Output:    fmt.Sprintf("Sent keys: %v", keys),
	}
	respData, _ := json.Marshal(resp)
	return &ToolResult{
		ForLLM:  string(respData),
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

// safeEnvVars is the allowlist of environment variable names preserved when
// filterEnv is enabled. Variables with a PICOCLAW_ prefix are always kept.
var safeEnvVars = map[string]bool{
	"PATH":         true,
	"HOME":         true,
	"USER":         true,
	"LANG":         true,
	"TERM":         true,
	"SHELL":        true,
	"TMPDIR":       true,
	"TMP":          true,
	"TEMP":         true,
	"SystemRoot":   true,
	"COMSPEC":      true,
	"USERPROFILE":  true,
	"APPDATA":      true,
	"LOCALAPPDATA": true,
}

// filterEnvironment returns a filtered copy of environ keeping only safe vars.
func filterEnvironment(environ []string) []string {
	filtered := make([]string, 0, len(safeEnvVars)+4)
	for _, entry := range environ {
		eqIdx := strings.IndexByte(entry, '=')
		if eqIdx < 0 {
			continue
		}
		name := entry[:eqIdx]
		if safeEnvVars[name] || strings.HasPrefix(name, "PICOCLAW_") {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}
