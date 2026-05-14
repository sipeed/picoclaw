package evolution_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sipeed/picoclaw/pkg/evolution"
)

func TestApplyLifecycleStateDeletedRemovesSkillFile(t *testing.T) {
	workspace := t.TempDir()
	skillDir := filepath.Join(workspace, "skills", "weather")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte("# weather\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	err := evolution.ApplyLifecycleState(
		evolution.NewPaths(workspace, ""),
		evolution.SkillProfile{SkillName: "weather"},
		evolution.SkillStatusDeleted,
	)
	if err != nil {
		t.Fatalf("ApplyLifecycleState: %v", err)
	}

	if _, err := os.Stat(skillPath); !os.IsNotExist(err) {
		t.Fatalf("skill file should be removed, stat err = %v", err)
	}
}

func TestApplyLifecycleStateDeletedRequiresResolvedWorkspace(t *testing.T) {
	err := evolution.ApplyLifecycleState(
		evolution.Paths{RootDir: filepath.Join(t.TempDir(), "shared-evolution")},
		evolution.SkillProfile{SkillName: "weather"},
		evolution.SkillStatusDeleted,
	)
	if err == nil {
		t.Fatal("expected error when workspace cannot be resolved")
	}
}

func TestApplyLifecycleStateDeletedRequiresSkillName(t *testing.T) {
	workspace := t.TempDir()

	err := evolution.ApplyLifecycleState(
		evolution.NewPaths(workspace, ""),
		evolution.SkillProfile{WorkspaceID: workspace},
		evolution.SkillStatusDeleted,
	)
	if err == nil {
		t.Fatal("expected error when skill name is empty")
	}
}

func TestApplyLifecycleStateDeletedRejectsTraversalSkillName(t *testing.T) {
	workspace := t.TempDir()

	err := evolution.ApplyLifecycleState(
		evolution.NewPaths(workspace, ""),
		evolution.SkillProfile{WorkspaceID: workspace, SkillName: "../escape"},
		evolution.SkillStatusDeleted,
	)
	if err == nil {
		t.Fatal("expected error for traversal skill name")
	}
}
