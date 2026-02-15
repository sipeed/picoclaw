package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// openapiSpec holds the OpenAPI YAML spec, set via SetOpenAPISpec.
var openapiSpec []byte

// SetOpenAPISpec sets the embedded OpenAPI spec content.
func SetOpenAPISpec(spec []byte) {
	openapiSpec = spec
}

// version is set by the caller (main.go) via SetVersion.
var apiVersion = "dev"

// SetVersion sets the version string returned by the health endpoint.
func SetVersion(v string) {
	apiVersion = v
}

// API request/response types matching OpenAPI schemas
type ChatRequest struct {
	SessionID string `json:"session_id,omitempty"`
	Message   string `json:"message"`
}

type ChatResponse struct {
	SessionID string `json:"session_id"`
	Response  string `json:"response"`
}

type SessionList struct {
	Sessions []string `json:"sessions"`
}

type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

type ErrorResponse struct {
	Error string `json:"error"`
	Code  int    `json:"code"`
}

// APIServer holds the dependencies for API handlers.
type APIServer struct {
	agentLoop *agent.AgentLoop
	config    config.APIConfig
}

// NewAPIServer creates a new API server.
func NewAPIServer(agentLoop *agent.AgentLoop, cfg config.APIConfig) *APIServer {
	return &APIServer{
		agentLoop: agentLoop,
		config:    cfg,
	}
}

// Handler returns an http.Handler with all API routes and auth middleware.
func (s *APIServer) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/chat", s.handleChat)
	mux.HandleFunc("/api/sessions", s.handleSessions)
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/openapi.yaml", s.handleSpec)

	publicPaths := []string{"/api/health", "/api/openapi.yaml"}
	return AuthMiddleware(s.config.APIKey, publicPaths, mux)
}

func (s *APIServer) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Message == "" {
		writeError(w, http.StatusBadRequest, "message is required")
		return
	}

	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = fmt.Sprintf("api:%s", uuid.New().String())
	}

	logger.InfoCF("api", "Chat request", map[string]interface{}{
		"session_id": sessionID,
		"message":    req.Message,
	})

	response, err := s.agentLoop.ProcessDirect(context.Background(), req.Message, sessionID)
	if err != nil {
		logger.ErrorCF("api", "Chat error", map[string]interface{}{
			"error": err.Error(),
		})
		writeError(w, http.StatusInternalServerError, "failed to process message")
		return
	}

	writeJSON(w, http.StatusOK, ChatResponse{
		SessionID: sessionID,
		Response:  response,
	})
}

func (s *APIServer) handleSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	sessions := s.agentLoop.GetSessionManager().ListSessions()
	writeJSON(w, http.StatusOK, SessionList{Sessions: sessions})
}

func (s *APIServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	writeJSON(w, http.StatusOK, HealthResponse{
		Status:  "ok",
		Version: apiVersion,
	})
}

func (s *APIServer) handleSpec(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	w.Header().Set("Content-Type", "application/x-yaml")
	w.WriteHeader(http.StatusOK)
	w.Write(openapiSpec)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, message string) {
	writeJSON(w, code, ErrorResponse{Error: message, Code: code})
}
