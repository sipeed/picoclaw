package agent

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/session"
	"github.com/sipeed/picoclaw/pkg/tools"
)

// ====================== Config & Constants ======================
const maxSubTurnDepth = 3

var (
	ErrDepthLimitExceeded   = errors.New("sub-turn depth limit exceeded")
	ErrInvalidSubTurnConfig = errors.New("invalid sub-turn config")
)

// ====================== SubTurn Config ======================
type SubTurnConfig struct {
	Model        string
	Tools        []tools.Tool
	SystemPrompt string
	MaxTokens    int
	// Can be extended with temperature, topP, etc.
}

// ====================== Sub-turn Events (Aligned with EventBus) ======================
type SubTurnSpawnEvent struct {
	ParentID string
	ChildID  string
	Config   SubTurnConfig
}

type SubTurnEndEvent struct {
	ChildID string
	Result  *tools.ToolResult
	Err     error
}

type SubTurnResultDeliveredEvent struct {
	ParentID string
	ChildID  string
	Result   *tools.ToolResult
}

type SubTurnOrphanResultEvent struct {
	ParentID string
	ChildID  string
	Result   *tools.ToolResult
}

// ====================== turnState (Simplified, reusable with existing structs) ======================
type turnState struct {
	ctx            context.Context
	cancelFunc     context.CancelFunc // Used to cancel all children when this turn finishes
	turnID         string
	parentTurnID   string
	depth          int
	childTurnIDs   []string
	pendingResults chan *tools.ToolResult
	session        session.SessionStore
	mu             sync.Mutex
	isFinished     bool // Marks if the parent Turn has ended
}

// ====================== Helper Functions ======================
var globalTurnCounter int64

func generateTurnID() string {
	return fmt.Sprintf("subturn-%d", atomic.AddInt64(&globalTurnCounter, 1))
}

func newTurnState(ctx context.Context, id string, parent *turnState) *turnState {
	turnCtx, cancel := context.WithCancel(ctx)
	return &turnState{
		ctx:          turnCtx,
		cancelFunc:   cancel,
		turnID:       id,
		parentTurnID: parent.turnID,
		depth:        parent.depth + 1,
		session:      newEphemeralSession(parent.session),
		// NOTE: In this PoC, I use a fixed-size channel (16).
		// Under high concurrency or long-running sub-turns, this might fill up and cause
		// intermediate results to be discarded in deliverSubTurnResult.
		// For production, consider an unbounded queue or a blocking strategy with backpressure.
		pendingResults: make(chan *tools.ToolResult, 16),
	}
}

// Finish marks the turn as finished and cancels its context, aborting any running sub-turns.
func (ts *turnState) Finish() {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.isFinished = true
	if ts.cancelFunc != nil {
		ts.cancelFunc()
	}
}

// ephemeralSessionStore is a pure in-memory SessionStore for SubTurns.
// It never writes to disk, keeping sub-turn history isolated from the parent session.
type ephemeralSessionStore struct {
	mu      sync.Mutex
	history []providers.Message
	summary string
}

func (e *ephemeralSessionStore) AddMessage(sessionKey, role, content string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.history = append(e.history, providers.Message{Role: role, Content: content})
}

func (e *ephemeralSessionStore) AddFullMessage(sessionKey string, msg providers.Message) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.history = append(e.history, msg)
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

func newEphemeralSession(_ session.SessionStore) session.SessionStore {
	return &ephemeralSessionStore{}
}

// ====================== Core Function: spawnSubTurn ======================
func spawnSubTurn(ctx context.Context, al *AgentLoop, parentTS *turnState, cfg SubTurnConfig) (result *tools.ToolResult, err error) {
	// 1. Depth limit check
	if parentTS.depth >= maxSubTurnDepth {
		return nil, ErrDepthLimitExceeded
	}

	// 2. Config validation
	if cfg.Model == "" {
		return nil, ErrInvalidSubTurnConfig
	}

	// Create a sub-context for the child turn to support cancellation
	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// 3. Create child Turn state
	childID := generateTurnID()
	childTS := newTurnState(childCtx, childID, parentTS)

	// 4. Establish parent-child relationship (thread-safe)
	parentTS.mu.Lock()
	parentTS.childTurnIDs = append(parentTS.childTurnIDs, childID)
	parentTS.mu.Unlock()

	// 5. Emit Spawn event (currently using Mock, will be replaced by real EventBus)
	MockEventBus.Emit(SubTurnSpawnEvent{
		ParentID: parentTS.turnID,
		ChildID:  childID,
		Config:   cfg,
	})

	// 6. Defer emitting End event, and recover from panics to ensure it's always fired
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("subturn panicked: %v", r)
		}

		MockEventBus.Emit(SubTurnEndEvent{
			ChildID: childID,
			Result:  result,
			Err:     err,
		})
	}()

	// 7. Execute sub-turn via the real agent loop.
	// Build a child AgentInstance from SubTurnConfig, inheriting defaults from the parent agent.
	result, err = runTurn(childCtx, al, childTS, cfg)

	// 8. Deliver result back to parent Turn
	deliverSubTurnResult(parentTS, childID, result)

	return result, err
}

// ====================== Result Delivery ======================
func deliverSubTurnResult(parentTS *turnState, childID string, result *tools.ToolResult) {
	parentTS.mu.Lock()
	defer parentTS.mu.Unlock()

	// Emit ResultDelivered event
	MockEventBus.Emit(SubTurnResultDeliveredEvent{
		ParentID: parentTS.turnID,
		ChildID:  childID,
		Result:   result,
	})

	if !parentTS.isFinished {
		// Parent Turn is still running → Place in pending queue (handled automatically by parent loop in next round)
		select {
		case parentTS.pendingResults <- result:
		default:
			fmt.Println("[SubTurn] warning: pendingResults channel full")
		}
		return
	}

	// Parent Turn has ended
	// emit an OrphanResultEvent so the system/UI can handle this late arrival.
	if result != nil {
		MockEventBus.Emit(SubTurnOrphanResultEvent{
			ParentID: parentTS.turnID,
			ChildID:  childID,
			Result:   result,
		})
	}
}

// runTurn builds a temporary AgentInstance from SubTurnConfig and delegates to
// the real agent loop. The child's ephemeral session is used for history so it
// never pollutes the parent session.
func runTurn(ctx context.Context, al *AgentLoop, ts *turnState, cfg SubTurnConfig) (*tools.ToolResult, error) {
	// Derive candidates from the requested model using the parent loop's provider.
	defaultProvider := al.GetConfig().Agents.Defaults.Provider
	candidates := providers.ResolveCandidates(
		providers.ModelConfig{Primary: cfg.Model},
		defaultProvider,
	)

	// Build a minimal AgentInstance for this sub-turn.
	// It reuses the parent loop's provider and config, but gets its own
	// ephemeral session store and tool registry.
	toolRegistry := tools.NewToolRegistry()
	for _, t := range cfg.Tools {
		toolRegistry.Register(t)
	}

	parentAgent := al.GetRegistry().GetDefaultAgent()
	childAgent := &AgentInstance{
		ID:                        ts.turnID,
		Model:                     cfg.Model,
		MaxIterations:             parentAgent.MaxIterations,
		MaxTokens:                 cfg.MaxTokens,
		Temperature:               parentAgent.Temperature,
		ThinkingLevel:             parentAgent.ThinkingLevel,
		ContextWindow:             cfg.MaxTokens,
		SummarizeMessageThreshold: parentAgent.SummarizeMessageThreshold,
		SummarizeTokenPercent:     parentAgent.SummarizeTokenPercent,
		Provider:                  parentAgent.Provider,
		Sessions:                  ts.session,
		ContextBuilder:            parentAgent.ContextBuilder,
		Tools:                     toolRegistry,
		Candidates:                candidates,
	}
	if childAgent.MaxTokens == 0 {
		childAgent.MaxTokens = parentAgent.MaxTokens
		childAgent.ContextWindow = parentAgent.ContextWindow
	}

	finalContent, err := al.runAgentLoop(ctx, childAgent, processOptions{
		SessionKey:      ts.turnID,
		UserMessage:     cfg.SystemPrompt,
		DefaultResponse: "",
		EnableSummary:   false,
		SendResponse:    false,
	})
	if err != nil {
		return nil, err
	}
	return &tools.ToolResult{ForLLM: finalContent}, nil
}

// ====================== Other Types ======================
