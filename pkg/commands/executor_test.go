package commands

import (
	"context"
	"testing"
)

func TestExecutor_RegisteredButUnsupported_ReturnsRejected(t *testing.T) {
	defs := []Definition{{Name: "show", Channels: []string{"telegram"}}}
	ex := NewExecutor(NewRegistry(defs))

	res := ex.Execute(context.Background(), Request{Channel: "whatsapp", Text: "/show"}, nil)
	if res.Outcome != OutcomeRejected {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeRejected)
	}
}

func TestExecutor_UnknownSlashCommand_ReturnsPassthrough(t *testing.T) {
	defs := []Definition{{Name: "show", Channels: []string{"telegram"}}}
	ex := NewExecutor(NewRegistry(defs))

	res := ex.Execute(context.Background(), Request{Channel: "telegram", Text: "/unknown"}, nil)
	if res.Outcome != OutcomePassthrough {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomePassthrough)
	}
}

func TestExecutor_SupportedCommandWithHandler_ReturnsHandled(t *testing.T) {
	called := false
	defs := []Definition{
		{
			Name:     "help",
			Channels: []string{"telegram"},
			Handler: func(context.Context, Request) error {
				called = true
				return nil
			},
		},
	}
	ex := NewExecutor(NewRegistry(defs))

	res := ex.Execute(context.Background(), Request{Channel: "telegram", Text: "/help@my_bot"}, nil)
	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if !called {
		t.Fatalf("expected handler to be called")
	}
}
