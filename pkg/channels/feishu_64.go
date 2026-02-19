//go:build amd64 || arm64 || riscv64 || mips64 || ppc64

package channels

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkdispatcher "github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/utils"
)

type FeishuChannel struct {
	*BaseChannel
	config   config.FeishuConfig
	client   *lark.Client
	wsClient *larkws.Client

	mu     sync.Mutex
	cancel context.CancelFunc
}

func NewFeishuChannel(cfg config.FeishuConfig, messagesCfg config.MessagesConfig, bus *bus.MessageBus) (*FeishuChannel, error) {
	base := NewBaseChannel("feishu", cfg, messagesCfg, bus, cfg.AllowFrom)

	return &FeishuChannel{
		BaseChannel: base,
		config:      cfg,
		client:      lark.NewClient(cfg.AppID, cfg.AppSecret),
	}, nil
}

func (c *FeishuChannel) Start(ctx context.Context) error {
	if c.config.AppID == "" || c.config.AppSecret == "" {
		return fmt.Errorf("feishu app_id or app_secret is empty")
	}

	dispatcher := larkdispatcher.NewEventDispatcher(c.config.VerificationToken, c.config.EncryptKey).
		OnP2MessageReceiveV1(c.handleMessageReceive)

	runCtx, cancel := context.WithCancel(ctx)

	c.mu.Lock()
	c.cancel = cancel
	c.wsClient = larkws.NewClient(
		c.config.AppID,
		c.config.AppSecret,
		larkws.WithEventHandler(dispatcher),
	)
	wsClient := c.wsClient
	c.mu.Unlock()

	c.setRunning(true)
	logger.InfoC("feishu", "Feishu channel started (websocket mode)")

	go func() {
		if err := wsClient.Start(runCtx); err != nil {
			logger.ErrorCF("feishu", "Feishu websocket stopped with error", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}()

	return nil
}

func (c *FeishuChannel) Stop(ctx context.Context) error {
	c.mu.Lock()
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
	c.wsClient = nil
	c.mu.Unlock()

	c.setRunning(false)
	logger.InfoC("feishu", "Feishu channel stopped")
	return nil
}

func (c *FeishuChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return fmt.Errorf("feishu channel not running")
	}

	if msg.ChatID == "" {
		return fmt.Errorf("chat ID is empty")
	}

	payload, err := json.Marshal(map[string]string{"text": msg.Content})
	if err != nil {
		return fmt.Errorf("failed to marshal feishu content: %w", err)
	}

	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(larkim.ReceiveIdTypeChatId).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(msg.ChatID).
			MsgType(larkim.MsgTypeText).
			Content(string(payload)).
			Uuid(fmt.Sprintf("picoclaw-%d", time.Now().UnixNano())).
			Build()).
		Build()

	resp, err := c.client.Im.V1.Message.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to send feishu message: %w", err)
	}

	if !resp.Success() {
		return fmt.Errorf("feishu api error: code=%d msg=%s", resp.Code, resp.Msg)
	}

	logger.DebugCF("feishu", "Feishu message sent", map[string]interface{}{
		"chat_id": msg.ChatID,
	})

	return nil
}

func (c *FeishuChannel) handleMessageReceive(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
	if event == nil || event.Event == nil || event.Event.Message == nil {
		return nil
	}

	message := event.Event.Message
	sender := event.Event.Sender

	chatID := stringValue(message.ChatId)
	if chatID == "" {
		return nil
	}

	senderID := extractFeishuSenderID(sender)
	if senderID == "" {
		senderID = "unknown"
	}

	content := extractFeishuMessageContent(message)
	if content == "" {
		content = "[empty message]"
	}

	// Check allowlist before processing
	if !c.IsAllowed(senderID) {
		logger.DebugCF("feishu", "Message blocked by allowlist", map[string]interface{}{
			"sender_id": senderID,
		})
		return nil
	}

	// Determine chat type: p2p = direct, group = group chat
	chatType := stringValue(message.ChatType)
	isGroup := chatType == "group"
	isDirect := chatType == "p2p"

	// Check if bot was mentioned
	// Without a reliable bot ID in scope, we conservatively only
	// treat mentions in direct (p2p) chats as relevant to the bot.
	wasMentioned := false
	if isDirect && message.Mentions != nil && len(message.Mentions) > 0 {
		wasMentioned = true
	}

	metadata := map[string]string{}
	messageID := stringValue(message.MessageId)
	if messageID != "" {
		metadata["message_id"] = messageID
	}
	if messageType := stringValue(message.MessageType); messageType != "" {
		metadata["message_type"] = messageType
	}
	if chatType != "" {
		metadata["chat_type"] = chatType
	}
	if sender != nil && sender.TenantKey != nil {
		metadata["tenant_key"] = *sender.TenantKey
	}

	logger.InfoCF("feishu", "Feishu message received", map[string]interface{}{
		"sender_id": senderID,
		"chat_id":   chatID,
		"preview":   utils.Truncate(content, 80),
	})

	// Add emoji ack reaction based on configuration
	ackReaction := c.MessagesConfig().AckReaction
	if ackReaction != "" && messageID != "" {
		shouldAck := ShouldAckReaction(AckReactionParams{
			Scope:               AckReactionScope(c.MessagesConfig().AckReactionScope),
			IsDirect:            isDirect,
			IsGroup:             isGroup,
			IsMentionableGroup:  true, // Feishu groups support mentions
			RequireMention:      true,
			CanDetectMention:    true,
			WasMentioned:        wasMentioned,
			ShouldBypassMention: false,
		})

		if shouldAck {
			if err := c.addMessageReaction(ctx, messageID, ackReaction); err != nil {
				logger.ErrorCF("feishu", "Failed to add emoji reaction", map[string]interface{}{
					"error":      err.Error(),
					"message_id": messageID,
				})
			}
		}
	}

	c.HandleMessage(senderID, chatID, content, nil, metadata)
	return nil
}

func extractFeishuSenderID(sender *larkim.EventSender) string {
	if sender == nil || sender.SenderId == nil {
		return ""
	}

	if sender.SenderId.UserId != nil && *sender.SenderId.UserId != "" {
		return *sender.SenderId.UserId
	}
	if sender.SenderId.OpenId != nil && *sender.SenderId.OpenId != "" {
		return *sender.SenderId.OpenId
	}
	if sender.SenderId.UnionId != nil && *sender.SenderId.UnionId != "" {
		return *sender.SenderId.UnionId
	}

	return ""
}

func extractFeishuMessageContent(message *larkim.EventMessage) string {
	if message == nil || message.Content == nil || *message.Content == "" {
		return ""
	}

	if message.MessageType != nil && *message.MessageType == larkim.MsgTypeText {
		var textPayload struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal([]byte(*message.Content), &textPayload); err == nil {
			return textPayload.Text
		}
	}

	return *message.Content
}

func stringValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func (c *FeishuChannel) addMessageReaction(ctx context.Context, messageID, emojiType string) error {
	req := larkim.NewCreateMessageReactionReqBuilder().
		MessageId(messageID).
		Body(larkim.NewCreateMessageReactionReqBodyBuilder().
			ReactionType(larkim.NewEmojiBuilder().
				EmojiType(emojiType).
				Build()).
			Build()).
		Build()

	resp, err := c.client.Im.V1.MessageReaction.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create message reaction: %w", err)
	}

	if !resp.Success() {
		return fmt.Errorf("feishu reaction api error: code=%d msg=%s", resp.Code, resp.Msg)
	}

	logger.DebugCF("feishu", "Emoji reaction added", map[string]interface{}{
		"message_id": messageID,
		"emoji_type": emojiType,
	})

	return nil
}
