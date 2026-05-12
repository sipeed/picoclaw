package agent

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/sipeed/picoclaw/pkg/bus"
	runtimeevents "github.com/sipeed/picoclaw/pkg/events"
)

const defaultEventSubscriberBuffer = 16

// EventSubscription identifies a legacy subscriber channel returned by
// AgentLoop.SubscribeEvents.
type EventSubscription struct {
	ID uint64
	C  <-chan Event
}

type legacyEventSubscription struct {
	cancel context.CancelFunc
	sub    runtimeevents.Subscription
}

var (
	legacyEventSubSeq  atomic.Uint64
	legacyEventSubLock sync.Map
)

// SubscribeEvents exposes the previous in-agent event subscription API on top
// of the runtime event bus for tests and compatibility.
func (al *AgentLoop) SubscribeEvents(buffer int) EventSubscription {
	if buffer <= 0 {
		buffer = defaultEventSubscriberBuffer
	}

	out := make(chan Event, buffer)
	if al == nil || al.runtimeEvents == nil {
		close(out)
		return EventSubscription{C: out}
	}

	ctx, cancel := context.WithCancel(context.Background())
	sub, in, err := al.runtimeEvents.Channel().
		Source("agent").
		OfKind(legacyAgentEventKinds()...).
		SubscribeChan(ctx, runtimeevents.SubscribeOptions{
			Name:   "legacy-agent-events",
			Buffer: buffer,
		})
	if err != nil {
		cancel()
		close(out)
		return EventSubscription{C: out}
	}

	id := legacyEventSubSeq.Add(1)
	legacyEventSubLock.Store(id, legacyEventSubscription{cancel: cancel, sub: sub})
	go func() {
		defer legacyEventSubLock.LoadAndDelete(id)
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case evt, ok := <-in:
				if !ok {
					return
				}
				select {
				case out <- legacyEventFromRuntimeEvent(evt):
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return EventSubscription{ID: id, C: out}
}

func (al *AgentLoop) UnsubscribeEvents(id uint64) {
	if id == 0 {
		return
	}
	value, ok := legacyEventSubLock.LoadAndDelete(id)
	if !ok {
		return
	}
	sub := value.(legacyEventSubscription)
	sub.cancel()
	if sub.sub != nil {
		_ = sub.sub.Close()
	}
}

func legacyEventFromRuntimeEvent(evt runtimeevents.Event) Event {
	meta := hookMetaFromRuntimeEvent(evt)
	return Event{
		Kind:    evt.Kind,
		Time:    evt.Time,
		Meta:    meta,
		Context: turnContextFromRuntimeScope(evt.Scope),
		Payload: evt.Payload,
	}
}

func hookMetaFromRuntimeEvent(evt runtimeevents.Event) HookMeta {
	meta := HookMeta{
		AgentID:      evt.Scope.AgentID,
		TurnID:       evt.Scope.TurnID,
		ParentTurnID: evt.Correlation.ParentTurnID,
		SessionKey:   evt.Scope.SessionKey,
		TracePath:    evt.Correlation.TraceID,
	}
	if evt.Attrs != nil {
		if source, ok := evt.Attrs["agent_source"].(string); ok {
			meta.Source = source
		}
		if iteration, ok := evt.Attrs["iteration"].(int); ok {
			meta.Iteration = iteration
		}
	}
	return meta
}

func turnContextFromRuntimeScope(scope runtimeevents.Scope) *TurnContext {
	if scope.Channel == "" &&
		scope.Account == "" &&
		scope.ChatID == "" &&
		scope.ChatType == "" &&
		scope.TopicID == "" &&
		scope.SpaceID == "" &&
		scope.SpaceType == "" &&
		scope.SenderID == "" &&
		scope.MessageID == "" {
		return nil
	}
	return &TurnContext{
		Inbound: &bus.InboundContext{
			Channel:   scope.Channel,
			Account:   scope.Account,
			ChatID:    scope.ChatID,
			ChatType:  scope.ChatType,
			TopicID:   scope.TopicID,
			SpaceID:   scope.SpaceID,
			SpaceType: scope.SpaceType,
			SenderID:  scope.SenderID,
			MessageID: scope.MessageID,
		},
	}
}

func legacyAgentEventKinds() []runtimeevents.Kind {
	return []runtimeevents.Kind{
		EventKindTurnStart,
		EventKindTurnEnd,
		EventKindLLMRequest,
		EventKindLLMDelta,
		EventKindLLMResponse,
		EventKindLLMRetry,
		EventKindContextCompress,
		EventKindSessionSummarize,
		EventKindToolExecStart,
		EventKindToolExecEnd,
		EventKindToolExecSkipped,
		EventKindSteeringInjected,
		EventKindFollowUpQueued,
		EventKindInterruptReceived,
		EventKindSubTurnSpawn,
		EventKindSubTurnEnd,
		EventKindSubTurnResultDelivered,
		EventKindSubTurnOrphan,
		EventKindError,
	}
}
