package commands

import (
	"context"
	"strings"
	"testing"
)

func TestShowListHandlers_ChannelPolicy(t *testing.T) {
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), nil)

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

	var whatsappReply string
	handledWhatsApp := ex.Execute(context.Background(), Request{
		Channel: "whatsapp",
		Text:    "/show channel",
		Reply: func(text string) error {
			whatsappReply = text
			return nil
		},
	})
	if handledWhatsApp.Outcome != OutcomeHandled {
		t.Fatalf("whatsapp /show outcome=%v, want=%v", handledWhatsApp.Outcome, OutcomeHandled)
	}
	if handledWhatsApp.Command != "show" {
		t.Fatalf("whatsapp /show command=%q, want=%q", handledWhatsApp.Command, "show")
	}
	if whatsappReply != "Current Channel: whatsapp" {
		t.Fatalf("whatsapp /show reply=%q, want=%q", whatsappReply, "Current Channel: whatsapp")
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
}

func TestShowListHandlers_ListHandledOnAllChannels(t *testing.T) {
	rt := &Runtime{
		GetEnabledChannels: func() []string {
			return []string{"telegram"}
		},
		ListSkillNames: func() []string {
			return []string{"shell"}
		},
	}
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Channel: "whatsapp",
		Text:    "/list channels",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("whatsapp /list outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if res.Command != "list" {
		t.Fatalf("whatsapp /list command=%q, want=%q", res.Command, "list")
	}
	if !strings.Contains(reply, "telegram") {
		t.Fatalf("whatsapp /list reply=%q, expected enabled channels content", reply)
	}

	reply = ""
	res = ex.Execute(context.Background(), Request{
		Channel: "whatsapp",
		Text:    "/list skills",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("whatsapp /list skills outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if !strings.Contains(reply, "shell") {
		t.Fatalf("whatsapp /list skills reply=%q, expected installed skills content", reply)
	}
}
