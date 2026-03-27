package api

import (
	"encoding/json"
	"net/http"

	"github.com/sipeed/picoclaw/pkg/config"
)

type systemVersionResponse struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit,omitempty"`
	BuildTime string `json:"build_time,omitempty"`
	GoVersion string `json:"go_version"`
}

func (h *Handler) registerVersionRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/system/version", h.handleGetVersion)
}

func (h *Handler) handleGetVersion(w http.ResponseWriter, _ *http.Request) {
	buildTime, goVer := config.FormatBuildInfo()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(systemVersionResponse{
		Version:   config.GetVersion(),
		GitCommit: config.GitCommit,
		BuildTime: buildTime,
		GoVersion: goVer,
	})
}
