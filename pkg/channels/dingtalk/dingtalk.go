// PicoClaw - Ultra-lightweight personal AI agent
// DingTalk channel implementation using Stream Mode

package dingtalk

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/open-dingtalk/dingtalk-stream-sdk-go/chatbot"
	"github.com/open-dingtalk/dingtalk-stream-sdk-go/client"

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
	// Map to store session webhooks for each chat
	sessionWebhooks sync.Map // chatID -> sessionWebhook
	// Health monitoring
	lastMessageTime time.Time
	mu              sync.RWMutex
}

// NewDingTalkChannel creates a new DingTalk channel instance
func NewDingTalkChannel(cfg config.DingTalkConfig, messageBus *bus.MessageBus) (*DingTalkChannel, error) {
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, fmt.Errorf("dingtalk client_id and client_secret are required")
	}

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

	// Start the stream client
	if err := c.startStreamClient(); err != nil {
		return err
	}

	// Start health monitoring goroutine
	go c.healthMonitor()

	c.SetRunning(true)
	logger.InfoC("dingtalk", "DingTalk channel started (Stream Mode)")
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

	logger.DebugCF("dingtalk", "Health check: silence duration %v", silenceDuration)

	if silenceDuration >= maxSilenceDuration {
		logger.InfoCF("dingtalk", "Connection appears stale (no messages for %v), triggering recovery", silenceDuration)
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
			logger.InfoCF("dingtalk", "Connection recovered successfully")
			c.mu.Lock()
			c.lastMessageTime = time.Now()
			c.mu.Unlock()
			return
		}

		logger.WarnCF("dingtalk", "Recovery failed: %v, retrying in %v", err, recoveryRetryDelay)
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

// Send sends a message to DingTalk via the chatbot reply API
func (c *DingTalkChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	// Get session webhook from storage
	sessionWebhookRaw, ok := c.sessionWebhooks.Load(msg.ChatID)
	if !ok {
		return fmt.Errorf("no session_webhook found for chat %s, cannot send message", msg.ChatID)
	}

	sessionWebhook, ok := sessionWebhookRaw.(string)
	if !ok {
		return fmt.Errorf("invalid session_webhook type for chat %s", msg.ChatID)
	}

	logger.DebugCF("dingtalk", "Sending message", map[string]any{
		"chat_id": msg.ChatID,
		"preview": utils.Truncate(msg.Content, 100),
	})

	// Use the session webhook to send the reply
	return c.SendDirectReply(ctx, sessionWebhook, msg.Content)
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

	// Store the session webhook for this chat so we can reply later
	c.sessionWebhooks.Store(chatID, data.SessionWebhook)

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
