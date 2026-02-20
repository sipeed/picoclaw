package auth

import "testing"

func TestNewStatusSubcommand(t *testing.T) {
	cmd := newStatusCommand()

	if cmd == nil {
		t.Fatalf("expected non-nil command")
	}

	if cmd.Short != "Show current auth status" {
		t.Errorf("expected command short description, got %q", cmd.Short)
	}

	if cmd.HasFlags() {
		t.Error("expected command to have no flags")
	}
}
