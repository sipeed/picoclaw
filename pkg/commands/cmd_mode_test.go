package commands

import (
	"context"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

func newModeTestRuntime() *Runtime {
	return &Runtime{
		Config: &config.Config{
			Agents: config.AgentsConfig{
				Defaults: config.AgentDefaults{
					ModelName: "gpt-5.4-mini",
					Routing: &config.RoutingConfig{
						LightModel: "openrouter-free",
					},
				},
			},
		},
	}
}

func TestBoostCommand_ArmsNextModel(t *testing.T) {
	rt := newModeTestRuntime()
	var armed string
	rt.ArmNextModelMode = func(value string) error {
		armed = value
		return nil
	}

	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/boost",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if armed != "gpt-5.4-mini" {
		t.Fatalf("armed=%q, want %q", armed, "gpt-5.4-mini")
	}
	if reply != "Boost armed. Next message will use gpt-5.4-mini." {
		t.Fatalf("reply=%q, want boost confirmation", reply)
	}
}

func TestPaidCommand_SetsPersistentModel(t *testing.T) {
	rt := newModeTestRuntime()
	var persistent string
	rt.SetSessionModelMode = func(value string) error {
		persistent = value
		return nil
	}

	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/paid",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if persistent != "gpt-5.4-mini" {
		t.Fatalf("persistent=%q, want %q", persistent, "gpt-5.4-mini")
	}
	if reply != "Session mode set to paid (gpt-5.4-mini)." {
		t.Fatalf("reply=%q, want paid confirmation", reply)
	}
}

func TestFreeCommand_SetsPersistentModel(t *testing.T) {
	rt := newModeTestRuntime()
	var persistent string
	rt.SetSessionModelMode = func(value string) error {
		persistent = value
		return nil
	}

	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/free",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if persistent != "openrouter-free" {
		t.Fatalf("persistent=%q, want %q", persistent, "openrouter-free")
	}
	if reply != "Session mode set to free (openrouter-free)." {
		t.Fatalf("reply=%q, want free confirmation", reply)
	}
}

func TestStatusCommand_ReportsPendingBoost(t *testing.T) {
	rt := newModeTestRuntime()
	rt.GetModelInfo = func() (string, string) {
		return "gpt-5.4-mini", "openai"
	}
	rt.GetSessionModelMode = func() (string, string) {
		return "openrouter-free", "gpt-5.4-mini"
	}

	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/status",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if !containsAll(reply, []string{
		"Current Model: gpt-5.4-mini (Provider: openai)",
		"Session Mode: boost armed for next message (gpt-5.4-mini)",
		"Pending Boost: gpt-5.4-mini",
		"Paid Model: gpt-5.4-mini",
		"Free Model: openrouter-free",
	}) {
		t.Fatalf("reply=%q, missing expected status content", reply)
	}
}

func containsAll(text string, want []string) bool {
	for _, s := range want {
		if !strings.Contains(text, s) {
			return false
		}
	}
	return true
}
