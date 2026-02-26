package agent

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/routing"
	"github.com/sipeed/picoclaw/pkg/session"
	"github.com/sipeed/picoclaw/pkg/tools"
)

// AgentInstance represents a fully configured agent with its own workspace,
// session manager, context builder, and tool registry.
type AgentInstance struct {
	ID               string
	Name             string
	Model            string
	Fallbacks        []string
	Workspace        string
	MaxIterations    int
	MaxTokens        int
	Temperature      float64
	ContextWindow    int
	Provider         providers.LLMProvider              // Default provider for backward compatibility
	ProviderRegistry *providers.ProviderRegistry        // Registry for multi-provider fallback
	Sessions         *session.SessionManager
	ContextBuilder   *ContextBuilder
	Tools            *tools.ToolRegistry
	Subagents        *config.SubagentsConfig
	SkillsFilter     []string
	Candidates       []providers.FallbackCandidate
}

// NewAgentInstance creates an agent instance from config.
func NewAgentInstance(
	agentCfg *config.AgentConfig,
	defaults *config.AgentDefaults,
	cfg *config.Config,
	provider providers.LLMProvider,
	providerRegistry *providers.ProviderRegistry,
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
	// First, look up model names in model_list to get full model strings
	primaryModelString := resolveModelString(cfg, model)
	fallbackModelStrings := resolveFallbackModelStrings(cfg, fallbacks)

	modelCfg := providers.ModelConfig{
		Primary:   primaryModelString,
		Fallbacks: fallbackModelStrings,
	}
	candidates := providers.ResolveCandidates(modelCfg, defaults.Provider)

	return &AgentInstance{
		ID:               agentID,
		Name:             agentName,
		Model:            model,
		Fallbacks:        fallbacks,
		Workspace:        workspace,
		MaxIterations:    maxIter,
		MaxTokens:        maxTokens,
		Temperature:      temperature,
		ContextWindow:    maxTokens,
		Provider:         provider,
		ProviderRegistry: providerRegistry,
		Sessions:         sessionsManager,
		ContextBuilder:   contextBuilder,
		Tools:            toolsRegistry,
		Subagents:        subagents,
		SkillsFilter:     skillsFilter,
		Candidates:       candidates,
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

// resolveModelString looks up a model name in model_list and returns the full model string.
// If the model name already contains a "/" (like "openrouter/free"), it's returned as-is.
// If the model name is not found in model_list, it's returned as-is (for backward compatibility).
func resolveModelString(cfg *config.Config, modelName string) string {
	// If it already looks like a full model string (protocol/model), return it as-is
	if strings.Contains(modelName, "/") {
		return modelName
	}

	// Look up in model_list
	modelCfg, err := cfg.GetModelConfig(modelName)
	if err != nil {
		// Model not found in model_list, return as-is for backward compatibility
		return modelName
	}

	// Return the full model string (e.g., "antigravity/gemini-3-flash")
	return modelCfg.Model
}

// resolveFallbackModelStrings looks up multiple model names in model_list and returns their full model strings.
func resolveFallbackModelStrings(cfg *config.Config, modelNames []string) []string {
	result := make([]string, 0, len(modelNames))
	for _, name := range modelNames {
		result = append(result, resolveModelString(cfg, name))
	}
	return result
}
