package qq

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/tencent-connect/botgo"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/event"
	"github.com/tencent-connect/botgo/openapi"
	"github.com/tencent-connect/botgo/token"
	"golang.org/x/oauth2"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/identity"
	"github.com/sipeed/picoclaw/pkg/logger"
)

type QQChannel struct {
	*channels.BaseChannel
	config         config.QQConfig
	api            openapi.OpenAPI
	tokenSource    oauth2.TokenSource
	ctx            context.Context
	cancel         context.CancelFunc
	sessionManager botgo.SessionManager
	processedIDs   map[string]bool
	lastMsgIDs     map[string]string
	msgSeqByChat   map[string]uint32
	chatKindByID   map[string]string
	mu             sync.RWMutex
}

const (
	qqChatKindDirect = "direct"
	qqChatKindGroup  = "group"
)

type qqC2CMessageToCreate struct {
	Content string `json:"content,omitempty"`
	MsgType int    `json:"msg_type"`
	MsgID   string `json:"msg_id,omitempty"`
	MsgSeq  uint32 `json:"msg_seq,omitempty"`
}

func (m qqC2CMessageToCreate) GetEventID() string {
	return ""
}

func (m qqC2CMessageToCreate) GetSendType() dto.SendType {
	return dto.Text
}

func NewQQChannel(cfg config.QQConfig, messageBus *bus.MessageBus) (*QQChannel, error) {
	base := channels.NewBaseChannel("qq", cfg, messageBus, cfg.AllowFrom,
		channels.WithGroupTrigger(cfg.GroupTrigger),
		channels.WithReasoningChannelID(cfg.ReasoningChannelID),
	)

	return &QQChannel{
		BaseChannel:  base,
		config:       cfg,
		processedIDs: make(map[string]bool),
		lastMsgIDs:   make(map[string]string),
		msgSeqByChat: make(map[string]uint32),
		chatKindByID: make(map[string]string),
	}, nil
}

func (c *QQChannel) Start(ctx context.Context) error {
	if c.config.AppID == "" || c.config.AppSecret == "" {
		return fmt.Errorf("QQ app_id and app_secret not configured")
	}

	logger.InfoC("qq", "Starting QQ bot (WebSocket mode)")

	// create token source
	credentials := &token.QQBotCredentials{
		AppID:     c.config.AppID,
		AppSecret: c.config.AppSecret,
	}
	c.tokenSource = token.NewQQBotTokenSource(credentials)

	// create child context
	c.ctx, c.cancel = context.WithCancel(ctx)

	// start auto-refresh token goroutine
	if err := token.StartRefreshAccessToken(c.ctx, c.tokenSource); err != nil {
		return fmt.Errorf("failed to start token refresh: %w", err)
	}

	// initialize OpenAPI client
	c.api = botgo.NewOpenAPI(c.config.AppID, c.tokenSource).WithTimeout(5 * time.Second)

	// register event handlers
	intent := event.RegisterHandlers(
		c.handleC2CMessage(),
		c.handleGroupATMessage(),
	)

	// get WebSocket endpoint
	wsInfo, err := c.api.WS(c.ctx, nil, "")
	if err != nil {
		return fmt.Errorf("failed to get websocket info: %w", err)
	}

	logger.InfoCF("qq", "Got WebSocket info", map[string]any{
		"shards": wsInfo.Shards,
	})

	// create and save sessionManager
	c.sessionManager = botgo.NewSessionManager()

	// start WebSocket connection in goroutine to avoid blocking
	go func() {
		if err := c.sessionManager.Start(wsInfo, c.tokenSource, &intent); err != nil {
			logger.ErrorCF("qq", "WebSocket session error", map[string]any{
				"error": err.Error(),
			})
			c.SetRunning(false)
		}
	}()

	c.SetRunning(true)
	logger.InfoC("qq", "QQ bot started successfully")

	return nil
}

func (c *QQChannel) Stop(ctx context.Context) error {
	logger.InfoC("qq", "Stopping QQ bot")
	c.SetRunning(false)

	if c.cancel != nil {
		c.cancel()
	}

	return nil
}

func (c *QQChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	// construct message
	msgToCreate := c.buildC2CMessage(msg.ChatID, msg.Content)

	chatKind := c.resolveChatKind(msg.ChatID)

	var err error
	if chatKind == qqChatKindGroup {
		_, err = c.api.PostGroupMessage(ctx, msg.ChatID, msgToCreate)
	} else {
		_, err = c.api.PostC2CMessage(ctx, msg.ChatID, msgToCreate)
	}
	if err != nil {
		logger.ErrorCF("qq", "Failed to send QQ message", map[string]any{
			"error": err.Error(),
			"chat":  msg.ChatID,
			"kind":  chatKind,
		})
		return fmt.Errorf("qq send: %w", channels.ErrTemporary)
	}

	return nil
}

// handleC2CMessage handles QQ private messages
func (c *QQChannel) handleC2CMessage() event.C2CMessageEventHandler {
	return func(event *dto.WSPayload, data *dto.WSC2CMessageData) error {
		// deduplication check
		if c.isDuplicate(data.ID) {
			return nil
		}

		// extract user info
		var senderID string
		if data.Author != nil && data.Author.ID != "" {
			senderID = data.Author.ID
		} else {
			logger.WarnC("qq", "Received message with no sender ID")
			return nil
		}

		// extract message content
		content := data.Content
		if content == "" {
			logger.DebugC("qq", "Received empty message, ignoring")
			return nil
		}

		logger.InfoCF("qq", "Received C2C message", map[string]any{
			"sender": senderID,
			"length": len(content),
		})

		// 转发到消息总线
		metadata := map[string]string{}

		sender := bus.SenderInfo{
			Platform:    "qq",
			PlatformID:  data.Author.ID,
			CanonicalID: identity.BuildCanonicalID("qq", data.Author.ID),
		}

<<<<<<< HEAD:pkg/channels/qq.go
		c.recordInboundMessage(senderID, data.ID, qqChatKindDirect)

		c.HandleMessage(senderID, senderID, content, []string{}, metadata)
=======
		if !c.IsAllowedSender(sender) {
			return nil
		}

		c.HandleMessage(c.ctx,
			bus.Peer{Kind: "direct", ID: senderID},
			data.ID,
			senderID,
			senderID,
			content,
			[]string{},
			metadata,
			sender,
		)
>>>>>>> origin/main:pkg/channels/qq/qq.go

		return nil
	}
}

// handleGroupATMessage handles QQ group @ messages
func (c *QQChannel) handleGroupATMessage() event.GroupATMessageEventHandler {
	return func(event *dto.WSPayload, data *dto.WSGroupATMessageData) error {
		// deduplication check
		if c.isDuplicate(data.ID) {
			return nil
		}

		// extract user info
		var senderID string
		if data.Author != nil && data.Author.ID != "" {
			senderID = data.Author.ID
		} else {
			logger.WarnC("qq", "Received group message with no sender ID")
			return nil
		}

		// extract message content (remove @ bot part)
		content := data.Content
		if content == "" {
			logger.DebugC("qq", "Received empty group message, ignoring")
			return nil
		}

		// GroupAT event means bot is always mentioned; apply group trigger filtering
		respond, cleaned := c.ShouldRespondInGroup(true, content)
		if !respond {
			return nil
		}
		content = cleaned

		logger.InfoCF("qq", "Received group AT message", map[string]any{
			"sender": senderID,
			"group":  data.GroupID,
			"length": len(content),
		})

		// 转发到消息总线（使用 GroupID 作为 ChatID）
		metadata := map[string]string{
			"group_id": data.GroupID,
		}

<<<<<<< HEAD:pkg/channels/qq.go
		c.recordInboundMessage(data.GroupID, data.ID, qqChatKindGroup)

		c.HandleMessage(senderID, data.GroupID, content, []string{}, metadata)
=======
		sender := bus.SenderInfo{
			Platform:    "qq",
			PlatformID:  data.Author.ID,
			CanonicalID: identity.BuildCanonicalID("qq", data.Author.ID),
		}

		if !c.IsAllowedSender(sender) {
			return nil
		}

		c.HandleMessage(c.ctx,
			bus.Peer{Kind: "group", ID: data.GroupID},
			data.ID,
			senderID,
			data.GroupID,
			content,
			[]string{},
			metadata,
			sender,
		)
>>>>>>> origin/main:pkg/channels/qq/qq.go

		return nil
	}
}

// isDuplicate 检查消息是否重复
func (c *QQChannel) isDuplicate(messageID string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.processedIDs[messageID] {
		return true
	}

	c.processedIDs[messageID] = true

	// 简单清理：限制 map 大小
	if len(c.processedIDs) > 10000 {
		// 清空一半
		count := 0
		for id := range c.processedIDs {
			if count >= 5000 {
				break
			}
			delete(c.processedIDs, id)
			count++
		}
	}

	return false
}

func (c *QQChannel) recordInboundMessage(chatID, messageID, chatKind string) {
	if chatID == "" || messageID == "" {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.lastMsgIDs[chatID] = messageID
	c.msgSeqByChat[chatID] = 0
	if chatKind != "" {
		c.chatKindByID[chatID] = chatKind
	}
}

func (c *QQChannel) buildC2CMessage(chatID, content string) *qqC2CMessageToCreate {
	msg := &qqC2CMessageToCreate{
		Content: content,
		MsgType: int(dto.TextMsg),
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if lastMsgID := c.lastMsgIDs[chatID]; lastMsgID != "" {
		c.msgSeqByChat[chatID]++
		msg.MsgID = lastMsgID
		msg.MsgSeq = c.msgSeqByChat[chatID]
	}

	return msg
}

func (c *QQChannel) resolveChatKind(chatID string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if kind := c.chatKindByID[chatID]; kind != "" {
		return kind
	}

	return qqChatKindDirect
}
