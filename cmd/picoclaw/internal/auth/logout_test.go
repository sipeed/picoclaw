package auth

import "testing"

func TestNewLogoutSubcommand(t *testing.T) {
	cmd := newLogoutCommand()

	if cmd == nil {
		t.Fatalf("expected non-nil command")
	}

	if cmd.Short != "Remove stored credentials" {
		t.Errorf("expected command short description, got %q", cmd.Short)
	}

	if !cmd.HasFlags() {
		t.Error("expected command to have flags")
	}

	if cmd.Flags().Lookup("provider") == nil {
		t.Error("expected command to have provider flag")
	}
}
