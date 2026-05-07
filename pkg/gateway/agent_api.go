package gateway

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/sipeed/picoclaw/pkg/agent/manager"
	"github.com/sipeed/picoclaw/pkg/health"
)

var agentManager *manager.Manager

func init() {
	agentManager = manager.NewManager(getWorkspacePath() + "/agents")
}

func getWorkspacePath() string {
	if w := os.Getenv("PICOCLAW_WORKSPACE"); w != "" {
		return w
	}
	home, _ := os.UserHomeDir()
	return home + "/.picoclaw/workspace"
}

// Agent API Handlers

func handleAgentsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	agents, err := agentManager.ListAgents()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(manager.AgentListResponse{Agents: agents})
}

func handleAgentGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	slug := r.URL.Query().Get("slug")
	if slug == "" {
		http.Error(w, "slug parameter required", http.StatusBadRequest)
		return
	}

	agent, err := agentManager.GetAgent(slug)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agent)
}

func handleAgentCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req manager.AgentCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	agent, err := agentManager.CreateAgent(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agent)
}

func handleAgentUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	slug := r.URL.Query().Get("slug")
	if slug == "" {
		http.Error(w, "slug parameter required", http.StatusBadRequest)
		return
	}

	var req manager.AgentUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	agent, err := agentManager.UpdateAgent(slug, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agent)
}

func handleAgentDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	slug := r.URL.Query().Get("slug")
	if slug == "" {
		http.Error(w, "slug parameter required", http.StatusBadRequest)
		return
	}

	if err := agentManager.DeleteAgent(slug); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Agent deleted"})
}

func handleAgentImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	agent, err := agentManager.ImportAgent(req.Content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agent)
}

// RegisterAgentAPI registers the agent API routes with the health server.
func RegisterAgentAPI(s *health.Server) {
	s.HandleFunc("/api/agents", handleAgentsList)
	s.HandleFunc("/api/agent", handleAgentGet)
	s.HandleFunc("/api/agent/create", handleAgentCreate)
	s.HandleFunc("/api/agent/update", handleAgentUpdate)
	s.HandleFunc("/api/agent/delete", handleAgentDelete)
	s.HandleFunc("/api/agent/import", handleAgentImport)
}
