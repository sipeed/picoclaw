// PicoClaw - Ultra-lightweight personal AI agent
// DingTalk channel implementation using Stream Mode with proactive messaging support

package dingtalk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/open-dingtalk/dingtalk-stream-sdk-go/chatbot"
	"github.com/open-dingtalk/dingtalk-stream-sdk-go/client"
	dinglog "github.com/open-dingtalk/dingtalk-stream-sdk-go/logger"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/identity"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/utils"
)

const dingtalkAPIBase = "https://api.dingtalk.com"

// chatInfo stores information needed for proactive messaging
type chatInfo struct {
	sessionWebhook     string
	sessionWebhookExp  time.Time // From sessionWebhookExpiredTime
	senderStaffId      string    // For single chat proactive send
	openConversationId string    // For group chat proactive send (ConversationId)
	conversationType   string    // "1" = single, "2" = group
}

// BatchSendResponse represents the batch send API response
type BatchSendResponse struct {
	Code               string            `json:"code"`
	Message            string            `json:"message"`
	ProcessQueryKeys   map[string]string `json:"processQueryKeys"`
	InvalidStaffIdList []string          `json:"invalidStaffIdList"`
}

// DingTalkChannel implements the Channel interface for DingTalk (钉钉)
// It uses WebSocket for receiving messages via stream mode and API for sending
type DingTalkChannel struct {
	*channels.BaseChannel
	config       config.DingTalkConfig
	clientID     string // AppKey (also used as robotCode for proactive messaging)
	clientSecret string // AppSecret
	streamClient *client.StreamClient
	ctx          context.Context
	cancel       context.CancelFunc
	// Map to store chat info for each chat (includes session webhook and proactive send info)
	chatInfos sync.Map // chatID -> *chatInfo

	// HTTP client for proactive API calls
	httpClient  *http.Client
	accessToken string
	tokenExpiry time.Time
	tokenMu     sync.RWMutex
}

// NewDingTalkChannel creates a new DingTalk channel instance
func NewDingTalkChannel(cfg config.DingTalkConfig, messageBus *bus.MessageBus) (*DingTalkChannel, error) {
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, fmt.Errorf("dingtalk client_id and client_secret are required")
	}

	// Set the logger for the Stream SDK
	dinglog.SetLogger(logger.NewLogger("dingtalk"))

	base := channels.NewBaseChannel("dingtalk", cfg, messageBus, cfg.AllowFrom,
		channels.WithMaxMessageLength(20000),
		channels.WithGroupTrigger(cfg.GroupTrigger),
		channels.WithReasoningChannelID(cfg.ReasoningChannelID),
	)

	return &DingTalkChannel{
		BaseChannel:  base,
		config:       cfg,
		clientID:     cfg.ClientID, // Also used as robotCode for proactive messaging
		clientSecret: cfg.ClientSecret,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// Start initializes the DingTalk channel with Stream Mode
func (c *DingTalkChannel) Start(ctx context.Context) error {
	logger.InfoC("dingtalk", "Starting DingTalk channel (Stream Mode)...")

	c.ctx, c.cancel = context.WithCancel(ctx)

	// Create credential config
	cred := client.NewAppCredentialConfig(c.clientID, c.clientSecret)

	// Create the stream client with options
	c.streamClient = client.NewStreamClient(
		client.WithAppCredential(cred),
		client.WithAutoReconnect(true),
	)

	// Register chatbot callback handler (IChatBotMessageHandler is a function type)
	c.streamClient.RegisterChatBotCallbackRouter(c.onChatBotMessageReceived)

	// Start the stream client
	if err := c.streamClient.Start(c.ctx); err != nil {
		return fmt.Errorf("failed to start stream client: %w", err)
	}

	// Get initial access token for proactive messaging
	if err := c.refreshAccessToken(); err != nil {
		logger.WarnCF("dingtalk", "Failed to get initial access token", map[string]any{
			"error": err.Error(),
		})
	}

	// Start token refresh goroutine
	go c.tokenRefreshLoop()

	c.SetRunning(true)
	logger.InfoC("dingtalk", "DingTalk channel started (Stream Mode)")
	return nil
}

// Stop gracefully stops the DingTalk channel
func (c *DingTalkChannel) Stop(ctx context.Context) error {
	logger.InfoC("dingtalk", "Stopping DingTalk channel...")

	if c.cancel != nil {
		c.cancel()
	}

	if c.streamClient != nil {
		c.streamClient.Close()
	}

	c.SetRunning(false)
	logger.InfoC("dingtalk", "DingTalk channel stopped")
	return nil
}

// Send sends a message to DingTalk with fallback from session_webhook to proactive API
func (c *DingTalkChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	logger.DebugCF("dingtalk", "Sending message", map[string]any{
		"chat_id": msg.ChatID,
		"preview": utils.Truncate(msg.Content, 100),
	})

	// 1. Try session_webhook first (if available and not expired)
	if info, ok := c.getChatInfo(msg.ChatID); ok {
		if info.sessionWebhook != "" && time.Now().Before(info.sessionWebhookExp) {
			err := c.SendDirectReply(ctx, info.sessionWebhook, msg.Content)
			if err == nil {
				return nil
			}
			// Log error and fall through to proactive API
			logger.DebugCF("dingtalk", "session_webhook failed, trying proactive API", map[string]any{
				"error": err.Error(),
			})
		}
	}

	// 2. Fall back to proactive API
	return c.sendProactive(ctx, msg.ChatID, msg.Content)
}

// getChatInfo safely retrieves chat info
func (c *DingTalkChannel) getChatInfo(chatID string) (*chatInfo, bool) {
	raw, ok := c.chatInfos.Load(chatID)
	if !ok {
		return nil, false
	}
	info, ok := raw.(*chatInfo)
	return info, ok
}

// sendProactive sends a message using the proactive API
// If chatInfo exists (from prior user message), use stored info for conversation type.
// If not, assume single chat and use chatID directly as staffId - this allows
// proactive messaging to users whose staffId is known (e.g., from state/config).
func (c *DingTalkChannel) sendProactive(ctx context.Context, chatID, content string) error {
	accessToken := c.getAccessToken()
	if accessToken == "" {
		return fmt.Errorf("no valid access token available: %w", channels.ErrTemporary)
	}

	info, ok := c.getChatInfo(chatID)
	if ok {
		// Use stored info (preferred - we know conversation type)
		if info.conversationType == "1" {
			// Single chat - use batch send API
			return c.sendProactiveSingleChat(ctx, accessToken, info.senderStaffId, content)
		}
		// Group chat - use group messages API
		return c.sendProactiveGroupChat(ctx, accessToken, info.openConversationId, content)
	}

	// No stored chatInfo - assume single chat and use chatID directly as staffId
	// This enables proactive messaging without requiring prior user interaction
	logger.DebugCF("dingtalk", "No stored chatInfo, assuming single chat", map[string]any{
		"chat_id": chatID,
	})
	return c.sendProactiveSingleChat(ctx, accessToken, chatID, content)
}

// sendProactiveSingleChat sends message via oToMessages/batchSend API for single chats
// robotCode = clientID (AppKey)
func (c *DingTalkChannel) sendProactiveSingleChat(ctx context.Context, accessToken, staffId, content string) error {
	msgParam := buildMarkdownMsgParam("PicoClaw", content)
	reqBody := map[string]any{
		"robotCode": c.clientID, // robotCode = AppKey
		"userIds":   []string{staffId},
		"msgKey":    "sampleMarkdown",
		"msgParam":  msgParam,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	apiURL := fmt.Sprintf("%s/v1.0/robot/oToMessages/batchSend", dingtalkAPIBase)
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Acs-Dingtalk-Access-Token", accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", channels.ErrTemporary)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var result BatchSendResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if resp.StatusCode != http.StatusOK || (result.Code != "" && result.Code != "0" && result.Code != "success") {
		return fmt.Errorf("dingtalk API error: %s (code: %s, status: %d)", result.Message, result.Code, resp.StatusCode)
	}

	return nil
}

// sendProactiveGroupChat sends message via groupMessages API for group chats
// robotCode = clientID (AppKey)
func (c *DingTalkChannel) sendProactiveGroupChat(
	ctx context.Context,
	accessToken, openConversationId, content string,
) error {
	msgParam := buildMarkdownMsgParam("PicoClaw", content)
	reqBody := map[string]any{
		"openConversationId": openConversationId,
		"robotCode":          c.clientID, // robotCode = AppKey
		"msgKey":             "sampleMarkdown",
		"msgParam":           msgParam,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	apiURL := fmt.Sprintf("%s/v1.0/robot/groupMessages/send", dingtalkAPIBase)
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Acs-Dingtalk-Access-Token", accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", channels.ErrTemporary)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var result BatchSendResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if resp.StatusCode != http.StatusOK || (result.Code != "" && result.Code != "0" && result.Code != "success") {
		return fmt.Errorf("dingtalk API error: %s (code: %s, status: %d)", result.Message, result.Code, resp.StatusCode)
	}

	return nil
}

// onChatBotMessageReceived implements the IChatBotMessageHandler function signature
// This is called by the Stream SDK when a new message arrives
// IChatBotMessageHandler is: func(c context.Context, data *chatbot.BotCallbackDataModel) ([]byte, error)
func (c *DingTalkChannel) onChatBotMessageReceived(
	ctx context.Context,
	data *chatbot.BotCallbackDataModel,
) ([]byte, error) {
	// Extract message content from Text field
	content := data.Text.Content
	if content == "" {
		// Try to extract from Content interface{} if Text is empty
		if contentMap, ok := data.Content.(map[string]any); ok {
			if textContent, ok := contentMap["content"].(string); ok {
				content = textContent
			}
		}
	}

	if content == "" {
		return nil, nil // Ignore empty messages
	}

	senderID := data.SenderStaffId
	senderNick := data.SenderNick
	chatID := senderID
	if data.ConversationType != "1" {
		// For group chats
		chatID = data.ConversationId
	}

	// Parse expiry time from sessionWebhookExpiredTime
	var webhookExpiry time.Time
	if data.SessionWebhookExpiredTime > 0 {
		webhookExpiry = time.Unix(data.SessionWebhookExpiredTime/1000, 0)
	}

	// Store extended chat info for proactive messaging
	info := &chatInfo{
		sessionWebhook:     data.SessionWebhook,
		sessionWebhookExp:  webhookExpiry,
		senderStaffId:      data.SenderStaffId,
		openConversationId: data.ConversationId,
		conversationType:   data.ConversationType,
	}
	c.chatInfos.Store(chatID, info)

	metadata := map[string]string{
		"sender_name":       senderNick,
		"conversation_id":   data.ConversationId,
		"conversation_type": data.ConversationType,
		"platform":          "dingtalk",
		"session_webhook":   data.SessionWebhook,
	}

	var peer bus.Peer
	if data.ConversationType == "1" {
		peer = bus.Peer{Kind: "direct", ID: senderID}
	} else {
		peer = bus.Peer{Kind: "group", ID: data.ConversationId}
		// In group chats, apply unified group trigger filtering
		respond, cleaned := c.ShouldRespondInGroup(false, content)
		if !respond {
			return nil, nil
		}
		content = cleaned
	}

	logger.DebugCF("dingtalk", "Received message", map[string]any{
		"sender_nick": senderNick,
		"sender_id":   senderID,
		"preview":     utils.Truncate(content, 50),
	})

	// Build sender info
	sender := bus.SenderInfo{
		Platform:    "dingtalk",
		PlatformID:  senderID,
		CanonicalID: identity.BuildCanonicalID("dingtalk", senderID),
		DisplayName: senderNick,
	}

	if !c.IsAllowedSender(sender) {
		return nil, nil
	}

	// Handle the message through the base channel
	c.HandleMessage(ctx, peer, "", senderID, chatID, content, nil, metadata, sender)

	// Return nil to indicate we've handled the message asynchronously
	// The response will be sent through the message bus
	return nil, nil
}

// SendDirectReply sends a direct reply using the session webhook
func (c *DingTalkChannel) SendDirectReply(ctx context.Context, sessionWebhook, content string) error {
	replier := chatbot.NewChatbotReplier()

	// Convert string content to []byte for the API
	contentBytes := []byte(content)
	titleBytes := []byte("PicoClaw")

	// Send markdown formatted reply
	err := replier.SimpleReplyMarkdown(
		ctx,
		sessionWebhook,
		titleBytes,
		contentBytes,
	)
	if err != nil {
		return fmt.Errorf("dingtalk send: %w", channels.ErrTemporary)
	}

	return nil
}

// refreshAccessToken fetches a new access token from DingTalk API
// API: POST /v1.0/oauth2/accessToken
// Body: {"appKey": "...", "appSecret": "..."}
// Response: {"accessToken": "...", "expireIn": 7200}
func (c *DingTalkChannel) refreshAccessToken() error {
	reqBody := map[string]string{
		"appKey":    c.clientID,
		"appSecret": c.clientSecret,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	apiURL := fmt.Sprintf("%s/v1.0/oauth2/accessToken", dingtalkAPIBase)

	req, err := http.NewRequestWithContext(c.ctx, "POST", apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to request access token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("access token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"accessToken"`
		ExpireIn    int    `json:"expireIn"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	c.tokenMu.Lock()
	c.accessToken = tokenResp.AccessToken
	// Refresh 5 minutes before expiry
	c.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpireIn-300) * time.Second)
	c.tokenMu.Unlock()

	logger.DebugCF("dingtalk", "Access token refreshed successfully", map[string]any{
		"expire_in": tokenResp.ExpireIn,
	})
	return nil
}

// tokenRefreshLoop periodically refreshes the access token
func (c *DingTalkChannel) tokenRefreshLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			if err := c.refreshAccessToken(); err != nil {
				logger.ErrorCF("dingtalk", "Failed to refresh access token", map[string]any{
					"error": err.Error(),
				})
			}
		}
	}
}

// getAccessToken returns the current valid access token
func (c *DingTalkChannel) getAccessToken() string {
	c.tokenMu.RLock()
	defer c.tokenMu.RUnlock()

	if time.Now().After(c.tokenExpiry) {
		return ""
	}

	return c.accessToken
}

// buildMarkdownMsgParam builds the msgParam for markdown messages
func buildMarkdownMsgParam(title, content string) string {
	param := map[string]string{
		"title": title,
		"text":  content,
	}
	data, _ := json.Marshal(param)
	return string(data)
}
