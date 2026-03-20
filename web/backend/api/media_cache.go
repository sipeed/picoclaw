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

// handleMediaCache dispatches GET (list) and DELETE (clear all) for media cache.
func (h *Handler) handleMediaCache(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listMediaCache(w, r)
	case http.MethodDelete:
		h.deleteAllMediaCache(w, r)
	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func (h *Handler) listMediaCache(w http.ResponseWriter, r *http.Request) {
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

func (h *Handler) deleteAllMediaCache(w http.ResponseWriter, _ *http.Request) {
	mc, err := h.openMediaCache()
	if err != nil {
		http.Error(w, `{"error":"media cache not available"}`, http.StatusServiceUnavailable)
		return
	}
	defer mc.Close()

	removed, err := mc.DeleteAll()
	if err != nil {
		http.Error(w, `{"error":"failed to delete cache"}`, http.StatusInternalServerError)
		return
	}

	// Clean up .ocr_cache directory
	cfg, _ := config.LoadConfig(h.configPath)
	if cfg != nil {
		ocrDir := filepath.Join(cfg.WorkspacePath(), ".ocr_cache")
		os.RemoveAll(ocrDir)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"deleted": removed})
}

// handleMediaCacheContent dispatches GET (content) and DELETE (single entry).
// /api/media-cache/{hash}
func (h *Handler) handleMediaCacheContent(w http.ResponseWriter, r *http.Request) {
	hash := filepath.Base(r.URL.Path)
	if hash == "" || hash == "media-cache" {
		http.Error(w, `{"error":"hash required"}`, http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getMediaCacheContent(w, hash)
	case http.MethodDelete:
		h.deleteMediaCacheEntry(w, hash)
	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func (h *Handler) getMediaCacheContent(w http.ResponseWriter, hash string) {
	mc, err := h.openMediaCache()
	if err != nil {
		http.Error(w, `{"error":"media cache not available"}`, http.StatusServiceUnavailable)
		return
	}
	defer mc.Close()

	entry, ok := mc.GetEntry(hash, mediacache.TypePDFOCR)
	if !ok {
		entry, ok = mc.GetEntry(hash, mediacache.TypePDFText)
	}
	if !ok {
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

func (h *Handler) deleteMediaCacheEntry(w http.ResponseWriter, hash string) {
	mc, err := h.openMediaCache()
	if err != nil {
		http.Error(w, `{"error":"media cache not available"}`, http.StatusServiceUnavailable)
		return
	}
	defer mc.Close()

	// Delete all types for this hash, clean up files
	for _, t := range []string{mediacache.TypePDFOCR, mediacache.TypePDFText, mediacache.TypeImageDesc} {
		entry, err := mc.Delete(hash, t)
		if err != nil {
			continue
		}
		if entry.FilePath != "" {
			// Remove the per-hash subdirectory if it exists
			dir := filepath.Dir(entry.FilePath)
			if filepath.Base(dir) == hash {
				os.RemoveAll(dir)
			} else {
				os.Remove(entry.FilePath)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
