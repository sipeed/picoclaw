package agent

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
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
	// mu protects Model, Candidates and SubagentMgr which can be changed at
	// runtime by the /switch model command while the LLM loop is running.
	mu sync.RWMutex

	ID                        string
	Name                      string
	Model                     string // registry key (model name)
	Fallbacks                 []string
	Workspace                 string
	MaxIterations             int
	MaxTokens                 int
	Temperature               float64
	ContextWindow             int
	SummarizeMessageThreshold int
	SummarizeTokenPercent     int
	Sessions                  *session.SessionManager
	ContextBuilder            *ContextBuilder
	Tools                     *tools.ToolRegistry
	Subagents                 *config.SubagentsConfig
	SkillsFilter              []string
	Candidates                []providers.FallbackCandidate
	SubagentMgr               *tools.SubagentManager // updated on /switch model
	AllowReadPaths            []*regexp.Regexp        // compiled path whitelist for reads
	AllowWritePaths           []*regexp.Regexp        // compiled path whitelist for writes
}

// NewAgentInstance creates an agent instance from config.
// The ModelRegistry is used to resolve the provider and model ID.
func NewAgentInstance(
	agentCfg *config.AgentConfig,
	defaults *config.AgentDefaults,
	cfg *config.Config,
	modelRegistry *providers.ModelRegistry,
) *AgentInstance {
	workspace := resolveAgentWorkspace(agentCfg, defaults)
	os.MkdirAll(workspace, 0o755)

	modelName := resolveAgentModel(agentCfg, defaults)
	fallbacks := resolveAgentFallbacks(agentCfg, defaults)

	// Model stays as the model name (registry key); provider is resolved at call time.
	model := modelName

	restrict := defaults.RestrictToWorkspace
	readRestrict := restrict && !defaults.AllowReadOutsideWorkspace

	// Compile path whitelist patterns once and store on the instance.
	allowReadPaths := compilePatterns(cfg.Tools.AllowReadPaths)
	allowWritePaths := compilePatterns(cfg.Tools.AllowWritePaths)

	toolsRegistry := tools.NewToolRegistry()
	toolsRegistry.Register(tools.NewReadFileTool(workspace, readRestrict, allowReadPaths))
	toolsRegistry.Register(tools.NewWriteFileTool(workspace, restrict, allowWritePaths))
	toolsRegistry.Register(tools.NewListDirTool(workspace, readRestrict, allowReadPaths))
	execTool, err := tools.NewExecToolWithConfig(workspace, restrict, cfg)
	if err != nil {
		log.Fatalf("Critical error: unable to initialize exec tool: %v", err)
	}
	toolsRegistry.Register(execTool)

	toolsRegistry.Register(tools.NewEditFileTool(workspace, restrict, allowWritePaths))
	toolsRegistry.Register(tools.NewAppendFileTool(workspace, restrict, allowWritePaths))

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

	summarizeMessageThreshold := defaults.SummarizeMessageThreshold
	if summarizeMessageThreshold == 0 {
		summarizeMessageThreshold = 20
	}

	summarizeTokenPercent := defaults.SummarizeTokenPercent
	if summarizeTokenPercent == 0 {
		summarizeTokenPercent = 75
	}

	// Build fallback candidates from the registry using model names as keys.
	allModels := append([]string{modelName}, fallbacks...)
	seen := make(map[string]bool)
	var candidates []providers.FallbackCandidate
	for _, name := range allModels {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		if _, ok := modelRegistry.Get(name); ok {
			candidates = append(candidates, providers.FallbackCandidate{Model: name})
		} else {
			logger.WarnCF("agent", "Model not found in registry, skipping",
				map[string]any{"model": name, "agent_id": agentID})
		}
	}
	if len(candidates) == 0 {
		logger.WarnCF("agent", "Agent has no valid candidates; will use registry default",
			map[string]any{"agent_id": agentID, "configured_model": modelName})
	}

	return &AgentInstance{
		ID:                        agentID,
		Name:                      agentName,
		Model:                     model,
		Fallbacks:                 fallbacks,
		Workspace:                 workspace,
		MaxIterations:             maxIter,
		MaxTokens:                 maxTokens,
		Temperature:               temperature,
		ContextWindow:             maxTokens,
		SummarizeMessageThreshold: summarizeMessageThreshold,
		SummarizeTokenPercent:     summarizeTokenPercent,
		Sessions:                  sessionsManager,
		ContextBuilder:            contextBuilder,
		Tools:                     toolsRegistry,
		Subagents:                 subagents,
		SkillsFilter:              skillsFilter,
		Candidates:                candidates,
		AllowReadPaths:            allowReadPaths,
		AllowWritePaths:           allowWritePaths,
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

func compilePatterns(patterns []string) []*regexp.Regexp {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			fmt.Printf("Warning: invalid path pattern %q: %v\n", p, err)
			continue
		}
		compiled = append(compiled, re)
	}
	return compiled
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

// getModelSnapshot returns a consistent snapshot of the mutable routing fields.
// Use this at the start of each LLM iteration rather than reading fields directly.
func (a *AgentInstance) getModelSnapshot() (model string, candidates []providers.FallbackCandidate) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	model = a.Model
	candidates = make([]providers.FallbackCandidate, len(a.Candidates))
	copy(candidates, a.Candidates)
	return
}

// switchModel atomically updates Model.
// Returns the previous model name.
func (a *AgentInstance) switchModel(model string) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	old := a.Model
	a.Model = model
	if a.SubagentMgr != nil {
		a.SubagentMgr.UpdateModel(model)
	}
	return old
}
