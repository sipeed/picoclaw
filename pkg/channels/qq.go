package channels

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
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

type QQChannel struct {
	*BaseChannel
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
	base := NewBaseChannel("qq", cfg, messageBus, cfg.AllowFrom)

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
			c.setRunning(false)
		}
	}()

	c.setRunning(true)
	logger.InfoC("qq", "QQ bot started successfully")

	return nil
}

func (c *QQChannel) Stop(ctx context.Context) error {
	logger.InfoC("qq", "Stopping QQ bot")
	c.setRunning(false)

	if c.cancel != nil {
		c.cancel()
	}

	return nil
}

func (c *QQChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return fmt.Errorf("QQ bot not running")
	}

	// construct message
	msgToCreate := &dto.MessageToCreate{
		Content: msg.Content,
	}

	// send C2C message
	_, err := c.api.PostC2CMessage(ctx, msg.ChatID, msgToCreate)
	if err != nil {
		logger.ErrorCF("qq", "Failed to send C2C message", map[string]any{
			"error": err.Error(),
		})
		return err
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

		// forward to message bus
		metadata := map[string]string{
			"message_id": data.ID,
			"peer_kind":  "direct",
			"peer_id":    senderID,
		}

		c.HandleMessage(senderID, senderID, content, []string{}, metadata)

		return nil
	}
}

// handleGroupATMessage handles group @messages
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

		// extract message content (remove @bot part)
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

		// forward to message bus (use GroupID as ChatID)
		metadata := map[string]string{
			"message_id": data.ID,
			"group_id":   data.GroupID,
			"peer_kind":  "group",
			"peer_id":    data.GroupID,
		}

		c.HandleMessage(senderID, data.GroupID, content, []string{}, metadata)

		return nil
	}
}

// isDuplicate checks if message is duplicate
func (c *QQChannel) isDuplicate(messageID string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.processedIDs[messageID] {
		return true
	}

	c.processedIDs[messageID] = true

	// simple cleanup: limit map size
	if len(c.processedIDs) > 10000 {
		// clear half
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
