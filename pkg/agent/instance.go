package agent

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/git"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/routing"
	"github.com/sipeed/picoclaw/pkg/session"
	"github.com/sipeed/picoclaw/pkg/tools"
)

// AgentInstance represents a fully configured agent with its own workspace,

// session manager, context builder, and tool registry.

type AgentInstance struct {
	ID string

	Name string

	Model string

	Fallbacks []string

	Workspace string

	MaxIterations int

	TaskReminderInterval int

	MaxTokens int

	Temperature float64

	ContextWindow int

	Provider providers.LLMProvider

	Sessions *session.LegacyAdapter

	ContextBuilder *ContextBuilder

	Tools *tools.ToolRegistry

	Subagents *config.SubagentsConfig

	SkillsFilter []string

	Candidates []providers.FallbackCandidate

	PlanModel string

	PlanFallbacks []string

	PlanCandidates []providers.FallbackCandidate

	// SubagentMgr is set during registerSharedTools when orchestration is enabled.

	// Used by runAgentLoop to wait for spawned subagents before worktree cleanup.

	SubagentMgr *tools.SubagentManager

	// Interview staleness tracking: consecutive turns where MEMORY.md was not updated.

	interviewStaleCount int

	interviewMemoryLen int

	// Per-session worktree isolation

	worktrees map[string]*git.WorktreeInfo // sessionKey → worktree

	worktreeMu sync.RWMutex
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

	execTool, err := tools.NewExecToolWithConfig(workspace, restrict, cfg)
	if err != nil {
		log.Fatalf("Critical error: unable to initialize exec tool: %v", err)
	}

	toolsRegistry.Register(execTool)

	toolsRegistry.Register(tools.NewBgMonitorTool(execTool))

	toolsRegistry.Register(tools.NewEditFileTool(workspace, restrict))

	toolsRegistry.Register(tools.NewAppendFileTool(workspace, restrict))

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

	sessionsManager := session.NewLegacyAdapter(store)

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

	// Apply defaults.Orchestration: if the flag is set, ensure orchestration is enabled.

	if defaults.Orchestration {
		if subagents == nil {
			subagents = &config.SubagentsConfig{Enabled: true}
		} else {
			subagents.Enabled = true
		}
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

	// Resolve fallback candidates

	modelCfg := providers.ModelConfig{
		Primary: model,

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

	// Resolve plan model (for interviewing/review phases)

	planModel := resolvePlanModel(agentCfg, defaults)

	planFallbacks := resolvePlanFallbacks(agentCfg, defaults)

	var planCandidates []providers.FallbackCandidate

	if planModel != "" {
		planModelCfg := providers.ModelConfig{
			Primary: planModel,

			Fallbacks: planFallbacks,
		}

		planCandidates = providers.ResolveCandidates(planModelCfg, defaults.Provider)
	}

	// Startup cleanup: prune orphaned worktrees

	worktreesDir := filepath.Join(workspace, ".worktrees")

	if repoRoot := git.FindRepoRoot(workspace); repoRoot != "" {
		git.PruneOrphaned(repoRoot, worktreesDir)
	}

	return &AgentInstance{
		ID: agentID,

		Name: agentName,

		Model: model,

		Fallbacks: fallbacks,

		Workspace: workspace,

		MaxIterations: maxIter,

		TaskReminderInterval: reminderInterval,

		MaxTokens: maxTokens,

		Temperature: temperature,

		ContextWindow: maxTokens,

		Provider: provider,

		Sessions: sessionsManager,

		ContextBuilder: contextBuilder,

		Tools: toolsRegistry,

		Subagents: subagents,

		SkillsFilter: skillsFilter,

		Candidates: candidates,

		PlanModel: planModel,

		PlanFallbacks: planFallbacks,

		PlanCandidates: planCandidates,
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

// ActivateWorktree creates a worktree for a session.

// projectDir is the git repository to create the worktree in.

// If empty, falls back to ai.Workspace.

// Worktree path: <workspace>/.worktrees/<branch-basename>/

func (ai *AgentInstance) ActivateWorktree(sessionKey, taskName, projectDir string) (*git.WorktreeInfo, error) {
	if projectDir == "" {
		projectDir = ai.Workspace
	}

	repoRoot := git.FindRepoRoot(projectDir)

	if repoRoot == "" {
		return nil, fmt.Errorf("directory is not a git repository: %s", projectDir)
	}

	branchName := git.SanitizeBranchName(taskName)

	baseName := git.BranchBaseName(branchName)

	wtPath := filepath.Join(ai.Workspace, ".worktrees", baseName)

	wt, err := git.CreateWorktree(repoRoot, wtPath, branchName)
	if err != nil {
		return nil, err
	}

	ai.worktreeMu.Lock()

	if ai.worktrees == nil {
		ai.worktrees = make(map[string]*git.WorktreeInfo)
	}

	ai.worktrees[sessionKey] = wt

	ai.worktreeMu.Unlock()

	return wt, nil
}

// DeactivateWorktree safe-disposes the session's worktree.

func (ai *AgentInstance) DeactivateWorktree(sessionKey, commitMsg string, discard bool) (*git.DisposeResult, error) {
	ai.worktreeMu.Lock()

	wt, ok := ai.worktrees[sessionKey]

	if ok {
		delete(ai.worktrees, sessionKey)
	}

	ai.worktreeMu.Unlock()

	if !ok || wt == nil {
		return nil, nil
	}

	repoRoot := git.FindRepoRoot(ai.Workspace)

	if repoRoot == "" {
		return nil, fmt.Errorf("workspace is not a git repository")
	}

	// Even on discard, SafeDispose auto-commits first for safety

	if commitMsg != "" && git.HasUncommittedChanges(wt.Path) {
		_ = git.AutoCommit(wt.Path, commitMsg)
	}

	result := git.SafeDispose(repoRoot, wt)

	return &result, nil
}

// GetWorktree returns the session's active worktree, or nil.

func (ai *AgentInstance) GetWorktree(sessionKey string) *git.WorktreeInfo {
	ai.worktreeMu.RLock()

	defer ai.worktreeMu.RUnlock()

	return ai.worktrees[sessionKey]
}

// IsInWorktree returns true if the session has an active worktree.

func (ai *AgentInstance) IsInWorktree(sessionKey string) bool {
	return ai.GetWorktree(sessionKey) != nil
}

// EffectiveWorkspace returns worktree path for session, or original Workspace.

func (ai *AgentInstance) EffectiveWorkspace(sessionKey string) string {
	if wt := ai.GetWorktree(sessionKey); wt != nil {
		return wt.Path
	}

	return ai.Workspace
}

// GetWorktreeBranch returns the branch name for the session's worktree, or "".

func (ai *AgentInstance) GetWorktreeBranch(sessionKey string) string {
	if wt := ai.GetWorktree(sessionKey); wt != nil {
		return wt.Branch
	}

	return ""
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
