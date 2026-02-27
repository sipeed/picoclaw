package channels

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/utils"
)

const (
	defaultWebhookHost                    = "127.0.0.1"
	defaultWebhookPort                    = 18794
	defaultWebhookInboundPath             = "/v1/inbound"
	defaultWebhookOutboundPath            = "/v1/outbound"
	defaultWebhookConnectorURL            = "http://127.0.0.1:19400/v1/outbound"
	defaultWebhookConnectorTimeoutSeconds = 10
)

type webhookOutboundPayload struct {
	To      string `json:"to"`
	Content string `json:"content"`
	Peer    string `json:"peer,omitempty"`
}

// WebhookChannel accepts generic local webhook deliveries and forwards outbound messages.
type WebhookChannel struct {
	*BaseChannel
	config       config.WebhookConfig
	server       *http.Server
	httpClient   *http.Client
	ctx          context.Context
	cancel       context.CancelFunc
	inboundPath  string
	outboundPath string
	connectorURL string
}

func NewWebhookChannel(cfg config.WebhookConfig, messageBus *bus.MessageBus) (*WebhookChannel, error) {
	base := NewBaseChannel("webhook", cfg, messageBus, cfg.AllowFrom)
	timeoutSeconds := cfg.ConnectorTimeout
	if timeoutSeconds <= 0 {
		timeoutSeconds = defaultWebhookConnectorTimeoutSeconds
	}

	return &WebhookChannel{
		BaseChannel: base,
		config:      cfg,
		httpClient: &http.Client{
			Timeout: time.Duration(timeoutSeconds) * time.Second,
		},
	}, nil
}

func (c *WebhookChannel) Name() string {
	return "webhook"
}

func (c *WebhookChannel) Start(ctx context.Context) error {
	logger.InfoC("webhook", "Starting webhook channel")

	host := strings.TrimSpace(c.config.WebhookHost)
	if host == "" {
		host = defaultWebhookHost
	}
	port := c.config.WebhookPort
	if port == 0 {
		port = defaultWebhookPort
	}
	c.inboundPath = strings.TrimSpace(c.config.WebhookPath)
	if c.inboundPath == "" {
		c.inboundPath = defaultWebhookInboundPath
	}
	c.outboundPath = strings.TrimSpace(c.config.SendPath)
	if c.outboundPath == "" {
		c.outboundPath = defaultWebhookOutboundPath
	}
	c.connectorURL = strings.TrimSpace(c.config.ConnectorURL)
	if c.connectorURL == "" {
		c.connectorURL = defaultWebhookConnectorURL
	}
	if c.inboundPath == c.outboundPath {
		return fmt.Errorf("webhook inbound path and outbound path must differ: both are %q", c.inboundPath)
	}

	mux := http.NewServeMux()
	mux.HandleFunc(c.inboundPath, c.handleInbound)
	mux.HandleFunc(c.outboundPath, c.handleOutbound)

	addr := fmt.Sprintf("%s:%d", host, port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	c.ctx, c.cancel = context.WithCancel(ctx)
	c.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	c.setRunning(true)

	go func() {
		if err := c.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			logger.ErrorCF("webhook", "Webhook server error", map[string]any{
				"error": err.Error(),
			})
		}
	}()

	logger.InfoCF("webhook", "Webhook server listening", map[string]any{
		"address":       addr,
		"inbound_path":  c.inboundPath,
		"outbound_path": c.outboundPath,
		"connector_url": c.connectorURL,
	})

	return nil
}

func (c *WebhookChannel) Stop(ctx context.Context) error {
	logger.InfoC("webhook", "Stopping webhook channel")

	if c.cancel != nil {
		c.cancel()
	}

	if c.server != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := c.server.Shutdown(shutdownCtx); err != nil {
			logger.ErrorCF("webhook", "Webhook server shutdown error", map[string]any{
				"error": err.Error(),
			})
		}
	}

	c.setRunning(false)
	return nil
}

func (c *WebhookChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	payload := webhookOutboundPayload{
		To:      strings.TrimSpace(msg.ChatID),
		Content: msg.Content,
	}
	if payload.To == "" || strings.TrimSpace(payload.Content) == "" {
		return fmt.Errorf("outbound webhook requires non-empty to/content")
	}
	if err := c.forwardOutbound(ctx, payload); err != nil {
		return err
	}

	logger.DebugCF("webhook", "Forwarded outbound message to connector", map[string]any{
		"chat_id": msg.ChatID,
		"preview": utils.Truncate(msg.Content, 80),
	})
	return nil
}

func (c *WebhookChannel) handleInbound(w http.ResponseWriter, r *http.Request) {
	if !c.validateJSONRequest(w, r) {
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	senderID := webhookSenderID(r, payload)
	if !c.IsAllowed(senderID) {
		http.Error(w, "Sender not allowed", http.StatusForbidden)
		return
	}

	chatID := webhookChatID(r, senderID, payload)
	content, payloadJSON := webhookPayloadContent(payload)

	metadata := map[string]string{
		"platform":          "webhook",
		"sender_id":         senderID,
		"chat_id":           chatID,
		"request_id":        strings.TrimSpace(r.Header.Get("x-request-id")),
		"content_type":      strings.TrimSpace(r.Header.Get("Content-Type")),
		"payload_json":      payloadJSON,
		"delivery_protocol": "webhook",
	}
	if value := strings.TrimSpace(r.Header.Get("x-clawdentity-agent-did")); value != "" {
		metadata["clawdentity_agent_did"] = value
	}
	if value := strings.TrimSpace(r.Header.Get("x-clawdentity-to-agent-did")); value != "" {
		metadata["clawdentity_to_agent_did"] = value
	}

	c.HandleMessage(senderID, chatID, content, nil, metadata)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte(`{"status":"accepted"}`))
}

func (c *WebhookChannel) handleOutbound(w http.ResponseWriter, r *http.Request) {
	if !c.validateJSONRequest(w, r) {
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var payload webhookOutboundPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}
	payload.To = strings.TrimSpace(payload.To)
	payload.Peer = strings.TrimSpace(payload.Peer)
	if payload.To == "" {
		http.Error(w, "Missing required field: to", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(payload.Content) == "" {
		http.Error(w, "Missing required field: content", http.StatusBadRequest)
		return
	}

	if err := c.forwardOutbound(r.Context(), payload); err != nil {
		logger.ErrorCF("webhook", "Failed to forward outbound webhook payload", map[string]any{
			"error": err.Error(),
		})
		http.Error(w, "Failed to forward outbound payload", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte(`{"status":"accepted"}`))
}

func (c *WebhookChannel) validateJSONRequest(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return false
	}

	contentType := strings.TrimSpace(r.Header.Get("Content-Type"))
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil || !strings.EqualFold(mediaType, "application/json") {
		http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
		return false
	}

	if c.config.Token != "" {
		token := webhookAuthToken(r)
		if token != c.config.Token {
			http.Error(w, "Invalid token", http.StatusForbidden)
			return false
		}
	}

	return true
}

func (c *WebhookChannel) forwardOutbound(ctx context.Context, payload webhookOutboundPayload) error {
	if c.connectorURL == "" {
		c.connectorURL = strings.TrimSpace(c.config.ConnectorURL)
		if c.connectorURL == "" {
			c.connectorURL = defaultWebhookConnectorURL
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal outbound payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.connectorURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build outbound connector request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send outbound connector request: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("connector returned status %d", resp.StatusCode)
	}

	return nil
}

func webhookAuthToken(r *http.Request) string {
	if token := strings.TrimSpace(r.Header.Get("x-webhook-token")); token != "" {
		return token
	}

	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if auth == "" {
		return ""
	}
	if len(auth) >= 7 && strings.EqualFold(auth[:7], "Bearer ") {
		return strings.TrimSpace(auth[7:])
	}

	return auth
}

func webhookSenderID(r *http.Request, payload any) string {
	for _, key := range []string{"x-webhook-sender-id", "x-clawdentity-agent-did"} {
		value := strings.TrimSpace(r.Header.Get(key))
		if value != "" {
			return value
		}
	}

	if value := webhookPayloadStringField(payload, "userId", "sender_id"); value != "" {
		return value
	}

	if host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr)); err == nil && host != "" {
		return host
	}

	if addr := strings.TrimSpace(r.RemoteAddr); addr != "" {
		return addr
	}

	return "webhook"
}

func webhookChatID(r *http.Request, senderID string, payload any) string {
	for _, key := range []string{"x-webhook-chat-id", "x-clawdentity-to-agent-did"} {
		value := strings.TrimSpace(r.Header.Get(key))
		if value != "" {
			return value
		}
	}

	if value := webhookPayloadStringField(payload, "chatId", "chat_id"); value != "" {
		return value
	}

	return senderID
}

func webhookPayloadStringField(payload any, keys ...string) string {
	asMap, ok := payload.(map[string]any)
	if !ok {
		return ""
	}

	for _, key := range keys {
		value, exists := asMap[key]
		if !exists {
			continue
		}
		text, ok := value.(string)
		if !ok {
			continue
		}
		trimmed := strings.TrimSpace(text)
		if trimmed != "" {
			return trimmed
		}
	}

	return ""
}

func webhookPayloadContent(payload any) (string, string) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", "{}"
	}
	payloadJSON := string(payloadBytes)

	asMap, ok := payload.(map[string]any)
	if !ok {
		return payloadJSON, payloadJSON
	}

	for _, key := range []string{"content", "text", "message"} {
		value, exists := asMap[key]
		if !exists {
			continue
		}
		text, ok := value.(string)
		if !ok {
			continue
		}
		trimmed := strings.TrimSpace(text)
		if trimmed != "" {
			return text, payloadJSON
		}
	}

	return payloadJSON, payloadJSON
}
