package migrate

import "testing"

func TestNewMigrateCommand(t *testing.T) {
	cmd := NewMigrateCommand()

	if cmd == nil {
		t.Fatalf("expected non-nil command")
	}

	if cmd.Use != "migrate" {
		t.Errorf("expected command name 'migrate', got %q", cmd.Use)
	}

	if cmd.Short != "Migrate from OpenClaw to PicoClaw" {
		t.Errorf("expected command short description, got %q", cmd.Short)
	}

	if len(cmd.Aliases) > 0 {
		t.Errorf("expected command to have no aliases, got %d", len(cmd.Aliases))
	}

	if !cmd.HasExample() {
		t.Error("expected command to have example")
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

	if cmd.Flags().Lookup("dry-run") == nil {
		t.Error("expected command to have dry-run flag")
	}

	if cmd.Flags().Lookup("refresh") == nil {
		t.Error("expected command to have refresh flag")
	}

	if cmd.Flags().Lookup("config-only") == nil {
		t.Error("expected command to have config-only flag")
	}

	if cmd.Flags().Lookup("workspace-only") == nil {
		t.Error("expected command to have workspace-only flag")
	}

	if cmd.Flags().Lookup("force") == nil {
		t.Error("expected command to have force flag")
	}

	if cmd.Flags().Lookup("openclaw-home") == nil {
		t.Error("expected command to have openclaw-home flag")
	}

	if cmd.Flags().Lookup("picoclaw-home") == nil {
		t.Error("expected command to have picoclaw-home flag")
	}
}
