//go:build windows
// +build windows

package logger

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/sys/windows"
)

func initPanicFile(panicFile string) io.WriteCloser {
	file, err := os.OpenFile(panicFile, os.O_WRONLY|os.O_CREATE|os.O_SYNC|os.O_APPEND, 0600)
	if err != nil {
		panic(fmt.Sprintf("error in open panic: %v", err))
	}
	err = windows.SetStdHandle(windows.STD_ERROR_HANDLE, windows.Handle(file.Fd()))
	if err != nil {
		panic(fmt.Sprintf("Failed to redirect stderr to file: %v", err))
	}
	os.Stderr = file
	return file
}
