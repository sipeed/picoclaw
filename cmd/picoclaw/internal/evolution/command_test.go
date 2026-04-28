package evolutioncmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/evolution"
	"github.com/sipeed/picoclaw/pkg/providers"
)

func TestNewEvolutionCommand(t *testing.T) {
	cmd := NewEvolutionCommand()
	if cmd.Use != "evolution" {
		t.Fatalf("Use = %q, want evolution", cmd.Use)
	}

	names := make([]string, 0, len(cmd.Commands()))
	for _, sub := range cmd.Commands() {
		names = append(names, sub.Name())
	}
	sort.Strings(names)

	want := []string{"apply", "drafts", "prune", "review", "rollback", "run-once", "status"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("commands = %v, want %v", names, want)
	}
}

func TestDraftsListsWorkspaceDraftDetails(t *testing.T) {
	sharedState := filepath.Join(t.TempDir(), "shared-evolution")
	workspace := t.TempDir()
	otherWorkspace := t.TempDir()
	configureEvolutionCommandTest(t, workspace, sharedState)
	store := evolution.NewStore(evolution.NewPaths(workspace, sharedState))

	drafts := []evolution.SkillDraft{
		{
			ID:              "draft-1",
			WorkspaceID:     workspace,
			SourceRecordID:  "pattern-1",
			TargetSkillName: "weather",
			DraftType:       evolution.DraftTypeShortcut,
			ChangeKind:      evolution.ChangeKindAppend,
			HumanSummary:    "Prefer native-name query first",
			Status:          evolution.DraftStatusCandidate,
		},
		{
			ID:              "draft-2",
			WorkspaceID:     otherWorkspace,
			SourceRecordID:  "pattern-2",
			TargetSkillName: "maps",
			DraftType:       evolution.DraftTypeWorkflow,
			ChangeKind:      evolution.ChangeKindReplace,
			HumanSummary:    "Other workspace draft",
			Status:          evolution.DraftStatusAccepted,
		},
	}
	if err := store.SaveDrafts(drafts); err != nil {
		t.Fatalf("SaveDrafts: %v", err)
	}

	cmd := newDraftsCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := out.String()
	for _, want := range []string{
		"id=draft-1",
		"status=candidate",
		"target=weather",
		"change=append",
		"summary=Prefer native-name query first",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
	if strings.Contains(output, "draft-2") {
		t.Fatalf("output should exclude other workspace draft:\n%s", output)
	}
}

func TestReviewShowsDraftDetails(t *testing.T) {
	workspace := t.TempDir()
	configureEvolutionCommandTest(t, workspace, "")
	store := evolution.NewStore(evolution.NewPaths(workspace, ""))

	now := time.Unix(1700000000, 0).UTC()
	if err := store.SaveDrafts([]evolution.SkillDraft{
		{
			ID:              "draft-review",
			WorkspaceID:     workspace,
			CreatedAt:       now,
			SourceRecordID:  "pattern-1",
			TargetSkillName: "weather",
			DraftType:       evolution.DraftTypeShortcut,
			ChangeKind:      evolution.ChangeKindAppend,
			HumanSummary:    "Prefer native-name query first",
			IntendedUseCases: []string{
				"weather native-name path",
			},
			PreferredEntryPath: []string{"weather"},
			AvoidPatterns:      []string{"avoid starting with geocode before using weather"},
			BodyOrPatch:        "## Start Here\nUse native-name query first.\n",
			Status:             evolution.DraftStatusCandidate,
			ReviewNotes:        []string{"local structural validation completed"},
		},
	}); err != nil {
		t.Fatalf("SaveDrafts: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(workspace, "skills", "weather"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(workspace, "skills", "weather", "SKILL.md"),
		[]byte("---\nname: weather\ndescription: weather helper\n---\n# Weather\n## Start Here\nUse city names first.\n"),
		0o644,
	); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := store.SaveProfile(evolution.SkillProfile{
		SkillName:          "weather",
		WorkspaceID:        workspace,
		CurrentVersion:     "draft-stable",
		Status:             evolution.SkillStatusActive,
		Origin:             "evolved",
		HumanSummary:       "Weather helper",
		ChangeReason:       "stable weather path",
		IntendedUseCases:   []string{"basic weather lookup"},
		PreferredEntryPath: []string{"geocode", "weather"},
		AvoidPatterns:      []string{"avoid translating city names twice"},
		LastUsedAt:         now.Add(-time.Hour),
		UseCount:           5,
		VersionHistory: []evolution.SkillVersionEntry{
			{
				Version:   "draft-old",
				Action:    "create",
				Timestamp: now.Add(-48 * time.Hour),
				DraftID:   "draft-old",
				Summary:   "initial stable version",
			},
			{
				Version:   "draft-stable",
				Action:    "manual_apply:append",
				Timestamp: now.Add(-2 * time.Hour),
				DraftID:   "draft-stable",
				Summary:   "manual CLI apply: stable weather path",
			},
		},
	}); err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}

	cmd := newReviewCommand()
	cmd.SetArgs([]string{"draft-review"})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := out.String()
	for _, want := range []string{
		"id=draft-review",
		"source=pattern-1",
		"target=weather",
		"type=shortcut",
		"change=append",
		"status=candidate",
		"review_notes=local structural validation completed",
		"intended_use_cases=weather native-name path",
		"preferred_entry_path=weather",
		"avoid_patterns=avoid starting with geocode before using weather",
		"profile:",
		"skill=weather status=active version=draft-stable uses=5 reason=stable weather path",
		"current_preferred_entry_path=geocode -> weather",
		"recent_history:",
		"version=draft-stable action=manual_apply:append draft_id=draft-stable summary=manual CLI apply: stable weather path",
		"impact_preview:",
		"will_update_existing_skill=true",
		"expected_effect=append a new section onto the current skill",
		"current_body:",
		"Use city names first.",
		"rendered_body:",
		"## Start Here",
		"diff_preview:",
		"--- current",
		"+++ rendered",
		"@@",
		"+## Start Here",
		"+Use native-name query first.",
		"body:",
		"Use native-name query first.",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}

func TestApplyCommandAppliesDraftAndMarksAccepted(t *testing.T) {
	workspace := t.TempDir()
	configureEvolutionCommandTest(t, workspace, "")
	store := evolution.NewStore(evolution.NewPaths(workspace, ""))

	if err := store.SaveDrafts([]evolution.SkillDraft{
		{
			ID:              "draft-apply",
			WorkspaceID:     workspace,
			SourceRecordID:  "pattern-1",
			TargetSkillName: "weather",
			DraftType:       evolution.DraftTypeShortcut,
			ChangeKind:      evolution.ChangeKindCreate,
			HumanSummary:    "Create weather shortcut",
			BodyOrPatch:     "---\nname: weather\ndescription: weather helper\n---\n# Weather\n## Start Here\nUse native-name query first.\n",
			Status:          evolution.DraftStatusCandidate,
		},
	}); err != nil {
		t.Fatalf("SaveDrafts: %v", err)
	}

	cmd := newApplyCommand()
	cmd.SetArgs([]string{"draft-apply"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	drafts, err := store.LoadDrafts()
	if err != nil {
		t.Fatalf("LoadDrafts: %v", err)
	}
	if len(drafts) != 1 {
		t.Fatalf("len(drafts) = %d, want 1", len(drafts))
	}
	if drafts[0].Status != evolution.DraftStatusAccepted {
		t.Fatalf("draft status = %q, want accepted", drafts[0].Status)
	}
	if drafts[0].UpdatedAt == nil {
		t.Fatal("expected UpdatedAt to be set after apply")
	}

	skillPath := filepath.Join(workspace, "skills", "weather", "SKILL.md")
	data, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), "Use native-name query first.") {
		t.Fatalf("unexpected skill body:\n%s", string(data))
	}

	profile, err := store.LoadProfile("weather")
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	if profile.CurrentVersion != "draft-apply" {
		t.Fatalf("CurrentVersion = %q, want draft-apply", profile.CurrentVersion)
	}
	if profile.Status != evolution.SkillStatusActive {
		t.Fatalf("Status = %q, want active", profile.Status)
	}
}

func TestRollbackCommandRestoresLatestBackupAndUpdatesProfile(t *testing.T) {
	workspace := t.TempDir()
	configureEvolutionCommandTest(t, workspace, "")
	store := evolution.NewStore(evolution.NewPaths(workspace, ""))

	skillDir := filepath.Join(workspace, "skills", "weather")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	skillPath := filepath.Join(skillDir, "SKILL.md")
	original := "---\nname: weather\ndescription: weather helper\n---\n# Weather\nOld stable body.\n"
	if err := os.WriteFile(skillPath, []byte(original), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	applier := evolution.NewApplier(evolution.NewPaths(workspace, ""), func() time.Time {
		return time.Unix(1700000000, 0).UTC()
	})
	if err := applier.ApplyDraft(context.Background(), workspace, evolution.SkillDraft{
		ID:              "draft-new",
		WorkspaceID:     workspace,
		SourceRecordID:  "pattern-2",
		TargetSkillName: "weather",
		DraftType:       evolution.DraftTypeWorkflow,
		ChangeKind:      evolution.ChangeKindReplace,
		HumanSummary:    "Replace weather body",
		BodyOrPatch:     "---\nname: weather\ndescription: weather helper\n---\n# Weather\nNew risky body.\n",
		Status:          evolution.DraftStatusAccepted,
	}); err != nil {
		t.Fatalf("ApplyDraft: %v", err)
	}

	if err := store.SaveDrafts([]evolution.SkillDraft{
		{
			ID:              "draft-new",
			WorkspaceID:     workspace,
			SourceRecordID:  "pattern-2",
			TargetSkillName: "weather",
			DraftType:       evolution.DraftTypeWorkflow,
			ChangeKind:      evolution.ChangeKindReplace,
			HumanSummary:    "Replace weather body",
			BodyOrPatch:     "---\nname: weather\ndescription: weather helper\n---\n# Weather\nNew risky body.\n",
			Status:          evolution.DraftStatusAccepted,
		},
	}); err != nil {
		t.Fatalf("SaveDrafts: %v", err)
	}

	if err := store.SaveProfile(evolution.SkillProfile{
		SkillName:      "weather",
		WorkspaceID:    workspace,
		CurrentVersion: "draft-new",
		Status:         evolution.SkillStatusActive,
		Origin:         "evolved",
		HumanSummary:   "Weather skill",
		LastUsedAt:     time.Unix(1700000000, 0).UTC(),
		UseCount:       3,
		RetentionScore: 0.8,
		VersionHistory: []evolution.SkillVersionEntry{
			{
				Version:   "draft-old",
				Action:    "create",
				Timestamp: time.Unix(1699990000, 0).UTC(),
				DraftID:   "draft-old",
				Summary:   "old stable",
			},
			{
				Version:   "draft-new",
				Action:    "replace",
				Timestamp: time.Unix(1700000000, 0).UTC(),
				DraftID:   "draft-new",
				Summary:   "new risky",
			},
		},
	}); err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}

	cmd := newRollbackCommand()
	cmd.SetArgs([]string{"weather"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	data, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != original {
		t.Fatalf("rollback did not restore original body:\n%s", string(data))
	}

	profile, err := store.LoadProfile("weather")
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	if profile.CurrentVersion != "draft-old" {
		t.Fatalf("CurrentVersion = %q, want draft-old", profile.CurrentVersion)
	}
	if len(profile.VersionHistory) != 3 {
		t.Fatalf("len(VersionHistory) = %d, want 3", len(profile.VersionHistory))
	}
	last := profile.VersionHistory[len(profile.VersionHistory)-1]
	if last.Action != "manual_rollback" {
		t.Fatalf("Action = %q, want manual_rollback", last.Action)
	}
	if !last.Rollback {
		t.Fatal("expected rollback history entry")
	}

	drafts, err := store.LoadDrafts()
	if err != nil {
		t.Fatalf("LoadDrafts: %v", err)
	}
	if len(drafts) != 1 {
		t.Fatalf("len(drafts) = %d, want 1", len(drafts))
	}
	if drafts[0].Status != evolution.DraftStatusQuarantined {
		t.Fatalf("draft status = %q, want quarantined after rollback", drafts[0].Status)
	}
	if len(drafts[0].ReviewNotes) == 0 || !strings.Contains(strings.Join(drafts[0].ReviewNotes, " "), "rolled back") {
		t.Fatalf("ReviewNotes = %v, want rollback note", drafts[0].ReviewNotes)
	}
}

func TestDraftGeneratorForRunOnce_FallsBackWhenProviderCreationFails(t *testing.T) {
	originalFactory := createEvolutionProvider
	createEvolutionProvider = func(*config.Config) (providers.LLMProvider, string, error) {
		return nil, "", errors.New("provider init failed")
	}
	defer func() {
		createEvolutionProvider = originalFactory
	}()

	workspace := t.TempDir()
	generator := draftGeneratorForRunOnce(&config.Config{}, workspace)

	draft, err := generator.GenerateDraft(t.Context(), evolution.LearningRecord{
		ID:          "rule-1",
		Summary:     "weather native-name path",
		EventCount:  2,
		SuccessRate: 1,
		WinningPath: []string{"weather"},
	}, nil)
	if err != nil {
		t.Fatalf("GenerateDraft: %v", err)
	}
	if draft.TargetSkillName != "weather" {
		t.Fatalf("TargetSkillName = %q, want weather", draft.TargetSkillName)
	}
	if draft.BodyOrPatch == "" {
		t.Fatal("expected fallback template draft body")
	}
}

func TestDraftGeneratorForRunOnce_UsesExplicitModelIDFromProviderFactory(t *testing.T) {
	originalFactory := createEvolutionProvider
	provider := &runOnceProvider{
		defaultModel: "",
		response: &providers.LLMResponse{
			Content: `{"target_skill_name":"weather","draft_type":"shortcut","change_kind":"append","human_summary":"Prefer native-name query first","body_or_patch":"## Start Here\nUse native-name query first."}`,
		},
	}
	createEvolutionProvider = func(*config.Config) (providers.LLMProvider, string, error) {
		return provider, "configured-model-id", nil
	}
	defer func() {
		createEvolutionProvider = originalFactory
	}()

	workspace := t.TempDir()
	generator := draftGeneratorForRunOnce(&config.Config{}, workspace)

	draft, err := generator.GenerateDraft(context.Background(), evolution.LearningRecord{
		ID:          "rule-1",
		Summary:     "weather native-name path",
		EventCount:  2,
		SuccessRate: 1,
		WinningPath: []string{"weather"},
	}, nil)
	if err != nil {
		t.Fatalf("GenerateDraft: %v", err)
	}
	if provider.lastModel != "configured-model-id" {
		t.Fatalf("lastModel = %q, want configured-model-id", provider.lastModel)
	}
	if draft.TargetSkillName != "weather" {
		t.Fatalf("TargetSkillName = %q, want weather", draft.TargetSkillName)
	}
	if draft.HumanSummary != "Prefer native-name query first" {
		t.Fatalf("HumanSummary = %q, want %q", draft.HumanSummary, "Prefer native-name query first")
	}
}

func TestPruneAppendsLifecycleHistory(t *testing.T) {
	workspace := t.TempDir()
	configureEvolutionCommandTest(t, workspace, "")
	store := evolution.NewStore(evolution.NewPaths(workspace, ""))

	skillDir := filepath.Join(workspace, "skills", "weather")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# weather\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := store.SaveProfile(evolution.SkillProfile{
		SkillName:      "weather",
		WorkspaceID:    workspace,
		Status:         evolution.SkillStatusArchived,
		Origin:         "evolved",
		HumanSummary:   "weather helper",
		LastUsedAt:     time.Now().Add(-366 * 24 * time.Hour),
		RetentionScore: 0.05,
		VersionHistory: []evolution.SkillVersionEntry{{
			Version:   "v1",
			Action:    "create",
			Timestamp: time.Now().Add(-400 * 24 * time.Hour),
			Summary:   "initial version",
		}},
	}); err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}

	cmd := newPruneCommand()
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	profile, err := store.LoadProfile("weather")
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	if profile.Status != evolution.SkillStatusDeleted {
		t.Fatalf("Status = %q, want %q", profile.Status, evolution.SkillStatusDeleted)
	}
	if len(profile.VersionHistory) != 2 {
		t.Fatalf("len(VersionHistory) = %d, want 2", len(profile.VersionHistory))
	}
	last := profile.VersionHistory[len(profile.VersionHistory)-1]
	if last.Action != "lifecycle:deleted" {
		t.Fatalf("Action = %q, want lifecycle:deleted", last.Action)
	}
	if !strings.Contains(last.Summary, "archived -> deleted") {
		t.Fatalf("Summary = %q, want lifecycle transition summary", last.Summary)
	}
}

func TestPruneScopesProfilesToCurrentWorkspaceInSharedState(t *testing.T) {
	sharedState := filepath.Join(t.TempDir(), "shared-evolution")
	workspace := t.TempDir()
	otherWorkspace := t.TempDir()
	configureEvolutionCommandTest(t, workspace, sharedState)
	store := evolution.NewStore(evolution.NewPaths(workspace, sharedState))

	if err := os.MkdirAll(filepath.Join(workspace, "skills", "mine"), 0o755); err != nil {
		t.Fatalf("MkdirAll current skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "skills", "mine", "SKILL.md"), []byte("# mine\n"), 0o644); err != nil {
		t.Fatalf("WriteFile current skill: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(otherWorkspace, "skills", "theirs"), 0o755); err != nil {
		t.Fatalf("MkdirAll other skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(otherWorkspace, "skills", "theirs", "SKILL.md"), []byte("# theirs\n"), 0o644); err != nil {
		t.Fatalf("WriteFile other skill: %v", err)
	}

	profiles := []evolution.SkillProfile{
		{
			SkillName:      "mine",
			WorkspaceID:    workspace,
			Status:         evolution.SkillStatusArchived,
			Origin:         "evolved",
			LastUsedAt:     time.Now().Add(-366 * 24 * time.Hour),
			RetentionScore: 0.05,
		},
		{
			SkillName:      "theirs",
			WorkspaceID:    otherWorkspace,
			Status:         evolution.SkillStatusArchived,
			Origin:         "evolved",
			LastUsedAt:     time.Now().Add(-366 * 24 * time.Hour),
			RetentionScore: 0.05,
		},
		{
			SkillName:      "legacy",
			Status:         evolution.SkillStatusArchived,
			Origin:         "evolved",
			LastUsedAt:     time.Now().Add(-366 * 24 * time.Hour),
			RetentionScore: 0.05,
		},
	}
	for _, profile := range profiles {
		if err := store.SaveProfile(profile); err != nil {
			t.Fatalf("SaveProfile(%s): %v", profile.SkillName, err)
		}
	}

	cmd := newPruneCommand()
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	mine, err := store.LoadProfile("mine")
	if err != nil {
		t.Fatalf("LoadProfile(mine): %v", err)
	}
	if mine.Status != evolution.SkillStatusDeleted {
		t.Fatalf("mine.Status = %q, want %q", mine.Status, evolution.SkillStatusDeleted)
	}
	if len(mine.VersionHistory) != 1 || mine.VersionHistory[0].Action != "lifecycle:deleted" {
		t.Fatalf("mine.VersionHistory = %+v, want lifecycle delete entry", mine.VersionHistory)
	}

	theirs, err := store.LoadProfile("theirs")
	if err != nil {
		t.Fatalf("LoadProfile(theirs): %v", err)
	}
	if theirs.Status != evolution.SkillStatusArchived {
		t.Fatalf("theirs.Status = %q, want archived", theirs.Status)
	}
	if len(theirs.VersionHistory) != 0 {
		t.Fatalf("theirs.VersionHistory = %+v, want unchanged", theirs.VersionHistory)
	}
	if _, statErr := os.Stat(filepath.Join(otherWorkspace, "skills", "theirs", "SKILL.md")); statErr != nil {
		t.Fatalf("other workspace skill should remain, stat err = %v", statErr)
	}

	legacy, err := store.LoadProfile("legacy")
	if err != nil {
		t.Fatalf("LoadProfile(legacy): %v", err)
	}
	if legacy.Status != evolution.SkillStatusArchived {
		t.Fatalf("legacy.Status = %q, want archived", legacy.Status)
	}
	if len(legacy.VersionHistory) != 0 {
		t.Fatalf("legacy.VersionHistory = %+v, want unchanged", legacy.VersionHistory)
	}
}

func TestPruneDoesNotMutateProfileWhenDeleteActionFails(t *testing.T) {
	workspace := t.TempDir()
	configureEvolutionCommandTest(t, workspace, "")
	paths := evolution.NewPaths(workspace, "")
	store := evolution.NewStore(paths)

	legacyProfile := evolution.SkillProfile{
		SkillName:      "../escape",
		WorkspaceID:    workspace,
		Status:         evolution.SkillStatusArchived,
		Origin:         "evolved",
		LastUsedAt:     time.Now().Add(-366 * 24 * time.Hour),
		RetentionScore: 0.05,
		VersionHistory: []evolution.SkillVersionEntry{{
			Version:   "v1",
			Action:    "create",
			Timestamp: time.Now().Add(-400 * 24 * time.Hour),
			Summary:   "initial version",
		}},
	}
	if err := os.MkdirAll(paths.ProfilesDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	data, err := json.MarshalIndent(legacyProfile, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent: %v", err)
	}
	if err := os.WriteFile(filepath.Join(paths.ProfilesDir, "weather.json"), data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cmd := newPruneCommand()
	err = cmd.Execute()
	if err == nil {
		t.Fatal("expected prune to fail when delete action cannot resolve skill path")
	}

	profile, loadErr := store.LoadProfile("weather")
	if loadErr != nil {
		t.Fatalf("LoadProfile: %v", loadErr)
	}
	if profile.Status != evolution.SkillStatusArchived {
		t.Fatalf("Status = %q, want %q", profile.Status, evolution.SkillStatusArchived)
	}
	if len(profile.VersionHistory) != 1 {
		t.Fatalf("len(VersionHistory) = %d, want 1", len(profile.VersionHistory))
	}
}

func TestStatusShowsDraftAndProfileDistributions(t *testing.T) {
	workspace := t.TempDir()
	configureEvolutionCommandTest(t, workspace, "")
	store := evolution.NewStore(evolution.NewPaths(workspace, ""))

	now := time.Unix(1700000000, 0).UTC()
	drafts := []evolution.SkillDraft{
		{ID: "d1", WorkspaceID: workspace, TargetSkillName: "skill-a", HumanSummary: "first candidate", ChangeKind: evolution.ChangeKindAppend, Status: evolution.DraftStatusCandidate, UpdatedAt: timePtr(now.Add(-2 * time.Hour))},
		{ID: "d2", WorkspaceID: workspace, TargetSkillName: "skill-b", HumanSummary: "middle quarantine", ChangeKind: evolution.ChangeKindReplace, Status: evolution.DraftStatusQuarantined, UpdatedAt: timePtr(now.Add(-1 * time.Hour))},
		{ID: "d3", WorkspaceID: workspace, TargetSkillName: "skill-c", HumanSummary: "latest accepted", ChangeKind: evolution.ChangeKindCreate, Status: evolution.DraftStatusAccepted, UpdatedAt: timePtr(now)},
	}
	if err := store.SaveDrafts(drafts); err != nil {
		t.Fatalf("SaveDrafts: %v", err)
	}

	profiles := []evolution.SkillProfile{
		{SkillName: "skill-a", WorkspaceID: workspace, Status: evolution.SkillStatusActive, CurrentVersion: "v-a", UseCount: 5, ChangeReason: "active reason", LastUsedAt: now.Add(-3 * time.Hour)},
		{SkillName: "skill-b", WorkspaceID: workspace, Status: evolution.SkillStatusCold, CurrentVersion: "v-b", UseCount: 2, ChangeReason: "cold reason", LastUsedAt: now.Add(-2 * time.Hour)},
		{SkillName: "skill-c", WorkspaceID: workspace, Status: evolution.SkillStatusArchived, CurrentVersion: "v-c", UseCount: 1, ChangeReason: "archived reason", LastUsedAt: now},
		{SkillName: "skill-d", WorkspaceID: workspace, Status: evolution.SkillStatusDeleted, CurrentVersion: "v-d", UseCount: 0, ChangeReason: "deleted reason", LastUsedAt: now.Add(-4 * time.Hour)},
	}
	for _, profile := range profiles {
		if err := store.SaveProfile(profile); err != nil {
			t.Fatalf("SaveProfile: %v", err)
		}
	}

	cmd := newStatusCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := out.String()
	for _, want := range []string{
		"drafts=3",
		"drafts_by_status=candidate:1 quarantined:1 accepted:1",
		"profiles=4",
		"profiles_by_status=active:1 cold:1 archived:1 deleted:1",
		"draft_items:",
		"id=d3 status=accepted target=skill-c change=create summary=latest accepted",
		"profile_items:",
		"skill=skill-c status=archived version=v-c uses=1 reason=archived reason",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}

func TestStatusScopesCountsToCurrentWorkspaceInSharedState(t *testing.T) {
	sharedState := filepath.Join(t.TempDir(), "shared-evolution")
	workspace := t.TempDir()
	otherWorkspace := t.TempDir()
	configureEvolutionCommandTest(t, workspace, sharedState)
	store := evolution.NewStore(evolution.NewPaths(workspace, sharedState))

	drafts := []evolution.SkillDraft{
		{ID: "d1", WorkspaceID: workspace, TargetSkillName: "skill-a", Status: evolution.DraftStatusCandidate},
		{ID: "d2", WorkspaceID: workspace, TargetSkillName: "skill-b", Status: evolution.DraftStatusAccepted},
		{ID: "d3", WorkspaceID: otherWorkspace, TargetSkillName: "skill-c", Status: evolution.DraftStatusQuarantined},
		{ID: "d4", TargetSkillName: "legacy-draft", Status: evolution.DraftStatusCandidate},
	}
	if err := store.SaveDrafts(drafts); err != nil {
		t.Fatalf("SaveDrafts: %v", err)
	}

	profiles := []evolution.SkillProfile{
		{SkillName: "skill-a", WorkspaceID: workspace, Status: evolution.SkillStatusActive},
		{SkillName: "skill-b", WorkspaceID: workspace, Status: evolution.SkillStatusCold},
		{SkillName: "skill-c", WorkspaceID: otherWorkspace, Status: evolution.SkillStatusDeleted},
		{SkillName: "legacy", Status: evolution.SkillStatusArchived},
	}
	for _, profile := range profiles {
		if err := store.SaveProfile(profile); err != nil {
			t.Fatalf("SaveProfile(%s): %v", profile.SkillName, err)
		}
	}

	cmd := newStatusCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := out.String()
	for _, want := range []string{
		"drafts=2",
		"drafts_by_status=candidate:1 quarantined:0 accepted:1",
		"profiles=2",
		"profiles_by_status=active:1 cold:1 archived:0 deleted:0",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}

func TestApplyCommandAddsExplicitManualAuditInfo(t *testing.T) {
	workspace := t.TempDir()
	configureEvolutionCommandTest(t, workspace, "")
	store := evolution.NewStore(evolution.NewPaths(workspace, ""))

	if err := store.SaveDrafts([]evolution.SkillDraft{
		{
			ID:              "draft-apply-audit",
			WorkspaceID:     workspace,
			SourceRecordID:  "pattern-1",
			TargetSkillName: "weather",
			DraftType:       evolution.DraftTypeShortcut,
			ChangeKind:      evolution.ChangeKindCreate,
			HumanSummary:    "Create weather shortcut",
			BodyOrPatch:     "---\nname: weather\ndescription: weather helper\n---\n# Weather\n## Start Here\nUse native-name query first.\n",
			Status:          evolution.DraftStatusCandidate,
		},
	}); err != nil {
		t.Fatalf("SaveDrafts: %v", err)
	}

	cmd := newApplyCommand()
	cmd.SetArgs([]string{"draft-apply-audit"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	drafts, err := store.LoadDrafts()
	if err != nil {
		t.Fatalf("LoadDrafts: %v", err)
	}
	if len(drafts) != 1 {
		t.Fatalf("len(drafts) = %d, want 1", len(drafts))
	}
	if len(drafts[0].ReviewNotes) == 0 || !strings.Contains(strings.Join(drafts[0].ReviewNotes, " "), "manually applied via CLI") {
		t.Fatalf("ReviewNotes = %v, want manual apply note", drafts[0].ReviewNotes)
	}

	profile, err := store.LoadProfile("weather")
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	last := profile.VersionHistory[len(profile.VersionHistory)-1]
	if last.Action != "manual_apply:create" {
		t.Fatalf("Action = %q, want manual_apply:create", last.Action)
	}
	if !strings.Contains(last.Summary, "manual CLI apply") {
		t.Fatalf("Summary = %q, want manual CLI apply summary", last.Summary)
	}
}

func TestRollbackCommandAddsExplicitManualAuditInfo(t *testing.T) {
	workspace := t.TempDir()
	configureEvolutionCommandTest(t, workspace, "")
	store := evolution.NewStore(evolution.NewPaths(workspace, ""))

	skillDir := filepath.Join(workspace, "skills", "weather")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	skillPath := filepath.Join(skillDir, "SKILL.md")
	original := "---\nname: weather\ndescription: weather helper\n---\n# Weather\nOld stable body.\n"
	if err := os.WriteFile(skillPath, []byte(original), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	applier := evolution.NewApplier(evolution.NewPaths(workspace, ""), func() time.Time {
		return time.Unix(1700000000, 0).UTC()
	})
	if err := applier.ApplyDraft(context.Background(), workspace, evolution.SkillDraft{
		ID:              "draft-risky",
		WorkspaceID:     workspace,
		SourceRecordID:  "pattern-2",
		TargetSkillName: "weather",
		DraftType:       evolution.DraftTypeWorkflow,
		ChangeKind:      evolution.ChangeKindReplace,
		HumanSummary:    "Replace weather body",
		BodyOrPatch:     "---\nname: weather\ndescription: weather helper\n---\n# Weather\nRisky body.\n",
		Status:          evolution.DraftStatusAccepted,
	}); err != nil {
		t.Fatalf("ApplyDraft: %v", err)
	}

	if err := store.SaveDrafts([]evolution.SkillDraft{{
		ID:              "draft-risky",
		WorkspaceID:     workspace,
		SourceRecordID:  "pattern-2",
		TargetSkillName: "weather",
		DraftType:       evolution.DraftTypeWorkflow,
		ChangeKind:      evolution.ChangeKindReplace,
		HumanSummary:    "Replace weather body",
		BodyOrPatch:     "---\nname: weather\ndescription: weather helper\n---\n# Weather\nRisky body.\n",
		Status:          evolution.DraftStatusAccepted,
	}}); err != nil {
		t.Fatalf("SaveDrafts: %v", err)
	}
	if err := store.SaveProfile(evolution.SkillProfile{
		SkillName:      "weather",
		WorkspaceID:    workspace,
		CurrentVersion: "draft-risky",
		Status:         evolution.SkillStatusActive,
		Origin:         "evolved",
		HumanSummary:   "Weather skill",
		LastUsedAt:     time.Unix(1700000000, 0).UTC(),
		VersionHistory: []evolution.SkillVersionEntry{
			{Version: "draft-old", Action: "create", Timestamp: time.Unix(1699990000, 0).UTC(), DraftID: "draft-old", Summary: "old stable"},
			{Version: "draft-risky", Action: "replace", Timestamp: time.Unix(1700000000, 0).UTC(), DraftID: "draft-risky", Summary: "new risky"},
		},
	}); err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}

	cmd := newRollbackCommand()
	cmd.SetArgs([]string{"weather"})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	profile, err := store.LoadProfile("weather")
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	last := profile.VersionHistory[len(profile.VersionHistory)-1]
	if last.Action != "manual_rollback" {
		t.Fatalf("Action = %q, want manual_rollback", last.Action)
	}
	if !strings.Contains(last.Summary, "manual CLI rollback") {
		t.Fatalf("Summary = %q, want manual rollback summary", last.Summary)
	}
	if last.DraftID != "draft-risky" {
		t.Fatalf("DraftID = %q, want draft-risky", last.DraftID)
	}

	drafts, err := store.LoadDrafts()
	if err != nil {
		t.Fatalf("LoadDrafts: %v", err)
	}
	if len(drafts) != 1 {
		t.Fatalf("len(drafts) = %d, want 1", len(drafts))
	}
	if len(drafts[0].ReviewNotes) == 0 || !strings.Contains(strings.Join(drafts[0].ReviewNotes, " "), "backup=") {
		t.Fatalf("ReviewNotes = %v, want backup path note", drafts[0].ReviewNotes)
	}
}

func configureEvolutionCommandTest(t *testing.T, workspace, stateDir string) {
	t.Helper()

	configPath := filepath.Join(t.TempDir(), "config.json")
	t.Setenv(config.EnvConfig, configPath)
	t.Setenv(config.EnvHome, t.TempDir())

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace: workspace,
			},
		},
		Evolution: config.EvolutionConfig{
			Enabled:  true,
			Mode:     "observe",
			StateDir: stateDir,
		},
	}
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
}

type runOnceProvider struct {
	response     *providers.LLMResponse
	err          error
	defaultModel string
	lastModel    string
}

func (p *runOnceProvider) Chat(
	_ context.Context,
	_ []providers.Message,
	_ []providers.ToolDefinition,
	model string,
	_ map[string]any,
) (*providers.LLMResponse, error) {
	p.lastModel = model
	return p.response, p.err
}

func (p *runOnceProvider) GetDefaultModel() string {
	return p.defaultModel
}
