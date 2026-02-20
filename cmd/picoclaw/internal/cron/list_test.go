package cron

import "testing"

func TestNewListSubcommand(t *testing.T) {
	fn := func() string { return "" }
	cmd := newListCommand(fn)

	if cmd == nil {
		t.Fatalf("expected non-nil command")
	}

	if cmd.Short != "List all scheduled jobs" {
		t.Errorf("expected command short description, got %q", cmd.Short)
	}
}
