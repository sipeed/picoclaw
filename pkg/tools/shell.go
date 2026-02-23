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
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
)

type ExecTool struct {
	workingDir          string
	timeout             time.Duration
	denyPatterns        []*regexp.Regexp
	allowPatterns       []*regexp.Regexp
	restrictToWorkspace bool
	processes           *ProcessManager
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

var (
	guardPathPattern  = regexp.MustCompile(`[A-Za-z]:\\[^\\\"']+|/[^\s\"']+`)
	envVarNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

func NewExecTool(workingDir string, restrict bool) *ExecTool {
	return NewExecToolWithConfig(workingDir, restrict, nil)
}

func NewExecToolWithConfig(workingDir string, restrict bool, config *config.Config) *ExecTool {
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
						fmt.Printf("Invalid custom deny pattern %q: %v\n", pattern, err)
						continue
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
		processes:           NewProcessManager(defaultProcessMaxOutputChars),
	}
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
			"background": map[string]any{
				"type":        "boolean",
				"description": "Start command in background and manage it via process tool",
			},
			"yield_ms": map[string]any{
				"type":        "integer",
				"description": "Wait this many milliseconds before returning running status",
				"minimum":     0.0,
			},
			"timeout_seconds": map[string]any{
				"type":        "integer",
				"description": "Override command timeout in seconds (0 disables timeout)",
				"minimum":     0.0,
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

	cwd := t.workingDir
	if wd, ok := args["working_dir"].(string); ok && wd != "" {
		if t.restrictToWorkspace && t.workingDir != "" {
			resolvedWD, err := validatePath(wd, t.workingDir, true)
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

	background, err := parseBoolArg(args, "background", false)
	if err != nil {
		return ErrorResult(err.Error())
	}

	yieldMS, err := parseOptionalIntArg(args, "yield_ms", 0, 0, 60*60*1000)
	if err != nil {
		return ErrorResult(err.Error())
	}

	timeoutSeconds, hasTimeoutOverride, err := readOptionalIntArg(args, "timeout_seconds", 0, 24*60*60)
	if err != nil {
		return ErrorResult(err.Error())
	}

	timeout := t.timeout
	if hasTimeoutOverride {
		if timeoutSeconds == 0 {
			timeout = 0
		} else {
			timeout = time.Duration(timeoutSeconds) * time.Second
		}
	}

	if !background && yieldMS <= 0 {
		return t.executeSync(ctx, command, cwd, timeout)
	}

	return t.executeManaged(
		ctx,
		command,
		cwd,
		background,
		time.Duration(yieldMS)*time.Millisecond,
		timeout,
	)
}

func (t *ExecTool) executeSync(ctx context.Context, command, cwd string, timeout time.Duration) *ToolResult {
	// timeout == 0 means no timeout.
	var cmdCtx context.Context
	var cancel context.CancelFunc
	if timeout > 0 {
		cmdCtx, cancel = context.WithTimeout(ctx, timeout)
	} else {
		cmdCtx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	cmd := shellCommand(cmdCtx, command)
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
			msg := fmt.Sprintf("Command timed out after %v", timeout)
			return &ToolResult{
				ForLLM:  msg,
				ForUser: msg,
				IsError: true,
			}
		}
		output += fmt.Sprintf("\nExit code: %v", err)
	}

	output = truncateExecOutput(output)

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

func (t *ExecTool) executeManaged(
	ctx context.Context,
	command, cwd string,
	background bool,
	yield time.Duration,
	timeout time.Duration,
) *ToolResult {
	if t.processes == nil {
		return t.executeSync(ctx, command, cwd, timeout)
	}

	baseCtx := context.Background()
	var cmdCtx context.Context
	var cancel context.CancelFunc
	if timeout > 0 {
		cmdCtx, cancel = context.WithTimeout(baseCtx, timeout)
	} else {
		cmdCtx, cancel = context.WithCancel(baseCtx)
	}

	cmd := shellCommand(cmdCtx, command)
	if cwd != "" {
		cmd.Dir = cwd
	}
	prepareCommandForTermination(cmd)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return ErrorResult(fmt.Sprintf("failed to attach stdout: %v", err))
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return ErrorResult(fmt.Sprintf("failed to attach stderr: %v", err))
	}
	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return ErrorResult(fmt.Sprintf("failed to attach stdin: %v", err))
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return ErrorResult(fmt.Sprintf("failed to start command: %v", err))
	}

	sessionID := t.processes.StartSession(command, cwd, cmd, stdinPipe, cancel)
	done := make(chan struct{})
	go t.watchManagedCommand(sessionID, cmdCtx, cmd, stdoutPipe, stderrPipe, done)

	if background {
		return t.runningSessionResult(sessionID)
	}

	if yield <= 0 {
		yield = 10 * time.Second
	}
	timer := time.NewTimer(yield)
	defer timer.Stop()

	select {
	case <-done:
		pollResult, err := t.processes.Poll(sessionID, 0)
		if err != nil {
			return ErrorResult(fmt.Sprintf("failed to read managed command result: %v", err))
		}
		return formatManagedCompletion(pollResult, timeout)
	case <-timer.C:
		return t.runningSessionResult(sessionID)
	case <-ctx.Done():
		_, _ = t.processes.Kill(sessionID)
		return ErrorResult(fmt.Sprintf("command canceled: %v", ctx.Err()))
	}
}

func (t *ExecTool) watchManagedCommand(
	sessionID string,
	cmdCtx context.Context,
	cmd *exec.Cmd,
	stdoutPipe, stderrPipe io.ReadCloser,
	done chan<- struct{},
) {
	defer close(done)

	stdoutDone := make(chan struct{})
	stderrDone := make(chan struct{})
	go func() {
		t.streamManagedOutput(sessionID, stdoutPipe, false)
		close(stdoutDone)
	}()
	go func() {
		t.streamManagedOutput(sessionID, stderrPipe, true)
		close(stderrDone)
	}()

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- cmd.Wait()
	}()

	var waitErr error
	select {
	case waitErr = <-waitDone:
	case <-cmdCtx.Done():
		_ = terminateProcessTree(cmd)
		select {
		case waitErr = <-waitDone:
		case <-time.After(2 * time.Second):
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			waitErr = <-waitDone
		}
	}

	<-stdoutDone
	<-stderrDone

	t.processes.MarkExited(sessionID, waitErr, errors.Is(cmdCtx.Err(), context.DeadlineExceeded))
}

func (t *ExecTool) streamManagedOutput(sessionID string, reader io.ReadCloser, stderr bool) {
	defer reader.Close()

	buf := make([]byte, 4096)
	wroteStderrHeader := false
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			if stderr && !wroteStderrHeader {
				t.processes.AppendOutput(sessionID, "\nSTDERR:\n")
				wroteStderrHeader = true
			}
			t.processes.AppendOutput(sessionID, string(buf[:n]))
		}
		if err != nil {
			return
		}
	}
}

func (t *ExecTool) runningSessionResult(sessionID string) *ToolResult {
	payload := map[string]any{
		"status":     "running",
		"session_id": sessionID,
	}
	if snap, ok := t.processes.GetSnapshot(sessionID); ok {
		payload["pid"] = snap.PID
		payload["started_at"] = snap.StartedAt
		payload["command"] = snap.Command
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to encode background exec result: %v", err))
	}
	return SilentResult(string(data))
}

func formatManagedCompletion(result ProcessPollResult, timeout time.Duration) *ToolResult {
	output := truncateExecOutput(result.Output)
	status := strings.ToLower(result.Session.Status)
	if status == "" {
		status = "completed"
	}

	switch status {
	case "completed":
		return UserResult(output)
	case "timeout":
		msg := fmt.Sprintf("Command timed out after %v", timeout)
		if output != "(no output)" {
			msg = output + "\n" + msg
		}
		return ErrorResult(msg)
	default:
		if result.Session.ExitError != "" && !strings.Contains(output, result.Session.ExitError) {
			output += "\nExit code: " + result.Session.ExitError
		}
		return &ToolResult{
			ForLLM:  output,
			ForUser: output,
			IsError: true,
		}
	}
}

func shellCommand(ctx context.Context, command string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", command)
	}
	return exec.CommandContext(ctx, "sh", "-c", command)
}

func truncateExecOutput(output string) string {
	if output == "" {
		output = "(no output)"
	}

	const maxLen = 10000
	if len(output) > maxLen {
		output = output[:maxLen] + fmt.Sprintf("\n... (truncated, %d more chars)", len(output)-maxLen)
	}
	return output
}

func readOptionalIntArg(args map[string]any, key string, minVal, maxVal int) (int, bool, error) {
	raw, exists := args[key]
	if !exists {
		return 0, false, nil
	}

	n, err := toInt(raw)
	if err != nil {
		return 0, true, fmt.Errorf("%s must be an integer", key)
	}
	if n < minVal || n > maxVal {
		return 0, true, fmt.Errorf("%s must be between %d and %d", key, minVal, maxVal)
	}
	return n, true, nil
}

func (t *ExecTool) guardCommand(command, cwd string) string {
	cmd := strings.TrimSpace(command)
	lower := strings.ToLower(cmd)

	for _, pattern := range t.denyPatterns {
		if pattern.MatchString(lower) {
			return "Command blocked by safety guard (dangerous pattern detected)"
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

		matches := guardPathPattern.FindAllStringIndex(cmd, -1)
		for _, match := range matches {
			raw := cmd[match[0]:match[1]]
			if pathMatchIsEnvAssignmentValue(cmd, match[0]) {
				continue
			}

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

func pathMatchIsEnvAssignmentValue(command string, matchStart int) bool {
	if matchStart <= 0 || matchStart > len(command) {
		return false
	}

	tokenStart := strings.LastIndexAny(command[:matchStart], " \t\r\n") + 1
	if tokenStart < 0 || tokenStart >= matchStart {
		return false
	}

	prefix := command[tokenStart:matchStart]
	eq := strings.Index(prefix, "=")
	if eq <= 0 {
		return false
	}

	return envVarNamePattern.MatchString(prefix[:eq])
}

func (t *ExecTool) SetTimeout(timeout time.Duration) {
	t.timeout = timeout
}

func (t *ExecTool) ProcessManager() *ProcessManager {
	return t.processes
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
