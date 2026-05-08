package api

import (
	"encoding/json"
	"net/http"
	"strings"

	picoclawagent "github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/seahorse"
	"github.com/sipeed/picoclaw/pkg/memory"
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
		{ID: picoclawagent.ResearchAgentLiterature, Name: "Literature Analyzer", Active: true, Progress: 94, RAM: "2.8M", Type: "research"},
		{ID: picoclawagent.ResearchAgentExtractor, Name: "Data Extractor", Active: true, Progress: 87, RAM: "3.2M", Type: "research"},
		{ID: picoclawagent.ResearchAgentValidator, Name: "Fact Validator", Active: true, Progress: 76, RAM: "2.1M", Type: "research"},
		{ID: picoclawagent.ResearchAgentSynthesizer, Name: "Synthesizer", Active: true, Progress: 65, RAM: "4.1M", Type: "research"},
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
	// TODO: Integrate with seahorse store when properly configured
	nodes := []seahorse.ResearchGraphNode{
		{Name: "Neural Networks", Abbr: "NN", X: 150, Y: 80},
		{Name: "Transformers", Abbr: "TFM", X: 150, Y: 120},
		{Name: "LLM Optimization", Abbr: "LLM", X: 150, Y: 160},
		{Name: "Edge Computing", Abbr: "EDG", X: 150, Y: 210},
		{Name: "Multi-Agent Systems", Abbr: "MAS", X: 150, Y: 260},
		{Name: "Vision Models", Abbr: "VM", X: 150, Y: 310},
		{Name: "RAG Systems", Abbr: "RAG", X: 650, Y: 80},
		{Name: "Knowledge Graphs", Abbr: "KG", X: 650, Y: 150},
		{Name: "Agent Architecture", Abbr: "AA", X: 650, Y: 220},
		{Name: "Fine-tuning Methods", Abbr: "FTM", X: 650, Y: 290},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(researchGraphResponse{Nodes: nodes})
}

func (h *Handler) handleUpdateResearchGraph(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func (h *Handler) handleListResearchReports(w http.ResponseWriter, r *http.Request) {
	// TODO: Integrate with memory store when properly configured
	reports := []memory.ResearchReport{
		{ID: "1", Title: "AI trends 2026", Pages: 18, Words: 5400, Status: "in-progress", Progress: 75},
		{ID: "2", Title: "Quantum computing", Pages: 42, Words: 12600, Status: "complete"},
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