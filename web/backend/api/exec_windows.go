//go:build windows

package api

import (
	"os/exec"
	"syscall"
)

func launcherExecCommand(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	applyLauncherProcAttrs(cmd)
	return cmd
}

func applyLauncherProcAttrs(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = true
}
