package agent

import (
	"fmt"
	"strings"
	"sync"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/routing"
)

// AgentRegistry manages multiple agent instances and routes messages to them.
type AgentRegistry struct {
	agents   map[string]*AgentInstance
	cfg      *config.Config
	resolver *routing.RouteResolver
	mu       sync.RWMutex
}

// NewAgentRegistry creates a registry from config, instantiating all agents.
func NewAgentRegistry(
	cfg *config.Config,
	provider providers.LLMProvider,
) *AgentRegistry {
	registry := &AgentRegistry{
		agents:   make(map[string]*AgentInstance),
		cfg:      cfg,
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
	return r.defaultAgentLocked()
}

// GetDefaultAgentModel returns the active model name of the default agent.
func (r *AgentRegistry) GetDefaultAgentModel() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	agent := r.defaultAgentLocked()
	if agent == nil {
		return ""
	}
	return agent.Model
}

// SwitchDefaultAgentModel switches the default agent to a named model from config.model_list.
// It returns old and new runtime model IDs.
func (r *AgentRegistry) SwitchDefaultAgentModel(modelName string) (string, string, error) {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return "", "", fmt.Errorf("model name is required")
	}

	if r.cfg == nil {
		return "", "", fmt.Errorf("registry config not available")
	}

	modelCfg, err := r.cfg.GetModelConfig(modelName)
	if err != nil {
		return "", "", err
	}

	resolved := *modelCfg
	if resolved.Workspace == "" {
		resolved.Workspace = r.cfg.WorkspacePath()
	}

	provider, modelID, err := providers.CreateProviderFromConfig(&resolved)
	if err != nil {
		return "", "", err
	}

	protocol, _ := providers.ExtractProtocol(resolved.Model)

	r.mu.Lock()
	defer r.mu.Unlock()
	agent := r.defaultAgentLocked()
	if agent == nil {
		return "", "", fmt.Errorf("no default agent configured")
	}

	oldModel := agent.Model
	agent.Provider = provider
	agent.Model = modelID
	agent.Candidates = []providers.FallbackCandidate{{
		Provider: protocol,
		Model:    modelID,
	}}

	return oldModel, modelID, nil
}

func (r *AgentRegistry) defaultAgentLocked() *AgentInstance {
	if agent, ok := r.agents["main"]; ok {
		return agent
	}
	for _, agent := range r.agents {
		return agent
	}
	return nil
}
