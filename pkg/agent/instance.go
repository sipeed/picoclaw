package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/isolation"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/memory"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/routing"
	"github.com/sipeed/picoclaw/pkg/session"
	"github.com/sipeed/picoclaw/pkg/tools"
)

// AgentInstance represents a fully configured agent with its own workspace,
// session manager, context builder, and tool registry.
type AgentInstance struct {
	ID                        string
	Name                      string
	Model                     string
	Fallbacks                 []string
	Workspace                 string
	MaxIterations             int
	MaxTokens                 int
	Temperature               float64
	ThinkingLevel             ThinkingLevel
	ContextWindow             int
	SummarizeMessageThreshold int
	SummarizeTokenPercent     int
	Provider                  providers.LLMProvider
	Sessions                  session.SessionStore
	ContextBuilder            *ContextBuilder
	Tools                     *tools.ToolRegistry
	Definition                AgentContextDefinition
	Subagents                 *config.SubagentsConfig
	SkillsFilter              []string
	MCPServerAllowlist        map[string]struct{}
	Candidates                []providers.FallbackCandidate

	// Router is non-nil when model routing is configured and the light model
	// was successfully resolved. It scores each incoming message and decides
	// whether to route to LightCandidates or stay with Candidates.
	Router *routing.Router
	// LightCandidates holds the resolved provider candidates for the light model.
	// Pre-computed at agent creation to avoid repeated model_list lookups at runtime.
	LightCandidates []providers.FallbackCandidate
	// LightProvider is the concrete provider instance for the configured light model.
	// It is only used when routing selects the light tier for a turn.
	LightProvider providers.LLMProvider
	// CandidateProviders maps "provider/model" keys to per-candidate LLMProvider
	// instances. This allows each fallback model to use its own api_base and api_key
	// from model_list, instead of inheriting the primary model's provider config.
	CandidateProviders map[string]providers.LLMProvider
}

// NewAgentInstance creates an agent instance from config.
func NewAgentInstance(
	agentCfg *config.AgentConfig,
	defaults *config.AgentDefaults,
	cfg *config.Config,
	provider providers.LLMProvider,
) *AgentInstance {
	if cfg != nil {
		// Keep the subprocess isolation runtime aligned with the latest loaded config
		// before any tools or providers start spawning child processes.
		isolation.Configure(cfg)
	}

	workspace := resolveAgentWorkspace(agentCfg, defaults)
	os.MkdirAll(workspace, 0o755)

	definition := loadAgentDefinition(workspace)

	model := resolveAgentModel(agentCfg, defaults, definition)
	fallbacks := resolveAgentFallbacks(agentCfg, defaults)

	restrict := defaults.RestrictToWorkspace
	readRestrict := restrict && !defaults.AllowReadOutsideWorkspace

	// Compile path whitelist patterns from config.
	allowReadPaths := buildAllowReadPatterns(cfg)
	allowWritePaths := compilePatterns(cfg.Tools.AllowWritePaths)
	agentToolAllowlist := resolveAgentToolAllowlist(definition)
	agentMCPServerAllowlist := resolveAgentMCPServerAllowlist(definition)

	toolsRegistry := tools.NewToolRegistry()
	toolsRegistry.SetAllowlist(agentToolAllowlist)

	if cfg.Tools.IsToolEnabled("read_file") {
		maxReadFileSize := cfg.Tools.ReadFile.MaxReadFileSize
		switch cfg.Tools.ReadFile.EffectiveMode() {
		case config.ReadFileModeLines:
			toolsRegistry.Register(tools.NewReadFileLinesTool(workspace, readRestrict, maxReadFileSize, allowReadPaths))
		default:
			toolsRegistry.Register(tools.NewReadFileBytesTool(workspace, readRestrict, maxReadFileSize, allowReadPaths))
		}
	}
	if cfg.Tools.IsToolEnabled("write_file") {
		toolsRegistry.Register(tools.NewWriteFileTool(workspace, restrict, allowWritePaths))
	}
	if cfg.Tools.IsToolEnabled("list_dir") {
		toolsRegistry.Register(tools.NewListDirTool(workspace, readRestrict, allowReadPaths))
	}
	if cfg.Tools.IsToolEnabled("exec") {
		execTool, err := tools.NewExecToolWithConfig(workspace, restrict, cfg, allowReadPaths)
		if err != nil {
			logger.ErrorCF("agent", "Failed to initialize exec tool; continuing without exec",
				map[string]any{"error": err.Error()})
		} else {
			toolsRegistry.Register(execTool)
		}
	}

	if cfg.Tools.IsToolEnabled("edit_file") {
		toolsRegistry.Register(tools.NewEditFileTool(workspace, restrict, allowWritePaths))
	}
	if cfg.Tools.IsToolEnabled("append_file") {
		toolsRegistry.Register(tools.NewAppendFileTool(workspace, restrict, allowWritePaths))
	}

	sessionsDir := filepath.Join(workspace, "sessions")
	sessions := initSessionStore(sessionsDir)

	mcpDiscoveryActive := agentHasDiscoverableMCPServers(cfg, agentMCPServerAllowlist)
	contextBuilder := NewContextBuilder(workspace).
		WithToolDiscovery(
			mcpDiscoveryActive && cfg.Tools.MCP.Discovery.UseBM25,
			mcpDiscoveryActive && cfg.Tools.MCP.Discovery.UseRegex,
		).
		WithSplitOnMarker(cfg.Agents.Defaults.SplitOnMarker)

	agentID := routing.DefaultAgentID
	agentName := ""
	var subagents *config.SubagentsConfig
	var skillsFilter []string

	if agentCfg != nil {
		agentID = routing.NormalizeAgentID(agentCfg.ID)
		agentName = agentCfg.Name
		if definition.Agent != nil && strings.TrimSpace(definition.Agent.Frontmatter.Name) != "" {
			agentName = strings.TrimSpace(definition.Agent.Frontmatter.Name)
		}
		subagents = agentCfg.Subagents
		skillsFilter = resolveAgentSkillsFilter(agentCfg, definition)
	}
	provider = resolvePrimaryProviderForAgent(cfg, workspace, agentID, model, provider)
	warnOnUnknownAgentMCPServerDeclarations(agentID, workspace, cfg, definition)

	maxIter := defaults.MaxToolIterations
	if maxIter == 0 {
		maxIter = 20
	}

	maxTokens := defaults.MaxTokens
	if maxTokens == 0 {
		maxTokens = 8192
	}

	contextWindow := defaults.ContextWindow
	if contextWindow == 0 {
		// Default heuristic: 4x the output token limit.
		// Most models have context windows well above their output limits
		// (e.g., GPT-4o 128k ctx / 16k out, Claude 200k ctx / 8k out).
		// 4x is a conservative lower bound that avoids premature
		// summarization while remaining safe — the reactive
		// forceCompression handles any overshoot.
		contextWindow = maxTokens * 4
	}

	temperature := 0.7
	if defaults.Temperature != nil {
		temperature = *defaults.Temperature
	}

	var thinkingLevelStr string
	if mc, err := cfg.GetModelConfig(model); err == nil {
		thinkingLevelStr = mc.ThinkingLevel
	}
	thinkingLevel := parseThinkingLevel(thinkingLevelStr)

	summarizeMessageThreshold := defaults.SummarizeMessageThreshold
	if summarizeMessageThreshold == 0 {
		summarizeMessageThreshold = 20
	}

	summarizeTokenPercent := defaults.SummarizeTokenPercent
	if summarizeTokenPercent == 0 {
		summarizeTokenPercent = 75
	}

	// Resolve fallback candidates
	candidates := resolveModelCandidates(cfg, defaults.Provider, model, fallbacks)

	candidateProviders := make(map[string]providers.LLMProvider)
	populateCandidateProvidersFromNames(cfg, workspace, fallbacks, candidateProviders)

	// Model routing setup: pre-resolve light model candidates at creation time
	// to avoid repeated model_list lookups on every incoming message.
	var router *routing.Router
	var lightCandidates []providers.FallbackCandidate
	var lightProvider providers.LLMProvider
	if rc := defaults.Routing; rc != nil && rc.Enabled && rc.LightModel != "" {
		resolved := resolveModelCandidates(cfg, defaults.Provider, rc.LightModel, nil)
		if len(resolved) > 0 {
			lightModelCfg, err := resolvedModelConfig(cfg, rc.LightModel, workspace)
			if err != nil {
				logger.WarnCF(
					"agent",
					"Routing light model config invalid; routing disabled",
					map[string]any{
						"light_model": rc.LightModel,
						"agent_id":    agentID,
						"error":       err.Error(),
					},
				)
			} else {
				lp, _, err := providers.CreateProviderFromConfig(lightModelCfg)
				if err != nil {
					logger.WarnCF("agent", "Routing light model provider init failed; routing disabled",
						map[string]any{"light_model": rc.LightModel, "agent_id": agentID, "error": err.Error()})
				} else {
					router = routing.New(routing.RouterConfig{
						LightModel: rc.LightModel,
						Threshold:  rc.Threshold,
					})
					lightCandidates = resolved
					lightProvider = lp
					populateCandidateProvidersFromNames(cfg, workspace, []string{rc.LightModel}, candidateProviders)
				}
			}
		} else {
			logger.WarnCF("agent", "Routing light model not found; routing disabled",
				map[string]any{"light_model": rc.LightModel, "agent_id": agentID})
		}
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
		ThinkingLevel:             thinkingLevel,
		ContextWindow:             contextWindow,
		SummarizeMessageThreshold: summarizeMessageThreshold,
		SummarizeTokenPercent:     summarizeTokenPercent,
		Provider:                  provider,
		Sessions:                  sessions,
		ContextBuilder:            contextBuilder,
		Tools:                     toolsRegistry,
		Definition:                definition,
		Subagents:                 subagents,
		SkillsFilter:              skillsFilter,
		MCPServerAllowlist:        agentMCPServerAllowlist,
		Candidates:                candidates,
		Router:                    router,
		LightCandidates:           lightCandidates,
		LightProvider:             lightProvider,
		CandidateProviders:        candidateProviders,
	}
}

// populateCandidateProvidersFromNames resolves each model name (alias or
// "provider/model") via resolvedModelConfig and creates a dedicated LLMProvider
// for it. This reuses the canonical config resolution path (GetModelConfig) so
// alias handling and load-balancing stay consistent with the rest of the codebase.
func populateCandidateProvidersFromNames(
	cfg *config.Config,
	workspace string,
	names []string,
	out map[string]providers.LLMProvider,
) {
	if cfg == nil || len(names) == 0 {
		return
	}
	for _, name := range names {
		mc, err := resolvedModelConfig(cfg, strings.TrimSpace(name), workspace)
		if err != nil {
			logger.WarnCF("agent",
				"fallback provider: no model_list entry found; will inherit primary provider credentials",
				map[string]any{"name": name, "error": err.Error()})
			continue
		}
		protocol, modelID := providers.ExtractProtocol(mc)
		key := providers.ModelKey(protocol, modelID)
		if _, exists := out[key]; exists {
			continue
		}
		p, _, err := providers.CreateProviderFromConfig(mc)
		if err != nil {
			logger.WarnCF("agent", "fallback provider: failed to create provider",
				map[string]any{"model": mc.Model, "error": err.Error()})
			continue
		}
		out[key] = p
	}
}

// resolvePrimaryProviderForAgent resolves a dedicated provider for the active
// primary model when the model points at a model_list entry. This keeps the
// agent's single-candidate path aligned with the selected model's own
// provider/api_base/api_key instead of inheriting the process default provider.
func resolvePrimaryProviderForAgent(
	cfg *config.Config,
	workspace string,
	agentID string,
	model string,
	fallback providers.LLMProvider,
) providers.LLMProvider {
	model = strings.TrimSpace(model)
	if cfg == nil || model == "" {
		return fallback
	}

	modelCfg := lookupModelConfigByRef(cfg, model)
	if modelCfg == nil {
		return fallback
	}
	clone := *modelCfg
	if clone.Workspace == "" {
		clone.Workspace = workspace
	}

	resolvedProvider, _, err := providers.CreateProviderFromConfig(&clone)
	if err != nil {
		logger.WarnCF("agent", "Primary model provider init failed; using injected provider",
			map[string]any{
				"agent_id": agentID,
				"model":    model,
				"error":    err.Error(),
			})
		return fallback
	}
	if resolvedProvider == nil {
		return fallback
	}
	return resolvedProvider
}

// resolveAgentWorkspace determines the workspace directory for an agent.
func resolveAgentWorkspace(agentCfg *config.AgentConfig, defaults *config.AgentDefaults) string {
	if agentCfg != nil && strings.TrimSpace(agentCfg.Workspace) != "" {
		return expandHome(strings.TrimSpace(agentCfg.Workspace))
	}
	// Use the configured default workspace (respects PICOCLAW_HOME)
	if agentCfg == nil || agentCfg.Default || agentCfg.ID == "" ||
		routing.NormalizeAgentID(agentCfg.ID) == "main" {
		return expandHome(defaults.Workspace)
	}
	// For named agents without explicit workspace, use default workspace with agent ID suffix
	id := routing.NormalizeAgentID(agentCfg.ID)
	return filepath.Join(expandHome(defaults.Workspace), "..", "workspace-"+id)
}

// resolveAgentModel resolves the primary model for an agent.
func resolveAgentModel(
	agentCfg *config.AgentConfig,
	defaults *config.AgentDefaults,
	definition AgentContextDefinition,
) string {
	if definition.Agent != nil && strings.TrimSpace(definition.Agent.Frontmatter.Model) != "" {
		return strings.TrimSpace(definition.Agent.Frontmatter.Model)
	}
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

func resolveAgentSkillsFilter(
	agentCfg *config.AgentConfig,
	definition AgentContextDefinition,
) []string {
	if definition.Agent != nil && definition.Agent.Frontmatter.Skills != nil {
		return append([]string(nil), definition.Agent.Frontmatter.Skills...)
	}
	if agentCfg == nil || agentCfg.Skills == nil {
		return nil
	}
	return append([]string(nil), agentCfg.Skills...)
}

func (a *AgentInstance) AllowsMCPServer(serverName string) bool {
	if a == nil || a.MCPServerAllowlist == nil {
		return true
	}
	_, ok := a.MCPServerAllowlist[strings.ToLower(strings.TrimSpace(serverName))]
	return ok
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

func buildAllowReadPatterns(cfg *config.Config) []*regexp.Regexp {
	var configured []string
	if cfg != nil {
		configured = cfg.Tools.AllowReadPaths
	}

	compiled := compilePatterns(configured)
	mediaDirPattern := regexp.MustCompile(mediaTempDirPattern())
	for _, pattern := range compiled {
		if pattern.String() == mediaDirPattern.String() {
			return compiled
		}
	}

	return append(compiled, mediaDirPattern)
}

func mediaTempDirPattern() string {
	sep := regexp.QuoteMeta(string(os.PathSeparator))
	return "^" + regexp.QuoteMeta(filepath.Clean(media.TempDir())) + "(?:" + sep + "|$)"
}

// Close releases resources held by the agent's session store.
func (a *AgentInstance) Close() error {
	if a.Sessions != nil {
		return a.Sessions.Close()
	}
	return nil
}

// initSessionStore creates the session persistence backend.
// It uses the JSONL store by default and auto-migrates legacy JSON sessions.
// Falls back to SessionManager if the JSONL store cannot be initialized or
// if migration fails (which indicates the store cannot write reliably).
func initSessionStore(dir string) session.SessionStore {
	store, err := memory.NewJSONLStore(dir)
	if err != nil {
		logger.WarnCF("agent", "Memory JSONL store init failed; falling back to json sessions",
			map[string]any{"error": err.Error()})
		return session.NewSessionManager(dir)
	}

	if n, merr := memory.MigrateFromJSON(context.Background(), dir, store); merr != nil {
		// Migration failure means the store could not write data.
		// Fall back to SessionManager to avoid a split state where
		// some sessions are in JSONL and others remain in JSON.
		logger.WarnCF("agent", "Memory migration failed; falling back to json sessions",
			map[string]any{"error": merr.Error()})
		store.Close()
		return session.NewSessionManager(dir)
	} else if n > 0 {
		logger.InfoCF("agent", "Memory migrated to JSONL", map[string]any{"sessions_migrated": n})
	}

	return session.NewJSONLBackend(store)
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
