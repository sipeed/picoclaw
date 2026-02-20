package gateway

import "testing"

func TestNewGatewayCommand(t *testing.T) {
	cmd := NewGatewayCommand()

	if cmd == nil {
		t.Fatalf("expected non-nil command")
	}

	if cmd.Use != "gateway" {
		t.Errorf("expected command name 'gateway', got %q", cmd.Use)
	}

	if cmd.Short != "Start picoclaw gateway" {
		t.Errorf("expected command short description, got %q", cmd.Short)
	}

	if len(cmd.Aliases) != 1 {
		t.Errorf("expected command to have 1 alias, got %d", len(cmd.Aliases))
	}

	if !cmd.HasAlias("g") {
		t.Errorf("expected command to have alias 'g'")
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

	if cmd.HasSubCommands() {
		t.Error("expected command to have no subcommands")
	}

	if !cmd.HasFlags() {
		t.Error("expected command to have flags")
	}

	if cmd.Flags().Lookup("debug") == nil {
		t.Error("expected command to have debug flag")
	}
}
