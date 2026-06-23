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
	// pongWait is how long the client will wait for a pong from the router
	// before considering the connection stale. Must be > router's pingInterval (30s).
	pongWait = 90 * time.Second

	// writeTimeout is the deadline for sending a message to the router.
	writeTimeout = 10 * time.Second

	// maxReconnectAttempts is how many times to retry connecting before giving up.
	maxReconnectAttempts = 3
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

	if err := c.dial(); err != nil {
		c.cancel()
		return err
	}

	c.SetRunning(true)
	logger.InfoC("whatsapp", "WhatsApp channel connected")

	go c.listen()

	return nil
}

// dial establishes a WebSocket connection to the bridge router and
// configures ping/pong keepalive and read deadlines.
func (c *WhatsAppChannel) dial() error {
	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 10 * time.Second

	conn, resp, err := dialer.Dial(c.url, nil)
	if resp != nil {
		resp.Body.Close()
	}
	if err != nil {
		return fmt.Errorf("failed to connect to WhatsApp bridge: %w", err)
	}

	// Pong handler: extend the read deadline every time the router
	// responds to our ping with a pong (or sends its own).
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	// Initial read deadline so a silent connection is detected promptly.
	conn.SetReadDeadline(time.Now().Add(pongWait))

	c.mu.Lock()
	c.conn = conn
	c.connected = true
	c.mu.Unlock()

	return nil
}

// reconnect attempts to re-establish the WebSocket connection after a
// disconnect. Uses exponential backoff between attempts.
func (c *WhatsAppChannel) reconnect() error {
	backoff := time.Second
	for i := range maxReconnectAttempts {
		logger.InfoCF("whatsapp", "Reconnect attempt %d/%d", map[string]any{
			"attempt": i + 1,
			"max":     maxReconnectAttempts,
		})

		if err := c.dial(); err != nil {
			logger.ErrorCF("whatsapp", "Reconnect attempt failed", map[string]any{
				"attempt": i + 1,
				"error":   err.Error(),
			})
			if i < maxReconnectAttempts-1 {
				time.Sleep(backoff)
				backoff *= 2
			}
			continue
		}

		logger.InfoC("whatsapp", "Reconnected successfully")
		return nil
	}

	return fmt.Errorf("all %d reconnect attempts failed", maxReconnectAttempts)
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

	if err := c.conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
		return nil, fmt.Errorf("whatsapp set write deadline: %w", channels.ErrTemporary)
	}
	if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		c.conn.SetWriteDeadline(time.Time{})
		return nil, fmt.Errorf("whatsapp send: %w", channels.ErrTemporary)
	}
	c.conn.SetWriteDeadline(time.Time{})

	return nil, nil
}

// listen is the main read loop. It keeps a single goroutine reading from
// the WebSocket so that control frames (ping/pong) are processed promptly.
// Each incoming message is dispatched to a separate goroutine so processing
// never blocks the read loop.
func (c *WhatsAppChannel) listen() {
	defer logger.InfoC("whatsapp", "listen loop exited")

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		c.mu.Lock()
		conn := c.conn
		c.mu.Unlock()

		if conn == nil {
			// Connection lost — attempt to reconnect
			if err := c.reconnect(); err != nil {
				logger.ErrorC("whatsapp", "permanent reconnect failure, will retry in 5s")
				time.Sleep(5 * time.Second)
				continue
			}
			// Re-fetch the new conn for the read below
			c.mu.Lock()
			conn = c.conn
			c.mu.Unlock()
			if conn == nil {
				continue
			}
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			logger.ErrorCF("whatsapp", "WhatsApp read error", map[string]any{
				"error": err.Error(),
			})

			// Tear down the dead connection
			c.mu.Lock()
			if c.conn != nil {
				c.conn.Close()
				c.conn = nil
			}
			c.connected = false
			c.mu.Unlock()

			// Don't tight-loop — back off before reconnect attempt
			time.Sleep(2 * time.Second)
			continue
		}

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
			// Process message async so the read loop can continue
			// processing control frames (ping/pong) without blocking.
			go c.handleIncomingMessage(msg)
		}
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
