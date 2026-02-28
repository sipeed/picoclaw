//go:build !windows

package sandbox

import (
	"os/exec"

	"golang.org/x/sys/unix"
)

func prepareCommandForTermination(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	cmd.SysProcAttr = &unix.SysProcAttr{Setpgid: true}
}

func terminateProcessTree(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	pid := cmd.Process.Pid
	if pid <= 0 {
		return nil
	}

	// Kill the entire process group spawned by the shell command.
	_ = unix.Kill(-pid, unix.SIGKILL)
	// Fallback kill on the shell process itself.
	_ = cmd.Process.Kill()
	return nil
}
