package main

import (
	"testing"
)

func TestNewPicoclawCommand(t *testing.T) {
	cmd := NewPicoclawCommand()

	if cmd == nil {
		t.Fatalf("expected non-nil command")
	}

	if cmd.Use != "picoclaw" {
		t.Errorf("expected command name 'picoclaw', got %q", cmd.Use)
	}

	if cmd.Short != "picoclaw — Personal AI Assistant" {
		t.Errorf("expected command short description, got %q", cmd.Short)
	}

	if !cmd.HasSubCommands() {
		t.Error("expected command to have subcommands")
	}

	if !cmd.HasAvailableSubCommands() {
		t.Error("expected command to have available subcommands")
	}

	if cmd.HasFlags() {
		t.Error("expected command to have no flags")
	}

	if cmd.Run != nil {
		t.Error("expected command to have nil Run()")
	}

	if cmd.RunE != nil {
		t.Error("expected command to have nil RunE()")
	}

	if cmd.PersistentPreRun != nil {
		t.Error("expected command to have nil PersistentPreRun()")
	}

	if cmd.PersistentPostRun != nil {
		t.Error("expected command to have nil PersistentPostRun()")
	}

	allowedCommands := map[string]struct{}{
		"agent":   {},
		"auth":    {},
		"cron":    {},
		"gateway": {},
		"migrate": {},
		"onboard": {},
		"skills":  {},
		"status":  {},
		"version": {},
	}

	for _, subcmd := range cmd.Commands() {
		if _, found := allowedCommands[subcmd.Name()]; !found {
			t.Errorf("unexpected subcommand %q", subcmd.Name())
		}

		if cmd.Hidden {
			t.Errorf("expected subcommand %q to be visible", subcmd.Name())
		}
	}
}
