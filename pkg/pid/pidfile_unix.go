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

// isPicoclawProcess checks whether the process with the given PID is actually
// a picoclaw gateway process by reading /proc/<pid>/comm.
func isPicoclawProcess(pid int) bool {
	if pid <= 0 {
		return false
	}
	// Read /proc/<pid>/comm to get the process name
	comm, err := os.ReadFile(procCommPath(pid))
	if err != nil {
		return false
	}
	// Check if the process name contains "picoclaw"
	return containsString(string(comm), "picoclaw")
}

// procCommPath returns the path to the comm file for a given PID.
func procCommPath(pid int) string {
	return "/proc/" + itoa(pid) + "/comm"
}

// itoa converts an integer to its string representation.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	negative := n < 0
	if negative {
		n = -n
	}
	digits := make([]byte, 0, 10)
	for n > 0 {
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}
	// Reverse the digits
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}
	if negative {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}

// containsString checks if s contains substr.
func containsString(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
