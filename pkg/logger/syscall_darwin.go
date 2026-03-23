//go:build darwin
// +build darwin

package logger

import (
	"syscall"
)

func Dup2(oldfd int, newfd int) error {
	return syscall.Dup2(oldfd, newfd)
}
