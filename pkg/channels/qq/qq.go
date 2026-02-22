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
	mu             sync.RWMutex
}

func NewQQChannel(cfg config.QQConfig, messageBus *bus.MessageBus) (*QQChannel, error) {
	base := channels.NewBaseChannel("qq", cfg, messageBus, cfg.AllowFrom)

	return &QQChannel{
		BaseChannel:  base,
		config:       cfg,
		processedIDs: make(map[string]bool),
	}, nil
}

func (c *QQChannel) Start(ctx context.Context) error {
	if c.config.AppID == "" || c.config.AppSecret == "" {
		return fmt.Errorf("QQ app_id and app_secret not configured")
	}

	logger.InfoC("qq", "Starting QQ bot (WebSocket mode)")

	// 创建 token source
	credentials := &token.QQBotCredentials{
		AppID:     c.config.AppID,
		AppSecret: c.config.AppSecret,
	}
	c.tokenSource = token.NewQQBotTokenSource(credentials)

	// 创建子 context
	c.ctx, c.cancel = context.WithCancel(ctx)

	// 启动自动刷新 token 协程
	if err := token.StartRefreshAccessToken(c.ctx, c.tokenSource); err != nil {
		return fmt.Errorf("failed to start token refresh: %w", err)
	}

	// 初始化 OpenAPI 客户端
	c.api = botgo.NewOpenAPI(c.config.AppID, c.tokenSource).WithTimeout(5 * time.Second)

	// 注册事件处理器
	intent := event.RegisterHandlers(
		c.handleC2CMessage(),
		c.handleGroupATMessage(),
	)

	// 获取 WebSocket 接入点
	wsInfo, err := c.api.WS(c.ctx, nil, "")
	if err != nil {
		return fmt.Errorf("failed to get websocket info: %w", err)
	}

	logger.InfoCF("qq", "Got WebSocket info", map[string]any{
		"shards": wsInfo.Shards,
	})

	// 创建并保存 sessionManager
	c.sessionManager = botgo.NewSessionManager()

	// 在 goroutine 中启动 WebSocket 连接，避免阻塞
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

	// 构造消息
	msgToCreate := &dto.MessageToCreate{
		Content: msg.Content,
	}

	// C2C 消息发送
	_, err := c.api.PostC2CMessage(ctx, msg.ChatID, msgToCreate)
	if err != nil {
		logger.ErrorCF("qq", "Failed to send C2C message", map[string]any{
			"error": err.Error(),
		})
		return fmt.Errorf("qq send: %w", channels.ErrTemporary)
	}

	return nil
}

// handleC2CMessage 处理 QQ 私聊消息
func (c *QQChannel) handleC2CMessage() event.C2CMessageEventHandler {
	return func(event *dto.WSPayload, data *dto.WSC2CMessageData) error {
		// 去重检查
		if c.isDuplicate(data.ID) {
			return nil
		}

		// 提取用户信息
		var senderID string
		if data.Author != nil && data.Author.ID != "" {
			senderID = data.Author.ID
		} else {
			logger.WarnC("qq", "Received message with no sender ID")
			return nil
		}

		// 提取消息内容
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

		c.HandleMessage(
			bus.Peer{Kind: "direct", ID: senderID},
			data.ID,
			senderID,
			senderID,
			content,
			[]string{},
			metadata,
		)

		return nil
	}
}

// handleGroupATMessage 处理群@消息
func (c *QQChannel) handleGroupATMessage() event.GroupATMessageEventHandler {
	return func(event *dto.WSPayload, data *dto.WSGroupATMessageData) error {
		// 去重检查
		if c.isDuplicate(data.ID) {
			return nil
		}

		// 提取用户信息
		var senderID string
		if data.Author != nil && data.Author.ID != "" {
			senderID = data.Author.ID
		} else {
			logger.WarnC("qq", "Received group message with no sender ID")
			return nil
		}

		// 提取消息内容（去掉 @ 机器人部分）
		content := data.Content
		if content == "" {
			logger.DebugC("qq", "Received empty group message, ignoring")
			return nil
		}

		logger.InfoCF("qq", "Received group AT message", map[string]any{
			"sender": senderID,
			"group":  data.GroupID,
			"length": len(content),
		})

		// 转发到消息总线（使用 GroupID 作为 ChatID）
		metadata := map[string]string{
			"group_id": data.GroupID,
		}

		c.HandleMessage(
			bus.Peer{Kind: "group", ID: data.GroupID},
			data.ID,
			senderID,
			data.GroupID,
			content,
			[]string{},
			metadata,
		)

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
