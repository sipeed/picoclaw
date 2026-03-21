package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
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
	mu    sync.Mutex
	queue []providers.Message
	mode  SteeringMode
}

func newSteeringQueue(mode SteeringMode) *steeringQueue {
	return &steeringQueue{
		mode: mode,
	}
}

// push enqueues a steering message.
func (sq *steeringQueue) push(msg providers.Message) error {
	sq.mu.Lock()
	defer sq.mu.Unlock()
	if len(sq.queue) >= MaxQueueSize {
		return fmt.Errorf("steering queue is full")
	}
	sq.queue = append(sq.queue, msg)
	return nil
}

// dequeue removes and returns pending steering messages according to the
// configured mode. Returns nil when the queue is empty.
func (sq *steeringQueue) dequeue() []providers.Message {
	sq.mu.Lock()
	defer sq.mu.Unlock()

	if len(sq.queue) == 0 {
		return nil
	}

	switch sq.mode {
	case SteeringAll:
		msgs := sq.queue
		sq.queue = nil
		return msgs
	default: // one-at-a-time
		msg := sq.queue[0]
		sq.queue[0] = providers.Message{} // Clear reference for GC
		sq.queue = sq.queue[1:]
		return []providers.Message{msg}
	}
}

// len returns the number of queued messages.
func (sq *steeringQueue) len() int {
	sq.mu.Lock()
	defer sq.mu.Unlock()
	return len(sq.queue)
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

// --- AgentLoop steering API ---

// Steer enqueues a user message to be injected into the currently running
// agent loop. The message will be picked up after the current tool finishes
// executing, causing any remaining tool calls in the batch to be skipped.
func (al *AgentLoop) Steer(msg providers.Message) error {
	if al.steering == nil {
		return fmt.Errorf("steering queue is not initialized")
	}
	if err := al.steering.push(msg); err != nil {
		logger.WarnCF("agent", "Failed to enqueue steering message", map[string]any{
			"error": err.Error(),
			"role":  msg.Role,
		})
		return err
	}
	logger.DebugCF("agent", "Steering message enqueued", map[string]any{
		"role":        msg.Role,
		"content_len": len(msg.Content),
		"queue_len":   al.steering.len(),
	})

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
// to poll for steering messages. Returns nil when no messages are pending.
func (al *AgentLoop) dequeueSteeringMessages() []providers.Message {
	if al.steering == nil {
		return nil
	}
	return al.steering.dequeue()
}

// Continue resumes an idle agent by dequeuing any pending steering messages
// and running them through the agent loop. This is used when the agent's last
// message was from the assistant (i.e., it has stopped processing) and the
// user has since enqueued steering messages.
//
// If no steering messages are pending, it returns an empty string.
func (al *AgentLoop) Continue(ctx context.Context, sessionKey, channel, chatID string) (string, error) {
	steeringMsgs := al.dequeueSteeringMessages()
	if len(steeringMsgs) == 0 {
		return "", nil
	}

	agent := al.GetRegistry().GetDefaultAgent()
	if agent == nil {
		return "", fmt.Errorf("no default agent available")
	}

	// Build a combined user message from the steering messages.
	var contents []string
	for _, msg := range steeringMsgs {
		contents = append(contents, msg.Content)
	}
	combinedContent := strings.Join(contents, "\n")

	return al.runAgentLoop(ctx, agent, processOptions{
		SessionKey:              sessionKey,
		Channel:                 channel,
		ChatID:                  chatID,
		UserMessage:             combinedContent,
		DefaultResponse:         defaultResponse,
		EnableSummary:           true,
		SendResponse:            false,
		SkipInitialSteeringPoll: true,
	})
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
		case result := <-ts.pendingResults:
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

	// Rollback session history to the state before this turn started.
	// This must happen AFTER Finish() to ensure no child turns are still writing.
	if ts.session != nil {
		currentHistory := ts.session.GetHistory("")
		if len(currentHistory) > ts.initialHistoryLength {
			logger.InfoCF("agent", "Rolling back session history", map[string]any{
				"from": len(currentHistory),
				"to":   ts.initialHistoryLength,
			})
			// SetHistory with the truncated slice to rollback
			ts.session.SetHistory("", currentHistory[:ts.initialHistoryLength])
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

// InterruptGraceful is an alias for Steer() to match the design document naming.
// It gracefully interrupts the current execution by injecting a user message
// that will be processed after the current tool finishes.
func (al *AgentLoop) InterruptGraceful(msg providers.Message) error {
	return al.Steer(msg)
}

// InterruptHard is an alias for HardAbort() to match the design document naming.
// It immediately terminates execution and rolls back the session state.
func (al *AgentLoop) InterruptHard(sessionKey string) error {
	return al.HardAbort(sessionKey)
}

// InjectSteering is an alias for Steer() to match the design document naming.
// It injects a steering message into the currently running agent loop.
func (al *AgentLoop) InjectSteering(msg providers.Message) error {
	return al.Steer(msg)
}
