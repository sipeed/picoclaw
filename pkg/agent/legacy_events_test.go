package agent

import (
	"context"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	runtimeevents "github.com/sipeed/picoclaw/pkg/events"
)

func TestSubscribeEventsFiltersRuntimeBusToLegacyAgentEvents(t *testing.T) {
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         t.TempDir(),
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 3,
			},
		},
	}
	al := NewAgentLoop(cfg, bus.NewMessageBus(), &simpleMockProvider{response: "ok"})
	defer al.Close()

	sub := al.SubscribeEvents(4)
	defer al.UnsubscribeEvents(sub.ID)

	al.RuntimeEventBus().Publish(context.Background(), runtimeevents.Event{
		Kind:   runtimeevents.KindGatewayReady,
		Source: runtimeevents.Source{Component: "gateway"},
	})
	select {
	case evt := <-sub.C:
		t.Fatalf("legacy subscriber received non-agent runtime event: %s", evt.Kind)
	case <-time.After(50 * time.Millisecond):
	}

	al.RuntimeEventBus().Publish(context.Background(), runtimeevents.Event{
		Kind:   runtimeevents.KindAgentTurnStart,
		Source: runtimeevents.Source{Component: "agent", Name: "main"},
		Scope: runtimeevents.Scope{
			AgentID:    "main",
			TurnID:     "turn-1",
			SessionKey: "session-1",
			Channel:    "telegram",
			Account:    "bot-1",
			ChatID:     "chat-1",
			ChatType:   "private",
			TopicID:    "topic-1",
			SpaceID:    "space-1",
			SpaceType:  "dm",
			SenderID:   "sender-1",
			MessageID:  "message-1",
		},
		Payload: TurnStartPayload{UserMessage: "hello"},
	})

	evt := waitForEvent(t, sub.C, 2*time.Second, nil)
	if evt.Kind != EventKindTurnStart {
		t.Fatalf("event kind = %q, want %q", evt.Kind, EventKindTurnStart)
	}
	if evt.Context == nil || evt.Context.Inbound == nil {
		t.Fatalf("expected legacy event inbound context, got %#v", evt.Context)
	}
	if got := evt.Context.Inbound.Channel; got != "telegram" {
		t.Fatalf("inbound channel = %q, want telegram", got)
	}
	if got := evt.Context.Inbound.ChatID; got != "chat-1" {
		t.Fatalf("inbound chat_id = %q, want chat-1", got)
	}
	if got := evt.Context.Inbound.MessageID; got != "message-1" {
		t.Fatalf("inbound message_id = %q, want message-1", got)
	}
}
