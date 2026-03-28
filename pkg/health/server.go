package health

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
)

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
	chatFunc      func(ctx context.Context, message, sessionID, chatID string) (string, error)
	apiKey        string
	chatResults   map[string]*chatStatus
	chatResultsMu sync.RWMutex
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

func NewServer(host string, port int) *Server {
	mux := http.NewServeMux()
	s := &Server{
		ready:       false,
		checks:      make(map[string]Check),
		startTime:   time.Now(),
		chatResults: make(map[string]*chatStatus),
	}

	mux.HandleFunc("/health", s.healthHandler)
	mux.HandleFunc("/ready", s.readyHandler)
	mux.HandleFunc("/reload", s.reloadHandler)
	mux.HandleFunc("/chat", s.chatHandler)
	mux.HandleFunc("/cgat", s.chatHandler)

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

func (s *Server) verifyAPIKey(r *http.Request) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.apiKey == "" {
		return true
	}
	return r.Header.Get("X-API-Key") == s.apiKey
}

func (s *Server) reloadHandler(w http.ResponseWriter, r *http.Request) {
	if !s.verifyAPIKey(r) {
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

// HandlerMux is the interface for registering HTTP handlers, used by
// RegisterOnMux so that callers can pass any mux implementation
// (e.g. *http.ServeMux or a custom dynamic mux).
type HandlerMux interface {
	Handle(pattern string, handler http.Handler)
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
}

// RegisterOnMux registers /health, /ready, /reload and /chat handlers onto the given mux.
// This allows the health endpoints to be served by a shared HTTP server.
func (s *Server) RegisterOnMux(mux HandlerMux) {
	mux.HandleFunc("/health", s.healthHandler)
	mux.HandleFunc("/ready", s.readyHandler)
	mux.HandleFunc("/reload", s.reloadHandler)
	mux.HandleFunc("/chat", s.chatHandler)
	mux.HandleFunc("/cgat", s.chatHandler)
	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		logger.Error("GATEWAY IS HITTING ITSELF FOR LLM CALLS!")
		http.Error(w, "GATEWAY LOOP DETECTION", http.StatusLoopDetected)
	})
}

// chatHandler handles POST /chat (initiate async) and GET /chat (poll for result).
// POST body: {"message": "...", "session_id": "..." (optional)}
// POST response: {"session_id": "...", "status": "pending"}
// GET query: ?session_id=...
// GET response: {"response": "...", "status": "completed"}
func (s *Server) chatHandler(w http.ResponseWriter, r *http.Request) {
	if !s.verifyAPIKey(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ChatResponse{Error: "unauthorized"})
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

func statusString(ok bool) string {
	if ok {
		return "ok"
	}
	return "fail"
}
