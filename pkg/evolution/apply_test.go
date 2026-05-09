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

func TestApplier_CreateDraftRendersDeployableSkillWithoutLearningTrace(t *testing.T) {
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
		BodyOrPatch: strings.Join([]string{
			"---",
			"name: weather",
			"description: Create combined shortcut perform-mathematical-calculations-by-via-theorems for: Perform mathematical calculations by applying specific theorems and their associated rules.",
			"---",
			"# Weather",
			"",
			"## Learned Context",
			"- Learned task: use native-name weather lookup.",
			"",
			"## Source Evidence",
			"- Evidence: learned from task records: task-1",
			"",
			"## Procedure",
			"Use native-name query first.",
			"",
		}, "\n"),
		Status: evolution.DraftStatusAccepted,
	}

	if err := applier.ApplyDraft(context.Background(), workspace, draft); err != nil {
		t.Fatalf("ApplyDraft: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(workspace, "skills", "weather", "SKILL.md"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)
	for _, forbidden := range []string{
		"Create combined shortcut",
		"perform-mathematical-calculations-by-via-theorems for:",
		"Learned Context",
		"Learned task",
		"Source Evidence",
		"task records",
	} {
		if strings.Contains(content, forbidden) {
			t.Fatalf("deployed skill contains %q:\n%s", forbidden, content)
		}
	}
	if !strings.Contains(content, "Use native-name query first.") {
		t.Fatalf("deployed skill lost procedure:\n%s", content)
	}
	if !strings.Contains(content, "description: Perform mathematical calculations by applying specific theorems and their associated rules.") {
		t.Fatalf("deployed skill did not clean description:\n%s", content)
	}
}

func TestApplier_CreateDraftDoesNotRewriteEvolutionDomainTextOrFrontmatter(t *testing.T) {
	workspace := t.TempDir()
	applier := evolution.NewApplier(evolution.NewPaths(workspace, ""), func() time.Time {
		return time.Unix(1700000000, 0).UTC()
	})

	draft := evolution.SkillDraft{
		ID:              "draft-evolution-domain",
		WorkspaceID:     workspace,
		SourceRecordID:  "rule-evolution-domain",
		TargetSkillName: "agent-evolution-helper",
		DraftType:       evolution.DraftTypeShortcut,
		ChangeKind:      evolution.ChangeKindCreate,
		HumanSummary:    "agent evolution helper",
		BodyOrPatch:     "---\nname: agent-evolution-helper\ndescription: Explain agent evolution workflows.\n---\n# Agent Evolution Helper\nUse this skill to reason about agent evolution behavior.\n",
		Status:          evolution.DraftStatusAccepted,
	}

	if err := applier.ApplyDraft(context.Background(), workspace, draft); err != nil {
		t.Fatalf("ApplyDraft: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(workspace, "skills", "agent-evolution-helper", "SKILL.md"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "name: agent-evolution-helper") {
		t.Fatalf("frontmatter name was rewritten:\n%s", content)
	}
	if strings.Contains(content, "agent-update-helper") {
		t.Fatalf("frontmatter name should not be rewritten:\n%s", content)
	}
	if !strings.Contains(content, "agent evolution behavior") {
		t.Fatalf("domain text should preserve evolution wording:\n%s", content)
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

func TestApplier_CreateDraftRejectsMismatchedFrontmatterName(t *testing.T) {
	workspace := t.TempDir()
	applier := evolution.NewApplier(evolution.NewPaths(workspace, ""), func() time.Time {
		return time.Unix(1700000000, 0).UTC()
	})

	draft := evolution.SkillDraft{
		ID:              "draft-mismatched-name",
		WorkspaceID:     workspace,
		SourceRecordID:  "rule-mismatched-name",
		TargetSkillName: "weather",
		DraftType:       evolution.DraftTypeShortcut,
		ChangeKind:      evolution.ChangeKindCreate,
		HumanSummary:    "weather helper",
		BodyOrPatch:     "---\nname: other-skill\ndescription: other helper\n---\n# Other\nUse something else.\n",
	}

	err := applier.ApplyDraft(context.Background(), workspace, draft)
	if err == nil {
		t.Fatal("expected ApplyDraft to fail")
	}
	if !strings.Contains(err.Error(), "frontmatter name") {
		t.Fatalf("error = %v, want frontmatter name mismatch", err)
	}
	if _, statErr := os.Stat(filepath.Join(workspace, "skills", "weather", "SKILL.md")); !os.IsNotExist(statErr) {
		t.Fatalf("expected no skill file, got err=%v", statErr)
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

func TestApplier_FailedNewSkillDoesNotLeaveEmptyDirectory(t *testing.T) {
	workspace := t.TempDir()
	applier := evolution.NewApplier(evolution.NewPaths(workspace, ""), func() time.Time {
		return time.Unix(1700000000, 0).UTC()
	})

	draft := evolution.SkillDraft{
		ID:              "draft-invalid-new-skill",
		WorkspaceID:     workspace,
		SourceRecordID:  "rule-invalid-new-skill",
		TargetSkillName: "calculate-100-via-theorems",
		DraftType:       evolution.DraftTypeWorkflow,
		ChangeKind:      evolution.ChangeKindCreate,
		HumanSummary:    "broken new skill",
		BodyOrPatch:     "invalid-frontmatter",
		Status:          evolution.DraftStatusAccepted,
	}

	err := applier.ApplyDraft(context.Background(), workspace, draft)
	if err == nil {
		t.Fatal("expected ApplyDraft to fail")
	}

	skillPath := filepath.Join(workspace, "skills", "calculate-100-via-theorems", "SKILL.md")
	if _, statErr := os.Stat(skillPath); !os.IsNotExist(statErr) {
		t.Fatalf("expected no skill file, got err=%v", statErr)
	}
	skillDir := filepath.Dir(skillPath)
	if _, statErr := os.Stat(skillDir); !os.IsNotExist(statErr) {
		t.Fatalf("expected no leftover skill dir, got err=%v", statErr)
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

func TestApplier_AppendDraftAllowsExistingExtraFrontmatterFields(t *testing.T) {
	workspace := t.TempDir()
	skillDir := filepath.Join(workspace, "skills", "weather")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	original := strings.Join([]string{
		"---",
		"name: weather",
		"description: valid",
		"# Human-authored metadata should not block append updates.",
		"homepage: https://example.com/weather",
		"aliases:",
		"- forecast",
		"metadata:",
		"  owner: human",
		"---",
		"# Weather",
		"## Start Here",
		"Use city names.",
		"",
	}, "\n")
	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte(original), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	applier := evolution.NewApplier(evolution.NewPaths(workspace, ""), func() time.Time {
		return time.Unix(1700000000, 0).UTC()
	})

	draft := evolution.SkillDraft{
		ID:              "draft-append-extra-frontmatter",
		WorkspaceID:     workspace,
		SourceRecordID:  "rule-append-extra-frontmatter",
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
	for _, want := range []string{
		"homepage: https://example.com/weather",
		"aliases:",
		"- forecast",
		"metadata:",
		"  owner: human",
		"Prefer native-name query first.",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("appended content missing %q:\n%s", want, content)
		}
	}
}

func TestApplier_CreateDraftRejectsExtraFrontmatterFields(t *testing.T) {
	workspace := t.TempDir()
	applier := evolution.NewApplier(evolution.NewPaths(workspace, ""), func() time.Time {
		return time.Unix(1700000000, 0).UTC()
	})

	draft := evolution.SkillDraft{
		ID:              "draft-create-extra-frontmatter",
		WorkspaceID:     workspace,
		SourceRecordID:  "rule-create-extra-frontmatter",
		TargetSkillName: "weather",
		DraftType:       evolution.DraftTypeShortcut,
		ChangeKind:      evolution.ChangeKindCreate,
		HumanSummary:    "weather helper",
		BodyOrPatch:     "---\nname: weather\ndescription: weather helper\nhomepage: https://example.com/weather\n---\n# Weather\nUse weather.\n",
	}

	err := applier.ApplyDraft(context.Background(), workspace, draft)
	if err == nil {
		t.Fatal("expected ApplyDraft to fail")
	}
	if !strings.Contains(err.Error(), "unsupported skill frontmatter field") {
		t.Fatalf("error = %v, want unsupported field", err)
	}
}

func TestApplier_AppendDraftDoesNotRewriteExistingLearningTerms(t *testing.T) {
	workspace := t.TempDir()
	skillDir := filepath.Join(workspace, "skills", "weather")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	original := "---\nname: weather\ndescription: valid\n---\n# Weather\n## Evolution Notes\nKeep this manually-authored Learned phrase unchanged.\n"
	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte(original), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	applier := evolution.NewApplier(evolution.NewPaths(workspace, ""), func() time.Time {
		return time.Unix(1700000000, 0).UTC()
	})

	draft := evolution.SkillDraft{
		ID:              "draft-append-clean",
		WorkspaceID:     workspace,
		SourceRecordID:  "rule-append-clean",
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
	if !strings.Contains(content, "## Evolution Notes") {
		t.Fatalf("existing heading was rewritten:\n%s", content)
	}
	if !strings.Contains(content, "Keep this manually-authored Learned phrase unchanged.") {
		t.Fatalf("existing body was rewritten:\n%s", content)
	}
	if strings.Contains(content, "## Learned Pattern") {
		t.Fatalf("new patch should be deploy-sanitized:\n%s", content)
	}
	if !strings.Contains(content, "## Usage Pattern") {
		t.Fatalf("new patch missing sanitized heading:\n%s", content)
	}
}

func TestApplier_AppendDraftStripsPlainMarkdownTopLevelHeading(t *testing.T) {
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
		ID:              "draft-append-plain-doc",
		WorkspaceID:     workspace,
		SourceRecordID:  "rule-append-plain-doc",
		TargetSkillName: "weather",
		DraftType:       evolution.DraftTypeWorkflow,
		ChangeKind:      evolution.ChangeKindAppend,
		HumanSummary:    "append draft",
		BodyOrPatch:     "# Weather\n## Procedure\nPrefer native-name query first.\n",
	}

	if err := applier.ApplyDraft(context.Background(), workspace, draft); err != nil {
		t.Fatalf("ApplyDraft: %v", err)
	}

	got, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(got)
	if strings.Count(content, "# Weather") != 1 {
		t.Fatalf("appended content should not duplicate top-level heading:\n%s", content)
	}
	if !strings.Contains(content, "## Procedure") || !strings.Contains(content, "Prefer native-name query first.") {
		t.Fatalf("appended content lost patch body:\n%s", content)
	}
}

func TestApplier_AppendAndMergeRejectFullDocumentPatchWithMismatchedName(t *testing.T) {
	for _, kind := range []evolution.ChangeKind{evolution.ChangeKindAppend, evolution.ChangeKindMerge} {
		t.Run(string(kind), func(t *testing.T) {
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

			err := applier.ApplyDraft(context.Background(), workspace, evolution.SkillDraft{
				ID:              "draft-mismatched-patch",
				WorkspaceID:     workspace,
				SourceRecordID:  "rule-mismatched-patch",
				TargetSkillName: "weather",
				DraftType:       evolution.DraftTypeWorkflow,
				ChangeKind:      kind,
				HumanSummary:    "append draft",
				BodyOrPatch:     "---\nname: other-skill\ndescription: wrong target\n---\n# Other Skill\n## Procedure\nDo something else.\n",
			})
			if err == nil {
				t.Fatal("expected ApplyDraft to fail")
			}
			if !strings.Contains(err.Error(), "patch frontmatter name") {
				t.Fatalf("error = %v, want patch frontmatter name mismatch", err)
			}
			got, readErr := os.ReadFile(skillPath)
			if readErr != nil {
				t.Fatalf("ReadFile: %v", readErr)
			}
			if string(got) != original {
				t.Fatalf("skill content changed unexpectedly:\n%s", string(got))
			}
		})
	}
}

func TestApplier_AppendDraftStripsFullSkillDocumentPatch(t *testing.T) {
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
		ID:              "draft-append-full-doc",
		WorkspaceID:     workspace,
		SourceRecordID:  "rule-append-full-doc",
		TargetSkillName: "weather",
		DraftType:       evolution.DraftTypeWorkflow,
		ChangeKind:      evolution.ChangeKindAppend,
		HumanSummary:    "append draft",
		BodyOrPatch:     "---\nname: weather\ndescription: duplicate document\n---\n# Weather\n## Procedure\nPrefer native-name query first.\n",
	}

	if err := applier.ApplyDraft(context.Background(), workspace, draft); err != nil {
		t.Fatalf("ApplyDraft: %v", err)
	}

	got, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(got)
	if strings.Count(content, "---") != 2 {
		t.Fatalf("appended content should keep only original frontmatter:\n%s", content)
	}
	if strings.Count(content, "# Weather") != 1 {
		t.Fatalf("appended content should not duplicate top-level heading:\n%s", content)
	}
	if !strings.Contains(content, "## Procedure") || !strings.Contains(content, "Prefer native-name query first.") {
		t.Fatalf("appended content lost patch body:\n%s", content)
	}
}

func TestApplier_BackupsAreScopedByWorkspace(t *testing.T) {
	sharedState := t.TempDir()
	workspaceA := t.TempDir()
	workspaceB := t.TempDir()
	now := time.Unix(1700000000, 0).UTC()

	for workspace, body := range map[string]string{
		workspaceA: "---\nname: weather\ndescription: valid\n---\n# Weather\nworkspace A\n",
		workspaceB: "---\nname: weather\ndescription: valid\n---\n# Weather\nworkspace B\n",
	} {
		skillDir := filepath.Join(workspace, "skills", "weather")
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			t.Fatalf("MkdirAll(%s): %v", workspace, err)
		}
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(body), 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", workspace, err)
		}
	}

	for _, workspace := range []string{workspaceA, workspaceB} {
		applier := evolution.NewApplier(evolution.NewPaths(workspace, sharedState), func() time.Time {
			return now
		})
		if err := applier.ApplyDraft(context.Background(), workspace, evolution.SkillDraft{
			ID:              "draft-replace",
			WorkspaceID:     workspace,
			SourceRecordID:  "rule-replace",
			TargetSkillName: "weather",
			DraftType:       evolution.DraftTypeWorkflow,
			ChangeKind:      evolution.ChangeKindReplace,
			HumanSummary:    "replace weather",
			BodyOrPatch:     "---\nname: weather\ndescription: replacement\n---\n# Weather\nreplacement\n",
		}); err != nil {
			t.Fatalf("ApplyDraft(%s): %v", workspace, err)
		}
	}

	var backupBodies []string
	if err := filepath.WalkDir(filepath.Join(sharedState, "backups"), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || entry.Name() != "SKILL.md" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		backupBodies = append(backupBodies, string(data))
		return nil
	}); err != nil {
		t.Fatalf("WalkDir(backups): %v", err)
	}

	if len(backupBodies) != 2 {
		t.Fatalf("backup count = %d, want 2", len(backupBodies))
	}
	joined := strings.Join(backupBodies, "\n")
	if !strings.Contains(joined, "workspace A") || !strings.Contains(joined, "workspace B") {
		t.Fatalf("backups should preserve both workspace bodies:\n%s", joined)
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
