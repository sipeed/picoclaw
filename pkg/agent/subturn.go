package agent

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

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
	Result  *ToolResult
	Err     error
}

type SubTurnResultDeliveredEvent struct {
	ParentID string
	ChildID  string
	Result   *ToolResult
}

type SubTurnOrphanResultEvent struct {
	ParentID string
	ChildID  string
	Result   *ToolResult
}

// ====================== turnState (Simplified, reusable with existing structs) ======================
type turnState struct {
	ctx            context.Context
	cancelFunc     context.CancelFunc // Used to cancel all children when this turn finishes
	turnID         string
	parentTurnID   string
	depth          int
	childTurnIDs   []string
	pendingResults chan *ToolResult
	session        *Session
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
		pendingResults: make(chan *ToolResult, 16),
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

// newEphemeralSession - Pure in-memory temporary Session (avoids polluting the main session)
func newEphemeralSession(parent *Session) *Session {
	// In a real project, it's recommended to copy only necessary fields; simplified here.
	return &Session{
		History: make([]Message, 0, len(parent.History)),
	}
}

// ====================== Core Function: spawnSubTurn ======================
func spawnSubTurn(ctx context.Context, parentTS *turnState, cfg SubTurnConfig) (result *ToolResult, err error) {
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

	// 7. Execute full runTurn (follows the main execution path, all hooks, steering, and interrupts are effective!)
	// Pass the childCtx so the sub-turn can be interrupted if the parent is cancelled.
	result, err = runTurn(childCtx, childTS, childTS.session, cfg)

	// 8. Deliver result back to parent Turn
	deliverSubTurnResult(parentTS, childID, result)

	return result, err
}

// ====================== Result Delivery ======================
func deliverSubTurnResult(parentTS *turnState, childID string, result *ToolResult) {
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

// ====================== Placeholder Function (Actually reuses runTurn in loop.go) ======================
func runTurn(ctx context.Context, ts *turnState, session *Session, cfg SubTurnConfig) (*ToolResult, error) {
	// TODO: Directly call the existing runTurn implementation in your project here
	// Ensure the existing runTurn respects the context for cancellation.
	return &ToolResult{Content: "Sub-turn executed successfully"}, nil
}

// ====================== Other Types (Reused or simplified from existing code) ======================
type ToolResult struct {
	Content string
}

func (r *ToolResult) ToMessage() Message {
	return Message{Content: r.Content}
}

type Session struct {
	History []Message
}

type Message struct {
	Content string
}
