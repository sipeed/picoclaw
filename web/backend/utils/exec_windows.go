//go:build windows

package utils

import (
	"os/exec"
	"syscall"
)

// LauncherExecCommand creates an exec.Cmd with the HideWindow attribute set
// to prevent console window flashes on Windows.
func LauncherExecCommand(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	ApplyLauncherProcAttrs(cmd)
	return cmd
}

// ApplyLauncherProcAttrs applies Windows-specific process attributes to hide
// the console window. It is safe to call with a nil cmd.
func ApplyLauncherProcAttrs(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = true
}
