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
)

func processRunning(pid int) bool {

	if pid <= 0 {

		return false

	}

	// kill(0) can return success for zombie processes too, so inspect /proc

	// state and treat zombies as not-running for timeout cleanup assertions.

	err := syscall.Kill(pid, 0)

	if err != nil && err != syscall.EPERM {

		return false

	}

	data, readErr := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")

	if readErr != nil {

		return false

	}

	raw := string(data)

	end := strings.LastIndex(raw, ")")

	if end == -1 || end+2 >= len(raw) {

		return true // best effort fallback

	}

	fields := strings.Fields(raw[end+2:])

	if len(fields) == 0 {

		return true // best effort fallback

	}

	state := fields[0]

	return state != "Z"

}

func TestShellTool_TimeoutKillsChildProcess(t *testing.T) {

	tool, err := NewExecTool(t.TempDir(), false)

	if err != nil {

		t.Errorf("unable to configure exec tool: %s", err)

	}

	tool.SetTimeout(500 * time.Millisecond)

	args := map[string]any{

		// Spawn a child process that would outlive the shell unless process-group kill is used.

		"command": "sleep 60 & echo $! > child.pid; wait",
	}

	result := tool.Execute(context.Background(), args)

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

		if !processRunning(childPID) {

			return

		}

		time.Sleep(50 * time.Millisecond)

	}

	t.Fatalf("child process %d is still running after timeout", childPID)

}
