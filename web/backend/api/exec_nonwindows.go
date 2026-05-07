//go:build !windows

package api

import "os/exec"

func launcherExecCommand(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

func applyLauncherProcAttrs(_ *exec.Cmd) {}
