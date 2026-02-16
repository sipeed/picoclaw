//go:build windows

package tools

import (
	"fmt"
	"os/exec"
	"strconv"
	"syscall"
)

func prepareCommandForTreeControl(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}

func killCommandTree(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	pid := strconv.Itoa(cmd.Process.Pid)
	killCmd := exec.Command("taskkill", "/T", "/F", "/PID", pid)
	if err := killCmd.Run(); err != nil {
		return fmt.Errorf("taskkill failed for pid %s: %w", pid, err)
	}

	return nil
}
