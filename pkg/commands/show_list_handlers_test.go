package commands

import (
	"context"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestShowListHandlers_ChannelPolicy(t *testing.T) {
	cfg := &config.Config{}
	ex := NewExecutor(NewRegistry(BuiltinDefinitions(cfg)))

	var telegramReply string
	handled := ex.Execute(context.Background(), Request{
		Channel: "telegram",
		Text:    "/show channel",
		Reply: func(text string) error {
			telegramReply = text
			return nil
		},
	})
	if handled.Outcome != OutcomeHandled {
		t.Fatalf("telegram /show outcome=%v, want=%v", handled.Outcome, OutcomeHandled)
	}
	if telegramReply != "Current Channel: telegram" {
		t.Fatalf("telegram /show reply=%q, want=%q", telegramReply, "Current Channel: telegram")
	}

	rejected := ex.Execute(context.Background(), Request{
		Channel: "whatsapp",
		Text:    "/show channel",
	})
	if rejected.Outcome != OutcomeRejected {
		t.Fatalf("whatsapp /show outcome=%v, want=%v", rejected.Outcome, OutcomeRejected)
	}
	if rejected.Command != "show" {
		t.Fatalf("whatsapp /show command=%q, want=%q", rejected.Command, "show")
	}
	if rejected.Reply != "Command /show is not supported on whatsapp." {
		t.Fatalf("whatsapp /show reply=%q, want=%q", rejected.Reply, "Command /show is not supported on whatsapp.")
	}

	passthrough := ex.Execute(context.Background(), Request{
		Channel: "whatsapp",
		Text:    "/foo",
	})
	if passthrough.Outcome != OutcomePassthrough {
		t.Fatalf("whatsapp /foo outcome=%v, want=%v", passthrough.Outcome, OutcomePassthrough)
	}
	if passthrough.Command != "foo" {
		t.Fatalf("whatsapp /foo command=%q, want=%q", passthrough.Command, "foo")
	}
	if passthrough.Reply != "" {
		t.Fatalf("whatsapp /foo reply=%q, want empty", passthrough.Reply)
	}
}

func TestShowListHandlers_ListRejectsUnsupportedChannel(t *testing.T) {
	cfg := &config.Config{}
	ex := NewExecutor(NewRegistry(BuiltinDefinitions(cfg)))

	res := ex.Execute(context.Background(), Request{
		Channel: "whatsapp",
		Text:    "/list channels",
	})
	if res.Outcome != OutcomeRejected {
		t.Fatalf("whatsapp /list outcome=%v, want=%v", res.Outcome, OutcomeRejected)
	}
	if res.Command != "list" {
		t.Fatalf("whatsapp /list command=%q, want=%q", res.Command, "list")
	}
	if res.Reply != "Command /list is not supported on whatsapp." {
		t.Fatalf("whatsapp /list reply=%q, want=%q", res.Reply, "Command /list is not supported on whatsapp.")
	}
}
