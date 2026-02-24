package tools

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
)

const (
	bgMaxLifetime  = 45 * time.Minute
	bgRingBufSize  = 32 * 1024 // 32KB
	bgInitCapture  = 3 * time.Second
	bgMaxProcesses = 10
)

// ringBuffer is a thread-safe circular buffer that retains the most recent bytes.
type ringBuffer struct {
	mu   sync.Mutex
	buf  []byte
	size int
}

func newRingBuffer(size int) *ringBuffer {
	return &ringBuffer{size: size}
}

// Write appends data to the ring buffer, dropping oldest bytes if capacity is exceeded.
func (rb *ringBuffer) Write(p []byte) (int, error) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.buf = append(rb.buf, p...)
	if len(rb.buf) > rb.size {
		rb.buf = rb.buf[len(rb.buf)-rb.size:]
	}
	return len(p), nil
}

// String returns the current buffer contents.
func (rb *ringBuffer) String() string {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	return string(rb.buf)
}

// Lines returns the last n lines from the buffer.
func (rb *ringBuffer) Lines(n int) []string {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	if len(rb.buf) == 0 {
		return nil
	}
	all := strings.Split(string(rb.buf), "\n")
	// Remove trailing empty element from final newline
	if len(all) > 0 && all[len(all)-1] == "" {
		all = all[:len(all)-1]
	}
	if n <= 0 || n >= len(all) {
		return all
	}
	return all[len(all)-n:]
}

// Match checks if any line in the buffer matches the given regex pattern.
// Returns the first matching line, or empty string if no match.
func (rb *ringBuffer) Match(pattern *regexp.Regexp) string {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	for _, line := range strings.Split(string(rb.buf), "\n") {
		if pattern.MatchString(line) {
			return line
		}
	}
	return ""
}

// Len returns the current number of bytes in the buffer.
func (rb *ringBuffer) Len() int {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	return len(rb.buf)
}

// bgProcess represents a background process managed by ExecTool.
type bgProcess struct {
	id        string
	command   string
	cmd       *exec.Cmd
	pid       int
	startedAt time.Time
	output    *ringBuffer
	done      chan struct{} // closed when process exits
	exitErr   error
	cancel    context.CancelFunc // cancels the monitor goroutine
}

// isRunning returns true if the process has not yet exited.
func (bp *bgProcess) isRunning() bool {
	select {
	case <-bp.done:
		return false
	default:
		return true
	}
}

type ExecTool struct {
	workingDir          string
	timeout             time.Duration
	denyPatterns        []*regexp.Regexp
	allowPatterns       []*regexp.Regexp
	restrictToWorkspace bool

	// Background process management
	bgMu        sync.Mutex
	bgProcesses map[string]*bgProcess
	bgNextID    int
	bgShutdown  context.CancelFunc // cancels all bg monitor goroutines
	bgCtx       context.Context
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
	regexp.MustCompile(`\bgit\s+checkout\b`),
	regexp.MustCompile(`\bgit\s+switch\b`),
	regexp.MustCompile(`\bssh\b.*@`),
	regexp.MustCompile(`\beval\b`),
	regexp.MustCompile(`\bsource\s+.*\.sh\b`),
}

func NewExecTool(workingDir string, restrict bool) *ExecTool {
	return NewExecToolWithConfig(workingDir, restrict, nil)
}

func NewExecToolWithConfig(workingDir string, restrict bool, config *config.Config) *ExecTool {
	denyPatterns := make([]*regexp.Regexp, 0)

	enableDenyPatterns := true
	if config != nil {
		execConfig := config.Tools.Exec
		enableDenyPatterns = execConfig.EnableDenyPatterns
		if enableDenyPatterns {
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
			} else {
				denyPatterns = append(denyPatterns, defaultDenyPatterns...)
			}
		} else {
			// If deny patterns are disabled, we won't add any patterns, allowing all commands.
			fmt.Println("Warning: deny patterns are disabled. All commands will be allowed.")
		}
	} else {
		denyPatterns = append(denyPatterns, defaultDenyPatterns...)
	}

	bgCtx, bgCancel := context.WithCancel(context.Background())

	return &ExecTool{
		workingDir:          workingDir,
		timeout:             5 * time.Minute,
		denyPatterns:        denyPatterns,
		allowPatterns:       nil,
		restrictToWorkspace: restrict,
		bgProcesses:         make(map[string]*bgProcess),
		bgCtx:               bgCtx,
		bgShutdown:          bgCancel,
	}
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
				"type":        "string",
				"description": "The shell command to execute",
			},
			"working_dir": map[string]any{
				"type":        "string",
				"description": "Optional working directory for the command",
			},
			"background": map[string]any{
				"type":        "boolean",
				"description": "Run the command in the background. Returns immediately with a process ID.",
			},
			"bg_action": map[string]any{
				"type":        "string",
				"enum":        []string{"output", "kill"},
				"description": "Action on a background process: 'output' to get latest output, 'kill' to stop it.",
			},
			"bg_id": map[string]any{
				"type":        "string",
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
				ForLLM:  msg,
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

// executeBg starts a background process and returns immediately.
func (t *ExecTool) executeBg(command, cwd string) *ToolResult {
	t.bgMu.Lock()

	// Check max processes limit
	running := 0
	for _, bp := range t.bgProcesses {
		if bp.isRunning() {
			running++
		}
	}
	if running >= bgMaxProcesses {
		t.bgMu.Unlock()
		return ErrorResult(fmt.Sprintf("maximum background processes reached (%d). Kill an existing one first.", bgMaxProcesses))
	}

	t.bgNextID++
	id := fmt.Sprintf("bg-%d", t.bgNextID)
	t.bgMu.Unlock()

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

	output := newRingBuffer(bgRingBufSize)

	// Use pipes to capture output
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to create stdout pipe: %v", err))
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to create stderr pipe: %v", err))
	}

	if err := cmd.Start(); err != nil {
		return ErrorResult(fmt.Sprintf("failed to start background command: %v", err))
	}

	monitorCtx, monitorCancel := context.WithCancel(t.bgCtx)

	bp := &bgProcess{
		id:        id,
		command:   command,
		cmd:       cmd,
		pid:       cmd.Process.Pid,
		startedAt: time.Now(),
		output:    output,
		done:      make(chan struct{}),
		cancel:    monitorCancel,
	}

	t.bgMu.Lock()
	t.bgProcesses[id] = bp
	t.bgMu.Unlock()

	// io.Copy goroutines: pipe stdout/stderr into ring buffer
	go io.Copy(output, stdoutPipe)
	go io.Copy(output, stderrPipe)

	// cmd.Wait goroutine
	waitDone := make(chan error, 1)
	go func() {
		waitDone <- cmd.Wait()
	}()

	// Monitor goroutine: handles lifetime timer, process exit, and shutdown
	go func() {
		lifetime := time.NewTimer(getBgMaxLifetime())
		defer lifetime.Stop()

		select {
		case err := <-waitDone:
			// Process exited naturally
			bp.exitErr = err
			close(bp.done)
		case <-lifetime.C:
			// Max lifetime exceeded — kill
			_ = terminateProcessTree(cmd)
			select {
			case err := <-waitDone:
				bp.exitErr = err
			case <-time.After(2 * time.Second):
				if cmd.Process != nil {
					_ = cmd.Process.Kill()
				}
				bp.exitErr = <-waitDone
			}
			close(bp.done)
		case <-monitorCtx.Done():
			// Shutdown or explicit kill via cancel
			_ = terminateProcessTree(cmd)
			select {
			case err := <-waitDone:
				bp.exitErr = err
			case <-time.After(2 * time.Second):
				if cmd.Process != nil {
					_ = cmd.Process.Kill()
				}
				bp.exitErr = <-waitDone
			}
			select {
			case <-bp.done:
			default:
				close(bp.done)
			}
		}
	}()

	// Capture initial output (wait up to bgInitCapture)
	time.Sleep(bgInitCapture)
	initialOutput := output.String()

	var sb strings.Builder
	fmt.Fprintf(&sb, "Background process started.\n")
	fmt.Fprintf(&sb, "  id:  %s\n", id)
	fmt.Fprintf(&sb, "  pid: %d\n", bp.pid)
	fmt.Fprintf(&sb, "  cmd: %s\n", command)
	fmt.Fprintf(&sb, "  max lifetime: %s\n", getBgMaxLifetime())
	if initialOutput != "" {
		fmt.Fprintf(&sb, "\nInitial output:\n%s", initialOutput)
	}

	return &ToolResult{
		ForLLM:  sb.String(),
		ForUser: fmt.Sprintf("Background process %s (pid=%d) started: %s", id, bp.pid, command),
	}
}

// handleBgAction handles bg_action=output and bg_action=kill.
func (t *ExecTool) handleBgAction(action, bgID string) *ToolResult {
	if bgID == "" {
		return ErrorResult("bg_id is required for bg_action")
	}

	t.bgMu.Lock()
	bp, ok := t.bgProcesses[bgID]
	t.bgMu.Unlock()

	if !ok {
		return ErrorResult(fmt.Sprintf("background process %q not found", bgID))
	}

	switch action {
	case "output":
		return t.bgOutput(bp)
	case "kill":
		return t.bgKill(bp)
	default:
		return ErrorResult(fmt.Sprintf("unknown bg_action %q (use 'output' or 'kill')", action))
	}
}

func (t *ExecTool) bgOutput(bp *bgProcess) *ToolResult {
	var sb strings.Builder
	fmt.Fprintf(&sb, "[%s] pid=%d %s\n", bp.id, bp.pid, bp.command)

	if bp.isRunning() {
		uptime := time.Since(bp.startedAt).Truncate(time.Second)
		fmt.Fprintf(&sb, "Status: running (uptime: %s, max: %s)\n", uptime, getBgMaxLifetime())
	} else {
		ran := time.Since(bp.startedAt).Truncate(time.Second)
		if bp.exitErr != nil {
			fmt.Fprintf(&sb, "Status: exited with error (ran: %s): %v\n", ran, bp.exitErr)
		} else {
			fmt.Fprintf(&sb, "Status: exited=0 (ran: %s)\n", ran)
		}
	}

	output := bp.output.String()
	if output == "" {
		fmt.Fprintf(&sb, "\n(no output)")
	} else {
		fmt.Fprintf(&sb, "\nOutput:\n%s", output)
	}

	return &ToolResult{
		ForLLM:  sb.String(),
		ForUser: sb.String(),
	}
}

func (t *ExecTool) bgKill(bp *bgProcess) *ToolResult {
	if bp.isRunning() {
		bp.cancel() // triggers monitor goroutine cleanup
		// Wait for process to actually exit
		select {
		case <-bp.done:
		case <-time.After(5 * time.Second):
		}
	}

	t.bgMu.Lock()
	delete(t.bgProcesses, bp.id)
	t.bgMu.Unlock()

	msg := fmt.Sprintf("Background process %s (pid=%d) terminated: %s", bp.id, bp.pid, bp.command)
	return &ToolResult{
		ForLLM:  msg,
		ForUser: msg,
	}
}

// BgProcesses returns a snapshot of background processes for use by bg_monitor.
func (t *ExecTool) BgProcesses() map[string]*bgProcess {
	t.bgMu.Lock()
	defer t.bgMu.Unlock()
	snapshot := make(map[string]*bgProcess, len(t.bgProcesses))
	for k, v := range t.bgProcesses {
		snapshot[k] = v
	}
	return snapshot
}

// RuntimeStatus implements StatusProvider for system prompt injection.
func (t *ExecTool) RuntimeStatus() string {
	t.bgMu.Lock()
	defer t.bgMu.Unlock()

	if len(t.bgProcesses) == 0 {
		return ""
	}

	// Sort by ID for stable output
	ids := make([]string, 0, len(t.bgProcesses))
	for id := range t.bgProcesses {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	var sb strings.Builder
	sb.WriteString("## Background Processes\n\n")
	for _, id := range ids {
		bp := t.bgProcesses[id]
		if bp.isRunning() {
			uptime := time.Since(bp.startedAt).Truncate(time.Second)
			fmt.Fprintf(&sb, "  [%s] pid=%d running  (uptime: %s, max: %s) %s\n",
				id, bp.pid, uptime, getBgMaxLifetime(), bp.command)
		} else {
			ran := time.Since(bp.startedAt).Truncate(time.Second)
			if bp.exitErr != nil {
				fmt.Fprintf(&sb, "  [%s] pid=%d exited=err (ran: %s) %s\n",
					id, bp.pid, ran, bp.command)
			} else {
				fmt.Fprintf(&sb, "  [%s] pid=%d exited=0 (ran: %s) %s\n",
					id, bp.pid, ran, bp.command)
			}
		}
	}
	sb.WriteString("\nUse exec with bg_action=\"output\" / \"kill\" and bg_id to manage.\n")
	sb.WriteString("Use bg_monitor for list/watch/tail operations.")

	return sb.String()
}

// Shutdown terminates all background processes. Call on application exit.
func (t *ExecTool) Shutdown() {
	t.bgShutdown() // cancel all monitor goroutines

	t.bgMu.Lock()
	procs := make([]*bgProcess, 0, len(t.bgProcesses))
	for _, bp := range t.bgProcesses {
		procs = append(procs, bp)
	}
	t.bgMu.Unlock()

	// Wait for all processes to exit
	for _, bp := range procs {
		select {
		case <-bp.done:
		case <-time.After(5 * time.Second):
			// Force kill if still running
			if bp.cmd.Process != nil {
				_ = bp.cmd.Process.Kill()
			}
		}
	}
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
		agentCLI := isAgentCLICommand(cmd)
		for _, token := range strings.Fields(cmd) {
			token = strings.Trim(token, "\"'")

			if !filepath.IsAbs(token) {
				continue
			}

			p := filepath.Clean(token)
			rel, err := filepath.Rel(cwdPath, p)
			if err != nil {
				continue
			}

			if strings.HasPrefix(rel, "..") {
				// Path is outside workspace — allow if it's an executable binary
				if isExecutable(p) {
					continue
				}
				// Agent CLI slash commands: skip non-existent paths
				// (e.g., "/review" is a command, not a file).
				if agentCLI {
					if _, statErr := os.Stat(p); os.IsNotExist(statErr) {
						continue
					}
				}
				return "Command blocked by safety guard (path outside working dir)"
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
	return info.Mode()&0111 != 0
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

// SetBgMaxLifetimeForTest overrides bgMaxLifetime for testing purposes.
// This is exposed only for tests; the returned function restores the original value.
var bgMaxLifetimeOverride time.Duration

func SetBgMaxLifetimeForTest(d time.Duration) func() {
	old := bgMaxLifetimeOverride
	bgMaxLifetimeOverride = d
	return func() { bgMaxLifetimeOverride = old }
}

func getBgMaxLifetime() time.Duration {
	if bgMaxLifetimeOverride > 0 {
		return bgMaxLifetimeOverride
	}
	return bgMaxLifetime
}
