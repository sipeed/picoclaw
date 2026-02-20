package cron

import "testing"

func TestNewRemoveSubcommand(t *testing.T) {
	fn := func() string { return "" }
	cmd := newRemoveCommand(fn)

	if cmd == nil {
		t.Fatalf("expected non-nil command")
	}

	if cmd.Short != "Remove a job by ID" {
		t.Errorf("expected command short description, got %q", cmd.Short)
	}

	if !cmd.HasExample() {
		t.Error("expected command to have example")
	}
}
