//go:build !embed

package tracer

import (
	"io/fs"
	"os"
)

// frontendFS returns the frontend dist from disk.
// dir defaults to "tracer/frontend/dist" relative to the repo root.
func frontendFS(dir string) fs.FS {
	if dir == "" {
		dir = "tracer/frontend/dist"
	}
	return os.DirFS(dir)
}
