//go:build !windows

package pid

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
)

// isProcessRunning checks whether a process with the given PID is alive
// on Unix-like systems using signal(0). It additionally verifies that
// the process is actually a picoclaw instance by reading /proc/<pid>/cmdline
// when available. This prevents stale PID files from blocking startup
// when the PID has been reused by an unrelated process.
func isProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal(nil) does not kill the process but checks existence on Unix.
	err = p.Signal(syscall.Signal(0))
	if err != nil {
		var errno syscall.Errno
		// EPERM means the process exists but we are not allowed to signal it.
		if errors.As(err, &errno) && errno == syscall.EPERM {
			// Process exists but we can't signal it.
			// Verify it's actually picoclaw before claiming it's our instance.
			return isPicoclawProcess(pid)
		}
		return false
	}

	// Process exists and we can signal it. Verify it's actually picoclaw
	// to avoid false positives from PID reuse.
	return isPicoclawProcess(pid)
}

// isPicoclawProcess checks whether the process with the given PID is
// a picoclaw gateway instance by reading /proc/<pid>/cmdline (Linux).
// Returns true if the process name contains "picoclaw", or if the check
// cannot be performed (conservative fallback for non-Linux Unix).
func isPicoclawProcess(pid int) bool {
	cmdlinePath := fmt.Sprintf("/proc/%d/cmdline", pid)
	cmdline, err := os.ReadFile(cmdlinePath)
	if err != nil {
		// Cannot verify — conservatively assume it might be our process
		// to avoid breaking on systems without /proc (macOS, BSD, etc.).
		return true
	}

	// cmdline is null-separated; first arg is the executable path.
	// Check if the executable or any arg contains "picoclaw".
	fields := strings.Split(string(cmdline), "\x00")
	for _, f := range fields {
		if strings.Contains(strings.ToLower(f), "picoclaw") {
			return true
		}
	}

	return false
}