// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

import (
	"context"
	"sync"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/mcp"
	"github.com/sipeed/picoclaw/pkg/tools"
)

type mcpRuntime struct {
	initOnce sync.Once
	mu       sync.Mutex
	manager  *mcp.Manager
	initErr  error
}

func (r *mcpRuntime) setManager(manager *mcp.Manager) {
	r.mu.Lock()
	r.manager = manager
	r.initErr = nil
	r.mu.Unlock()
}

func (r *mcpRuntime) getInitErr() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.initErr
}

func (r *mcpRuntime) takeManager() *mcp.Manager {
	r.mu.Lock()
	defer r.mu.Unlock()
	manager := r.manager
	r.manager = nil
	return manager
}

func (r *mcpRuntime) hasManager() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.manager != nil
}

func (r *mcpRuntime) getManager() *mcp.Manager {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.manager
}

// EnsureMCPInitialized loads MCP servers/tools once so both Run() and direct
// agent mode share the same initialization path.
func (al *AgentLoop) EnsureMCPInitialized(ctx context.Context) error {
	if !al.cfg.Tools.IsToolEnabled("mcp") {
		return nil
	}

	if len(al.cfg.Tools.MCP.Servers) == 0 {
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
		mcpManager := mcp.NewManager()

		defaultAgent := al.registry.GetDefaultAgent()
		workspacePath := al.cfg.WorkspacePath()
		if defaultAgent != nil && defaultAgent.Workspace != "" {
			workspacePath = defaultAgent.Workspace
		}

		if err := mcpManager.LoadFromMCPConfig(ctx, al.cfg.Tools.MCP, workspacePath); err != nil {
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

		al.mcp.setManager(mcpManager)

		// Register MCP and discovery tools for all currently known agents
		agentIDs := al.registry.ListAgentIDs()
		for _, agentID := range agentIDs {
			agent, ok := al.registry.GetAgent(agentID)
			if !ok {
				continue
			}
			al.RegisterMCPToolsToAgent(agentID, agent)
		}

		logger.InfoCF("agent", "MCP initialization complete",
			map[string]any{
				"server_count": len(mcpManager.GetServers()),
				"agent_count":  len(agentIDs),
			})
	})

	return al.mcp.getInitErr()
}

// RegisterMCPToolsToAgent registers all currently active MCP tools and discovery tools to the given agent instance.
func (al *AgentLoop) RegisterMCPToolsToAgent(agentID string, agent *AgentInstance) {
	if !al.cfg.Tools.MCP.Enabled {
		return
	}

	mcpManager := al.mcp.getManager()
	if mcpManager == nil {
		return
	}

	// 1. Register MCP server tools
	servers := mcpManager.GetServers()
	uniqueTools := 0
	totalRegistrations := 0

	for serverName, conn := range servers {
		uniqueTools += len(conn.Tools)

		serverCfg := al.cfg.Tools.MCP.Servers[serverName]
		registerAsHidden := serverIsDeferred(al.cfg.Tools.MCP.Discovery.Enabled, serverCfg)

		for _, tool := range conn.Tools {
			mcpTool := tools.NewMCPTool(mcpManager, serverName, tool)
			mcpTool.SetWorkspace(agent.Workspace)
			mcpTool.SetMaxInlineTextRunes(al.cfg.Tools.MCP.GetMaxInlineTextChars())

			if registerAsHidden {
				agent.Tools.RegisterHidden(mcpTool)
			} else {
				agent.Tools.Register(mcpTool)
			}
			totalRegistrations++
		}
	}

	if totalRegistrations > 0 {
		logger.DebugCF("agent", "Registered MCP tools to agent",
			map[string]any{
				"agent_id":     agentID,
				"server_count": len(servers),
				"tool_count":   totalRegistrations,
			})
	}

	// 2. Initializes Discovery Tools only if enabled by configuration
	if al.cfg.Tools.MCP.Discovery.Enabled {
		useBM25 := al.cfg.Tools.MCP.Discovery.UseBM25
		useRegex := al.cfg.Tools.MCP.Discovery.UseRegex

		if useBM25 || useRegex {
			ttl := al.cfg.Tools.MCP.Discovery.TTL
			if ttl <= 0 {
				ttl = 5
			}
			maxSearchResults := al.cfg.Tools.MCP.Discovery.MaxSearchResults
			if maxSearchResults <= 0 {
				maxSearchResults = 5
			}

			if useRegex {
				agent.Tools.Register(tools.NewRegexSearchTool(agent.Tools, ttl, maxSearchResults))
			}
			if useBM25 {
				agent.Tools.Register(tools.NewBM25SearchTool(agent.Tools, ttl, maxSearchResults))
			}

			logger.DebugCF("agent", "Initialized tool discovery for agent", map[string]any{
				"agent_id": agentID, "bm25": useBM25, "regex": useRegex,
			})
		}
	}
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
