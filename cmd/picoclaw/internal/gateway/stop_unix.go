//go:build !windows

package gateway

import (
	"os"
	"syscall"
)

// stopProcess sends SIGTERM to the process for graceful shutdown on Unix-like systems.
func stopProcess(process *os.Process) error {
	return process.Signal(syscall.SIGTERM)
}
