package commands

import (
	"context"
	"errors"
	"testing"
)

func TestExecutor_RegisteredButUnsupported_ReturnsRejected(t *testing.T) {
	defs := []Definition{{Name: "show", Channels: []string{"telegram"}}}
	ex := NewExecutor(NewRegistry(defs))

	res := ex.Execute(context.Background(), Request{Channel: "whatsapp", Text: "/show"})
	if res.Outcome != OutcomeRejected {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeRejected)
	}
}

func TestExecutor_UnknownSlashCommand_ReturnsPassthrough(t *testing.T) {
	defs := []Definition{{Name: "show", Channels: []string{"telegram"}}}
	ex := NewExecutor(NewRegistry(defs))

	res := ex.Execute(context.Background(), Request{Channel: "telegram", Text: "/unknown"})
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

	res := ex.Execute(context.Background(), Request{Channel: "telegram", Text: "/help@my_bot"})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if !called {
		t.Fatalf("expected handler to be called")
	}
}

func TestExecutor_AliasUnsupportedChannel_ReturnsRejected(t *testing.T) {
	defs := []Definition{
		{
			Name:     "show",
			Aliases:  []string{"display"},
			Channels: []string{"telegram"},
		},
	}
	ex := NewExecutor(NewRegistry(defs))

	res := ex.Execute(context.Background(), Request{Channel: "whatsapp", Text: "/display"})
	if res.Outcome != OutcomeRejected {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeRejected)
	}
	if res.Command != "show" {
		t.Fatalf("command=%q, want=%q", res.Command, "show")
	}
}

func TestExecutor_AliasWithHandler_ReturnsHandled(t *testing.T) {
	called := false
	defs := []Definition{
		{
			Name:     "new",
			Aliases:  []string{"reset"},
			Channels: []string{"telegram"},
			Handler: func(context.Context, Request) error {
				called = true
				return nil
			},
		},
	}
	ex := NewExecutor(NewRegistry(defs))

	res := ex.Execute(context.Background(), Request{Channel: "telegram", Text: "/reset"})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if res.Command != "new" {
		t.Fatalf("command=%q, want=%q", res.Command, "new")
	}
	if !called {
		t.Fatalf("expected handler to be called")
	}
}

func TestExecutor_SupportedCommandWithNilHandler_ReturnsPassthrough(t *testing.T) {
	defs := []Definition{
		{Name: "session", Channels: []string{"telegram"}},
	}
	ex := NewExecutor(NewRegistry(defs))

	res := ex.Execute(context.Background(), Request{Channel: "telegram", Text: "/session list"})
	if res.Outcome != OutcomePassthrough {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomePassthrough)
	}
	if res.Command != "session" {
		t.Fatalf("command=%q, want=%q", res.Command, "session")
	}
}

func TestExecutor_NilHandlerDoesNotMaskLaterHandler(t *testing.T) {
	called := false
	defs := []Definition{
		{Name: "session", Channels: []string{"telegram"}},
		{
			Name:     "session",
			Channels: []string{"telegram"},
			Handler: func(context.Context, Request) error {
				called = true
				return nil
			},
		},
	}
	ex := NewExecutor(NewRegistry(defs))

	res := ex.Execute(context.Background(), Request{Channel: "telegram", Text: "/session"})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if res.Command != "session" {
		t.Fatalf("command=%q, want=%q", res.Command, "session")
	}
	if !called {
		t.Fatalf("expected later handler to be called")
	}
}

func TestExecutor_HandlerErrorIsPropagated(t *testing.T) {
	wantErr := errors.New("handler failed")
	defs := []Definition{
		{
			Name:     "help",
			Channels: []string{"telegram"},
			Handler: func(context.Context, Request) error {
				return wantErr
			},
		},
	}
	ex := NewExecutor(NewRegistry(defs))

	res := ex.Execute(context.Background(), Request{Channel: "telegram", Text: "/help"})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if !errors.Is(res.Err, wantErr) {
		t.Fatalf("err=%v, want=%v", res.Err, wantErr)
	}
}
