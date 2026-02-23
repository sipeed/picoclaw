// PicoClaw - Ultra-lightweight personal AI agent
// WebSocket channel implementation for local LAN chat

package websocket

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net"
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

// clientConn wraps a WebSocket connection with a write mutex for safe concurrent writes
type clientConn struct {
	conn    *websocket.Conn
	writeMu sync.Mutex
}

// writeMessage safely writes a message to the WebSocket connection
func (c *clientConn) writeMessage(messageType int, data []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.conn.WriteMessage(messageType, data)
}

// writeControl safely writes a control message to the WebSocket connection
func (c *clientConn) writeControl(messageType int, data []byte, deadline time.Time) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.conn.WriteControl(messageType, data, deadline)
}

// close closes the underlying WebSocket connection
func (c *clientConn) close() error {
	return c.conn.Close()
}

// Channel implements the Channel interface for WebSocket connections
type Channel struct {
	config    config.WebSocketConfig
	bus       *bus.MessageBus
	running   bool
	allowList []string
	token     string // Authentication token (optional)
	server    *http.Server
	upgrader  websocket.Upgrader
	clients   sync.Map // map[string]*clientConn
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

	// Validate allow_from configuration at startup
	if err := validateAllowList(cfg.AllowFrom); err != nil {
		return nil, fmt.Errorf("invalid allow_from configuration: %w", err)
	}

	return &Channel{
		config:    cfg,
		bus:       messageBus,
		allowList: cfg.AllowFrom,
		token:     strings.TrimSpace(cfg.Token),
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

// validateAllowList validates the allow_from configuration at startup
// This prevents silent failures during runtime when invalid rules are encountered
func validateAllowList(allowList []string) error {
	if len(allowList) == 0 {
		// Empty list is valid (allows all)
		return nil
	}

	var invalidRules []string
	validRuleCount := 0

	for _, allowed := range allowList {
		allowed = strings.TrimSpace(allowed)
		if allowed == "" {
			continue
		}

		// Check if it's a CIDR notation
		if strings.Contains(allowed, "/") {
			if _, _, err := net.ParseCIDR(allowed); err != nil {
				invalidRules = append(invalidRules, fmt.Sprintf("%s (CIDR parse error: %v)", allowed, err))
				continue
			}
			validRuleCount++
		} else {
			// Validate as plain IP address
			// Strip zone identifier for validation
			ipStr := allowed
			if idx := strings.IndexByte(ipStr, '%'); idx != -1 {
				ipStr = ipStr[:idx]
			}
			if net.ParseIP(ipStr) == nil {
				invalidRules = append(invalidRules, fmt.Sprintf("%s (invalid IP address)", allowed))
				continue
			}
			validRuleCount++
		}
	}

	// If there are invalid rules, return error with details
	if len(invalidRules) > 0 {
		return fmt.Errorf("found %d invalid rule(s) in allow_from: %s",
			len(invalidRules), strings.Join(invalidRules, "; "))
	}

	// All rules are valid
	return nil
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

// IsAllowed checks if a client IP is allowed to use this channel
// The clientID should be in the format "ip:port" (from r.RemoteAddr)
// Supports:
//   - Exact IP match: "192.168.1.5"
//   - CIDR notation: "192.168.1.0/24", "10.0.0.0/8"
//   - IPv6: "::1", "fe80::/10"
func (c *Channel) IsAllowed(clientID string) bool {
	if len(c.allowList) == 0 {
		return true
	}

	// Extract IP address from "ip:port" format
	clientIPStr := c.extractIP(clientID)
	if clientIPStr == "" {
		logger.WarnCF("websocket", "Failed to extract IP from client ID", map[string]any{
			"client_id": clientID,
		})
		return false
	}

	// Parse client IP (strip zone identifier for IPv6 link-local addresses)
	clientIPStr = c.stripZone(clientIPStr)
	clientIP := net.ParseIP(clientIPStr)
	if clientIP == nil {
		logger.WarnCF("websocket", "Invalid client IP address", map[string]any{
			"client_ip": clientIPStr,
		})
		return false
	}

	// Check against allow list
	for _, allowed := range c.allowList {
		// Try CIDR notation first
		if strings.Contains(allowed, "/") {
			_, ipNet, err := net.ParseCIDR(allowed)
			if err != nil {
				// This should never happen as we validate at startup
				// But keep for safety
				logger.ErrorCF("websocket", "Invalid CIDR in allow list (should be caught at startup)", map[string]any{
					"cidr":  allowed,
					"error": err.Error(),
				})
				continue
			}
			if ipNet.Contains(clientIP) {
				return true
			}
		} else {
			// Exact IP match (strip zone for comparison)
			allowedStripped := c.stripZone(allowed)
			if strings.EqualFold(allowedStripped, clientIPStr) {
				return true
			}
		}
	}
	return false
}

// extractIP extracts the IP address from "ip:port" format
// Handles:
//   - "192.168.1.5:8080" -> "192.168.1.5"
//   - "[::1]:8080" -> "::1"
//   - "[::1]" -> "::1" (brackets without port)
//   - "::1" -> "::1" (no brackets, no port)
func (c *Channel) extractIP(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		// Handle IPv6 addresses with brackets but no port: [::1] -> ::1
		if strings.HasPrefix(addr, "[") && strings.HasSuffix(addr, "]") {
			return addr[1 : len(addr)-1]
		}
		// Otherwise assume it's already just an IP (no port)
		return addr
	}
	return host
}

// stripZone removes the zone identifier from IPv6 addresses
// For example: "fe80::1%eth0" -> "fe80::1"
// This is necessary because net.ParseIP doesn't handle zone identifiers
func (c *Channel) stripZone(ipStr string) string {
	if idx := strings.IndexByte(ipStr, '%'); idx != -1 {
		return ipStr[:idx]
	}
	return ipStr
}

// setRunning sets the running state
func (c *Channel) setRunning(running bool) {
	c.clientsMu.Lock()
	defer c.clientsMu.Unlock()
	c.running = running
}

// HandleMessage processes an incoming message and publishes it to the bus
// Note: Authorization is already checked during WebSocket connection establishment
func (c *Channel) HandleMessage(senderID, chatID, content string, media []string, metadata map[string]string) {
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
		if client, ok := value.(*clientConn); ok {
			client.close()
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
			if client, ok := conn.(*clientConn); ok {
				err := client.writeMessage(websocket.TextMessage, data)
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
		if client, ok := value.(*clientConn); ok {
			if err := client.writeMessage(websocket.TextMessage, data); err != nil {
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

// validateToken validates the authentication token from the request
// Supports multiple token sources:
//  1. Query parameter: ?token=xxx
//  2. Authorization header: Bearer xxx
//  3. Sec-WebSocket-Protocol header: token (WebSocket standard approach)
func (c *Channel) validateToken(r *http.Request) bool {
	if c.token == "" {
		return true // No token configured, allow
	}

	// Method 1: Check query parameter
	if tokenParam := r.URL.Query().Get("token"); tokenParam != "" {
		return tokenParam == c.token
	}

	// Method 2: Check Authorization header (Bearer token)
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		if strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			return token == c.token
		}
	}

	// Method 3: Check Sec-WebSocket-Protocol header
	// Client can send: Sec-WebSocket-Protocol: token
	// This is a standard WebSocket approach for authentication
	protocol := r.Header.Get("Sec-WebSocket-Protocol")
	if protocol != "" {
		// Support both "token" format and "bearer-<token>" format
		if protocol == c.token {
			return true
		}
		if strings.HasPrefix(protocol, "bearer-") {
			token := strings.TrimPrefix(protocol, "bearer-")
			return token == c.token
		}
	}

	return false
}

// handleWebSocket handles WebSocket connection upgrades
func (c *Channel) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Check token authentication if configured
	if c.token != "" {
		if !c.validateToken(r) {
			logger.WarnCF("websocket", "Unauthorized connection attempt - invalid token", map[string]any{
				"remote_addr": r.RemoteAddr,
				"user_agent":  r.UserAgent(),
			})
			http.Error(w, "Unauthorized: invalid or missing token", http.StatusUnauthorized)
			return
		}
	}

	// Check IP authorization
	clientID := r.RemoteAddr
	if !c.IsAllowed(clientID) {
		clientIP := c.extractIP(clientID)
		logger.WarnCF("websocket", "Unauthorized connection attempt - IP not allowed", map[string]any{
			"client_ip": clientIP,
			"client_id": clientID,
		})
		http.Error(w, "Forbidden: IP not allowed", http.StatusForbidden)
		return
	}

	conn, err := c.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.ErrorCF("websocket", "Failed to upgrade connection", map[string]any{
			"error": err.Error(),
		})
		return
	}

	client := &clientConn{conn: conn}
	c.clients.Store(clientID, client)

	logger.InfoCF("websocket", "New client connected", map[string]any{
		"client_id": clientID,
		"client_ip": c.extractIP(clientID),
	})

	// Handle client messages
	go c.handleClient(clientID, client)
}

// handleClient processes messages from a WebSocket client
func (c *Channel) handleClient(clientID string, client *clientConn) {
	defer func() {
		client.close()
		c.clients.Delete(clientID)
		logger.InfoCF("websocket", "Client disconnected", map[string]any{
			"client_id": clientID,
		})
	}()

	// Set up ping/pong handlers for connection health check
	client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.conn.SetPongHandler(func(string) error {
		client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
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
			_, message, err := client.conn.ReadMessage()
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
			if err := client.writeControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
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
