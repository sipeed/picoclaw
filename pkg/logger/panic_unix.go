//go:build !windows

package logger

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/sys/unix"
)

func initPanicFile(panicFile string) io.WriteCloser {
	file, err := os.OpenFile(panicFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND|os.O_SYNC, 0o600)
	if err != nil {
		panic(fmt.Sprintf("error in open panic: %v", err))
	}
	if err = unix.Dup2(int(file.Fd()), int(os.Stderr.Fd())); err != nil {
		panic(fmt.Sprintf("error in syscall.Dup2: %v", err))
	}
	return file
}
