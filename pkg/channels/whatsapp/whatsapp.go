package whatsapp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/identity"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/utils"
)

const (
	whatsAppReadTimeout    = 90 * time.Second
	whatsAppWriteTimeout   = 10 * time.Second
	whatsAppReconnectDelay = 2 * time.Second
)

type WhatsAppChannel struct {
	*channels.BaseChannel
	conn      *websocket.Conn
	config    *config.WhatsAppSettings
	url       string
	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.Mutex
	connected bool
}

func NewWhatsAppChannel(
	bc *config.Channel,
	cfg *config.WhatsAppSettings,
	bus *bus.MessageBus,
) (*WhatsAppChannel, error) {
	base := channels.NewBaseChannel(
		"whatsapp",
		cfg,
		bus,
		bc.AllowFrom,
		channels.WithMaxMessageLength(65536),
		channels.WithReasoningChannelID(bc.ReasoningChannelID),
	)

	return &WhatsAppChannel{
		BaseChannel: base,
		config:      cfg,
		url:         cfg.BridgeURL,
		connected:   false,
	}, nil
}

func (c *WhatsAppChannel) Start(ctx context.Context) error {
	logger.InfoCF("whatsapp", "Starting WhatsApp channel", map[string]any{
		"bridge_url": c.url,
	})

	c.ctx, c.cancel = context.WithCancel(ctx)

	if err := c.connect(); err != nil {
		c.cancel()
		return err
	}

	c.SetRunning(true)
	logger.InfoC("whatsapp", "WhatsApp channel connected")

	go c.listen()

	return nil
}

func (c *WhatsAppChannel) connect() error {
	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 10 * time.Second

	conn, resp, err := dialer.Dial(c.url, nil)
	if resp != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		return fmt.Errorf("failed to connect to WhatsApp bridge: %w", err)
	}

	c.configureConn(conn)

	c.mu.Lock()
	oldConn := c.conn
	c.conn = conn
	c.connected = true
	c.mu.Unlock()

	if oldConn != nil && oldConn != conn {
		_ = oldConn.Close()
	}

	return nil
}

func (c *WhatsAppChannel) configureConn(conn *websocket.Conn) {
	_ = conn.SetReadDeadline(time.Now().Add(whatsAppReadTimeout))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(whatsAppReadTimeout))
	})
	conn.SetPingHandler(func(appData string) error {
		if err := conn.SetReadDeadline(time.Now().Add(whatsAppReadTimeout)); err != nil {
			return err
		}
		return conn.WriteControl(
			websocket.PongMessage,
			[]byte(appData),
			time.Now().Add(whatsAppWriteTimeout),
		)
	})
}

func (c *WhatsAppChannel) Stop(ctx context.Context) error {
	logger.InfoC("whatsapp", "Stopping WhatsApp channel...")

	// Cancel context first to signal listen goroutine to exit
	if c.cancel != nil {
		c.cancel()
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			logger.ErrorCF("whatsapp", "Error closing WhatsApp connection", map[string]any{
				"error": err.Error(),
			})
		}
		c.conn = nil
	}

	c.connected = false
	c.SetRunning(false)

	return nil
}

func (c *WhatsAppChannel) Send(ctx context.Context, msg bus.OutboundMessage) ([]string, error) {
	if !c.IsRunning() {
		return nil, channels.ErrNotRunning
	}

	// Check ctx before acquiring lock
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return nil, fmt.Errorf("whatsapp connection not established: %w", channels.ErrTemporary)
	}

	payload := map[string]any{
		"type":    "message",
		"to":      msg.ChatID,
		"content": msg.Content,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}

	_ = c.conn.SetWriteDeadline(time.Now().Add(whatsAppWriteTimeout))
	if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		_ = c.conn.SetWriteDeadline(time.Time{})
		return nil, fmt.Errorf("whatsapp send: %w", channels.ErrTemporary)
	}
	_ = c.conn.SetWriteDeadline(time.Time{})

	return nil, nil
}

func (c *WhatsAppChannel) listen() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		conn := c.currentConn()
		if conn == nil {
			if err := c.connect(); err != nil {
				logger.WarnCF("whatsapp", "WhatsApp reconnect failed", map[string]any{
					"error": err.Error(),
				})
				if !c.waitBeforeReconnect() {
					return
				}
				continue
			}
			logger.InfoC("whatsapp", "WhatsApp channel reconnected")
			continue
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			logger.ErrorCF("whatsapp", "WhatsApp read error", map[string]any{
				"error": err.Error(),
			})
			c.disconnectConn(conn)
			continue
		}
		_ = conn.SetReadDeadline(time.Now().Add(whatsAppReadTimeout))

		var msg map[string]any
		if err := json.Unmarshal(message, &msg); err != nil {
			logger.ErrorCF("whatsapp", "Failed to unmarshal WhatsApp message", map[string]any{
				"error": err.Error(),
			})
			continue
		}

		msgType, ok := msg["type"].(string)
		if !ok {
			continue
		}

		if msgType == "message" {
			go c.handleIncomingMessage(msg)
		}
	}
}

func (c *WhatsAppChannel) currentConn() *websocket.Conn {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn
}

func (c *WhatsAppChannel) disconnectConn(conn *websocket.Conn) {
	c.mu.Lock()
	if c.conn == conn {
		c.conn = nil
		c.connected = false
	}
	c.mu.Unlock()
	_ = conn.Close()
}

func (c *WhatsAppChannel) waitBeforeReconnect() bool {
	select {
	case <-c.ctx.Done():
		return false
	case <-time.After(whatsAppReconnectDelay):
		return true
	}
}

func (c *WhatsAppChannel) handleIncomingMessage(msg map[string]any) {
	senderID, ok := msg["from"].(string)
	if !ok {
		return
	}

	chatID, ok := msg["chat"].(string)
	if !ok {
		chatID = senderID
	}

	content, ok := msg["content"].(string)
	if !ok {
		content = ""
	}

	var mediaPaths []string
	if mediaData, ok := msg["media"].([]any); ok {
		mediaPaths = make([]string, 0, len(mediaData))
		for _, m := range mediaData {
			if path, ok := m.(string); ok {
				mediaPaths = append(mediaPaths, path)
			}
		}
	}

	metadata := make(map[string]string)
	var messageID string
	if mid, ok := msg["id"].(string); ok {
		messageID = mid
	}
	if userName, ok := msg["from_name"].(string); ok {
		metadata["user_name"] = userName
	}

	logger.InfoCF("whatsapp", "WhatsApp message received", map[string]any{
		"sender":  senderID,
		"preview": utils.Truncate(content, 50),
	})

	sender := bus.SenderInfo{
		Platform:    "whatsapp",
		PlatformID:  senderID,
		CanonicalID: identity.BuildCanonicalID("whatsapp", senderID),
	}
	if display, ok := metadata["user_name"]; ok {
		sender.DisplayName = display
	}

	if !c.IsAllowedSender(sender) {
		return
	}

	inboundCtx := bus.InboundContext{
		Channel:   "whatsapp",
		ChatID:    chatID,
		SenderID:  senderID,
		MessageID: messageID,
		Raw:       metadata,
	}
	if chatID == senderID {
		inboundCtx.ChatType = "direct"
	} else {
		inboundCtx.ChatType = "group"
	}

	c.HandleInboundContext(c.ctx, chatID, content, mediaPaths, inboundCtx, sender)
}
