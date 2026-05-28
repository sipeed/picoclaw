package gateway

import (
	"time"

	"github.com/sipeed/picoclaw/pkg/agent"
	runtimeevents "github.com/sipeed/picoclaw/pkg/events"
)

type gatewayEventPayload struct {
	DurationMS int64  `json:"duration_ms,omitempty"`
	Error      string `json:"error,omitempty"`
}

func publishGatewayEvent(
	al *agent.AgentLoop,
	kind runtimeevents.Kind,
	startedAt time.Time,
	err error,
) {
	if al == nil || al.RuntimeEventBus() == nil {
		return
	}

	severity := runtimeevents.SeverityInfo
	payload := gatewayEventPayload{}
	if !startedAt.IsZero() {
		payload.DurationMS = time.Since(startedAt).Milliseconds()
	}
	if err != nil {
		severity = runtimeevents.SeverityError
		payload.Error = err.Error()
	}

	al.RuntimeEventBus().PublishNonBlocking(runtimeevents.Event{
		Kind:     kind,
		Source:   runtimeevents.Source{Component: "gateway"},
		Severity: severity,
		Payload:  payload,
		Attrs:    gatewayEventAttrs(payload),
	})
}

func gatewayEventAttrs(payload gatewayEventPayload) map[string]any {
	attrs := map[string]any{}
	if payload.DurationMS > 0 {
		attrs["duration_ms"] = payload.DurationMS
	}
	if payload.Error != "" {
		attrs["error"] = payload.Error
	}
	return attrs
}
