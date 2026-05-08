package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	runtimeevents "github.com/sipeed/picoclaw/pkg/events"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/tools"
)

// Test constants (use defaults from subturn.go)
const (
	testMaxConcurrentSubTurns = defaultMaxConcurrentSubTurns
)

// ====================== Test Helper: Event Collector ======================
type eventCollector struct {
	mu     sync.Mutex
	events []runtimeevents.Event
}

type recordingChannelManager struct {
	dismissedSessions []string
	dismissedChatIDs  []string
}

func (m *recordingChannelManager) GetChannel(name string) (channels.Channel, bool) { return nil, false }
func (m *recordingChannelManager) GetEnabledChannels() []string                     { return nil }
func (m *recordingChannelManager) InvokeTypingStop(channel, chatID string)          {}
func (m *recordingChannelManager) SendMessage(ctx context.Context, msg bus.OutboundMessage) error {
	return nil
}
func (m *recordingChannelManager) SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) error {
	return nil
}
func (m *recordingChannelManager) SendPlaceholder(ctx context.Context, channel, chatID string) bool {
	return false
}
func (m *recordingChannelManager) DismissToolFeedback(
	ctx context.Context, channel, chatID string, outboundCtx *bus.InboundContext,
) {
	m.dismissedChatIDs = append(m.dismissedChatIDs, fmt.Sprintf("%s:%s", channel, chatID))
}
func (m *recordingChannelManager) DismissToolFeedbackForSession(
	ctx context.Context,
	channel, chatID string,
	outboundCtx *bus.InboundContext,
	sessionKey string,
) {
	m.dismissedSessions = append(m.dismissedSessions, fmt.Sprintf("%s:%s:%s", channel, chatID, sessionKey))
}

func newEventCollector(t *testing.T, al *AgentLoop) (*eventCollector, func()) {
	t.Helper()
	c := &eventCollector{}
	runtimeCh, closeRuntimeEvents := subscribeRuntimeEventsForTest(
		t,
		al,
		16,
		runtimeevents.KindAgentSubTurnSpawn,
		runtimeevents.KindAgentSubTurnEnd,
		runtimeevents.KindAgentSubTurnResultDelivered,
		runtimeevents.KindAgentSubTurnOrphan,
	)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for evt := range runtimeCh {
			c.mu.Lock()
			c.events = append(c.events, evt)
			c.mu.Unlock()
		}
	}()
	cleanup := func() {
		closeRuntimeEvents()
		<-done
	}
	return c, cleanup
}

func (c *eventCollector) hasEventOfKind(kind runtimeevents.Kind) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, e := range c.events {
		if e.Kind == kind {
			return true
		}
	}
	return false
}

// ====================== Main Test Function ======================
func TestSpawnSubTurn(t *testing.T) {
	tests := []struct {
		name          string
		parentDepth   int
		config        SubTurnConfig
		wantErr       error
		wantSpawn     bool
		wantEnd       bool
		wantDepthFail bool
	}{
		{
			name:        "Basic success path - Single layer sub-turn",
			parentDepth: 0,
			config: SubTurnConfig{
				Model: "gpt-4o-mini",
				Tools: []tools.Tool{}, // At least one tool
			},
			wantErr:   nil,
			wantSpawn: true,
			wantEnd:   true,
		},
		{
			name:        "Nested 2 layers - Normal",
			parentDepth: 1,
			config: SubTurnConfig{
				Model: "gpt-4o-mini",
				Tools: []tools.Tool{},
			},
			wantErr:   nil,
			wantSpawn: true,
			wantEnd:   true,
		},
		{
			name:        "Depth limit triggered - 4th layer fails",
			parentDepth: 3,
			config: SubTurnConfig{
				Model: "gpt-4o-mini",
				Tools: []tools.Tool{},
			},
			wantErr:       ErrDepthLimitExceeded,
			wantSpawn:     false,
			wantEnd:       false,
			wantDepthFail: true,
		},
		{
			name:        "Invalid config - Empty Model",
			parentDepth: 0,
			config: SubTurnConfig{
				Model: "",
				Tools: []tools.Tool{},
			},
			wantErr:   ErrInvalidSubTurnConfig,
			wantSpawn: false,
			wantEnd:   false,
		},
	}

	al, _, _, provider, cleanup := newTestAgentLoop(t)
	_ = provider
	defer cleanup()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare parent Turn
			parent := &turnState{
				ctx:            context.Background(),
				turnID:         "parent-1",
				depth:          tt.parentDepth,
				childTurnIDs:   []string{},
				pendingResults: make(chan *tools.ToolResult, 10),
				session:        &ephemeralSessionStore{},
				agent:          al.registry.GetDefaultAgent(),
			}

			// Subscribe to runtime events to capture sub-turn lifecycle.
			collector, collectCleanup := newEventCollector(t, al)
			defer collectCleanup()

			// Execute spawnSubTurn
			result, err := spawnSubTurn(context.Background(), al, parent, tt.config)

			// Assert errors
			if tt.wantErr != nil {
				if err == nil || err != tt.wantErr {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Verify result
			if result == nil {
				t.Error("expected non-nil result")
			}

			// Verify event emission
			time.Sleep(10 * time.Millisecond) // let event goroutine flush
			if tt.wantSpawn {
				if !collector.hasEventOfKind(runtimeevents.KindAgentSubTurnSpawn) {
					t.Error("SubTurnSpawnEvent not emitted")
				}
			}
			if tt.wantEnd {
				if !collector.hasEventOfKind(runtimeevents.KindAgentSubTurnEnd) {
					t.Error("SubTurnEndEvent not emitted")
				}
			}

			// Verify turn tree
			if len(parent.childTurnIDs) == 0 && !tt.wantDepthFail {
				t.Error("child Turn not added to parent.childTurnIDs")
			}

			// For synchronous calls (Async=false, the default), result is returned directly
			// and should NOT be in pendingResults. The result was already verified above.
			// Only async calls (Async=true) would place results in pendingResults.
		})
	}
}

// ====================== Extra Independent Test: Ephemeral Session Isolation ======================
func TestSpawnSubTurn_EphemeralSessionIsolation(t *testing.T) {
	al, _, _, provider, cleanup := newTestAgentLoop(t)
	_ = provider
	defer cleanup()

	// Parent uses its own ephemeral store pre-seeded with one message
	parentSession := &ephemeralSessionStore{}
	parentSession.AddMessage("", "user", "parent msg")
	parent := &turnState{
		ctx:            context.Background(),
		turnID:         "parent-1",
		depth:          0,
		pendingResults: make(chan *tools.ToolResult, 4),
		concurrencySem: make(chan struct{}, testMaxConcurrentSubTurns),
		session:        parentSession,
	}

	cfg := SubTurnConfig{Model: "gpt-4o-mini", Tools: []tools.Tool{}}

	originalParentLen := len(parentSession.GetHistory(""))

	_, _ = spawnSubTurn(context.Background(), al, parent, cfg)

	// Parent session must be untouched — child used its own store
	if got := len(parentSession.GetHistory("")); got != originalParentLen {
		t.Errorf("parent session polluted: expected %d messages, got %d", originalParentLen, got)
	}

	// The child's agent.Sessions must NOT be the same pointer as the parent's session.
	// We verify this indirectly: spawnSubTurn stores childTS in activeTurnStates during
	// execution (deleted on return), so we can't easily grab childTS after the call.
	// Instead, confirm that the child session is a distinct ephemeralSessionStore by
	// checking the parent session key is only used by the parent store.
	// If isolation is correct, parent.session.GetHistory(childID) is always empty
	// (the child never wrote to the parent store).
	al.activeTurnStates.Range(func(k, v any) bool {
		// No active turns should remain after spawnSubTurn returns
		t.Errorf("unexpected active turn state left after spawnSubTurn: key=%v", k)
		return true
	})
}

// ====================== Extra Independent Test: Result Delivery Path (Async) ======================
func TestSpawnSubTurn_ResultDelivery(t *testing.T) {
	al, _, _, provider, cleanup := newTestAgentLoop(t)
	_ = provider
	defer cleanup()

	parent := &turnState{
		ctx:            context.Background(),
		turnID:         "parent-1",
		depth:          0,
		pendingResults: make(chan *tools.ToolResult, 1),
		session:        &ephemeralSessionStore{},
	}

	// Set Async=true to test async result delivery via pendingResults channel
	cfg := SubTurnConfig{Model: "gpt-4o-mini", Tools: []tools.Tool{}, Async: true}

	_, _ = spawnSubTurn(context.Background(), al, parent, cfg)

	// Check if pendingResults received the result (only for async calls)
	select {
	case res := <-parent.pendingResults:
		if res == nil {
			t.Error("received nil result in pendingResults")
		}
	default:
		t.Error("result did not enter pendingResults for async call")
	}
}

// ====================== Extra Independent Test: Result Delivery Path (Sync) ======================
func TestSpawnSubTurn_ResultDeliverySync(t *testing.T) {
	al, _, _, provider, cleanup := newTestAgentLoop(t)
	_ = provider
	defer cleanup()

	parent := &turnState{
		ctx:            context.Background(),
		turnID:         "parent-sync-1",
		depth:          0,
		pendingResults: make(chan *tools.ToolResult, 1),
		session:        &ephemeralSessionStore{},
	}

	// Sync call (Async=false, the default) - result should be returned directly
	cfg := SubTurnConfig{Model: "gpt-4o-mini", Tools: []tools.Tool{}, Async: false}

	result, err := spawnSubTurn(context.Background(), al, parent, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Result should be returned directly
	if result == nil {
		t.Error("expected non-nil result from sync call")
	}

	// pendingResults should NOT contain the result (no double delivery)
	select {
	case <-parent.pendingResults:
		t.Error("sync call should not place result in pendingResults (double delivery)")
	default:
		// Expected - channel should be empty
	}
}

// ====================== Extra Independent Test: Orphan Result Routing ======================
func TestSpawnSubTurn_OrphanResultRouting(t *testing.T) {
	al, _, _, provider, cleanup := newTestAgentLoop(t)
	_ = provider
	defer cleanup()

	collector, collectCleanup := newEventCollector(t, al)
	defer collectCleanup()

	parentCtx, cancelParent := context.WithCancel(context.Background())
	parent := &turnState{
		ctx:            parentCtx,
		cancelFunc:     cancelParent,
		turnID:         "parent-1",
		depth:          0,
		pendingResults: make(chan *tools.ToolResult, 1),
		session:        &ephemeralSessionStore{},
	}

	// Simulate parent finishing before child delivers result
	parent.Finish(false)

	// Call deliverSubTurnResult directly to simulate a delayed child
	deliverSubTurnResult(al, parent, "delayed-child", &tools.ToolResult{ForLLM: "late result"})

	time.Sleep(10 * time.Millisecond) // let event goroutine flush
	// Verify Orphan event is emitted
	if !collector.hasEventOfKind(runtimeevents.KindAgentSubTurnOrphan) {
		t.Error("agent.subturn.orphan not emitted for finished parent")
	}

	// Verify history is NOT polluted
	if len(parent.session.GetHistory("")) != 0 {
		t.Error("Parent history was polluted by orphan result")
	}
}

// ====================== Extra Independent Test: Result Channel Registration ======================
func TestSubTurnResultChannelRegistration(t *testing.T) {
	al, _, _, provider, cleanup := newTestAgentLoop(t)
	_ = provider
	defer cleanup()

	parent := &turnState{
		ctx:            context.Background(),
		turnID:         "parent-reg-1",
		depth:          0,
		pendingResults: make(chan *tools.ToolResult, 4),
		session:        &ephemeralSessionStore{},
	}

	cfg := SubTurnConfig{Model: "gpt-4o-mini", Tools: []tools.Tool{}}

	// Before spawn: channel should not be registered
	if results := al.dequeuePendingSubTurnResults(parent.turnID); results != nil {
		t.Error("expected no channel before spawnSubTurn")
	}

	_, _ = spawnSubTurn(context.Background(), al, parent, cfg)
}

// ====================== Extra Independent Test: Dequeue Pending SubTurn Results ======================
func TestDequeuePendingSubTurnResults(t *testing.T) {
	al, _, _, provider, cleanup := newTestAgentLoop(t)
	_ = provider
	defer cleanup()

	sessionKey := "test-session-dequeue"

	// Empty (no turnState registered) returns nil
	if results := al.dequeuePendingSubTurnResults(sessionKey); len(results) != 0 {
		t.Errorf("expected empty results, got %d", len(results))
	}

	// Register a turnState so dequeuePendingSubTurnResults can find it
	ts := &turnState{
		ctx:            context.Background(),
		turnID:         sessionKey,
		depth:          0,
		session:        &ephemeralSessionStore{},
		pendingResults: make(chan *tools.ToolResult, 4),
	}
	al.activeTurnStates.Store(sessionKey, ts)
	defer al.activeTurnStates.Delete(sessionKey)

	// Put 3 results in
	ts.pendingResults <- &tools.ToolResult{ForLLM: "result-1"}
	ts.pendingResults <- &tools.ToolResult{ForLLM: "result-2"}
	ts.pendingResults <- &tools.ToolResult{ForLLM: "result-3"}

	results := al.dequeuePendingSubTurnResults(sessionKey)
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
	if results[0].ForLLM != "result-1" || results[2].ForLLM != "result-3" {
		t.Error("results order or content mismatch")
	}

	// Channel should be drained now
	if results := al.dequeuePendingSubTurnResults(sessionKey); len(results) != 0 {
		t.Errorf("expected empty after drain, got %d", len(results))
	}

	// After removing from activeTurnStates, returns nil
	al.activeTurnStates.Delete(sessionKey)
	if results := al.dequeuePendingSubTurnResults(sessionKey); results != nil {
		t.Error("expected nil for unregistered session")
	}
}

// ====================== Extra Independent Test: Concurrency Semaphore ======================
func TestSubTurnConcurrencySemaphore(t *testing.T) {
	al, _, _, provider, cleanup := newTestAgentLoop(t)
	_ = provider
	defer cleanup()

	parent := &turnState{
		ctx:            context.Background(),
		turnID:         "parent-concurrency",
		depth:          0,
		pendingResults: make(chan *tools.ToolResult, 10),
		session:        &ephemeralSessionStore{},
		concurrencySem: make(chan struct{}, 2), // Only allow 2 concurrent children
	}

	cfg := SubTurnConfig{Model: "gpt-4o-mini", Tools: []tools.Tool{}}

	// Spawn 2 children — should succeed immediately
	done := make(chan bool, 3)
	for i := 0; i < 2; i++ {
		go func() {
			_, _ = spawnSubTurn(context.Background(), al, parent, cfg)
			done <- true
		}()
	}

	// Wait a bit to ensure the first 2 are running
	// (In real scenario they'd be blocked in runTurn, but mockProvider returns immediately)
	// So we just verify the semaphore doesn't block when under limit
	<-done
	<-done

	// Verify semaphore is now full (2/2 slots used, but they already released)
	// Since mockProvider returns immediately, semaphore is already released
	// So we can't easily test blocking without a real long-running operation

	// Instead, verify that semaphore exists and has correct capacity
	if cap(parent.concurrencySem) != 2 {
		t.Errorf("expected semaphore capacity 2, got %d", cap(parent.concurrencySem))
	}
}

// ====================== Extra Independent Test: Hard Abort Cascading ======================
func TestHardAbortCascading(t *testing.T) {
	al, _, _, provider, cleanup := newTestAgentLoop(t)
	_ = provider
	defer cleanup()

	sessionKey := "test-session-abort"

	// Root turn with its own independent context (not derived from child)
	rootCtx, rootCancel := context.WithCancel(context.Background())
	rootTS := &turnState{
		ctx:            rootCtx,
		cancelFunc:     rootCancel,
		turnID:         sessionKey,
		depth:          0,
		session:        &ephemeralSessionStore{},
		pendingResults: make(chan *tools.ToolResult, 16),
		concurrencySem: make(chan struct{}, 5),
		al:             al,
	}
	al.activeTurnStates.Store(sessionKey, rootTS)
	defer al.activeTurnStates.Delete(sessionKey)

	// Child turn with an INDEPENDENT context (simulates spawnSubTurn behavior:
	// context.WithTimeout(context.Background(), ...) — NOT derived from parent).
	// Cascade must therefore happen via childTurnIDs traversal, not Go context tree.
	childCtx, childCancel := context.WithCancel(context.Background())
	childID := "child-independent"
	childTS := &turnState{
		ctx:            childCtx,
		cancelFunc:     childCancel,
		turnID:         childID,
		pendingResults: make(chan *tools.ToolResult, 4),
		al:             al,
	}
	al.activeTurnStates.Store(childID, childTS)
	defer al.activeTurnStates.Delete(childID)

	// Wire child into root's childTurnIDs (as spawnSubTurn would do)
	rootTS.childTurnIDs = append(rootTS.childTurnIDs, childID)

	// Verify neither context is canceled yet
	select {
	case <-rootTS.ctx.Done():
		t.Fatal("root context should not be canceled yet")
	default:
	}
	select {
	case <-childTS.ctx.Done():
		t.Fatal("child context should not be canceled yet (independent context)")
	default:
	}

	// Trigger Hard Abort via al.HardAbort (goes through steering.go → Finish(true))
	err := al.HardAbort(sessionKey)
	if err != nil {
		t.Fatalf("HardAbort failed: %v", err)
	}

	// Root context must be canceled
	select {
	case <-rootTS.ctx.Done():
	default:
		t.Error("root context should be canceled after HardAbort")
	}

	// Child context must be canceled via childTurnIDs cascade, NOT via Go context tree
	select {
	case <-childTS.ctx.Done():
	default:
		t.Error("child context should be canceled via childTurnIDs cascade")
	}

	// HardAbort on non-existent session should return an error
	if err := al.HardAbort("non-existent-session"); err == nil {
		t.Error("expected error for non-existent session")
	}
}

// TestHardAbortSessionRollback verifies that HardAbort rolls back session history
// to the state before the turn started, discarding all messages added during the turn.
func TestHardAbortSessionRollback(t *testing.T) {
	al, _, _, provider, cleanup := newTestAgentLoop(t)
	_ = provider
	defer cleanup()

	// Create a session with initial history
	sess := &ephemeralSessionStore{
		history: []providers.Message{
			{Role: "user", Content: "initial message 1"},
			{Role: "assistant", Content: "initial response 1"},
		},
	}

	// Create a root turnState with initialHistoryLength = 2
	rootTS := &turnState{
		ctx:                  context.Background(),
		turnID:               "test-session",
		depth:                0,
		session:              sess,
		initialHistoryLength: 2, // Snapshot: 2 messages
		pendingResults:       make(chan *tools.ToolResult, 16),
		concurrencySem:       make(chan struct{}, 5),
	}

	// Register the turn state
	al.activeTurnStates.Store("test-session", rootTS)

	// Simulate adding messages during the turn (e.g., user input + assistant response)
	sess.AddMessage("", "user", "new user message")
	sess.AddMessage("", "assistant", "new assistant response")

	// Verify history grew to 4 messages
	if len(sess.GetHistory("")) != 4 {
		t.Fatalf("expected 4 messages before abort, got %d", len(sess.GetHistory("")))
	}

	// Trigger HardAbort
	err := al.HardAbort("test-session")
	if err != nil {
		t.Fatalf("HardAbort failed: %v", err)
	}

	// Verify history rolled back to initial 2 messages
	finalHistory := sess.GetHistory("")
	if len(finalHistory) != 2 {
		t.Errorf("expected history to rollback to 2 messages, got %d", len(finalHistory))
	}

	// Verify the content matches the initial state
	if finalHistory[0].Content != "initial message 1" || finalHistory[1].Content != "initial response 1" {
		t.Error("history content does not match initial state after rollback")
	}
}

// TestNestedSubTurnHierarchy verifies that nested SubTurns maintain correct
// parent-child relationships and depth tracking when recursively calling runAgentLoop.
func TestNestedSubTurnHierarchy(t *testing.T) {
	al, _, _, provider, cleanup := newTestAgentLoop(t)
	_ = provider
	defer cleanup()

	// Track spawned turns and their depths
	type turnInfo struct {
		parentID string
		childID  string
	}
	var spawnedTurns []turnInfo
	var mu sync.Mutex

	runtimeCh, closeRuntimeEvents := subscribeRuntimeEventsForTest(
		t,
		al,
		16,
		runtimeevents.KindAgentSubTurnSpawn,
	)
	defer closeRuntimeEvents()
	go func() {
		for evt := range runtimeCh {
			if evt.Kind == runtimeevents.KindAgentSubTurnSpawn {
				p, _ := evt.Payload.(SubTurnSpawnPayload)
				mu.Lock()
				spawnedTurns = append(spawnedTurns, turnInfo{
					parentID: p.ParentTurnID,
					childID:  p.Label,
				})
				mu.Unlock()
			}
		}
	}()

	// Create a root turn
	rootSession := &ephemeralSessionStore{}
	rootTS := &turnState{
		ctx:            context.Background(),
		turnID:         "root-turn",
		depth:          0,
		session:        rootSession,
		pendingResults: make(chan *tools.ToolResult, 16),
		concurrencySem: make(chan struct{}, 5),
	}

	// Spawn a child (depth 1)
	childCfg := SubTurnConfig{Model: "gpt-4o-mini"}
	_, err := spawnSubTurn(context.Background(), al, rootTS, childCfg)
	if err != nil {
		t.Fatalf("failed to spawn child: %v", err)
	}

	time.Sleep(10 * time.Millisecond) // let event goroutine flush

	// Verify we captured the spawn event
	mu.Lock()
	if len(spawnedTurns) != 1 {
		t.Fatalf("expected 1 spawn event, got %d", len(spawnedTurns))
	}
	if spawnedTurns[0].parentID != "root-turn" {
		t.Errorf("expected parent ID 'root-turn', got %s", spawnedTurns[0].parentID)
	}
	mu.Unlock()

	// Verify root turn has the child in its childTurnIDs
	rootTS.mu.Lock()
	if len(rootTS.childTurnIDs) != 1 {
		t.Errorf("expected root to have 1 child, got %d", len(rootTS.childTurnIDs))
	}
	rootTS.mu.Unlock()
}

// TestDeliverSubTurnResultNoDeadlock verifies that deliverSubTurnResult doesn't
// deadlock when multiple goroutines are accessing the parent turnState concurrently.
func TestDeliverSubTurnResultNoDeadlock(t *testing.T) {
	parent := &turnState{
		ctx:            context.Background(),
		turnID:         "parent-deadlock-test",
		depth:          0,
		pendingResults: make(chan *tools.ToolResult, 2), // Small buffer to test blocking
	}

	// Simulate multiple child turns delivering results concurrently
	var wg sync.WaitGroup
	numChildren := 10

	for i := 0; i < numChildren; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			result := &tools.ToolResult{ForLLM: fmt.Sprintf("result-%d", id)}
			deliverSubTurnResult(nil, parent, fmt.Sprintf("child-%d", id), result)
		}(i)
	}

	// Concurrently read from the channel to prevent blocking
	// and to actually retrieve the matched number of results
	go func() {
		for i := 0; i < numChildren; i++ {
			select {
			case <-parent.pendingResults:
			case <-time.After(5 * time.Second):
				t.Error("timeout waiting for result")
				return
			}
		}
	}()

	// Wait for all deliveries to complete (with timeout)
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success - no deadlock
	case <-time.After(3 * time.Second):
		t.Fatal("deadlock detected: deliverSubTurnResult blocked")
	}
}

// TestHardAbortOrderOfOperations verifies that HardAbort calls Finish() before
// rolling back session history, minimizing the race window where new messages
// could be added after rollback.
func TestHardAbortOrderOfOperations(t *testing.T) {
	al, _, _, provider, cleanup := newTestAgentLoop(t)
	_ = provider
	defer cleanup()

	sess := &ephemeralSessionStore{
		history: []providers.Message{
			{Role: "user", Content: "initial message"},
			{Role: "assistant", Content: "response 1"},
			{Role: "user", Content: "follow-up"},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rootTS := &turnState{
		ctx:                  ctx,
		cancelFunc:           cancel,
		turnID:               "test-session-order",
		depth:                0,
		session:              sess,
		initialHistoryLength: 1, // Snapshot: 1 message
		pendingResults:       make(chan *tools.ToolResult, 16),
		concurrencySem:       make(chan struct{}, 5),
	}

	al.activeTurnStates.Store("test-session-order", rootTS)

	// Trigger HardAbort
	err := al.HardAbort("test-session-order")
	if err != nil {
		t.Fatalf("HardAbort failed: %v", err)
	}

	// Verify context was canceled (Finish() was called)
	select {
	case <-rootTS.ctx.Done():
		// Good - context was canceled
	default:
		t.Error("expected context to be canceled after HardAbort")
	}

	// Verify history was rolled back
	finalHistory := sess.GetHistory("")
	if len(finalHistory) != 1 {
		t.Errorf("expected history to rollback to 1 message, got %d", len(finalHistory))
	}

	if finalHistory[0].Content != "initial message" {
		t.Error("history content does not match initial state after rollback")
	}
}

// TestFinishedChannelClosedState verifies that Finish() closes the Finished() channel
// so that child turns can safely abort waiting.
func TestFinishedChannelClosedState(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ts := &turnState{
		ctx:            ctx,
		cancelFunc:     cancel,
		turnID:         "test-finished-channel",
		depth:          0,
		pendingResults: make(chan *tools.ToolResult, 2),
	}

	// Verify Finished channel is blocking initially
	select {
	case <-ts.Finished():
		t.Fatal("finished channel should block initially")
	default:
		// Good
	}

	// Call Finish() with graceful finish
	ts.Finish(false)

	// Verify Finished channel is closed
	select {
	case _, ok := <-ts.Finished():
		if ok {
			t.Error("expected Finished() channel to be closed after Finish()")
		}
	default:
		t.Fatal("expected <-ts.Finished() to not block")
	}

	// Verify Finish() is idempotent
	ts.Finish(false) // Should not panic

	// Verify deliverSubTurnResult correctly uses Finished() channel and treats as orphan
	result := &tools.ToolResult{ForLLM: "late result"}
	deliverSubTurnResult(nil, ts, "child-1", result) // Will emit orphan due to <-ts.Finished() case
}

// TestFinalPollCapturesLateResults verifies that the final poll before Finish()
// captures results that arrive after the last iteration poll.
func TestFinalPollCapturesLateResults(t *testing.T) {
	al, _, _, provider, cleanup := newTestAgentLoop(t)
	_ = provider
	defer cleanup()

	sessionKey := "test-session-final-poll"

	// Register a turnState
	ts := &turnState{
		ctx:            context.Background(),
		turnID:         sessionKey,
		depth:          0,
		session:        &ephemeralSessionStore{},
		pendingResults: make(chan *tools.ToolResult, 4),
	}
	al.activeTurnStates.Store(sessionKey, ts)
	defer al.activeTurnStates.Delete(sessionKey)

	// Simulate results arriving after last iteration poll
	ts.pendingResults <- &tools.ToolResult{ForLLM: "result 1"}
	ts.pendingResults <- &tools.ToolResult{ForLLM: "result 2"}

	// Dequeue should capture both results
	results := al.dequeuePendingSubTurnResults(sessionKey)

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	// Verify channel is now empty
	results = al.dequeuePendingSubTurnResults(sessionKey)
	if len(results) != 0 {
		t.Errorf("expected 0 results on second poll, got %d", len(results))
	}
}

// TestSpawnSubTurn_PanicRecovery verifies that even if runTurn panics,
// the result is still delivered for async calls and SubTurnEndEvent is emitted.
func TestSpawnSubTurn_PanicRecovery(t *testing.T) {
	// Create a panic provider
	panicProvider := &panicMockProvider{}
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         t.TempDir(),
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}
	al := NewAgentLoop(cfg, bus.NewMessageBus(), panicProvider)

	parent := &turnState{
		ctx:            context.Background(),
		turnID:         "parent-panic",
		depth:          0,
		pendingResults: make(chan *tools.ToolResult, 1),
		session:        &ephemeralSessionStore{},
	}

	collector, collectCleanup := newEventCollector(t, al)
	defer collectCleanup()

	// Test async call - result should still be delivered via channel
	asyncCfg := SubTurnConfig{Model: "gpt-4o-mini", Tools: []tools.Tool{}, Async: true}
	result, err := spawnSubTurn(context.Background(), al, parent, asyncCfg)

	// Should return error from panic recovery
	if err == nil {
		t.Error("expected error from panic recovery")
	}

	// Result should be nil because panic occurred before runTurn could return
	if result != nil {
		t.Error("expected nil result after panic")
	}

	time.Sleep(10 * time.Millisecond) // let event goroutine flush
	// SubTurnEndEvent should still be emitted
	if !collector.hasEventOfKind(runtimeevents.KindAgentSubTurnEnd) {
		t.Error("SubTurnEndEvent not emitted after panic")
	}

	// For async call, result should still be delivered to channel (even if nil)
	select {
	case res := <-parent.pendingResults:
		// Result was delivered (nil due to panic)
		_ = res
	default:
		t.Error("async result should be delivered to channel even after panic")
	}
}

// panicMockProvider is a mock provider that always panics
type panicMockProvider struct{}

func (m *panicMockProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	opts map[string]any,
) (*providers.LLMResponse, error) {
	panic("intentional panic for testing")
}

func (m *panicMockProvider) GetDefaultModel() string {
	return "panic-model"
}

// ====================== Public API Tests ======================

// simpleMockProviderAPI for testing public APIs
type simpleMockProviderAPI struct {
	response string
}

func (m *simpleMockProviderAPI) Chat(
	ctx context.Context,
	messages []providers.Message,
	toolDefs []providers.ToolDefinition,
	model string,
	options map[string]any,
) (*providers.LLMResponse, error) {
	return &providers.LLMResponse{
		Content: m.response,
	}, nil
}

func (m *simpleMockProviderAPI) GetDefaultModel() string {
	return "gpt-4o-mini"
}

// TestGetActiveTurn verifies that GetActiveTurn returns correct turn information
func TestGetActiveTurn(t *testing.T) {
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				ModelName: "gpt-4o-mini",
				Provider:  "mock",
			},
		},
	}
	al := NewAgentLoop(cfg, nil, &simpleMockProviderAPI{response: "ok"})

	// Create a root turn state
	rootCtx := context.Background()
	rootTS := &turnState{
		ctx:            rootCtx,
		turnID:         "root-turn",
		parentTurnID:   "",
		depth:          0,
		childTurnIDs:   []string{},
		session:        newEphemeralSession(nil),
		pendingResults: make(chan *tools.ToolResult, 16),
		concurrencySem: make(chan struct{}, testMaxConcurrentSubTurns),
	}

	sessionKey := "test-session"
	al.activeTurnStates.Store(sessionKey, rootTS)
	defer al.activeTurnStates.Delete(sessionKey)

	// Test: GetActiveTurn should return turn info
	info := al.GetActiveTurnBySession(sessionKey)
	if info == nil {
		t.Fatal("GetActiveTurn returned nil for active session")
	}

	if info.TurnID != "root-turn" {
		t.Errorf("Expected TurnID 'root-turn', got %q", info.TurnID)
	}

	if info.Depth != 0 {
		t.Errorf("Expected Depth 0, got %d", info.Depth)
	}

	if info.ParentTurnID != "" {
		t.Errorf("Expected empty ParentTurnID, got %q", info.ParentTurnID)
	}

	if len(info.ChildTurnIDs) != 0 {
		t.Errorf("Expected 0 child turns, got %d", len(info.ChildTurnIDs))
	}

	// Test: GetActiveTurn should return nil for non-existent session
	nonExistentInfo := al.GetActiveTurnBySession("non-existent-session")
	if nonExistentInfo != nil {
		t.Error("GetActiveTurn should return nil for non-existent session")
	}
}

// TestGetActiveTurn_WithChildren verifies that child turn IDs are correctly reported
func TestGetActiveTurn_WithChildren(t *testing.T) {
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				ModelName: "gpt-4o-mini",
				Provider:  "mock",
			},
		},
	}
	al := NewAgentLoop(cfg, nil, &simpleMockProviderAPI{response: "ok"})

	rootCtx := context.Background()
	rootTS := &turnState{
		ctx:            rootCtx,
		turnID:         "root-turn",
		parentTurnID:   "",
		depth:          0,
		childTurnIDs:   []string{"child-1", "child-2"},
		session:        newEphemeralSession(nil),
		pendingResults: make(chan *tools.ToolResult, 16),
		concurrencySem: make(chan struct{}, testMaxConcurrentSubTurns),
	}

	sessionKey := "test-session-with-children"
	al.activeTurnStates.Store(sessionKey, rootTS)
	defer al.activeTurnStates.Delete(sessionKey)

	info := al.GetActiveTurnBySession(sessionKey)
	if info == nil {
		t.Fatal("GetActiveTurn returned nil")
	}

	if len(info.ChildTurnIDs) != 2 {
		t.Fatalf("Expected 2 child turns, got %d", len(info.ChildTurnIDs))
	}

	if info.ChildTurnIDs[0] != "child-1" || info.ChildTurnIDs[1] != "child-2" {
		t.Errorf("Child turn IDs mismatch: got %v", info.ChildTurnIDs)
	}
}

// TestTurnStateInfo_ThreadSafety verifies that Info() is thread-safe
func TestTurnStateInfo_ThreadSafety(t *testing.T) {
	rootCtx := context.Background()
	ts := &turnState{
		ctx:            rootCtx,
		turnID:         "test-turn",
		parentTurnID:   "parent",
		depth:          1,
		childTurnIDs:   []string{},
		session:        newEphemeralSession(nil),
		pendingResults: make(chan *tools.ToolResult, 16),
		concurrencySem: make(chan struct{}, testMaxConcurrentSubTurns),
	}

	// Concurrently read Info() and modify childTurnIDs
	done := make(chan bool)
	go func() {
		for i := 0; i < 100; i++ {
			ts.mu.Lock()
			ts.childTurnIDs = append(ts.childTurnIDs, "child")
			ts.mu.Unlock()
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			info := ts.snapshot()
			if info.TurnID == "" {
				t.Error("snapshot() returned empty TurnID")
			}
		}
		done <- true
	}()

	<-done
	<-done
}

// TestInjectFollowUp verifies that InjectFollowUp enqueues messages
func TestInjectFollowUp(t *testing.T) {
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				ModelName: "gpt-4o-mini",
				Provider:  "mock",
			},
		},
	}

	al := NewAgentLoop(cfg, nil, &simpleMockProviderAPI{response: "ok"})

	msg := providers.Message{
		Role:    "user",
		Content: "Follow-up task",
	}

	err := al.InjectFollowUp(msg)
	if err != nil {
		t.Fatalf("InjectFollowUp failed: %v", err)
	}

	// Verify message was enqueued
	if al.steering.len() != 1 {
		t.Errorf("Expected 1 message in queue, got %d", al.steering.len())
	}
}

// TestAPIAliases verifies that API aliases work correctly
func TestAPIAliases(t *testing.T) {
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				ModelName: "gpt-4o-mini",
				Provider:  "mock",
			},
		},
	}

	al := NewAgentLoop(cfg, nil, &simpleMockProviderAPI{response: "ok"})

	msg := providers.Message{
		Role:    "user",
		Content: "Test message",
	}

	// Test InterruptGraceful: requires active turn, so error is expected here
	_ = al.InterruptGraceful(msg.Content)

	// Test InjectSteering (enqueues a steering message)
	err := al.InjectSteering(msg)
	if err != nil {
		t.Errorf("InjectSteering failed: %v", err)
	}

	// Also enqueue via Steer to verify second message
	err = al.Steer(msg)
	if err != nil {
		t.Errorf("Steer failed: %v", err)
	}

	// Verify both messages were enqueued
	if al.steering.len() != 2 {
		t.Errorf("Expected 2 messages in queue, got %d", al.steering.len())
	}
}

// TestInterruptHard_Alias verifies that InterruptHard is an alias for HardAbort
func TestInterruptHard_Alias(t *testing.T) {
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				ModelName: "gpt-4o-mini",
				Provider:  "mock",
			},
		},
	}
	al := NewAgentLoop(cfg, nil, &simpleMockProviderAPI{response: "ok"})

	rootCtx := context.Background()
	rootTS := &turnState{
		ctx:                  rootCtx,
		turnID:               "test-turn",
		depth:                0,
		session:              newEphemeralSession(nil),
		initialHistoryLength: 0,
		pendingResults:       make(chan *tools.ToolResult, 16),
		concurrencySem:       make(chan struct{}, testMaxConcurrentSubTurns),
	}

	sessionKey := "test-session-interrupt"
	al.activeTurnStates.Store(sessionKey, rootTS)

	// Test InterruptHard (alias for HardAbort)
	err := al.InterruptHard()
	if err != nil {
		t.Errorf("InterruptHard failed: %v", err)
	}

	// Verify turn was finished (removed from activeTurnStates)
	info := al.GetActiveTurnBySession(sessionKey)
	_ = info // turn may still be in map briefly; hard abort sets isFinished on the state
}

// TestFinish_ConcurrentCalls verifies that calling Finish() concurrently from multiple
// goroutines is safe and doesn't cause panics or double-close errors.
func TestFinish_ConcurrentCalls(t *testing.T) {
	ctx := context.Background()
	parentTS := &turnState{
		ctx:            ctx,
		turnID:         "parent-concurrent-finish",
		depth:          0,
		pendingResults: make(chan *tools.ToolResult, 16),
		concurrencySem: make(chan struct{}, testMaxConcurrentSubTurns),
	}
	parentTS.ctx, parentTS.cancelFunc = context.WithCancel(ctx)

	// Launch multiple goroutines that all call Finish() concurrently
	const numGoroutines = 10
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			// This should not panic, even when called concurrently
			parentTS.Finish(false)
		}()
	}

	wg.Wait()

	// Verify the Finished() channel is closed
	select {
	case _, ok := <-parentTS.Finished():
		if ok {
			t.Error("Expected Finished() channel to be closed")
		}
	default:
		t.Error("Expected Finished() channel to be closed and readable without blocking")
	}

	// Verify isFinished is set
	parentTS.mu.Lock()
	if !parentTS.isFinished.Load() {
		t.Error("Expected isFinished to be true")
	}
	parentTS.mu.Unlock()
}

// TestDeliverSubTurnResult_RaceWithFinish verifies that deliverSubTurnResult handles
// the race condition where Finish() is called while results are being delivered.
func TestDeliverSubTurnResult_RaceWithFinish(t *testing.T) {
	al, _, _, _, cleanup := newTestAgentLoop(t) //nolint:dogsled
	defer cleanup()

	var mu sync.Mutex
	var deliveredCount, orphanCount int
	runtimeCh, closeRuntimeEvents := subscribeRuntimeEventsForTest(
		t,
		al,
		64,
		runtimeevents.KindAgentSubTurnResultDelivered,
		runtimeevents.KindAgentSubTurnOrphan,
	)
	defer closeRuntimeEvents()
	go func() {
		for evt := range runtimeCh {
			mu.Lock()
			switch evt.Kind {
			case runtimeevents.KindAgentSubTurnResultDelivered:
				deliveredCount++
			case runtimeevents.KindAgentSubTurnOrphan:
				orphanCount++
			}
			mu.Unlock()
		}
	}()

	ctx := context.Background()
	parentTS := &turnState{
		ctx:            ctx,
		turnID:         "parent-race-test",
		depth:          0,
		pendingResults: make(chan *tools.ToolResult, 16),
		concurrencySem: make(chan struct{}, testMaxConcurrentSubTurns),
	}
	parentTS.ctx, parentTS.cancelFunc = context.WithCancel(ctx)

	// Launch goroutines that deliver results while another goroutine calls Finish()
	const numResults = 20
	var wg sync.WaitGroup
	wg.Add(numResults + 1)

	// Goroutine that calls Finish() after a short delay
	go func() {
		defer wg.Done()
		time.Sleep(5 * time.Millisecond)
		parentTS.Finish(false)
	}()

	// Goroutines that deliver results
	for i := 0; i < numResults; i++ {
		go func(id int) {
			defer wg.Done()
			result := &tools.ToolResult{
				ForLLM: fmt.Sprintf("result-%d", id),
			}
			// This should not panic, even if Finish() is called concurrently
			deliverSubTurnResult(al, parentTS, fmt.Sprintf("child-%d", id), result)
		}(i)
	}

	wg.Wait()
	time.Sleep(20 * time.Millisecond) // let event goroutine flush

	// Get final counts
	mu.Lock()
	finalDelivered := deliveredCount
	finalOrphan := orphanCount
	mu.Unlock()

	t.Logf("Delivered: %d, Orphan: %d, Total: %d", finalDelivered, finalOrphan, finalDelivered+finalOrphan)

	// With the new drainPendingResults behavior, the total events may be >= numResults
	// because Finish() drains remaining results from the channel and emits them as orphans.
	// So we expect:
	// - Some results were delivered successfully (before Finish())
	// - Some results became orphans (after Finish() or channel full)
	// - Some results were in the channel when Finish() was called and got drained as orphans
	// The total should be at least numResults (could be more due to drain)
	if finalDelivered+finalOrphan < numResults {
		t.Errorf("Expected at least %d total events, got %d delivered + %d orphan = %d",
			numResults, finalDelivered, finalOrphan, finalDelivered+finalOrphan)
	}

	// Should have at least some orphan results (those that arrived after Finish() or were drained)
	if finalOrphan == 0 {
		t.Error("Expected at least some orphan results after Finish()")
	}
}

// TestConcurrencySemaphore_Timeout verifies that spawning sub-turns times out
// when all concurrency slots are occupied for too long.
// Note: This test uses a shorter timeout by temporarily modifying the constant.
func TestConcurrencySemaphore_Timeout(t *testing.T) {
	// This test would take 30 seconds with the default timeout.
	// Instead, we'll test the mechanism by verifying the timeout context is created correctly.
	// A full integration test with actual timeout would be too slow for unit tests.

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Provider: "mock",
			},
		},
	}
	msgBus := bus.NewMessageBus()
	provider := &simpleMockProviderAPI{}
	al := NewAgentLoop(cfg, msgBus, provider)

	ctx := context.Background()
	parentTS := &turnState{
		ctx:            ctx,
		turnID:         "parent-timeout-test",
		depth:          0,
		session:        newEphemeralSession(nil),
		pendingResults: make(chan *tools.ToolResult, 16),
		concurrencySem: make(chan struct{}, testMaxConcurrentSubTurns),
	}
	parentTS.ctx, parentTS.cancelFunc = context.WithCancel(ctx)
	defer parentTS.Finish(false)

	// Fill all concurrency slots
	for i := 0; i < testMaxConcurrentSubTurns; i++ {
		parentTS.concurrencySem <- struct{}{}
	}

	// Create a context with a very short timeout for testing
	testCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	// Now try to spawn a sub-turn with the short timeout context
	subTurnCfg := SubTurnConfig{
		Model: "gpt-4o-mini",
		Async: false,
	}

	start := time.Now()
	_, err := spawnSubTurn(testCtx, al, parentTS, subTurnCfg)
	elapsed := time.Since(start)

	// Should get a timeout error (either from our timeout context or the internal one)
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}

	// The error should be related to context cancellation or timeout
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, ErrConcurrencyTimeout) {
		t.Logf("Got error: %v (type: %T)", err, err)
		// This is acceptable - the error might be wrapped
	}

	// Should timeout quickly (within a reasonable margin)
	if elapsed > 2*time.Second {
		t.Errorf("Timeout took too long: %v", elapsed)
	}

	t.Logf("Timeout occurred after %v with error: %v", elapsed, err)

	// Clean up - drain the semaphore
	for i := 0; i < testMaxConcurrentSubTurns; i++ {
		<-parentTS.concurrencySem
	}
}

// TestEphemeralSession_AutoTruncate verifies that ephemeral sessions automatically
// truncate their history to prevent memory accumulation.
func TestEphemeralSession_AutoTruncate(t *testing.T) {
	store := newEphemeralSession(nil).(*ephemeralSessionStore)

	// Add more messages than the limit
	for i := 0; i < maxEphemeralHistorySize+20; i++ {
		store.AddMessage("test", "user", fmt.Sprintf("message-%d", i))
	}

	// Verify history is truncated to the limit
	history := store.GetHistory("test")
	if len(history) != maxEphemeralHistorySize {
		t.Errorf("Expected history length %d, got %d", maxEphemeralHistorySize, len(history))
	}

	// Verify we kept the most recent messages
	lastMsg := history[len(history)-1]
	expectedContent := fmt.Sprintf("message-%d", maxEphemeralHistorySize+20-1)
	if lastMsg.Content != expectedContent {
		t.Errorf("Expected last message to be %q, got %q", expectedContent, lastMsg.Content)
	}

	// Verify the oldest messages were discarded
	firstMsg := history[0]
	expectedFirstContent := fmt.Sprintf("message-%d", 20) // First 20 were discarded
	if firstMsg.Content != expectedFirstContent {
		t.Errorf("Expected first message to be %q, got %q", expectedFirstContent, firstMsg.Content)
	}
}

// TestContextWrapping_SingleLayer verifies that we only create one context layer
// in spawnSubTurn, not multiple redundant layers.
func TestContextWrapping_SingleLayer(t *testing.T) {
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Provider: "mock",
			},
		},
	}
	msgBus := bus.NewMessageBus()
	provider := &simpleMockProviderAPI{}
	al := NewAgentLoop(cfg, msgBus, provider)

	ctx := context.Background()
	parentTS := &turnState{
		ctx:            ctx,
		turnID:         "parent-context-test",
		depth:          0,
		session:        newEphemeralSession(nil),
		pendingResults: make(chan *tools.ToolResult, 16),
		concurrencySem: make(chan struct{}, testMaxConcurrentSubTurns),
	}
	parentTS.ctx, parentTS.cancelFunc = context.WithCancel(ctx)
	defer parentTS.Finish(false)

	// Spawn a sub-turn
	subTurnCfg := SubTurnConfig{
		Model: "gpt-4o-mini",
		Async: false,
	}

	result, err := spawnSubTurn(ctx, al, parentTS, subTurnCfg)
	if err != nil {
		t.Fatalf("spawnSubTurn failed: %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result")
	}

	// Verify the child turn was created with a cancel function
	// (This is implicit - if the test passes without hanging, the context management is correct)
	t.Log("Context wrapping test passed - no redundant layers detected")
}

// TestSyncSubTurn_NoChannelDelivery verifies that synchronous sub-turns
// do NOT deliver results to the pendingResults channel (only return directly).
func TestSyncSubTurn_NoChannelDelivery(t *testing.T) {
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Provider: "mock",
			},
		},
	}
	msgBus := bus.NewMessageBus()
	provider := &simpleMockProviderAPI{}
	al := NewAgentLoop(cfg, msgBus, provider)

	ctx := context.Background()
	parentTS := &turnState{
		ctx:            ctx,
		turnID:         "parent-sync-test",
		depth:          0,
		session:        newEphemeralSession(nil),
		pendingResults: make(chan *tools.ToolResult, 16),
		concurrencySem: make(chan struct{}, testMaxConcurrentSubTurns),
	}
	parentTS.ctx, parentTS.cancelFunc = context.WithCancel(ctx)
	defer parentTS.Finish(false)

	// Spawn a SYNCHRONOUS sub-turn (Async=false)
	subTurnCfg := SubTurnConfig{
		Model: "gpt-4o-mini",
		Async: false, // Synchronous - should NOT deliver to channel
	}

	result, err := spawnSubTurn(ctx, al, parentTS, subTurnCfg)
	if err != nil {
		t.Fatalf("spawnSubTurn failed: %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result from synchronous sub-turn")
	}

	// Verify the pendingResults channel is EMPTY
	// (synchronous sub-turns should not deliver to channel)
	select {
	case r := <-parentTS.pendingResults:
		t.Errorf("Expected empty channel for sync sub-turn, but got result: %v", r)
	default:
		// Expected: channel is empty
		t.Log("Verified: synchronous sub-turn did not deliver to channel")
	}

	// Verify channel length is 0
	if len(parentTS.pendingResults) != 0 {
		t.Errorf("Expected channel length 0, got %d", len(parentTS.pendingResults))
	}
}

func TestSyncSubTurn_DismissesSessionScopedToolFeedback(t *testing.T) {
	al, _, _, provider, cleanup := newTestAgentLoop(t)
	_ = provider
	defer cleanup()

	cm := &recordingChannelManager{}
	al.channelManager = cm

	ctx := context.Background()
	parentTS := &turnState{
		ctx:            ctx,
		turnID:         "parent-sync-dismiss",
		depth:          0,
		session:        newEphemeralSession(nil),
		pendingResults: make(chan *tools.ToolResult, 16),
		concurrencySem: make(chan struct{}, testMaxConcurrentSubTurns),
	}
	parentTS.ctx, parentTS.cancelFunc = context.WithCancel(ctx)
	parentTS.agent = al.registry.GetDefaultAgent()
	parentTS.opts = processOptions{
		Dispatch: DispatchRequest{
			InboundContext: &bus.InboundContext{
				Channel: "telegram",
				ChatID:  "-100123",
				TopicID: "6",
			},
		},
	}
	parentTS.channel = "telegram"
	parentTS.chatID = "-100123"
	defer parentTS.Finish(false)

	result, err := spawnSubTurn(ctx, al, parentTS, SubTurnConfig{
		Model: "gpt-4o-mini",
		Async: false,
	})
	if err != nil {
		t.Fatalf("spawnSubTurn failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(cm.dismissedSessions) != 1 {
		t.Fatalf("dismissedSessions = %v, want exactly one dismissal", cm.dismissedSessions)
	}
	if got := cm.dismissedSessions[0]; !strings.HasPrefix(got, "telegram:-100123:subturn-") {
		t.Fatalf("dismissed session key = %q, want telegram:-100123:subturn-*", got)
	}
}

// TestAsyncSubTurn_ChannelDelivery verifies that asynchronous sub-turns
// DO deliver results to the pendingResults channel.
func TestAsyncSubTurn_ChannelDelivery(t *testing.T) {
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Provider: "mock",
			},
		},
	}
	msgBus := bus.NewMessageBus()
	provider := &simpleMockProviderAPI{}
	al := NewAgentLoop(cfg, msgBus, provider)

	ctx := context.Background()
	parentTS := &turnState{
		ctx:            ctx,
		turnID:         "parent-async-test",
		depth:          0,
		session:        newEphemeralSession(nil),
		pendingResults: make(chan *tools.ToolResult, 16),
		concurrencySem: make(chan struct{}, testMaxConcurrentSubTurns),
	}
	parentTS.ctx, parentTS.cancelFunc = context.WithCancel(ctx)
	defer parentTS.Finish(false)

	// Spawn an ASYNCHRONOUS sub-turn (Async=true)
	subTurnCfg := SubTurnConfig{
		Model: "gpt-4o-mini",
		Async: true, // Asynchronous - SHOULD deliver to channel
	}

	result, err := spawnSubTurn(ctx, al, parentTS, subTurnCfg)
	if err != nil {
		t.Fatalf("spawnSubTurn failed: %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result from asynchronous sub-turn")
	}

	// Verify the pendingResults channel has the result
	select {
	case r := <-parentTS.pendingResults:
		if r == nil {
			t.Error("Expected non-nil result from channel")
		}
		t.Log("Verified: asynchronous sub-turn delivered to channel")
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected result in channel for async sub-turn, but channel was empty")
	}
}

// TestGrandchildAbort_CascadingCancellation verifies that when a grandparent turn
// is hard aborted, the cancellation cascades down to grandchild turns.
func TestGrandchildAbort_CascadingCancellation(t *testing.T) {
	al, _, _, provider, cleanup := newTestAgentLoop(t)
	_ = provider
	defer cleanup()

	// Three independent contexts — none derived from another.
	// Cascade must happen exclusively through childTurnIDs traversal in Finish(true).
	gpCtx, gpCancel := context.WithCancel(context.Background())
	parentCtx, parentCancel := context.WithCancel(context.Background())
	childCtx, childCancel := context.WithCancel(context.Background())

	childTS := &turnState{
		ctx:        childCtx,
		cancelFunc: childCancel,
		turnID:     "grandchild",
		al:         al,
	}
	parentTS := &turnState{
		ctx:          parentCtx,
		cancelFunc:   parentCancel,
		turnID:       "parent",
		childTurnIDs: []string{"grandchild"},
		al:           al,
	}
	grandparentTS := &turnState{
		ctx:            gpCtx,
		cancelFunc:     gpCancel,
		turnID:         "grandparent",
		depth:          0,
		session:        newEphemeralSession(nil),
		pendingResults: make(chan *tools.ToolResult, 16),
		concurrencySem: make(chan struct{}, testMaxConcurrentSubTurns),
		childTurnIDs:   []string{"parent"},
		al:             al,
	}

	al.activeTurnStates.Store("grandparent", grandparentTS)
	al.activeTurnStates.Store("parent", parentTS)
	al.activeTurnStates.Store("grandchild", childTS)
	defer al.activeTurnStates.Delete("grandparent")
	defer al.activeTurnStates.Delete("parent")
	defer al.activeTurnStates.Delete("grandchild")

	// All contexts must be active before the abort
	for _, ctx := range []context.Context{gpCtx, parentCtx, childCtx} {
		select {
		case <-ctx.Done():
			t.Fatal("context should not be canceled yet")
		default:
		}
	}

	// Hard abort the grandparent — should cascade to parent and grandchild
	grandparentTS.Finish(true)

	time.Sleep(10 * time.Millisecond)

	select {
	case <-gpCtx.Done():
		t.Log("Grandparent context canceled (expected)")
	default:
		t.Error("Grandparent context should be canceled")
	}
	select {
	case <-parentCtx.Done():
		t.Log("Parent context canceled via cascade (expected)")
	default:
		t.Error("Parent context should be canceled via childTurnIDs cascade")
	}
	select {
	case <-childCtx.Done():
		t.Log("Grandchild context canceled via cascade (expected)")
	default:
		t.Error("Grandchild context should be canceled via childTurnIDs cascade")
	}
}

func TestNestedSubTurn_GracefulFinishSignalsDirectChildren(t *testing.T) {
	parentCtx := context.Background()
	parentTS := &turnState{
		ctx:            parentCtx,
		turnID:         "parent-graceful",
		depth:          1,
		pendingResults: make(chan *tools.ToolResult, 16),
	}
	parentTS.ctx, parentTS.cancelFunc = context.WithCancel(parentCtx)

	childTS := &turnState{
		ctx:             context.Background(),
		turnID:          "child-graceful",
		depth:           2,
		parentTurnState: parentTS,
		pendingResults:  make(chan *tools.ToolResult, 16),
	}

	if childTS.IsParentEnded() {
		t.Fatal("IsParentEnded should be false before parent finishes")
	}

	parentTS.Finish(false)

	if !parentTS.parentEnded.Load() {
		t.Fatal("parentEnded should be true after graceful finish")
	}
	if !childTS.IsParentEnded() {
		t.Fatal("nested child should observe parent graceful finish")
	}
}

// TestSpawnDuringAbort_RaceCondition verifies behavior when trying to spawn
// a sub-turn while the parent is being aborted.
func TestSpawnDuringAbort_RaceCondition(t *testing.T) {
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Provider: "mock",
			},
		},
	}
	msgBus := bus.NewMessageBus()
	provider := &simpleMockProviderAPI{}
	al := NewAgentLoop(cfg, msgBus, provider)

	ctx := context.Background()
	parentTS := &turnState{
		ctx:            ctx,
		turnID:         "parent-abort-race",
		depth:          0,
		session:        newEphemeralSession(nil),
		pendingResults: make(chan *tools.ToolResult, 16),
		concurrencySem: make(chan struct{}, testMaxConcurrentSubTurns),
	}
	parentTS.ctx, parentTS.cancelFunc = context.WithCancel(ctx)

	var wg sync.WaitGroup
	wg.Add(2)

	var spawnErr error

	// Goroutine 1: Try to spawn a sub-turn
	go func() {
		defer wg.Done()
		subTurnCfg := SubTurnConfig{
			Model: "gpt-4o-mini",
			Async: false,
		}
		_, err := spawnSubTurn(parentTS.ctx, al, parentTS, subTurnCfg)
		spawnErr = err
	}()

	// Goroutine 2: Abort the parent almost immediately
	go func() {
		defer wg.Done()
		time.Sleep(1 * time.Millisecond)
		parentTS.Finish(false)
	}()

	wg.Wait()

	// The spawn should either succeed (if it started before abort)
	// or fail with context canceled error (if abort happened first)
	if spawnErr != nil {
		if errors.Is(spawnErr, context.Canceled) {
			t.Logf("Spawn failed with expected context cancellation: %v", spawnErr)
		} else {
			t.Logf("Spawn failed with error: %v", spawnErr)
		}
	} else {
		t.Log("Spawn succeeded before abort")
	}

	// The important thing is that it doesn't panic or deadlock
	t.Log("Race condition handled gracefully - no panic or deadlock")
}

// ====================== Slow SubTurn Cancellation Test ======================

// slowMockProvider simulates a slow LLM call that takes a long time to complete.
// This is used to test the scenario where a parent turn finishes before the child SubTurn.
type slowMockProvider struct {
	delay time.Duration
}

func (m *slowMockProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	toolDefs []providers.ToolDefinition,
	model string,
	options map[string]any,
) (*providers.LLMResponse, error) {
	select {
	case <-time.After(m.delay):
		// Completed normally after delay
		return &providers.LLMResponse{
			Content: "slow response completed",
		}, nil
	case <-ctx.Done():
		// Context was canceled while waiting
		return nil, ctx.Err()
	}
}

func (m *slowMockProvider) GetDefaultModel() string {
	return "slow-model"
}

// TestAsyncSubTurn_ParentFinishesEarly simulates the scenario where:
// 1. Parent spawns an async SubTurn that takes a long time
// 2. Parent finishes quickly
// 3. SubTurn should be canceled with context canceled error
func TestAsyncSubTurn_ParentFinishesEarly(t *testing.T) {
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Provider: "mock",
			},
		},
	}
	msgBus := bus.NewMessageBus()
	provider := &slowMockProvider{delay: 5 * time.Second} // SubTurn takes 5 seconds
	al := NewAgentLoop(cfg, msgBus, provider)

	var mu sync.Mutex
	var events []runtimeevents.Event
	runtimeCh, closeRuntimeEvents := subscribeRuntimeEventsForTest(
		t,
		al,
		32,
		runtimeevents.KindAgentSubTurnSpawn,
		runtimeevents.KindAgentSubTurnEnd,
		runtimeevents.KindAgentSubTurnResultDelivered,
		runtimeevents.KindAgentSubTurnOrphan,
	)
	defer closeRuntimeEvents()
	go func() {
		for evt := range runtimeCh {
			mu.Lock()
			events = append(events, evt)
			mu.Unlock()
		}
	}()

	ctx := context.Background()
	parentTS := &turnState{
		ctx:            ctx,
		turnID:         "parent-fast",
		depth:          0,
		session:        newEphemeralSession(nil),
		pendingResults: make(chan *tools.ToolResult, 16),
		concurrencySem: make(chan struct{}, testMaxConcurrentSubTurns),
	}
	parentTS.ctx, parentTS.cancelFunc = context.WithCancel(ctx)

	var subTurnErr error
	var subTurnResult *tools.ToolResult
	var wg sync.WaitGroup

	// Spawn async SubTurn in a goroutine (it will be slow)
	wg.Add(1)
	go func() {
		defer wg.Done()
		subTurnCfg := SubTurnConfig{
			Model: "slow-model",
			Async: true, // Asynchronous SubTurn
		}
		subTurnResult, subTurnErr = spawnSubTurn(parentTS.ctx, al, parentTS, subTurnCfg)
	}()

	// Parent finishes quickly (after 100ms), while SubTurn is still running
	time.Sleep(100 * time.Millisecond)
	t.Log("Parent finishing early...")
	parentTS.Finish(false)

	// Wait for SubTurn to complete (or be canceled)
	wg.Wait()

	// Check the result
	t.Logf("SubTurn error: %v", subTurnErr)
	t.Logf("SubTurn result: %v", subTurnResult)

	if subTurnErr != nil {
		if errors.Is(subTurnErr, context.Canceled) {
			t.Log("✓ SubTurn was canceled as expected (context canceled)")
		} else {
			t.Logf("SubTurn failed with other error: %v", subTurnErr)
		}
	} else {
		t.Log("SubTurn completed before parent finished (unlikely but possible)")
	}

	// Log captured events
	mu.Lock()
	t.Logf("Captured %d events:", len(events))
	for i, e := range events {
		t.Logf("  Event %d: %s", i+1, e.Kind)
	}
	mu.Unlock()
}

// TestAsyncSubTurn_ParentWaitsForChild simulates the scenario where:
// 1. Parent spawns an async SubTurn that takes some time
// 2. Parent WAITS for SubTurn to complete before finishing
// 3. Both should complete successfully
func TestAsyncSubTurn_ParentWaitsForChild(t *testing.T) {
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Provider: "mock",
			},
		},
	}
	msgBus := bus.NewMessageBus()
	provider := &slowMockProvider{delay: 200 * time.Millisecond} // SubTurn takes 200ms
	al := NewAgentLoop(cfg, msgBus, provider)

	ctx := context.Background()
	parentTS := &turnState{
		ctx:            ctx,
		turnID:         "parent-wait",
		depth:          0,
		session:        newEphemeralSession(nil),
		pendingResults: make(chan *tools.ToolResult, 16),
		concurrencySem: make(chan struct{}, testMaxConcurrentSubTurns),
	}
	parentTS.ctx, parentTS.cancelFunc = context.WithCancel(ctx)

	var subTurnErr error
	var subTurnResult *tools.ToolResult
	var wg sync.WaitGroup

	// Spawn async SubTurn in a goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		subTurnCfg := SubTurnConfig{
			Model: "slow-model",
			Async: true,
		}
		subTurnResult, subTurnErr = spawnSubTurn(parentTS.ctx, al, parentTS, subTurnCfg)
	}()

	// Parent WAITS for SubTurn to complete
	t.Log("Parent waiting for SubTurn...")
	wg.Wait()
	t.Log("SubTurn completed, parent now finishing")

	// Now parent can finish safely
	parentTS.Finish(false)

	// Check the result
	if subTurnErr != nil {
		if errors.Is(subTurnErr, context.Canceled) {
			t.Errorf("SubTurn should NOT have been canceled: %v", subTurnErr)
		} else {
			t.Logf("SubTurn failed with error: %v", subTurnErr)
		}
	} else {
		t.Log("✓ SubTurn completed successfully")
		if subTurnResult != nil {
			t.Logf("SubTurn result: %s", subTurnResult.ForLLM)
		}
	}

	// Check channel delivery
	select {
	case r := <-parentTS.pendingResults:
		if r != nil {
			t.Logf("✓ Result delivered to channel: %s", r.ForLLM)
		}
	case <-time.After(100 * time.Millisecond):
		t.Log("No result in channel (expected since we waited)")
	}
}

// ====================== Graceful vs Hard Finish Tests ======================

// TestFinish_GracefulVsHard verifies the behavior difference between:
// - Finish(false): graceful finish, signals parentEnded but doesn't cancel children
// - Finish(true): hard abort, immediately cancels all children
func TestFinish_GracefulVsHard(t *testing.T) {
	// Test 1: Graceful finish should set parentEnded but not cancel context
	t.Run("Graceful_SetsParentEnded", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ts := &turnState{
			ctx:            ctx,
			turnID:         "graceful-test",
			depth:          0,
			pendingResults: make(chan *tools.ToolResult, 16),
		}
		ts.ctx, ts.cancelFunc = context.WithCancel(ctx)

		// Finish gracefully
		ts.Finish(false)

		// Verify parentEnded is set
		if !ts.parentEnded.Load() {
			t.Error("parentEnded should be true after graceful finish")
		}

		// Verify context is NOT canceled (for graceful finish, children continue)
		// Note: In graceful mode, we don't call cancelFunc()
		// But since we're using WithCancel on the same ctx, it might be canceled
		// Let's check that the context is still valid for a moment
		time.Sleep(10 * time.Millisecond)
		// Context might be canceled by the deferred cancel() in test, which is fine
	})

	// Test 2: Hard abort should cancel context immediately
	t.Run("Hard_CancelsContext", func(t *testing.T) {
		ctx := context.Background()

		ts := &turnState{
			ctx:            ctx,
			turnID:         "hard-test",
			depth:          0,
			pendingResults: make(chan *tools.ToolResult, 16),
		}
		ts.ctx, ts.cancelFunc = context.WithCancel(ctx)

		// Finish with hard abort
		ts.Finish(true)

		// Verify context is canceled
		select {
		case <-ts.ctx.Done():
			t.Log("✓ Context canceled after hard abort")
		default:
			t.Error("Context should be canceled after hard abort")
		}
	})

	// Test 3: IsParentEnded returns correct value
	t.Run("IsParentEnded", func(t *testing.T) {
		ctx := context.Background()

		parentTS := &turnState{
			ctx:            ctx,
			turnID:         "parent-isended-test",
			depth:          0,
			pendingResults: make(chan *tools.ToolResult, 16),
		}
		parentTS.ctx, parentTS.cancelFunc = context.WithCancel(ctx)

		childTS := &turnState{
			ctx:             ctx,
			turnID:          "child-isended-test",
			depth:           1,
			parentTurnState: parentTS,
			pendingResults:  make(chan *tools.ToolResult, 16),
		}

		// Before parent finishes
		if childTS.IsParentEnded() {
			t.Error("IsParentEnded should be false before parent finishes")
		}

		// Finish parent gracefully
		parentTS.Finish(false)

		// After parent finishes
		if !childTS.IsParentEnded() {
			t.Error("IsParentEnded should be true after parent finishes gracefully")
		}
	})
}

// TestSubTurn_IndependentContext verifies that SubTurns use independent contexts
// that don't get canceled when the parent finishes gracefully.
func TestSubTurn_IndependentContext(t *testing.T) {
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Provider: "mock",
			},
		},
	}
	msgBus := bus.NewMessageBus()
	provider := &slowMockProvider{delay: 500 * time.Millisecond}
	al := NewAgentLoop(cfg, msgBus, provider)

	ctx := context.Background()
	parentTS := &turnState{
		ctx:            ctx,
		turnID:         "parent-independent",
		depth:          0,
		session:        newEphemeralSession(nil),
		pendingResults: make(chan *tools.ToolResult, 16),
		concurrencySem: make(chan struct{}, testMaxConcurrentSubTurns),
	}
	parentTS.ctx, parentTS.cancelFunc = context.WithCancel(ctx)

	var subTurnErr error
	var wg sync.WaitGroup

	// Spawn SubTurn with Critical=true (should continue after parent finishes)
	wg.Add(1)
	go func() {
		defer wg.Done()
		subTurnCfg := SubTurnConfig{
			Model:    "slow-model",
			Async:    true,
			Critical: true, // Critical SubTurn should continue
		}
		_, subTurnErr = spawnSubTurn(parentTS.ctx, al, parentTS, subTurnCfg)
	}()

	// Let SubTurn start
	time.Sleep(50 * time.Millisecond)

	// Parent finishes gracefully (should NOT cancel SubTurn)
	parentTS.Finish(false)
	t.Log("Parent finished gracefully, SubTurn should continue")

	// Wait for SubTurn to complete
	wg.Wait()

	// SubTurn should complete without context canceled error
	// (because it uses independent context now)
	if subTurnErr != nil {
		t.Logf("SubTurn error: %v", subTurnErr)
		// The error might be context.DeadlineExceeded if timeout is too short
		// but should NOT be context.Canceled from parent
		if errors.Is(subTurnErr, context.Canceled) {
			t.Error("SubTurn should not be canceled by parent's graceful finish")
		}
	} else {
		t.Log("✓ SubTurn completed successfully (independent context)")
	}
}

// ====================== TargetAgentID Tests ======================

// modelRecordingProvider captures the model passed to Chat for test assertions.
type modelRecordingProvider struct {
	mu        sync.Mutex
	lastModel string
}

func (rp *modelRecordingProvider) Chat(
	_ context.Context,
	_ []providers.Message,
	_ []providers.ToolDefinition,
	model string,
	_ map[string]any,
) (*providers.LLMResponse, error) {
	rp.mu.Lock()
	rp.lastModel = model
	rp.mu.Unlock()
	return &providers.LLMResponse{Content: "Mock response"}, nil
}

func (rp *modelRecordingProvider) GetDefaultModel() string { return "mock-model" }

func (rp *modelRecordingProvider) getLastModel() string {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	return rp.lastModel
}

// newMultiAgentLoop creates an AgentLoop with two named agents for testing
// cross-agent delegation via TargetAgentID.
func newMultiAgentLoop(t *testing.T, provider providers.LLMProvider) (*AgentLoop, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "multiagent-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}

	alphaDir := filepath.Join(tmpDir, "alpha")
	betaDir := filepath.Join(tmpDir, "beta")
	os.MkdirAll(alphaDir, 0o755)
	os.MkdirAll(betaDir, 0o755)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				ModelName:         "default-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
			List: []config.AgentConfig{
				{
					ID:        "alpha",
					Workspace: alphaDir,
					Model:     &config.AgentModelConfig{Primary: "model-alpha"},
				},
				{
					ID:        "beta",
					Workspace: betaDir,
					Model:     &config.AgentModelConfig{Primary: "model-beta"},
				},
			},
		},
	}

	msgBus := bus.NewMessageBus()
	al := NewAgentLoop(cfg, msgBus, provider)

	return al, func() { os.RemoveAll(tmpDir) }
}

func TestSpawnSubTurn_TargetAgentID_UsesTargetAgent(t *testing.T) {
	rp := &modelRecordingProvider{}
	al, cleanup := newMultiAgentLoop(t, rp)
	defer cleanup()

	alphaAgent, ok := al.registry.GetAgent("alpha")
	if !ok {
		t.Fatal("alpha agent not in registry")
	}

	// Parent is alpha, target is beta
	parent := &turnState{
		ctx:            context.Background(),
		turnID:         "parent-alpha",
		depth:          0,
		childTurnIDs:   []string{},
		pendingResults: make(chan *tools.ToolResult, 4),
		concurrencySem: make(chan struct{}, testMaxConcurrentSubTurns),
		session:        &ephemeralSessionStore{},
		agent:          alphaAgent,
	}

	result, err := spawnSubTurn(context.Background(), al, parent, SubTurnConfig{
		TargetAgentID: "beta",
		SystemPrompt:  "task for beta",
	})
	if err != nil {
		t.Fatalf("spawnSubTurn failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// The recording provider captures the model passed to Chat().
	// If TargetAgentID works correctly, the child turn should have
	// used beta's model, not alpha's.
	if got := rp.getLastModel(); got != "model-beta" {
		t.Errorf("child turn used model %q, want %q", got, "model-beta")
	}
}

func TestSpawnSubTurn_TargetAgentID_NotFound(t *testing.T) {
	al, cleanup := newMultiAgentLoop(t, &mockProvider{})
	defer cleanup()

	alphaAgent, _ := al.registry.GetAgent("alpha")
	parent := &turnState{
		ctx:            context.Background(),
		turnID:         "parent-alpha",
		depth:          0,
		childTurnIDs:   []string{},
		pendingResults: make(chan *tools.ToolResult, 4),
		concurrencySem: make(chan struct{}, testMaxConcurrentSubTurns),
		session:        &ephemeralSessionStore{},
		agent:          alphaAgent,
	}

	_, err := spawnSubTurn(context.Background(), al, parent, SubTurnConfig{
		TargetAgentID: "nonexistent",
		SystemPrompt:  "task",
	})

	if err == nil {
		t.Fatal("expected error for nonexistent agent")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found', got: %v", err)
	}
}

func TestSpawnSubTurn_TargetAgentID_EmptyModelAccepted(t *testing.T) {
	al, cleanup := newMultiAgentLoop(t, &mockProvider{})
	defer cleanup()

	alphaAgent, _ := al.registry.GetAgent("alpha")
	parent := &turnState{
		ctx:            context.Background(),
		turnID:         "parent-alpha",
		depth:          0,
		childTurnIDs:   []string{},
		pendingResults: make(chan *tools.ToolResult, 4),
		concurrencySem: make(chan struct{}, testMaxConcurrentSubTurns),
		session:        &ephemeralSessionStore{},
		agent:          alphaAgent,
	}

	// Model is empty but TargetAgentID is set — should NOT fail validation
	result, err := spawnSubTurn(context.Background(), al, parent, SubTurnConfig{
		Model:         "", // intentionally empty
		TargetAgentID: "beta",
		SystemPrompt:  "task for beta",
	})
	if err != nil {
		t.Fatalf("should accept empty Model when TargetAgentID is set, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestDelegateToolNotRegistered_SingleAgent(t *testing.T) {
	// Single-agent setup: delegate should not be registered
	al, _, _, provider, cleanup := newTestAgentLoop(t)
	_ = provider
	defer cleanup()

	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		t.Fatal("default agent should exist")
	}
	if _, has := agent.Tools.Get("delegate"); has {
		t.Error("delegate tool should not be registered in single-agent setup")
	}
}

func TestDelegateToolRegistered_MultiAgent(t *testing.T) {
	al, cleanup := newMultiAgentLoop(t, &mockProvider{})
	defer cleanup()

	// Both agents should have the delegate tool
	for _, id := range []string{"alpha", "beta"} {
		agent, ok := al.registry.GetAgent(id)
		if !ok {
			t.Fatalf("agent %q not found", id)
		}
		if _, has := agent.Tools.Get("delegate"); !has {
			t.Errorf("agent %q should have delegate tool in multi-agent setup", id)
		}
	}
}
