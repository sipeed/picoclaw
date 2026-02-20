package auth

import "testing"

func TestNewAuthCommand(t *testing.T) {
	cmd := NewAuthCommand()

	if cmd == nil {
		t.Fatalf("expected non-nil command")
	}

	if cmd.Use != "auth" {
		t.Errorf("expected command name 'auth', got %q", cmd.Use)
	}

	if cmd.Short != "Manage authentication (login, logout, status)" {
		t.Errorf("expected command short description, got %q", cmd.Short)
	}

	if len(cmd.Aliases) > 0 {
		t.Errorf("expected command to have no aliases, got %d", len(cmd.Aliases))
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

	if cmd.HasFlags() {
		t.Error("expected command to have no flags")
	}

	if !cmd.HasSubCommands() {
		t.Error("expected command to have subcommands")
	}

	allowedCommands := map[string]struct{}{
		"login":  {},
		"logout": {},
		"status": {},
		"models": {},
	}

	for _, subcmd := range cmd.Commands() {
		if _, found := allowedCommands[subcmd.Name()]; !found {
			t.Errorf("unexpected subcommand %q", subcmd.Name())
		}

		if len(subcmd.Aliases) > 0 {
			t.Errorf("expected subcommand %q to have no aliases, got %d", subcmd.Name(), len(subcmd.Aliases))
		}

		if cmd.Hidden {
			t.Errorf("expected subcommand %q to be visible", subcmd.Name())
		}

		if subcmd.HasSubCommands() {
			t.Errorf("expected subcommand `%s` to have no subcommands", subcmd.Name())
		}

		if subcmd.Run != nil {
			t.Errorf("expected subcommand `%s` to have nil Run()", subcmd.Name())
		}

		if subcmd.RunE == nil {
			t.Errorf("expected subcommand `%s` to have non-nil RunE()", subcmd.Name())
		}

		if subcmd.PersistentPreRun != nil {
			t.Errorf("expected subcommand `%s` to have nil PersistentPreRun()", subcmd.Name())
		}

		if subcmd.PersistentPostRun != nil {
			t.Errorf("expected subcommand `%s` to have nil PersistentPostRun()", subcmd.Name())
		}
	}
}
