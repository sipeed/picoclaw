package multiagent

import (
	"context"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// ActiveRun represents a running handoff or spawn that can be cancelled.
type ActiveRun struct {
	SessionKey string
	AgentID    string
	ParentKey  string             // parent session key ("" for top-level)
	Cancel     context.CancelFunc // cancels this run's context
	StartedAt  time.Time
}

// RunRegistry tracks active agent runs for cascade cancellation.
// Thread-safe via sync.Map.
type RunRegistry struct {
	runs sync.Map // sessionKey -> *ActiveRun
}

// NewRunRegistry creates an empty run registry.
func NewRunRegistry() *RunRegistry {
	return &RunRegistry{}
}

// Register adds an active run to the registry.
func (r *RunRegistry) Register(run *ActiveRun) {
	r.runs.Store(run.SessionKey, run)
	logger.DebugCF("cascade", "Run registered",
		map[string]interface{}{
			"session_key": run.SessionKey,
			"agent_id":    run.AgentID,
			"parent_key":  run.ParentKey,
		})
}

// Deregister removes a run from the registry (normal completion).
func (r *RunRegistry) Deregister(sessionKey string) {
	r.runs.Delete(sessionKey)
	logger.DebugCF("cascade", "Run deregistered",
		map[string]interface{}{
			"session_key": sessionKey,
		})
}

// CascadeStop cancels a run and all its descendants.
// Returns the number of runs cancelled. Uses a seen-set to prevent infinite loops.
func (r *RunRegistry) CascadeStop(sessionKey string) int {
	seen := make(map[string]bool)
	killed := r.cascadeStop(sessionKey, seen)
	if killed > 0 {
		logger.InfoCF("cascade", "Cascade stop completed",
			map[string]interface{}{
				"root_key": sessionKey,
				"killed":   killed,
			})
	}
	return killed
}

func (r *RunRegistry) cascadeStop(sessionKey string, seen map[string]bool) int {
	if seen[sessionKey] {
		return 0
	}
	seen[sessionKey] = true
	killed := 0

	// Cancel and remove this run
	if v, ok := r.runs.LoadAndDelete(sessionKey); ok {
		run := v.(*ActiveRun)
		run.Cancel()
		killed++
		logger.DebugCF("cascade", "Run cancelled",
			map[string]interface{}{
				"session_key": sessionKey,
				"agent_id":    run.AgentID,
			})
	}

	// Find and cascade-stop all children (runs whose ParentKey == sessionKey)
	r.runs.Range(func(key, value interface{}) bool {
		childRun := value.(*ActiveRun)
		if childRun.ParentKey == sessionKey {
			killed += r.cascadeStop(key.(string), seen)
		}
		return true
	})

	return killed
}

// StopAll cancels every active run. Returns the number cancelled.
func (r *RunRegistry) StopAll() int {
	killed := 0
	r.runs.Range(func(key, value interface{}) bool {
		run := value.(*ActiveRun)
		run.Cancel()
		r.runs.Delete(key)
		killed++
		return true
	})
	if killed > 0 {
		logger.InfoCF("cascade", "Stop all completed",
			map[string]interface{}{"killed": killed})
	}
	return killed
}

// ActiveCount returns the number of currently active runs.
func (r *RunRegistry) ActiveCount() int {
	count := 0
	r.runs.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	return count
}

// GetChildren returns session keys of all direct children of the given parent.
func (r *RunRegistry) GetChildren(parentKey string) []string {
	var children []string
	r.runs.Range(func(key, value interface{}) bool {
		run := value.(*ActiveRun)
		if run.ParentKey == parentKey {
			children = append(children, key.(string))
		}
		return true
	})
	return children
}
