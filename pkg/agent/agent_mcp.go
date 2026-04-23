// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

import (
	"context"
	"fmt"
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

func (r *mcpRuntime) reset() *mcp.Manager {
	r.mu.Lock()
	manager := r.manager
	r.manager = nil
	r.initErr = nil
	r.initOnce = sync.Once{}
	r.mu.Unlock()
	return manager
}

func (r *mcpRuntime) setManager(manager *mcp.Manager) {
	r.mu.Lock()
	r.manager = manager
	r.initErr = nil
	r.mu.Unlock()
}

func (r *mcpRuntime) setInitErr(err error) {
	r.mu.Lock()
	r.initErr = err
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
		mcpManager := mcp.NewManager()

		defaultAgent := al.registry.GetDefaultAgent()
		workspacePath := al.cfg.WorkspacePath()
		if defaultAgent != nil && defaultAgent.Workspace != "" {
			workspacePath = defaultAgent.Workspace
		}

		if err := mcpManager.LoadFromMCPConfig(ctx, al.cfg.Tools.MCP, workspacePath); err != nil {
			al.mcp.setInitErr(fmt.Errorf("failed to load MCP servers: %w", err))
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

		// Register MCP tools for all agents
		servers := mcpManager.GetServers()
		uniqueTools := 0
		totalRegistrations := 0
		agentIDs := al.registry.ListAgentIDs()
		agentCount := len(agentIDs)

		for serverName, conn := range servers {
			uniqueTools += len(conn.Tools)

			// Determine whether this server's tools should be deferred (hidden).
			// Per-server "deferred" field takes precedence over the global Discovery.Enabled.
			serverCfg := al.cfg.Tools.MCP.Servers[serverName]
			registerAsHidden := serverIsDeferred(al.cfg.Tools.MCP.Discovery.Enabled, serverCfg)

			for _, tool := range conn.Tools {
				for _, agentID := range agentIDs {
					agent, ok := al.registry.GetAgent(agentID)
					if !ok {
						continue
					}

					mcpTool := tools.NewMCPTool(mcpManager, serverName, tool)
					mcpTool.SetWorkspace(agent.Workspace)
					mcpTool.SetMaxInlineTextRunes(al.cfg.Tools.MCP.GetMaxInlineTextChars())

					if registerAsHidden {
						agent.Tools.RegisterHidden(mcpTool)
					} else {
						agent.Tools.Register(mcpTool)
					}

					totalRegistrations++
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
		logger.InfoCF("agent", "MCP tools registered successfully",
			map[string]any{
				"server_count":        len(servers),
				"unique_tools":        uniqueTools,
				"total_registrations": totalRegistrations,
				"agent_count":         agentCount,
			})

		// Initializes Discovery Tools only if enabled by configuration
		if al.cfg.Tools.MCP.Enabled && al.cfg.Tools.MCP.Discovery.Enabled {
			useBM25 := al.cfg.Tools.MCP.Discovery.UseBM25
			useRegex := al.cfg.Tools.MCP.Discovery.UseRegex

			// Fail fast: If discovery is enabled but no search method is turned on
			if !useBM25 && !useRegex {
				al.mcp.setInitErr(fmt.Errorf(
					"tool discovery is enabled but neither 'use_bm25' nor 'use_regex' is set to true in the configuration",
				))
				if closeErr := mcpManager.Close(); closeErr != nil {
					logger.ErrorCF("agent", "Failed to close MCP manager",
						map[string]any{
							"error": closeErr.Error(),
						})
				}
				return
			}

			ttl := al.cfg.Tools.MCP.Discovery.TTL
			if ttl <= 0 {
				ttl = 5 // Default value
			}

			maxSearchResults := al.cfg.Tools.MCP.Discovery.MaxSearchResults
			if maxSearchResults <= 0 {
				maxSearchResults = 5 // Default value
			}

			logger.InfoCF("agent", "Initializing tool discovery", map[string]any{
				"bm25": useBM25, "regex": useRegex, "ttl": ttl, "max_results": maxSearchResults,
			})

			for _, agentID := range agentIDs {
				agent, ok := al.registry.GetAgent(agentID)
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
