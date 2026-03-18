package agent

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/session"
	"github.com/sipeed/picoclaw/pkg/tools"
)

// ====================== Context Keys ======================
type turnStateKeyType struct{}

var turnStateKey = turnStateKeyType{}

func withTurnState(ctx context.Context, ts *turnState) context.Context {
	return context.WithValue(ctx, turnStateKey, ts)
}

// TurnStateFromContext retrieves turnState from context (exported for tools)
func TurnStateFromContext(ctx context.Context) *turnState {
	return turnStateFromContext(ctx)
}

func turnStateFromContext(ctx context.Context) *turnState {
	ts, _ := ctx.Value(turnStateKey).(*turnState)
	return ts
}

// ====================== turnState ======================

type turnState struct {
	ctx                  context.Context
	cancelFunc           context.CancelFunc // Used to cancel all children when this turn finishes
	turnID               string
	parentTurnID         string
	depth                int
	childTurnIDs         []string // MUST be accessed under mu lock or maybe add a getter method
	pendingResults       chan *tools.ToolResult
	session              session.SessionStore
	initialHistoryLength int // Snapshot of session history length at turn start, for rollback on hard abort
	mu                   sync.Mutex
	isFinished           bool          // MUST be accessed under mu lock
	closeOnce            sync.Once     // Ensures pendingResults channel is closed exactly once
	concurrencySem       chan struct{} // Limits concurrent child sub-turns

	// parentEnded signals that the parent turn has finished gracefully.
	// Child SubTurns should check this via IsParentEnded() to decide whether
	// to continue running (Critical=true) or exit gracefully (Critical=false).
	parentEnded atomic.Bool

	// parentTurnState holds a reference to the parent turnState.
	// This allows child SubTurns to check if the parent has ended.
	// Nil for root turns.
	parentTurnState *turnState

	// lastFinishReason stores the finish_reason from the last LLM call.
	// Used by SubTurn to detect truncation and retry.
	// MUST be accessed under mu lock.
	lastFinishReason string
}

// ====================== Public API ======================

// TurnInfo provides read-only information about an active turn.
type TurnInfo struct {
	TurnID       string
	ParentTurnID string
	Depth        int
	ChildTurnIDs []string
	IsFinished   bool
}

// GetActiveTurn retrieves information about the currently active turn for a session.
// Returns nil if no active turn exists for the given session key.
func (al *AgentLoop) GetActiveTurn(sessionKey string) *TurnInfo {
	tsInterface, ok := al.activeTurnStates.Load(sessionKey)
	if !ok {
		return nil
	}

	ts, ok := tsInterface.(*turnState)
	if !ok {
		return nil
	}

	return ts.Info()
}

// Info returns a read-only snapshot of the turn state information.
// This method is thread-safe and can be called concurrently.
func (ts *turnState) Info() *TurnInfo {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	// Create a copy of childTurnIDs to avoid race conditions
	childIDs := make([]string, len(ts.childTurnIDs))
	copy(childIDs, ts.childTurnIDs)

	return &TurnInfo{
		TurnID:       ts.turnID,
		ParentTurnID: ts.parentTurnID,
		Depth:        ts.depth,
		ChildTurnIDs: childIDs,
		IsFinished:   ts.isFinished,
	}
}

// ====================== Helper Functions ======================

func newTurnState(ctx context.Context, id string, parent *turnState) *turnState {
	// Note: We don't create a new context with cancel here because the caller
	// (spawnSubTurn) already creates one. The turnState stores the context and
	// cancelFunc provided by the caller to avoid redundant context wrapping.
	return &turnState{
		ctx:            ctx,
		cancelFunc:     nil, // Will be set by the caller
		turnID:         id,
		parentTurnID:   parent.turnID,
		depth:          parent.depth + 1,
		session:        newEphemeralSession(parent.session),
		parentTurnState: parent, // Store reference to parent for IsParentEnded() checks
		// NOTE: In this PoC, I use a fixed-size channel (16).
		// Under high concurrency or long-running sub-turns, this might fill up and cause
		// intermediate results to be discarded in deliverSubTurnResult.
		// For production, consider an unbounded queue or a blocking strategy with backpressure.
		pendingResults: make(chan *tools.ToolResult, 16),
		concurrencySem: make(chan struct{}, maxConcurrentSubTurns),
	}
}

// IsParentEnded returns true if the parent turn has finished gracefully.
// This is safe to call from child SubTurn goroutines.
// Returns false if this is a root turn (no parent).
func (ts *turnState) IsParentEnded() bool {
	if ts.parentTurnState == nil {
		return false
	}
	return ts.parentTurnState.parentEnded.Load()
}

// SetLastFinishReason updates the last finish reason (thread-safe).
func (ts *turnState) SetLastFinishReason(reason string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.lastFinishReason = reason
}

// GetLastFinishReason retrieves the last finish reason (thread-safe).
func (ts *turnState) GetLastFinishReason() string {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return ts.lastFinishReason
}

// IsParentEnded is a convenience method to check if parent ended.
// It returns the value of the parent's parentEnded atomic flag.

// Finish marks the turn as finished.
//
// If isHardAbort is true (Hard Abort):
//   - Cancels all child contexts immediately via cancelFunc
//   - Used for user-initiated termination (e.g., "stop now")
//
// If isHardAbort is false (Graceful Finish):
//   - Only signals parentEnded for graceful child exit
//   - Children check IsParentEnded() and decide whether to continue or exit
//   - Critical SubTurns continue running and deliver orphan results
//   - Non-Critical SubTurns exit gracefully without error
//
// In both cases, the pendingResults channel is closed to signal
// that no more results will be delivered.
func (ts *turnState) Finish(isHardAbort bool) {
	ts.mu.Lock()
	ts.isFinished = true
	resultChan := ts.pendingResults
	ts.mu.Unlock()

	if isHardAbort {
		// Hard abort: immediately cancel all children
		if ts.cancelFunc != nil {
			ts.cancelFunc()
		}
	} else {
		// Graceful finish: signal parent ended, let children decide
		ts.parentEnded.Store(true)
	}

	// Use sync.Once to ensure the channel is closed exactly once, even if Finish() is called concurrently.
	// This prevents "close of closed channel" panics.
	ts.closeOnce.Do(func() {
		if resultChan != nil {
			close(resultChan)
			// Drain any remaining results from the channel and emit them as orphan events.
			// This prevents goroutine leaks and ensures all results are accounted for.
			ts.drainPendingResults(resultChan)
		}
	})
}

// drainPendingResults drains all remaining results from the closed channel
// and emits them as orphan events. This must be called after the channel is closed.
func (ts *turnState) drainPendingResults(ch chan *tools.ToolResult) {
	for result := range ch {
		if result != nil {
			MockEventBus.Emit(SubTurnOrphanResultEvent{
				ParentID: ts.turnID,
				ChildID:  "unknown", // We don't know which child this came from
				Result:   result,
			})
		}
	}
}

// ====================== Ephemeral Session Store ======================

// ephemeralSessionStore is a pure in-memory SessionStore for SubTurns.
// It never writes to disk, keeping sub-turn history isolated from the parent session.
// It automatically truncates history when it exceeds maxEphemeralHistorySize to prevent memory accumulation.
type ephemeralSessionStore struct {
	mu      sync.Mutex
	history []providers.Message
	summary string
}

func (e *ephemeralSessionStore) AddMessage(sessionKey, role, content string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.history = append(e.history, providers.Message{Role: role, Content: content})
	e.autoTruncate()
}

func (e *ephemeralSessionStore) AddFullMessage(sessionKey string, msg providers.Message) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.history = append(e.history, msg)
	e.autoTruncate()
}

// autoTruncate automatically limits history size to prevent memory accumulation.
// Must be called with mu held.
func (e *ephemeralSessionStore) autoTruncate() {
	if len(e.history) > maxEphemeralHistorySize {
		// Keep only the most recent messages
		e.history = e.history[len(e.history)-maxEphemeralHistorySize:]
	}
}

func (e *ephemeralSessionStore) GetHistory(key string) []providers.Message {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]providers.Message, len(e.history))
	copy(out, e.history)
	return out
}

func (e *ephemeralSessionStore) GetSummary(key string) string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.summary
}

func (e *ephemeralSessionStore) SetSummary(key, summary string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.summary = summary
}

func (e *ephemeralSessionStore) SetHistory(key string, history []providers.Message) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.history = make([]providers.Message, len(history))
	copy(e.history, history)
}

func (e *ephemeralSessionStore) TruncateHistory(key string, keepLast int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.history) > keepLast {
		e.history = e.history[len(e.history)-keepLast:]
	}
}

func (e *ephemeralSessionStore) Save(key string) error { return nil }
func (e *ephemeralSessionStore) Close() error          { return nil }

// newEphemeralSession creates a new isolated ephemeral session for a sub-turn.
//
// IMPORTANT: The parent session parameter is intentionally unused (marked with _).
// This is by design according to issue #1316: sub-turns use completely isolated
// ephemeral sessions that do NOT inherit history from the parent session.
//
// Rationale for isolation:
//   - Sub-turns are independent execution contexts with their own prompts
//   - Inheriting parent history could cause context pollution
//   - Each sub-turn should start with a clean slate
//   - Memory is managed independently (auto-truncation at maxEphemeralHistorySize)
//   - Results are communicated back via the result channel, not via shared history
//
// If future requirements need parent history inheritance, this design decision
// should be reconsidered with careful attention to memory management and context size.
func newEphemeralSession(_ session.SessionStore) session.SessionStore {
	return &ephemeralSessionStore{}
}
