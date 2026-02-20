package cron

import "testing"

func TestDisableSubcommand(t *testing.T) {
	fn := func() string { return "" }
	cmd := newDisableCommand(fn)

	if cmd == nil {
		t.Fatalf("expected non-nil command")
	}

	if cmd.Use != "disable" {
		t.Errorf("expected command name 'disable', got %q", cmd.Use)
	}

	if cmd.Short != "Disable a job" {
		t.Errorf("expected command short description, got %q", cmd.Short)
	}

	if !cmd.HasExample() {
		t.Error("expected command to have example")
	}
}
