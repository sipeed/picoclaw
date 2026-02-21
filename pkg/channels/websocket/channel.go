// PicoClaw - Ultra-lightweight personal AI agent
// WebSocket channel implementation for local LAN chat

package websocket

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

//go:embed chat.html
var chatHTML []byte

//go:embed logo.jpg
var logoImage []byte

// Channel implements the Channel interface for WebSocket connections
type Channel struct {
	config    config.WebSocketConfig
	bus       *bus.MessageBus
	running   bool
	allowList []string
	server    *http.Server
	upgrader  websocket.Upgrader
	clients   sync.Map // map[string]*websocket.Conn
	clientsMu sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

// WebSocketMessage represents the JSON message format
type WebSocketMessage struct {
	Type      string `json:"type"`      // "chat", "status", "error", "system"
	Content   string `json:"content"`   // Message content
	Sender    string `json:"sender"`    // Sender ID
	Timestamp int64  `json:"timestamp"` // Unix timestamp in milliseconds
	SessionID string `json:"session_id,omitempty"`
}

// NewChannel creates a new WebSocket channel instance
func NewChannel(cfg config.WebSocketConfig, messageBus *bus.MessageBus) (*Channel, error) {
	if cfg.Port == 0 {
		cfg.Port = 8080
	}
	if cfg.Host == "" {
		cfg.Host = "0.0.0.0"
	}

	return &Channel{
		config:    cfg,
		bus:       messageBus,
		allowList: cfg.AllowFrom,
		running:   false,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins for local LAN usage
				// For production, you should implement proper origin checking
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}, nil
}

// Name returns the channel name
func (c *Channel) Name() string {
	return "websocket"
}

// IsRunning returns whether the channel is currently running
func (c *Channel) IsRunning() bool {
	c.clientsMu.RLock()
	defer c.clientsMu.RUnlock()
	return c.running
}

// IsAllowed checks if a sender ID is allowed to use this channel
func (c *Channel) IsAllowed(senderID string) bool {
	if len(c.allowList) == 0 {
		return true
	}
	for _, allowed := range c.allowList {
		if strings.EqualFold(allowed, senderID) {
			return true
		}
	}
	return false
}

// setRunning sets the running state
func (c *Channel) setRunning(running bool) {
	c.clientsMu.Lock()
	defer c.clientsMu.Unlock()
	c.running = running
}

// HandleMessage processes an incoming message and publishes it to the bus
func (c *Channel) HandleMessage(senderID, chatID, content string, media []string, metadata map[string]string) {
	if !c.IsAllowed(senderID) {
		return
	}

	// Build session key: channel:chatID
	sessionKey := fmt.Sprintf("%s:%s", c.Name(), chatID)

	msg := bus.InboundMessage{
		Channel:    c.Name(),
		SenderID:   senderID,
		ChatID:     chatID,
		Content:    content,
		Media:      media,
		SessionKey: sessionKey,
		Metadata:   metadata,
	}

	c.bus.PublishInbound(msg)
}

// Start initializes and starts the WebSocket server
func (c *Channel) Start(ctx context.Context) error {
	c.ctx, c.cancel = context.WithCancel(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", c.handleWebSocket)
	mux.HandleFunc("/", c.handleIndex)
	mux.HandleFunc("/assets/logo.jpg", c.handleLogo)

	addr := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)
	c.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	c.setRunning(true)
	logger.InfoCF("websocket", "WebSocket channel starting", map[string]any{
		"address": addr,
	})

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		if err := c.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Check for immediate startup errors
	select {
	case err := <-errCh:
		c.setRunning(false)
		return fmt.Errorf("failed to start WebSocket server: %w", err)
	case <-time.After(100 * time.Millisecond):
		logger.InfoCF("websocket", "WebSocket channel started successfully", map[string]any{
			"address": addr,
		})
		return nil
	}
}

// Stop gracefully shuts down the WebSocket server
func (c *Channel) Stop(ctx context.Context) error {
	logger.InfoC("websocket", "Stopping WebSocket channel")

	if c.cancel != nil {
		c.cancel()
	}

	// Close all client connections
	c.clients.Range(func(key, value any) bool {
		if conn, ok := value.(*websocket.Conn); ok {
			conn.Close()
		}
		c.clients.Delete(key)
		return true
	})

	// Shutdown HTTP server
	if c.server != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := c.server.Shutdown(shutdownCtx); err != nil {
			logger.ErrorCF("websocket", "Error shutting down server", map[string]any{
				"error": err.Error(),
			})
		}
	}

	c.setRunning(false)
	logger.InfoC("websocket", "WebSocket channel stopped")
	return nil
}

// Send sends a message to the specified chat (client)
func (c *Channel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return fmt.Errorf("websocket channel not running")
	}

	wsMsg := WebSocketMessage{
		Type:      "chat",
		Content:   msg.Content,
		Sender:    "assistant",
		Timestamp: time.Now().UnixMilli(),
	}

	data, err := json.Marshal(wsMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Ensure UTF-8 validity
	if !json.Valid(data) {
		logger.ErrorCF("websocket", "Invalid JSON data", map[string]any{
			"content_preview": msg.Content[:min(len(msg.Content), 100)],
		})
		return fmt.Errorf("invalid JSON message")
	}

	// Send to specific client or broadcast
	if msg.ChatID != "" && msg.ChatID != "broadcast" {
		// Send to specific client
		if conn, ok := c.clients.Load(msg.ChatID); ok {
			if wsConn, ok := conn.(*websocket.Conn); ok {
				err := wsConn.WriteMessage(websocket.TextMessage, data)
				if err != nil {
					// Connection may be dead, clean it up
					logger.WarnCF("websocket", "Failed to send to client, removing connection", map[string]any{
						"client": msg.ChatID,
						"error":  err.Error(),
					})
					c.clients.Delete(msg.ChatID)
					return err
				}
				return nil
			}
		}
		return fmt.Errorf("client %s not found", msg.ChatID)
	}

	// Broadcast to all connected clients
	var lastErr error
	deadClients := make([]any, 0)
	c.clients.Range(func(key, value any) bool {
		if conn, ok := value.(*websocket.Conn); ok {
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				logger.WarnCF("websocket", "Failed to send to client", map[string]any{
					"client": key,
					"error":  err.Error(),
				})
				deadClients = append(deadClients, key)
				lastErr = err
			}
		}
		return true
	})

	// Clean up dead connections
	for _, key := range deadClients {
		c.clients.Delete(key)
		logger.InfoCF("websocket", "Removed dead client connection", map[string]any{
			"client": key,
		})
	}

	return lastErr
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// handleWebSocket handles WebSocket connection upgrades
func (c *Channel) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := c.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.ErrorCF("websocket", "Failed to upgrade connection", map[string]any{
			"error": err.Error(),
		})
		return
	}

	// Generate client ID from remote address
	clientID := r.RemoteAddr
	c.clients.Store(clientID, conn)

	logger.InfoCF("websocket", "New client connected", map[string]any{
		"client_id": clientID,
	})

	// Handle client messages
	go c.handleClient(clientID, conn)
}

// handleClient processes messages from a WebSocket client
func (c *Channel) handleClient(clientID string, conn *websocket.Conn) {
	defer func() {
		conn.Close()
		c.clients.Delete(clientID)
		logger.InfoCF("websocket", "Client disconnected", map[string]any{
			"client_id": clientID,
		})
	}()

	// Set up ping/pong handlers for connection health check
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Start ping ticker
	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	// Channel for messages
	messageChan := make(chan []byte, 10)
	defer close(messageChan)

	// Read messages in a goroutine
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					logger.ErrorCF("websocket", "WebSocket error", map[string]any{
						"client_id": clientID,
						"error":     err.Error(),
					})
				}
				return
			}
			messageChan <- message
		}
	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-pingTicker.C:
			// Send ping to check connection health
			if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
				logger.WarnCF("websocket", "Failed to send ping", map[string]any{
					"client_id": clientID,
					"error":     err.Error(),
				})
				return
			}
		case message, ok := <-messageChan:
			if !ok {
				return
			}

			var wsMsg WebSocketMessage
			if err := json.Unmarshal(message, &wsMsg); err != nil {
				logger.WarnCF("websocket", "Failed to parse message", map[string]any{
					"client_id": clientID,
					"error":     err.Error(),
				})
				continue
			}

			// Process chat messages
			if wsMsg.Type == "chat" {
				// Check allowlist
				if !c.IsAllowed(clientID) {
					logger.WarnCF("websocket", "Unauthorized client", map[string]any{
						"client_id": clientID,
					})
					continue
				}

				logger.DebugCF("websocket", "Received message", map[string]any{
					"client_id": clientID,
					"content":   wsMsg.Content,
				})

				// Send to agent via message bus
				c.HandleMessage(clientID, clientID, wsMsg.Content, nil, nil)
			}
		}
	}
}

// handleLogo serves the logo image
func (c *Channel) handleLogo(w http.ResponseWriter, r *http.Request) {
	// Use embedded logo image
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "public, max-age=86400") // Cache for 1 day
	w.Write(logoImage)
}

// handleIndex serves a simple HTML chat interface
func (c *Channel) handleIndex(w http.ResponseWriter, r *http.Request) {
	// Use embedded HTML file
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(chatHTML)
}
