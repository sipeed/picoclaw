package agent

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/sipeed/picoclaw/pkg/agent/sandbox"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/routing"
	"github.com/sipeed/picoclaw/pkg/session"
	"github.com/sipeed/picoclaw/pkg/tools"
)

// AgentInstance represents a fully configured agent with its own workspace,
// session manager, context builder, and tool registry.
type AgentInstance struct {
	ID             string
	Name           string
	Model          string
	Fallbacks      []string
	Workspace      string
	MaxIterations  int
	MaxTokens      int
	Temperature    float64
	ContextWindow  int
	Provider       providers.LLMProvider
	Sessions       *session.SessionManager
	ContextBuilder *ContextBuilder
	Tools          *tools.ToolRegistry
	SandboxManager sandbox.Manager
	Subagents      *config.SubagentsConfig
	SkillsFilter   []string
	Candidates     []providers.FallbackCandidate
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

	restrict := defaults.RestrictToWorkspace
	roContainer := isContainerReadOnlySandbox(cfg)
	toolsRegistry := tools.NewToolRegistry()

	sandboxManager := sandbox.NewFromConfigWithAgent(workspace, restrict, cfg, agentID)
	isToolEnabled := func(toolName string) bool {
		if isSandboxModeOff(cfg) {
			return true
		}
		return sandbox.IsToolSandboxEnabled(cfg, toolName)
	}

	if isToolEnabled("read_file") {
		toolsRegistry.Register(tools.NewReadFileTool(workspace, restrict))
	}
	if !roContainer && isToolEnabled("write_file") {
		toolsRegistry.Register(tools.NewWriteFileTool(workspace, restrict))
	}
	if isToolEnabled("list_dir") {
		toolsRegistry.Register(tools.NewListDirTool(workspace, restrict))
	}
	if isToolEnabled("exec") {
		execTool, err := tools.NewExecToolWithConfig(workspace, restrict, cfg)
		if err != nil {
			log.Fatalf("Critical error: unable to initialize exec tool: %v", err)
		}
		toolsRegistry.Register(execTool)
	}
	if !roContainer {
		if isToolEnabled("edit_file") {
			toolsRegistry.Register(tools.NewEditFileTool(workspace, restrict))
		}
		if isToolEnabled("append_file") {
			toolsRegistry.Register(tools.NewAppendFileTool(workspace, restrict))
		}
	}

	sessionsDir := filepath.Join(workspace, "sessions")
	sessionsManager := session.NewSessionManager(sessionsDir)

	contextBuilder := NewContextBuilder(workspace)

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

	modelCfg := providers.ModelConfig{
		Primary:   model,
		Fallbacks: fallbacks,
	}
	resolveFromModelList := func(raw string) (string, bool) {
		ensureProtocol := func(model string) string {
			model = strings.TrimSpace(model)
			if model == "" {
				return ""
			}
			if strings.Contains(model, "/") {
				return model
			}
			return "openai/" + model
		}

		raw = strings.TrimSpace(raw)
		if raw == "" {
			return "", false
		}

		if cfg != nil {
			if mc, err := cfg.GetModelConfig(raw); err == nil && mc != nil && strings.TrimSpace(mc.Model) != "" {
				return ensureProtocol(mc.Model), true
			}

			for i := range cfg.ModelList {
				fullModel := strings.TrimSpace(cfg.ModelList[i].Model)
				if fullModel == "" {
					continue
				}
				if fullModel == raw {
					return ensureProtocol(fullModel), true
				}
				_, modelID := providers.ExtractProtocol(fullModel)
				if modelID == raw {
					return ensureProtocol(fullModel), true
				}
			}
		}

		return "", false
	}

	candidates := providers.ResolveCandidatesWithLookup(modelCfg, defaults.Provider, resolveFromModelList)

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
		SandboxManager: sandboxManager,
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
	defaultWS := expandHome(defaults.Workspace)
	if agentCfg == nil || agentCfg.Default || agentCfg.ID == "" || routing.NormalizeAgentID(agentCfg.ID) == "main" {
		return defaultWS
	}
	parent := filepath.Dir(defaultWS)
	id := routing.NormalizeAgentID(agentCfg.ID)
	return filepath.Join(parent, "workspace-"+id)
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

func isContainerReadOnlySandbox(cfg *config.Config) bool {
	if cfg == nil {
		return false
	}
	return cfg.Agents.Defaults.Sandbox.Mode == config.SandboxModeAll &&
		cfg.Agents.Defaults.Sandbox.WorkspaceAccess == config.WorkspaceAccessRO
}

func isSandboxModeOff(cfg *config.Config) bool {
	if cfg == nil {
		return false
	}
	return cfg.Agents.Defaults.Sandbox.Mode == config.SandboxModeOff
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
