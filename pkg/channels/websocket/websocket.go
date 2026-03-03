package websocket

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
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
	conn    *websocket.Conn
	ctx     context.Context
	cancel  context.CancelFunc
	mu      sync.Mutex // guards conn
	writeMu sync.Mutex // guards writes
}

// NewWebSocketChannel creates a new WebSocket channel (stub).
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
	}
	return ch, nil
}

func (c *WebSocketChannel) Start(ctx context.Context) error {
	return nil
}

func (c *WebSocketChannel) Stop(ctx context.Context) error {
	return nil
}

func (c *WebSocketChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}
	return nil
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
