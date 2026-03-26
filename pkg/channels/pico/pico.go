package pico

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

// picoConn represents a single WebSocket connection.
type picoConn struct {
	id        string
	conn      *websocket.Conn
	sessionID string
	writeMu   sync.Mutex
	closed    atomic.Bool
	cancel    context.CancelFunc // cancels per-connection goroutines (e.g. pingLoop)
}

// writeJSON sends a JSON message to the connection with write locking.
func (pc *picoConn) writeJSON(v any) error {
	if pc.closed.Load() {
		return fmt.Errorf("connection closed")
	}
	pc.writeMu.Lock()
	defer pc.writeMu.Unlock()
	return pc.conn.WriteJSON(v)
}

// close closes the connection.
func (pc *picoConn) close() {
	if pc.closed.CompareAndSwap(false, true) {
		if pc.cancel != nil {
			pc.cancel()
		}
		pc.conn.Close()
	}
}

// PicoChannel implements the native Pico Protocol WebSocket channel.
// It serves as the reference implementation for all optional capability interfaces.
type PicoChannel struct {
	*channels.BaseChannel
	config             config.PicoConfig
	upgrader           websocket.Upgrader
	connections        map[string]*picoConn            // connID -> *picoConn
	sessionConnections map[string]map[string]*picoConn // sessionID -> connID -> *picoConn
	connsMu            sync.RWMutex
	ctx                context.Context
	cancel             context.CancelFunc
}

// NewPicoChannel creates a new Pico Protocol channel.
func NewPicoChannel(cfg config.PicoConfig, messageBus *bus.MessageBus) (*PicoChannel, error) {
	if cfg.Token() == "" {
		return nil, fmt.Errorf("pico token is required")
	}

	base := channels.NewBaseChannel("pico", cfg, messageBus, cfg.AllowFrom)

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

	return &PicoChannel{
		BaseChannel: base,
		config:      cfg,
		upgrader: websocket.Upgrader{
			CheckOrigin:     checkOrigin,
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		connections:        make(map[string]*picoConn),
		sessionConnections: make(map[string]map[string]*picoConn),
	}, nil
}

// createAndAddConnection checks MaxConnections and registers a connection atomically.
func (c *PicoChannel) createAndAddConnection(conn *websocket.Conn, sessionID string, maxConns int) (*picoConn, error) {
	c.connsMu.Lock()
	defer c.connsMu.Unlock()
	if len(c.connections) >= maxConns {
		return nil, channels.ErrTemporary
	}

	var connID string
	for {
		connID = uuid.New().String()
		if _, exists := c.connections[connID]; !exists {
			break
		}
	}

	pc := &picoConn{
		id:        connID,
		conn:      conn,
		sessionID: sessionID,
	}

	c.connections[pc.id] = pc
	bySession, ok := c.sessionConnections[pc.sessionID]
	if !ok {
		bySession = make(map[string]*picoConn)
		c.sessionConnections[pc.sessionID] = bySession
	}
	bySession[pc.id] = pc

	return pc, nil
}

// removeConnection deletes a connection from indexes and returns it when found.
func (c *PicoChannel) removeConnection(connID string) *picoConn {
	c.connsMu.Lock()
	defer c.connsMu.Unlock()

	pc, ok := c.connections[connID]
	if !ok {
		return nil
	}

	delete(c.connections, connID)
	if bySession, ok := c.sessionConnections[pc.sessionID]; ok {
		delete(bySession, connID)
		if len(bySession) == 0 {
			delete(c.sessionConnections, pc.sessionID)
		}
	}

	return pc
}

// takeAllConnections snapshots and clears all connection indexes.
func (c *PicoChannel) takeAllConnections() []*picoConn {
	c.connsMu.Lock()
	defer c.connsMu.Unlock()

	all := make([]*picoConn, 0, len(c.connections))
	for _, pc := range c.connections {
		all = append(all, pc)
	}
	clear(c.connections)
	clear(c.sessionConnections)

	return all
}

// sessionConnectionsSnapshot returns all active connections for a session.
func (c *PicoChannel) sessionConnectionsSnapshot(sessionID string) []*picoConn {
	c.connsMu.RLock()
	defer c.connsMu.RUnlock()

	bySession, ok := c.sessionConnections[sessionID]
	if !ok || len(bySession) == 0 {
		return nil
	}

	conns := make([]*picoConn, 0, len(bySession))
	for _, pc := range bySession {
		conns = append(conns, pc)
	}
	return conns
}

// currentConnCount returns a lock-protected snapshot of active connection count.
func (c *PicoChannel) currentConnCount() int {
	c.connsMu.RLock()
	defer c.connsMu.RUnlock()
	return len(c.connections)
}

// Start implements Channel.
func (c *PicoChannel) Start(ctx context.Context) error {
	logger.InfoC("pico", "Starting Pico Protocol channel")
	c.ctx, c.cancel = context.WithCancel(ctx)
	c.SetRunning(true)
	logger.InfoC("pico", "Pico Protocol channel started")
	return nil
}

// Stop implements Channel.
func (c *PicoChannel) Stop(ctx context.Context) error {
	logger.InfoC("pico", "Stopping Pico Protocol channel")
	c.SetRunning(false)

	// Close all connections
	for _, pc := range c.takeAllConnections() {
		pc.close()
	}

	if c.cancel != nil {
		c.cancel()
	}

	logger.InfoC("pico", "Pico Protocol channel stopped")
	return nil
}

// WebhookPath implements channels.WebhookHandler.
func (c *PicoChannel) WebhookPath() string { return "/pico/" }

// ServeHTTP implements http.Handler for the shared HTTP server.
func (c *PicoChannel) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/pico")

	switch path {
	case "/ws", "/ws/":
		c.handleWebSocket(w, r)
	default:
		http.NotFound(w, r)
	}
}

// Send implements Channel — sends a message to the appropriate WebSocket connection.
func (c *PicoChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	outMsg := newMessage(TypeMessageCreate, map[string]any{
		"content": msg.Content,
	})

	return c.broadcastToSession(msg.ChatID, outMsg)
}

// EditMessage implements channels.MessageEditor.
func (c *PicoChannel) EditMessage(ctx context.Context, chatID string, messageID string, content string) error {
	outMsg := newMessage(TypeMessageUpdate, map[string]any{
		"message_id": messageID,
		"content":    content,
	})
	return c.broadcastToSession(chatID, outMsg)
}

// StartTyping implements channels.TypingCapable.
func (c *PicoChannel) StartTyping(ctx context.Context, chatID string) (func(), error) {
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
// It sends a placeholder message via the Pico Protocol that will later be
// edited to the actual response via EditMessage (channels.MessageEditor).
func (c *PicoChannel) SendPlaceholder(ctx context.Context, chatID string) (string, error) {
	if !c.config.Placeholder.Enabled {
		return "", nil
	}

	text := c.config.Placeholder.GetRandomText()

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

// broadcastToSession sends a message to all connections with a matching session.
func (c *PicoChannel) broadcastToSession(chatID string, msg PicoMessage) error {
	// chatID format: "pico:<sessionID>"
	sessionID := strings.TrimPrefix(chatID, "pico:")
	msg.SessionID = sessionID

	var sent bool
	for _, pc := range c.sessionConnectionsSnapshot(sessionID) {
		if err := pc.writeJSON(msg); err != nil {
			logger.DebugCF("pico", "Write to connection failed", map[string]any{
				"conn_id": pc.id,
				"error":   err.Error(),
			})
		} else {
			sent = true
		}
	}

	if !sent {
		return fmt.Errorf("no active connections for session %s: %w", sessionID, channels.ErrSendFailed)
	}
	return nil
}

// handleWebSocket upgrades the HTTP connection and manages the WebSocket lifecycle.
func (c *PicoChannel) handleWebSocket(w http.ResponseWriter, r *http.Request) {
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
	if c.currentConnCount() >= maxConns {
		http.Error(w, "too many connections", http.StatusServiceUnavailable)
		return
	}

	// Echo the matched subprotocol back so the browser accepts the upgrade.
	var responseHeader http.Header
	if proto := c.matchedSubprotocol(r); proto != "" {
		responseHeader = http.Header{"Sec-WebSocket-Protocol": {proto}}
	}

	conn, err := c.upgrader.Upgrade(w, r, responseHeader)
	if err != nil {
		logger.ErrorCF("pico", "WebSocket upgrade failed", map[string]any{
			"error": err.Error(),
		})
		return
	}

	// Determine session ID from query param or generate one
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	pc, err := c.createAndAddConnection(conn, sessionID, maxConns)
	if err != nil {
		_ = conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseTryAgainLater, "too many connections"),
			time.Now().Add(2*time.Second),
		)
		_ = conn.Close()
		return
	}

	logger.InfoCF("pico", "WebSocket client connected", map[string]any{
		"conn_id":    pc.id,
		"session_id": sessionID,
	})

	go c.readLoop(pc)
}

// authenticate checks the request for a valid token:
//  1. Authorization: Bearer <token> header
//  2. Sec-WebSocket-Protocol "token.<value>" (for browsers that can't set headers)
//  3. Query parameter "token" (only when AllowTokenQuery is on)
func (c *PicoChannel) authenticate(r *http.Request) bool {
	token := c.config.Token()
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

	// Check Sec-WebSocket-Protocol subprotocol ("token.<value>")
	if c.matchedSubprotocol(r) != "" {
		return true
	}

	// Check query parameter only when explicitly allowed
	if c.config.AllowTokenQuery {
		if r.URL.Query().Get("token") == token {
			return true
		}
	}

	return false
}

// matchedSubprotocol returns the "token.<value>" subprotocol that matches
// the configured token, or "" if none do.
func (c *PicoChannel) matchedSubprotocol(r *http.Request) string {
	token := c.config.Token()
	for _, proto := range websocket.Subprotocols(r) {
		if after, ok := strings.CutPrefix(proto, "token."); ok && after == token {
			return proto
		}
	}
	return ""
}

// readLoop reads messages from a WebSocket connection.
func (c *PicoChannel) readLoop(pc *picoConn) {
	defer func() {
		pc.close()
		if removed := c.removeConnection(pc.id); removed != nil {
			logger.InfoCF("pico", "WebSocket client disconnected", map[string]any{
				"conn_id":    removed.id,
				"session_id": removed.sessionID,
			})
		}
	}()

	readTimeout := time.Duration(c.config.ReadTimeout) * time.Second
	if readTimeout <= 0 {
		readTimeout = 60 * time.Second
	}

	_ = pc.conn.SetReadDeadline(time.Now().Add(readTimeout))
	pc.conn.SetPongHandler(func(appData string) error {
		_ = pc.conn.SetReadDeadline(time.Now().Add(readTimeout))
		return nil
	})

	// Start ping ticker
	pingInterval := time.Duration(c.config.PingInterval) * time.Second
	if pingInterval <= 0 {
		pingInterval = 30 * time.Second
	}
	go c.pingLoop(pc, pingInterval)

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		_, rawMsg, err := pc.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				logger.DebugCF("pico", "WebSocket read error", map[string]any{
					"conn_id": pc.id,
					"error":   err.Error(),
				})
			}
			return
		}

		_ = pc.conn.SetReadDeadline(time.Now().Add(readTimeout))

		var msg PicoMessage
		if err := json.Unmarshal(rawMsg, &msg); err != nil {
			errMsg := newError("invalid_message", "failed to parse message")
			pc.writeJSON(errMsg)
			continue
		}

		c.handleMessage(pc, msg)
	}
}

// pingLoop sends periodic ping frames to keep the connection alive.
func (c *PicoChannel) pingLoop(pc *picoConn, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
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

// handleMessage processes an inbound Pico Protocol message.
func (c *PicoChannel) handleMessage(pc *picoConn, msg PicoMessage) {
	switch msg.Type {
	case TypePing:
		pong := newMessage(TypePong, nil)
		pong.ID = msg.ID
		pc.writeJSON(pong)

	case TypeMessageSend:
		c.handleMessageSend(pc, msg)

	default:
		errMsg := newError("unknown_type", fmt.Sprintf("unknown message type: %s", msg.Type))
		pc.writeJSON(errMsg)
	}
}

// handleMessageSend processes an inbound message.send from a client.
func (c *PicoChannel) handleMessageSend(pc *picoConn, msg PicoMessage) {
	content, _ := msg.Payload["content"].(string)
	if strings.TrimSpace(content) == "" {
		errMsg := newError("empty_content", "message content is empty")
		pc.writeJSON(errMsg)
		return
	}

	sessionID := msg.SessionID
	if sessionID == "" {
		sessionID = pc.sessionID
	}

	chatID := "pico:" + sessionID
	senderID := "pico-user"

	peer := bus.Peer{Kind: "direct", ID: "pico:" + sessionID}

	metadata := map[string]string{
		"platform":   "pico",
		"session_id": sessionID,
		"conn_id":    pc.id,
	}

	logger.DebugCF("pico", "Received message", map[string]any{
		"session_id": sessionID,
		"preview":    truncate(content, 50),
	})

	sender := bus.SenderInfo{
		Platform:    "pico",
		PlatformID:  senderID,
		CanonicalID: identity.BuildCanonicalID("pico", senderID),
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
