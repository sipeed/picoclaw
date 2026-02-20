package skills

import "testing"

func TestNewSearchSubcommand(t *testing.T) {
	cmd := newSearchCommand(nil)

	if cmd == nil {
		t.Fatalf("expected non-nil command")
	}

	if cmd.Use != "search" {
		t.Errorf("expected command name 'search', got %q", cmd.Use)
	}

	if cmd.Short != "Search available skills" {
		t.Errorf("expected command short description, got %q", cmd.Short)
	}

	if cmd.Run == nil {
		t.Error("expected command to have non-nil Run()")
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
