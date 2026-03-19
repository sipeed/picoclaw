package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/mediacache"
)

func (h *Handler) registerMediaCacheRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/media-cache", h.handleMediaCache)
	mux.HandleFunc("/api/media-cache/", h.handleMediaCacheContent)
}

type mediaCacheEntryJSON struct {
	Hash       string `json:"hash"`
	Type       string `json:"type"`
	Result     string `json:"result"`
	FilePath   string `json:"file_path,omitempty"`
	Pages      int    `json:"pages,omitempty"`
	CreatedAt  string `json:"created_at"`
	AccessedAt string `json:"accessed_at"`
}

func (h *Handler) openMediaCache() (*mediacache.Cache, error) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		return nil, err
	}
	ws := cfg.WorkspacePath()
	return mediacache.Open(filepath.Join(ws, "media_cache.db"))
}

// handleMediaCache lists all media cache entries.
func (h *Handler) handleMediaCache(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	mc, err := h.openMediaCache()
	if err != nil {
		http.Error(w, `{"error":"media cache not available"}`, http.StatusServiceUnavailable)
		return
	}
	defer mc.Close()

	typeFilter := r.URL.Query().Get("type")
	entries, err := mc.List(typeFilter)
	if err != nil {
		http.Error(w, `{"error":"failed to list cache entries"}`, http.StatusInternalServerError)
		return
	}

	result := make([]mediaCacheEntryJSON, 0, len(entries))
	for _, e := range entries {
		result = append(result, mediaCacheEntryJSON{
			Hash:       e.Hash,
			Type:       e.Type,
			Result:     e.Result,
			FilePath:   e.FilePath,
			Pages:      e.Pages,
			CreatedAt:  e.CreatedAt,
			AccessedAt: e.AccessedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleMediaCacheContent serves the full file content for a PDF OCR entry.
// GET /api/media-cache/{hash}
func (h *Handler) handleMediaCacheContent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	hash := filepath.Base(r.URL.Path)
	if hash == "" || hash == "media-cache" {
		http.Error(w, `{"error":"hash required"}`, http.StatusBadRequest)
		return
	}

	mc, err := h.openMediaCache()
	if err != nil {
		http.Error(w, `{"error":"media cache not available"}`, http.StatusServiceUnavailable)
		return
	}
	defer mc.Close()

	entry, ok := mc.GetEntry(hash, mediacache.TypePDFOCR)
	if !ok {
		// Try image_desc
		result, ok := mc.Get(hash, mediacache.TypeImageDesc)
		if !ok {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"hash":    hash,
			"type":    mediacache.TypeImageDesc,
			"content": result,
		})
		return
	}

	// Read the full markdown file
	content, err := os.ReadFile(entry.FilePath)
	if err != nil {
		http.Error(w, `{"error":"file not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"hash":      hash,
		"type":      mediacache.TypePDFOCR,
		"content":   string(content),
		"file_path": entry.FilePath,
		"pages":     entry.Pages,
	})
}
