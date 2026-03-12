package web

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// WebSocketConnection represents an active WebSocket connection for chat
type WebSocketConnection struct {
	conn     *websocket.Conn
	chatID   string
	channel  string
	mu       sync.Mutex
	done     chan struct{}
	server   *Server
	sendChan chan *WebSocketMessage
}

// WebSocketMessage represents a message sent over WebSocket
type WebSocketMessage struct {
	Type      string                 `json:"type"`      // "message", "response", "error", "status"
	Content   string                 `json:"content"`   // Message content
	ChatID    string                 `json:"chat_id"`
	Role      string                 `json:"role"`      // "user", "assistant"
	Error     string                 `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp int64                  `json:"timestamp"`
}

// WSConnManager manages active WebSocket connections
type WSConnManager struct {
	mu          sync.RWMutex
	connections map[string]*WebSocketConnection // chatID -> connection
}

// NewWSConnManager creates a new WebSocket connection manager
func NewWSConnManager() *WSConnManager {
	return &WSConnManager{
		connections: make(map[string]*WebSocketConnection),
	}
}

// Register registers a new WebSocket connection
func (wm *WSConnManager) Register(conn *WebSocketConnection) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.connections[conn.chatID] = conn
	logger.DebugCF("web.ws", "WebSocket connection registered", map[string]any{
		"chat_id": conn.chatID,
	})
}

// Unregister removes a WebSocket connection
func (wm *WSConnManager) Unregister(chatID string) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	delete(wm.connections, chatID)
	logger.DebugCF("web.ws", "WebSocket connection unregistered", map[string]any{
		"chat_id": chatID,
	})
}

// Send sends a message to a specific connection
func (wm *WSConnManager) Send(chatID string, msg *WebSocketMessage) error {
	wm.mu.RLock()
	conn, exists := wm.connections[chatID]
	wm.mu.RUnlock()

	if !exists {
		return nil // Connection already closed, ignore
	}

	select {
	case conn.sendChan <- msg:
		return nil
	case <-conn.done:
		return nil // Connection closed
	default:
		// Channel full, log and drop message
		logger.WarnCF("web.ws", "Send channel full, message dropped", map[string]any{
			"chat_id": chatID,
		})
		return nil
	}
}

// Broadcast sends a message to all connections (used for system messages)
func (wm *WSConnManager) Broadcast(msg *WebSocketMessage) {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	for _, conn := range wm.connections {
		select {
		case conn.sendChan <- msg:
		case <-conn.done:
		default:
		}
	}
}

// GetConnection returns a connection by chatID (for testing or special cases)
func (wm *WSConnManager) GetConnection(chatID string) *WebSocketConnection {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	return wm.connections[chatID]
}

// ConnectionCount returns the number of active connections
func (wm *WSConnManager) ConnectionCount() int {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	return len(wm.connections)
}

// NewWebSocketConnection creates a new WebSocket connection wrapper
func NewWebSocketConnection(wsConn *websocket.Conn, chatID string, server *Server) *WebSocketConnection {
	return &WebSocketConnection{
		conn:     wsConn,
		chatID:   chatID,
		channel:  "web",
		done:     make(chan struct{}),
		server:   server,
		sendChan: make(chan *WebSocketMessage, 100), // Increase buffer size
	}
}

// Read reads messages from the WebSocket connection
func (wc *WebSocketConnection) Read(ctx context.Context, handler func(*WebSocketMessage) error) error {
	wc.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	wc.conn.SetPongHandler(func(string) error {
		wc.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-wc.done:
			return nil
		default:
		}

		var msg WebSocketMessage
		err := wc.conn.ReadJSON(&msg)
		if err != nil {
			if !websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.DebugCF("web.ws", "WebSocket read error", map[string]any{
					"error":   err.Error(),
					"chat_id": wc.chatID,
				})
			}
			return err
		}

		// Set chatID if not provided
		if msg.ChatID == "" {
			msg.ChatID = wc.chatID
		}

		// Set timestamp if not provided
		if msg.Timestamp == 0 {
			msg.Timestamp = time.Now().Unix()
		}

		if err := handler(&msg); err != nil {
			logger.WarnCF("web.ws", "Handler error", map[string]any{
				"error":   err.Error(),
				"chat_id": wc.chatID,
			})
		}
	}
}

// Write writes messages to the WebSocket connection
func (wc *WebSocketConnection) Write(ctx context.Context) error {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-wc.done:
			return nil
		case msg := <-wc.sendChan:
			if msg == nil {
				return fmt.Errorf("nil message received")
			}
			wc.mu.Lock()
			err := wc.conn.WriteJSON(msg)
			wc.mu.Unlock()
			if err != nil {
				logger.DebugCF("web.ws", "WebSocket write error", map[string]any{
					"error":   err.Error(),
					"chat_id": wc.chatID,
				})
				return err
			}
			logger.DebugCF("web.ws", "Message sent via WebSocket", map[string]any{
				"chat_id":    wc.chatID,
				"type":       msg.Type,
				"role":       msg.Role,
				"content_len": len(msg.Content),
			})
		case <-ticker.C:
			wc.mu.Lock()
			err := wc.conn.WriteMessage(websocket.PingMessage, []byte{})
			wc.mu.Unlock()
			if err != nil {
				return err
			}
		}
	}
}

// Send sends a message through the WebSocket connection
func (wc *WebSocketConnection) Send(msg *WebSocketMessage) error {
	select {
	case wc.sendChan <- msg:
		logger.DebugCF("web.ws", "Message queued for sending", map[string]any{
			"chat_id":    wc.chatID,
			"type":       msg.Type,
			"role":       msg.Role,
			"content_len": len(msg.Content),
		})
		return nil
	case <-wc.done:
		return fmt.Errorf("websocket connection closed")
	default:
		logger.WarnCF("web.ws", "Send channel full, dropping message", map[string]any{
			"chat_id": wc.chatID,
		})
		return fmt.Errorf("send channel full")
	}
}

// Close closes the WebSocket connection
func (wc *WebSocketConnection) Close() error {
	close(wc.done)
	wc.mu.Lock()
	defer wc.mu.Unlock()
	return wc.conn.Close()
}

// IsOpen checks if the connection is still open
func (wc *WebSocketConnection) IsOpen() bool {
	select {
	case <-wc.done:
		return false
	default:
		return true
	}
}

// upgrader is the WebSocket upgrader configuration
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, you should validate the origin header
		return true
	},
}
