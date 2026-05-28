//go:build !windows

package tools

import (
	"os/exec"
	"syscall"
)

func setSysProcAttrForPty(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
