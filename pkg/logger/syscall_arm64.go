//go:build linux && (arm64 || arm64be || arm || armbe)
// +build linux
// +build arm64 arm64be arm armbe

package logger

import (
	"syscall"
)

func Dup2(oldfd int, newfd int) error {
	return syscall.Dup3(oldfd, newfd, 0)
}
