//go:build linux && amd64
// +build linux,amd64

package logger

import (
	"syscall"
)

func Dup2(oldfd int, newfd int) error {
	return syscall.Dup2(oldfd, newfd)
}
