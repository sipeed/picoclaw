package api

import (
	"encoding/json"
	"net/http"

	"github.com/sipeed/picoclaw/pkg/tools"
)

type uiLanguageRequest struct {
	Language string `json:"language"`
}

func (h *Handler) registerUIRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/ui/language", h.handleSetUILanguage)
}

func (h *Handler) handleSetUILanguage(w http.ResponseWriter, r *http.Request) {
	var req uiLanguageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	tools.SetPreferredWebSearchLanguage(req.Language)
	w.WriteHeader(http.StatusNoContent)
}
