package api

import (
	"encoding/json"
	"log"
	"net/http"
)

// registerPassphraseRoutes binds the passphrase management endpoints.
func (h *Handler) registerPassphraseRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/credential/passphrase", h.handleSetPassphrase)
	mux.HandleFunc("GET /api/credential/passphrase/status", h.handlePassphraseStatus)
}

// handleSetPassphrase stores the supplied passphrase in the in-memory
// SecureStore, then attempts to auto-start the gateway if it is not running.
//
//	POST /api/credential/passphrase
//	Body: {"passphrase": "..."}
func (h *Handler) handleSetPassphrase(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Passphrase string `json:"passphrase"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if body.Passphrase == "" {
		http.Error(w, "passphrase must not be empty", http.StatusBadRequest)
		return
	}

	h.passphraseStore.SetString(body.Passphrase)

	// Mark state as pending before launching gateway
	h.passphraseMu.Lock()
	h.passphraseLastState = passphraseStatePending
	h.passphraseMu.Unlock()

	// Try to start the gateway now that the passphrase is available.
	// credential.PassphraseProvider points to passphraseStore.Get, so
	// gatewayStartReady() (and all LoadConfig calls) will resolve enc://
	// credentials correctly using the newly stored passphrase.
	go func() {
		gateway.mu.Lock()
		defer gateway.mu.Unlock()
		if isGatewayProcessAliveLocked() {
			return
		}
		pid, err := h.startGatewayLocked("starting")
		if err != nil {
			log.Printf("Failed to start gateway after passphrase unlock: %v", err)
			// startGatewayLocked failed before spawning the process, so the exit
			// goroutine will never run. Transition pending → failed manually.
			// Do NOT clear the passphrase: the failure may be a config issue
			// (e.g. missing default model), not a wrong passphrase. Keeping it
			// allows the user to access /api/config to fix the problem.
			h.passphraseMu.Lock()
			if h.passphraseLastState == passphraseStatePending {
				h.passphraseLastState = passphraseStateFailed
			}
			h.passphraseMu.Unlock()
			return
		}
		log.Printf("Gateway started after passphrase unlock (PID: %d)", pid)
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status": "ok",
	})
}

// handlePassphraseStatus reports whether a passphrase is currently stored.
//
//	GET /api/credential/passphrase/status
func (h *Handler) handlePassphraseStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"passphrase_set": h.passphraseStore.IsSet(),
	})
}
