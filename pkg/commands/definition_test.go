package commands

import (
	"testing"
)

func TestDefinition_EffectiveUsage_NoSubCommands(t *testing.T) {
	d := Definition{Name: "start", Usage: "/start"}
	if got := d.EffectiveUsage(); got != "/start" {
		t.Fatalf("EffectiveUsage()=%q, want %q", got, "/start")
	}
}

func TestDefinition_EffectiveUsage_WithSubCommands(t *testing.T) {
	d := Definition{
		Name: "show",
		SubCommands: []SubCommand{
			{Name: "model"},
			{Name: "channel"},
			{Name: "agents"},
		},
	}
	want := "/show [model|channel|agents]"
	if got := d.EffectiveUsage(); got != want {
		t.Fatalf("EffectiveUsage()=%q, want %q", got, want)
	}
}

func TestDefinition_EffectiveUsage_WithArgsUsage(t *testing.T) {
	d := Definition{
		Name: "session",
		SubCommands: []SubCommand{
			{Name: "list"},
			{Name: "resume", ArgsUsage: "<id>"},
		},
	}
	want := "/session [list|resume <id>]"
	if got := d.EffectiveUsage(); got != want {
		t.Fatalf("EffectiveUsage()=%q, want %q", got, want)
	}
}
