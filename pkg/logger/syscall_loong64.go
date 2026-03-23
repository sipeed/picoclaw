//go:build linux && loong64
// +build linux,loong64

package logger

import (
	"syscall"
)

func Dup2(oldfd int, newfd int) error {
	return syscall.Dup3(oldfd, newfd, 0)
}
