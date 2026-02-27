//go:build !windows

package tools

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/agent/sandbox"
)

func processExists(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}

func TestShellTool_TimeoutKillsChildProcess(t *testing.T) {
	workspace := t.TempDir()
	tool := NewExecTool(workspace, false)
	tool.SetTimeout(500 * time.Millisecond)

	sb := sandbox.NewHostSandbox(workspace, false)
	err := sb.Start(context.Background())
	if err != nil {
		t.Fatalf("failed to start sandbox: %v", err)
	}
	defer sb.Prune(context.Background())

	ctx := sandbox.WithSandbox(context.Background(), sb)

	args := map[string]any{
		// Spawn a child process that would outlive the shell unless process-group kill is used.
		"command": "sleep 60 & echo $! > child.pid; wait",
	}

	result := tool.Execute(ctx, args)
	if !result.IsError {
		t.Fatalf("expected timeout error, got success: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "timed out") {
		t.Fatalf("expected timeout message, got: %s", result.ForLLM)
	}

	childPIDPath := filepath.Join(tool.workingDir, "child.pid")
	data, err := os.ReadFile(childPIDPath)
	if err != nil {
		t.Fatalf("failed to read child pid file: %v", err)
	}

	childPID, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		t.Fatalf("failed to parse child pid: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !processExists(childPID) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("child process %d is still running after timeout", childPID)
}
