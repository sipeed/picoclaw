package websocket

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
	"github.com/sipeed/picoclaw/pkg/logger"
)

// WSEnvelope is the typed envelope for all WebSocket messages.
type WSEnvelope struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// WSAuthData is the auth handshake payload sent on connect.
type WSAuthData struct {
	AgentID string `json:"agent_id"`
	Token   string `json:"token"`
}

// WSInboundMessage is the inbound message payload from the orchestrator.
type WSInboundMessage struct {
	SenderID   string            `json:"sender_id"`
	ChatID     string            `json:"chat_id"`
	Content    string            `json:"content"`
	SessionKey string            `json:"session_key,omitempty"`
	MessageID  string            `json:"message_id,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// WSOutboundMessage wraps bus.OutboundMessage with content_type for the wire.
type WSOutboundMessage struct {
	Channel     string `json:"channel"`
	ChatID      string `json:"chat_id"`
	Content     string `json:"content"`
	ContentType string `json:"content_type"`
}

// WebSocketChannel connects outward to an orchestrator via WebSocket.
type WebSocketChannel struct {
	*channels.BaseChannel
	config  config.WebSocketConfig
	mb      *bus.MessageBus
	conn    *websocket.Conn
	ctx     context.Context
	cancel  context.CancelFunc
	mu      sync.Mutex // guards conn
	writeMu sync.Mutex // guards writes
}

// NewWebSocketChannel creates a new WebSocket channel.
func NewWebSocketChannel(cfg config.WebSocketConfig, mb *bus.MessageBus) (*WebSocketChannel, error) {
	base := channels.NewBaseChannel(
		"websocket",
		cfg,
		mb,
		[]string(cfg.AllowFrom),
		channels.WithReasoningChannelID(cfg.ReasoningChannelID),
	)
	ch := &WebSocketChannel{
		BaseChannel: base,
		config:      cfg,
		mb:          mb,
	}
	return ch, nil
}

// connect dials the WebSocket server and sends the auth handshake.
func (c *WebSocketChannel) connect() error {
	conn, _, err := websocket.DefaultDialer.Dial(c.config.WSUrl, nil)
	if err != nil {
		return fmt.Errorf("websocket dial: %w", err)
	}

	// Send auth envelope.
	authData, err := json.Marshal(WSAuthData{
		AgentID: c.config.AgentID,
		Token:   c.config.AccessToken,
	})
	if err != nil {
		conn.Close()
		return fmt.Errorf("marshal auth: %w", err)
	}
	env := WSEnvelope{Type: "auth", Data: json.RawMessage(authData)}
	envBytes, err := json.Marshal(env)
	if err != nil {
		conn.Close()
		return fmt.Errorf("marshal auth envelope: %w", err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, envBytes); err != nil {
		conn.Close()
		return fmt.Errorf("write auth: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()
	return nil
}

// listen reads messages from the WebSocket connection and publishes them to the bus.
func (c *WebSocketChannel) listen() {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn == nil {
		return
	}

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			c.mu.Lock()
			c.conn = nil
			c.mu.Unlock()
			return
		}

		env, err := parseEnvelope(data)
		if err != nil {
			continue
		}

		if env.Type != "message" {
			continue
		}

		var msg WSInboundMessage
		if err := json.Unmarshal(env.Data, &msg); err != nil {
			logger.ErrorCF("websocket", "Failed to unmarshal inbound message", map[string]any{
				"error": err.Error(),
			})
			continue
		}

		inbound := bus.InboundMessage{
			Channel:    "websocket",
			SenderID:   msg.SenderID,
			ChatID:     msg.ChatID,
			Content:    msg.Content,
			MessageID:  msg.MessageID,
			SessionKey: msg.SessionKey,
			Metadata:   msg.Metadata,
		}

		if err := c.mb.PublishInbound(c.ctx, inbound); err != nil {
			logger.ErrorCF("websocket", "Failed to publish inbound", map[string]any{
				"error": err.Error(),
			})
			return
		}
	}
}

func (c *WebSocketChannel) Start(ctx context.Context) error {
	c.ctx, c.cancel = context.WithCancel(ctx)

	if err := c.connect(); err != nil {
		if c.config.ReconnectInterval <= 0 {
			return err
		}
		logger.ErrorCF("websocket", "Initial connect failed, will retry", map[string]any{
			"error": err.Error(),
		})
	} else {
		go c.listen()
	}

	if c.config.ReconnectInterval > 0 {
		go c.reconnectLoop()
	}

	c.SetRunning(true)
	return nil
}

func (c *WebSocketChannel) Stop(ctx context.Context) error {
	c.SetRunning(false)
	c.cancel()

	c.mu.Lock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.mu.Unlock()
	return nil
}

func (c *WebSocketChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	out := WSOutboundMessage{
		Channel:     msg.Channel,
		ChatID:      msg.ChatID,
		Content:     msg.Content,
		ContentType: DetectContentType(msg.Content),
	}
	outData, err := json.Marshal(out)
	if err != nil {
		return fmt.Errorf("marshal outbound: %w", err)
	}
	env := WSEnvelope{Type: "message", Data: json.RawMessage(outData)}
	envBytes, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal envelope: %w", err)
	}

	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn == nil {
		return channels.ErrTemporary
	}

	c.writeMu.Lock()
	err = conn.WriteMessage(websocket.TextMessage, envBytes)
	c.writeMu.Unlock()
	if err != nil {
		return channels.ErrTemporary
	}
	return nil
}

// reconnectLoop periodically reconnects when the connection is lost.
func (c *WebSocketChannel) reconnectLoop() {
	interval := time.Duration(c.config.ReconnectInterval) * time.Second
	if interval < time.Second {
		interval = time.Second
	}

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-time.After(interval):
		}

		c.mu.Lock()
		needsReconnect := c.conn == nil
		c.mu.Unlock()

		if !needsReconnect {
			continue
		}

		if err := c.connect(); err != nil {
			logger.ErrorCF("websocket", "Reconnect failed", map[string]any{
				"error": err.Error(),
			})
			continue
		}
		go c.listen()
	}
}

// DetectContentType returns "json" if content is valid JSON, otherwise "text".
func DetectContentType(content string) string {
	if json.Valid([]byte(content)) {
		return "json"
	}
	return "text"
}

// parseEnvelope parses raw bytes into a WSEnvelope.
func parseEnvelope(data []byte) (WSEnvelope, error) {
	var env WSEnvelope
	err := json.Unmarshal(data, &env)
	return env, err
}
