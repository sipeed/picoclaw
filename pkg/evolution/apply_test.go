package evolution_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/evolution"
)

func TestApplier_CreateDraftWritesSkillFile(t *testing.T) {
	workspace := t.TempDir()
	applier := evolution.NewApplier(evolution.NewPaths(workspace, ""), func() time.Time {
		return time.Unix(1700000000, 0).UTC()
	})

	draft := evolution.SkillDraft{
		ID:              "draft-1",
		WorkspaceID:     workspace,
		SourceRecordID:  "rule-1",
		TargetSkillName: "weather",
		DraftType:       evolution.DraftTypeShortcut,
		ChangeKind:      evolution.ChangeKindCreate,
		HumanSummary:    "weather helper",
		BodyOrPatch:     "---\nname: weather\ndescription: weather helper\n---\n# Weather\n## Start Here\nUse native-name query first.\n",
		Status:          evolution.DraftStatusAccepted,
	}

	if err := applier.ApplyDraft(context.Background(), workspace, draft); err != nil {
		t.Fatalf("ApplyDraft: %v", err)
	}

	skillPath := filepath.Join(workspace, "skills", "weather", "SKILL.md")
	data, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), "# Weather") {
		t.Fatalf("unexpected content: %s", string(data))
	}
}

func TestApplier_CreateDraftFailsWhenSkillAlreadyExists(t *testing.T) {
	workspace := t.TempDir()
	skillDir := filepath.Join(workspace, "skills", "weather")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	original := "---\nname: weather\ndescription: valid\n---\n# Weather\nold body\n"
	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte(original), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	applier := evolution.NewApplier(evolution.NewPaths(workspace, ""), func() time.Time {
		return time.Unix(1700000000, 0).UTC()
	})

	draft := evolution.SkillDraft{
		ID:              "draft-create-existing",
		WorkspaceID:     workspace,
		SourceRecordID:  "rule-create-existing",
		TargetSkillName: "weather",
		DraftType:       evolution.DraftTypeShortcut,
		ChangeKind:      evolution.ChangeKindCreate,
		HumanSummary:    "weather helper",
		BodyOrPatch:     "---\nname: weather\ndescription: replacement\n---\n# Weather\nnew body\n",
	}

	err := applier.ApplyDraft(context.Background(), workspace, draft)
	if err == nil {
		t.Fatal("expected ApplyDraft to fail")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("error = %v, want already exists", err)
	}

	got, readErr := os.ReadFile(skillPath)
	if readErr != nil {
		t.Fatalf("ReadFile: %v", readErr)
	}
	if string(got) != original {
		t.Fatalf("skill content changed unexpectedly:\n%s", string(got))
	}
}

func TestApplier_RollsBackOnInvalidSkillBody(t *testing.T) {
	workspace := t.TempDir()
	skillDir := filepath.Join(workspace, "skills", "weather")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	original := "---\nname: weather\ndescription: valid\n---\n# Weather\nold body\n"
	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte(original), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	applier := evolution.NewApplier(evolution.NewPaths(workspace, ""), func() time.Time {
		return time.Unix(1700000000, 0).UTC()
	})

	draft := evolution.SkillDraft{
		ID:              "draft-2",
		WorkspaceID:     workspace,
		SourceRecordID:  "rule-2",
		TargetSkillName: "weather",
		DraftType:       evolution.DraftTypeWorkflow,
		ChangeKind:      evolution.ChangeKindReplace,
		HumanSummary:    "broken draft",
		BodyOrPatch:     "invalid-frontmatter",
		Status:          evolution.DraftStatusAccepted,
	}

	err := applier.ApplyDraft(context.Background(), workspace, draft)
	if err == nil {
		t.Fatal("expected ApplyDraft to fail")
	}

	got, readErr := os.ReadFile(skillPath)
	if readErr != nil {
		t.Fatalf("ReadFile: %v", readErr)
	}
	if string(got) != original {
		t.Fatalf("skill content changed after rollback:\n%s", string(got))
	}
}

func TestApplier_ReplaceDraftFailsWhenSkillDoesNotExist(t *testing.T) {
	workspace := t.TempDir()
	applier := evolution.NewApplier(evolution.NewPaths(workspace, ""), func() time.Time {
		return time.Unix(1700000000, 0).UTC()
	})

	draft := evolution.SkillDraft{
		ID:              "draft-replace-missing",
		WorkspaceID:     workspace,
		SourceRecordID:  "rule-replace-missing",
		TargetSkillName: "weather",
		DraftType:       evolution.DraftTypeWorkflow,
		ChangeKind:      evolution.ChangeKindReplace,
		HumanSummary:    "replace missing skill",
		BodyOrPatch:     "---\nname: weather\ndescription: replacement\n---\n# Weather\nnew body\n",
	}

	err := applier.ApplyDraft(context.Background(), workspace, draft)
	if err == nil {
		t.Fatal("expected ApplyDraft to fail")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("error = %v, want does not exist", err)
	}

	skillPath := filepath.Join(workspace, "skills", "weather", "SKILL.md")
	if _, statErr := os.Stat(skillPath); !os.IsNotExist(statErr) {
		t.Fatalf("expected no skill file, got err=%v", statErr)
	}
}

func TestApplier_AppendDraftPreservesOriginalBody(t *testing.T) {
	workspace := t.TempDir()
	skillDir := filepath.Join(workspace, "skills", "weather")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	original := "---\nname: weather\ndescription: valid\n---\n# Weather\n## Start Here\nUse city names.\n"
	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte(original), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	applier := evolution.NewApplier(evolution.NewPaths(workspace, ""), func() time.Time {
		return time.Unix(1700000000, 0).UTC()
	})

	draft := evolution.SkillDraft{
		ID:              "draft-append",
		WorkspaceID:     workspace,
		SourceRecordID:  "rule-append",
		TargetSkillName: "weather",
		DraftType:       evolution.DraftTypeWorkflow,
		ChangeKind:      evolution.ChangeKindAppend,
		HumanSummary:    "append draft",
		BodyOrPatch:     "\n## Learned Pattern\nPrefer native-name query first.\n",
	}

	if err := applier.ApplyDraft(context.Background(), workspace, draft); err != nil {
		t.Fatalf("ApplyDraft: %v", err)
	}

	got, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(got)
	if !strings.Contains(content, "Use city names.") {
		t.Fatalf("appended content lost original body:\n%s", content)
	}
	if !strings.Contains(content, "Prefer native-name query first.") {
		t.Fatalf("appended content missing new body:\n%s", content)
	}
}

func TestApplier_MergeDraftAddsMergedKnowledgeSection(t *testing.T) {
	workspace := t.TempDir()
	skillDir := filepath.Join(workspace, "skills", "weather")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	original := "---\nname: weather\ndescription: valid\n---\n# Weather\n## Start Here\nUse city names.\n"
	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte(original), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	applier := evolution.NewApplier(evolution.NewPaths(workspace, ""), func() time.Time {
		return time.Unix(1700000000, 0).UTC()
	})

	draft := evolution.SkillDraft{
		ID:              "draft-merge",
		WorkspaceID:     workspace,
		SourceRecordID:  "rule-merge",
		TargetSkillName: "weather",
		DraftType:       evolution.DraftTypeWorkflow,
		ChangeKind:      evolution.ChangeKindMerge,
		HumanSummary:    "merge draft",
		BodyOrPatch:     "Prefer native-name query first.",
	}

	if err := applier.ApplyDraft(context.Background(), workspace, draft); err != nil {
		t.Fatalf("ApplyDraft: %v", err)
	}

	got, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(got)
	if !strings.Contains(content, "Use city names.") {
		t.Fatalf("merged content lost original body:\n%s", content)
	}
	if !strings.Contains(content, "## Merged Knowledge") {
		t.Fatalf("merged content missing merged section:\n%s", content)
	}
	if !strings.Contains(content, "Prefer native-name query first.") {
		t.Fatalf("merged content missing new knowledge:\n%s", content)
	}
}

func TestApplier_RejectsInvalidSkillName(t *testing.T) {
	workspace := t.TempDir()
	applier := evolution.NewApplier(evolution.NewPaths(workspace, ""), func() time.Time {
		return time.Unix(1700000000, 0).UTC()
	})

	for _, name := range []string{"../escape", "/tmp/escape"} {
		err := applier.ApplyDraft(context.Background(), workspace, evolution.SkillDraft{
			ID:              "draft-invalid-name",
			WorkspaceID:     workspace,
			SourceRecordID:  "rule-invalid-name",
			TargetSkillName: name,
			DraftType:       evolution.DraftTypeShortcut,
			ChangeKind:      evolution.ChangeKindCreate,
			HumanSummary:    "bad name",
			BodyOrPatch:     "---\nname: weather\ndescription: weather helper\n---\n# Weather\nbody\n",
		})
		if err == nil {
			t.Fatalf("TargetSkillName %q expected error", name)
		}
	}
}
