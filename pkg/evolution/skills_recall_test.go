package evolution_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sipeed/picoclaw/pkg/evolution"
)

func TestRecallSimilarSkills_ReturnsWorkspaceSkillFirst(t *testing.T) {
	workspace := t.TempDir()
	globalHome := t.TempDir()
	builtinRoot := t.TempDir()

	t.Setenv("HOME", globalHome)
	t.Setenv("PICOCLAW_BUILTIN_SKILLS", builtinRoot)

	mustWriteSkill := func(root, name, content string) {
		t.Helper()
		dir := filepath.Join(root, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll(%s): %v", dir, err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", name, err)
		}
	}

	mustWriteSkill(filepath.Join(workspace, "skills"), "weather", "---\nname: weather\ndescription: weather lookup\n---\n# Weather\nUse weather queries.\n")
	mustWriteSkill(filepath.Join(globalHome, ".picoclaw", "skills"), "release", "---\nname: release\ndescription: release flow\n---\n# Release\nRelease build.\n")
	mustWriteSkill(builtinRoot, "weather-fallback", "---\nname: weather-fallback\ndescription: weather backup\n---\n# Weather Fallback\nBackup weather path.\n")

	recaller := evolution.NewSkillsRecaller(workspace)
	matches, err := recaller.RecallSimilarSkills(evolution.LearningRecord{
		Kind:       evolution.RecordKindRule,
		Summary:    "weather native-name path",
		EventCount: 4,
	})
	if err != nil {
		t.Fatalf("RecallSimilarSkills: %v", err)
	}
	if len(matches) == 0 {
		t.Fatal("expected at least one match")
	}
	if matches[0].Name != "weather" {
		t.Fatalf("first match = %q, want weather", matches[0].Name)
	}
}
