package pico

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/identity"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// PicoClientChannel connects to a remote Pico Protocol WebSocket server.
type PicoClientChannel struct {
	*channels.BaseChannel
	config config.PicoClientConfig
	conn   *picoConn
	mu     sync.Mutex
	ctx    context.Context
	cancel context.CancelFunc
}

// NewPicoClientChannel creates a new Pico Protocol client channel.
func NewPicoClientChannel(
	cfg config.PicoClientConfig,
	messageBus *bus.MessageBus,
) (*PicoClientChannel, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("pico_client url is required")
	}

	base := channels.NewBaseChannel("pico_client", cfg, messageBus, cfg.AllowFrom)

	return &PicoClientChannel{
		BaseChannel: base,
		config:      cfg,
	}, nil
}

// Start dials the remote server and begins reading.
func (c *PicoClientChannel) Start(ctx context.Context) error {
	logger.InfoC("pico_client", "Starting Pico Client channel")
	c.ctx, c.cancel = context.WithCancel(ctx)

	if err := c.dial(); err != nil {
		c.cancel()
		return fmt.Errorf("pico_client initial connect: %w", err)
	}

	c.SetRunning(true)
	go c.reconnectLoop()

	logger.InfoCF("pico_client", "Connected", map[string]any{"url": c.config.URL})
	return nil
}

// Stop closes the connection.
func (c *PicoClientChannel) Stop(ctx context.Context) error {
	logger.InfoC("pico_client", "Stopping Pico Client channel")
	c.SetRunning(false)
	if c.cancel != nil {
		c.cancel()
	}
	c.mu.Lock()
	if c.conn != nil {
		c.conn.close()
	}
	c.mu.Unlock()
	logger.InfoC("pico_client", "Pico Client channel stopped")
	return nil
}

func (c *PicoClientChannel) dial() error {
	header := http.Header{}
	if c.config.Token.String() != "" {
		header.Set("Authorization", "Bearer "+c.config.Token.String())
	}

	ws, resp, err := websocket.DefaultDialer.DialContext(c.ctx, c.config.URL, header)
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	if err != nil {
		return err
	}

	connCtx, connCancel := context.WithCancel(c.ctx)

	pc := &picoConn{
		id:        uuid.New().String(),
		conn:      ws,
		sessionID: c.config.SessionID,
		cancel:    connCancel,
	}
	if pc.sessionID == "" {
		pc.sessionID = uuid.New().String()
	}

	c.mu.Lock()
	c.conn = pc
	c.mu.Unlock()

	go c.readLoop(connCtx, pc)
	return nil
}

// reconnectLoop re-dials when the connection drops.
func (c *PicoClientChannel) reconnectLoop() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		c.mu.Lock()
		pc := c.conn
		c.mu.Unlock()

		if pc == nil || pc.closed.Load() {
			backoff := 5 * time.Second
			logger.InfoC("pico_client", "Reconnecting...")
			if err := c.dial(); err != nil {
				logger.WarnCF("pico_client", "Reconnect failed", map[string]any{
					"error": err.Error(),
				})
				select {
				case <-c.ctx.Done():
					return
				case <-time.After(backoff):
				}
				continue
			}
			logger.InfoC("pico_client", "Reconnected")
		}

		select {
		case <-c.ctx.Done():
			return
		case <-time.After(1 * time.Second):
		}
	}
}

func (c *PicoClientChannel) readLoop(connCtx context.Context, pc *picoConn) {
	defer pc.close()

	readTimeout := time.Duration(c.config.ReadTimeout) * time.Second
	if readTimeout <= 0 {
		readTimeout = 60 * time.Second
	}

	_ = pc.conn.SetReadDeadline(time.Now().Add(readTimeout))
	pc.conn.SetPongHandler(func(string) error {
		return pc.conn.SetReadDeadline(time.Now().Add(readTimeout))
	})

	pingInterval := time.Duration(c.config.PingInterval) * time.Second
	if pingInterval <= 0 {
		pingInterval = 30 * time.Second
	}
	go c.pingLoop(connCtx, pc, pingInterval)

	for {
		select {
		case <-connCtx.Done():
			return
		default:
		}

		_, raw, err := pc.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(
				err,
				websocket.CloseGoingAway,
				websocket.CloseNormalClosure,
			) {
				logger.DebugCF("pico_client", "Read error", map[string]any{
					"error": err.Error(),
				})
			}
			return
		}

		_ = pc.conn.SetReadDeadline(time.Now().Add(readTimeout))

		var msg PicoMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}

		c.handleInbound(pc, msg)
	}
}

func (c *PicoClientChannel) pingLoop(connCtx context.Context, pc *picoConn, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-connCtx.Done():
			return
		case <-ticker.C:
			if pc.closed.Load() {
				return
			}
			pc.writeMu.Lock()
			err := pc.conn.WriteMessage(websocket.PingMessage, nil)
			pc.writeMu.Unlock()
			if err != nil {
				return
			}
		}
	}
}

// handleInbound processes messages from the remote server.
// In client mode the server sends message.create (responses) and the client
// sends message.send (user input). We treat message.create from the server
// as inbound user messages to feed into the agent loop.
func (c *PicoClientChannel) handleInbound(pc *picoConn, msg PicoMessage) {
	switch msg.Type {
	case TypePong:
		// response to our ping, ignore
	case TypeMessageCreate:
		// Server sent us a message — treat as inbound
		c.handleServerMessage(pc, msg)
	default:
		logger.DebugCF("pico_client", "Ignoring message type", map[string]any{
			"type": msg.Type,
		})
	}
}

func (c *PicoClientChannel) handleServerMessage(pc *picoConn, msg PicoMessage) {
	content, _ := msg.Payload["content"].(string)
	if strings.TrimSpace(content) == "" {
		return
	}

	sessionID := msg.SessionID
	if sessionID == "" {
		sessionID = pc.sessionID
	}

	chatID := "pico_client:" + sessionID
	senderID := "pico-remote"
	peer := bus.Peer{Kind: "direct", ID: chatID}

	sender := bus.SenderInfo{
		Platform:    "pico_client",
		PlatformID:  senderID,
		CanonicalID: identity.BuildCanonicalID("pico_client", senderID),
	}

	if !c.IsAllowedSender(sender) {
		return
	}

	c.HandleMessage(c.ctx, peer, msg.ID, senderID, chatID, content, nil, map[string]string{
		"platform":   "pico_client",
		"session_id": sessionID,
	}, sender)
}

// Send sends a message to the remote server.
func (c *PicoClientChannel) Send(ctx context.Context, msg bus.OutboundMessage) ([]string, error) {
	if !c.IsRunning() {
		return nil, channels.ErrNotRunning
	}
	c.mu.Lock()
	pc := c.conn
	c.mu.Unlock()
	if pc == nil || pc.closed.Load() {
		return nil, channels.ErrSendFailed
	}

	outMsg := newMessage(TypeMessageSend, map[string]any{
		"content": msg.Content,
	})
	outMsg.SessionID = strings.TrimPrefix(msg.ChatID, "pico_client:")
	return nil, pc.writeJSON(outMsg)
}

// StartTyping implements channels.TypingCapable.
func (c *PicoClientChannel) StartTyping(ctx context.Context, chatID string) (func(), error) {
	c.mu.Lock()
	pc := c.conn
	c.mu.Unlock()
	if pc == nil || pc.closed.Load() {
		return func() {}, nil
	}

	startMsg := newMessage(TypeTypingStart, nil)
	startMsg.SessionID = strings.TrimPrefix(chatID, "pico_client:")
	if err := pc.writeJSON(startMsg); err != nil {
		return func() {}, err
	}
	return func() {
		c.mu.Lock()
		currentPC := c.conn
		c.mu.Unlock()
		if currentPC == nil {
			return
		}
		stopMsg := newMessage(TypeTypingStop, nil)
		stopMsg.SessionID = strings.TrimPrefix(chatID, "pico_client:")
		currentPC.writeJSON(stopMsg)
	}, nil
}
