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
		fmt.Fprintf(os.Stdout, "Failed to open panic log file %s: %v\n", panicFile, err)
		return nil
	}
	if err = unix.Dup2(int(file.Fd()), int(os.Stderr.Fd())); err != nil {
		fmt.Fprintf(os.Stdout, "Failed to dup2 panic log: %v\n", err)
		file.Close()
		return nil
	}
	return file
}
