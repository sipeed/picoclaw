//go:build windows

package tools

import (
	"io"
	"os/exec"
	"strconv"
)

func killProcessGroup(pid int) error {
	_ = exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(pid)).Run()
	return nil
}

func (s *ProcessSession) handleReadError(err error) error {
	if err == io.EOF {
		return nil
	}
	return err
}
