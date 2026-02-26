package agent

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/routing"
	"github.com/sipeed/picoclaw/pkg/session"
	"github.com/sipeed/picoclaw/pkg/tools"
)

// AgentInstance represents a fully configured agent with its own workspace,
// session manager, context builder, and tool registry.
type AgentInstance struct {
	ID                string
	Name              string
	Model             string
	Fallbacks         []string
	Workspace         string
	MaxIterations     int
	MaxTokens         int
	Temperature       float64
	ContextWindow     int
	Provider          providers.LLMProvider
	Sessions          *session.SessionManager
	ContextBuilder    *ContextBuilder
	Tools             *tools.ToolRegistry
	Subagents         *config.SubagentsConfig
	skillsFilterMutex sync.RWMutex // Protects SkillsFilter
	SkillsFilter      []string
	Candidates        []providers.FallbackCandidate
}

// NewAgentInstance creates an agent instance from config.
func NewAgentInstance(
	agentCfg *config.AgentConfig,
	defaults *config.AgentDefaults,
	cfg *config.Config,
	provider providers.LLMProvider,
) *AgentInstance {
	workspace := resolveAgentWorkspace(agentCfg, defaults)
	os.MkdirAll(workspace, 0o755)

	model := resolveAgentModel(agentCfg, defaults)
	fallbacks := resolveAgentFallbacks(agentCfg, defaults)

	restrict := defaults.RestrictToWorkspace
	toolsRegistry := tools.NewToolRegistry()
	toolsRegistry.Register(tools.NewReadFileTool(workspace, restrict))
	toolsRegistry.Register(tools.NewWriteFileTool(workspace, restrict))
	toolsRegistry.Register(tools.NewListDirTool(workspace, restrict))
	toolsRegistry.Register(tools.NewExecToolWithConfig(workspace, restrict, cfg))
	toolsRegistry.Register(tools.NewEditFileTool(workspace, restrict))
	toolsRegistry.Register(tools.NewAppendFileTool(workspace, restrict))

	sessionsDir := filepath.Join(workspace, "sessions")
	sessionsManager := session.NewSessionManager(sessionsDir)

	contextBuilder := NewContextBuilder(workspace)

	agentID := routing.DefaultAgentID
	agentName := ""
	var subagents *config.SubagentsConfig
	var skillsFilter []string

	if agentCfg != nil {
		agentID = routing.NormalizeAgentID(agentCfg.ID)
		agentName = agentCfg.Name
		subagents = agentCfg.Subagents
		skillsFilter = agentCfg.Skills
	}

	maxIter := defaults.MaxToolIterations
	if maxIter == 0 {
		maxIter = 20
	}

	maxTokens := defaults.MaxTokens
	if maxTokens == 0 {
		maxTokens = 8192
	}

	temperature := 0.7
	if defaults.Temperature != nil {
		temperature = *defaults.Temperature
	}

	// Resolve fallback candidates
	modelCfg := providers.ModelConfig{
		Primary:   model,
		Fallbacks: fallbacks,
	}
	candidates := providers.ResolveCandidates(modelCfg, defaults.Provider)

	return &AgentInstance{
		ID:             agentID,
		Name:           agentName,
		Model:          model,
		Fallbacks:      fallbacks,
		Workspace:      workspace,
		MaxIterations:  maxIter,
		MaxTokens:      maxTokens,
		Temperature:    temperature,
		ContextWindow:  maxTokens,
		Provider:       provider,
		Sessions:       sessionsManager,
		ContextBuilder: contextBuilder,
		Tools:          toolsRegistry,
		Subagents:      subagents,
		SkillsFilter:   skillsFilter,
		Candidates:     candidates,
	}
}

// resolveAgentWorkspace determines the workspace directory for an agent.
func resolveAgentWorkspace(agentCfg *config.AgentConfig, defaults *config.AgentDefaults) string {
	if agentCfg != nil && strings.TrimSpace(agentCfg.Workspace) != "" {
		return expandHome(strings.TrimSpace(agentCfg.Workspace))
	}
	if agentCfg == nil || agentCfg.Default || agentCfg.ID == "" || routing.NormalizeAgentID(agentCfg.ID) == "main" {
		return expandHome(defaults.Workspace)
	}
	home, _ := os.UserHomeDir()
	id := routing.NormalizeAgentID(agentCfg.ID)
	return filepath.Join(home, ".picoclaw", "workspace-"+id)
}

// resolveAgentModel resolves the primary model for an agent.
func resolveAgentModel(agentCfg *config.AgentConfig, defaults *config.AgentDefaults) string {
	if agentCfg != nil && agentCfg.Model != nil && strings.TrimSpace(agentCfg.Model.Primary) != "" {
		return strings.TrimSpace(agentCfg.Model.Primary)
	}
	return defaults.GetModelName()
}

// resolveAgentFallbacks resolves the fallback models for an agent.
func resolveAgentFallbacks(agentCfg *config.AgentConfig, defaults *config.AgentDefaults) []string {
	if agentCfg != nil && agentCfg.Model != nil && agentCfg.Model.Fallbacks != nil {
		return agentCfg.Model.Fallbacks
	}
	return defaults.ModelFallbacks
}

func expandHome(path string) string {
	if path == "" {
		return path
	}
	if path[0] == '~' {
		home, _ := os.UserHomeDir()
		if len(path) > 1 && path[1] == '/' {
			return home + path[1:]
		}
		return home
	}
	return path
}

// SetSkillsFilter dynamically sets the skills filter for this agent.
// Modifying the filter will trigger ContextBuilder cache invalidation
// to ensure the next request uses the updated skill set.
//
// Parameters:
//   - filters: List of skill names to include. Empty or nil clears the filter.
//
// Example:
//
//	agent.SetSkillsFilter([]string{"customer-service", "faq"})
//	agent.SetSkillsFilter(nil) // Clear filter, all skills available
func (ai *AgentInstance) SetSkillsFilter(filters []string) {
	ai.skillsFilterMutex.Lock()
	defer ai.skillsFilterMutex.Unlock()

	// Create a copy to prevent external modification
	if filters == nil {
		ai.SkillsFilter = nil
	} else {
		ai.SkillsFilter = make([]string, len(filters))
		copy(ai.SkillsFilter, filters)
	}

	// Trigger context builder cache invalidation and update its filter
	if ai.ContextBuilder != nil {
		ai.ContextBuilder.SetSkillsFilter(filters)
	}
}

// EnableSkillRecommender enables intelligent skill recommendation for this agent.
// The recommender will automatically select relevant skills based on context.
//
// Parameters:
//   - model: Model to use for LLM-based recommendations (optional, uses agent's model if empty)
//   - weights: Optional custom weights for scoring algorithm (channel, keyword, history, recency)
//
// Example:
//
//	// Enable with default settings
//	agent.EnableSkillRecommender()
//
//	// Enable with custom weights
//	agent.EnableSkillRecommenderWithWeights(0.5, 0.3, 0.15, 0.05)
func (ai *AgentInstance) EnableSkillRecommender(model string, weights ...[4]float64) {
	if model == "" {
		model = ai.Model
	}

	recommender := NewSkillRecommender(
		ai.ContextBuilder.skillsLoader,
		ai.Provider,
		model,
	)

	// Set custom weights if provided
	if len(weights) > 0 {
		w := weights[0]
		recommender.SetWeights(w[0], w[1], w[2], w[3])
	}

	ai.ContextBuilder.SetSkillRecommender(recommender)

	logger.InfoCF("agent", "Skill recommender enabled",
		map[string]any{
			"agent_id": ai.ID,
			"model":    model,
		})
}

// EnableSkillRecommenderWithWeights enables skill recommender with custom weights.
// This is a convenience wrapper around EnableSkillRecommender.
func (ai *AgentInstance) EnableSkillRecommenderWithWeights(channel, keyword, history, recency float64) {
	ai.EnableSkillRecommender("", [4]float64{channel, keyword, history, recency})
}

// GetSkillsFilter returns the current skills filter.
// Returns nil if no filter is set (all skills available).
//
// The returned slice is a copy to prevent concurrent modification.
func (ai *AgentInstance) GetSkillsFilter() []string {
	ai.skillsFilterMutex.RLock()
	defer ai.skillsFilterMutex.RUnlock()

	if ai.SkillsFilter == nil {
		return nil
	}

	// Return a copy to prevent concurrent modification
	result := make([]string, len(ai.SkillsFilter))
	copy(result, ai.SkillsFilter)
	return result
}
