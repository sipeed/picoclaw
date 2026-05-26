package tracer

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

// DefaultLogPath returns the default gateway log path.
func DefaultLogPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".picoclaw", "logs", "gateway.log")
}

// Run starts the trace viewer HTTP server.
func Run(port int, logPath, frontendDir string) error {
	if logPath == "" {
		logPath = DefaultLogPath()
	}
	if logPath == "" {
		return fmt.Errorf("could not determine home directory; use --log to specify the gateway log path")
	}

	handler := newMux(logPath, frontendFS(frontendDir))

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("PicoClaw Trace Viewer → http://localhost%s\n", addr)
	fmt.Printf("Reading log: %s\n", logPath)

	return http.ListenAndServe(addr, handler)
}
