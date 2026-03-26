package agent

import (
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/mediacache"
	"github.com/sipeed/picoclaw/pkg/orch"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/routing"
	"github.com/sipeed/picoclaw/pkg/session"
	"github.com/sipeed/picoclaw/pkg/stats"
	"github.com/sipeed/picoclaw/pkg/tools"
	"github.com/sipeed/picoclaw/pkg/utils"
)

// loopExt holds fork-specific fields for AgentLoop.
// Embedded in AgentLoop so existing field access (al.stats, al.sessions, etc.) continues to work.
// Upstream additions to AgentLoop won't conflict with these fields.
type loopExt struct {
	stats *stats.Tracker // nil when --stats not passed

	sessions *SessionTracker

	orchBroadcaster *orch.Broadcaster // nil when --orchestration not set

	orchReporter orch.AgentReporter // always non-nil (Noop when disabled)

	planStartPending bool // set by /plan start to trigger LLM execution

	planClearHistory bool // set by /plan start clear to wipe history on transition

	sessionLocks sync.Map // sessionKey → *sessionSemaphore

	activeTasks sync.Map // sessionKey → *activeTask

	done chan struct{} // closed by Close() to stop background goroutines

	saveConfig func(*config.Config) error

	onHeartbeatThreadUpdate func(int)

	mediaCache *mediacache.Cache // nil when workspace unavailable
}

// initLoopExt initializes all fork-specific fields: stats tracker,
// session tracker, orchestration broadcaster, and background goroutines.
// Called from NewAgentLoop after the struct is constructed.
func (al *AgentLoop) initLoopExt(cfg *config.Config, registry *AgentRegistry, enableStats bool) {
	defaultAgent := registry.GetDefaultAgent()

	// Stats tracker
	if enableStats && defaultAgent != nil {
		al.stats = stats.NewTracker(defaultAgent.Workspace)
	}

	// Session tracker
	al.sessions = NewSessionTracker()

	// Orchestration broadcaster — needed if any agent has subagents enabled.
	// Note: instance.go maps defaults.Orchestration → Subagents.Enabled,
	// so --orchestration is automatically reflected here.
	al.orchReporter = orch.Noop
	for _, id := range registry.ListAgentIDs() {
		if a, ok := registry.GetAgent(id); ok && a.Subagents != nil && a.Subagents.Enabled {
			al.orchBroadcaster = orch.NewBroadcaster()
			al.orchReporter = al.orchBroadcaster
			break
		}
	}

	// Media cache (co-located with sessions.db in default agent workspace)
	if defaultAgent != nil {
		cachePath := filepath.Join(defaultAgent.Workspace, "media_cache.db")
		mc, err := mediacache.Open(cachePath)
		if err != nil {
			log.Printf("media cache: %v (caching disabled)", err)
		} else {
			al.mediaCache = mc
		}
	}

	// Shutdown signal channel
	al.done = make(chan struct{})

	// Background GC goroutine
	go al.gcLoop()
}

// closeExt releases fork-specific resources: done channel, stats tracker,
// and session stores for all agents.
func (al *AgentLoop) closeExt() {
	select {
	case <-al.done:
		// already closed
	default:
		close(al.done)
	}

	if al.stats != nil {
		al.stats.Close()
	}

	if al.mediaCache != nil {
		al.mediaCache.Close()
	}

	registry := al.GetRegistry()
	for _, agentID := range registry.ListAgentIDs() {
		if agent, ok := registry.GetAgent(agentID); ok {
			agent.Sessions.Close()
		}
	}
}

// pruneMediaCache removes stale entries from the media cache.
func (al *AgentLoop) pruneMediaCache() {
	if al.mediaCache == nil {
		return
	}
	const mediaCacheTTL = 7 * 24 * time.Hour
	if n, err := al.mediaCache.Prune(mediaCacheTTL); err != nil {
		logger.WarnCF("agent", "media cache prune error", map[string]any{"error": err.Error()})
	} else if n > 0 {
		logger.InfoCF("agent", "media cache pruned", map[string]any{"removed": n})
	}
}

// SetReloadFunc registers a callback to reload config from disk.
// Used by buildCommandsRuntime to wire the /reload command.
func (al *AgentLoop) SetReloadFunc(fn func() error) {
	al.reloadFunc = fn
}

// SetConfigSaver registers a callback to persist config changes.
func (al *AgentLoop) SetConfigSaver(fn func(*config.Config) error) {
	al.saveConfig = fn
}

// SetHeartbeatThreadUpdater registers a callback to apply runtime heartbeat thread updates.
func (al *AgentLoop) SetHeartbeatThreadUpdater(fn func(int)) {
	al.onHeartbeatThreadUpdate = fn
}

// registerOrchestrationTools registers spawn, subagent, answer, and review_plan
// tools for agents with orchestration enabled.
func registerOrchestrationTools(
	cfg *config.Config,
	agent *AgentInstance,
	agentID string,
	registry *AgentRegistry,
	provider providers.LLMProvider,
	msgBus *bus.MessageBus,
	al *AgentLoop,
) {
	if agent.Subagents == nil || !agent.Subagents.Enabled {
		return
	}

	webSearchOpts := tools.WebSearchToolOptions{
		BraveAPIKeys:         config.MergeAPIKeys(cfg.Tools.Web.Brave.APIKey(), cfg.Tools.Web.Brave.APIKeys()),
		BraveMaxResults:      cfg.Tools.Web.Brave.MaxResults,
		BraveEnabled:         cfg.Tools.Web.Brave.Enabled,
		TavilyAPIKeys:        config.MergeAPIKeys(cfg.Tools.Web.Tavily.APIKey(), cfg.Tools.Web.Tavily.APIKeys()),
		TavilyBaseURL:        cfg.Tools.Web.Tavily.BaseURL,
		TavilyMaxResults:     cfg.Tools.Web.Tavily.MaxResults,
		TavilyEnabled:        cfg.Tools.Web.Tavily.Enabled,
		DuckDuckGoMaxResults: cfg.Tools.Web.DuckDuckGo.MaxResults,
		DuckDuckGoEnabled:    cfg.Tools.Web.DuckDuckGo.Enabled,
		PerplexityAPIKeys: config.MergeAPIKeys(
			cfg.Tools.Web.Perplexity.APIKey(),
			cfg.Tools.Web.Perplexity.APIKeys(),
		),
		PerplexityMaxResults: cfg.Tools.Web.Perplexity.MaxResults,
		PerplexityEnabled:    cfg.Tools.Web.Perplexity.Enabled,
	}

	subagentManager := tools.NewSubagentManager(
		provider,
		agent.Model,
		agent.Workspace,
		msgBus,
		al.reporter(),
		webSearchOpts,
	)

	subagentManager.SetLLMOptions(agent.MaxTokens, agent.Temperature)

	// Wire session recorder for DAG persistence.
	recorder := newSessionRecorder(agent.Sessions.(*session.LegacyAdapter))
	conductorKey := routing.BuildAgentMainSessionKey(agent.ID)
	subagentManager.SetSessionRecorder(recorder, conductorKey)

	agent.SubagentMgr = subagentManager

	spawnTool := tools.NewSpawnTool(subagentManager)
	currentAgentID := agentID
	spawnTool.SetAllowlistChecker(func(targetAgentID string) bool {
		return registry.CanSpawnSubagent(currentAgentID, targetAgentID)
	})
	agent.Tools.Register(spawnTool)

	// Register blocking subagent tool alongside spawn
	agent.Tools.Register(tools.NewSubagentTool(subagentManager))

	// Register conductor-side escalation tools (answer questions, review plans)
	agent.Tools.Register(tools.NewAnswerSubagentTool(subagentManager))
	agent.Tools.Register(tools.NewReviewSubagentPlanTool(subagentManager))

	// Set orchestration mode on context builder
	agent.ContextBuilder.SetOrchestrationEnabled(true)
}

// handleTaskIntervention checks if a message is a reply to an active task and
// either cancels the task or injects a user intervention. Returns (response, handled).
func (al *AgentLoop) handleTaskIntervention(msg bus.InboundMessage) (string, bool) {
	taskID, ok := msg.Metadata["task_id"]
	if !ok || taskID == "" {
		return "", false
	}

	val, found := al.activeTasks.Load(taskID)
	if !found {
		// Task not found — fall through to normal processing
		return "", false
	}

	task := val.(*activeTask)

	content := strings.TrimSpace(msg.Content)
	lower := strings.ToLower(content)

	// Check for stop keywords
	stopKeywords := []string{
		"stop", "cancel", "abort",
		"停止", "中止", "やめて", //nolint:gosmopolitan // intentional CJK stop words
	}

	for _, kw := range stopKeywords {
		if lower == kw {
			task.cancel()

			logger.InfoCF("agent", "Task canceled by user intervention",
				map[string]any{"task_id": taskID})

			return "Task canceled.", true
		}
	}

	// Inject message into interrupt channel for the tool loop
	select {
	case task.interrupt <- content:
		logger.InfoCF("agent", "User intervention queued",
			map[string]any{"task_id": taskID, "content": utils.Truncate(content, 80)})
	default:
		logger.WarnCF("agent", "Interrupt channel full, message dropped",
			map[string]any{"task_id": taskID})
	}

	return "Intervention sent to running task.", true
}

// expandForkCommands expands fork-specific /skill, /use, and /plan commands in the message.
// Returns the compact form for history and any forced skill names.
func (al *AgentLoop) expandForkCommands(msg *bus.InboundMessage) (compact string, forcedSkills []string) {
	var expansionCompact string

	// Support both /skill and /use prefixes for skill loading
	content := strings.TrimSpace(msg.Content)
	originalContent := content
	if strings.HasPrefix(content, "/use ") {
		// Rewrite /use → /skill for the skill expander
		msg.Content = "/skill " + content[5:]
	}
	if expanded, cpt, ok := al.expandSkillCommand(*msg); ok {
		msg.Content = expanded
		expansionCompact = cpt
		// Extract skill name
		skillFields := strings.Fields(originalContent)
		if len(skillFields) >= 2 {
			forcedSkills = append(forcedSkills, skillFields[1])
		}
	} else if strings.HasPrefix(originalContent, "/use ") {
		// Skill expansion failed (no message or not found) — restore original /use content
		msg.Content = originalContent
	}

	if expanded, cpt, ok := al.expandPlanCommand(*msg); ok {
		msg.Content = expanded
		expansionCompact = cpt
	}

	return expansionCompact, forcedSkills
}
