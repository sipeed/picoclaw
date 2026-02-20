package channels

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

const pushoverTimeout = 10 * time.Second

type PushoverChannel struct {
	*BaseChannel
	config config.PushoverConfig
	client *http.Client
}

func NewPushoverChannel(cfg config.PushoverConfig, bus *bus.MessageBus) (*PushoverChannel, error) {
	base := NewBaseChannel("pushover", cfg, bus, nil)

	return &PushoverChannel{
		BaseChannel: base,
		config:      cfg,
		client: &http.Client{
			Timeout: pushoverTimeout,
		},
	}, nil
}

func (c *PushoverChannel) Name() string {
	return "pushover"
}

func (c *PushoverChannel) Start(ctx context.Context) error {
	logger.InfoC("pushover", "Starting Pushover channel")
	c.setRunning(true)
	return nil
}

func (c *PushoverChannel) Stop(ctx context.Context) error {
	logger.InfoC("pushover", "Stopping Pushover channel")
	c.setRunning(false)
	return nil
}

func (c *PushoverChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return fmt.Errorf("pushover channel not running")
	}

	if c.config.AppToken == "" || c.config.UserKey == "" {
		return fmt.Errorf("pushover app_token and user_key are required")
	}

	data := url.Values{}
	data.Set("token", c.config.AppToken)
	data.Set("user", c.config.UserKey)

	// Truncate message if too long (Pushover limit is 1024 characters)
	// Use runes to properly handle multi-byte UTF-8 characters
	message := msg.Content
	runes := []rune(message)
	if len(runes) > 1024 {
		// Reserve space for "..." so total length does not exceed 1024 characters
		message = string(runes[:1021]) + "..."
	}
	data.Set("message", message)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.pushover.net/1/messages.json", strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		// Read response body for debugging info
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)
		if len(bodyStr) > 200 {
			bodyStr = bodyStr[:200] + "..."
		}
		return fmt.Errorf("pushover API returned status %d: %s", resp.StatusCode, bodyStr)
	}

	logger.DebugCF("pushover", "Notification sent", map[string]any{
		"content_length": len(msg.Content),
	})

	return nil
}
