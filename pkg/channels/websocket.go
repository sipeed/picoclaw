package channels

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sipeed/picoclaw/pkg/broadcast"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// wsIncoming is the JSON message sent from APK to picoclaw.
type wsIncoming struct {
	Content   string   `json:"content"`
	SenderID  string   `json:"sender_id,omitempty"`
	Images    []string `json:"images,omitempty"`
	InputMode string   `json:"input_mode,omitempty"`
}

// wsOutgoing is the JSON message sent from picoclaw to APK.
type wsOutgoing struct {
	Content string `json:"content"`
	Type    string `json:"type,omitempty"`
}

// WebSocketChannel is a server-side WebSocket channel that accepts
// connections from clients (e.g. a Google Assistant replacement APK).
type WebSocketChannel struct {
	*BaseChannel
	config      config.WebSocketConfig
	server      *http.Server
	upgrader    websocket.Upgrader
	clients     map[*websocket.Conn]string // conn → clientID
	chatConns   map[string]*websocket.Conn // chatID → conn
	clientTypes map[string]string          // chatID → clientType (retained after disconnect)
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
}

func NewWebSocketChannel(cfg config.WebSocketConfig, msgBus *bus.MessageBus) (*WebSocketChannel, error) {
	base := NewBaseChannel("websocket", cfg, msgBus, cfg.AllowFrom)

	return &WebSocketChannel{
		BaseChannel: base,
		config:      cfg,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		clients:     make(map[*websocket.Conn]string),
		chatConns:   make(map[string]*websocket.Conn),
		clientTypes: make(map[string]string),
	}, nil
}

func (c *WebSocketChannel) Start(ctx context.Context) error {
	logger.InfoC("websocket", "Starting WebSocket channel server")

	c.ctx, c.cancel = context.WithCancel(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc(c.config.Path, c.handleWS)

	addr := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)
	c.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	c.setRunning(true)

	logger.InfoCF("websocket", "WebSocket server listening", map[string]interface{}{
		"host": c.config.Host,
		"port": c.config.Port,
		"path": c.config.Path,
	})

	go func() {
		if err := c.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.ErrorCF("websocket", "Server error", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}()

	return nil
}

func (c *WebSocketChannel) Stop(ctx context.Context) error {
	logger.InfoC("websocket", "Stopping WebSocket channel")
	c.setRunning(false)

	if c.cancel != nil {
		c.cancel()
	}

	c.mu.Lock()
	for conn, clientID := range c.clients {
		logger.DebugCF("websocket", "Closing client connection", map[string]interface{}{
			"client_id": clientID,
		})
		conn.Close()
	}
	c.clients = make(map[*websocket.Conn]string)
	c.chatConns = make(map[string]*websocket.Conn)
	c.clientTypes = make(map[string]string)
	c.mu.Unlock()

	if c.server != nil {
		if err := c.server.Shutdown(ctx); err != nil {
			logger.ErrorCF("websocket", "Server shutdown error", map[string]interface{}{
				"error": err.Error(),
			})
			return err
		}
	}

	logger.InfoC("websocket", "WebSocket channel stopped")
	return nil
}

func (c *WebSocketChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return fmt.Errorf("websocket channel not running")
	}

	c.mu.RLock()
	conn, ok := c.chatConns[msg.ChatID]
	clientType := c.clientTypes[msg.ChatID] // Read under lock to avoid data race
	c.mu.RUnlock()

	if !ok {
		// Connection not found — try broadcast fallback for "main" clients.
		return c.maybeBroadcast(msg, clientType, fmt.Errorf("no connection for chat %s", msg.ChatID))
	}

	out := wsOutgoing{Content: msg.Content, Type: msg.Type}
	data, err := json.Marshal(out)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	// Verify connection still exists (may have been removed during cleanup).
	if _, exists := c.clients[conn]; !exists {
		return c.maybeBroadcast(msg, c.clientTypes[msg.ChatID], fmt.Errorf("connection for chat %s no longer active", msg.ChatID))
	}

	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		logger.ErrorCF("websocket", "Failed to send message", map[string]interface{}{
			"chat_id": msg.ChatID,
			"error":   err.Error(),
		})
		return c.maybeBroadcast(msg, c.clientTypes[msg.ChatID], err)
	}

	return nil
}

// maybeBroadcast sends a message via Android broadcast if the disconnected
// client is of type "main". Status/status_end messages are ephemeral and
// skipped. Returns the original error if broadcast is not applicable.
// clientType must be read under lock by the caller to avoid data races.
func (c *WebSocketChannel) maybeBroadcast(msg bus.OutboundMessage, clientType string, originalErr error) error {
	// Status messages are ephemeral — don't broadcast.
	if msg.Type == "status" || msg.Type == "status_end" {
		return originalErr
	}

	// Only broadcast for "main" type clients.
	if clientType != "main" {
		return originalErr
	}

	logger.InfoCF("websocket", "Using broadcast fallback for disconnected main client", map[string]interface{}{
		"chat_id":     msg.ChatID,
		"content_len": len(msg.Content),
	})

	if err := broadcast.Send(broadcast.Message{
		Content: msg.Content,
		Type:    msg.Type,
	}); err != nil {
		return fmt.Errorf("ws send failed and broadcast fallback also failed: %w", err)
	}
	return nil
}

func (c *WebSocketChannel) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := c.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.ErrorCF("websocket", "Upgrade failed", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	clientID := r.URL.Query().Get("client_id")
	if clientID == "" {
		clientID = uuid.New().String()
	}

	clientType := r.URL.Query().Get("client_type")
	if clientType == "" {
		clientType = "main" // Default for backward compatibility
	}

	logger.InfoCF("websocket", "New WebSocket connection", map[string]interface{}{
		"client_id":   clientID,
		"client_type": clientType,
		"remote_addr": r.RemoteAddr,
	})

	chatID := fmt.Sprintf("ws:%s", clientID)

	c.mu.Lock()
	if oldConn, ok := c.chatConns[chatID]; ok {
		delete(c.clients, oldConn)
		oldConn.Close()
	}
	c.clients[conn] = clientID
	c.chatConns[chatID] = conn
	c.clientTypes[chatID] = clientType
	c.mu.Unlock()

	go c.readPump(conn, clientID, chatID, clientType)
}

// GetClientType returns the client type for a given chatID.
// Returns empty string if unknown. The value is retained after disconnect
// so that the broadcast fallback can check the type.
func (c *WebSocketChannel) GetClientType(chatID string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.clientTypes[chatID]
}

func (c *WebSocketChannel) readPump(conn *websocket.Conn, clientID, chatID, clientType string) {
	defer func() {
		c.mu.Lock()
		delete(c.clients, conn)
		delete(c.chatConns, chatID)
		c.mu.Unlock()
		conn.Close()

		logger.InfoCF("websocket", "Client disconnected", map[string]interface{}{
			"client_id": clientID,
		})
	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				logger.ErrorCF("websocket", "Read error", map[string]interface{}{
					"client_id": clientID,
					"error":     err.Error(),
				})
			}
			return
		}

		var incoming wsIncoming
		if err := json.Unmarshal(message, &incoming); err != nil {
			logger.ErrorCF("websocket", "Invalid JSON message", map[string]interface{}{
				"client_id": clientID,
				"error":     err.Error(),
			})
			continue
		}

		// Use sender_id from message if provided, otherwise use clientID.
		senderID := clientID
		if incoming.SenderID != "" {
			senderID = incoming.SenderID
		}

		content := incoming.Content
		var media []string

		// Convert base64 images directly to data URLs.
		for _, imgData := range incoming.Images {
			dataURL := "data:image/png;base64," + imgData
			media = append(media, dataURL)
		}

		logger.DebugCF("websocket", "Received message", map[string]interface{}{
			"client_id": clientID,
			"content":   incoming.Content,
			"images":    len(incoming.Images),
		})

		inputMode := incoming.InputMode
		if inputMode == "" {
			inputMode = "text"
		}
		metadata := map[string]string{
			"input_mode":  inputMode,
			"client_type": clientType,
		}
		c.HandleMessage(senderID, chatID, content, media, metadata)
	}
}
