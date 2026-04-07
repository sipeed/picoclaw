//go:build linux

package gateway

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

func verifyGatewayProcessIdentity(processID int) error {
	targetExe, err := os.Readlink(filepath.Join("/proc", strconv.Itoa(processID), "exe"))
	if err != nil {
		return fmt.Errorf("failed to inspect gateway process executable (PID: %d): %w", processID, err)
	}

	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to inspect current executable: %w", err)
	}

	if filepath.Base(targetExe) != filepath.Base(currentExe) {
		return fmt.Errorf("pid file points to a non-gateway process (PID: %d)", processID)
	}

	rawCmdline, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(processID), "cmdline"))
	if err != nil {
		return fmt.Errorf("failed to inspect gateway process command line (PID: %d): %w", processID, err)
	}

	argv := bytes.Split(rawCmdline, []byte{0})
	for _, arg := range argv[1:] {
		if string(arg) == "gateway" || string(arg) == "g" {
			return nil
		}
	}

	return fmt.Errorf("pid file points to a non-gateway process (PID: %d)", processID)
}
