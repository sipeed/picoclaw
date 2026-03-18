// PicoClaw - Ultra-lightweight personal AI agent
// DingTalk channel implementation using Stream Mode

package dingtalk

import (
	"context"
	"fmt"
	"sync"

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

// DingTalkChannel implements the Channel interface for DingTalk (钉钉)
// It uses WebSocket for receiving messages via stream mode and API for sending
type DingTalkChannel struct {
	*channels.BaseChannel
	config       config.DingTalkConfig
	clientID     string
	clientSecret string
	streamClient *client.StreamClient
	client       *Client
	ctx          context.Context
	cancel       context.CancelFunc
	// Map to store session webhooks for each chat
	sessionWebhooks sync.Map // chatID -> sessionWebhook
	// chatID -> cardInstanceID
	cardInstanceIDs sync.Map
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
	// dingtalk client
	dingTalkClient := NewClient(cfg.ClientID, cfg.ClientSecret,
		WithRobotCode(cfg.RobotCode),
		WithCardTemplateID(cfg.CardTemplateID),
		WithCardTemplateContentKey(cfg.CardTemplateContentKey))

	return &DingTalkChannel{
		BaseChannel:  base,
		config:       cfg,
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
		client:       dingTalkClient,
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

// Send sends a message to DingTalk via the chatbot reply API
func (c *DingTalkChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}
	// Check if we have a card instance ID for this chat (indicating we can send a card reply)
	cardInstanceIDRaw, ok := c.cardInstanceIDs.LoadAndDelete(msg.ChatID)
	if !ok {
		return c.SendDirectReply(ctx, msg)
	}
	cardInstanceID, ok := cardInstanceIDRaw.(string)
	if !ok {
		return c.SendDirectReply(ctx, msg)
	}
	return c.SendCardReply(ctx, cardInstanceID, msg.Content)
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

	// Try to create and deliver card (optional feature)
	// Only attempt this if a card template is configured; otherwise, fall back silently
	if c.config.CardTemplateID != "" {
		// If it fails, log the error but continue with normal message handling
		if cardID, err := c.tryCardCreateAndDeliver(ctx, data); err != nil {
			logger.WarnC("dingtalk", "Failed to create or deliver card, falling back to direct reply")
			// Store the session webhook for this chat so we can reply later
			c.sessionWebhooks.Store(chatID, data.SessionWebhook)
		} else {
			chatID = data.MsgId
			c.cardInstanceIDs.Store(chatID, cardID)
		}
	} else {
		// Card feature not configured; just store the session webhook for direct replies
		c.sessionWebhooks.Store(chatID, data.SessionWebhook)
	}

	// Handle the message through the base channel
	c.HandleMessage(ctx, peer, "", senderID, chatID, content, nil, metadata, sender)

	// Return nil to indicate we've handled the message asynchronously
	// The response will be sent through the message bus
	return nil, nil
}

// SendDirectReply sends a direct reply using the session webhook
func (c *DingTalkChannel) SendDirectReply(ctx context.Context, msg bus.OutboundMessage) error {
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
	replier := chatbot.NewChatbotReplier()

	// Convert string content to []byte for the API
	contentBytes := []byte(msg.Content)
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

func (c *DingTalkChannel) SendCardReply(ctx context.Context, cardInstanceID, content string) error {
	return c.client.CardStreaming(ctx, cardInstanceID, content)
}

func (c *DingTalkChannel) tryCardCreateAndDeliver(
	ctx context.Context,
	data *chatbot.BotCallbackDataModel,
) (string, error) {
	if c.config.CardTemplateID == "" {
		return "", fmt.Errorf("card_template_id is not configured, cannot create card")
	}
	return c.client.CardCreateAndDeliver(ctx, data)
}
