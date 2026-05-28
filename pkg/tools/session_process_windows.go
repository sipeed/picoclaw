//go:build windows

package tools

import (
	"os/exec"
	"strconv"
)

func killProcessGroup(pid int) error {
	_ = exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(pid)).Run()
	return nil
}
