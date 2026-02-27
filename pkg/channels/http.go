package channels

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

//go:embed http_templates/*
var httpTemplates embed.FS

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type HTTPChannel struct {
	*BaseChannel
	server   *http.Server
	config   *config.HTTPConfig
	clients  sync.Map // chatID -> *websocket.Conn
	mu       sync.RWMutex
	sessions map[string]*HTTPSession
}

type HTTPSession struct {
	ChatID    string
	Conn      *websocket.Conn
	Connected time.Time
}

type WSMessage struct {
	Type    string `json:"type"`
	Content string `json:"content"`
	ChatID  string `json:"chat_id,omitempty"`
}

func NewHTTPChannel(cfg *config.HTTPConfig, messageBus *bus.MessageBus) (*HTTPChannel, error) {
	base := NewBaseChannel("http", cfg, messageBus, cfg.AllowFrom)

	return &HTTPChannel{
		BaseChannel: base,
		config:      cfg,
		sessions:    make(map[string]*HTTPSession),
	}, nil
}

func (c *HTTPChannel) Start(ctx context.Context) error {
	host := c.config.Host
	if host == "" {
		host = "0.0.0.0"
	}
	port := c.config.Port
	if port == 0 {
		port = 8080
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", c.handleIndex)
	mux.HandleFunc("/ws", c.handleWebSocket)
	mux.HandleFunc("/api/chat", c.handleChatAPI)

	c.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", host, port),
		Handler: mux,
	}

	go func() {
		logger.InfoCF("http", "HTTP server starting", map[string]any{
			"address": fmt.Sprintf("http://%s:%d", host, port),
		})
		if err := c.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.ErrorCF("http", "HTTP server error", map[string]any{
				"error": err.Error(),
			})
		}
	}()

	c.setRunning(true)
	logger.InfoCF("http", "HTTP channel started", map[string]any{
		"host": host,
		"port": port,
	})

	return nil
}

func (c *HTTPChannel) Stop(ctx context.Context) error {
	logger.InfoC("http", "Stopping HTTP server...")
	c.setRunning(false)

	if c.server != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := c.server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("failed to shutdown HTTP server: %w", err)
		}
	}

	return nil
}

func (c *HTTPChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return fmt.Errorf("http channel not running")
	}

	c.mu.RLock()
	session, exists := c.sessions[msg.ChatID]
	c.mu.RUnlock()

	if !exists {
		logger.DebugCF("http", "No active session for chat", map[string]any{
			"chat_id": msg.ChatID,
		})
		return nil
	}

	// Render Markdown to HTML before sending
	renderedContent := renderMarkdown(msg.Content)

	wsMsg := WSMessage{
		Type:    "response",
		Content: renderedContent,
		ChatID:  msg.ChatID,
	}

	if err := session.Conn.WriteJSON(wsMsg); err != nil {
		logger.ErrorCF("http", "Failed to send WebSocket message", map[string]any{
			"error": err.Error(),
		})
		c.removeSession(msg.ChatID)
		return err
	}

	return nil
}

func (c *HTTPChannel) handleIndex(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFS(httpTemplates, "http_templates/index.html")
	if err != nil {
		http.Error(w, "Failed to load template", http.StatusInternalServerError)
		return
	}

	data := struct {
		Title string
	}{
		Title: "PicoClaw Web Chat",
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		logger.ErrorCF("http", "Failed to execute template", map[string]any{
			"error": err.Error(),
		})
	}
}

func (c *HTTPChannel) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.ErrorCF("http", "WebSocket upgrade failed", map[string]any{
			"error": err.Error(),
		})
		return
	}

	chatID := generateChatID(r)

	c.mu.Lock()
	c.sessions[chatID] = &HTTPSession{
		ChatID:    chatID,
		Conn:      conn,
		Connected: time.Now(),
	}
	c.mu.Unlock()

	logger.InfoCF("http", "WebSocket client connected", map[string]any{
		"chat_id": chatID,
	})

	defer func() {
		c.removeSession(chatID)
		conn.Close()
	}()

	for {
		var msg WSMessage
		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.ErrorCF("http", "WebSocket read error", map[string]any{
					"error": err.Error(),
				})
			}
			break
		}

		switch msg.Type {
		case "message":
			c.handleIncomingMessage(chatID, msg.Content)
		case "ping":
			_ = conn.WriteJSON(WSMessage{Type: "pong"})
		}
	}
}

func (c *HTTPChannel) handleChatAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Content string `json:"content"`
		ChatID  string `json:"chat_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.ChatID == "" {
		req.ChatID = generateChatID(r)
	}

	c.handleIncomingMessage(req.ChatID, req.Content)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"chat_id": req.ChatID,
	})
}

func (c *HTTPChannel) handleIncomingMessage(chatID, content string) {
	senderID := "web_user"

	logger.DebugCF("http", "Received message", map[string]any{
		"chat_id": chatID,
		"content": content,
	})

	c.HandleMessage(senderID, chatID, content, nil, map[string]string{
		"peer_kind": "direct",
		"peer_id":   chatID,
	})
}

func (c *HTTPChannel) removeSession(chatID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if session, exists := c.sessions[chatID]; exists {
		session.Conn.Close()
		delete(c.sessions, chatID)
		logger.InfoCF("http", "WebSocket client disconnected", map[string]any{
			"chat_id": chatID,
		})
	}
}

func generateChatID(r *http.Request) string {
	return fmt.Sprintf("web_%d", time.Now().UnixNano())
}

// renderMarkdown converts Markdown to HTML using goldmark
func renderMarkdown(input string) string {
	// Debug: log first 100 chars of input
	// logger.DebugF("http renderMarkdown input", map[string]any{"input": input[:min(100, len(input))]})
	
	// Convert newlines to <br> before markdown processing
	input = strings.ReplaceAll(input, "\n", "  \n")
	
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.TaskList,
			extension.Typographer,
		),
	)

	var buf strings.Builder
	if err := md.Convert([]byte(input), &buf); err != nil {
		logger.ErrorCF("http", "renderMarkdown error", map[string]any{"error": err.Error()})
		return input // fallback to plain text on error
	}
	
	// Add target="_blank" to all links for security and UX
	html := buf.String()
	
	// Pattern to match <a href="..."> links
	// This regex adds target="_blank" and rel="noopener noreferrer" to all <a> tags
	// that don't already have a target attribute
	re := regexp.MustCompile(`<a\s+([^>]*?)href="([^"]*?)"([^>]*?)>`)
	html = re.ReplaceAllStringFunc(html, func(match string) string {
		// Check if target attribute already exists
		if strings.Contains(match, `target=`) {
			return match
		}
		// Insert target and rel attributes before closing >
		return strings.Replace(match, `>`, ` target="_blank" rel="noopener noreferrer">`, 1)
	})
	
	return html
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}