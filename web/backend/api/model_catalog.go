package api

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/fileutil"
)

// CatalogModel represents a single model entry in a saved catalog.
type CatalogModel struct {
	ID      string         `json:"id"`
	OwnedBy string         `json:"owned_by,omitempty"`
	Extra   map[string]any `json:"extra,omitempty"`
}

// CatalogEntry is a saved list of upstream models fetched for a specific provider+key combination.
type CatalogEntry struct {
	ID         string         `json:"id"`
	Provider   string         `json:"provider"`
	APIBase    string         `json:"api_base"`
	APIKeyMask string         `json:"api_key_mask"`
	Models     []CatalogModel `json:"models"`
	FetchedAt  string         `json:"fetched_at"`
}

// CatalogStore holds all saved model catalogs.
type CatalogStore struct {
	Entries map[string]*CatalogEntry `json:"entries"`
}

func catalogFilePath() string {
	return filepath.Join(config.GetHome(), "model_catalogs.json")
}

// generateCatalogKey creates a deterministic key for a provider+base+key combination.
func generateCatalogKey(provider, apiBase, apiKey string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	apiBase = strings.TrimRight(strings.TrimSpace(apiBase), "/")
	hash := sha256.Sum256([]byte(apiKey))
	return fmt.Sprintf("%s|%s|%x", provider, apiBase, hash[:6])
}

// maskAPIKeyValue masks an API key for display, keeping first 4 and last 4 chars.
func maskAPIKeyValue(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

func loadCatalogs() (*CatalogStore, error) {
	path := catalogFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &CatalogStore{Entries: make(map[string]*CatalogEntry)}, nil
		}
		return nil, err
	}
	var store CatalogStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}
	if store.Entries == nil {
		store.Entries = make(map[string]*CatalogEntry)
	}
	return &store, nil
}

func saveCatalogs(store *CatalogStore) error {
	path := catalogFilePath()
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return fileutil.WriteFileAtomic(path, data, 0o600)
}

// SaveCatalog persists a fetched model list for a given provider+key combination.
// If a catalog with the same key already exists, it is updated.
func SaveCatalog(provider, apiBase, apiKey string, models []CatalogModel) error {
	store, err := loadCatalogs()
	if err != nil {
		return err
	}
	key := generateCatalogKey(provider, apiBase, apiKey)
	store.Entries[key] = &CatalogEntry{
		ID:         key,
		Provider:   strings.ToLower(strings.TrimSpace(provider)),
		APIBase:    strings.TrimRight(strings.TrimSpace(apiBase), "/"),
		APIKeyMask: maskAPIKeyValue(apiKey),
		Models:     models,
		FetchedAt:  time.Now().UTC().Format(time.RFC3339),
	}
	return saveCatalogs(store)
}

// handleListCatalogs returns all saved model catalogs.
//
//	GET /api/models/catalog
func (h *Handler) handleListCatalogs(w http.ResponseWriter, r *http.Request) {
	store, err := loadCatalogs()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load catalogs: %v", err), http.StatusInternalServerError)
		return
	}

	entries := make([]*CatalogEntry, 0, len(store.Entries))
	for _, e := range store.Entries {
		entries = append(entries, e)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"entries": entries,
		"total":   len(entries),
	})
}

// handleDeleteCatalog deletes a saved model catalog by ID.
//
//	DELETE /api/models/catalog/{id}
func (h *Handler) handleDeleteCatalog(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	store, err := loadCatalogs()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load catalogs: %v", err), http.StatusInternalServerError)
		return
	}

	if _, ok := store.Entries[id]; !ok {
		http.Error(w, "catalog not found", http.StatusNotFound)
		return
	}

	delete(store.Entries, id)
	if err := saveCatalogs(store); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save catalogs: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
