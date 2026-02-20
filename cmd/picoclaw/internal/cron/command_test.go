package cron

import "testing"

func TestNewCronCommand(t *testing.T) {
	cmd := NewCronCommand()

	if cmd == nil {
		t.Fatalf("expected non-nil command")
	}

	if cmd.Short != "Manage scheduled tasks" {
		t.Errorf("expected command short description, got %q", cmd.Short)
	}

	if len(cmd.Aliases) != 1 {
		t.Errorf("expected command to have exactly one alias, got %d", len(cmd.Aliases))
	}

	if !cmd.HasAlias("c") {
		t.Errorf("expected command to have alias `c`, got %v", cmd.Aliases)
	}

	if cmd.HasFlags() {
		t.Error("expected command to have no flags")
	}

	if cmd.Run != nil {
		t.Error("expected command to have nil Run()")
	}

	if cmd.RunE == nil {
		t.Error("expected command to have non-nil RunE()")
	}

	if cmd.PersistentPreRunE == nil {
		t.Error("expected command to have non-nil PersistentPreRunE()")
	}

	if cmd.PersistentPreRun != nil {
		t.Error("expected command to have nil PersistentPreRun()")
	}

	if cmd.PersistentPostRun != nil {
		t.Error("expected command to have nil PersistentPostRun()")
	}

	if !cmd.HasSubCommands() {
		t.Error("expected command to have subcommands")
	}

	allowedCommands := map[string]struct{}{
		"list":    {},
		"add":     {},
		"remove":  {},
		"enable":  {},
		"disable": {},
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
