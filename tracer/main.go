// tracer is a debug trace viewer for the PicoClaw gateway.
//
// It reads ~/.picoclaw/logs/gateway.log (JSON Lines format) and serves a
// React UI that shows per-turn LLM calls, system prompts, messages, tool
// definitions, and tool executions.
//
// Usage:
//
//	go run ./tracer                          # dev mode, serves frontend from disk
//	go build -tags embed -o tracer ./tracer  # single binary with embedded frontend
//	./tracer --port 7331
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func defaultLogPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".picoclaw", "logs", "gateway.log")
}

func main() {
	port := flag.Int("port", 7331, "port to listen on")
	logPath := flag.String("log", defaultLogPath(), "path to picoclaw gateway.log")
	flag.Parse()

	if *logPath == "" {
		log.Fatal("could not determine home directory; pass --log explicitly")
	}

	frontend := frontendFS()
	handler := newMux(*logPath, frontend)

	addr := fmt.Sprintf(":%d", *port)
	fmt.Printf("PicoClaw Trace Viewer → http://localhost%s\n", addr)
	fmt.Printf("Reading log: %s\n", *logPath)

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}
