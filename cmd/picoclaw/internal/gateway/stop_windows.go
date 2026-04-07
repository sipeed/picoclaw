//go:build windows

package gateway

import (
	"os"
)

// stopProcess kills the process on Windows (SIGTERM is not supported).
func stopProcess(process *os.Process) error {
	return process.Kill()
}
