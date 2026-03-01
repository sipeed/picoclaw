package commands

import (
	"context"
	"testing"
)

func TestDispatcher_MatchSlashCommand(t *testing.T) {
	called := false
	defs := []Definition{
		{
			Name: "help",
			Handler: func(context.Context, Request) error {
				called = true
				return nil
			},
		},
	}
	d := NewDispatcher(NewRegistry(defs))

	res := d.Dispatch(context.Background(), Request{
		Channel: "telegram",
		Text:    "/help",
	})
	if !res.Matched || !called || res.Err != nil {
		t.Fatalf("dispatch result = %+v, called=%v", res, called)
	}
}

func TestDispatcher_DoesNotMatchWithoutSlash(t *testing.T) {
	d := NewDispatcher(NewRegistry([]Definition{{Name: "help"}}))

	res := d.Dispatch(context.Background(), Request{
		Channel: "telegram",
		Text:    "help",
	})
	if res.Matched {
		t.Fatalf("expected unmatched for plain text, got %+v", res)
	}
}

func TestDispatcher_MatchTelegramMentionSyntax(t *testing.T) {
	called := false
	d := NewDispatcher(NewRegistry([]Definition{
		{
			Name: "help",
			Handler: func(context.Context, Request) error {
				called = true
				return nil
			},
		},
	}))

	res := d.Dispatch(context.Background(), Request{
		Channel: "telegram",
		Text:    "/help@my_bot",
	})
	if !res.Matched || !res.Handled || !called || res.Err != nil {
		t.Fatalf("dispatch result = %+v, called=%v", res, called)
	}
}

func TestDispatcher_PassThroughDefinitionWithoutHandler(t *testing.T) {
	d := NewDispatcher(NewRegistry([]Definition{
		{Name: "session"}, // menu-only / pass-through definition
	}))

	res := d.Dispatch(context.Background(), Request{
		Channel: "telegram",
		Text:    "/session list",
	})
	if res.Matched {
		t.Fatalf("expected pass-through unmatched result, got %+v", res)
	}
}
