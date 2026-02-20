package skills

import "testing"

func TestNewShowSubcommand(t *testing.T) {
	cmd := newShowCommand(nil)

	if cmd == nil {
		t.Fatalf("expected non-nil command")
	}

	if cmd.Use != "show" {
		t.Errorf("expected command name 'show', got %q", cmd.Use)
	}

	if cmd.Short != "Show skill details" {
		t.Errorf("expected command short description, got %q", cmd.Short)
	}

	if cmd.Run == nil {
		t.Error("expected command to have non-nil Run()")
	}

	if !cmd.HasExample() {
		t.Error("expected command to have example")
	}

	if cmd.HasSubCommands() {
		t.Error("expected command to have no subcommands")
	}

	if cmd.HasFlags() {
		t.Error("expected command to have no flags")
	}

	if len(cmd.Aliases) > 0 {
		t.Errorf("expected command to have no aliases, got %d", len(cmd.Aliases))
	}
}
