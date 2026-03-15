package agent

import (
	"strings"
	"sync"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/orch"
	"github.com/sipeed/picoclaw/pkg/stats"
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

	// Shutdown signal channel
	al.done = make(chan struct{})

	// Background GC goroutine
	go al.gcLoop()
}

// SetConfigSaver registers a callback to persist config changes.
func (al *AgentLoop) SetConfigSaver(fn func(*config.Config) error) {
	al.saveConfig = fn
}

// SetHeartbeatThreadUpdater registers a callback to apply runtime heartbeat thread updates.
func (al *AgentLoop) SetHeartbeatThreadUpdater(fn func(int)) {
	al.onHeartbeatThreadUpdate = fn
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

// expandForkCommands expands fork-specific /skill and /plan commands in the message.
// Returns the modified message and the compact form for history.
func (al *AgentLoop) expandForkCommands(msg *bus.InboundMessage) string {
	var expansionCompact string

	if expanded, compact, ok := al.expandSkillCommand(*msg); ok {
		msg.Content = expanded
		expansionCompact = compact
	}

	if expanded, compact, ok := al.expandPlanCommand(*msg); ok {
		msg.Content = expanded
		expansionCompact = compact
	}

	return expansionCompact
}
