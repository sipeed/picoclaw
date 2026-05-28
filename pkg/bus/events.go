package bus

import (
	runtimeevents "github.com/sipeed/picoclaw/pkg/events"
)

type busPublishFailedPayload struct {
	Stream string `json:"stream"`
	Error  string `json:"error"`
}

type busClosePayload struct {
	Drained int `json:"drained,omitempty"`
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
