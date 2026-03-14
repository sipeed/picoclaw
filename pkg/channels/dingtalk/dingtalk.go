// PicoClaw - Ultra-lightweight personal AI agent
// DingTalk channel implementation using Stream Mode

package dingtalk

import (
	"context"
	"fmt"
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

// Health check constants
const (
	healthCheckInterval = 5 * time.Minute  // Check every 5 minutes
	maxSilenceDuration  = 30 * time.Minute // Max time without messages before recovery
	recoveryDelay       = 2 * time.Second  // Delay before reconnecting
	recoveryRetryDelay  = 30 * time.Second // Delay before retrying failed recovery
)

// DingTalkSession stores complete session information for proactive messaging
type DingTalkSession struct {
	SessionWebhook            string    `json:"session_webhook"`
	SessionWebhookExpiredTime int64     `json:"session_webhook_expired_time"` // milliseconds timestamp
	OpenConversationId        string    `json:"open_conversation_id"`         // Group chat ID
	SenderStaffId             string    `json:"sender_staff_id"`              // User ID for direct chat
	ConversationType          string    `json:"conversation_type"`            // "1" = direct, else = group
	LastUpdated               time.Time `json:"last_updated"`
}

// DingTalkChannel implements the Channel interface for DingTalk (钉钉)
// It uses WebSocket for receiving messages via stream mode and API for sending
type DingTalkChannel struct {
	*channels.BaseChannel
	config       config.DingTalkConfig
	clientID     string
	clientSecret string
	streamClient *client.StreamClient
	ctx          context.Context
	cancel       context.CancelFunc
	// Map to store session info for each chat
	sessions sync.Map // chatID -> *DingTalkSession
	// Token management for proactive messaging via robot API
	accessToken string
	tokenExpiry time.Time
	tokenMu     sync.RWMutex
	httpClient  *http.Client
	// Health monitoring
	lastMessageTime time.Time
	mu              sync.RWMutex
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
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
	}, nil
}

// Start initializes the DingTalk channel with Stream Mode
func (c *DingTalkChannel) Start(ctx context.Context) error {
	logger.InfoC("dingtalk", "Starting DingTalk channel (Stream Mode)...")

	c.ctx, c.cancel = context.WithCancel(ctx)
	c.lastMessageTime = time.Now() // Initialize on start

	// Initialize HTTP client for robot API
	c.httpClient = &http.Client{Timeout: 30 * time.Second}

	// If proactive send is enabled, get initial token and start refresh loop
	if c.config.ProactiveSend {
		if err := c.refreshAccessToken(); err != nil {
			logger.WarnCF("dingtalk", "Failed to get initial access token, proactive send may not work", map[string]any{
				"error": err.Error(),
			})
		}
		go c.tokenRefreshLoop()
	}

	// Start the stream client
	if err := c.startStreamClient(); err != nil {
		return err
	}

	// Start health monitoring goroutine
	go c.healthMonitor()

	c.SetRunning(true)
	logger.InfoCF("dingtalk", "DingTalk channel started", map[string]any{
		"proactive_send": c.config.ProactiveSend,
	})
	return nil
}

// startStreamClient creates and starts the stream client
func (c *DingTalkChannel) startStreamClient() error {
	// Create credential config
	cred := client.NewAppCredentialConfig(c.clientID, c.clientSecret)

	// Create the stream client with options
	c.streamClient = client.NewStreamClient(
		client.WithAppCredential(cred),
		client.WithAutoReconnect(true),
	)

	// Register chatbot callback handler
	c.streamClient.RegisterChatBotCallbackRouter(c.onChatBotMessageReceived)

	// Start the stream client
	if err := c.streamClient.Start(c.ctx); err != nil {
		return fmt.Errorf("failed to start stream client: %w", err)
	}

	return nil
}

// healthMonitor periodically checks connection health and triggers recovery if needed
func (c *DingTalkChannel) healthMonitor() {
	ticker := time.NewTicker(healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.checkAndRecover()
		}
	}
}

// checkAndRecover checks if connection is stale and triggers recovery
func (c *DingTalkChannel) checkAndRecover() {
	c.mu.RLock()
	silenceDuration := time.Since(c.lastMessageTime)
	c.mu.RUnlock()

	logger.DebugCF("dingtalk", "Health check: silence duration", map[string]any{"duration": silenceDuration})

	if silenceDuration >= maxSilenceDuration {
		logger.InfoCF(
			"dingtalk",
			"Connection appears stale (no messages for configured duration), triggering recovery",
			map[string]any{"silenceDuration": silenceDuration},
		)
		c.recoverConnection()
	}
}

// recoverConnection attempts to recover the stream connection
func (c *DingTalkChannel) recoverConnection() {
	// Close old client
	if c.streamClient != nil {
		c.streamClient.Close()
		time.Sleep(recoveryDelay)
	}

	// Attempt to reconnect
	for {
		select {
		case <-c.ctx.Done():
			logger.InfoC("dingtalk", "Recovery aborted: context cancelled")
			return
		default:
		}

		err := c.startStreamClient()
		if err == nil {
			logger.InfoCF("dingtalk", "Connection recovered successfully", map[string]any{})
			c.mu.Lock()
			c.lastMessageTime = time.Now()
			c.mu.Unlock()
			return
		}

		logger.WarnCF(
			"dingtalk",
			"Recovery failed, retrying",
			map[string]any{"error": err, "retryDelay": recoveryRetryDelay},
		)
		time.Sleep(recoveryRetryDelay)
	}
}

// updateLastMessageTime updates the last message timestamp
func (c *DingTalkChannel) updateLastMessageTime() {
	c.mu.Lock()
	c.lastMessageTime = time.Now()
	c.mu.Unlock()
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

// Send sends a message to DingTalk
// Priority: 1) session_webhook (if valid), 2) robot API (if proactive_send enabled)
func (c *DingTalkChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	logger.DebugCF("dingtalk", "Sending message", map[string]any{
		"chat_id": msg.ChatID,
		"preview": utils.Truncate(msg.Content, 100),
	})

	// 1. Try session webhook first
	session := c.getSession(msg.ChatID)
	if session != nil && c.isSessionWebhookValid(session) {
		err := c.SendDirectReply(ctx, session.SessionWebhook, msg.Content)
		if err == nil {
			return nil
		}
		// Session webhook failed, log and try fallback
		logger.WarnCF("dingtalk", "Session webhook send failed, trying fallback", map[string]any{
			"error":   err.Error(),
			"chat_id": msg.ChatID,
		})
	}

	// 2. Fallback to robot API if proactive send is enabled
	if !c.config.ProactiveSend {
		return fmt.Errorf("no valid session_webhook for chat %s and proactive_send is disabled", msg.ChatID)
	}

	return c.sendViaRobotAPI(ctx, msg.ChatID, msg.Content)
}

// getSession retrieves session info for a chat
func (c *DingTalkChannel) getSession(chatID string) *DingTalkSession {
	if v, ok := c.sessions.Load(chatID); ok {
		if session, ok := v.(*DingTalkSession); ok {
			return session
		}
	}
	return nil
}

// isSessionWebhookValid checks if the session webhook is still valid
func (c *DingTalkChannel) isSessionWebhookValid(session *DingTalkSession) bool {
	if session.SessionWebhook == "" {
		return false
	}
	// Check expiry if available (SessionWebhookExpiredTime is in milliseconds)
	if session.SessionWebhookExpiredTime > 0 {
		return time.Now().UnixMilli() < session.SessionWebhookExpiredTime
	}
	return true // No expiry info, assume valid
}

// onChatBotMessageReceived implements the IChatBotMessageHandler function signature
// This is called by the Stream SDK when a new message arrives
// IChatBotMessageHandler is: func(c context.Context, data *chatbot.BotCallbackDataModel) ([]byte, error)
func (c *DingTalkChannel) onChatBotMessageReceived(
	ctx context.Context,
	data *chatbot.BotCallbackDataModel,
) ([]byte, error) {
	// Update last message time for health monitoring
	c.updateLastMessageTime()

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

	// Store complete session info for proactive messaging
	session := &DingTalkSession{
		SessionWebhook:            data.SessionWebhook,
		SessionWebhookExpiredTime: data.SessionWebhookExpiredTime,
		OpenConversationId:        data.ConversationId,
		SenderStaffId:             data.SenderStaffId,
		ConversationType:          data.ConversationType,
		LastUpdated:               time.Now(),
	}
	c.sessions.Store(chatID, session)

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
