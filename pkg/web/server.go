package web

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sipeed/picoclaw/pkg/voice"
	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/skills"
)

const sessionCookieName = "picoclaw_session"
const sessionTTL = 24 * time.Hour

// Server is the web management UI server.
type Server struct {
	cfg            *config.Config
	configPath     string
	mu             sync.RWMutex
	sessions       map[string]time.Time
	httpServer     *http.Server
	agentLoop      *agent.AgentLoop
	msgBus         *bus.MessageBus
	responseChans  map[string]chan string       // Map chatID -> response channel (HTTP mode)
	responseMu     sync.RWMutex
	wsConnManager  *WSConnManager                // WebSocket connection manager
	ttsSynthesizer *voice.AliyunTTSSynthesizer   // TTS synthesizer
}

// NewServer creates a new web management server.
func NewServer(cfg *config.Config, configPath string) *Server {
	s := &Server{
		cfg:            cfg,
		configPath:     configPath,
		sessions:       make(map[string]time.Time),
		responseChans:  make(map[string]chan string),
		wsConnManager:  NewWSConnManager(),
	}

	// Initialize TTS synthesizer if voice is enabled
	if cfg.Voice.Enabled && cfg.Voice.TTS.Enabled {
		ttsConfig := voice.AliyunTTSConfig{
			APIKey:     cfg.Voice.TTS.APIKey,
			Voice:      cfg.Voice.TTS.Voice,
			Speed:      cfg.Voice.TTS.Speed,
			Volume:     cfg.Voice.TTS.Volume,
			Pitch:      cfg.Voice.TTS.Pitch,
			Format:     cfg.Voice.TTS.Format,
			SampleRate: cfg.Voice.TTS.SampleRate,
		}
		s.ttsSynthesizer = voice.NewAliyunTTSSynthesizer(ttsConfig)
		logger.InfoCF("web", "TTS synthesizer initialized", map[string]any{
			"voice":       ttsConfig.Voice,
			"format":      ttsConfig.Format,
			"sample_rate": ttsConfig.SampleRate,
		})
	}

	mux := http.NewServeMux()

	// API routes (must be registered before static file handler)
	mux.HandleFunc("/api/auth/login", s.handleLogin)
	mux.HandleFunc("/api/auth/logout", s.handleLogout)
	mux.HandleFunc("/api/config", s.withAuth(s.handleConfig))
	mux.HandleFunc("/api/status", s.withAuth(s.handleStatus))
	mux.HandleFunc("/api/skills", s.withAuth(s.handleSkills))
	mux.HandleFunc("/api/chat", s.withAuth(s.handleChat))
	mux.HandleFunc("/api/ws/chat", s.withAuth(s.handleWebSocketChat))
	mux.HandleFunc("/api/voice/tts", s.withAuth(s.handleTTS))

	// Serve embedded static files
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(fmt.Sprintf("web: failed to sub static fs: %v", err))
	}
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	host := cfg.Web.Host
	if host == "" {
		host = "0.0.0.0"
	}
	port := cfg.Web.Port
	if port == 0 {
		port = 18799
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return s
}

// SetAgentLoop injects the agent loop and message bus into the server.
// This enables the web server to communicate with the agent.
func (s *Server) SetAgentLoop(agentLoop *agent.AgentLoop, msgBus *bus.MessageBus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.agentLoop = agentLoop
	s.msgBus = msgBus
	
	// Start listening for agent responses
	go s.listenForResponses()
}

// listenForResponses listens for outbound messages from the agent and routes them to waiting handlers.
func (s *Server) listenForResponses() {
	s.mu.RLock()
	msgBus := s.msgBus
	s.mu.RUnlock()

	if msgBus == nil {
		return
	}

	// Create a background context for listening
	ctx := context.Background()

	for {
		// Listen for outbound messages from agent
		msg, ok := msgBus.SubscribeOutbound(ctx)
		if !ok {
			continue
		}

		// Only handle web channel responses
		if msg.Channel != "web" {
			continue
		}

		// Route to waiting handler if exists
		s.responseMu.RLock()
		responseChan, exists := s.responseChans[msg.ChatID]
		s.responseMu.RUnlock()

		if exists {
			// Try to send response without blocking
			select {
			case responseChan <- msg.Content:
				logger.DebugCF("web", "Response routed to handler", map[string]any{
					"chat_id": msg.ChatID,
					"length":  len(msg.Content),
				})
			default:
				// Channel is full or closed, skip
			}
		}

		// Also send to WebSocket connection if it exists
		wsConn := s.wsConnManager.GetConnection(msg.ChatID)
		if wsConn != nil && wsConn.IsOpen() {
			wsMsg := &WebSocketMessage{
				Type:      "message",
				Role:      "assistant",
				ChatID:    msg.ChatID,
				Content:   msg.Content,
				Timestamp: time.Now().Unix(),
			}
			if err := wsConn.Send(wsMsg); err != nil {
				logger.DebugCF("web.ws", "Failed to send response to WebSocket", map[string]any{
					"chat_id": msg.ChatID,
					"error":   err.Error(),
				})
			} else {
				content := msg.Content
				if len(content) > 50 {
					content = content[:50]
				}
				logger.InfoCF("web.ws", "Response sent to WebSocket", map[string]any{
					"chat_id": msg.ChatID,
					"content": content,
				})
			}
		} else {
			logger.WarnCF("web.ws", "WebSocket connection not found for message", map[string]any{
				"chat_id": msg.ChatID,
				"found":   wsConn != nil,
				"is_open": wsConn != nil && wsConn.IsOpen(),
			})
		}
	}
}

// registerResponseChannel registers a channel to receive the agent's response for a chat ID.
func (s *Server) registerResponseChannel(chatID string) chan string {
	responseChan := make(chan string, 1)
	s.responseMu.Lock()
	s.responseChans[chatID] = responseChan
	s.responseMu.Unlock()
	return responseChan
}

// unregisterResponseChannel removes the response channel for a chat ID.
func (s *Server) unregisterResponseChannel(chatID string) {
	s.responseMu.Lock()
	delete(s.responseChans, chatID)
	s.responseMu.Unlock()
}

// Addr returns the server listen address.
func (s *Server) Addr() string {
	return s.httpServer.Addr
}

// Start starts the server and blocks.
func (s *Server) Start() error {
	logger.InfoCF("web", "Web management UI started", map[string]any{"addr": s.httpServer.Addr})
	fmt.Printf("🦞 Web管理界面已启动: http://%s\n", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// StartContext starts the server and respects context cancellation.
func (s *Server) StartContext(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.httpServer.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(shutdownCtx)
	}
}

// Stop gracefully stops the server.
func (s *Server) Stop(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// --- Session helpers ---

func (s *Server) generateToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *Server) isValidSession(token string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	expiry, ok := s.sessions[token]
	if !ok {
		return false
	}
	return time.Now().Before(expiry)
}

func (s *Server) cleanExpiredSessions() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for token, expiry := range s.sessions {
		if now.After(expiry) {
			delete(s.sessions, token)
		}
	}
}

// --- Middleware ---

func (s *Server) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil || !s.isValidSession(cookie.Value) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}
		next(w, r)
	}
}

func jsonResponse(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// --- Handlers ---

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	s.mu.RLock()
	wantUser := s.cfg.Web.Username
	wantPass := s.cfg.Web.Password
	s.mu.RUnlock()

	if req.Username != wantUser || req.Password != wantPass {
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "用户名或密码不正确"})
		return
	}

	// Clean stale sessions periodically
	s.cleanExpiredSessions()

	token := s.generateToken()
	s.mu.Lock()
	s.sessions[token] = time.Now().Add(sessionTTL)
	s.mu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		HttpOnly: true,
		Path:     "/",
		MaxAge:   int(sessionTTL.Seconds()),
		SameSite: http.SameSiteLaxMode,
	})

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		s.mu.Lock()
		delete(s.sessions, cookie.Value)
		s.mu.Unlock()
	}

	http.SetCookie(w, &http.Cookie{
		Name:   sessionCookieName,
		Value:  "",
		MaxAge: -1,
		Path:   "/",
	})

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.mu.RLock()
		cfg := s.cfg
		s.mu.RUnlock()
		jsonResponse(w, http.StatusOK, cfg)

	case http.MethodPut:
		var newCfg config.Config
		if err := json.NewDecoder(r.Body).Decode(&newCfg); err != nil {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		if err := config.SaveConfig(s.configPath, &newCfg); err != nil {
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		// Detect which sections changed
		s.mu.RLock()
		oldCfg := s.cfg
		s.mu.RUnlock()

		// Update in-memory config
		s.mu.Lock()
		s.cfg = &newCfg
		s.mu.Unlock()

		logger.InfoCF("web", "Config saved via web UI", nil)

		// TODO: Implement change detection and hot reload
		// For now, just log that config was saved
		_ = oldCfg // Use oldCfg to avoid unused variable error

		// Return simple response for now
		jsonResponse(w, http.StatusOK, map[string]any{
			"status": "saved",
		})

	default:
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	hostname, _ := os.Hostname()

	s.mu.RLock()
	modelCount := len(s.cfg.ModelList)
	enabledChannels := countEnabledChannels(s.cfg)
	s.mu.RUnlock()

	jsonResponse(w, http.StatusOK, map[string]any{
		"status":           "running",
		"hostname":         hostname,
		"model_count":      modelCount,
		"enabled_channels": enabledChannels,
	})
}

func countEnabledChannels(cfg *config.Config) int {
	count := 0
	ch := cfg.Channels
	if ch.Telegram.Enabled {
		count++
	}
	if ch.Discord.Enabled {
		count++
	}
	if ch.Feishu.Enabled {
		count++
	}
	if ch.DingTalk.Enabled {
		count++
	}
	if ch.Slack.Enabled {
		count++
	}
	if ch.LINE.Enabled {
		count++
	}
	if ch.OneBot.Enabled {
		count++
	}
	if ch.WeCom.Enabled {
		count++
	}
	if ch.WeComApp.Enabled {
		count++
	}
	if ch.QQ.Enabled {
		count++
	}
	if ch.WhatsApp.Enabled {
		count++
	}
	if ch.MaixCam.Enabled {
		count++
	}
	return count
}

// SkillsDirInfo describes a single skills source directory.
type SkillsDirInfo struct {
	Label  string `json:"label"`
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
}

// SkillsResponse is the JSON payload returned by /api/skills.
type SkillsResponse struct {
	Total  int                `json:"total"`
	Dirs   []SkillsDirInfo    `json:"dirs"`
	Skills []skills.SkillInfo `json:"skills"`
}

func (s *Server) handleSkills(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	cfg := s.cfg
	s.mu.RUnlock()

	// Resolve the same three paths the agent uses.
	workspacePath := cfg.WorkspacePath()
	globalSkillsDir := filepath.Join(globalConfigDir(), "skills")
	wd, _ := os.Getwd()
	builtinSkillsDir := filepath.Join(wd, "skills")

	dirs := []SkillsDirInfo{
		{
			Label:  "workspace",
			Path:   filepath.Join(workspacePath, "skills"),
			Exists: dirExists(filepath.Join(workspacePath, "skills")),
		},
		{
			Label:  "global",
			Path:   globalSkillsDir,
			Exists: dirExists(globalSkillsDir),
		},
		{
			Label:  "builtin",
			Path:   builtinSkillsDir,
			Exists: dirExists(builtinSkillsDir),
		},
	}

	loader := skills.NewSkillsLoader(workspacePath, globalSkillsDir, builtinSkillsDir)
	list := loader.ListSkills()

	jsonResponse(w, http.StatusOK, SkillsResponse{
		Total:  len(list),
		Dirs:   dirs,
		Skills: list,
	})
}

// globalConfigDir returns ~/.picoclaw (same as agent/context.go).
func globalConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".picoclaw")
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// ChatRequest represents a chat message from the web UI.
type ChatRequest struct {
	Message string `json:"message"`
	ChatID  string `json:"chat_id,omitempty"` // Session ID for conversation history
}

// ChatResponse represents the AI response.
type ChatResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
	ChatID  string `json:"chat_id"`
}

// handleChat processes chat messages and communicates with the agent.
func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Check if HTTP chat is enabled
	s.mu.RLock()
	httpEnabled := s.cfg.Web.Chat.HTTP
	chatTimeout := time.Duration(s.cfg.Web.Chat.Timeout) * time.Second
	s.mu.RUnlock()

	if !httpEnabled {
		jsonResponse(w, http.StatusServiceUnavailable, map[string]string{
			"error": "HTTP chat mode is disabled",
		})
		return
	}

	// Default timeout to 10 seconds if not configured
	if chatTimeout == 0 {
		chatTimeout = 10 * time.Second
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Message == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "message cannot be empty"})
		return
	}

	// Use a default chat ID if not provided
	chatID := req.ChatID
	if chatID == "" {
		chatID = "web-http-" + fmt.Sprintf("%d", time.Now().UnixNano())
	}

	// Check if agent loop is initialized
	s.mu.RLock()
	agentLoop := s.agentLoop
	msgBus := s.msgBus
	s.mu.RUnlock()

	if agentLoop == nil || msgBus == nil {
		jsonResponse(w, http.StatusServiceUnavailable, map[string]string{
			"error": "Agent not initialized",
		})
		return
	}

	// Register response channel before sending message
	responseChan := s.registerResponseChannel(chatID)
	defer s.unregisterResponseChannel(chatID)

	// Send message to agent via inbound message bus
	inboundMsg := bus.InboundMessage{
		Channel:  "web",
		SenderID: "web-http",
		ChatID:   chatID,
		Content:  req.Message,
	}
	msgBus.PublishInbound(context.Background(), inboundMsg)

	// Wait for agent response with configurable timeout
	ctx, cancel := context.WithTimeout(context.Background(), chatTimeout)
	defer cancel()

	select {
	case agentResponse := <-responseChan:
		// Got response from agent
		resp := ChatResponse{
			Success: true,
			Message: agentResponse,
			ChatID:  chatID,
		}
		jsonResponse(w, http.StatusOK, resp)
	case <-ctx.Done():
		// Timeout - respond with acknowledgement
		resp := ChatResponse{
			Success: true,
			Message: "Message sent to Agent. Processing...",
			ChatID:  chatID,
		}
		jsonResponse(w, http.StatusOK, resp)
	}
}

// handleWebSocketChat handles WebSocket connections for real-time chat
func (s *Server) handleWebSocketChat(w http.ResponseWriter, r *http.Request) {
	// Check if WebSocket chat is enabled
	s.mu.RLock()
	wsEnabled := s.cfg.Web.Chat.WebSocket
	chatTimeout := time.Duration(s.cfg.Web.Chat.Timeout) * time.Second
	s.mu.RUnlock()

	if !wsEnabled {
		jsonResponse(w, http.StatusServiceUnavailable, map[string]string{
			"error": "WebSocket chat mode is disabled",
		})
		return
	}

	// Default timeout to 10 seconds if not configured
	if chatTimeout == 0 {
		chatTimeout = 10 * time.Second
	}

	// Upgrade to WebSocket
	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.WarnCF("web.ws", "WebSocket upgrade failed", map[string]any{
			"error": err.Error(),
		})
		return
	}
	defer wsConn.Close()

	// Use a consistent chat ID for WebSocket connections (matches frontend default)
	chatID := "web-ws-session"

	// Create WebSocket connection wrapper
	wsConnWrapper := NewWebSocketConnection(wsConn, chatID, s)
	s.wsConnManager.Register(wsConnWrapper)
	logger.InfoCF("web.ws", "WebSocket chat connection established", map[string]any{
		"chat_id": chatID,
	})
	defer func() {
		wsConnWrapper.Close()
		s.wsConnManager.Unregister(chatID)
	}()

	// Check if agent loop is initialized
	s.mu.RLock()
	agentLoop := s.agentLoop
	msgBus := s.msgBus
	s.mu.RUnlock()

	if agentLoop == nil || msgBus == nil {
		wsConnWrapper.Send(&WebSocketMessage{
			Type:      "error",
			ChatID:    chatID,
			Error:     "Agent not initialized",
			Timestamp: time.Now().Unix(),
		})
		return
	}

	// Send welcome message
	wsConnWrapper.Send(&WebSocketMessage{
		Type:      "status",
		ChatID:    chatID,
		Content:   "Connected to PicoClaw Web Chat",
		Timestamp: time.Now().Unix(),
	})

	// Create context for goroutine coordination
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start reading messages in a goroutine
	readDone := make(chan error, 1)
	go func() {
		readDone <- wsConnWrapper.Read(ctx, func(msg *WebSocketMessage) error {
			if msg.Type != "message" {
				return nil
			}

			if msg.Content == "" {
				wsConnWrapper.Send(&WebSocketMessage{
					Type:      "error",
					ChatID:    msg.ChatID,
					Error:     "Message cannot be empty",
					Timestamp: time.Now().Unix(),
				})
				return nil
			}

			// Send message to agent via inbound message bus
			inboundMsg := bus.InboundMessage{
				Channel:  "web",
				SenderID: "web-ws",
				ChatID:   msg.ChatID,
				Content:  msg.Content,
			}
			logger.DebugCF("web.ws", "Sending inbound message to agent", map[string]any{
				"chat_id": msg.ChatID,
				"content": msg.Content,
			})
			msgBus.PublishInbound(context.Background(), inboundMsg)

			// Send typing indicator
			wsConnWrapper.Send(&WebSocketMessage{
				Type:      "status",
				ChatID:    msg.ChatID,
				Content:   "Agent is processing...",
				Timestamp: time.Now().Unix(),
			})

			return nil
		})
	}()

	// Start writing messages in a goroutine
	writeDone := make(chan error, 1)
	go func() {
		writeDone <- wsConnWrapper.Write(ctx)
	}()

	// Register response channel so listenForResponses() can route messages to this WebSocket connection
	s.responseMu.Lock()
	s.responseChans[chatID] = make(chan string, 1)
	s.responseMu.Unlock()
	defer func() {
		s.responseMu.Lock()
		delete(s.responseChans, chatID)
		s.responseMu.Unlock()
	}()

	// Wait for either read or write to fail, or context cancellation
	select {
	case err := <-readDone:
		if err != nil && !websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			logger.DebugCF("web.ws", "WebSocket read error", map[string]any{
				"error":   err.Error(),
				"chat_id": chatID,
			})
		}
		cancel()
	case err := <-writeDone:
		if err != nil && !websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			logger.DebugCF("web.ws", "WebSocket write error", map[string]any{
				"error":   err.Error(),
				"chat_id": chatID,
			})
		}
		cancel()
	}
}

// TTSRequest TTS 请求
type TTSRequest struct {
	Text  string `json:"text"`
	Voice string `json:"voice"`
}

// handleTTS 处理语音合成请求
func (s *Server) handleTTS(w http.ResponseWriter, r *http.Request) {
	// Only support POST method
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Check if TTS is enabled
	s.mu.RLock()
	ttsSynthesizer := s.ttsSynthesizer
	cfg := s.cfg
	s.mu.RUnlock()

	if !cfg.Voice.Enabled || !cfg.Voice.TTS.Enabled {
		jsonResponse(w, http.StatusServiceUnavailable, map[string]string{"error": "TTS is not enabled"})
		return
	}

	if ttsSynthesizer == nil {
		jsonResponse(w, http.StatusServiceUnavailable, map[string]string{"error": "TTS synthesizer not initialized"})
		return
	}

	// Parse request
	var req TTSRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Text == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "text is required"})
		return
	}

	// Override voice if specified in request
	if req.Voice != "" {
		ttsSynthesizer.SetVoice(req.Voice)
	}

	// Synthesize speech
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	audioData, err := ttsSynthesizer.Synthesize(ctx, req.Text)
	if err != nil {
		logger.ErrorCF("web.voice", "TTS synthesis failed", map[string]any{
			"error":       err.Error(),
			"text_length": len(req.Text),
		})
		jsonResponse(w, http.StatusInternalServerError, map[string]string{
			"error": "TTS synthesis failed: " + err.Error(),
		})
		return
	}

	// Return audio as base64
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"audio":   base64.StdEncoding.EncodeToString(audioData),
		"format":  cfg.Voice.TTS.Format,
		"voice":   ttsSynthesizer.GetVoice(),
	})
}
