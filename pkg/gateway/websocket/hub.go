package websocket

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Message types for research events
const (
	MsgTypeAgentUpdate  = "agent_update"
	MsgTypeReportUpdate = "report_update"
	MsgTypeConfigChange = "config_change"
	MsgTypeError        = "error"
)

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Broadcast messages to all clients
	broadcast chan []byte

	// Mutex for thread-safe operations
	mu sync.RWMutex
}

// Client represents a WebSocket client
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

// Message represents a WebSocket message
type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// AgentUpdatePayload represents an agent status update
type AgentUpdatePayload struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Active   bool   `json:"active"`
	Progress int    `json:"progress"`
	Status   string `json:"status"`
	Type     string `json:"type"`
}

// ReportUpdatePayload represents a report status update
type ReportUpdatePayload struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	Progress int    `json:"progress"`
	Words    int    `json:"words"`
	Pages    int    `json:"pages"`
}

// ConfigChangePayload represents a config change event
type ConfigChangePayload struct {
	Type            string `json:"type"`
	Depth           string `json:"depth"`
	RestrictToGraph bool   `json:"restrict_to_graph"`
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 256),
	}
}

// Run starts the hub's message pump
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// BroadcastAgentUpdate broadcasts an agent update to all clients
func (h *Hub) BroadcastAgentUpdate(update AgentUpdatePayload) {
	msg := Message{
		Type:    MsgTypeAgentUpdate,
		Payload: mustMarshal(update),
	}
	h.broadcast <- mustMarshal(msg)
}

// BroadcastReportUpdate broadcasts a report update to all clients
func (h *Hub) BroadcastReportUpdate(update ReportUpdatePayload) {
	msg := Message{
		Type:    MsgTypeReportUpdate,
		Payload: mustMarshal(update),
	}
	h.broadcast <- mustMarshal(msg)
}

// BroadcastConfigChange broadcasts a config change to all clients
func (h *Hub) BroadcastConfigChange(update ConfigChangePayload) {
	msg := Message{
		Type:    MsgTypeConfigChange,
		Payload: mustMarshal(update),
	}
	h.broadcast <- mustMarshal(msg)
}

// ServeHTTP handles WebSocket upgrades
func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	 upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			// Allow all origins for now
			return true
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := &Client{
		hub:  h,
		conn: conn,
		send: make(chan []byte, 256),
	}

	h.register <- client

	// Start goroutines for read/write
	go client.writePump()
	go client.readPump()
}

// readPump reads messages from the WebSocket connection
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Time{})
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Time{})
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		// Handle incoming messages (e.g., subscribe to specific events)
		var msg Message
		if err := json.Unmarshal(message, &msg); err == nil {
			// Process subscription messages if needed
			_ = msg // Currently we broadcast everything, no per-client subscriptions
		}
	}
}

// writePump writes messages to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current WebSocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func mustMarshal(v interface{}) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}