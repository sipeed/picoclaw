package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"
)

func InitPanic(filePath string) (func(), error) {
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}
	writer := initPanicFile(filePath)
	if writer == nil {
		return nil, fmt.Errorf("failed to create log file: %s", filePath)
	}
	return func() {
		defer writer.Close()
		if err := recover(); err != nil {
			now := time.Now().Format("2006-01-02 15:04:05")
			stack := debug.Stack()
			logMsg := "\n\n====================\n[" + now + "] PANIC OCCURRED: " + fmt.Sprintf(
				"%v",
				err,
			) + "\n" + string(
				stack,
			)

			writer.Write([]byte(logMsg))

			os.Exit(1)
		}
	}, nil
}
