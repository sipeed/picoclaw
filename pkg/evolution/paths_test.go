package evolution

import (
	"path/filepath"
	"testing"
)

func TestNewPaths_DefaultRoot(t *testing.T) {
	workspace := "/tmp/workspace"

	paths := NewPaths(workspace, "")

	wantRoot := filepath.Join(workspace, "state", "evolution")
	if paths.RootDir != wantRoot {
		t.Fatalf("RootDir = %q, want %q", paths.RootDir, wantRoot)
	}
	if paths.LearningRecords != filepath.Join(wantRoot, "learning-records.jsonl") {
		t.Fatalf("LearningRecords = %q", paths.LearningRecords)
	}
	if paths.TaskRecords != filepath.Join(wantRoot, "task-records.jsonl") {
		t.Fatalf("TaskRecords = %q", paths.TaskRecords)
	}
	if paths.PatternRecords != filepath.Join(wantRoot, "pattern-records.jsonl") {
		t.Fatalf("PatternRecords = %q", paths.PatternRecords)
	}
	if paths.SkillDrafts != filepath.Join(wantRoot, "skill-drafts.json") {
		t.Fatalf("SkillDrafts = %q", paths.SkillDrafts)
	}
	if paths.ProfilesDir != filepath.Join(wantRoot, "profiles") {
		t.Fatalf("ProfilesDir = %q", paths.ProfilesDir)
	}
	if paths.BackupsDir != filepath.Join(wantRoot, "backups") {
		t.Fatalf("BackupsDir = %q", paths.BackupsDir)
	}
}

func TestNewPaths_UsesOverride(t *testing.T) {
	workspace := "/tmp/workspace"
	override := "/tmp/custom-evolution"

	paths := NewPaths(workspace, override)

	if paths.RootDir != override {
		t.Fatalf("RootDir = %q, want %q", paths.RootDir, override)
	}
	if paths.LearningRecords != filepath.Join(override, "learning-records.jsonl") {
		t.Fatalf("LearningRecords = %q", paths.LearningRecords)
	}
	if paths.TaskRecords != filepath.Join(override, "task-records.jsonl") {
		t.Fatalf("TaskRecords = %q", paths.TaskRecords)
	}
	if paths.PatternRecords != filepath.Join(override, "pattern-records.jsonl") {
		t.Fatalf("PatternRecords = %q", paths.PatternRecords)
	}
	if paths.SkillDrafts != filepath.Join(override, "skill-drafts.json") {
		t.Fatalf("SkillDrafts = %q", paths.SkillDrafts)
	}
	if paths.ProfilesDir != filepath.Join(override, "profiles") {
		t.Fatalf("ProfilesDir = %q", paths.ProfilesDir)
	}
	if paths.BackupsDir != filepath.Join(override, "backups") {
		t.Fatalf("BackupsDir = %q", paths.BackupsDir)
	}
}

func TestNewPaths_BlankOverrideFallsBackToDefaultRoot(t *testing.T) {
	workspace := "/tmp/workspace"

	paths := NewPaths(workspace, " \t\n ")

	wantRoot := filepath.Join(workspace, "state", "evolution")
	if paths.RootDir != wantRoot {
		t.Fatalf("RootDir = %q, want %q", paths.RootDir, wantRoot)
	}
}

func TestNewPaths_TrimmedOverrideIsUsed(t *testing.T) {
	workspace := "/tmp/workspace"
	override := "  /tmp/custom-evolution  "

	paths := NewPaths(workspace, override)

	if paths.RootDir != "/tmp/custom-evolution" {
		t.Fatalf("RootDir = %q, want %q", paths.RootDir, "/tmp/custom-evolution")
	}
}
