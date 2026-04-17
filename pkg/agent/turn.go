package agent

import (
	"context"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/session"
	"github.com/sipeed/picoclaw/pkg/tools"
)

type TurnPhase string

const (
	TurnPhaseSetup      TurnPhase = "setup"
	TurnPhaseRunning    TurnPhase = "running"
	TurnPhaseTools      TurnPhase = "tools"
	TurnPhaseFinalizing TurnPhase = "finalizing"
	TurnPhaseCompleted  TurnPhase = "completed"
	TurnPhaseAborted    TurnPhase = "aborted"
)

type ActiveTurnInfo struct {
	TurnID       string
	AgentID      string
	SessionKey   string
	Channel      string
	ChatID       string
	UserMessage  string
	Phase        TurnPhase
	Iteration    int
	StartedAt    time.Time
	Depth        int
	ParentTurnID string
	ChildTurnIDs []string
}

type turnResult struct {
	finalContent string
	status       TurnEndStatus
	followUps    []bus.InboundMessage
}

type turnState struct {
	mu sync.RWMutex

	agent *AgentInstance
	opts  processOptions
	scope turnEventScope

	turnID     string
	agentID    string
	sessionKey string
	turnCtx    *TurnContext

	channel     string
	chatID      string
	userMessage string
	media       []string

	phase        TurnPhase
	iteration    int
	startedAt    time.Time
	finalContent string

	followUps []bus.InboundMessage

	gracefulInterrupt     bool
	gracefulInterruptHint string
	gracefulTerminalUsed  bool
	hardAbort             bool
	providerCancel        context.CancelFunc
	turnCancel            context.CancelFunc

	restorePointHistory []providers.Message
	restorePointSummary string
	persistedMessages   []providers.Message

	// SubTurn support (from HEAD)
	depth                int                    // SubTurn depth (0 for root turn)
	parentTurnID         string                 // Parent turn ID (empty for root turn)
	childTurnIDs         []string               // Child turn IDs
	pendingResults       chan *tools.ToolResult // Channel for SubTurn results
	concurrencySem       chan struct{}          // Semaphore for limiting concurrent SubTurns
	isFinished           atomic.Bool            // Whether this turn has finished
	session              session.SessionStore   // Session store reference
	initialHistoryLength int                    // Snapshot of history length at turn start

	// Additional SubTurn fields
	ctx             context.Context    // Context for this turn
	cancelFunc      context.CancelFunc // Cancel function for this turn's context
	critical        bool               // Whether this SubTurn should continue after parent ends
	parentTurnState *turnState         // Reference to parent turnState
	parentEnded     atomic.Bool        // Whether parent has ended
	closeOnce       sync.Once          // Ensures pendingResults channel is closed once
	finishedChan    chan struct{}      // Closed when turn finishes

	// Token budget tracking
	tokenBudget      *atomic.Int64        // Shared token budget counter
	lastFinishReason string               // Last LLM finish_reason
	lastUsage        *providers.UsageInfo // Last LLM usage info

	// Back-reference to the owning AgentLoop (set for SubTurns only, used for hard abort cascade)
	al *AgentLoop
}

func newTurnState(agent *AgentInstance, opts processOptions, scope turnEventScope) *turnState {
	ts := &turnState{
		agent:       agent,
		opts:        opts,
		scope:       scope,
		turnID:      scope.turnID,
		agentID:     agent.ID,
		sessionKey:  opts.Dispatch.SessionKey,
		turnCtx:     cloneTurnContext(scope.context),
		channel:     opts.Dispatch.Channel(),
		chatID:      opts.Dispatch.ChatID(),
		userMessage: opts.Dispatch.UserMessage,
		media:       append([]string(nil), opts.Dispatch.Media...),
		phase:       TurnPhaseSetup,
		startedAt:   time.Now(),
	}

	// Bind session store and capture initial history length for rollback logic
	if agent != nil && agent.Sessions != nil {
		ts.session = agent.Sessions
		ts.initialHistoryLength = len(agent.Sessions.GetHistory(opts.Dispatch.SessionKey))
	}

	return ts
}

func (al *AgentLoop) registerActiveTurn(ts *turnState) {
	al.activeTurnStates.Store(ts.sessionKey, ts)
}

func (al *AgentLoop) clearActiveTurn(ts *turnState) {
	al.activeTurnStates.Delete(ts.sessionKey)
}

func (al *AgentLoop) getActiveTurnState(sessionKey string) *turnState {
	if val, ok := al.activeTurnStates.Load(sessionKey); ok {
		if ts, ok := val.(*turnState); ok {
			return ts
		}
		// Unexpected non-*turnState value — treat as "no active turn" to avoid
		// panics. This should not happen under normal operation.
	}
	return nil
}

// getAnyActiveTurnState returns any active turn state (for backward compatibility)
func (al *AgentLoop) getAnyActiveTurnState() *turnState {
	var firstTS *turnState
	al.activeTurnStates.Range(func(key, value any) bool {
		if ts, ok := value.(*turnState); ok {
			firstTS = ts
			return false
		}
		return true
	})
	return firstTS
}

func (al *AgentLoop) GetActiveTurn() *ActiveTurnInfo {
	// For backward compatibility, return the first active turn found
	// In the new architecture, there can be multiple concurrent turns
	var firstTS *turnState
	al.activeTurnStates.Range(func(key, value any) bool {
		if ts, ok := value.(*turnState); ok {
			firstTS = ts
			return false
		}
		return true
	})
	if firstTS == nil {
		return nil
	}
	info := firstTS.snapshot()
	return &info
}

func (al *AgentLoop) GetActiveTurnBySession(sessionKey string) *ActiveTurnInfo {
	ts := al.getActiveTurnState(sessionKey)
	if ts == nil {
		return nil
	}
	info := ts.snapshot()
	return &info
}

func (ts *turnState) snapshot() ActiveTurnInfo {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	return ActiveTurnInfo{
		TurnID:       ts.turnID,
		AgentID:      ts.agentID,
		SessionKey:   ts.sessionKey,
		Channel:      ts.channel,
		ChatID:       ts.chatID,
		UserMessage:  ts.userMessage,
		Phase:        ts.phase,
		Iteration:    ts.iteration,
		StartedAt:    ts.startedAt,
		Depth:        ts.depth,
		ParentTurnID: ts.parentTurnID,
		ChildTurnIDs: append([]string(nil), ts.childTurnIDs...),
	}
}

func (ts *turnState) setPhase(phase TurnPhase) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.phase = phase
}

func (ts *turnState) setIteration(iteration int) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.iteration = iteration
}

func (ts *turnState) currentIteration() int {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.iteration
}

func (ts *turnState) setFinalContent(content string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.finalContent = content
}

func (ts *turnState) finalContentLen() int {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return len(ts.finalContent)
}

func (ts *turnState) setTurnCancel(cancel context.CancelFunc) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.turnCancel = cancel
}

func (ts *turnState) setProviderCancel(cancel context.CancelFunc) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.providerCancel = cancel
}

func (ts *turnState) clearProviderCancel(_ context.CancelFunc) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.providerCancel = nil
}

func (ts *turnState) requestGracefulInterrupt(hint string) bool {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	if ts.hardAbort {
		return false
	}
	ts.gracefulInterrupt = true
	ts.gracefulInterruptHint = hint
	return true
}

func (ts *turnState) gracefulInterruptRequested() (bool, string) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.gracefulInterrupt && !ts.gracefulTerminalUsed, ts.gracefulInterruptHint
}

func (ts *turnState) markGracefulTerminalUsed() {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.gracefulTerminalUsed = true
}

func (ts *turnState) requestHardAbort() bool {
	ts.mu.Lock()
	if ts.hardAbort {
		ts.mu.Unlock()
		return false
	}
	ts.hardAbort = true
	turnCancel := ts.turnCancel
	providerCancel := ts.providerCancel
	ts.mu.Unlock()

	if providerCancel != nil {
		providerCancel()
	}
	if turnCancel != nil {
		turnCancel()
	}
	return true
}

func (ts *turnState) hardAbortRequested() bool {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.hardAbort
}

func (ts *turnState) eventMeta(source, tracePath string) EventMeta {
	snap := ts.snapshot()
	return EventMeta{
		AgentID:     snap.AgentID,
		TurnID:      snap.TurnID,
		SessionKey:  snap.SessionKey,
		Iteration:   snap.Iteration,
		Source:      source,
		TracePath:   tracePath,
		turnContext: cloneTurnContext(ts.turnCtx),
	}
}

func (ts *turnState) captureRestorePoint(history []providers.Message, summary string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.restorePointHistory = append([]providers.Message(nil), history...)
	ts.restorePointSummary = summary
}

func (ts *turnState) recordPersistedMessage(msg providers.Message) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.persistedMessages = append(ts.persistedMessages, msg)
}

func (ts *turnState) refreshRestorePointFromSession(agent *AgentInstance) {
	history := agent.Sessions.GetHistory(ts.sessionKey)
	summary := agent.Sessions.GetSummary(ts.sessionKey)

	ts.mu.RLock()
	persisted := append([]providers.Message(nil), ts.persistedMessages...)
	ts.mu.RUnlock()

	if matched := matchingTurnMessageTail(history, persisted); matched > 0 {
		history = append([]providers.Message(nil), history[:len(history)-matched]...)
	}

	ts.captureRestorePoint(history, summary)
}

// ingestMessage calls the ContextManager's Ingest method for a persisted message.
// Errors are logged but never block the turn.
func (ts *turnState) ingestMessage(ctx context.Context, al *AgentLoop, msg providers.Message) {
	if al.contextManager == nil {
		return
	}
	if err := al.contextManager.Ingest(ctx, &IngestRequest{
		SessionKey: ts.sessionKey,
		Message:    msg,
	}); err != nil {
		logger.WarnCF("agent", "Context manager ingest failed", map[string]any{
			"session_key": ts.sessionKey,
			"error":       err.Error(),
		})
	}
}

func (ts *turnState) restoreSession(agent *AgentInstance) error {
	ts.mu.RLock()
	history := append([]providers.Message(nil), ts.restorePointHistory...)
	summary := ts.restorePointSummary
	ts.mu.RUnlock()

	agent.Sessions.SetHistory(ts.sessionKey, history)
	agent.Sessions.SetSummary(ts.sessionKey, summary)
	return agent.Sessions.Save(ts.sessionKey)
}

func matchingTurnMessageTail(history, persisted []providers.Message) int {
	maxMatch := min(len(history), len(persisted))
	for size := maxMatch; size > 0; size-- {
		if reflect.DeepEqual(history[len(history)-size:], persisted[len(persisted)-size:]) {
			return size
		}
	}
	return 0
}

func (ts *turnState) interruptHintMessage() providers.Message {
	_, hint := ts.gracefulInterruptRequested()
	content := "Interrupt requested. Stop scheduling tools and provide a short final summary."
	if hint != "" {
		content += "\n\nInterrupt hint: " + hint
	}
	return providers.Message{
		Role:    "user",
		Content: content,
	}
}

// SubTurn-related methods

// Finish marks the turn as finished and closes the pendingResults channel
func (ts *turnState) Finish(isHardAbort bool) {
	ts.isFinished.Store(true)

	// Close pendingResults channel exactly once
	ts.closeOnce.Do(func() {
		if ts.pendingResults != nil {
			close(ts.pendingResults)
		}
		ts.mu.Lock()
		if ts.finishedChan == nil {
			ts.finishedChan = make(chan struct{})
		}
		close(ts.finishedChan)
		ts.mu.Unlock()
	})

	// If this is a graceful finish (not hard abort), signal to children
	if !isHardAbort && ts.parentTurnState == nil {
		// This is a root turn finishing gracefully
		ts.parentEnded.Store(true)
	}

	// Cancel the turn context
	if ts.cancelFunc != nil {
		ts.cancelFunc()
	}

	// Hard abort cascades to all child turns
	if isHardAbort && ts.al != nil {
		ts.mu.RLock()
		children := append([]string(nil), ts.childTurnIDs...)
		ts.mu.RUnlock()
		for _, childID := range children {
			if val, ok := ts.al.activeTurnStates.Load(childID); ok {
				if child, ok := val.(*turnState); ok {
					child.Finish(true)
				}
			}
		}
	}
}

// Finished returns whether the turn has finished
func (ts *turnState) Finished() chan struct{} {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	if ts.finishedChan == nil {
		ts.finishedChan = make(chan struct{})
	}
	return ts.finishedChan
}

// IsParentEnded checks if the parent turn has ended
func (ts *turnState) IsParentEnded() bool {
	if ts.parentTurnState == nil {
		return false
	}
	return ts.parentTurnState.parentEnded.Load()
}

// GetLastFinishReason returns the last LLM finish_reason
func (ts *turnState) GetLastFinishReason() string {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.lastFinishReason
}

// SetLastFinishReason sets the last LLM finish_reason
func (ts *turnState) SetLastFinishReason(reason string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.lastFinishReason = reason
}

// GetLastUsage returns the last LLM usage info
func (ts *turnState) GetLastUsage() *providers.UsageInfo {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.lastUsage
}

// SetLastUsage sets the last LLM usage info
func (ts *turnState) SetLastUsage(usage *providers.UsageInfo) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.lastUsage = usage
}

// Context helper functions for SubTurn

type turnStateKeyType struct{}

var turnStateKey = turnStateKeyType{}

func withTurnState(ctx context.Context, ts *turnState) context.Context {
	return context.WithValue(ctx, turnStateKey, ts)
}

func turnStateFromContext(ctx context.Context) *turnState {
	ts, _ := ctx.Value(turnStateKey).(*turnState)
	return ts
}

// TurnStateFromContext retrieves turnState from context (exported for tools)
func TurnStateFromContext(ctx context.Context) *turnState {
	return turnStateFromContext(ctx)
}
