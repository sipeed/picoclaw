package agent

import (
	"sync"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/orch"
	"github.com/sipeed/picoclaw/pkg/stats"
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

// SetConfigSaver registers a callback to persist config changes.
func (al *AgentLoop) SetConfigSaver(fn func(*config.Config) error) {
	al.saveConfig = fn
}

// SetHeartbeatThreadUpdater registers a callback to apply runtime heartbeat thread updates.
func (al *AgentLoop) SetHeartbeatThreadUpdater(fn func(int)) {
	al.onHeartbeatThreadUpdate = fn
}
