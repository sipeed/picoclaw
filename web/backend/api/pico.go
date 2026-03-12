package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
)

// registerPicoRoutes binds Pico Channel management endpoints to the ServeMux.
func (h *Handler) registerPicoRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/pico/token", h.handleGetPicoToken)
	mux.HandleFunc("POST /api/pico/token", h.handleRegenPicoToken)
	mux.HandleFunc("POST /api/pico/setup", h.handlePicoSetup)
}

// handleGetPicoToken returns the current WS token and URL for the frontend.
//
//	GET /api/pico/token
func (h *Handler) handleGetPicoToken(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	wsURL := h.buildWsURL(r, cfg)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"token":   cfg.Channels.Pico.Token,
		"ws_url":  wsURL,
		"enabled": cfg.Channels.Pico.Enabled,
	})
}

// handleRegenPicoToken generates a new Pico WebSocket token and saves it.
//
//	POST /api/pico/token
func (h *Handler) handleRegenPicoToken(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	token := generateSecureToken()
	cfg.Channels.Pico.Token = token

	if err := config.SaveConfig(h.configPath, cfg); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save config: %v", err), http.StatusInternalServerError)
		return
	}

	wsURL := h.buildWsURL(r, cfg)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"token":  token,
		"ws_url": wsURL,
	})
}

// ensurePicoChannel checks if the Pico Channel is properly configured and
// enables it with sensible defaults if not. Returns true if config was changed.
func (h *Handler) ensurePicoChannel() (bool, error) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		return false, fmt.Errorf("failed to load config: %w", err)
	}

	changed := false

	if !cfg.Channels.Pico.Enabled {
		cfg.Channels.Pico.Enabled = true
		changed = true
	}

	if cfg.Channels.Pico.Token == "" {
		cfg.Channels.Pico.Token = generateSecureToken()
		changed = true
	}

	if !cfg.Channels.Pico.AllowTokenQuery {
		cfg.Channels.Pico.AllowTokenQuery = true
		changed = true
	}

	// Make sure origins are allowed (frontend might be running on a different port like 5173 during dev)
	if len(cfg.Channels.Pico.AllowOrigins) == 0 {
		cfg.Channels.Pico.AllowOrigins = []string{"*"}
		changed = true
	}

	if changed {
		if err := config.SaveConfig(h.configPath, cfg); err != nil {
			return false, fmt.Errorf("failed to save config: %w", err)
		}
	}

	return changed, nil
}

// handlePicoSetup automatically configures everything needed for the Pico Channel to work.
//
//	POST /api/pico/setup
func (h *Handler) handlePicoSetup(w http.ResponseWriter, r *http.Request) {
	changed, err := h.ensurePicoChannel()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	wsURL := h.buildWsURL(r, cfg)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"token":   cfg.Channels.Pico.Token,
		"ws_url":  wsURL,
		"enabled": true,
		"changed": changed,
	})
}

// generateSecureToken creates a random 32-character hex string.
func generateSecureToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to something pseudo-random if crypto/rand fails
		return fmt.Sprintf("pico_%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
