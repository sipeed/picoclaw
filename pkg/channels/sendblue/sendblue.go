// Package sendblue connects direct iMessage/SMS conversations to the agent bus.
package sendblue

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
)

var phoneNumber = regexp.MustCompile(`^\+[1-9][0-9]{6,14}$`)

// SendblueChannel uses the gateway's shared webhook server.
// Duplicate suppression covers the last 4096 accepted events in this process.
type SendblueChannel struct {
	*channels.BaseChannel
	config *config.SendblueSettings
	client *http.Client
	mu     sync.Mutex
	seen   map[string]struct{}
	ring   [4096]string
	next   int
}

func NewSendblueChannel(bc *config.Channel, cfg *config.SendblueSettings, b *bus.MessageBus) (*SendblueChannel, error) {
	if strings.TrimSpace(cfg.APIKey.String()) == "" || strings.TrimSpace(cfg.APISecret.String()) == "" ||
		strings.TrimSpace(cfg.WebhookSecret.String()) == "" {
		return nil, fmt.Errorf("sendblue: api_key, api_secret and webhook_secret are required")
	}
	if !phoneNumber.MatchString(cfg.FromNumber) {
		return nil, fmt.Errorf("sendblue: from_number must be an E.164 phone number")
	}
	if len(bc.AllowFrom) == 0 {
		return nil, fmt.Errorf("sendblue: allow_from requires E.164 numbers or explicit '*'")
	}
	for _, number := range bc.AllowFrom {
		if number != "*" && !phoneNumber.MatchString(number) {
			return nil, fmt.Errorf("sendblue: invalid allow_from phone number")
		}
	}
	return &SendblueChannel{
		BaseChannel: channels.NewBaseChannel(config.ChannelSendblue, cfg, b, bc.AllowFrom,
			channels.WithMaxMessageLength(2000)),
		config: cfg,
		client: &http.Client{
			Timeout:       30 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
		seen: make(map[string]struct{}),
	}, nil
}

func (c *SendblueChannel) Start(ctx context.Context) error { c.SetRunning(true); return nil }
func (c *SendblueChannel) Stop(ctx context.Context) error {
	c.SetRunning(false)
	c.client.CloseIdleConnections()
	return nil
}

// WebhookPath includes the instance name so accounts do not share a route.
func (c *SendblueChannel) WebhookPath() string {
	return "/webhook/sendblue/" + url.PathEscape(c.Name())
}

type incoming struct {
	Handle   string `json:"message_handle"`
	Outbound *bool  `json:"is_outbound"`
	From     string `json:"from_number"`
	To       string `json:"to_number"`
	Content  string `json:"content"`
	MediaURL string `json:"media_url"`
	GroupID  string `json:"group_id"`
	Status   string `json:"status"`
}

func (c *SendblueChannel) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !c.IsRunning() {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	if subtle.ConstantTimeCompare(
		[]byte(r.Header.Get("Sb-Signing-Secret")),
		[]byte(c.config.WebhookSecret.String()),
	) != 1 {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 64*1024))
	if err != nil {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		return
	}
	var event incoming
	if json.Unmarshal(body, &event) != nil || event.Handle == "" || len(event.Handle) > 256 || event.Outbound == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if *event.Outbound || event.GroupID != "" || event.Status != "RECEIVED" || event.To != c.config.FromNumber ||
		!phoneNumber.MatchString(event.From) ||
		!c.IsAllowed(event.From) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	content := event.Content
	if event.MediaURL != "" {
		content += "\n[Attachment received; Sendblue channel supports text only.]"
	}
	if strings.TrimSpace(content) == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.seen[event.Handle]; exists {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	err = c.HandleInboundContext(ctx, event.From, content, nil, bus.InboundContext{
		Channel: c.Name(), ChatID: event.From, ChatType: "direct", SenderID: event.From,
		MessageID: event.Handle,
	})
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	delete(c.seen, c.ring[c.next])
	c.ring[c.next] = event.Handle
	c.next = (c.next + 1) % len(c.ring)
	c.seen[event.Handle] = struct{}{}
	w.WriteHeader(http.StatusNoContent)
}

// Send deliberately returns ErrSendFailed for ambiguous POST outcomes: retrying
// without a provider idempotency key could send the same text twice.
func (c *SendblueChannel) Send(ctx context.Context, msg bus.OutboundMessage) ([]string, error) {
	if !c.IsRunning() {
		return nil, channels.ErrNotRunning
	}
	if !phoneNumber.MatchString(msg.ChatID) {
		return nil, fmt.Errorf("sendblue: invalid recipient: %w", channels.ErrSendFailed)
	}
	if strings.TrimSpace(msg.Content) == "" {
		return nil, nil
	}
	body, err := json.Marshal(
		map[string]string{"from_number": c.config.FromNumber, "number": msg.ChatID, "content": msg.Content},
	)
	if err != nil {
		return nil, fmt.Errorf("sendblue: invalid message: %w", channels.ErrSendFailed)
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://api.sendblue.com/api/send-message",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("sendblue: invalid request: %w", channels.ErrSendFailed)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Sb-Api-Key-Id", c.config.APIKey.String())
	req.Header.Set("Sb-Api-Secret-Key", c.config.APISecret.String())
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf(
			"sendblue: send outcome unknown; check message history before retrying: %w",
			channels.ErrSendFailed,
		)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("sendblue: HTTP %d: %w", resp.StatusCode, channels.ErrSendFailed)
	}
	var result struct {
		Handle    string          `json:"message_handle"`
		Status    string          `json:"status"`
		ErrorCode json.RawMessage `json:"error_code"`
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024+1))
	if err != nil || len(data) > 64*1024 || json.Unmarshal(data, &result) != nil || result.Handle == "" {
		return nil, fmt.Errorf("sendblue: invalid send response: %w", channels.ErrSendFailed)
	}
	if len(result.ErrorCode) > 0 && string(result.ErrorCode) != "null" && string(result.ErrorCode) != "0" {
		return nil, fmt.Errorf("sendblue: message rejected: %w", channels.ErrSendFailed)
	}
	switch result.Status {
	case "QUEUED", "SENT", "DELIVERED", "READ":
		return []string{result.Handle}, nil
	default:
		return nil, fmt.Errorf("sendblue: message not accepted: %w", channels.ErrSendFailed)
	}
}
