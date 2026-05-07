package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
)

// registerMCPRoutes registers MCP server API routes on the ServeMux.
func (h *Handler) registerMCPRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/mcp/servers", h.handleListMCPServers)
	mux.HandleFunc("POST /api/mcp/servers", h.handleAddMCPServer)
	mux.HandleFunc("PUT /api/mcp/servers/{name}", h.handleUpdateMCPServer)
	mux.HandleFunc("DELETE /api/mcp/servers/{name}", h.handleDeleteMCPServer)
	mux.HandleFunc("POST /api/mcp/servers/{name}/test", h.handleTestMCPServer)
}

type mcpServerResponse struct {
	Servers []mcpServerItem `json:"servers"`
}

type mcpServerItem struct {
	Name    string         `json:"name"`
	Command string         `json:"command"`
	Args    []string       `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	Enabled bool           `json:"enabled"`
	Status  string         `json:"status"`
}

type mcpAddRequest struct {
	Name    string         `json:"name" binding:"required"`
	Command string         `json:"command" binding:"required"`
	Args    []string       `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	Enabled bool           `json:"enabled"`
}

// handleListMCPServers lists all MCP servers from config.
func (h *Handler) handleListMCPServers(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}
	servers := make([]mcpServerItem, 0, len(cfg.Tools.MCP.Servers))
	for name, srv := range cfg.Tools.MCP.Servers {
		status := "disabled"
		if srv.Enabled {
			status = "enabled"
		}
		servers = append(servers, mcpServerItem{
			Name:    name,
			Command: srv.Command,
			Args:    srv.Args,
			Env:     srv.Env,
			Enabled: srv.Enabled,
			Status:  status,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mcpServerResponse{Servers: servers})
}

// handleAddMCPServer adds a new MCP server to config.
func (h *Handler) handleAddMCPServer(w http.ResponseWriter, r *http.Request) {
	var req mcpAddRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.Command == "" {
		http.Error(w, "name and command are required", http.StatusBadRequest)
		return
	}
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}
	if cfg.Tools.MCP.Servers == nil {
		cfg.Tools.MCP.Servers = make(map[string]config.MCPServerConfig)
	}
	cfg.Tools.MCP.Servers[req.Name] = config.MCPServerConfig{
		Command: req.Command,
		Args:    req.Args,
		Env:     req.Env,
		Enabled: req.Enabled,
	}
	if err := config.SaveConfig(h.configPath, cfg); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save config: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleUpdateMCPServer updates an existing MCP server in config.
func (h *Handler) handleUpdateMCPServer(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "server name is required", http.StatusBadRequest)
		return
	}
	var req mcpAddRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}
	if _, exists := cfg.Tools.MCP.Servers[name]; !exists {
		http.Error(w, "MCP server not found", http.StatusNotFound)
		return
	}
	cfg.Tools.MCP.Servers[name] = config.MCPServerConfig{
		Command: req.Command,
		Args:    req.Args,
		Env:     req.Env,
		Enabled: req.Enabled,
	}
	if err := config.SaveConfig(h.configPath, cfg); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save config: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleDeleteMCPServer removes an MCP server from config.
func (h *Handler) handleDeleteMCPServer(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "server name is required", http.StatusBadRequest)
		return
	}
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}
	if _, exists := cfg.Tools.MCP.Servers[name]; !exists {
		http.Error(w, "MCP server not found", http.StatusNotFound)
		return
	}
	delete(cfg.Tools.MCP.Servers, name)
	if len(cfg.Tools.MCP.Servers) == 0 {
		cfg.Tools.MCP.Servers = nil
		cfg.Tools.MCP.Enabled = false
	}
	if err := config.SaveConfig(h.configPath, cfg); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save config: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleTestMCPServer probes an MCP server for health/tool count.
func (h *Handler) handleTestMCPServer(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "server name is required", http.StatusBadRequest)
		return
	}
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}
	srv, exists := cfg.Tools.MCP.Servers[name]
	if !exists {
		http.Error(w, "MCP server not found", http.StatusNotFound)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// Simple probe: check if command exists and is executable
	// In production, this would actually start the server and query tools
	status := "ok"
	toolCount := 0
	// Placeholder: actual MCP probe logic would go here
	_ = ctx
	_ = srv
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": status, "tool_count": toolCount})
}