package agent

import (
	"time"

	runtimeevents "github.com/sipeed/picoclaw/pkg/events"
)

// HookMeta contains correlation fields shared by agent hook requests and
// runtime events emitted from turn processing.
type HookMeta struct {
	AgentID      string
	TurnID       string
	ParentTurnID string
	SessionKey   string
	Iteration    int
	TracePath    string
	Source       string
	turnContext  *TurnContext
}

// EventKind is the legacy in-agent event kind alias kept for tests and
// compatibility shims on top of the runtime event bus.
type EventKind = runtimeevents.Kind

const (
	EventKindTurnStart              EventKind = runtimeevents.KindAgentTurnStart
	EventKindTurnEnd                EventKind = runtimeevents.KindAgentTurnEnd
	EventKindLLMRequest             EventKind = runtimeevents.KindAgentLLMRequest
	EventKindLLMDelta               EventKind = runtimeevents.KindAgentLLMDelta
	EventKindLLMResponse            EventKind = runtimeevents.KindAgentLLMResponse
	EventKindLLMRetry               EventKind = runtimeevents.KindAgentLLMRetry
	EventKindContextCompress        EventKind = runtimeevents.KindAgentContextCompress
	EventKindSessionSummarize       EventKind = runtimeevents.KindAgentSessionSummarize
	EventKindToolExecStart          EventKind = runtimeevents.KindAgentToolExecStart
	EventKindToolExecEnd            EventKind = runtimeevents.KindAgentToolExecEnd
	EventKindToolExecSkipped        EventKind = runtimeevents.KindAgentToolExecSkipped
	EventKindSteeringInjected       EventKind = runtimeevents.KindAgentSteeringInjected
	EventKindFollowUpQueued         EventKind = runtimeevents.KindAgentFollowUpQueued
	EventKindInterruptReceived      EventKind = runtimeevents.KindAgentInterruptReceived
	EventKindSubTurnSpawn           EventKind = runtimeevents.KindAgentSubTurnSpawn
	EventKindSubTurnEnd             EventKind = runtimeevents.KindAgentSubTurnEnd
	EventKindSubTurnResultDelivered EventKind = runtimeevents.KindAgentSubTurnResultDelivered
	EventKindSubTurnOrphan          EventKind = runtimeevents.KindAgentSubTurnOrphan
	EventKindError                  EventKind = runtimeevents.KindAgentError
)

// EventMeta is the legacy name for hook metadata.
type EventMeta = HookMeta

// Event is the legacy agent event envelope exposed by SubscribeEvents and a
// handful of tests. Runtime code publishes pkg/events.Event internally.
type Event struct {
	Kind    EventKind
	Time    time.Time
	Meta    EventMeta
	Context *TurnContext
	Payload any
}
