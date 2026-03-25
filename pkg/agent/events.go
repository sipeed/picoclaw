package agent

import (
	"fmt"
	"time"
)

// EventKind identifies a structured agent-loop event.
type EventKind uint8

const (
	// EventKindTurnStart is emitted when a turn begins processing.
	EventKindTurnStart EventKind = iota
	// EventKindTurnEnd is emitted when a turn finishes, successfully or with an error.
	EventKindTurnEnd
	// EventKindLLMRequest is emitted before a provider chat request is made.
	EventKindLLMRequest
	// EventKindLLMDelta is emitted when a streaming provider yields a partial delta.
	EventKindLLMDelta
	// EventKindLLMResponse is emitted after a provider chat response is received.
	EventKindLLMResponse
	// EventKindLLMRetry is emitted when an LLM request is retried.
	EventKindLLMRetry
	// EventKindContextCompress is emitted when session history is forcibly compressed.
	EventKindContextCompress
	// EventKindSessionSummarize is emitted when asynchronous summarization completes.
	EventKindSessionSummarize
	// EventKindToolExecStart is emitted immediately before a tool executes.
	EventKindToolExecStart
	// EventKindToolExecEnd is emitted immediately after a tool finishes executing.
	EventKindToolExecEnd
	// EventKindToolExecSkipped is emitted when a queued tool call is skipped.
	EventKindToolExecSkipped
	// EventKindSteeringInjected is emitted when queued steering is injected into context.
	EventKindSteeringInjected
	// EventKindFollowUpQueued is emitted when an async tool queues a follow-up system message.
	EventKindFollowUpQueued
	// EventKindInterruptReceived is emitted when a soft interrupt message is accepted.
	EventKindInterruptReceived
	// EventKindSubTurnSpawn is emitted when a sub-turn is spawned.
	EventKindSubTurnSpawn
	// EventKindSubTurnEnd is emitted when a sub-turn finishes.
	EventKindSubTurnEnd
	// EventKindSubTurnResultDelivered is emitted when a sub-turn result is delivered.
	EventKindSubTurnResultDelivered
	// EventKindSubTurnOrphan is emitted when a sub-turn result cannot be delivered.
	EventKindSubTurnOrphan
	// EventKindError is emitted when a turn encounters an execution error.
	EventKindError

	eventKindCount
)

var eventKindNames = [...]string{
	"turn_start",
	"turn_end",
	"llm_request",
	"llm_delta",
	"llm_response",
	"llm_retry",
	"context_compress",
	"session_summarize",
	"tool_exec_start",
	"tool_exec_end",
	"tool_exec_skipped",
	"steering_injected",
	"follow_up_queued",
	"interrupt_received",
	"subturn_spawn",
	"subturn_end",
	"subturn_result_delivered",
	"subturn_orphan",
	"error",
}

// String returns the stable string form of an EventKind.
func (k EventKind) String() string {
	if k >= eventKindCount {
		return fmt.Sprintf("event_kind(%d)", k)
	}
	return eventKindNames[k]
}

// Event is the structured envelope broadcast by the agent EventBus.
type Event struct {
	Kind    EventKind
	Time    time.Time
	Meta    EventMeta
	Payload any
}

// EventMeta contains correlation fields shared by all agent-loop events.
type EventMeta struct {
	AgentID      string
	TurnID       string
	ParentTurnID string
	SessionKey   string
	Iteration    int
	TracePath    string
	Source       string
}

// TurnEndStatus describes the terminal state of a turn.
type TurnEndStatus string

const (
	// TurnEndStatusCompleted indicates the turn finished normally.
	TurnEndStatusCompleted TurnEndStatus = "completed"
	// TurnEndStatusError indicates the turn ended because of an error.
	TurnEndStatusError TurnEndStatus = "error"
	// TurnEndStatusAborted indicates the turn was hard-aborted and rolled back.
	TurnEndStatusAborted TurnEndStatus = "aborted"
)

// TurnStartPayload describes the start of a turn.
type TurnStartPayload struct {
	Channel     string
	ChatID      string
	UserMessage string
	MediaCount  int
}

// TurnEndPayload describes the completion of a turn.
type TurnEndPayload struct {
	Status          TurnEndStatus
	Iterations      int
	Duration        time.Duration
	FinalContentLen int
}

// LLMRequestPayload describes an outbound LLM request.
type LLMRequestPayload struct {
	Model         string
	MessagesCount int
	ToolsCount    int
	MaxTokens     int
	Temperature   float64
}

// LLMResponsePayload describes an inbound LLM response.
type LLMResponsePayload struct {
	ContentLen   int
	ToolCalls    int
	HasReasoning bool
}

// LLMDeltaPayload describes a streamed LLM delta.
type LLMDeltaPayload struct {
	ContentDeltaLen   int
	ReasoningDeltaLen int
}

// LLMRetryPayload describes a retry of an LLM request.
type LLMRetryPayload struct {
	Attempt    int
	MaxRetries int
	Reason     string
	Error      string
	Backoff    time.Duration
}

// ContextCompressReason identifies why emergency compression ran.
type ContextCompressReason string

const (
	// ContextCompressReasonProactive indicates compression before the first LLM call.
	ContextCompressReasonProactive ContextCompressReason = "proactive_budget"
	// ContextCompressReasonRetry indicates compression during context-error retry handling.
	ContextCompressReasonRetry ContextCompressReason = "llm_retry"
)

// ContextCompressPayload describes a forced history compression.
type ContextCompressPayload struct {
	Reason            ContextCompressReason
	DroppedMessages   int
	RemainingMessages int
}

// SessionSummarizePayload describes a completed async session summarization.
type SessionSummarizePayload struct {
	SummarizedMessages int
	KeptMessages       int
	SummaryLen         int
	OmittedOversized   bool
}

// ToolExecStartPayload describes a tool execution request.
type ToolExecStartPayload struct {
	Tool      string
	Arguments map[string]any
}

// ToolExecEndPayload describes the outcome of a tool execution.
type ToolExecEndPayload struct {
	Tool       string
	Duration   time.Duration
	ForLLMLen  int
	ForUserLen int
	IsError    bool
	Async      bool
}

// ToolExecSkippedPayload describes a skipped tool call.
type ToolExecSkippedPayload struct {
	Tool   string
	Reason string
}

// SteeringInjectedPayload describes steering messages appended before the next LLM call.
type SteeringInjectedPayload struct {
	Count           int
	TotalContentLen int
}

// FollowUpQueuedPayload describes an async follow-up queued back into the inbound bus.
type FollowUpQueuedPayload struct {
	SourceTool string
	Channel    string
	ChatID     string
	ContentLen int
}

type InterruptKind string

const (
	InterruptKindSteering InterruptKind = "steering"
	InterruptKindGraceful InterruptKind = "graceful"
	InterruptKindHard     InterruptKind = "hard_abort"
)

// InterruptReceivedPayload describes accepted turn-control input.
type InterruptReceivedPayload struct {
	Kind       InterruptKind
	Role       string
	ContentLen int
	QueueDepth int
	HintLen    int
}

// SubTurnSpawnPayload describes the creation of a child turn.
type SubTurnSpawnPayload struct {
	AgentID      string
	Label        string
	ParentTurnID string
}

// SubTurnEndPayload describes the completion of a child turn.
type SubTurnEndPayload struct {
	AgentID string
	Status  string
}

// SubTurnResultDeliveredPayload describes delivery of a sub-turn result.
type SubTurnResultDeliveredPayload struct {
	TargetChannel string
	TargetChatID  string
	ContentLen    int
}

// SubTurnOrphanPayload describes a sub-turn result that could not be delivered.
type SubTurnOrphanPayload struct {
	ParentTurnID string
	ChildTurnID  string
	Reason       string
}

// ErrorPayload describes an execution error inside the agent loop.
type ErrorPayload struct {
	Stage   string
	Message string
}
