//go:build !windows

package utils

import "os/exec"

// LauncherExecCommand creates an exec.Cmd. On non-Windows platforms, this is
// a simple wrapper around exec.Command.
func LauncherExecCommand(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

// ApplyLauncherProcAttrs is a no-op on non-Windows platforms.
func ApplyLauncherProcAttrs(_ *exec.Cmd) {}
