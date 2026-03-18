//go:build !windows

package tools

import (
	"io"
	"syscall"
)

func killProcessGroup(pid int) error {
	if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil {
		_ = syscall.Kill(pid, syscall.SIGKILL)
	}
	return nil
}

func (s *ProcessSession) handleReadError(err error) error {
	if err == io.EOF {
		return nil
	}
	if err == syscall.EIO {
		return nil
	}
	return err
}
