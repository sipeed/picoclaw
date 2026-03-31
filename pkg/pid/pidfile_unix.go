//go:build !windows

package pid

import (
	"os"
	"syscall"
)

// isProcessRunning checks whether a process with the given PID is alive
// on Unix-like systems using signal(0).
func isProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal(nil) does not kill the process but checks existence on Unix.
	return p.Signal(syscall.Signal(0)) == nil
}
