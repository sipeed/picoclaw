package skills

import "testing"

func TestNewSkillsCommand(t *testing.T) {
	cmd := NewSkillsCommand()

	if cmd == nil {
		t.Fatalf("expected non-nil command")
	}

	if cmd.Use != "skills" {
		t.Errorf("expected command name 'skills', got %q", cmd.Use)
	}

	if cmd.Short != "Manage skills" {
		t.Errorf("expected command short description, got %q", cmd.Short)
	}

	if len(cmd.Aliases) > 0 {
		t.Errorf("expected command to have no aliases, got %d", len(cmd.Aliases))
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
		t.Error("expected command to have persistent pre-run hook")
	}

	if cmd.PersistentPreRun != nil {
		t.Error("expected command to have nil PersistentPreRun()")
	}

	if cmd.PersistentPostRun != nil {
		t.Error("expected command to have nil PersistentPostRun()")
	}
}
