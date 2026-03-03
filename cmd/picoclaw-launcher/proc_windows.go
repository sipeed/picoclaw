package main

import (
	"os/exec"
	"syscall"
)

// hideProcessWindow sets CREATE_NO_WINDOW on the process so no console
// window appears and no taskbar entry is created.
func hideProcessWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}
