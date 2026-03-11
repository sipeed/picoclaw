package commands

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestExecutor_RegisteredWithoutHandler_ReturnsPassthrough(t *testing.T) {
	defs := []Definition{{Name: "show"}}
	ex := NewExecutor(NewRegistry(defs), nil)

	res := ex.Execute(context.Background(), Request{Channel: "whatsapp", Text: "/show"})
	if res.Outcome != OutcomePassthrough {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomePassthrough)
	}
}

func TestExecutor_UnknownSlashCommand_ReturnsPassthrough(t *testing.T) {
	defs := []Definition{{Name: "show"}}
	ex := NewExecutor(NewRegistry(defs), nil)

	res := ex.Execute(context.Background(), Request{Channel: "telegram", Text: "/unknown"})
	if res.Outcome != OutcomePassthrough {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomePassthrough)
	}
}

func TestExecutor_SupportedCommandWithHandler_ReturnsHandled(t *testing.T) {
	called := false
	defs := []Definition{
		{
			Name: "help",
			Handler: func(context.Context, Request, *Runtime) error {
				called = true
				return nil
			},
		},
	}
	ex := NewExecutor(NewRegistry(defs), nil)

	res := ex.Execute(context.Background(), Request{Channel: "telegram", Text: "/help@my_bot"})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if !called {
		t.Fatalf("expected handler to be called")
	}
}

func TestExecutor_AliasWithoutHandler_ReturnsPassthrough(t *testing.T) {
	defs := []Definition{
		{
			Name:    "show",
			Aliases: []string{"display"},
		},
	}
	ex := NewExecutor(NewRegistry(defs), nil)

	res := ex.Execute(context.Background(), Request{Channel: "whatsapp", Text: "/display"})
	if res.Outcome != OutcomePassthrough {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomePassthrough)
	}
	if res.Command != "show" {
		t.Fatalf("command=%q, want=%q", res.Command, "show")
	}
}

func TestExecutor_AliasWithHandler_ReturnsHandled(t *testing.T) {
	called := false
	defs := []Definition{
		{
			Name:    "clear",
			Aliases: []string{"reset"},
			Handler: func(context.Context, Request, *Runtime) error {
				called = true
				return nil
			},
		},
	}
	ex := NewExecutor(NewRegistry(defs), nil)

	res := ex.Execute(context.Background(), Request{Channel: "telegram", Text: "/reset"})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if res.Command != "clear" {
		t.Fatalf("command=%q, want=%q", res.Command, "clear")
	}
	if !called {
		t.Fatalf("expected handler to be called")
	}
}

func TestExecutor_SupportedCommandWithNilHandler_ReturnsPassthrough(t *testing.T) {
	defs := []Definition{
		{Name: "placeholder"},
	}
	ex := NewExecutor(NewRegistry(defs), nil)

	res := ex.Execute(context.Background(), Request{Channel: "telegram", Text: "/placeholder list"})
	if res.Outcome != OutcomePassthrough {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomePassthrough)
	}
	if res.Command != "placeholder" {
		t.Fatalf("command=%q, want=%q", res.Command, "placeholder")
	}
}

func TestExecutor_NilHandlerDoesNotMaskLaterHandler(t *testing.T) {
	// With Lookup-based dispatch, the first registered definition for a name wins.
	// A definition with nil Handler and no SubCommands returns Passthrough.
	defs := []Definition{
		{Name: "placeholder"},
	}
	ex := NewExecutor(NewRegistry(defs), nil)

	res := ex.Execute(context.Background(), Request{Channel: "telegram", Text: "/placeholder"})
	if res.Outcome != OutcomePassthrough {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomePassthrough)
	}
	if res.Command != "placeholder" {
		t.Fatalf("command=%q, want=%q", res.Command, "placeholder")
	}
}

func TestExecutor_HandlerErrorIsPropagated(t *testing.T) {
	wantErr := errors.New("handler failed")
	defs := []Definition{
		{
			Name: "help",
			Handler: func(context.Context, Request, *Runtime) error {
				return wantErr
			},
		},
	}
	ex := NewExecutor(NewRegistry(defs), nil)

	res := ex.Execute(context.Background(), Request{Channel: "telegram", Text: "/help"})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if !errors.Is(res.Err, wantErr) {
		t.Fatalf("err=%v, want=%v", res.Err, wantErr)
	}
}

func TestExecutor_SupportsBangPrefixAndCaseInsensitiveCommand(t *testing.T) {
	called := false
	defs := []Definition{
		{
			Name: "help",
			Handler: func(context.Context, Request, *Runtime) error {
				called = true
				return nil
			},
		},
	}
	ex := NewExecutor(NewRegistry(defs), nil)

	res := ex.Execute(context.Background(), Request{Channel: "telegram", Text: "!HELP"})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if !called {
		t.Fatalf("expected handler to be called")
	}
}

func TestExecutor_SubCommand_RoutesToCorrectHandler(t *testing.T) {
	modelCalled := false
	defs := []Definition{
		{
			Name: "show",
			SubCommands: []SubCommand{
				{Name: "model", Handler: func(_ context.Context, _ Request, _ *Runtime) error {
					modelCalled = true
					return nil
				}},
				{Name: "channel"},
			},
		},
	}
	ex := NewExecutor(NewRegistry(defs), nil)

	res := ex.Execute(context.Background(), Request{Text: "/show model"})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if !modelCalled {
		t.Fatal("model sub-command handler was not called")
	}
}

func TestExecutor_SubCommand_NoArg_RepliesUsage(t *testing.T) {
	defs := []Definition{
		{
			Name: "show",
			SubCommands: []SubCommand{
				{Name: "model"},
				{Name: "channel"},
			},
		},
	}
	ex := NewExecutor(NewRegistry(defs), nil)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text:  "/show",
		Reply: func(text string) error { reply = text; return nil },
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if reply != "Usage: /show [model|channel]" {
		t.Fatalf("reply=%q, want usage message", reply)
	}
}

func TestExecutor_SubCommand_UnknownArg_RepliesError(t *testing.T) {
	defs := []Definition{
		{
			Name: "show",
			SubCommands: []SubCommand{
				{Name: "model"},
			},
		},
	}
	ex := NewExecutor(NewRegistry(defs), nil)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text:  "/show foobar",
		Reply: func(text string) error { reply = text; return nil },
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if !strings.Contains(reply, "foobar") {
		t.Fatalf("reply=%q, should mention unknown sub-command", reply)
	}
}

func TestExecutor_SubCommand_NilHandler_ReturnsPassthrough(t *testing.T) {
	defs := []Definition{
		{
			Name: "show",
			SubCommands: []SubCommand{
				{Name: "model"}, // nil Handler
			},
		},
	}
	ex := NewExecutor(NewRegistry(defs), nil)

	res := ex.Execute(context.Background(), Request{Text: "/show model"})
	if res.Outcome != OutcomePassthrough {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomePassthrough)
	}
}
