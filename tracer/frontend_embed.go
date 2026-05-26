//go:build embed

package tracer

import (
	"embed"
	"io/fs"
	"log"
)

//go:embed all:frontend/dist
var _embeddedFrontend embed.FS

// frontendFS returns the embedded frontend dist.
// dir is ignored when built with -tags embed.
func frontendFS(_ string) fs.FS {
	sub, err := fs.Sub(_embeddedFrontend, "frontend/dist")
	if err != nil {
		log.Fatalf("tracer: embed error: %v", err)
	}
	return sub
}
