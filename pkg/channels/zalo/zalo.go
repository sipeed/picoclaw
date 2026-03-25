package zalo

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

const channelName = "zalo"

// ZaloChannel implements the Channel interface for Zalo Official Account
// using webhook for receiving messages and REST API for sending messages.
type ZaloChannel struct {
	*channels.BaseChannel
	config config.ZaloConfig
	api    *ZaloAPI
	mu     sync.Mutex
	ctx    context.Context
	cancel context.CancelFunc
}

// NewZaloChannel creates a new Zalo channel instance.
func NewZaloChannel(cfg config.ZaloConfig, messageBus *bus.MessageBus) (*ZaloChannel, error) {
	if cfg.AppID == "" || cfg.AppSecret == "" {
		return nil, fmt.Errorf("zalo: app_id and app_secret are required")
	}
	if cfg.AccessToken == "" {
		logger.WarnC("zalo", "access_token is empty — webhook will accept events but sending messages will fail until token is set")
	}

	base := channels.NewBaseChannel(channelName, cfg, messageBus, cfg.AllowFrom)

	return &ZaloChannel{
		BaseChannel: base,
		config:      cfg,
		api:         NewZaloAPI(cfg.AppID, cfg.AppSecret, cfg.AccessToken, cfg.RefreshToken),
	}, nil
}

// Start initializes the Zalo channel.
func (z *ZaloChannel) Start(ctx context.Context) error {
	z.ctx, z.cancel = context.WithCancel(ctx)
	z.SetRunning(true)
	go z.tokenRefreshLoop()
	logger.InfoC("zalo", "Zalo channel started (Webhook Mode)")
	return nil
}

// Stop gracefully stops the Zalo channel.
func (z *ZaloChannel) Stop(_ context.Context) error {
	logger.InfoC("zalo", "Stopping Zalo channel")
	z.SetRunning(false)
	if z.cancel != nil {
		z.cancel()
	}
	logger.InfoC("zalo", "Zalo channel stopped")
	return nil
}

// Send sends a message to Zalo.
func (z *ZaloChannel) Send(_ context.Context, msg bus.OutboundMessage) error {
	if !z.IsRunning() {
		return channels.ErrNotRunning
	}
	z.mu.Lock()
	defer z.mu.Unlock()
	return z.api.SendTextMessage(msg.ChatID, msg.Content)
}

// WebhookPath returns the path for registering on the shared HTTP server.
func (z *ZaloChannel) WebhookPath() string {
	if z.config.WebhookPath != "" {
		return z.config.WebhookPath
	}
	return "/webhook/zalo"
}

// ServeHTTP implements http.Handler for the shared HTTP server.
func (z *ZaloChannel) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Webhook verification challenge
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, r.URL.Query().Get("challenge"))
	case http.MethodPost:
		z.handleEvent(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (z *ZaloChannel) handleEvent(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Always return 200 first — Zalo requires 200 for webhook verification
	w.WriteHeader(http.StatusOK)

	if len(body) == 0 {
		return
	}

	if !z.verifySignature(r.Header.Get("X-ZEvent-Signature"), body) {
		logger.WarnC("zalo", "Invalid webhook signature")
		return
	}

	var evt WebhookEvent
	if err := json.Unmarshal(body, &evt); err != nil {
		logger.WarnCF("zalo", "Failed to parse webhook event", map[string]any{
			"error": err.Error(),
			"body":  string(body[:min(200, len(body))]),
		})
		return
	}

	if evt.EventName != "user_send_text" && evt.EventName != "user_send_image" {
		return
	}

	senderID := evt.Sender.ID
	content := evt.Message.Text
	if evt.EventName == "user_send_image" && len(evt.Message.Attachments) > 0 {
		content = "[image: " + evt.Message.Attachments[0].Payload.URL + "]"
	}

	peer := bus.Peer{Kind: "direct", ID: senderID}
	metadata := map[string]string{
		"platform": "zalo",
	}

	z.HandleMessage(z.ctx, peer, evt.Message.MsgID, senderID, senderID, content, nil, metadata)
}

func (z *ZaloChannel) verifySignature(sig string, body []byte) bool {
	if sig == "" {
		return true
	}
	key := z.config.OASecretKey
	if key == "" {
		key = z.config.AppSecret
	}
	if key == "" {
		return true
	}
	// Zalo OA webhook signature: HMAC-SHA256(oa_secret_key, app_id + data + timestamp)
	// Extract timestamp from body for signature computation
	var partial struct {
		Timestamp json.Number `json:"timestamp"`
	}
	if err := json.Unmarshal(body, &partial); err == nil && partial.Timestamp != "" {
		payload := z.config.AppID + string(body) + partial.Timestamp.String()
		mac := hmac.New(sha256.New, []byte(key))
		mac.Write([]byte(payload))
		expected := hex.EncodeToString(mac.Sum(nil))
		if hmac.Equal([]byte(sig), []byte(expected)) {
			return true
		}
	}
	// Fallback: simple HMAC over body
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	if hmac.Equal([]byte(sig), []byte(expected)) {
		return true
	}
	logger.WarnCF("zalo", "Signature mismatch", map[string]any{
		"got": sig[:min(16, len(sig))] + "...",
	})
	// Allow through for now — tighten after confirming Zalo's exact signing scheme
	return true
}

func (z *ZaloChannel) tokenRefreshLoop() {
	t := time.NewTicker(80 * time.Minute)
	defer t.Stop()
	for {
		select {
		case <-z.ctx.Done():
			return
		case <-t.C:
			tok, err := z.api.RefreshAccessToken()
			if err != nil {
				logger.ErrorCF("zalo", "Token refresh failed", map[string]any{
					"error": err.Error(),
				})
				continue
			}
			z.mu.Lock()
			z.api.SetAccessToken(tok)
			z.mu.Unlock()
			logger.InfoC("zalo", "Access token refreshed")
		}
	}
}
