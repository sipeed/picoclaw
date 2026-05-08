package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/memory"
	"github.com/sipeed/picoclaw/pkg/seahorse"
)

func (h *Handler) registerResearchRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/research/agents", h.handleListResearchAgents)
	mux.HandleFunc("PUT /api/research/agents/{id}/toggle", h.handleToggleResearchAgent)
	mux.HandleFunc("GET /api/research/graph", h.handleListResearchGraph)
	mux.HandleFunc("PUT /api/research/graph/nodes", h.handleUpdateResearchGraph)
	mux.HandleFunc("GET /api/research/reports", h.handleListResearchReports)
	mux.HandleFunc("PUT /api/research/reports", h.handleUpdateResearchReport)
	mux.HandleFunc("PUT /api/research/config", h.handleUpdateResearchConfig)
}

type researchAgentResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Active   bool   `json:"active"`
	Progress int    `json:"progress"`
	RAM      string `json:"ram"`
	Type     string `json:"type"`
}

type researchGraphResponse struct {
	Nodes []seahorse.ResearchGraphNode `json:"nodes"`
}

type researchReportResponse struct {
	Reports []memory.ResearchReport `json:"reports"`
}

func (h *Handler) handleListResearchAgents(w http.ResponseWriter, r *http.Request) {
	agents := []researchAgentResponse{
		{ID: agent.ResearchAgentLiterature, Name: "Literature Analyzer", Active: true, Progress: 94, RAM: "2.8M", Type: "research"},
		{ID: agent.ResearchAgentExtractor, Name: "Data Extractor", Active: true, Progress: 87, RAM: "3.2M", Type: "research"},
		{ID: agent.ResearchAgentValidator, Name: "Fact Validator", Active: true, Progress: 76, RAM: "2.1M", Type: "research"},
		{ID: agent.ResearchAgentSynthesizer, Name: "Synthesizer", Active: true, Progress: 65, RAM: "4.1M", Type: "research"},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agents)
}

func (h *Handler) handleToggleResearchAgent(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.PathValue("id"), "")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "toggled", "id": id})
}

func (h *Handler) handleListResearchGraph(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, `{"error": "failed to load config"}`, http.StatusInternalServerError)
		return
	}
	store := seahorse.NewStore(cfg)
	defer store.Close()

	nodes, err := store.ListResearchNodes()
	if err != nil {
		http.Error(w, `{"error": "failed to list nodes"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(researchGraphResponse{Nodes: nodes})
}

func (h *Handler) handleUpdateResearchGraph(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func (h *Handler) handleListResearchReports(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, `{"error": "failed to load config"}`, http.StatusInternalServerError)
		return
	}
	store := memory.NewStore(cfg)
	defer store.Close()

	reports, err := store.ListResearchReports()
	if err != nil {
		http.Error(w, `{"error": "failed to list reports"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(researchReportResponse{Reports: reports})
}

func (h *Handler) handleUpdateResearchReport(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func (h *Handler) handleUpdateResearchConfig(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "config updated"})
}