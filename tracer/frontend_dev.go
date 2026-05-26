//go:build !embed

package main

import (
	"io/fs"
	"os"
)

// frontendFS returns the frontend dist directory from disk.
// Used in development — run `make build-frontend` once to populate frontend/dist.
func frontendFS() fs.FS {
	return os.DirFS("frontend/dist")
}
