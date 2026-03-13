package tools

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	bgMaxLifetime = 45 * time.Minute

	bgRingBufSize = 32 * 1024 // 32KB

	bgInitCapture = 3 * time.Second

	bgMaxProcesses = 10
)

// ringBuffer is a thread-safe circular buffer that retains the most recent bytes.

type ringBuffer struct {
	mu sync.Mutex

	buf []byte

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
	id string

	command string

	cmd *exec.Cmd

	pid int

	startedAt time.Time

	output *ringBuffer

	done chan struct{} // closed when process exits

	exitErr error

	cancel context.CancelFunc // cancels the monitor goroutine
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

		return ErrorResult(

			fmt.Sprintf("maximum background processes reached (%d). Kill an existing one first.", bgMaxProcesses),
		)
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
		id: id,

		command: command,

		cmd: cmd,

		pid: cmd.Process.Pid,

		startedAt: time.Now(),

		output: output,

		done: make(chan struct{}),

		cancel: monitorCancel,
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
		ForLLM: sb.String(),

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
		ForLLM: sb.String(),

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
		ForLLM: msg,

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
