// Standalone tracer binary. Run with:
//
//	go run ./cmd/tracer
//	go build -o tracer-bin ./cmd/tracer
package main

import (
	"flag"
	"log"

	"github.com/sipeed/picoclaw/tracer"
)

func main() {
	port := flag.Int("port", 7331, "port to listen on")
	logPath := flag.String("log", "", "path to gateway.log (default: ~/.picoclaw/logs/gateway.log)")
	frontendDir := flag.String("frontend-dir", "", "path to frontend/dist (dev only)")
	flag.Parse()

	if err := tracer.Run(*port, *logPath, *frontendDir); err != nil {
		log.Fatal(err)
	}
}
