// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/mcp"
	"github.com/sipeed/picoclaw/pkg/tools"
)

type mcpRuntime struct {
	initOnce  sync.Once
	mu        sync.Mutex
	attempted bool
	manager   mcpController
	caller    tools.MCPManager
	servers   map[string]*mcp.ServerConnection
	initErr   error
	lastErr   error
}

type mcpController interface {
	tools.MCPManager
	LoadFromMCPConfig(ctx context.Context, mcpCfg config.MCPConfig, workspacePath string) error
	GetServers() map[string]*mcp.ServerConnection
	Close() error
}

var newMCPController = func() mcpController {
	return mcp.NewManager()
}

func (r *mcpRuntime) setManager(manager mcpController) {
	r.mu.Lock()
	r.attempted = true
	r.manager = manager
	r.caller = manager
	if manager != nil {
		r.servers = manager.GetServers()
	} else {
		r.servers = nil
	}
	r.initErr = nil
	r.lastErr = nil
	r.mu.Unlock()
}

func (r *mcpRuntime) setInitErr(err error) {
	r.mu.Lock()
	r.attempted = true
	r.initErr = err
	r.lastErr = err
	r.mu.Unlock()
}

func (r *mcpRuntime) setStatusErr(err error) {
	r.mu.Lock()
	r.attempted = true
	r.lastErr = err
	r.mu.Unlock()
}

func (r *mcpRuntime) getInitErr() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.initErr
}

func (r *mcpRuntime) takeManager() mcpController {
	r.mu.Lock()
	defer r.mu.Unlock()
	manager := r.manager
	r.manager = nil
	r.caller = nil
	r.servers = nil
	return manager
}

func (r *mcpRuntime) replaceForReload(manager mcpController, attempted bool) mcpController {
	r.mu.Lock()
	defer r.mu.Unlock()

	oldManager := r.manager
	r.initOnce = sync.Once{}
	r.attempted = attempted
	r.manager = manager
	r.initErr = nil
	r.lastErr = nil
	if manager != nil {
		r.caller = manager
		r.servers = manager.GetServers()
	} else {
		r.caller = nil
		r.servers = nil
	}
	if attempted {
		r.initOnce.Do(func() {})
	}
	return oldManager
}

func (r *mcpRuntime) hasManager() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.manager != nil
}

type mcpStatusSnapshot struct {
	attempted bool
	lastErr   error
	servers   map[string]*mcp.ServerConnection
}

func (r *mcpRuntime) statusSnapshot() mcpStatusSnapshot {
	r.mu.Lock()
	defer r.mu.Unlock()

	servers := make(map[string]*mcp.ServerConnection, len(r.servers))
	for name, conn := range r.servers {
		servers[name] = conn
	}

	return mcpStatusSnapshot{
		attempted: r.attempted,
		lastErr:   r.lastErr,
		servers:   servers,
	}
}

func (r *mcpRuntime) registrationSnapshot() (tools.MCPManager, map[string]*mcp.ServerConnection) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.caller == nil || len(r.servers) == 0 {
		return nil, nil
	}

	servers := make(map[string]*mcp.ServerConnection, len(r.servers))
	for name, conn := range r.servers {
		servers[name] = conn
	}
	return r.caller, servers
}

func formatMCPStatus(cfg *config.Config, snap mcpStatusSnapshot) string {
	if cfg == nil {
		return "MCP status unavailable: config not loaded."
	}

	if !cfg.Tools.IsToolEnabled("mcp") || !cfg.Tools.MCP.Enabled {
		return "MCP is disabled."
	}

	configured := cfg.Tools.MCP.Servers
	if len(configured) == 0 {
		return "MCP is enabled, but no servers are configured."
	}

	lines := []string{
		fmt.Sprintf("MCP Enabled: yes"),
		fmt.Sprintf("Initialization Attempted: %s", yesNo(snap.attempted)),
		fmt.Sprintf("Connected Servers: %d/%d", len(snap.servers), countEnabledMCPServers(configured)),
	}

	if snap.lastErr != nil {
		lines = append(lines, fmt.Sprintf("Last Init Error: %s", snap.lastErr.Error()))
	}

	names := make([]string, 0, len(configured))
	for name, serverCfg := range configured {
		if !serverCfg.Enabled {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)

	if len(names) == 0 {
		lines = append(lines, "No enabled MCP servers.")
		return strings.Join(lines, "\n")
	}

	lines = append(lines, "", "Servers:")
	for _, name := range names {
		serverCfg := configured[name]
		conn, connected := snap.servers[name]
		toolCount := 0
		if conn != nil {
			toolCount = len(conn.Tools)
		}
		lines = append(lines, fmt.Sprintf(
			"- %s: %s, transport=%s, tools=%d%s",
			name,
			connectionStatusLabel(connected),
			mcpTransportLabel(serverCfg),
			toolCount,
			mcpEndpointSummary(serverCfg),
		))
	}

	return strings.Join(lines, "\n")
}

func countEnabledMCPServers(servers map[string]config.MCPServerConfig) int {
	count := 0
	for _, serverCfg := range servers {
		if serverCfg.Enabled {
			count++
		}
	}
	return count
}

func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func connectionStatusLabel(connected bool) string {
	if connected {
		return "connected"
	}
	return "not connected"
}

func mcpTransportLabel(serverCfg config.MCPServerConfig) string {
	transportType := strings.TrimSpace(serverCfg.Type)
	if transportType != "" {
		return transportType
	}
	if strings.TrimSpace(serverCfg.URL) != "" {
		return "sse"
	}
	if strings.TrimSpace(serverCfg.Command) != "" {
		return "stdio"
	}
	return "unknown"
}

func mcpEndpointSummary(serverCfg config.MCPServerConfig) string {
	if url := strings.TrimSpace(serverCfg.URL); url != "" {
		return fmt.Sprintf(", url=%s", url)
	}
	if cmd := strings.TrimSpace(serverCfg.Command); cmd != "" {
		return fmt.Sprintf(", command=%s", cmd)
	}
	return ""
}

type mcpRegistrationSummary struct {
	serverCount        int
	uniqueTools        int
	totalRegistrations int
	agentCount         int
}

func registerMCPToolsOnRegistry(
	registry *AgentRegistry,
	cfg *config.Config,
	caller tools.MCPManager,
	servers map[string]*mcp.ServerConnection,
) mcpRegistrationSummary {
	if registry == nil || cfg == nil || caller == nil || len(servers) == 0 {
		return mcpRegistrationSummary{}
	}
	if !cfg.Tools.IsToolEnabled("mcp") || !cfg.Tools.MCP.Enabled {
		return mcpRegistrationSummary{}
	}

	agentIDs := registry.ListAgentIDs()
	summary := mcpRegistrationSummary{
		serverCount: len(servers),
		agentCount:  len(agentIDs),
	}

	for serverName, conn := range servers {
		serverCfg, ok := cfg.Tools.MCP.Servers[serverName]
		if !ok || !serverCfg.Enabled {
			continue
		}

		summary.uniqueTools += len(conn.Tools)
		registerAsHidden := serverIsDeferred(cfg.Tools.MCP.Discovery.Enabled, serverCfg)

		for _, tool := range conn.Tools {
			for _, agentID := range agentIDs {
				agent, ok := registry.GetAgent(agentID)
				if !ok {
					continue
				}

				mcpTool := tools.NewMCPTool(caller, serverName, tool)
				mcpTool.SetWorkspace(agent.Workspace)
				mcpTool.SetMaxInlineTextRunes(cfg.Tools.MCP.GetMaxInlineTextChars())

				if registerAsHidden {
					agent.Tools.RegisterHidden(mcpTool)
				} else {
					agent.Tools.Register(mcpTool)
				}

				summary.totalRegistrations++
				logger.DebugCF("agent", "Registered MCP tool",
					map[string]any{
						"agent_id": agentID,
						"server":   serverName,
						"tool":     tool.Name,
						"name":     mcpTool.Name(),
						"deferred": registerAsHidden,
					})
			}
		}
	}

	return summary
}

func registerMCPDiscoveryToolsOnRegistry(registry *AgentRegistry, cfg *config.Config) error {
	if registry == nil || cfg == nil || !cfg.Tools.MCP.Enabled || !cfg.Tools.MCP.Discovery.Enabled {
		return nil
	}

	useBM25 := cfg.Tools.MCP.Discovery.UseBM25
	useRegex := cfg.Tools.MCP.Discovery.UseRegex
	if !useBM25 && !useRegex {
		return fmt.Errorf(
			"tool discovery is enabled but neither 'use_bm25' nor 'use_regex' is set to true in the configuration",
		)
	}

	ttl := cfg.Tools.MCP.Discovery.TTL
	if ttl <= 0 {
		ttl = 5
	}

	maxSearchResults := cfg.Tools.MCP.Discovery.MaxSearchResults
	if maxSearchResults <= 0 {
		maxSearchResults = 5
	}

	logger.InfoCF("agent", "Initializing tool discovery", map[string]any{
		"bm25": useBM25, "regex": useRegex, "ttl": ttl, "max_results": maxSearchResults,
	})

	for _, agentID := range registry.ListAgentIDs() {
		agent, ok := registry.GetAgent(agentID)
		if !ok {
			continue
		}

		if useRegex {
			agent.Tools.Register(tools.NewRegexSearchTool(agent.Tools, ttl, maxSearchResults))
		}
		if useBM25 {
			agent.Tools.Register(tools.NewBM25SearchTool(agent.Tools, ttl, maxSearchResults))
		}
	}

	return nil
}

// ensureMCPInitialized loads MCP servers/tools once so both Run() and direct
// agent mode share the same initialization path.
func (al *AgentLoop) ensureMCPInitialized(ctx context.Context) error {
	if !al.cfg.Tools.IsToolEnabled("mcp") {
		return nil
	}

	if al.cfg.Tools.MCP.Servers == nil || len(al.cfg.Tools.MCP.Servers) == 0 {
		logger.WarnCF("agent", "MCP is enabled but no servers are configured, skipping MCP initialization", nil)
		return nil
	}

	findValidServer := false
	for _, serverCfg := range al.cfg.Tools.MCP.Servers {
		if serverCfg.Enabled {
			findValidServer = true
		}
	}
	if !findValidServer {
		logger.WarnCF("agent", "MCP is enabled but no valid servers are configured, skipping MCP initialization", nil)
		return nil
	}

	al.mcp.initOnce.Do(func() {
		mcpManager := newMCPController()

		defaultAgent := al.registry.GetDefaultAgent()
		workspacePath := al.cfg.WorkspacePath()
		if defaultAgent != nil && defaultAgent.Workspace != "" {
			workspacePath = defaultAgent.Workspace
		}

		if err := mcpManager.LoadFromMCPConfig(ctx, al.cfg.Tools.MCP, workspacePath); err != nil {
			al.mcp.setStatusErr(err)
			logger.WarnCF("agent", "Failed to load MCP servers, MCP tools will not be available",
				map[string]any{
					"error": err.Error(),
				})
			if closeErr := mcpManager.Close(); closeErr != nil {
				logger.ErrorCF("agent", "Failed to close MCP manager",
					map[string]any{
						"error": closeErr.Error(),
					})
			}
			return
		}

		servers := mcpManager.GetServers()
		summary := registerMCPToolsOnRegistry(al.registry, al.cfg, mcpManager, servers)
		logger.InfoCF("agent", "MCP tools registered successfully",
			map[string]any{
				"server_count":        summary.serverCount,
				"unique_tools":        summary.uniqueTools,
				"total_registrations": summary.totalRegistrations,
				"agent_count":         summary.agentCount,
			})

		if err := registerMCPDiscoveryToolsOnRegistry(al.registry, al.cfg); err != nil {
			al.mcp.setInitErr(err)
			if closeErr := mcpManager.Close(); closeErr != nil {
				logger.ErrorCF("agent", "Failed to close MCP manager",
					map[string]any{
						"error": closeErr.Error(),
					})
			}
			return
		}

		al.mcp.setManager(mcpManager)
	})

	return al.mcp.getInitErr()
}

// serverIsDeferred reports whether an MCP server's tools should be registered
// as hidden (deferred/discovery mode).
//
// The per-server Deferred field takes precedence over the global discoveryEnabled
// default. When Deferred is nil, discoveryEnabled is used as the fallback.
func serverIsDeferred(discoveryEnabled bool, serverCfg config.MCPServerConfig) bool {
	if !discoveryEnabled {
		return false
	}
	if serverCfg.Deferred != nil {
		return *serverCfg.Deferred
	}
	return true
}
