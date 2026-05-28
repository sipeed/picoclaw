package agent

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/routing"
	"github.com/sipeed/picoclaw/pkg/session"
)

func TestNormalizeProcessOptions_PopulatesDispatchFromLegacyFields(t *testing.T) {
	opts := normalizeProcessOptions(processOptions{
		SessionKey:       "session-1",
		SessionAliases:   []string{"legacy:one"},
		Channel:          "telegram",
		ChatID:           "chat-1",
		MessageID:        "msg-1",
		ReplyToMessageID: "reply-1",
		SenderID:         "user-1",
		UserMessage:      "hello",
		Media:            []string{"media://one"},
	})

	if opts.Dispatch.SessionKey != "session-1" {
		t.Fatalf("Dispatch.SessionKey = %q, want session-1", opts.Dispatch.SessionKey)
	}
	if len(opts.Dispatch.SessionAliases) != 1 || opts.Dispatch.SessionAliases[0] != "legacy:one" {
		t.Fatalf("Dispatch.SessionAliases = %v, want [legacy:one]", opts.Dispatch.SessionAliases)
	}
	if opts.Dispatch.Channel() != "telegram" || opts.Dispatch.ChatID() != "chat-1" {
		t.Fatalf(
			"dispatch addressing = (%q,%q), want (telegram,chat-1)",
			opts.Dispatch.Channel(),
			opts.Dispatch.ChatID(),
		)
	}
	if opts.Dispatch.SenderID() != "user-1" || opts.Dispatch.MessageID() != "msg-1" {
		t.Fatalf("dispatch sender/message = (%q,%q)", opts.Dispatch.SenderID(), opts.Dispatch.MessageID())
	}
	if opts.Dispatch.ReplyToMessageID() != "reply-1" {
		t.Fatalf("Dispatch.ReplyToMessageID() = %q, want reply-1", opts.Dispatch.ReplyToMessageID())
	}
	if opts.Dispatch.UserMessage != "hello" {
		t.Fatalf("Dispatch.UserMessage = %q, want hello", opts.Dispatch.UserMessage)
	}
	if len(opts.Dispatch.Media) != 1 || opts.Dispatch.Media[0] != "media://one" {
		t.Fatalf("Dispatch.Media = %v, want [media://one]", opts.Dispatch.Media)
	}
}

func TestNormalizeProcessOptions_UsesDispatchAsSourceOfTruth(t *testing.T) {
	inbound := &bus.InboundContext{
		Channel:          "slack",
		ChatID:           "C123",
		ChatType:         "channel",
		SenderID:         "U123",
		MessageID:        "m-1",
		ReplyToMessageID: "parent-1",
	}
	route := &routing.ResolvedRoute{
		AgentID:   "support",
		Channel:   "slack",
		AccountID: "workspace-a",
		MatchedBy: "dispatch.rule:test",
		SessionPolicy: routing.SessionPolicy{
			Dimensions: []string{"chat", "sender"},
		},
	}
	scope := &session.SessionScope{
		Version:    session.ScopeVersionV1,
		AgentID:    "support",
		Channel:    "slack",
		Account:    "workspace-a",
		Dimensions: []string{"chat"},
		Values: map[string]string{
			"chat": "channel:c123",
		},
	}

	opts := normalizeProcessOptions(processOptions{
		Dispatch: DispatchRequest{
			SessionKey:     "sk_v1_example",
			SessionAliases: []string{"agent:support:slack:channel:c123"},
			InboundContext: inbound,
			RouteResult:    route,
			SessionScope:   scope,
			UserMessage:    "hello",
			Media:          []string{"media://one"},
		},
	})

	if opts.SessionKey != "sk_v1_example" {
		t.Fatalf("SessionKey = %q, want sk_v1_example", opts.SessionKey)
	}
	if opts.Channel != "slack" || opts.ChatID != "C123" {
		t.Fatalf("legacy mirrors = (%q,%q), want (slack,C123)", opts.Channel, opts.ChatID)
	}
	if opts.SenderID != "U123" || opts.MessageID != "m-1" {
		t.Fatalf("legacy sender/message = (%q,%q)", opts.SenderID, opts.MessageID)
	}
	if opts.ReplyToMessageID != "parent-1" {
		t.Fatalf("ReplyToMessageID = %q, want parent-1", opts.ReplyToMessageID)
	}
	if opts.RouteResult == nil || opts.RouteResult.AgentID != "support" {
		t.Fatalf("RouteResult = %#v, want support route", opts.RouteResult)
	}
	if opts.SessionScope == nil || opts.SessionScope.AgentID != "support" {
		t.Fatalf("SessionScope = %#v, want support scope", opts.SessionScope)
	}
}

func TestNormalizeProcessOptions_InfersLegacyChatTypeFromSessionScope(t *testing.T) {
	opts := normalizeProcessOptions(processOptions{
		Channel:     "telegram",
		ChatID:      "-100123",
		SenderID:    "user-1",
		UserMessage: "hello",
		SessionScope: &session.SessionScope{
			Version:    session.ScopeVersionV1,
			AgentID:    "main",
			Channel:    "telegram",
			Dimensions: []string{"chat"},
			Values: map[string]string{
				"chat": "group:-100123",
			},
		},
	})

	if opts.Dispatch.InboundContext == nil {
		t.Fatal("Dispatch.InboundContext is nil")
	}
	if opts.Dispatch.InboundContext.ChatType != "group" {
		t.Fatalf("Dispatch.InboundContext.ChatType = %q, want group", opts.Dispatch.InboundContext.ChatType)
	}
}
