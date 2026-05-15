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

	mcpCfg := filterMCPConfigServers(al.cfg.Tools.MCP, al.registry.allowedMCPServers())
	if mcpCfg.Servers == nil || len(mcpCfg.Servers) == 0 {
		logger.InfoCF(
			"agent",
			"No MCP servers selected after applying per-agent mcpServers allowlists",
			nil,
		)
		return nil
	}

	findValidServer := false
	for _, serverCfg := range mcpCfg.Servers {
		if serverCfg.Enabled {
			findValidServer = true
		}
	}
	if !findValidServer {
		logger.WarnCF("agent", "MCP is enabled but no valid servers are configured, skipping MCP initialization", nil)
		return nil
	}

	al.mcp.initOnce.Do(func() {
		mcpManager := mcp.NewManager(mcp.WithRuntimeEvents(al.runtimeEvents))

		defaultAgent := al.registry.GetDefaultAgent()
		workspacePath := al.cfg.WorkspacePath()
		if defaultAgent != nil && defaultAgent.Workspace != "" {
			workspacePath = defaultAgent.Workspace
		}

		if err := mcpManager.LoadFromMCPConfig(ctx, mcpCfg, workspacePath); err != nil {
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
			serverCfg := mcpCfg.Servers[serverName]
			registerAsHidden := serverIsDeferred(al.cfg.Tools.MCP.Discovery.Enabled, serverCfg)
			registeredToolsByAgent := make(map[string]map[string]struct{}, len(agentIDs))

			for _, tool := range conn.Tools {
				for _, agentID := range agentIDs {
					agent, ok := al.registry.GetAgent(agentID)
					if !ok {
						continue
					}
					if !agent.AllowsMCPServer(serverName) {
						logger.DebugCF("agent", "Skipped MCP tool registration by agent mcpServers allowlist",
							map[string]any{
								"agent_id": agentID,
								"server":   serverName,
								"tool":     tool.Name,
							})
						continue
					}

					mcpTool := tools.NewMCPTool(mcpManager, serverName, tool)
					toolName := mcpTool.Name()
					mcpTool.SetWorkspace(agent.Workspace)
					mcpTool.SetMaxInlineTextRunes(al.cfg.Tools.MCP.GetMaxInlineTextChars())
					mcpTool.SetEventPublisher(al.runtimeEvents)

					var registered bool
					if registerAsHidden {
						registered = registerHiddenToolIfAllowed(agent, mcpTool)
					} else {
						registered = registerToolIfAllowed(agent, mcpTool)
					}
					if !registered || !toolRegistryIncludes(agent.Tools, toolName) {
						continue
					}

					recordRegisteredMCPTool(registeredToolsByAgent, agentID, toolName)
					totalRegistrations++
					logger.DebugCF("agent", "Registered MCP tool",
						map[string]any{
							"agent_id": agentID,
							"server":   serverName,
							"tool":     tool.Name,
							"name":     toolName,
							"deferred": registerAsHidden,
						})
				}
			}

			for _, agentID := range agentIDs {
				agent, ok := al.registry.GetAgent(agentID)
				if !ok {
					continue
				}
				registerMCPServerPromptContributor(
					agentID,
					agent,
					serverName,
					len(registeredToolsByAgent[agentID]),
					registerAsHidden,
				)
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
				if !agentHasDiscoverableMCPServers(al.cfg, agent.MCPServerPolicy) {
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

func registerMCPServerPromptContributor(
	agentID string,
	agent *AgentInstance,
	serverName string,
	toolCount int,
	registerAsHidden bool,
) {
	if agent == nil || agent.ContextBuilder == nil || toolCount <= 0 {
		return
	}
	if err := agent.ContextBuilder.RegisterPromptContributor(mcpServerPromptContributor{
		serverName: serverName,
		toolCount:  toolCount,
		deferred:   registerAsHidden,
	}); err != nil {
		logger.WarnCF("agent", "Failed to register MCP prompt contributor",
			map[string]any{
				"agent_id": agentID,
				"server":   serverName,
				"error":    err.Error(),
			})
	}
}

func recordRegisteredMCPTool(
	registeredToolsByAgent map[string]map[string]struct{},
	agentID, toolName string,
) {
	if registeredToolsByAgent[agentID] == nil {
		registeredToolsByAgent[agentID] = make(map[string]struct{})
	}
	registeredToolsByAgent[agentID][toolName] = struct{}{}
}

func toolRegistryIncludes(registry *tools.ToolRegistry, name string) bool {
	if registry == nil {
		return false
	}
	return registry.HasRegistered(name)
}

func filterMCPConfigServers(
	mcpCfg config.MCPConfig,
	allowed map[string]struct{},
) config.MCPConfig {
	if allowed == nil {
		return mcpCfg
	}

	filtered := mcpCfg
	filtered.Servers = make(map[string]config.MCPServerConfig)
	normalizedAllowed := make(map[string]struct{}, len(allowed))
	for serverName := range allowed {
		name := normalizeMCPServerName(serverName)
		if name == "" {
			continue
		}
		normalizedAllowed[name] = struct{}{}
	}
	for serverName, serverCfg := range mcpCfg.Servers {
		if _, ok := normalizedAllowed[normalizeMCPServerName(serverName)]; ok {
			filtered.Servers[serverName] = serverCfg
		}
	}

	return filtered
}

func agentHasDiscoverableMCPServers(cfg *config.Config, allowed *PatternPolicy) bool {
	if cfg == nil || !cfg.Tools.MCP.Enabled || !cfg.Tools.MCP.Discovery.Enabled {
		return false
	}

	filtered := cfg.Tools.MCP
	if allowed != nil {
		filtered.Servers = make(map[string]config.MCPServerConfig)
		for serverName, serverCfg := range cfg.Tools.MCP.Servers {
			if toolAllowedByPolicy(allowed, normalizeMCPServerName(serverName)) {
				filtered.Servers[serverName] = serverCfg
			}
		}
	}
	for _, serverCfg := range filtered.Servers {
		if serverCfg.Enabled && serverIsDeferred(cfg.Tools.MCP.Discovery.Enabled, serverCfg) {
			return true
		}
	}

	return false
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
