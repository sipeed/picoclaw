package agent

import (
	"sync"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/routing"
	"github.com/sipeed/picoclaw/pkg/tools"
)

// AgentRegistry manages multiple agent instances and routes messages to them.
type AgentRegistry struct {
	cfg      *config.Config
	agents   map[string]*AgentInstance
	resolver *routing.RouteResolver
	mu       sync.RWMutex
}

// NewAgentRegistry creates a registry from config, instantiating all agents.
func NewAgentRegistry(
	cfg *config.Config,
	provider providers.LLMProvider,
) *AgentRegistry {
	registry := &AgentRegistry{
		cfg:      cfg,
		agents:   make(map[string]*AgentInstance),
		resolver: routing.NewRouteResolver(cfg),
	}

	agentConfigs := cfg.Agents.List
	if len(agentConfigs) == 0 {
		implicitAgent := &config.AgentConfig{
			ID:      "main",
			Default: true,
		}
		instance := NewAgentInstance(implicitAgent, &cfg.Agents.Defaults, cfg, provider)
		registry.agents["main"] = instance
		logger.InfoCF("agent", "Created implicit main agent (no agents.list configured)", nil)
	} else {
		for i := range agentConfigs {
			ac := &agentConfigs[i]
			id := routing.NormalizeAgentID(ac.ID)
			instance := NewAgentInstance(ac, &cfg.Agents.Defaults, cfg, provider)
			registry.agents[id] = instance
			logger.InfoCF("agent", "Registered agent",
				map[string]any{
					"agent_id":  id,
					"name":      ac.Name,
					"workspace": instance.Workspace,
					"model":     instance.Model,
				})
		}
	}

	for _, instance := range registry.agents {
		if instance.ContextBuilder != nil {
			instance.ContextBuilder.WithAgentDiscovery(instance.ID, registry.ListSpawnableAgents)
		}
	}

	return registry
}

// GetAgent returns the agent instance for a given ID.
func (r *AgentRegistry) GetAgent(agentID string) (*AgentInstance, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id := routing.NormalizeAgentID(agentID)
	agent, ok := r.agents[id]
	return agent, ok
}

// ResolveRoute determines which agent handles the normalized inbound context.
func (r *AgentRegistry) ResolveRoute(inbound bus.InboundContext) routing.ResolvedRoute {
	return r.resolver.ResolveRoute(inbound)
}

// ListAgentIDs returns all registered agent IDs.
func (r *AgentRegistry) ListAgentIDs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]string, 0, len(r.agents))
	for id := range r.agents {
		ids = append(ids, id)
	}
	return ids
}

func (r *AgentRegistry) allowedMCPServers() map[string]struct{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.agents) == 0 {
		return nil
	}

	if r.cfg == nil {
		return nil
	}

	union := make(map[string]struct{})
	serverCfgs := r.cfg.Tools.MCP.Servers
	for _, agent := range r.agents {
		if agent == nil {
			continue
		}
		if agent.MCPServerPolicy == nil {
			return nil
		}
		for serverName := range serverCfgs {
			if agent.AllowsMCPServer(serverName) {
				union[normalizeMCPServerName(serverName)] = struct{}{}
			}
		}
	}

	return union
}

// CanSpawnSubagent checks if parentAgentID is allowed to spawn targetAgentID.
func (r *AgentRegistry) CanSpawnSubagent(parentAgentID, targetAgentID string) bool {
	parent, ok := r.GetAgent(parentAgentID)
	if !ok {
		return false
	}
	return agentAllowsSubagent(parent, routing.NormalizeAgentID(targetAgentID))
}

func agentAllowsSubagent(parent *AgentInstance, targetNorm string) bool {
	if parent == nil || parent.Subagents == nil || parent.Subagents.AllowAgents == nil {
		return false
	}
	for _, allowed := range parent.Subagents.AllowAgents {
		if allowed == "*" {
			return true
		}
		if routing.NormalizeAgentID(allowed) == targetNorm {
			return true
		}
	}
	return false
}

func agentHasSpawnTool(agent *AgentInstance) bool {
	if agent == nil || agent.Tools == nil {
		return false
	}
	_, ok := agent.Tools.Get("spawn")
	return ok
}

// ForEachTool calls fn for every tool registered under the given name
// across all agents. This is useful for propagating dependencies (e.g.
// MediaStore) to tools after registry construction.
func (r *AgentRegistry) ForEachTool(name string, fn func(tools.Tool)) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, agent := range r.agents {
		if t, ok := agent.Tools.Get(name); ok {
			fn(t)
		}
	}
}

// Close releases resources held by all registered agents.
func (r *AgentRegistry) Close() {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, agent := range r.agents {
		if err := agent.Close(); err != nil {
			logger.WarnCF("agent", "Failed to close agent",
				map[string]any{"agent_id": agent.ID, "error": err.Error()})
		}
	}
}

// GetDefaultAgent returns the default agent instance.
func (r *AgentRegistry) GetDefaultAgent() *AgentInstance {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if id := r.defaultAgentIDLocked(); id != "" {
		if agent, ok := r.agents[id]; ok {
			return agent
		}
	}
	for id := range r.agents {
		return r.agents[id]
	}
	return nil
}
