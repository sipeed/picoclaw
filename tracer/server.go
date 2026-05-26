package tracer

import (
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"strings"
)

func newMux(logPath string, frontend fs.FS) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"ok": true, "log": logPath})
	})

	mux.HandleFunc("/api/traces", func(w http.ResponseWriter, r *http.Request) {
		turns, err := ParseLog(logPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, turns)
	})

	mux.HandleFunc("/api/traces/", func(w http.ResponseWriter, r *http.Request) {
		turnID := strings.TrimPrefix(r.URL.Path, "/api/traces/")
		if turnID == "" {
			http.NotFound(w, r)
			return
		}
		turns, err := ParseLog(logPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for _, t := range turns {
			if t.TurnID == turnID {
				writeJSON(w, t)
				return
			}
		}
		http.NotFound(w, r)
	})

	if frontend != nil {
		mux.Handle("/", http.FileServer(http.FS(frontend)))
	} else {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "frontend not built — run: make build-frontend", http.StatusNotFound)
		})
	}

	return loggingMiddleware(corsMiddleware(mux))
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("json encode error: %v", err)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
