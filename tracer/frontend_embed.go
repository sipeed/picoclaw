//go:build embed

package main

import (
	"embed"
	"io/fs"
	"log"
)

//go:embed all:frontend/dist
var _embeddedFrontend embed.FS

// frontendFS returns the embedded frontend dist.
// Built with: go build -tags embed ./tracer
func frontendFS() fs.FS {
	sub, err := fs.Sub(_embeddedFrontend, "frontend/dist")
	if err != nil {
		log.Fatalf("embed: %v", err)
	}
	return sub
}
