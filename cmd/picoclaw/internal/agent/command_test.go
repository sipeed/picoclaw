package agent

import "testing"

func TestNewAgentCommand(t *testing.T) {
	cmd := NewAgentCommand()

	if cmd == nil {
		t.Fatalf("expected non-nil command")
	}

	if cmd.Use != "agent" {
		t.Errorf("expected command name 'agent', got %q", cmd.Use)
	}

	if cmd.Short != "Interact with the agent directly" {
		t.Errorf("expected command short description, got %q", cmd.Short)
	}

	if len(cmd.Aliases) > 0 {
		t.Errorf("expected command to have no aliases, got %d", len(cmd.Aliases))
	}

	if cmd.HasSubCommands() {
		t.Error("expected command to have no subcommands")
	}

	if cmd.Run != nil {
		t.Error("expected command to have nil Run()")
	}

	if cmd.RunE == nil {
		t.Error("expected command to have non-nil RunE()")
	}

	if cmd.PersistentPreRun != nil {
		t.Error("expected command to have nil PersistentPreRun()")
	}

	if cmd.PersistentPostRun != nil {
		t.Error("expected command to have nil PersistentPostRun()")
	}

	if !cmd.HasFlags() {
		t.Error("expected command to have flags")
	}

	if cmd.Flags().Lookup("debug") == nil {
		t.Error("expected command to have debug flag")
	}

	if cmd.Flags().Lookup("message") == nil {
		t.Error("expected command to have message flag")
	}

	if cmd.Flags().Lookup("session") == nil {
		t.Error("expected command to have session flag")
	}

	if cmd.Flags().Lookup("model") == nil {
		t.Error("expected command to have model flag")
	}
}
