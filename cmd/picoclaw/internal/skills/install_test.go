package skills

import "testing"

func TestNewInstallSubcommand(t *testing.T) {
	cmd := newInstallCommand(nil)

	if cmd == nil {
		t.Fatalf("expected non-nil command")
	}

	if cmd.Use != "install" {
		t.Errorf("expected command name 'install', got %q", cmd.Use)
	}

	if cmd.Short != "Install skill from GitHub" {
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
