package status

import "testing"

func TestNewStatusCommand(t *testing.T) {
	cmd := NewStatusCommand()

	if cmd == nil {
		t.Fatalf("expected non-nil command")
	}

	if cmd.Use != "status" {
		t.Errorf("expected command name 'status', got %q", cmd.Use)
	}

	if len(cmd.Aliases) != 1 {
		t.Errorf("expected command to have 1 alias, got %d", len(cmd.Aliases))
	}

	if !cmd.HasAlias("s") {
		t.Errorf("expected command to have alias 's'")
	}

	if cmd.Short != "Show picoclaw status" {
		t.Errorf("expected command short description, got %q", cmd.Short)
	}

	if cmd.HasSubCommands() {
		t.Error("expected command to have no subcommands")
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
}
