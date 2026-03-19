package agent

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/routing"
	"github.com/sipeed/picoclaw/pkg/session"
	"github.com/sipeed/picoclaw/pkg/tools"
)

// AgentInstance represents a fully configured agent with its own workspace,
// session manager, context builder, and tool registry.
type AgentInstance struct {
	instanceExt // fork-specific fields (see instance_ext.go)

	ID                        string
	Name                      string
	Model                     string
	Fallbacks                 []string
	Workspace                 string
	MaxIterations             int
	TaskReminderInterval      int
	MaxTokens                 int
	Temperature               float64
	ThinkingLevel             ThinkingLevel
	ContextWindow             int
	SummarizeMessageThreshold int
	SummarizeTokenPercent     int
	Provider                  providers.LLMProvider
	Sessions                  *session.LegacyAdapter
	ContextBuilder            *ContextBuilder
	Tools                     *tools.ToolRegistry
	Subagents                 *config.SubagentsConfig
	SkillsFilter              []string
	Candidates                []providers.FallbackCandidate
	PlanModel                 string
	PlanFallbacks             []string
	PlanCandidates            []providers.FallbackCandidate

	// Router is non-nil when model routing is configured and the light model
	// was successfully resolved. It scores each incoming message and decides
	// whether to route to LightCandidates or stay with Candidates.
	Router *routing.Router
	// LightCandidates holds the resolved provider candidates for the light model.
	// Pre-computed at agent creation to avoid repeated model_list lookups at runtime.
	LightCandidates []providers.FallbackCandidate
}

// Close releases resources held by the agent instance.
// If the provider implements StatefulProvider, its Close method is called.
func (ai *AgentInstance) Close() error {
	if sp, ok := ai.Provider.(providers.StatefulProvider); ok {
		sp.Close()
	}
	return nil
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
	readRestrict := restrict && !defaults.AllowReadOutsideWorkspace

	// Compile path whitelist patterns from config.
	allowReadPaths := buildAllowReadPatterns(cfg)
	allowWritePaths := compilePatterns(cfg.Tools.AllowWritePaths)

	toolsRegistry := tools.NewToolRegistry()

	if cfg.Tools.IsToolEnabled("read_file") {
		maxReadFileSize := cfg.Tools.ReadFile.MaxReadFileSize
		toolsRegistry.Register(tools.NewReadFileTool(workspace, readRestrict, maxReadFileSize, allowReadPaths))
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
			log.Fatalf("Critical error: unable to initialize exec tool: %v", err)
		}
		toolsRegistry.Register(execTool)
	}

	if cfg.Tools.IsToolEnabled("edit_file") {
		toolsRegistry.Register(tools.NewEditFileTool(workspace, restrict, allowWritePaths))
	}
	if cfg.Tools.IsToolEnabled("append_file") {
		toolsRegistry.Register(tools.NewAppendFileTool(workspace, restrict, allowWritePaths))
	}

	toolsRegistry.Register(tools.NewLogsTool())
	toolsRegistry.Register(tools.NewGitPushTool())
	toolsRegistry.Register(tools.NewCreatePRTool())

	dbPath := filepath.Join(workspace, "sessions.db")
	store, err := session.OpenSQLiteStore(dbPath)
	if err != nil {
		log.Fatalf("open session store: %v", err)
	}

	jsonDir := filepath.Join(workspace, "sessions")
	if n, merr := session.MigrateJSONSessions(jsonDir, store); merr != nil {
		log.Printf("session migration: %d migrated, error: %v", n, merr)
	} else if n > 0 {
		log.Printf("session migration: %d sessions migrated to SQLite", n)
	}

	if n, perr := store.Prune(session.DefaultPruneTTL); perr != nil {
		log.Printf("session prune error: %v", perr)
	} else if n > 0 {
		log.Printf("session prune: %d old sessions removed", n)
	}

	sessionsManager := session.NewLegacyAdapter(store)

	mcpDiscoveryActive := cfg.Tools.MCP.Enabled && cfg.Tools.MCP.Discovery.Enabled
	contextBuilder := NewContextBuilder(workspace).WithToolDiscovery(
		mcpDiscoveryActive && cfg.Tools.MCP.Discovery.UseBM25,
		mcpDiscoveryActive && cfg.Tools.MCP.Discovery.UseRegex,
	)

	agentID := routing.DefaultAgentID
	agentName := ""

	if agentCfg != nil {
		agentID = routing.NormalizeAgentID(agentCfg.ID)
		agentName = agentCfg.Name
	}

	maxIter := defaults.MaxToolIterations
	if maxIter == 0 {
		maxIter = 20
	}

	reminderInterval := defaults.TaskReminderInterval
	if reminderInterval == 0 {
		reminderInterval = 5
	}

	maxTokens := defaults.MaxTokens
	if maxTokens == 0 {
		maxTokens = 8192
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

	// Model routing setup: pre-resolve light model candidates at creation time
	// to avoid repeated model_list lookups on every incoming message.
	var router *routing.Router
	var lightCandidates []providers.FallbackCandidate
	if rc := defaults.Routing; rc != nil && rc.Enabled && rc.LightModel != "" {
		lightModelCfg := providers.ModelConfig{Primary: rc.LightModel}
		resolved := providers.ResolveCandidatesWithLookup(lightModelCfg, defaults.Provider, resolveFromModelList)
		if len(resolved) > 0 {
			router = routing.New(routing.RouterConfig{
				LightModel: rc.LightModel,
				Threshold:  rc.Threshold,
			})
			lightCandidates = resolved
		} else {
			log.Printf("routing: light_model %q not found in model_list — routing disabled for agent %q",
				rc.LightModel, agentID)
		}
	}

	agent := &AgentInstance{
		ID:                        agentID,
		Name:                      agentName,
		Model:                     model,
		Fallbacks:                 fallbacks,
		Workspace:                 workspace,
		MaxIterations:             maxIter,
		TaskReminderInterval:      reminderInterval,
		MaxTokens:                 maxTokens,
		Temperature:               temperature,
		ThinkingLevel:             thinkingLevel,
		ContextWindow:             maxTokens,
		SummarizeMessageThreshold: summarizeMessageThreshold,
		SummarizeTokenPercent:     summarizeTokenPercent,
		Provider:                  provider,
		Sessions:                  sessionsManager,
		ContextBuilder:            contextBuilder,
		Tools:                     toolsRegistry,
		Candidates:                candidates,
		Router:                    router,
		LightCandidates:           lightCandidates,
	}

	// Initialize fork-specific fields (subagents, plan model, worktree pruning).
	agent.initInstanceExt(agentCfg, defaults, cfg)

	return agent
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

// resolvePlanModel resolves the plan model for an agent (used during interviewing/review phases).
func resolvePlanModel(agentCfg *config.AgentConfig, defaults *config.AgentDefaults) string {
	if agentCfg != nil && agentCfg.PlanModel != nil && strings.TrimSpace(agentCfg.PlanModel.Primary) != "" {
		return strings.TrimSpace(agentCfg.PlanModel.Primary)
	}
	return defaults.PlanModel
}

// resolvePlanFallbacks resolves the plan model fallbacks for an agent.
func resolvePlanFallbacks(agentCfg *config.AgentConfig, defaults *config.AgentDefaults) []string {
	if agentCfg != nil && agentCfg.PlanModel != nil && agentCfg.PlanModel.Fallbacks != nil {
		return agentCfg.PlanModel.Fallbacks
	}
	return defaults.PlanModelFallbacks
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
