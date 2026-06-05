package agent

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/session"
)

func TestSessionNeedsUnansweredRecovery(t *testing.T) {
	tests := []struct {
		name    string
		history []providers.Message
		want    bool
	}{
		{
			name:    "empty",
			history: nil,
			want:    false,
		},
		{
			name: "assistant answered",
			history: []providers.Message{
				{Role: "user", Content: "hello"},
				{Role: "assistant", Content: "hi"},
			},
			want: false,
		},
		{
			name: "last user text",
			history: []providers.Message{
				{Role: "assistant", Content: "previous"},
				{Role: "user", Content: "follow up"},
			},
			want: true,
		},
		{
			name: "last user media",
			history: []providers.Message{
				{Role: "user", Media: []string{"media://image"}},
			},
			want: true,
		},
		{
			name: "empty user ignored",
			history: []providers.Message{
				{Role: "user"},
			},
			want: false,
		},
		{
			name: "observed passive message ignored",
			history: []providers.Message{
				{
					Role: "user",
					Content: "[observed group message from Anton; no reply requested; reason: " +
						"reply to a non-bot message without mentioning this bot]\n" +
						"this was not for the bot",
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sessionNeedsUnansweredRecovery(tt.history); got != tt.want {
				t.Fatalf("sessionNeedsUnansweredRecovery() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestInboundContextFromSessionScope(t *testing.T) {
	scope := &session.SessionScope{
		Version:    session.ScopeVersionV1,
		AgentID:    "main",
		Channel:    "telegram",
		Account:    "bot-a",
		Dimensions: []string{"chat"},
		Values: map[string]string{
			"chat": "group:-100123/42",
		},
	}
	inbound, ok := inboundContextFromSessionScope(scope)
	if !ok {
		t.Fatal("inboundContextFromSessionScope returned ok=false")
	}
	if inbound.Channel != "telegram" ||
		inbound.Account != "bot-a" ||
		inbound.ChatID != "-100123" ||
		inbound.ChatType != "group" ||
		inbound.TopicID != "42" {
		t.Fatalf("inbound = %+v, want telegram bot-a group -100123 topic 42", inbound)
	}
}

func TestAgentLoopRecoverUnansweredSessions(t *testing.T) {
	workspace := t.TempDir()
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         workspace,
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}
	msgBus := bus.NewMessageBus()
	defer msgBus.Close()
	al := NewAgentLoop(cfg, msgBus, &simpleMockProvider{response: "recovered answer"})
	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		t.Fatal("expected default agent")
	}

	sessionKey := "agent:main:telegram:group:-100123/42"
	metaStore, ok := agent.Sessions.(session.MetadataAwareSessionStore)
	if !ok {
		t.Fatal("expected metadata-aware session store")
	}
	metaStore.EnsureSessionMetadata(sessionKey, &session.SessionScope{
		Version:    session.ScopeVersionV1,
		AgentID:    "main",
		Channel:    "telegram",
		Dimensions: []string{"chat"},
		Values: map[string]string{
			"chat": "group:-100123/42",
		},
	}, nil)
	agent.Sessions.AddFullMessage(sessionKey, providers.Message{Role: "user", Content: "please continue"})

	if got := al.RecoverUnansweredSessions(context.Background()); got != 1 {
		t.Fatalf("RecoverUnansweredSessions() = %d, want 1", got)
	}

	history := agent.Sessions.GetHistory(sessionKey)
	if len(history) != 2 {
		t.Fatalf("history len = %d, want 2: %#v", len(history), history)
	}
	if history[0].Role != "user" || history[0].Content != "please continue" {
		t.Fatalf("history[0] = %+v, want original user", history[0])
	}
	if history[1].Role != "assistant" || history[1].Content != "recovered answer" {
		t.Fatalf("history[1] = %+v, want recovered assistant", history[1])
	}

	select {
	case outbound := <-msgBus.OutboundChan():
		if outbound.Channel != "telegram" || outbound.ChatID != "-100123" || outbound.Context.TopicID != "42" {
			t.Fatalf("outbound route = %+v, want telegram -100123 topic 42", outbound)
		}
		if outbound.Content != "recovered answer" {
			t.Fatalf("outbound content = %q, want recovered answer", outbound.Content)
		}
	default:
		t.Fatal("expected outbound recovery response")
	}
}

func TestAgentLoopRecoverUnansweredSessionsSkipsUnackedInboundSpool(t *testing.T) {
	workspace := t.TempDir()
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         workspace,
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}
	msgBus := bus.NewMessageBus()
	defer msgBus.Close()
	spool, err := bus.NewInboundSpool(filepath.Join(workspace, "spool"))
	if err != nil {
		t.Fatalf("NewInboundSpool() error = %v", err)
	}
	msgBus.SetInboundSpool(spool)
	al := NewAgentLoop(cfg, msgBus, &simpleMockProvider{response: "should not run"})
	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		t.Fatal("expected default agent")
	}

	sessionKey := "agent:main:telegram:group:-100123/42"
	metaStore, ok := agent.Sessions.(session.MetadataAwareSessionStore)
	if !ok {
		t.Fatal("expected metadata-aware session store")
	}
	metaStore.EnsureSessionMetadata(sessionKey, &session.SessionScope{
		Version:    session.ScopeVersionV1,
		AgentID:    "main",
		Channel:    "telegram",
		Dimensions: []string{"chat"},
		Values: map[string]string{
			"chat": "group:-100123/42",
		},
	}, nil)
	agent.Sessions.AddFullMessage(sessionKey, providers.Message{Role: "user", Content: "please continue"})

	if err := msgBus.PublishInbound(context.Background(), bus.InboundMessage{
		Context: bus.InboundContext{
			Channel:  "telegram",
			ChatID:   "-100123",
			ChatType: "group",
			TopicID:  "42",
			SenderID: "user1",
		},
		Content:    "please continue",
		SessionKey: sessionKey,
	}); err != nil {
		t.Fatalf("PublishInbound() error = %v", err)
	}

	if got := al.RecoverUnansweredSessions(context.Background()); got != 0 {
		t.Fatalf("RecoverUnansweredSessions() = %d, want 0", got)
	}
	history := agent.Sessions.GetHistory(sessionKey)
	if len(history) != 1 {
		t.Fatalf("history len = %d, want 1: %#v", len(history), history)
	}
	select {
	case outbound := <-msgBus.OutboundChan():
		t.Fatalf("unexpected outbound recovery response: %+v", outbound)
	default:
	}
}

func TestClearRecoveryPlaceholderDoesNotDeleteNewerTurn(t *testing.T) {
	al := &AgentLoop{}
	sessionKey := "agent:main:telegram:private:123"
	placeholder := &turnState{turnID: "pending-recovery"}
	newer := &turnState{turnID: "newer-turn"}

	al.activeTurnStates.Store(sessionKey, newer)
	al.clearRecoveryPlaceholder(sessionKey, placeholder)
	if current, ok := al.activeTurnStates.Load(sessionKey); !ok || current != newer {
		t.Fatalf("active turn = %v, %t; want newer turn preserved", current, ok)
	}

	al.activeTurnStates.Store(sessionKey, placeholder)
	al.clearRecoveryPlaceholder(sessionKey, placeholder)
	if current, ok := al.activeTurnStates.Load(sessionKey); ok {
		t.Fatalf("active turn = %v, want cleared placeholder", current)
	}
}
