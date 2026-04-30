package agent

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/session"
	"github.com/sipeed/picoclaw/pkg/tools"
)

// SteeringMode controls how queued steering messages are dequeued.
type SteeringMode string

const (
	// SteeringOneAtATime dequeues only the first queued message per poll.
	SteeringOneAtATime SteeringMode = "one-at-a-time"
	// SteeringAll drains the entire queue in a single poll.
	SteeringAll SteeringMode = "all"
	// MaxQueueSize number of possible messages in the Steering Queue
	MaxQueueSize = 10
	// manualSteeringScope is the legacy fallback queue used when no active
	// turn/session scope is available.
	manualSteeringScope = "__manual__"
)

// parseSteeringMode normalizes a config string into a SteeringMode.
func parseSteeringMode(s string) SteeringMode {
	switch s {
	case "all":
		return SteeringAll
	default:
		return SteeringOneAtATime
	}
}

// steeringQueue is a thread-safe queue of user messages that can be injected
// into a running agent loop to interrupt it between tool calls.
type steeringQueue struct {
	mu     sync.Mutex
	queues map[string][]providers.Message
	mode   SteeringMode
}

func newSteeringQueue(mode SteeringMode) *steeringQueue {
	return &steeringQueue{
		queues: make(map[string][]providers.Message),
		mode:   mode,
	}
}

func normalizeSteeringScope(scope string) string {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return manualSteeringScope
	}
	return scope
}

// push enqueues a steering message in the legacy fallback scope.
func (sq *steeringQueue) push(msg providers.Message) error {
	return sq.pushScope(manualSteeringScope, msg)
}

// pushScope enqueues a steering message for the provided scope.
func (sq *steeringQueue) pushScope(scope string, msg providers.Message) error {
	sq.mu.Lock()
	defer sq.mu.Unlock()

	scope = normalizeSteeringScope(scope)
	queue := sq.queues[scope]
	if len(queue) >= MaxQueueSize {
		return fmt.Errorf("steering queue is full")
	}
	sq.queues[scope] = append(queue, msg)
	return nil
}

// dequeue removes and returns pending steering messages from the legacy
// fallback scope according to the configured mode.
func (sq *steeringQueue) dequeue() []providers.Message {
	return sq.dequeueScope(manualSteeringScope)
}

// dequeueScope removes and returns pending steering messages for the provided
// scope according to the configured mode.
func (sq *steeringQueue) dequeueScope(scope string) []providers.Message {
	sq.mu.Lock()
	defer sq.mu.Unlock()

	return sq.dequeueLocked(normalizeSteeringScope(scope))
}

// dequeueScopeWithFallback drains the scoped queue first and falls back to the
// legacy manual scope for backwards compatibility.
func (sq *steeringQueue) dequeueScopeWithFallback(scope string) []providers.Message {
	sq.mu.Lock()
	defer sq.mu.Unlock()

	scope = strings.TrimSpace(scope)
	if scope != "" {
		if msgs := sq.dequeueLocked(scope); len(msgs) > 0 {
			return msgs
		}
	}

	return sq.dequeueLocked(manualSteeringScope)
}

func (sq *steeringQueue) dequeueLocked(scope string) []providers.Message {
	queue := sq.queues[scope]
	if len(queue) == 0 {
		return nil
	}

	switch sq.mode {
	case SteeringAll:
		msgs := append([]providers.Message(nil), queue...)
		delete(sq.queues, scope)
		return msgs
	default:
		msg := queue[0]
		queue[0] = providers.Message{} // Clear reference for GC
		queue = queue[1:]
		if len(queue) == 0 {
			delete(sq.queues, scope)
		} else {
			sq.queues[scope] = queue
		}
		return []providers.Message{msg}
	}
}

// len returns the number of queued messages across all scopes.
func (sq *steeringQueue) len() int {
	sq.mu.Lock()
	defer sq.mu.Unlock()

	total := 0
	for _, queue := range sq.queues {
		total += len(queue)
	}
	return total
}

// lenScope returns the number of queued messages for a specific scope.
func (sq *steeringQueue) lenScope(scope string) int {
	sq.mu.Lock()
	defer sq.mu.Unlock()
	return len(sq.queues[normalizeSteeringScope(scope)])
}

// setMode updates the steering mode.
func (sq *steeringQueue) setMode(mode SteeringMode) {
	sq.mu.Lock()
	defer sq.mu.Unlock()
	sq.mode = mode
}

// getMode returns the current steering mode.
func (sq *steeringQueue) getMode() SteeringMode {
	sq.mu.Lock()
	defer sq.mu.Unlock()
	return sq.mode
}

// Steer enqueues a user message to be injected into the currently running
// agent loop. The message will be picked up after the current tool finishes
// executing, causing any remaining tool calls in the batch to be skipped.
func (al *AgentLoop) Steer(msg providers.Message) error {
	scope := ""
	agentID := ""
	if ts := al.getAnyActiveTurnState(); ts != nil {
		scope = ts.sessionKey
		agentID = ts.agentID
	}
	return al.enqueueSteeringMessage(scope, agentID, msg)
}

func (al *AgentLoop) enqueueSteeringMessage(scope, agentID string, msg providers.Message) error {
	if al.steering == nil {
		return fmt.Errorf("steering queue is not initialized")
	}

	msg = steeringPromptMessage(msg)
	if err := al.steering.pushScope(scope, msg); err != nil {
		logger.WarnCF("agent", "Failed to enqueue steering message", map[string]any{
			"error": err.Error(),
			"role":  msg.Role,
			"scope": normalizeSteeringScope(scope),
		})
		return err
	}

	queueDepth := al.steering.lenScope(scope)
	logger.DebugCF("agent", "Steering message enqueued", map[string]any{
		"role":        msg.Role,
		"content_len": len(msg.Content),
		"media_count": len(msg.Media),
		"queue_len":   queueDepth,
		"scope":       normalizeSteeringScope(scope),
	})

	meta := EventMeta{
		Source:    "Steer",
		TracePath: "turn.interrupt.received",
	}
	if ts := al.getAnyActiveTurnState(); ts != nil {
		meta = ts.eventMeta("Steer", "turn.interrupt.received")
	} else {
		if strings.TrimSpace(agentID) != "" {
			meta.AgentID = agentID
		}
		normalizedScope := normalizeSteeringScope(scope)
		if normalizedScope != manualSteeringScope {
			meta.SessionKey = normalizedScope
		}
		if meta.AgentID == "" {
			if registry := al.GetRegistry(); registry != nil {
				if agent := registry.GetDefaultAgent(); agent != nil {
					meta.AgentID = agent.ID
				}
			}
		}
	}

	al.emitEvent(
		EventKindInterruptReceived,
		meta,
		InterruptReceivedPayload{
			Kind:       InterruptKindSteering,
			Role:       msg.Role,
			ContentLen: len(msg.Content),
			QueueDepth: queueDepth,
		},
	)

	return nil
}

// SteeringMode returns the current steering mode.
func (al *AgentLoop) SteeringMode() SteeringMode {
	if al.steering == nil {
		return SteeringOneAtATime
	}
	return al.steering.getMode()
}

// SetSteeringMode updates the steering mode.
func (al *AgentLoop) SetSteeringMode(mode SteeringMode) {
	if al.steering == nil {
		return
	}
	al.steering.setMode(mode)
}

// dequeueSteeringMessages is the internal method called by the agent loop
// to poll for steering messages in the legacy fallback scope.
func (al *AgentLoop) dequeueSteeringMessages() []providers.Message {
	if al.steering == nil {
		return nil
	}
	return al.steering.dequeue()
}

func (al *AgentLoop) dequeueSteeringMessagesForScope(scope string) []providers.Message {
	if al.steering == nil {
		return nil
	}
	return al.steering.dequeueScope(scope)
}

func (al *AgentLoop) dequeueSteeringMessagesForScopeWithFallback(scope string) []providers.Message {
	if al.steering == nil {
		return nil
	}
	return al.steering.dequeueScopeWithFallback(scope)
}

func (al *AgentLoop) pendingSteeringCountForScope(scope string) int {
	if al.steering == nil {
		return 0
	}
	return al.steering.lenScope(scope)
}

func (al *AgentLoop) continueWithSteeringMessages(
	ctx context.Context,
	agent *AgentInstance,
	sessionKey, channel, chatID string,
	scope *session.SessionScope,
	steeringMsgs []providers.Message,
) (string, error) {
	dispatch := DispatchRequest{
		SessionKey:   sessionKey,
		SessionScope: session.CloneScope(scope),
	}
	if channel != "" || chatID != "" {
		dispatch.InboundContext = &bus.InboundContext{
			Channel:  channel,
			ChatID:   chatID,
			ChatType: inferChatTypeFromSessionScope(scope),
		}
	}
	return al.runAgentLoop(ctx, agent, processOptions{
		Dispatch:                dispatch,
		DefaultResponse:         defaultResponse,
		EnableSummary:           true,
		SendResponse:            false,
		InitialSteeringMessages: steeringMsgs,
		SkipInitialSteeringPoll: true,
	})
}

func (al *AgentLoop) agentForSession(sessionKey string) *AgentInstance {
	registry := al.GetRegistry()
	if registry == nil {
		return nil
	}

	agentIDs := registry.ListAgentIDs()
	sort.Strings(agentIDs)
	for _, agentID := range agentIDs {
		agent, ok := registry.GetAgent(agentID)
		if !ok || agent == nil {
			continue
		}
		resolvedAgentID := session.ResolveAgentID(agent.Sessions, sessionKey)
		if resolvedAgentID == "" {
			continue
		}
		if scopedAgent, ok := registry.GetAgent(resolvedAgentID); ok {
			return scopedAgent
		}
	}

	return registry.GetDefaultAgent()
}

// Continue resumes an idle agent by dequeuing any pending steering messages
// and running them through the agent loop. This is used when the agent's last
// message was from the assistant (i.e., it has stopped processing) and the
// user has since enqueued steering messages.
//
// If no steering messages are pending, it returns an empty string.
func (al *AgentLoop) Continue(ctx context.Context, sessionKey, channel, chatID string) (string, error) {
	// Claim the session with a unique placeholder to prevent a TOCTOU race where two
	// concurrent Continue calls for the same session both pass the active-turn
	// check and create parallel turns. The placeholder is replaced by the real
	// turnState inside continueWithSteeringMessages → runAgentLoop → registerActiveTurn.
	placeholder := &turnState{
		turnID: "pending-continue-" + sessionKey + "-" + fmt.Sprintf("%d", al.turnSeq.Add(1)),
		phase:  TurnPhaseSetup,
	}
	if _, loaded := al.activeTurnStates.LoadOrStore(sessionKey, placeholder); loaded {
		if active := al.GetActiveTurnBySession(sessionKey); active != nil {
			return "", fmt.Errorf("turn %s is still active for session %q", active.TurnID, sessionKey)
		}
		// Another Continue just claimed the slot; let it handle the steering.
		return "", nil
	}

	if err := al.ensureHooksInitialized(ctx); err != nil {
		al.activeTurnStates.Delete(sessionKey)
		return "", err
	}
	if err := al.ensureMCPInitialized(ctx); err != nil {
		logger.WarnCF("agent", "MCP initialization failed, continuing steering without MCP tools",
			map[string]any{"error": err.Error(), "session_key": sessionKey})
	}

	steeringMsgs := al.dequeueSteeringMessagesForScopeWithFallback(sessionKey)
	if len(steeringMsgs) == 0 {
		al.activeTurnStates.Delete(sessionKey)
		return "", nil
	}

	agent := al.agentForSession(sessionKey)
	if agent == nil {
		al.activeTurnStates.Delete(sessionKey)
		return "", fmt.Errorf("no agent available for session %q", sessionKey)
	}

	if tool, ok := agent.Tools.Get("message"); ok {
		if resetter, ok := tool.(interface{ ResetSentInRound(sessionKey string) }); ok {
			resetter.ResetSentInRound(sessionKey)
		}
	}

	var scope *session.SessionScope
	if metaStore, ok := agent.Sessions.(session.MetadataAwareSessionStore); ok {
		scope = metaStore.GetSessionScope(sessionKey)
	}

	return al.continueWithSteeringMessages(ctx, agent, sessionKey, channel, chatID, scope, steeringMsgs)
}

func (al *AgentLoop) InterruptGraceful(hint string) error {
	ts := al.getAnyActiveTurnState()
	if ts == nil {
		return fmt.Errorf("no active turn")
	}
	if !ts.requestGracefulInterrupt(hint) {
		return fmt.Errorf("turn %s cannot accept graceful interrupt", ts.turnID)
	}

	al.emitEvent(
		EventKindInterruptReceived,
		ts.eventMeta("InterruptGraceful", "turn.interrupt.received"),
		InterruptReceivedPayload{
			Kind:    InterruptKindGraceful,
			HintLen: len(hint),
		},
	)

	return nil
}

// InterruptHard aborts an arbitrary active turn. In parallel mode this may
// target the wrong session. Prefer HardAbort(sessionKey) instead.
//
// Deprecated: Use HardAbort(sessionKey) for session-safe aborts.
func (al *AgentLoop) InterruptHard() error {
	ts := al.getAnyActiveTurnState()
	if ts == nil {
		return fmt.Errorf("no active turn")
	}
	if strings.HasPrefix(ts.turnID, "pending-") {
		return fmt.Errorf("turn is still initializing for session %s", ts.sessionKey)
	}
	if !ts.requestHardAbort() {
		return fmt.Errorf("turn %s is already aborting", ts.turnID)
	}

	al.emitEvent(
		EventKindInterruptReceived,
		ts.eventMeta("InterruptHard", "turn.interrupt.received"),
		InterruptReceivedPayload{
			Kind: InterruptKindHard,
		},
	)

	return nil
}

// ====================== SubTurn Result Polling ======================

// dequeuePendingSubTurnResults polls the SubTurn result channel for the given
// session and returns all available results without blocking.
// Returns nil if no active turn state exists for this session.
func (al *AgentLoop) dequeuePendingSubTurnResults(sessionKey string) []*tools.ToolResult {
	tsInterface, ok := al.activeTurnStates.Load(sessionKey)
	if !ok {
		return nil
	}
	ts, ok := tsInterface.(*turnState)
	if !ok {
		return nil
	}

	var results []*tools.ToolResult
	for {
		select {
		case result, ok := <-ts.pendingResults:
			if !ok {
				return results
			}
			if result != nil {
				results = append(results, result)
			}
		default:
			return results
		}
	}
}

// ====================== Hard Abort ======================

// HardAbort immediately cancels the running agent loop for the given session,
// cascading the cancellation to all child SubTurns. This is a destructive operation
// that terminates execution without waiting for graceful cleanup.
//
// Use this when the user explicitly requests immediate termination (e.g., "stop now", "abort").
// For graceful interruption that allows the agent to finish the current tool and summarize,
// use Steer() instead.
func (al *AgentLoop) HardAbort(sessionKey string) error {
	tsInterface, ok := al.activeTurnStates.Load(sessionKey)
	if !ok {
		return fmt.Errorf("no active turn state found for session %s", sessionKey)
	}

	ts, ok := tsInterface.(*turnState)
	if !ok {
		return fmt.Errorf("invalid turn state type for session %s", sessionKey)
	}

	if strings.HasPrefix(ts.turnID, "pending-") {
		return fmt.Errorf("turn is still initializing for session %s", sessionKey)
	}

	logger.InfoCF("agent", "Hard abort triggered", map[string]any{
		"session_key":            sessionKey,
		"turn_id":                ts.turnID,
		"depth":                  ts.depth,
		"initial_history_length": ts.initialHistoryLength,
	})

	// IMPORTANT: Trigger cascading cancellation FIRST to stop all child SubTurns
	// from adding more messages to the session. This prevents race conditions
	// where rollback happens while children are still writing.
	// Use isHardAbort=true for hard abort to immediately cancel all children.
	ts.Finish(true)

	// Roll back session history to the state before the turn started.
	if ts.session != nil {
		history := ts.session.GetHistory(sessionKey)
		if ts.initialHistoryLength < len(history) {
			ts.session.SetHistory(sessionKey, history[:ts.initialHistoryLength])
		}
	}

	return nil
}

// ====================== Follow-Up Injection ======================

// InjectFollowUp enqueues a message to be automatically processed after the current
// turn completes. Unlike Steer(), which interrupts the current execution, InjectFollowUp
// waits for the current turn to finish naturally before processing the message.
//
// This is useful for:
// - Automated workflows that need to chain multiple turns
// - Background tasks that should run after the main task completes
// - Scheduled follow-up actions
//
// The message will be processed via Continue() when the agent becomes idle.
func (al *AgentLoop) InjectFollowUp(msg providers.Message) error {
	// InjectFollowUp uses the same steering queue mechanism as Steer(),
	// but the semantic difference is in when it's called:
	// - Steer() is called during active execution to interrupt
	// - InjectFollowUp() is called when planning future work
	//
	// Both end up in the same queue and are processed by Continue()
	// when the agent is idle.
	return al.Steer(msg)
}

// ====================== API Aliases for Design Document Compatibility ======================

// InjectSteering is an alias for Steer() to match the design document naming.
// It injects a steering message into the currently running agent loop.
func (al *AgentLoop) InjectSteering(msg providers.Message) error {
	return al.Steer(msg)
}
