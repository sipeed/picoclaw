package api

import (
	"encoding/json"
	"net/http"
	"log"
)

// VoiceWebhookHandler receives incoming Twilio TwiML or Telnyx Call Control events
// and routes them to the active Agent session if a call is in progress.
func (h *Handler) VoiceWebhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// This is a minimal scaffold for the reverse webhook.
	// You would typically parse `r.ParseForm()` for Twilio or `json.NewDecoder` for Telnyx.
	
	// Mock responding with empty TwiML or 200 OK.
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><Response><Say>PicoClaw Voice Subsystem Active.</Say></Response>`))
	
	log.Printf("Received voice webhook event from %s", r.RemoteAddr)
}
