package agent

import (
	"fmt"
	"slices"
	"sync"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/routing"
)

// AgentRegistry manages multiple agent instances and routes messages to them.
type AgentRegistry struct {
	agents         map[string]*AgentInstance
	resolver       *routing.RouteResolver
	defaultAgentID string
	mu             sync.RWMutex
}

// NewAgentRegistry creates a registry from config, instantiating all agents.
func NewAgentRegistry(
	cfg *config.Config,
	provider providers.LLMProvider,
) *AgentRegistry {
	registry := &AgentRegistry{
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
		registry.defaultAgentID = "main"
		logger.InfoCF("agent", "Created implicit main agent (no agents.list configured)", nil)
	} else {
		for i := range agentConfigs {
			ac := &agentConfigs[i]
			id := routing.NormalizeAgentID(ac.ID)
			instance := NewAgentInstance(ac, &cfg.Agents.Defaults, cfg, provider)
			registry.agents[id] = instance
			if ac.Default && registry.defaultAgentID == "" {
				registry.defaultAgentID = id
			}
			logger.InfoCF("agent", "Registered agent",
				map[string]any{
					"agent_id":  id,
					"name":      ac.Name,
					"workspace": instance.Workspace,
					"model":     instance.Model,
				})
		}
	}

	if registry.defaultAgentID == "" {
		ids := make([]string, 0, len(registry.agents))
		for id := range registry.agents {
			ids = append(ids, id)
		}
		slices.Sort(ids)
		if len(ids) > 0 {
			registry.defaultAgentID = ids[0]
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

// ResolveRoute determines which agent handles the message.
func (r *AgentRegistry) ResolveRoute(input routing.RouteInput) routing.ResolvedRoute {
	return r.resolver.ResolveRoute(input)
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

// CanSpawnSubagent checks if parentAgentID is allowed to spawn targetAgentID.
func (r *AgentRegistry) CanSpawnSubagent(parentAgentID, targetAgentID string) bool {
	parent, ok := r.GetAgent(parentAgentID)
	if !ok {
		return false
	}
	if parent.Subagents == nil || parent.Subagents.AllowAgents == nil {
		return false
	}
	targetNorm := routing.NormalizeAgentID(targetAgentID)
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

// GetDefaultAgent returns the default agent instance.
func (r *AgentRegistry) GetDefaultAgent() *AgentInstance {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.defaultAgentID != "" {
		if agent, ok := r.agents[r.defaultAgentID]; ok {
			return agent
		}
	}
	if agent, ok := r.agents["main"]; ok {
		return agent
	}
	ids := make([]string, 0, len(r.agents))
	for id := range r.agents {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	if len(ids) > 0 {
		return r.agents[ids[0]]
	}
	return nil
}

// SwitchModel hot-swaps provider and effective model at runtime.
// It updates all agents that currently use oldModel to newModel, and refreshes
// provider + fallback candidates atomically by replacing agent pointers.
func (r *AgentRegistry) SwitchModel(
	cfg *config.Config,
	oldModel string,
	newModel string,
	newProvider providers.LLMProvider,
) error {
	r.mu.Lock()
	if len(r.agents) == 0 {
		r.mu.Unlock()
		return fmt.Errorf("no agents registered")
	}

	var oldProvider providers.LLMProvider
	for id, agent := range r.agents {
		if oldProvider == nil {
			oldProvider = agent.Provider
		}

		updated := *agent
		if updated.Model == oldModel {
			updated.Model = newModel
		}
		updated.Provider = newProvider
		updated.Candidates = ResolveCandidatesForModel(cfg, cfg.Agents.Defaults.Provider, updated.Model, updated.Fallbacks)
		r.agents[id] = &updated
	}
	r.mu.Unlock()

	if oldProvider != nil && oldProvider != newProvider {
		if cp, ok := oldProvider.(providers.StatefulProvider); ok {
			cp.Close()
		}
	}

	return nil
}
