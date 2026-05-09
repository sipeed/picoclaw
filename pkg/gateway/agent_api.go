package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/sipeed/picoclaw/pkg/agent/manager"
	"github.com/sipeed/picoclaw/pkg/agent/research"
	"github.com/sipeed/picoclaw/pkg/gateway/websocket"
	"github.com/sipeed/picoclaw/pkg/health"
	"github.com/sipeed/picoclaw/pkg/memory"
)

var agentManager *manager.Manager
var researchManager *research.Manager
var researchConfigStore *research.FileConfigStore
var wsHub *websocket.Hub

func init() {
	agentManager = manager.NewManager(getWorkspacePath() + "/agents")

	// Initialize research manager with the workspace directory
	researchDataDir := getWorkspacePath() + "/research"
	store, err := memory.NewJSONLStore(researchDataDir)
	if err != nil {
		// Continue with nil manager if store fails to initialize
		researchManager = nil
	} else {
		researchManager = research.NewManager(store)
	}

	// Initialize config store
	researchConfigStore = research.NewFileConfigStore(getWorkspacePath())

	// Initialize WebSocket hub
	wsHub = websocket.NewHub()
	go wsHub.Run()
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

// Research API Handlers

func handleResearchAgentsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if researchManager == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(research.AgentListResponse{Agents: []research.Agent{}})
		return
	}

	agents, err := researchManager.ListAgents()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(research.AgentListResponse{Agents: agents})
}

func handleResearchGraphList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if researchManager == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(research.NodeListResponse{Nodes: []research.Node{}})
		return
	}

	nodes, err := researchManager.ListNodes()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(research.NodeListResponse{Nodes: nodes})
}

func handleResearchReportsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if researchManager == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(research.ReportListResponse{Reports: []research.Report{}})
		return
	}

	reports, err := researchManager.ListReports()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(research.ReportListResponse{Reports: reports})
}

// handleResearchConfigGet returns the current research configuration
// handleResearchConfig handles both GET and PUT/POST for research configuration
func handleResearchConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if researchConfigStore == nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(research.DefaultConfig())
			return
		}

		cfg, err := researchConfigStore.GetResearchConfig()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cfg)

	case http.MethodPut, http.MethodPost:
		if researchConfigStore == nil {
			http.Error(w, "Config store not initialized", http.StatusInternalServerError)
			return
		}

		var cfg research.Config
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := researchConfigStore.SaveResearchConfig(cfg); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Broadcast config change to WebSocket clients
		wsHub.BroadcastConfigChange(websocket.ConfigChangePayload{
			Type:            cfg.Type,
			Depth:           cfg.Depth,
			RestrictToGraph: cfg.RestrictToGraph,
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cfg)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleResearchExport exports a research report as PDF or Markdown
func handleResearchExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	reportID := r.URL.Query().Get("id")
	format := r.URL.Query().Get("format")

	if reportID == "" {
		http.Error(w, "Report ID required", http.StatusBadRequest)
		return
	}

	if format == "" {
		format = "markdown"
	}

	if researchManager == nil {
		http.Error(w, "Research manager not initialized", http.StatusInternalServerError)
		return
	}

	report, err := researchManager.GetReport(reportID)
	if err != nil || report == nil {
		http.Error(w, "Report not found", http.StatusNotFound)
		return
	}

	// Generate export content
	var content string
	var contentType string
	var filename string

	switch format {
	case "pdf":
		// For PDF, we'll generate markdown and let the frontend handle conversion
		// In production, you'd use a PDF library like github.com/jung-kurt/gofpdf
		content = generateMarkdownReport(report)
		contentType = "text/plain"
		filename = fmt.Sprintf("%s.txt", report.ID)
	case "markdown":
		content = generateMarkdownReport(report)
		contentType = "text/markdown"
		filename = fmt.Sprintf("%s.md", report.ID)
	default:
		http.Error(w, "Unsupported format. Use 'pdf' or 'markdown'", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
	w.Write([]byte(content))
}

// generateMarkdownReport creates a Markdown representation of a research report
func generateMarkdownReport(report *research.Report) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s\n\n", report.Title))
	sb.WriteString("---\n\n")
	sb.WriteString(fmt.Sprintf("**Report ID:** %s\n", report.ID))
	sb.WriteString(fmt.Sprintf("**Status:** %s\n", report.Status))
	sb.WriteString(fmt.Sprintf("**Pages:** %d\n", report.Pages))
	sb.WriteString(fmt.Sprintf("**Words:** %d\n", report.Words))
	sb.WriteString(fmt.Sprintf("**Progress:** %d%%\n", report.Progress))

	if !report.CreatedAt.IsZero() {
		sb.WriteString(fmt.Sprintf("**Created:** %s\n", report.CreatedAt.Format("2006-01-02 15:04")))
	}
	if !report.UpdatedAt.IsZero() {
		sb.WriteString(fmt.Sprintf("**Last Updated:** %s\n", report.UpdatedAt.Format("2006-01-02 15:04")))
	}

	sb.WriteString("\n---\n\n")
	sb.WriteString("## Report Content\n\n")
	sb.WriteString("*Report content would be populated from the research data store.*\n")

	return sb.String()
}

// handleWebSocketUpgrade handles WebSocket connections for real-time updates
func handleWebSocketUpgrade(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	wsHub.ServeHTTP(w, r)
}

// RegisterAgentAPI registers the agent API routes with the health server.
func RegisterAgentAPI(s *health.Server) {
	s.HandleFunc("/api/agents", handleAgentsList)
	s.HandleFunc("/api/agent", handleAgentGet)
	s.HandleFunc("/api/agent/create", handleAgentCreate)
	s.HandleFunc("/api/agent/update", handleAgentUpdate)
	s.HandleFunc("/api/agent/delete", handleAgentDelete)
	s.HandleFunc("/api/agent/import", handleAgentImport)

	// Research API routes
	s.HandleFunc("/api/research/agents", handleResearchAgentsList)
	s.HandleFunc("/api/research/graph", handleResearchGraphList)
	s.HandleFunc("/api/research/reports", handleResearchReportsList)
	s.HandleFunc("/api/research/config", handleResearchConfig)
	s.HandleFunc("/api/research/export", handleResearchExport)

	// WebSocket for real-time updates
	s.HandleFunc("/ws/research", handleWebSocketUpgrade)
}
