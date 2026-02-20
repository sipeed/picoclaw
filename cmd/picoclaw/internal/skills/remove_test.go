package skills

import "testing"

func TestNewRemoveSubcommand(t *testing.T) {
	cmd := newRemoveCommand(nil)

	if cmd == nil {
		t.Fatalf("expected non-nil command")
	}

	if cmd.Use != "remove" {
		t.Errorf("expected command name 'remove', got %q", cmd.Use)
	}

	if cmd.Short != "Remove installed skill" {
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

	if len(cmd.Aliases) != 2 {
		t.Errorf("expected command to have 2 aliases, got %d", len(cmd.Aliases))
	}

	if !cmd.HasAlias("rm") {
		t.Errorf("expected command to have alias 'rm'")
	}

	if !cmd.HasAlias("uninstall") {
		t.Errorf("expected command to have alias 'uninstall'")
	}
}
