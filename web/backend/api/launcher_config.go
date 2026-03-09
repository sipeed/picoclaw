package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sipeed/picoclaw/web/backend/launcherconfig"
)

type launcherConfigPayload struct {
	Port         int      `json:"port"`
	Public       bool     `json:"public"`
	AllowedCIDRs []string `json:"allowed_cidrs"`
}

func (h *Handler) registerLauncherConfigRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/system/launcher-config", h.handleGetLauncherConfig)
	mux.HandleFunc("PUT /api/system/launcher-config", h.handleUpdateLauncherConfig)
}

func (h *Handler) launcherConfigPath() string {
	return launcherconfig.PathForAppConfig(h.configPath)
}

func (h *Handler) launcherFallbackConfig() launcherconfig.Config {
	port := h.serverPort
	if port <= 0 {
		port = launcherconfig.DefaultPort
	}
	return launcherconfig.Config{
		Port:         port,
		Public:       h.serverPublic,
		AllowedCIDRs: append([]string(nil), h.serverCIDRs...),
	}
}

func (h *Handler) loadLauncherConfig() (launcherconfig.Config, error) {
	return launcherconfig.Load(h.launcherConfigPath(), h.launcherFallbackConfig())
}

func (h *Handler) handleGetLauncherConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.loadLauncherConfig()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load launcher config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(launcherConfigPayload{
		Port:         cfg.Port,
		Public:       cfg.Public,
		AllowedCIDRs: append([]string(nil), cfg.AllowedCIDRs...),
	})
}

func (h *Handler) handleUpdateLauncherConfig(w http.ResponseWriter, r *http.Request) {
	var payload launcherConfigPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	cfg := launcherconfig.Config{
		Port:         payload.Port,
		Public:       payload.Public,
		AllowedCIDRs: append([]string(nil), payload.AllowedCIDRs...),
	}
	if err := launcherconfig.Validate(cfg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := launcherconfig.Save(h.launcherConfigPath(), cfg); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save launcher config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(launcherConfigPayload{
		Port:         cfg.Port,
		Public:       cfg.Public,
		AllowedCIDRs: append([]string(nil), cfg.AllowedCIDRs...),
	})
}
