package api

import (
	"encoding/json"
	"net/http"

	"github.com/sipeed/picoclaw/pkg/updater"
)

// registerUpdateRoutes registers the self-update endpoint.
func (h *Handler) registerUpdateRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/update", h.handleUpdate)
}

type updateRequest struct {
	URL    string `json:"url,omitempty"`
	Binary string `json:"binary,omitempty"`
}

type updateResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

func (h *Handler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(updateResponse{Status: "error", Message: "method not allowed"})
		return
	}

	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	var req updateRequest
	if err := dec.Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(updateResponse{Status: "error", Message: "invalid request body"})
		return
	}

	binary := req.Binary
	if binary == "" {
		binary = "picoclaw-launcher"
	}

	if err := updater.UpdateSelfFromRelease(req.URL, "", "", binary); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(updateResponse{Status: "error", Message: err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(updateResponse{Status: "ok", Message: "update applied; restart to use new version"})
}
