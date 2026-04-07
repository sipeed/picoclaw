//go:build !windows

package pid

import (
	"errors"
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
	err = p.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}
	var errno syscall.Errno
	// EPERM means the process exists but we are not allowed to signal it.
	return errors.As(err, &errno) && errno == syscall.EPERM
}
