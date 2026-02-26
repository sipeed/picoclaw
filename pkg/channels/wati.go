package channels

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

const (
	watiDefaultAPIBase = "https://live-mt-server.wati.io"
	watiSendEndpoint   = "/api/ext/v3/conversations/messages/text"
)

// WATIChannel implements the Channel interface for WATI WhatsApp Business API
// using HTTP webhook for receiving messages and REST API for sending messages.
type WATIChannel struct {
	*BaseChannel
	config     config.WATIConfig
	httpServer *http.Server
	apiBase    string
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewWATIChannel creates a new WATI channel instance.
func NewWATIChannel(cfg config.WATIConfig, messageBus *bus.MessageBus) (*WATIChannel, error) {
	if cfg.APIToken == "" {
		return nil, fmt.Errorf("wati api_token is required")
	}

	base := NewBaseChannel("wati", cfg, messageBus, cfg.AllowFrom)

	apiBase := cfg.APIBaseURL
	if apiBase == "" {
		apiBase = watiDefaultAPIBase
	}
	apiBase = strings.TrimRight(apiBase, "/")

	return &WATIChannel{
		BaseChannel: base,
		config:      cfg,
		apiBase:     apiBase,
	}, nil
}

// Start launches the HTTP webhook server.
func (c *WATIChannel) Start(ctx context.Context) error {
	logger.InfoC("wati", "Starting WATI channel (Webhook Mode)")

	c.ctx, c.cancel = context.WithCancel(ctx)

	mux := http.NewServeMux()
	path := c.config.WebhookPath
	if path == "" {
		path = "/webhook/wati"
	}
	mux.HandleFunc(path, c.webhookHandler)

	addr := fmt.Sprintf("%s:%d", c.config.WebhookHost, c.config.WebhookPort)
	c.httpServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		logger.InfoCF("wati", "WATI webhook server listening", map[string]any{
			"addr": addr,
			"path": path,
		})
		if err := c.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.ErrorCF("wati", "Webhook server error", map[string]any{
				"error": err.Error(),
			})
		}
	}()

	c.setRunning(true)
	logger.InfoC("wati", "WATI channel started (Webhook Mode)")
	return nil
}

// Stop gracefully shuts down the HTTP server.
func (c *WATIChannel) Stop(ctx context.Context) error {
	logger.InfoC("wati", "Stopping WATI channel")

	if c.cancel != nil {
		c.cancel()
	}

	if c.httpServer != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := c.httpServer.Shutdown(shutdownCtx); err != nil {
			logger.ErrorCF("wati", "Webhook server shutdown error", map[string]any{
				"error": err.Error(),
			})
		}
	}

	c.setRunning(false)
	logger.InfoC("wati", "WATI channel stopped")
	return nil
}

// webhookHandler handles incoming WATI webhook requests.
func (c *WATIChannel) webhookHandler(w http.ResponseWriter, r *http.Request) {
	// GET: hub.challenge verification
	if r.Method == http.MethodGet {
		c.handleVerification(w, r)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.ErrorCF("wati", "Failed to read request body", map[string]any{
			"error": err.Error(),
		})
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Return 200 immediately
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))

	go c.processWebhook(body)
}

// handleVerification handles GET webhook verification (hub.challenge).
func (c *WATIChannel) handleVerification(w http.ResponseWriter, r *http.Request) {
	challenge := r.URL.Query().Get("hub.challenge")
	if challenge != "" {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(challenge))
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}
}

// processWebhook parses and handles a WATI webhook payload.
func (c *WATIChannel) processWebhook(body []byte) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		logger.ErrorCF("wati", "Failed to parse webhook payload", map[string]any{
			"error": err.Error(),
		})
		return
	}

	// WATI sends different payload structures — normalize fields
	text := getStringField(payload, "text", "message.text", "message.body")
	waID := getStringField(payload, "waId", "wa_id", "from")
	fromMe := getBoolField(payload, "fromMe", "from_me", "owner")

	if text == "" || fromMe {
		logger.DebugCF("wati", "Skipping message", map[string]any{
			"reason": map[bool]string{true: "no text", false: "fromMe"}[text == ""],
			"wa_id":  waID,
		})
		return
	}

	senderID := waID
	chatID := waID

	metadata := map[string]string{
		"platform":  "wati",
		"peer_kind": "direct",
		"peer_id":   waID,
	}

	logger.DebugCF("wati", "Received message", map[string]any{
		"sender_id": senderID,
		"wa_id":     waID,
	})

	c.HandleMessage(senderID, chatID, text, nil, metadata)
}

// Send sends a text message via the WATI v3 API.
func (c *WATIChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return fmt.Errorf("wati channel not running")
	}

	payload := map[string]string{
		"target": msg.ChatID,
		"text":   msg.Content,
	}

	endpoint := c.apiBase + watiSendEndpoint
	if err := c.callAPI(ctx, endpoint, payload); err != nil {
		return fmt.Errorf("failed to send WATI message: %w", err)
	}

	logger.DebugCF("wati", "Message sent", map[string]any{
		"chat_id": msg.ChatID,
	})

	return nil
}

// callAPI makes an authenticated POST request to the WATI API.
func (c *WATIChannel) callAPI(ctx context.Context, endpoint string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("WATI API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// getStringField extracts a string value from a map, trying multiple keys.
// Supports dotted keys like "message.text" for nested lookups.
func getStringField(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if strings.Contains(key, ".") {
			parts := strings.SplitN(key, ".", 2)
			if nested, ok := m[parts[0]].(map[string]any); ok {
				if val := getStringField(nested, parts[1]); val != "" {
					return val
				}
			}
			continue
		}
		if val, ok := m[key]; ok {
			switch v := val.(type) {
			case string:
				return v
			case float64:
				return fmt.Sprintf("%.0f", v)
			}
		}
	}
	return ""
}

// getBoolField extracts a boolean value from a map, trying multiple keys.
func getBoolField(m map[string]any, keys ...string) bool {
	for _, key := range keys {
		if val, ok := m[key]; ok {
			switch v := val.(type) {
			case bool:
				return v
			}
		}
	}
	return false
}
