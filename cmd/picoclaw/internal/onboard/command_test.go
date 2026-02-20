package onboard

import "testing"

func TestNewOnboardCommand(t *testing.T) {
	cmd := NewOnboardCommand()

	if cmd == nil {
		t.Fatalf("expected non-nil command")
	}

	if cmd.Use != "onboard" {
		t.Errorf("expected command name 'onboard', got %q", cmd.Use)
	}

	if cmd.Short != "Initialize picoclaw configuration and workspace" {
		t.Errorf("expected command short description, got %q", cmd.Short)
	}

	if len(cmd.Aliases) != 1 {
		t.Errorf("expected command to have 1 alias, got %d", len(cmd.Aliases))
	}

	if !cmd.HasAlias("o") {
		t.Errorf("expected command to have alias 'o'")
	}

	if cmd.Run == nil {
		t.Error("expected command to have non-nil Run()")
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

	if cmd.HasFlags() {
		t.Error("expected command to have no flags")
	}

	if cmd.HasSubCommands() {
		t.Error("expected command to have no subcommands")
	}
}
