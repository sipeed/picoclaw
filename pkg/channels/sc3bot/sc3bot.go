package sc3bot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/identity"
	"github.com/sipeed/picoclaw/pkg/logger"
)

const (
	apiBaseURL         = "https://bot-go.apijia.cn"
	requestTimeout     = 30 * time.Second
	pollInterval       = 5 * time.Second
	maxWebhookBodySize = 1 << 20 // 1 MiB

	webhookSecretHeader = "X-Sc3Bot-Webhook-Secret"
)

// SC3BotChannel implements the Channel interface for Server酱³ Bot API.
// Supports both polling mode (getUpdates) and webhook mode for receiving messages.
type SC3BotChannel struct {
	*channels.BaseChannel
	bc     *config.Channel
	config *config.SC3BotSettings
	client *http.Client
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Polling mode
	pollOffset int64
	pollMu     sync.Mutex
}

// SC3BotUpdate represents an update from the Bot API
type SC3BotUpdate struct {
	UpdateID int64          `json:"update_id"`
	Message  *SC3BotMessage `json:"message,omitempty"`
}

// SC3BotChat represents the chat info in a message
type SC3BotChat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

// SC3BotMessage represents a message in the Bot API
type SC3BotMessage struct {
	MessageID int64       `json:"message_id"`
	ChatID    int64       `json:"chat_id"`
	Chat      *SC3BotChat `json:"chat,omitempty"`
	Text      string      `json:"text,omitempty"`
}

// GetChatID returns the chat ID from either chat_id field or chat.id
func (m *SC3BotMessage) GetChatID() int64 {
	if m.ChatID != 0 {
		return m.ChatID
	}
	if m.Chat != nil {
		return m.Chat.ID
	}
	return 0
}

// SC3BotResponse represents the API response
type SC3BotResponse struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result,omitempty"`
}

// SC3BotUser represents the bot user info from getMe
type SC3BotUser struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	Username  string `json:"username,omitempty"`
}

// NewSC3BotChannel creates a new Server酱³ Bot channel instance.
func NewSC3BotChannel(
	bc *config.Channel,
	cfg *config.SC3BotSettings,
	messageBus *bus.MessageBus,
) (*SC3BotChannel, error) {
	if cfg.Token.String() == "" {
		return nil, fmt.Errorf("sc3bot: token is required")
	}

	client := &http.Client{
		Timeout: requestTimeout,
	}

	if cfg.Proxy != "" {
		proxyURL, err := url.Parse(cfg.Proxy)
		if err != nil {
			return nil, fmt.Errorf("sc3bot: invalid proxy URL: %w", err)
		}
		client.Transport = &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}
	}

	base := channels.NewBaseChannel(
		bc.Name(),
		cfg,
		messageBus,
		bc.AllowFrom,
		channels.WithMaxMessageLength(4000),
		channels.WithGroupTrigger(bc.GroupTrigger),
		channels.WithReasoningChannelID(bc.ReasoningChannelID),
	)

	return &SC3BotChannel{
		BaseChannel: base,
		bc:          bc,
		config:      cfg,
		client:      client,
	}, nil
}

// Start initializes the SC3Bot channel.
func (c *SC3BotChannel) Start(ctx context.Context) error {
	logger.InfoC("sc3bot", "Starting Server酱³ Bot channel...")

	// Verify bot token by calling getMe
	botInfo, err := c.getMe(ctx)
	if err != nil {
		return fmt.Errorf("sc3bot: failed to get bot info: %w", err)
	}

	logger.InfoCF("sc3bot", "Bot info retrieved", map[string]any{
		"bot_id":       botInfo.ID,
		"bot_name":     botInfo.FirstName,
		"bot_username": botInfo.Username,
	})

	c.ctx, c.cancel = context.WithCancel(ctx)
	c.SetRunning(true)

	// Start polling
	c.wg.Add(1)
	go c.pollLoop()
	logger.InfoC("sc3bot", "Started polling mode")

	logger.InfoC("sc3bot", "Server酱³ Bot channel started")
	return nil
}

// Stop gracefully stops the SC3Bot channel.
func (c *SC3BotChannel) Stop(ctx context.Context) error {
	logger.InfoC("sc3bot", "Stopping Server酱³ Bot channel...")

	if c.cancel != nil {
		c.cancel()
	}

	// Wait for polling goroutine to finish
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Graceful shutdown completed
	case <-ctx.Done():
		logger.WarnC("sc3bot", "Shutdown timeout, forcing stop")
	}

	c.SetRunning(false)
	logger.InfoC("sc3bot", "Server酱³ Bot channel stopped")
	return nil
}

// Send delivers a message to the specified chat.
func (c *SC3BotChannel) Send(ctx context.Context, msg bus.OutboundMessage) ([]string, error) {
	if !c.IsRunning() {
		return nil, channels.ErrNotRunning
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	chatID := msg.ChatID
	if chatID == "" {
		return nil, fmt.Errorf("sc3bot: chat_id is required")
	}

	content := msg.Content
	if content == "" {
		content = "(empty message)"
	}

	// Parse chat_id as integer
	chatIDInt, err := strconv.ParseInt(chatID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("sc3bot: invalid chat_id format: %w", err)
	}

	resp, err := c.sendMessage(ctx, chatIDInt, content)
	if err != nil {
		logger.ErrorCF("sc3bot", "Send failed", map[string]any{
			"chat_id": chatID,
			"error":   err.Error(),
		})
		return nil, err
	}

	logger.InfoCF("sc3bot", "Message sent", map[string]any{
		"chat_id": chatID,
	})

	return []string{fmt.Sprintf("%v", resp)}, nil
}

// StartTyping sends a typing action to the chat.
func (c *SC3BotChannel) StartTyping(ctx context.Context, chatID string) (stop func(), err error) {
	if !c.IsRunning() {
		return nil, channels.ErrNotRunning
	}

	chatIDInt, err := strconv.ParseInt(chatID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("sc3bot: invalid chat_id format: %w", err)
	}

	// Send typing action
	if err := c.sendChatAction(ctx, chatIDInt, "typing"); err != nil {
		return nil, err
	}

	// Return a no-op stop function since the typing indicator is temporary
	return func() {}, nil
}

// WebhookPath returns the path for webhook registration.
func (c *SC3BotChannel) WebhookPath() string {
	return "/webhook/sc3bot"
}

// ServeHTTP implements http.Handler for webhook mode.
func (c *SC3BotChannel) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.webhookHandler(w, r)
}

// webhookHandler handles incoming webhook requests.
func (c *SC3BotChannel) webhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verify webhook secret if configured
	if c.config.Secret != "" {
		secret := r.Header.Get(webhookSecretHeader) //nolint
		if secret != c.config.Secret {
			logger.WarnC("sc3bot", "Webhook secret mismatch")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	// Limit body size
	r.Body = http.MaxBytesReader(w, r.Body, maxWebhookBodySize)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.ErrorCF("sc3bot", "Failed to read webhook body", map[string]any{
			"error": err.Error(),
		})
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	var update SC3BotUpdate
	if err := json.Unmarshal(body, &update); err != nil {
		logger.ErrorCF("sc3bot", "Failed to parse webhook payload", map[string]any{
			"error": err.Error(),
		})
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Process the update using request context
	c.processUpdateWithContext(r.Context(), &update)

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

// pollLoop continuously polls for updates.
func (c *SC3BotChannel) pollLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	// Poll immediately on start
	c.pollOnce()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.pollOnce()
		}
	}
}

// pollOnce performs a single poll for updates.
func (c *SC3BotChannel) pollOnce() {
	updates, err := c.getUpdates(c.ctx)
	if err != nil {
		if c.ctx.Err() != nil {
			return // Context canceled
		}
		logger.ErrorCF("sc3bot", "Failed to get updates", map[string]any{
			"error": err.Error(),
		})
		return
	}

	for _, update := range updates {
		c.processUpdate(&update)

		// Update offset
		c.pollMu.Lock()
		if update.UpdateID >= c.pollOffset {
			c.pollOffset = update.UpdateID + 1
		}
		c.pollMu.Unlock()
	}
}

// processUpdate processes a single update (used in polling mode).
func (c *SC3BotChannel) processUpdate(update *SC3BotUpdate) {
	c.processUpdateWithContext(c.ctx, update)
}

// processUpdateWithContext processes a single update with the given context.
func (c *SC3BotChannel) processUpdateWithContext(ctx context.Context, update *SC3BotUpdate) {
	if update.Message == nil {
		return
	}

	msg := update.Message
	if msg.Text == "" {
		return
	}

	chatIDInt := msg.GetChatID()
	if chatIDInt == 0 {
		logger.ErrorC("sc3bot", "Received message with invalid chat_id=0, skipping")
		return
	}

	logger.InfoCF("sc3bot", "Received message", map[string]any{
		"update_id":  update.UpdateID,
		"message_id": msg.MessageID,
		"chat_id":    chatIDInt,
		"text":       msg.Text,
	})

	chatID := strconv.FormatInt(chatIDInt, 10)

	// Create sender info
	sender := bus.SenderInfo{
		Platform:    c.Name(),
		PlatformID:  chatID,
		CanonicalID: identity.BuildCanonicalID(c.Name(), chatID),
	}

	// Create inbound context
	inboundCtx := bus.InboundContext{
		Channel:   c.Name(),
		ChatID:    chatID,
		SenderID:  chatID,
		MessageID: strconv.FormatInt(msg.MessageID, 10),
	}

	// Handle the message using base channel
	logger.DebugCF("sc3bot", "Calling HandleMessageWithContext", map[string]any{
		"chat_id": chatID,
		"text":    msg.Text,
	})
	c.HandleMessageWithContext(ctx, chatID, msg.Text, nil, inboundCtx, sender)
	logger.DebugC("sc3bot", "HandleMessageWithContext completed")
}

// API methods

func (c *SC3BotChannel) getMe(ctx context.Context) (*SC3BotUser, error) {
	url := fmt.Sprintf("%s/bot%s/getMe", apiBaseURL, c.config.Token.String())

	resp, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var user SC3BotUser
	if err := json.Unmarshal(resp.Result, &user); err != nil {
		return nil, fmt.Errorf("failed to parse user info: %w", err)
	}

	return &user, nil
}

func (c *SC3BotChannel) sendMessage(ctx context.Context, chatID int64, text string) (bool, error) {
	url := fmt.Sprintf("%s/bot%s/sendMessage", apiBaseURL, c.config.Token.String())

	payload := map[string]any{
		"chat_id": chatID,
		"text":    text,
	}

	resp, err := c.doRequest(ctx, http.MethodPost, url, payload)
	if err != nil {
		return false, err
	}

	var result bool
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		// Some APIs return the sent message object instead of boolean
		return resp.OK, nil //nolint:nilerr
	}

	return result, nil
}

func (c *SC3BotChannel) sendChatAction(ctx context.Context, chatID int64, action string) error {
	url := fmt.Sprintf("%s/bot%s/sendChatAction", apiBaseURL, c.config.Token.String())

	payload := map[string]any{
		"chat_id": chatID,
		"action":  action,
	}

	_, err := c.doRequest(ctx, http.MethodPost, url, payload)
	return err
}

func (c *SC3BotChannel) getUpdates(ctx context.Context) ([]SC3BotUpdate, error) {
	c.pollMu.Lock()
	offset := c.pollOffset
	c.pollMu.Unlock()

	// Use timeout=0 for non-blocking poll
	url := fmt.Sprintf("%s/bot%s/getUpdates?timeout=0&offset=%d", apiBaseURL, c.config.Token.String(), offset)

	logger.DebugCF("sc3bot", "Calling getUpdates", map[string]any{
		"offset": offset,
	})

	resp, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		logger.ErrorCF("sc3bot", "getUpdates failed", map[string]any{
			"error": err.Error(),
		})
		return nil, err
	}

	var updates []SC3BotUpdate
	if err := json.Unmarshal(resp.Result, &updates); err != nil {
		return nil, fmt.Errorf("failed to parse updates: %w", err)
	}

	return updates, nil
}

func (c *SC3BotChannel) doRequest(ctx context.Context, method, url string, payload any) (*SC3BotResponse, error) {
	var body []byte
	var err error

	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var apiResp SC3BotResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !apiResp.OK {
		return nil, fmt.Errorf("API returned error: %s", string(respBody))
	}

	return &apiResp, nil
}

// Compile-time interface checks
var (
	_ channels.Channel        = (*SC3BotChannel)(nil)
	_ channels.TypingCapable  = (*SC3BotChannel)(nil)
	_ channels.WebhookHandler = (*SC3BotChannel)(nil)
)
