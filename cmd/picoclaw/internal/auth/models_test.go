package auth

import "testing"

func TestNewModelsCommand(t *testing.T) {
	cmd := newModelsCommand()

	if cmd == nil {
		t.Fatalf("expected non-nil command")
	}

	if cmd.Use != "models" {
		t.Errorf("expected command name 'models', got %q", cmd.Use)
	}

	if cmd.Short != "Show available models" {
		t.Errorf("expected command short description, got %q", cmd.Short)
	}

	if cmd.HasFlags() {
		t.Error("expected command to have no flags")
	}
}
