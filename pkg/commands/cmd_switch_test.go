package commands

import (
	"context"
	"fmt"
	"testing"
)

func TestSwitchModel_Success(t *testing.T) {
	rt := &Runtime{
		SwitchModel: func(value string) (string, error) {
			return "old-model", nil
		},
	}
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/switch model to gpt-4",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	want := "Switched model from old-model to gpt-4"
	if reply != want {
		t.Fatalf("reply=%q, want=%q", reply, want)
	}
}

func TestSwitchModel_MissingToKeyword(t *testing.T) {
	rt := &Runtime{
		SwitchModel: func(value string) (string, error) {
			return "old", nil
		},
	}
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/switch model gpt-4",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if reply != "Usage: /switch model to <name>" {
		t.Fatalf("reply=%q, want usage message", reply)
	}
}

func TestSwitchModel_MissingValue(t *testing.T) {
	rt := &Runtime{
		SwitchModel: func(value string) (string, error) {
			return "old", nil
		},
	}
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/switch model to",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if reply != "Usage: /switch model to <name>" {
		t.Fatalf("reply=%q, want usage message", reply)
	}
}

func TestSwitchModel_Error(t *testing.T) {
	rt := &Runtime{
		SwitchModel: func(value string) (string, error) {
			return "", fmt.Errorf("model not found")
		},
	}
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/switch model to bad-model",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if reply != "model not found" {
		t.Fatalf("reply=%q, want error message", reply)
	}
}

func TestSwitchModel_NilDep(t *testing.T) {
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), &Runtime{})

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/switch model to gpt-4",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if reply != "Command unavailable in current context." {
		t.Fatalf("reply=%q, want unavailable message", reply)
	}
}

func TestSwitchChannel_Redirect(t *testing.T) {
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), &Runtime{})

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/switch channel to telegram",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	want := "This command has moved. Please use: /check channel <name>"
	if reply != want {
		t.Fatalf("reply=%q, want=%q", reply, want)
	}
}

func TestCheckChannel_Success(t *testing.T) {
	rt := &Runtime{
		SwitchChannel: func(value string) error {
			return nil
		},
	}
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/check channel telegram",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	want := "Channel 'telegram' is available and enabled"
	if reply != want {
		t.Fatalf("reply=%q, want=%q", reply, want)
	}
}

func TestCheckChannel_Error(t *testing.T) {
	rt := &Runtime{
		SwitchChannel: func(value string) error {
			return fmt.Errorf("channel '%s' not found", value)
		},
	}
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/check channel unknown",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if reply != "channel 'unknown' not found" {
		t.Fatalf("reply=%q, want error message", reply)
	}
}

func TestCheckChannel_NilDep(t *testing.T) {
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), &Runtime{})

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/check channel telegram",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if reply != "Command unavailable in current context." {
		t.Fatalf("reply=%q, want unavailable message", reply)
	}
}

func TestCheckChannel_MissingValue(t *testing.T) {
	rt := &Runtime{
		SwitchChannel: func(value string) error {
			return nil
		},
	}
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/check channel",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if reply != "Usage: /check channel <name>" {
		t.Fatalf("reply=%q, want usage message", reply)
	}
}

func TestSwitch_BangPrefix(t *testing.T) {
	rt := &Runtime{
		SwitchModel: func(value string) (string, error) {
			return "old", nil
		},
	}
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "!switch model to gpt-4",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("! prefix: outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if reply != "Switched model from old to gpt-4" {
		t.Fatalf("! prefix: reply=%q, want success message", reply)
	}
}

func TestSwitch_NoSubCommand(t *testing.T) {
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), &Runtime{})

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/switch",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	// Should get usage message from executor's sub-command routing
	if reply == "" {
		t.Fatal("expected usage reply for bare /switch")
	}
}
