package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"
)

var panicWriter io.WriteCloser

func InitPanic(filePath string) (func(), error) {
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}
	writer := initPanicFile(filePath)
	if writer == nil {
		return nil, fmt.Errorf("failed to create log file: %s", filePath)
	}
	if panicWriter != nil {
		_ = panicWriter.Close()
	}
	panicWriter = writer
	return func() {
		defer func() {
			writer.Close()
			panicWriter = nil
		}()
		if err := recover(); err != nil {
			RecoverPanicNoExit(err)

			os.Exit(1)
		}
	}, nil
}

func RecoverPanicNoExit(err any) {
	if panicWriter == nil {
		Errorf("panicWriter is nil, should not happen")
		return
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	stack := debug.Stack()
	logMsg := "\n\n====================\n[" + now + "] PANIC OCCURRED: " + fmt.Sprintf(
		"%v",
		err,
	) + "\n" + string(
		stack,
	)

	panicWriter.Write([]byte(logMsg))
}
