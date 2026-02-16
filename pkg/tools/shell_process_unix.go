//go:build !windows

package tools

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func prepareCommandForTreeControl(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}

func killCommandTree(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err == nil {
		if killErr := syscall.Kill(-pgid, syscall.SIGKILL); killErr == nil || killErr == syscall.ESRCH {
			return nil
		} else {
			return fmt.Errorf("failed to kill process group %d: %w", pgid, killErr)
		}
	}

	if killErr := cmd.Process.Kill(); killErr != nil && killErr != os.ErrProcessDone {
		return fmt.Errorf("failed to kill process %d: %w", cmd.Process.Pid, killErr)
	}

	return nil
}
