package multiagent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// Spawn concurrency defaults.
// MaxChildrenPerAgent follows NVIDIA's stream scheduling pattern:
// limit parallel work to prevent resource exhaustion while maximizing throughput.
const (
	DefaultMaxChildren   = 5
	DefaultSpawnTimeout  = 5 * time.Minute
)

// SpawnRequest describes an async agent invocation.
type SpawnRequest struct {
	FromAgentID string
	ToAgentID   string
	Task        string
	Context     map[string]string // k-v to write to blackboard
	Depth       int
	Visited     []string
	MaxDepth    int
	ParentRunKey string
}

// SpawnResult is returned immediately to the caller (fire-and-forget).
type SpawnResult struct {
	RunID      string // unique identifier for this spawn
	SessionKey string // child session key for tracking
	Status     string // "accepted" or "rejected"
	Error      string // rejection reason if status != "accepted"
}

// SpawnOutcome is the final result written to the announcer when the spawn completes.
type SpawnOutcome struct {
	RunID      string
	SessionKey string
	AgentID    string
	Content    string
	Iterations int
	Success    bool
	Error      string
	Duration   time.Duration
}

// SpawnManager orchestrates async agent spawns with concurrency limiting
// (semaphore pattern, inspired by NVIDIA CUDA stream scheduling and
// Apple GCD quality-of-service queues).
type SpawnManager struct {
	registry    *RunRegistry
	announcer   *Announcer
	maxChildren int
	timeout     time.Duration

	// Per-parent semaphore: limits concurrent children per session.
	// Google's MapReduce uses similar fan-out caps per mapper.
	semaphores sync.Map // parentSessionKey -> *semaphore
}

type semaphore struct {
	ch chan struct{}
}

func newSemaphore(max int) *semaphore {
	return &semaphore{ch: make(chan struct{}, max)}
}

func (s *semaphore) acquire() bool {
	select {
	case s.ch <- struct{}{}:
		return true
	default:
		return false
	}
}

func (s *semaphore) release() {
	<-s.ch
}

func (s *semaphore) count() int {
	return len(s.ch)
}

// NewSpawnManager creates a spawn manager with the given limits.
func NewSpawnManager(registry *RunRegistry, announcer *Announcer, maxChildren int, timeout time.Duration) *SpawnManager {
	if maxChildren <= 0 {
		maxChildren = DefaultMaxChildren
	}
	if timeout <= 0 {
		timeout = DefaultSpawnTimeout
	}
	return &SpawnManager{
		registry:    registry,
		announcer:   announcer,
		maxChildren: maxChildren,
		timeout:     timeout,
	}
}

// AsyncSpawn launches an agent in a background goroutine and returns immediately.
// Inspired by Google's fan-out pattern and Anthropic's parallel tool execution.
// The result is delivered via the Announcer when the spawn completes.
func (sm *SpawnManager) AsyncSpawn(
	ctx context.Context,
	resolver AgentResolver,
	board *Blackboard,
	req SpawnRequest,
	channel, chatID string,
) *SpawnResult {
	// Generate unique run ID and session key.
	runID := fmt.Sprintf("spawn:%s:%s:%d", req.FromAgentID, req.ToAgentID, time.Now().UnixNano())
	childSessionKey := fmt.Sprintf("spawn:%s:%s:%d:%d", req.FromAgentID, req.ToAgentID, req.Depth, time.Now().UnixNano())

	// Acquire per-parent semaphore (NVIDIA stream scheduling pattern).
	sem := sm.getOrCreateSemaphore(req.ParentRunKey)
	if !sem.acquire() {
		return &SpawnResult{
			RunID:      runID,
			SessionKey: childSessionKey,
			Status:     "rejected",
			Error:      fmt.Sprintf("max concurrent children reached (%d/%d) for parent session", sem.count(), sm.maxChildren),
		}
	}

	// Create cancellable context with timeout to prevent goroutine leaks.
	// Microsoft Azure Functions uses similar timeout patterns for durable functions.
	spawnCtx, cancel := context.WithTimeout(ctx, sm.timeout)

	// Register in RunRegistry for cascade cancellation (built in Phase 3d).
	sm.registry.Register(&ActiveRun{
		SessionKey: childSessionKey,
		AgentID:    req.ToAgentID,
		ParentKey:  req.ParentRunKey,
		Cancel:     cancel,
		StartedAt:  time.Now(),
	})

	logger.InfoCF("spawn", "Async spawn started", map[string]interface{}{
		"run_id":    runID,
		"from":      req.FromAgentID,
		"to":        req.ToAgentID,
		"depth":     req.Depth,
		"parent":    req.ParentRunKey,
		"timeout":   sm.timeout.String(),
		"active":    sem.count(),
		"max":       sm.maxChildren,
	})

	// Fire-and-forget goroutine (Google MapReduce worker pattern).
	go func() {
		defer cancel()
		defer sem.release()
		defer sm.registry.Deregister(childSessionKey)

		start := time.Now()

		// Execute the handoff synchronously inside the goroutine.
		result := ExecuteHandoff(spawnCtx, resolver, board, HandoffRequest{
			FromAgentID:  req.FromAgentID,
			ToAgentID:    req.ToAgentID,
			Task:         req.Task,
			Context:      req.Context,
			Depth:        req.Depth,
			Visited:      req.Visited,
			MaxDepth:     req.MaxDepth,
			ParentRunKey: childSessionKey,
		}, channel, chatID)

		outcome := &SpawnOutcome{
			RunID:      runID,
			SessionKey: childSessionKey,
			AgentID:    req.ToAgentID,
			Content:    result.Content,
			Iterations: result.Iterations,
			Success:    result.Success,
			Error:      result.Error,
			Duration:   time.Since(start),
		}

		// Push result to parent via Announcer (Anthropic's auto-announce pattern).
		if sm.announcer != nil {
			sm.announcer.Deliver(req.ParentRunKey, &Announcement{
				FromSessionKey: childSessionKey,
				ToSessionKey:   req.ParentRunKey,
				RunID:          runID,
				AgentID:        req.ToAgentID,
				Content:        formatOutcomeMessage(outcome),
				Outcome:        outcome,
			})
		}

		logger.InfoCF("spawn", "Async spawn completed", map[string]interface{}{
			"run_id":     runID,
			"agent_id":   req.ToAgentID,
			"success":    result.Success,
			"iterations": result.Iterations,
			"duration":   outcome.Duration.Round(time.Millisecond).String(),
		})
	}()

	return &SpawnResult{
		RunID:      runID,
		SessionKey: childSessionKey,
		Status:     "accepted",
	}
}

// ActiveChildCount returns the number of active children for a parent session.
func (sm *SpawnManager) ActiveChildCount(parentSessionKey string) int {
	return len(sm.registry.GetChildren(parentSessionKey))
}

func (sm *SpawnManager) getOrCreateSemaphore(parentKey string) *semaphore {
	if v, ok := sm.semaphores.Load(parentKey); ok {
		return v.(*semaphore)
	}
	sem := newSemaphore(sm.maxChildren)
	actual, _ := sm.semaphores.LoadOrStore(parentKey, sem)
	return actual.(*semaphore)
}

func formatOutcomeMessage(o *SpawnOutcome) string {
	if !o.Success {
		return fmt.Sprintf("[Subagent %q failed after %s: %s]", o.AgentID, o.Duration.Round(time.Millisecond), o.Error)
	}
	return fmt.Sprintf("[Subagent %q completed in %s (%d iterations)]:\n%s",
		o.AgentID, o.Duration.Round(time.Millisecond), o.Iterations, o.Content)
}
