package bus

import (
	"time"

	runtimeevents "github.com/sipeed/picoclaw/pkg/events"
)

type busPublishFailedPayload struct {
	Stream string `json:"stream"`
	Error  string `json:"error"`
}

type busClosePayload struct {
	Drained int `json:"drained,omitempty"`
}

type busMessageDroppedPayload struct {
	Stream       string `json:"stream"`
	Reason       string `json:"reason"`
	WaitMS       int64  `json:"wait_ms"`
	QueueDepth   int    `json:"queue_depth"`
	QueueCap     int    `json:"queue_capacity"`
	DroppedTotal uint64 `json:"dropped_total"`
}

func (mb *MessageBus) publishFailure(stream string, scope runtimeevents.Scope, err error) {
	if mb == nil || err == nil {
		return
	}
	publisher, ok := mb.eventPublisher.Load().(EventPublisher)
	if !ok || publisher == nil {
		return
	}

	publisher.PublishNonBlocking(runtimeevents.Event{
		Kind:     runtimeevents.KindBusPublishFailed,
		Source:   runtimeevents.Source{Component: "bus", Name: stream},
		Scope:    scope,
		Severity: runtimeevents.SeverityError,
		Payload: busPublishFailedPayload{
			Stream: stream,
			Error:  err.Error(),
		},
		Attrs: map[string]any{
			"stream": stream,
			"error":  err.Error(),
		},
	})
}

func (mb *MessageBus) publishDrop(
	stream string,
	scope runtimeevents.Scope,
	reason string,
	wait time.Duration,
	queueDepth, queueCap int,
	droppedTotal uint64,
) {
	if mb == nil {
		return
	}
	publisher, ok := mb.eventPublisher.Load().(EventPublisher)
	if !ok || publisher == nil {
		return
	}

	publisher.PublishNonBlocking(runtimeevents.Event{
		Kind:     runtimeevents.KindBusMessageDropped,
		Source:   runtimeevents.Source{Component: "bus", Name: stream},
		Scope:    scope,
		Severity: runtimeevents.SeverityWarn,
		Payload: busMessageDroppedPayload{
			Stream:       stream,
			Reason:       reason,
			WaitMS:       wait.Milliseconds(),
			QueueDepth:   queueDepth,
			QueueCap:     queueCap,
			DroppedTotal: droppedTotal,
		},
		Attrs: map[string]any{
			"stream":         stream,
			"reason":         reason,
			"wait_ms":        wait.Milliseconds(),
			"queue_depth":    queueDepth,
			"queue_capacity": queueCap,
			"dropped_total":  droppedTotal,
		},
	})
}

func (mb *MessageBus) publishCloseEvent(kind runtimeevents.Kind, drained int) {
	if mb == nil {
		return
	}
	publisher, ok := mb.eventPublisher.Load().(EventPublisher)
	if !ok || publisher == nil {
		return
	}

	attrs := map[string]any{}
	if drained > 0 {
		attrs["drained"] = drained
	}
	publisher.PublishNonBlocking(runtimeevents.Event{
		Kind:     kind,
		Source:   runtimeevents.Source{Component: "bus"},
		Severity: runtimeevents.SeverityInfo,
		Payload:  busClosePayload{Drained: drained},
		Attrs:    attrs,
	})
}

func runtimeScopeFromInboundContext(ctx InboundContext) runtimeevents.Scope {
	return runtimeevents.Scope{
		Channel:   ctx.Channel,
		Account:   ctx.Account,
		ChatID:    ctx.ChatID,
		TopicID:   ctx.TopicID,
		SpaceID:   ctx.SpaceID,
		SpaceType: ctx.SpaceType,
		ChatType:  ctx.ChatType,
		SenderID:  ctx.SenderID,
		MessageID: ctx.MessageID,
	}
}

func runtimeScopeFromAudioChunk(chunk AudioChunk) runtimeevents.Scope {
	return runtimeevents.Scope{
		Channel: chunk.Channel,
		ChatID:  chunk.ChatID,
	}
}

func runtimeScopeFromVoiceControl(ctrl VoiceControl) runtimeevents.Scope {
	return runtimeevents.Scope{
		ChatID: ctrl.ChatID,
	}
}
