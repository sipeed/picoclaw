//go:build windows

package tools

import "os/exec"

func setSysProcAttrForPty(cmd *exec.Cmd) {
	// Windows doesn't support Setsid, and PTY is not available on Windows anyway.
	// This function is a no-op for Windows builds.
}
