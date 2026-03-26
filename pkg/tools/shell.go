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
	execToolExt // fork-specific fields (see shell_ext.go)

	workingDir string

	timeout time.Duration

	denyPatterns []*regexp.Regexp

	customAllowPatterns []*regexp.Regexp

	allowedPathPatterns []*regexp.Regexp

	restrictToWorkspace bool
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
	return `Execute shell commands. Use background=true for long-running commands (returns sessionId). Use pty=true for interactive commands (can combine with background=true). Use poll/read/write/send-keys/kill with sessionId to manage background sessions. Sessions auto-cleanup 30 minutes after process exits; use kill to terminate early. Output buffer limit: 1MB. Legacy: bg_action (output/kill) with bg_id also supported.`
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
			"pty": map[string]any{
				"type":        "string",
				"description": "Run in a pseudo-terminal (PTY) when available",
			},
			"cwd": map[string]any{
				"type":        "string",
				"description": "Working directory for the command",
			},
			"working_dir": map[string]any{
				"type":        "string",
				"description": "Optional working directory for the command (alias for cwd)",
			},
			"timeout": map[string]any{
				"type":        "integer",
				"description": "Timeout in seconds (0 = no timeout)",
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
	// Handle action-based dispatch if action is provided
	action, _ := args["action"].(string)
	if action != "" {
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

	// Legacy path: Handle bg_action first (output/kill)
	if bgAction, ok := args["bg_action"].(string); ok && bgAction != "" {
		bgID, _ := args["bg_id"].(string)
		return t.handleBgAction(bgAction, bgID)
	}

	return t.executeRun(ctx, args)
}

func (t *ExecTool) executeRun(ctx context.Context, args map[string]any) *ToolResult {
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

	if override := WorkspaceOverrideFromCtx(ctx); override != "" {
		cwd = override
	}

	// Support both "cwd" (upstream) and "working_dir" (fork) parameter names
	wd, _ := args["cwd"].(string)
	if wd == "" {
		wd, _ = args["working_dir"].(string)
	}
	if wd != "" {
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
			if ob.Len() > 0 {
				msg += "\n\nPartial output before timeout:\n" + ob.String()
			}
			return &ToolResult{
				ForLLM: msg,

				ForUser: msg,
				IsError: true,
				Err:     fmt.Errorf("command timeout: %w", err),
			}
		}

		// Extract detailed exit information
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode := exitErr.ExitCode()
			fmt.Fprintf(&ob, "\n\n[Command exited with code %d]", exitCode)

			// Add signal information if killed by signal (Unix)
			if exitCode == -1 {
				ob.WriteString(" (killed by signal)")
			}
		} else {
			fmt.Fprintf(&ob, "\n\n[Command failed: %v]", err)
		}
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

	if err := cmd.Start(); err != nil {
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
			if isAllowedPath(p, t.allowedPathPatterns) {
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
