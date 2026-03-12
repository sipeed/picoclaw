package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/identity"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// apiConn represents a single WebSocket connection.
type apiConn struct {
	id        string
	conn      *websocket.Conn
	sessionID string
	writeMu   sync.Mutex
	closed    atomic.Bool
}

// writeJSON sends a JSON message to the connection with write locking.
func (ac *apiConn) writeJSON(v any) error {
	if ac.closed.Load() {
		return fmt.Errorf("connection closed")
	}
	ac.writeMu.Lock()
	defer ac.writeMu.Unlock()
	return ac.conn.WriteJSON(v)
}

// close closes the connection.
func (ac *apiConn) close() {
	if ac.closed.CompareAndSwap(false, true) {
		ac.conn.Close()
	}
}

// APIChannel implements a WebSocket API channel for Flutter app integration.
// It supports streaming responses for real-time typewriter effect.
type APIChannel struct {
	*channels.BaseChannel
	config      config.APIConfig
	upgrader    websocket.Upgrader
	connections sync.Map // connID → *apiConn
	connCount   atomic.Int32
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewAPIChannel creates a new API channel.
func NewAPIChannel(cfg config.APIConfig, messageBus *bus.MessageBus) (*APIChannel, error) {
	if cfg.Token == "" {
		return nil, fmt.Errorf("api token is required")
	}

	base := channels.NewBaseChannel("api", cfg, messageBus, cfg.AllowFrom)

	allowOrigins := cfg.AllowOrigins
	checkOrigin := func(r *http.Request) bool {
		if len(allowOrigins) == 0 {
			return true // allow all if not configured
		}
		origin := r.Header.Get("Origin")
		for _, allowed := range allowOrigins {
			if allowed == "*" || allowed == origin {
				return true
			}
		}
		return false
	}

	return &APIChannel{
		BaseChannel: base,
		config:      cfg,
		upgrader: websocket.Upgrader{
			CheckOrigin:     checkOrigin,
			ReadBufferSize:  1024,
			WriteBufferSize: 4096, // Larger buffer for streaming
		},
	}, nil
}

// Start implements Channel.
func (c *APIChannel) Start(ctx context.Context) error {
	logger.InfoC("api", "Starting API channel")
	c.ctx, c.cancel = context.WithCancel(ctx)
	c.SetRunning(true)
	logger.InfoC("api", "API channel started")
	return nil
}

// Stop implements Channel.
func (c *APIChannel) Stop(ctx context.Context) error {
	logger.InfoC("api", "Stopping API channel")
	c.SetRunning(false)

	// Close all connections
	c.connections.Range(func(key, value any) bool {
		if ac, ok := value.(*apiConn); ok {
			ac.close()
		}
		c.connections.Delete(key)
		return true
	})

	if c.cancel != nil {
		c.cancel()
	}

	logger.InfoC("api", "API channel stopped")
	return nil
}

// WebhookPath implements channels.WebhookHandler.
func (c *APIChannel) WebhookPath() string { return "/api/" }

// ServeHTTP implements http.Handler for the shared HTTP server.
func (c *APIChannel) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api")

	switch {
	case path == "/ws" || path == "/ws/":
		c.handleWebSocket(w, r)
	case path == "/health" || path == "/health/":
		c.handleHealth(w, r)
	default:
		http.NotFound(w, r)
	}
}

// handleHealth returns the health status of the API channel.
func (c *APIChannel) handleHealth(w http.ResponseWriter, r *http.Request) {
	status := map[string]any{
		"status":       "ok",
		"running":      c.IsRunning(),
		"connections":  c.connCount.Load(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// Send implements Channel — sends a message to the appropriate WebSocket connection.
func (c *APIChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	outMsg := newMessage(TypeMessageCreate, map[string]any{
		"content": msg.Content,
	})

	return c.broadcastToSession(msg.ChatID, outMsg)
}

// EditMessage implements channels.MessageEditor.
func (c *APIChannel) EditMessage(ctx context.Context, chatID string, messageID string, content string) error {
	outMsg := newMessage(TypeMessageUpdate, map[string]any{
		"message_id": messageID,
		"content":    content,
	})
	return c.broadcastToSession(chatID, outMsg)
}

// StartTyping implements channels.TypingCapable.
func (c *APIChannel) StartTyping(ctx context.Context, chatID string) (func(), error) {
	startMsg := newMessage(TypeTypingStart, nil)
	if err := c.broadcastToSession(chatID, startMsg); err != nil {
		return func() {}, err
	}
	return func() {
		stopMsg := newMessage(TypeTypingStop, nil)
		c.broadcastToSession(chatID, stopMsg)
	}, nil
}

// SendPlaceholder implements channels.PlaceholderCapable.
func (c *APIChannel) SendPlaceholder(ctx context.Context, chatID string) (string, error) {
	if !c.config.Placeholder.Enabled {
		return "", nil
	}

	text := c.config.Placeholder.Text
	if text == "" {
		text = "Thinking... 💭"
	}

	msgID := uuid.New().String()
	outMsg := newMessage(TypeMessageCreate, map[string]any{
		"content":    text,
		"message_id": msgID,
	})

	if err := c.broadcastToSession(chatID, outMsg); err != nil {
		return "", err
	}

	return msgID, nil
}

// StreamChunk sends a streaming chunk to the client.
func (c *APIChannel) StreamChunk(chatID, messageID, chunk string) error {
	outMsg := newMessage(TypeMessageStream, map[string]any{
		"message_id": messageID,
		"chunk":      chunk,
	})
	return c.broadcastToSession(chatID, outMsg)
}

// StreamDone signals the end of streaming.
func (c *APIChannel) StreamDone(chatID, messageID string) error {
	outMsg := newMessage(TypeMessageDone, map[string]any{
		"message_id": messageID,
	})
	return c.broadcastToSession(chatID, outMsg)
}

// broadcastToSession sends a message to all connections with a matching session.
func (c *APIChannel) broadcastToSession(chatID string, msg APIMessage) error {
	// chatID format: "api:<sessionID>"
	sessionID := strings.TrimPrefix(chatID, "api:")
	msg.SessionID = sessionID

	var sent bool
	c.connections.Range(func(key, value any) bool {
		ac, ok := value.(*apiConn)
		if !ok {
			return true
		}
		if ac.sessionID == sessionID {
			if err := ac.writeJSON(msg); err != nil {
				logger.DebugCF("api", "Write to connection failed", map[string]any{
					"conn_id": ac.id,
					"error":   err.Error(),
				})
			} else {
				sent = true
			}
		}
		return true
	})

	if !sent {
		return fmt.Errorf("no active connections for session %s: %w", sessionID, channels.ErrSendFailed)
	}
	return nil
}

// handleWebSocket upgrades the HTTP connection and manages the WebSocket lifecycle.
func (c *APIChannel) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	if !c.IsRunning() {
		http.Error(w, "channel not running", http.StatusServiceUnavailable)
		return
	}

	// Authenticate
	if !c.authenticate(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Check connection limit
	maxConns := c.config.MaxConnections
	if maxConns <= 0 {
		maxConns = 100
	}
	if int(c.connCount.Load()) >= maxConns {
		http.Error(w, "too many connections", http.StatusServiceUnavailable)
		return
	}

	conn, err := c.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.ErrorCF("api", "WebSocket upgrade failed", map[string]any{
			"error": err.Error(),
		})
		return
	}

	// Determine session ID from query param or generate one
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	ac := &apiConn{
		id:        uuid.New().String(),
		conn:      conn,
		sessionID: sessionID,
	}

	c.connections.Store(ac.id, ac)
	c.connCount.Add(1)

	logger.InfoCF("api", "WebSocket client connected", map[string]any{
		"conn_id":    ac.id,
		"session_id": sessionID,
	})

	// Send connected message to client
	connectedMsg := newMessage(TypeConnected, map[string]any{
		"session_id": sessionID,
	})
	if err := ac.writeJSON(connectedMsg); err != nil {
		logger.ErrorCF("api", "Failed to send connected message", map[string]any{
			"error": err.Error(),
		})
		ac.close()
		c.connections.Delete(ac.id)
		c.connCount.Add(-1)
		return
	}

	go c.readLoop(ac)
}

// authenticate checks the Bearer token from the Authorization header.
func (c *APIChannel) authenticate(r *http.Request) bool {
	token := c.config.Token
	if token == "" {
		return false
	}

	// Check Authorization header
	auth := r.Header.Get("Authorization")
	if after, ok := strings.CutPrefix(auth, "Bearer "); ok {
		if after == token {
			return true
		}
	}

	// Check query parameter only when explicitly allowed
	if c.config.AllowTokenQuery {
		if r.URL.Query().Get("token") == token {
			return true
		}
	}

	return false
}

// readLoop reads messages from a WebSocket connection.
func (c *APIChannel) readLoop(ac *apiConn) {
	defer func() {
		ac.close()
		c.connections.Delete(ac.id)
		c.connCount.Add(-1)
		logger.InfoCF("api", "WebSocket client disconnected", map[string]any{
			"conn_id":    ac.id,
			"session_id": ac.sessionID,
		})
	}()

	readTimeout := time.Duration(c.config.ReadTimeout) * time.Second
	if readTimeout <= 0 {
		readTimeout = 60 * time.Second
	}

	_ = ac.conn.SetReadDeadline(time.Now().Add(readTimeout))
	ac.conn.SetPongHandler(func(appData string) error {
		_ = ac.conn.SetReadDeadline(time.Now().Add(readTimeout))
		return nil
	})

	// Start ping ticker
	pingInterval := time.Duration(c.config.PingInterval) * time.Second
	if pingInterval <= 0 {
		pingInterval = 30 * time.Second
	}
	go c.pingLoop(ac, pingInterval)

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		_, rawMsg, err := ac.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				logger.DebugCF("api", "WebSocket read error", map[string]any{
					"conn_id": ac.id,
					"error":   err.Error(),
				})
			}
			return
		}

		_ = ac.conn.SetReadDeadline(time.Now().Add(readTimeout))

		var msg APIMessage
		if err := json.Unmarshal(rawMsg, &msg); err != nil {
			errMsg := newError("invalid_message", "failed to parse message")
			ac.writeJSON(errMsg)
			continue
		}

		c.handleMessage(ac, msg)
	}
}

// pingLoop sends periodic ping frames to keep the connection alive.
func (c *APIChannel) pingLoop(ac *apiConn, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			if ac.closed.Load() {
				return
			}
			ac.writeMu.Lock()
			err := ac.conn.WriteMessage(websocket.PingMessage, nil)
			ac.writeMu.Unlock()
			if err != nil {
				return
			}
		}
	}
}

// handleMessage processes an inbound API Protocol message.
func (c *APIChannel) handleMessage(ac *apiConn, msg APIMessage) {
	switch msg.Type {
	case TypePing:
		pong := newMessage(TypePong, nil)
		pong.ID = msg.ID
		ac.writeJSON(pong)

	case TypeMessageSend:
		c.handleMessageSend(ac, msg)

	default:
		errMsg := newError("unknown_type", fmt.Sprintf("unknown message type: %s", msg.Type))
		ac.writeJSON(errMsg)
	}
}

// handleMessageSend processes an inbound message.send from a client.
func (c *APIChannel) handleMessageSend(ac *apiConn, msg APIMessage) {
	content, _ := msg.Payload["content"].(string)
	if strings.TrimSpace(content) == "" {
		errMsg := newError("empty_content", "message content is empty")
		ac.writeJSON(errMsg)
		return
	}

	sessionID := msg.SessionID
	if sessionID == "" {
		sessionID = ac.sessionID
	}

	chatID := "api:" + sessionID
	senderID := "api-user"

	peer := bus.Peer{Kind: "direct", ID: "api:" + sessionID}

	metadata := map[string]string{
		"platform":   "api",
		"session_id": sessionID,
		"conn_id":    ac.id,
	}

	logger.DebugCF("api", "Received message", map[string]any{
		"session_id": sessionID,
		"preview":    truncate(content, 50),
	})

	sender := bus.SenderInfo{
		Platform:    "api",
		PlatformID:  senderID,
		CanonicalID: identity.BuildCanonicalID("api", senderID),
	}

	if !c.IsAllowedSender(sender) {
		return
	}

	c.HandleMessage(c.ctx, peer, msg.ID, senderID, chatID, content, nil, metadata, sender)
}

// truncate truncates a string to maxLen runes.
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}