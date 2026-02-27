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

	streamingMu    sync.RWMutex
	streamingCache map[string]*FeishuStreamingSession

	mu     sync.Mutex
	cancel context.CancelFunc
}

func NewFeishuChannel(cfg config.FeishuConfig, bus *bus.MessageBus) (*FeishuChannel, error) {
	base := NewBaseChannel("feishu", cfg, bus, cfg.AllowFrom)

	return &FeishuChannel{
		BaseChannel:    base,
		config:         cfg,
		client:         lark.NewClient(cfg.AppID, cfg.AppSecret),
		streamingCache: make(map[string]*FeishuStreamingSession),
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
			logger.ErrorCF("feishu", "Feishu websocket stopped with error", map[string]any{
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

	msgType := c.resolveMessageType(msg.Content, msg.MsgType)

	switch msgType {
	case "card":
		return c.sendCardMessage(ctx, msg)
	case "streaming":
		return c.sendStreamingMessage(ctx, msg)
	default:
		return c.sendTextMessage(ctx, msg)
	}
}

func (c *FeishuChannel) resolveMessageType(content, explicitMsgType string) string {
	if explicitMsgType == "card" || explicitMsgType == "interactive" {
		return "card"
	}
	if explicitMsgType == "text" {
		return "text"
	}

	renderMode := c.config.RenderMode
	if renderMode == "" {
		renderMode = "raw"
	}

	switch renderMode {
	case "card":
		return "card"
	case "raw":
		return "text"
	case "auto":
		if ShouldUseCard(content) {
			return "card"
		}
		return "text"
	default:
		return "text"
	}
}

func (c *FeishuChannel) sendTextMessage(ctx context.Context, msg bus.OutboundMessage) error {
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

	logger.DebugCF("feishu", "Feishu text message sent", map[string]any{
		"chat_id": msg.ChatID,
	})

	return nil
}

func (c *FeishuChannel) sendCardMessage(ctx context.Context, msg bus.OutboundMessage) error {
	card := BuildMarkdownCard(msg.Content)
	cardJSON, err := json.Marshal(card)
	if err != nil {
		return fmt.Errorf("failed to marshal card: %w", err)
	}

	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(larkim.ReceiveIdTypeChatId).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(msg.ChatID).
			MsgType(larkim.MsgTypeInteractive).
			Content(string(cardJSON)).
			Uuid(fmt.Sprintf("picoclaw-card-%d", time.Now().UnixNano())).
			Build()).
		Build()

	resp, err := c.client.Im.V1.Message.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to send feishu card: %w", err)
	}

	if !resp.Success() {
		return fmt.Errorf("feishu card api error: code=%d msg=%s", resp.Code, resp.Msg)
	}

	logger.DebugCF("feishu", "Feishu card message sent", map[string]any{
		"chat_id": msg.ChatID,
	})

	return nil
}

func (c *FeishuChannel) sendStreamingMessage(ctx context.Context, msg bus.OutboundMessage) error {
	streamingEnabled := c.config.StreamingEnabled
	if !streamingEnabled {
		return c.sendCardMessage(ctx, msg)
	}

	c.streamingMu.Lock()
	session, ok := c.streamingCache[msg.ChatID]
	if !ok {
		interval := c.config.StreamingInterval
		if interval <= 0 {
			interval = 100
		}
		session = NewStreamingSession(c.client, c.config.AppID, c.config.AppSecret, "", interval)
		c.streamingCache[msg.ChatID] = session
	}
	c.streamingMu.Unlock()

	if !session.IsActive() {
		if err := session.Start(ctx, msg.ChatID); err != nil {
			logger.ErrorCF("feishu", "Failed to start streaming session", map[string]any{
				"error": err.Error(),
			})
			return c.sendCardMessage(ctx, msg)
		}
	}

	session.Update(msg.Content)

	if msg.Metadata != nil && msg.Metadata["streaming_end"] == "true" {
		c.streamingMu.Lock()
		delete(c.streamingCache, msg.ChatID)
		c.streamingMu.Unlock()
		session.Close(msg.Content)
		logger.DebugCF("feishu", "Streaming session closed", map[string]any{
			"chat_id": msg.ChatID,
		})
	}

	return nil
}

func (c *FeishuChannel) handleMessageReceive(_ context.Context, event *larkim.P2MessageReceiveV1) error {
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

	metadata := map[string]string{}
	if messageID := stringValue(message.MessageId); messageID != "" {
		metadata["message_id"] = messageID
	}
	if messageType := stringValue(message.MessageType); messageType != "" {
		metadata["message_type"] = messageType
	}
	if chatType := stringValue(message.ChatType); chatType != "" {
		metadata["chat_type"] = chatType
	}
	if sender != nil && sender.TenantKey != nil {
		metadata["tenant_key"] = *sender.TenantKey
	}

	chatType := stringValue(message.ChatType)
	if chatType == "p2p" {
		metadata["peer_kind"] = "direct"
		metadata["peer_id"] = senderID
	} else {
		metadata["peer_kind"] = "group"
		metadata["peer_id"] = chatID
	}

	logger.InfoCF("feishu", "Feishu message received", map[string]any{
		"sender_id": senderID,
		"chat_id":   chatID,
		"preview":   utils.Truncate(content, 80),
	})

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
