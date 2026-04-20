package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/webhook"
)

// registerWebhookRoutes binds webhook processing endpoints to the ServeMux.
func (h *Handler) registerWebhookRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/webhook/process", h.handleWebhookProcess)
	mux.HandleFunc("GET /api/webhook/status", h.handleWebhookStatus)
}

// handleWebhookProcess accepts asynchronous processing requests
//
//	POST /api/webhook/process
//
// Request body:
//
//	{
//	  "webhook_url": "https://your-app.com/callback",
//	  "payload": {
//	    "data": "any json data"
//	  }
//	}
//
// Response (202 Accepted):
//
//	{
//	  "job_id": "uuid",
//	  "status": "processing",
//	  "timestamp": "2026-04-17T10:00:00Z"
//	}
func (h *Handler) handleWebhookProcess(w http.ResponseWriter, r *http.Request) {
	var req webhook.ProcessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	processor := h.getWebhookProcessor()
	resp, err := processor.Submit(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(resp)
}

// handleWebhookStatus checks the status of a submitted job
//
//	GET /api/webhook/status?job_id=<uuid>
//
// Response (200 OK):
//
//	{
//	  "ID": "uuid",
//	  "WebhookURL": "https://...",
//	  "Status": "processing|completed|failed",
//	  "CreatedAt": "2026-04-17T10:00:00Z",
//	  "CompletedAt": "2026-04-17T10:00:05Z"
//	}
func (h *Handler) handleWebhookStatus(w http.ResponseWriter, r *http.Request) {
	jobID := r.URL.Query().Get("job_id")
	if jobID == "" {
		http.Error(w, "job_id query parameter required", http.StatusBadRequest)
		return
	}

	processor := h.getWebhookProcessor()
	job, exists := processor.GetJob(jobID)
	if !exists {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(job)
}

// getWebhookProcessor returns the webhook processor instance
// It lazily initializes the processor on first use
func (h *Handler) getWebhookProcessor() *webhook.Processor {
	h.webhookMu.Lock()
	defer h.webhookMu.Unlock()

	if h.webhookProcessor == nil {
		// Try to get Pico token to use AI processor
		token, wsURL := h.getPicoWebSocketConfig()

		if token != "" && wsURL != "" {
			// Use PicoClaw AI processor
			logger.InfoC("webhook", "Initializing webhook processor with PicoClaw AI")
			h.webhookProcessor = webhook.CreatePicoClawProcessor(wsURL, token)
		} else {
			// Fallback to example processor
			logger.WarnC("webhook", "Pico WebSocket not available, using example processor")
			h.webhookProcessor = webhook.CreateDefaultProcessor()
		}

		// Start cleanup goroutine
		go h.runWebhookCleanup()
	}

	return h.webhookProcessor
}

// getPicoWebSocketConfig gets the Pico WebSocket URL and composed token
func (h *Handler) getPicoWebSocketConfig() (token string, wsURL string) {
	// Load config to get Pico token and gateway port
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		logger.ErrorC("webhook", fmt.Sprintf("Failed to load config for webhook processor: %v", err))
		return "", ""
	}

	// Get Pico channel config
	bc := cfg.Channels.GetByType(config.ChannelPico)
	if bc == nil || !bc.Enabled {
		return "", ""
	}

	var picoCfg config.PicoSettings
	if err := bc.Decode(&picoCfg); err != nil {
		logger.ErrorC("webhook", fmt.Sprintf("Failed to decode Pico config: %v", err))
		return "", ""
	}

	picoToken := picoCfg.Token.String()
	if picoToken == "" {
		return "", ""
	}

	// Get the composed token (pico-<pid_token><pico_token>)
	composedToken := picoComposedToken("token." + picoToken)
	if composedToken == "" {
		logger.WarnC("webhook", "Failed to compose Pico token (gateway may not be running)")
		return "", ""
	}

	// Construct WebSocket URL to gateway's Pico channel endpoint
	gatewayPort := 18790
	if cfg.Gateway.Port != 0 {
		gatewayPort = cfg.Gateway.Port
	}
	wsURL = fmt.Sprintf("ws://localhost:%d/pico/ws", gatewayPort)

	return composedToken, wsURL
}

// runWebhookCleanup periodically cleans up old webhook jobs
func (h *Handler) runWebhookCleanup() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		h.webhookMu.Lock()
		processor := h.webhookProcessor
		h.webhookMu.Unlock()

		if processor != nil {
			processor.CleanupOldJobs(2 * time.Hour)
			logger.DebugC("webhook", "Cleaned up old webhook jobs")
		}
	}
}
