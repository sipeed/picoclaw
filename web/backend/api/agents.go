package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	manager "github.com/sipeed/picoclaw/pkg/agent/manager"
)

// Type aliases to use manager types directly
type agent = manager.Agent
type agentListResponse = manager.AgentListResponse
type agentResponse struct {
	Agent *manager.Agent `json:"agent,omitempty"`
}

// agentCreateRequest uses same fields as manager.AgentCreateRequest
type agentCreateRequest = manager.AgentCreateRequest

// agentUpdateRequest uses same fields as manager.AgentUpdateRequest
type agentUpdateRequest = manager.AgentUpdateRequest

func (h *Handler) registerAgentRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/agents", h.handleListAgents)
	mux.HandleFunc("/api/agent", h.handleGetAgent)
	mux.HandleFunc("/api/agent/create", h.handleCreateAgent)
	mux.HandleFunc("/api/agent/update", h.handleUpdateAgent)
	mux.HandleFunc("/api/agent/delete", h.handleDeleteAgent)
	mux.HandleFunc("/api/agent/import", h.handleImportAgent)
}

func (h *Handler) handleListAgents(w http.ResponseWriter, r *http.Request) {
	mgr := manager.NewManager("")
	agents, err := mgr.ListAgents()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list agents: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(manager.AgentListResponse{Agents: agents})
}

func (h *Handler) handleGetAgent(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimSpace(r.URL.Query().Get("slug"))
	if slug == "" {
		http.Error(w, "slug is required", http.StatusBadRequest)
		return
	}

	mgr := manager.NewManager("")
	a, err := mgr.GetAgent(slug)
	if err != nil {
		http.Error(w, fmt.Sprintf("Agent not found: %s", slug), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agentResponse{Agent: a})
}

func (h *Handler) handleCreateAgent(w http.ResponseWriter, r *http.Request) {
	var req manager.AgentCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if req.SystemPrompt == "" {
		http.Error(w, "system_prompt is required", http.StatusBadRequest)
		return
	}
	if req.Model == "" {
		http.Error(w, "model is required", http.StatusBadRequest)
		return
	}
	mgr := manager.NewManager("")
	created, err := mgr.CreateAgent(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create agent: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agentResponse{Agent: created})
}

func (h *Handler) handleUpdateAgent(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimSpace(r.URL.Query().Get("slug"))
	if slug == "" {
		http.Error(w, "slug is required", http.StatusBadRequest)
		return
	}

	var req manager.AgentUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	mgr := manager.NewManager("")
	updated, err := mgr.UpdateAgent(slug, req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update agent: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agentResponse{Agent: updated})
}

func (h *Handler) handleDeleteAgent(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimSpace(r.URL.Query().Get("slug"))
	if slug == "" {
		http.Error(w, "slug is required", http.StatusBadRequest)
		return
	}

	mgr := manager.NewManager("")
	if err := mgr.DeleteAgent(slug); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete agent: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

type agentImportRequest struct {
	Content string `json:"content" binding:"required"`
}

func (h *Handler) handleImportAgent(w http.ResponseWriter, r *http.Request) {
	var req agentImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		http.Error(w, "content is required", http.StatusBadRequest)
		return
	}

	mgr := manager.NewManager("")
	created, err := mgr.ImportAgent(req.Content)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to import agent: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agentResponse{Agent: created})
}