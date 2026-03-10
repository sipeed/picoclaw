package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"path"
	"strings"
)

//go:embed all:dist
var frontendFS embed.FS

// registerEmbedRoutes sets up the HTTP handler to serve the embedded frontend files
func registerEmbedRoutes(mux *http.ServeMux) {
	// Attempt to get the subdirectory 'dist' where Vite usually builds
	subFS, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		// Log a warning if dist doesn't exist yet (e.g., during development before a frontend build)
		log.Printf(
			"Warning: no 'dist' folder found in embedded frontend. " +
				"Ensure you run `pnpm build:backend` in the frontend directory " +
				"before building the Go backend.",
		)
		return
	}

	fileServer := http.FileServer(http.FS(subFS))

	// Serve static assets and fallback to index.html for SPA routes.
	mux.Handle(
		"/",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				http.NotFound(w, r)
				return
			}

			// Keep unknown API paths as 404 instead of falling back to SPA entry.
			if r.URL.Path == "/api" || strings.HasPrefix(r.URL.Path, "/api/") {
				http.NotFound(w, r)
				return
			}

			cleanPath := path.Clean(strings.TrimPrefix(r.URL.Path, "/"))
			if cleanPath == "." {
				cleanPath = ""
			}

			// Existing static files/directories should be served directly.
			if cleanPath != "" {
				if _, statErr := fs.Stat(subFS, cleanPath); statErr == nil {
					fileServer.ServeHTTP(w, r)
					return
				}
				// Missing asset-like paths should remain 404.
				if strings.Contains(path.Base(cleanPath), ".") {
					fileServer.ServeHTTP(w, r)
					return
				}
			}

			indexReq := r.Clone(r.Context())
			indexReq.URL.Path = "/"
			fileServer.ServeHTTP(w, indexReq)
		}),
	)
}
