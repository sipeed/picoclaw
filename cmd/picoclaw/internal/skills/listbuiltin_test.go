package skills

import "testing"

func TestNewListbuiltinSubcommand(t *testing.T) {
	cmd := newListBuiltinCommand()

	if cmd == nil {
		t.Fatalf("expected non-nil command")
	}

	if cmd.Use != "list-builtin" {
		t.Errorf("expected command name 'list-builtin', got %q", cmd.Use)
	}

	if cmd.Short != "List available builtin skills" {
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
