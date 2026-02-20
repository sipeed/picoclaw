package cron

import "testing"

func TestEnableSubcommand(t *testing.T) {
	fn := func() string { return "" }
	cmd := newEnableCommand(fn)

	if cmd == nil {
		t.Fatalf("expected non-nil command")
	}

	if cmd.Use != "enable" {
		t.Errorf("expected command name 'enable', got %q", cmd.Use)
	}

	if cmd.Short != "Enable a job" {
		t.Errorf("expected command short description, got %q", cmd.Short)
	}

	if !cmd.HasExample() {
		t.Error("expected command to have example")
	}
}
