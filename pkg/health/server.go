package health

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// Mux defines the interface required for registering health handlers.
type Mux interface {
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
}

// ChatRequest is the JSON body for POST /chat.
type ChatRequest struct {
	Message   string `json:"message"`
	SessionID string `json:"session_id,omitempty"`
	ChatID    string `json:"chat_id,omitempty"` // Alias for session_id to match PicoClaw terminology
}

// ChatResponse is the JSON response from /chat.
type ChatResponse struct {
	Response  string `json:"response,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	Status    string `json:"status,omitempty"`
	Error     string `json:"error,omitempty"`
}

type chatStatus struct {
	Response  string
	Error     error
	Done      bool
	CreatedAt time.Time
}

type Server struct {
	server        *http.Server
	mu            sync.RWMutex
	ready         bool
	checks        map[string]Check
	startTime     time.Time
	reloadFunc    func() error
	authToken     string // optional bearer token for protected endpoints
	chatFunc      func(ctx context.Context, message, sessionID, chatID string) (string, error)
	apiKey        string
	chatResults   map[string]*chatStatus
	chatResultsMu sync.RWMutex
	rateLimits    sync.Map // key: string (ID or IP), value: time.Time
}

type Check struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Message   string    `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type StatusResponse struct {
	Status string           `json:"status"`
	Uptime string           `json:"uptime"`
	Checks map[string]Check `json:"checks,omitempty"`
	Pid    int              `json:"pid"`
}

func NewServer(host string, port int, token string) *Server {
	mux := http.NewServeMux()
	s := &Server{
		ready:       false,
		checks:      make(map[string]Check),
		startTime:   time.Now(),
		authToken:   token,
		chatResults: make(map[string]*chatStatus),
	}

	mux.HandleFunc("/health", s.healthHandler)
	mux.HandleFunc("/ready", s.readyHandler)
	mux.HandleFunc("/reload", s.reloadHandler)
	mux.HandleFunc("/chat", s.chatHandler)

	// Start task cleanup goroutine
	go s.taskCleanupLoop()

	addr := fmt.Sprintf("%s:%d", host, port)
	s.server = &http.Server{
		Addr:        addr,
		Handler:     mux,
		ReadTimeout: 10 * time.Second,
		// WriteTimeout must be long enough for LLM inference; 5 min is generous.
		WriteTimeout: 5 * time.Minute,
	}

	return s
}

func (s *Server) Start() error {
	s.mu.Lock()
	s.ready = true
	s.mu.Unlock()
	return s.server.ListenAndServe()
}

func (s *Server) StartContext(ctx context.Context) error {
	s.mu.Lock()
	s.ready = true
	s.mu.Unlock()

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return s.server.Shutdown(context.Background())
	}
}

func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	s.ready = false
	s.mu.Unlock()
	return s.server.Shutdown(ctx)
}

func (s *Server) SetReady(ready bool) {
	s.mu.Lock()
	s.ready = ready
	s.mu.Unlock()
}

func (s *Server) RegisterCheck(name string, checkFn func() (bool, string)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	status, msg := checkFn()
	s.checks[name] = Check{
		Name:      name,
		Status:    statusString(status),
		Message:   msg,
		Timestamp: time.Now(),
	}
}

// SetReloadFunc sets the callback function for config reload.
func (s *Server) SetReloadFunc(fn func() error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reloadFunc = fn
}

// SetChatFunc sets the callback that processes /chat requests.
// fn receives the user message and an optional session ID and must return the
// agent's reply (or an error). It is called synchronously inside the HTTP
// handler, so the write timeout on the server governs the maximum duration.
func (s *Server) SetChatFunc(fn func(ctx context.Context, message, sessionID, chatID string) (string, error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.chatFunc = fn
}

// SetAPIKey sets the expected X-API-Key header value.
func (s *Server) SetAPIKey(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.apiKey = key
}

// SetAuthToken sets the expected Bearer token.
func (s *Server) SetAuthToken(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.authToken = token
}

func (s *Server) verifyAuth(r *http.Request) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// If no authentication is configured, allow the request.
	if s.apiKey == "" && s.authToken == "" {
		return true
	}

	// Check X-API-Key header.
	if s.apiKey != "" {
		gotKey := r.Header.Get("X-API-Key")
		if subtle.ConstantTimeCompare([]byte(gotKey), []byte(s.apiKey)) == 1 {
			return true
		}
	}

	// Check Authorization: Bearer <token> header.
	if s.authToken != "" {
		authHeader := r.Header.Get("Authorization")
		const prefix = "Bearer "
		if len(authHeader) > len(prefix) && strings.EqualFold(authHeader[:len(prefix)], prefix) {
			gotToken := authHeader[len(prefix):]
			if subtle.ConstantTimeCompare([]byte(gotToken), []byte(s.authToken)) == 1 {
				return true
			}
		}
	}

	return false
}

func (s *Server) reloadHandler(w http.ResponseWriter, r *http.Request) {
	if !s.verifyAuth(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed, use POST"})
		return
	}

	s.mu.Lock()
	reloadFunc := s.reloadFunc
	s.mu.Unlock()

	if reloadFunc == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "reload not configured"})
		return
	}

	if err := reloadFunc(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "reload triggered"})
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	uptime := time.Since(s.startTime)
	resp := StatusResponse{
		Status: "ok",
		Uptime: uptime.String(),
		Pid:    os.Getpid(),
	}

	json.NewEncoder(w).Encode(resp)
}

func (s *Server) readyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	s.mu.RLock()
	ready := s.ready
	checks := make(map[string]Check)
	maps.Copy(checks, s.checks)
	s.mu.RUnlock()

	if !ready {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(StatusResponse{
			Status: "not ready",
			Checks: checks,
		})
		return
	}

	for _, check := range checks {
		if check.Status == "fail" {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(StatusResponse{
				Status: "not ready",
				Checks: checks,
			})
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	uptime := time.Since(s.startTime)
	json.NewEncoder(w).Encode(StatusResponse{
		Status: "ready",
		Uptime: uptime.String(),
		Checks: checks,
	})
}

// HandlerMux defines the interface for an HTTP request multiplexer.
type HandlerMux interface {
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
}

// RegisterOnMux registers /health, /ready, /reload and /chat handlers onto the
// given mux. This allows the health endpoints to be served by a shared HTTP server.
func (s *Server) RegisterOnMux(mux HandlerMux) {
	mux.HandleFunc("/health", s.healthHandler)
	mux.HandleFunc("/ready", s.readyHandler)
	mux.HandleFunc("/reload", s.reloadHandler)
	mux.HandleFunc("/chat", s.chatHandler)
}

// chatHandler handles POST /chat (initiate async) and GET /chat (poll for result).
// POST body: {"message": "...", "session_id": "..." (optional)}
// POST response: {"session_id": "...", "status": "pending"}
// GET query: ?session_id=...
// GET response: {"response": "...", "status": "completed"}
func (s *Server) chatHandler(w http.ResponseWriter, r *http.Request) {
	if !s.verifyAuth(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ChatResponse{Error: "unauthorized"})
		return
	}

	if !s.checkRateLimit(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(ChatResponse{Error: "rate limit exceeded"})
		return
	}

	if r.Method == http.MethodPost {
		s.handlePostChat(w, r)
		return
	} else if r.Method == http.MethodGet {
		s.handleGetChat(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusMethodNotAllowed)
	json.NewEncoder(w).Encode(ChatResponse{Error: "method not allowed, use POST or GET"})
}

func (s *Server) handlePostChat(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	chatFunc := s.chatFunc
	s.mu.RUnlock()

	if chatFunc == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(ChatResponse{Error: "chat not configured"})
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ChatResponse{Error: "invalid JSON: " + err.Error()})
		return
	}
	if req.Message == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ChatResponse{Error: "message field is required"})
		return
	}

	sessionID := req.SessionID
	if sessionID == "" && req.ChatID != "" {
		sessionID = req.ChatID
	}

	chatID := req.ChatID
	if chatID == "" {
		// Try to extract ChatID/TenantID from common headers
		// These are ordered by specificity/reliability
		headers := []string{
			"X-PicoClaw-Chat-ID",
			"X-MS-CONVERSATION-ID", // Teams Conversation ID
			"X-MS-TENANT-ID",       // Teams Tenant ID
			"X-User-ID",
			"X-Session-ID",
			"X-MS-CLIENT-PRINCIPAL-ID",   // Azure App Service / Container Apps (EasyAuth)
			"X-MS-CLIENT-PRINCIPAL-NAME", // Azure App Service Email/Username
			"Ocp-Apim-Subscription-Id",   // Azure APIM (if configured)
		}

		for _, h := range headers {
			if val := r.Header.Get(h); val != "" {
				chatID = val
				break
			}
		}

		// Fallback to SessionID if provided in body, otherwise empty (global)
		if chatID == "" {
			chatID = req.SessionID
		}
	}
	chatID = s.sanitizeID(chatID)
	sessionID = s.sanitizeID(sessionID)

	if chatID != "" {
		logger.InfoCF("api", "Resolved isolation ID for request", map[string]any{
			"chat_id":    chatID,
			"session_id": sessionID,
		})
	} else {
		// Log all headers for debugging (excluding sensitive ones)
		headers := make(map[string]string)
		for k, v := range r.Header {
			if k == "Authorization" || k == "X-Api-Key" || k == "Ocp-Apim-Subscription-Key" {
				headers[k] = "REDACTED"
			} else if len(v) > 0 {
				headers[k] = v[0]
			}
		}
		logger.DebugCF("api", "Chat request received without explicit ChatID. Checking headers...", map[string]any{
			"headers": headers,
		})
	}

	if sessionID == "" {
		sessionID = fmt.Sprintf("chat-%d", time.Now().UnixNano())
	} else {
		// Even if provided, sanitize the user-provided sessionID again to be sure
		sessionID = s.sanitizeID(sessionID)
	}

	// Initialize status
	s.chatResultsMu.Lock()
	s.chatResults[sessionID] = &chatStatus{
		CreatedAt: time.Now(),
	}
	s.chatResultsMu.Unlock()

	// Start processing in background
	go func() {
		// Use a long-running context for the chat call, but don't bind to r.Context()
		// which will be cancelled when this request finishes.
		ctx := context.Background()
		logger.Debugf("Starting async chat for session %s", sessionID)
		reply, err := chatFunc(ctx, req.Message, sessionID, chatID)

		s.chatResultsMu.Lock()
		defer s.chatResultsMu.Unlock()
		if result, ok := s.chatResults[sessionID]; ok {
			result.Response = reply
			result.Error = err
			result.Done = true
			logger.Debugf("Finished async chat for session %s (err=%v)", sessionID, err)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(ChatResponse{
		SessionID: sessionID,
		Status:    "pending",
	})
}

func (s *Server) handleGetChat(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ChatResponse{Error: "session_id query parameter is required"})
		return
	}

	s.chatResultsMu.RLock()
	result, ok := s.chatResults[sessionID]
	if !ok {
		s.chatResultsMu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ChatResponse{Error: "session not found"})
		return
	}

	// Read fields while holding the lock to avoid race conditions
	done := result.Done
	response := result.Response
	errVal := result.Error
	s.chatResultsMu.RUnlock()

	if !done {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ChatResponse{
			SessionID: sessionID,
			Status:    "pending",
		})
		return
	}

	if errVal != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ChatResponse{
			SessionID: sessionID,
			Status:    "error",
			Error:     errVal.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ChatResponse{
		SessionID: sessionID,
		Status:    "completed",
		Response:  response,
	})
}

func (s *Server) taskCleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.chatResultsMu.Lock()
		now := time.Now()
		for id, status := range s.chatResults {
			// Keep pending tasks for 2 hours, completed/error for 1 hour
			expiry := time.Hour
			if !status.Done {
				expiry = 2 * time.Hour
			}

			if now.Sub(status.CreatedAt) > expiry {
				delete(s.chatResults, id)
				logger.Debugf("Cleaned up expired chat session %s", id)
			}
		}
		s.chatResultsMu.Unlock()
	}
}

func (s *Server) sanitizeID(id string) string {
	if len(id) > 128 {
		id = id[:128]
	}

	result := make([]rune, 0, len(id))
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			result = append(result, r)
		} else {
			result = append(result, '_')
		}
	}
	return string(result)
}

func (s *Server) checkRateLimit(r *http.Request) bool {
	// Simple rate limit: 1 request per second per ID or IP
	// This is defensive against automated spamming.
	key := r.Header.Get("X-PicoClaw-Chat-ID")
	if key == "" {
		key = r.RemoteAddr
		// Strip port if present
		if i := strings.LastIndex(key, ":"); i != -1 {
			key = key[:i]
		}
	}

	if val, ok := s.rateLimits.Load(key); ok {
		lastAccess := val.(time.Time)
		if time.Since(lastAccess) < time.Second {
			return false
		}
	}

	s.rateLimits.Store(key, time.Now())
	return true
}

func statusString(ok bool) string {
	if ok {
		return "ok"
	}
	return "fail"
}
