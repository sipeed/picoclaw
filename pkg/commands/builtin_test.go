package commands

import (
	"context"
	"strings"
	"testing"
)

func findDefinitionByName(t *testing.T, defs []Definition, name string) Definition {
	t.Helper()
	for _, def := range defs {
		if def.Name == name {
			return def
		}
	}
	t.Fatalf("missing /%s definition", name)
	return Definition{}
}

func TestBuiltinHelpHandler_ReturnsFormattedMessage(t *testing.T) {
	defs := BuiltinDefinitions()
	helpDef := findDefinitionByName(t, defs, "help")
	if helpDef.Handler == nil {
		t.Fatalf("/help handler should not be nil")
	}

	var reply string
	err := helpDef.Handler(context.Background(), Request{
		Text: "/help",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	}, nil)
	if err != nil {
		t.Fatalf("/help handler error: %v", err)
	}
	// Now uses auto-generated EffectiveUsage which includes agents
	if !strings.Contains(reply, "/show [model|channel|agents]") {
		t.Fatalf("/help reply missing /show usage, got %q", reply)
	}
	if !strings.Contains(reply, "/list [models|channels|agents]") {
		t.Fatalf("/help reply missing /list usage, got %q", reply)
	}
}

func TestBuiltinShowChannel_PreservesUserVisibleBehavior(t *testing.T) {
	defs := BuiltinDefinitions()
	ex := NewExecutor(NewRegistry(defs), nil)

	cases := []string{"telegram", "whatsapp"}
	for _, channel := range cases {
		var reply string
		res := ex.Execute(context.Background(), Request{
			Channel: channel,
			Text:    "/show channel",
			Reply: func(text string) error {
				reply = text
				return nil
			},
		})
		if res.Outcome != OutcomeHandled {
			t.Fatalf("/show channel on %s: outcome=%v, want=%v", channel, res.Outcome, OutcomeHandled)
		}
		want := "Current Channel: " + channel
		if reply != want {
			t.Fatalf("/show channel reply=%q, want=%q", reply, want)
		}
	}
}

func TestBuiltinListChannels_UsesGetEnabledChannels(t *testing.T) {
	rt := &Runtime{
		GetEnabledChannels: func() []string {
			return []string{"telegram", "slack"}
		},
	}
	defs := BuiltinDefinitions()
	ex := NewExecutor(NewRegistry(defs), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/list channels",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/list channels: outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if !strings.Contains(reply, "telegram") || !strings.Contains(reply, "slack") {
		t.Fatalf("/list channels reply=%q, want telegram and slack", reply)
	}
}

func TestBuiltinShowAgents_RestoresOldBehavior(t *testing.T) {
	rt := &Runtime{
		ListAgentIDs: func() []string {
			return []string{"default", "coder"}
		},
	}
	defs := BuiltinDefinitions()
	ex := NewExecutor(NewRegistry(defs), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/show agents",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/show agents: outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if !strings.Contains(reply, "default") || !strings.Contains(reply, "coder") {
		t.Fatalf("/show agents reply=%q, want agent IDs", reply)
	}
}

func TestBuiltinListAgents_RestoresOldBehavior(t *testing.T) {
	rt := &Runtime{
		ListAgentIDs: func() []string {
			return []string{"default", "coder"}
		},
	}
	defs := BuiltinDefinitions()
	ex := NewExecutor(NewRegistry(defs), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/list agents",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/list agents: outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if !strings.Contains(reply, "default") || !strings.Contains(reply, "coder") {
		t.Fatalf("/list agents reply=%q, want agent IDs", reply)
	}
}
