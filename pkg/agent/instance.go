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
	ID                            string
	Name                          string
	Model                         string
	Fallbacks                     []string
	Workspace                     string
	MaxIterations                 int
	MaxTokens                     int
	Temperature                   float64
	ContextWindow                 int
	Provider                      providers.LLMProvider
	Sessions                      *session.SessionManager
	ContextBuilder                *ContextBuilder
	Tools                         *tools.ToolRegistry
	SubagentManager               *tools.SubagentManager
	Subagents                     *config.SubagentsConfig
	SkillsFilter                  []string
	Candidates                    []providers.FallbackCandidate
	CompactionMode                string
	CompactionReserveTokens       int
	CompactionKeepRecentTokens    int
	CompactionMaxHistoryShare     float64
	MemoryFlushEnabled            bool
	MemoryFlushSoftThreshold      int
	ContextPruningMode            string
	ContextPruningIncludeChitChat bool
	ContextPruningSoftToolChars   int
	ContextPruningHardToolChars   int
	ContextPruningTriggerRatio    float64
	BootstrapSnapshotEnabled      bool
	MemoryVectorEnabled           bool
	MemoryVectorDimensions        int
	MemoryVectorTopK              int
	MemoryVectorMinScore          float64
	MemoryVectorMaxContextChars   int
	MemoryVectorRecentDailyDays   int
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
	execTool := tools.NewExecToolWithConfig(workspace, restrict, cfg)
	toolsRegistry.Register(execTool)
	toolsRegistry.Register(tools.NewProcessTool(execTool.ProcessManager()))
	toolsRegistry.Register(tools.NewEditFileTool(workspace, restrict))
	toolsRegistry.Register(tools.NewAppendFileTool(workspace, restrict))

	sessionsDir := filepath.Join(workspace, "sessions")
	sessionsManager := session.NewSessionManager(sessionsDir)
	toolsRegistry.Register(tools.NewSessionsListTool(sessionsManager))
	toolsRegistry.Register(tools.NewSessionsHistoryTool(sessionsManager))

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

	compactionMode := strings.TrimSpace(defaults.Compaction.Mode)
	if compactionMode == "" {
		compactionMode = "safeguard"
	}

	compactionReserveTokens := defaults.Compaction.ReserveTokens
	if compactionReserveTokens <= 0 {
		compactionReserveTokens = 2048
	}

	compactionKeepRecentTokens := defaults.Compaction.KeepRecentTokens
	if compactionKeepRecentTokens <= 0 {
		compactionKeepRecentTokens = 2048
	}

	compactionMaxHistoryShare := defaults.Compaction.MaxHistoryShare
	if compactionMaxHistoryShare <= 0 || compactionMaxHistoryShare > 0.9 {
		compactionMaxHistoryShare = 0.5
	}

	memoryFlushEnabled := defaults.Compaction.MemoryFlush.Enabled
	// Preserve default behavior if omitted from config.
	if !defaults.Compaction.MemoryFlush.Enabled && defaults.Compaction.MemoryFlush.SoftThresholdTokens == 0 {
		memoryFlushEnabled = true
	}

	memoryFlushSoftThreshold := defaults.Compaction.MemoryFlush.SoftThresholdTokens
	if memoryFlushSoftThreshold <= 0 {
		memoryFlushSoftThreshold = 1500
	}

	contextPruningMode := strings.TrimSpace(defaults.ContextPruning.Mode)
	if contextPruningMode == "" {
		contextPruningMode = "tools_only"
	}

	contextPruningSoftToolChars := defaults.ContextPruning.SoftToolResultChars
	if contextPruningSoftToolChars <= 0 {
		contextPruningSoftToolChars = 2000
	}

	contextPruningHardToolChars := defaults.ContextPruning.HardToolResultChars
	if contextPruningHardToolChars <= 0 {
		contextPruningHardToolChars = 350
	}

	contextPruningTriggerRatio := defaults.ContextPruning.TriggerRatio
	if contextPruningTriggerRatio <= 0 || contextPruningTriggerRatio >= 1 {
		contextPruningTriggerRatio = 0.8
	}

	bootstrapSnapshotEnabled := defaults.BootstrapSnapshot.Enabled

	memoryVectorEnabled := defaults.MemoryVector.Enabled
	memoryVectorDimensions := defaults.MemoryVector.Dimensions
	if memoryVectorDimensions <= 0 {
		memoryVectorDimensions = defaultMemoryVectorDimensions
	}

	memoryVectorTopK := defaults.MemoryVector.TopK
	if memoryVectorTopK <= 0 {
		memoryVectorTopK = defaultMemoryVectorTopK
	}

	memoryVectorMinScore := defaults.MemoryVector.MinScore
	if memoryVectorMinScore < 0 || memoryVectorMinScore >= 1 {
		memoryVectorMinScore = defaultMemoryVectorMinScore
	}

	memoryVectorMaxContextChars := defaults.MemoryVector.MaxContextChars
	if memoryVectorMaxContextChars <= 0 {
		memoryVectorMaxContextChars = defaultMemoryVectorMaxContextChars
	}

	memoryVectorRecentDailyDays := defaults.MemoryVector.RecentDailyDays
	if memoryVectorRecentDailyDays <= 0 {
		memoryVectorRecentDailyDays = defaultMemoryVectorRecentDailyDays
	}

	contextBuilder.SetRuntimeSettings(ContextRuntimeSettings{
		ContextWindowTokens:      maxTokens,
		PruningMode:              contextPruningMode,
		IncludeOldChitChat:       defaults.ContextPruning.IncludeOldChitChat,
		SoftToolResultChars:      contextPruningSoftToolChars,
		HardToolResultChars:      contextPruningHardToolChars,
		TriggerRatio:             contextPruningTriggerRatio,
		BootstrapSnapshotEnabled: bootstrapSnapshotEnabled,
		MemoryVectorEnabled:      memoryVectorEnabled,
		MemoryVectorDimensions:   memoryVectorDimensions,
		MemoryVectorTopK:         memoryVectorTopK,
		MemoryVectorMinScore:     memoryVectorMinScore,
		MemoryVectorMaxChars:     memoryVectorMaxContextChars,
		MemoryVectorRecentDays:   memoryVectorRecentDailyDays,
	})

	if memoryVectorEnabled {
		toolsRegistry.Register(NewMemorySearchTool(
			contextBuilder.memory,
			memoryVectorTopK,
			memoryVectorMinScore,
		))
		toolsRegistry.Register(NewMemoryGetTool(contextBuilder.memory))
	}

	// Resolve fallback candidates
	modelCfg := providers.ModelConfig{
		Primary:   model,
		Fallbacks: fallbacks,
	}
	candidates := providers.ResolveCandidates(modelCfg, defaults.Provider)

	return &AgentInstance{
		ID:                            agentID,
		Name:                          agentName,
		Model:                         model,
		Fallbacks:                     fallbacks,
		Workspace:                     workspace,
		MaxIterations:                 maxIter,
		MaxTokens:                     maxTokens,
		Temperature:                   temperature,
		ContextWindow:                 maxTokens,
		Provider:                      provider,
		Sessions:                      sessionsManager,
		ContextBuilder:                contextBuilder,
		Tools:                         toolsRegistry,
		Subagents:                     subagents,
		SkillsFilter:                  skillsFilter,
		Candidates:                    candidates,
		CompactionMode:                compactionMode,
		CompactionReserveTokens:       compactionReserveTokens,
		CompactionKeepRecentTokens:    compactionKeepRecentTokens,
		CompactionMaxHistoryShare:     compactionMaxHistoryShare,
		MemoryFlushEnabled:            memoryFlushEnabled,
		MemoryFlushSoftThreshold:      memoryFlushSoftThreshold,
		ContextPruningMode:            contextPruningMode,
		ContextPruningIncludeChitChat: defaults.ContextPruning.IncludeOldChitChat,
		ContextPruningSoftToolChars:   contextPruningSoftToolChars,
		ContextPruningHardToolChars:   contextPruningHardToolChars,
		ContextPruningTriggerRatio:    contextPruningTriggerRatio,
		BootstrapSnapshotEnabled:      bootstrapSnapshotEnabled,
		MemoryVectorEnabled:           memoryVectorEnabled,
		MemoryVectorDimensions:        memoryVectorDimensions,
		MemoryVectorTopK:              memoryVectorTopK,
		MemoryVectorMinScore:          memoryVectorMinScore,
		MemoryVectorMaxContextChars:   memoryVectorMaxContextChars,
		MemoryVectorRecentDailyDays:   memoryVectorRecentDailyDays,
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
