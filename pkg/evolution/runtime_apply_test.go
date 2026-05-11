package evolution_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/evolution"
)

func TestRuntime_RunColdPathOnce_ApplyModeWritesSkillAndProfile(t *testing.T) {
	root := t.TempDir()
	store := evolution.NewStore(evolution.NewPaths(root, ""))

	rule := evolution.LearningRecord{
		ID:          "rule-1",
		Kind:        evolution.RecordKindRule,
		WorkspaceID: root,
		CreatedAt:   time.Unix(1700000000, 0).UTC(),
		Summary:     "weather native-name path",
		Status:      evolution.RecordStatus("ready"),
		EventCount:  4,
	}
	if err := store.AppendLearningRecords([]evolution.LearningRecord{rule}); err != nil {
		t.Fatalf("AppendLearningRecords: %v", err)
	}

	rt, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config: config.EvolutionConfig{Enabled: true, Mode: "apply"},
		Now:    func() time.Time { return time.Unix(1700001000, 0).UTC() },
		Store:  store,
		Applier: evolution.NewApplier(evolution.NewPaths(root, ""), func() time.Time {
			return time.Unix(1700001000, 0).UTC()
		}),
		DraftGenerator: stubDraftGenerator{
			draft: evolution.SkillDraft{
				ID:              "draft-1",
				WorkspaceID:     root,
				SourceRecordID:  "rule-1",
				TargetSkillName: "weather",
				DraftType:       evolution.DraftTypeShortcut,
				ChangeKind:      evolution.ChangeKindCreate,
				HumanSummary:    "weather helper",
				IntendedUseCases: []string{
					"weather native-name path",
				},
				PreferredEntryPath: []string{"weather"},
				AvoidPatterns:      []string{"avoid translating city names before querying weather"},
				BodyOrPatch:        "---\nname: weather\ndescription: weather helper\n---\n# Weather\n## Start Here\nUse native-name query first.\n",
			},
		},
		Organizer:      evolution.NewOrganizer(evolution.OrganizerOptions{MinCaseCount: 3, MinSuccessRate: 0.7}),
		SkillsRecaller: evolution.NewSkillsRecaller(root),
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	if runErr := rt.RunColdPathOnce(context.Background(), root); runErr != nil {
		t.Fatalf("RunColdPathOnce: %v", runErr)
	}

	skillPath := filepath.Join(root, "skills", "weather", "SKILL.md")
	if _, statErr := os.Stat(skillPath); statErr != nil {
		t.Fatalf("expected skill file: %v", statErr)
	}

	profile, err := store.LoadProfile("weather")
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	if profile.Status != evolution.SkillStatusActive {
		t.Fatalf("Status = %q, want %q", profile.Status, evolution.SkillStatusActive)
	}
	if profile.CurrentVersion == "" {
		t.Fatal("CurrentVersion should not be empty")
	}
	if profile.ChangeReason != "weather helper" {
		t.Fatalf("ChangeReason = %q, want weather helper", profile.ChangeReason)
	}
	if len(profile.IntendedUseCases) != 1 || profile.IntendedUseCases[0] != "weather native-name path" {
		t.Fatalf("IntendedUseCases = %v, want [weather native-name path]", profile.IntendedUseCases)
	}
	if len(profile.PreferredEntryPath) != 1 || profile.PreferredEntryPath[0] != "weather" {
		t.Fatalf("PreferredEntryPath = %v, want [weather]", profile.PreferredEntryPath)
	}
	if len(profile.AvoidPatterns) != 1 ||
		profile.AvoidPatterns[0] != "avoid translating city names before querying weather" {
		t.Fatalf("AvoidPatterns = %v, want populated metadata", profile.AvoidPatterns)
	}

	drafts, err := store.LoadDrafts()
	if err != nil {
		t.Fatalf("LoadDrafts: %v", err)
	}
	if len(drafts) != 1 {
		t.Fatalf("len(drafts) = %d, want 1", len(drafts))
	}
	if drafts[0].Status != evolution.DraftStatusAccepted {
		t.Fatalf("draft status = %q, want %q", drafts[0].Status, evolution.DraftStatusAccepted)
	}
}

func TestRuntime_RunColdPathOnce_DraftModeKeepsCandidateDraft(t *testing.T) {
	root := t.TempDir()
	store := evolution.NewStore(evolution.NewPaths(root, ""))

	rule := evolution.LearningRecord{
		ID:          "rule-1",
		Kind:        evolution.RecordKindRule,
		WorkspaceID: root,
		CreatedAt:   time.Unix(1700000000, 0).UTC(),
		Summary:     "weather native-name path",
		Status:      evolution.RecordStatus("ready"),
		EventCount:  4,
	}
	if err := store.AppendLearningRecords([]evolution.LearningRecord{rule}); err != nil {
		t.Fatalf("AppendLearningRecords: %v", err)
	}

	rt, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config: config.EvolutionConfig{Enabled: true, Mode: "draft"},
		Now:    func() time.Time { return time.Unix(1700001000, 0).UTC() },
		Store:  store,
		Applier: evolution.NewApplier(evolution.NewPaths(root, ""), func() time.Time {
			return time.Unix(1700001000, 0).UTC()
		}),
		DraftGenerator: stubDraftGenerator{
			draft: evolution.SkillDraft{
				ID:              "draft-1",
				WorkspaceID:     root,
				SourceRecordID:  "rule-1",
				TargetSkillName: "weather",
				DraftType:       evolution.DraftTypeShortcut,
				ChangeKind:      evolution.ChangeKindCreate,
				HumanSummary:    "weather helper",
				BodyOrPatch:     "---\nname: weather\ndescription: weather helper\n---\n# Weather\n## Start Here\nUse native-name query first.\n",
			},
		},
		Organizer:      evolution.NewOrganizer(evolution.OrganizerOptions{MinCaseCount: 3, MinSuccessRate: 0.7}),
		SkillsRecaller: evolution.NewSkillsRecaller(root),
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	if runErr := rt.RunColdPathOnce(context.Background(), root); runErr != nil {
		t.Fatalf("RunColdPathOnce: %v", runErr)
	}

	if _, statErr := os.Stat(filepath.Join(root, "skills", "weather", "SKILL.md")); !os.IsNotExist(statErr) {
		t.Fatalf("expected no applied skill file, got err=%v", statErr)
	}
	if _, loadErr := store.LoadProfile("weather"); !os.IsNotExist(loadErr) {
		t.Fatalf("expected no profile, got err=%v", loadErr)
	}

	drafts, err := store.LoadDrafts()
	if err != nil {
		t.Fatalf("LoadDrafts: %v", err)
	}
	if len(drafts) != 1 {
		t.Fatalf("len(drafts) = %d, want 1", len(drafts))
	}
	if drafts[0].Status != evolution.DraftStatusCandidate {
		t.Fatalf("draft status = %q, want %q", drafts[0].Status, evolution.DraftStatusCandidate)
	}
}

func TestRuntime_RunColdPathOnce_DraftModeRefreshesExistingCandidateWithEvidence(t *testing.T) {
	root := t.TempDir()
	store := evolution.NewStore(evolution.NewPaths(root, ""))
	for _, source := range []struct {
		name string
		body string
	}{
		{name: "three-one-theorem", body: "Add 31 to the input value."},
		{name: "four-two-theorem", body: "Add 42 to the current value."},
		{name: "five-three-theorem", body: "Subtract 53 from the current value."},
	} {
		skillPath := filepath.Join(root, "skills", source.name, "SKILL.md")
		if err := os.MkdirAll(filepath.Dir(skillPath), 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		content := "---\nname: " + source.name + "\ndescription: theorem helper\n---\n# " + source.name + "\n" + source.body + "\n"
		if err := os.WriteFile(skillPath, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}

	success := true
	if err := store.SaveTaskRecords([]evolution.LearningRecord{{
		ID:             "task-1",
		Kind:           evolution.RecordKindTask,
		WorkspaceID:    root,
		CreatedAt:      time.Unix(1700000000, 0).UTC(),
		Summary:        "调用三一定理计算100",
		FinalOutput:    "100 + 31 = 131; 131 + 42 = 173; 173 - 53 = 120",
		Status:         evolution.RecordStatus("clustered"),
		Success:        &success,
		UsedSkillNames: []string{"three-one-theorem", "four-two-theorem", "five-three-theorem"},
	}}); err != nil {
		t.Fatalf("SaveTaskRecords: %v", err)
	}
	if err := store.SavePatternRecords([]evolution.LearningRecord{{
		ID:            "pattern-1",
		Kind:          evolution.RecordKindPattern,
		WorkspaceID:   root,
		CreatedAt:     time.Unix(1700000001, 0).UTC(),
		Summary:       "调用三一定理计算100",
		Status:        evolution.RecordStatus("ready"),
		TaskRecordIDs: []string{"task-1"},
	}}); err != nil {
		t.Fatalf("SavePatternRecords: %v", err)
	}
	if err := store.SaveDrafts([]evolution.SkillDraft{
		{
			ID:              "draft-pattern-1",
			WorkspaceID:     root,
			SourceRecordID:  "pattern-1",
			TargetSkillName: "learned-skill",
			DraftType:       evolution.DraftTypeShortcut,
			ChangeKind:      evolution.ChangeKindCreate,
			HumanSummary:    "old generic draft",
			BodyOrPatch:     "---\nname: learned-skill\ndescription: old\n---\n# Learned Skill\n\nNo explicit winning path was recorded.\n",
			Status:          evolution.DraftStatusCandidate,
		},
	}); err != nil {
		t.Fatalf("SaveDrafts: %v", err)
	}

	rt, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config: config.EvolutionConfig{Enabled: true, Mode: "draft"},
		Now:    func() time.Time { return time.Unix(1700001000, 0).UTC() },
		Store:  store,
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	if runErr := rt.RunColdPathOnce(context.Background(), root); runErr != nil {
		t.Fatalf("RunColdPathOnce: %v", runErr)
	}

	drafts, err := store.LoadDrafts()
	if err != nil {
		t.Fatalf("LoadDrafts: %v", err)
	}
	if len(drafts) != 1 {
		t.Fatalf("len(drafts) = %d, want 1", len(drafts))
	}
	if drafts[0].Status != evolution.DraftStatusCandidate {
		t.Fatalf("draft status = %q, want candidate", drafts[0].Status)
	}
	for _, want := range []string{
		"calculate-100-via-theorems",
		"Add 31 to the input value",
		"Subtract 53 from the current value",
		"100 + 31 = 131",
		"task-1",
	} {
		if !strings.Contains(drafts[0].BodyOrPatch, want) && drafts[0].TargetSkillName != want {
			t.Fatalf("refreshed draft missing %q:\nname=%s\n%s", want, drafts[0].TargetSkillName, drafts[0].BodyOrPatch)
		}
	}
}

func TestRuntime_RunColdPathOnce_ApplyModeAppliesExistingCandidateDraft(t *testing.T) {
	root := t.TempDir()
	store := evolution.NewStore(evolution.NewPaths(root, ""))

	rule := evolution.LearningRecord{
		ID:          "rule-1",
		Kind:        evolution.RecordKindRule,
		WorkspaceID: root,
		CreatedAt:   time.Unix(1700000000, 0).UTC(),
		Summary:     "weather native-name path",
		Status:      evolution.RecordStatus("ready"),
		EventCount:  4,
	}
	if err := store.AppendLearningRecords([]evolution.LearningRecord{rule}); err != nil {
		t.Fatalf("AppendLearningRecords: %v", err)
	}
	if err := store.SaveDrafts([]evolution.SkillDraft{
		{
			ID:              "draft-1",
			WorkspaceID:     root,
			SourceRecordID:  "rule-1",
			TargetSkillName: "weather",
			DraftType:       evolution.DraftTypeShortcut,
			ChangeKind:      evolution.ChangeKindCreate,
			HumanSummary:    "weather helper",
			BodyOrPatch:     "---\nname: weather\ndescription: weather helper\n---\n# Weather\n## Start Here\nUse native-name query first.\n",
			Status:          evolution.DraftStatusCandidate,
		},
	}); err != nil {
		t.Fatalf("SaveDrafts: %v", err)
	}

	rt, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config: config.EvolutionConfig{Enabled: true, Mode: "apply"},
		Now:    func() time.Time { return time.Unix(1700001000, 0).UTC() },
		Store:  store,
		Applier: evolution.NewApplier(evolution.NewPaths(root, ""), func() time.Time {
			return time.Unix(1700001000, 0).UTC()
		}),
		DraftGenerator: stubDraftGenerator{
			draft: evolution.SkillDraft{
				ID:              "draft-unused",
				WorkspaceID:     root,
				SourceRecordID:  "rule-1",
				TargetSkillName: "unused-weather",
				DraftType:       evolution.DraftTypeShortcut,
				ChangeKind:      evolution.ChangeKindCreate,
				HumanSummary:    "unused",
				BodyOrPatch:     "---\nname: unused-weather\ndescription: unused\n---\n# Unused\n",
			},
		},
		Organizer:      evolution.NewOrganizer(evolution.OrganizerOptions{MinCaseCount: 3, MinSuccessRate: 0.7}),
		SkillsRecaller: evolution.NewSkillsRecaller(root),
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	if runErr := rt.RunColdPathOnce(context.Background(), root); runErr != nil {
		t.Fatalf("RunColdPathOnce: %v", runErr)
	}

	if _, statErr := os.Stat(filepath.Join(root, "skills", "weather", "SKILL.md")); statErr != nil {
		t.Fatalf("expected existing candidate to be applied: %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(root, "skills", "unused-weather", "SKILL.md")); !os.IsNotExist(statErr) {
		t.Fatalf("expected source rule to stay skipped after applying existing draft, got err=%v", statErr)
	}
	profile, err := store.LoadProfile("weather")
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	if profile.CurrentVersion != "draft-1" {
		t.Fatalf("CurrentVersion = %q, want draft-1", profile.CurrentVersion)
	}

	drafts, err := store.LoadDrafts()
	if err != nil {
		t.Fatalf("LoadDrafts: %v", err)
	}
	if len(drafts) != 1 {
		t.Fatalf("len(drafts) = %d, want 1", len(drafts))
	}
	if drafts[0].Status != evolution.DraftStatusAccepted {
		t.Fatalf("draft status = %q, want %q", drafts[0].Status, evolution.DraftStatusAccepted)
	}
}

func TestRuntime_RunColdPathOnce_ApplyModeSkipsOrphanCandidateDraft(t *testing.T) {
	root := t.TempDir()
	store := evolution.NewStore(evolution.NewPaths(root, ""))

	rule := evolution.LearningRecord{
		ID:          "rule-1",
		Kind:        evolution.RecordKindRule,
		WorkspaceID: root,
		CreatedAt:   time.Unix(1700000000, 0).UTC(),
		Summary:     "weather native-name path",
		Status:      evolution.RecordStatus("ready"),
		EventCount:  4,
	}
	if err := store.AppendLearningRecords([]evolution.LearningRecord{rule}); err != nil {
		t.Fatalf("AppendLearningRecords: %v", err)
	}
	if err := store.SaveDrafts([]evolution.SkillDraft{
		{
			ID:              "draft-orphan",
			WorkspaceID:     root,
			SourceRecordID:  "missing-rule",
			TargetSkillName: "orphan-weather",
			DraftType:       evolution.DraftTypeShortcut,
			ChangeKind:      evolution.ChangeKindCreate,
			HumanSummary:    "orphan weather helper",
			BodyOrPatch:     "---\nname: orphan-weather\ndescription: orphan weather helper\n---\n# Orphan Weather\n## Start Here\nUse stale guidance.\n",
			Status:          evolution.DraftStatusCandidate,
		},
	}); err != nil {
		t.Fatalf("SaveDrafts: %v", err)
	}

	rt, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config: config.EvolutionConfig{Enabled: true, Mode: "apply"},
		Now:    func() time.Time { return time.Unix(1700001000, 0).UTC() },
		Store:  store,
		Applier: evolution.NewApplier(evolution.NewPaths(root, ""), func() time.Time {
			return time.Unix(1700001000, 0).UTC()
		}),
		DraftGenerator: stubDraftGenerator{
			draft: evolution.SkillDraft{
				ID:              "draft-valid",
				WorkspaceID:     root,
				SourceRecordID:  "rule-1",
				TargetSkillName: "valid-weather",
				DraftType:       evolution.DraftTypeShortcut,
				ChangeKind:      evolution.ChangeKindCreate,
				HumanSummary:    "valid weather helper",
				BodyOrPatch:     "---\nname: valid-weather\ndescription: valid weather helper\n---\n# Valid Weather\n",
			},
		},
		Organizer:      evolution.NewOrganizer(evolution.OrganizerOptions{MinCaseCount: 3, MinSuccessRate: 0.7}),
		SkillsRecaller: evolution.NewSkillsRecaller(root),
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	if runErr := rt.RunColdPathOnce(context.Background(), root); runErr != nil {
		t.Fatalf("RunColdPathOnce: %v", runErr)
	}

	if _, statErr := os.Stat(filepath.Join(root, "skills", "orphan-weather", "SKILL.md")); !os.IsNotExist(statErr) {
		t.Fatalf("orphan candidate draft should not be applied, got err=%v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(root, "skills", "valid-weather", "SKILL.md")); statErr != nil {
		t.Fatalf("expected current ready rule draft to be applied: %v", statErr)
	}
	drafts, err := store.LoadDrafts()
	if err != nil {
		t.Fatalf("LoadDrafts: %v", err)
	}
	statusByID := map[string]evolution.DraftStatus{}
	for _, draft := range drafts {
		statusByID[draft.ID] = draft.Status
	}
	if statusByID["draft-orphan"] != evolution.DraftStatusCandidate {
		t.Fatalf("orphan draft status = %q, want candidate", statusByID["draft-orphan"])
	}
}

func TestRuntime_RunColdPathOnce_ApplyModeNormalizesExistingCombinedCandidateDraft(t *testing.T) {
	root := t.TempDir()
	store := evolution.NewStore(evolution.NewPaths(root, ""))
	writeSkillForCombinedShortcutTest(t, root, "three-one-theorem", "Add 31 to the input before continuing.")
	writeSkillForCombinedShortcutTest(t, root, "four-two-theorem", "Add 42 to the intermediate result.")
	writeSkillForCombinedShortcutTest(t, root, "five-three-theorem", "Subtract 53 to produce the final result.")

	rule := evolution.LearningRecord{
		ID:          "rule-1",
		Kind:        evolution.RecordKindRule,
		WorkspaceID: root,
		CreatedAt:   time.Unix(1700000000, 0).UTC(),
		Summary:     "calculate 100",
		Status:      evolution.RecordStatus("ready"),
		EventCount:  4,
		SuccessRate: 1,
		WinningPath: []string{"three-one-theorem", "four-two-theorem", "five-three-theorem"},
	}
	if err := store.AppendLearningRecords([]evolution.LearningRecord{rule}); err != nil {
		t.Fatalf("AppendLearningRecords: %v", err)
	}
	if err := store.SaveDrafts([]evolution.SkillDraft{
		{
			ID:              "draft-1",
			WorkspaceID:     root,
			SourceRecordID:  "rule-1",
			TargetSkillName: "five-three-theorem",
			DraftType:       evolution.DraftTypeShortcut,
			ChangeKind:      evolution.ChangeKindAppend,
			HumanSummary:    "messy generated combined skill",
			BodyOrPatch: strings.Join([]string{
				"Prefer the full theorem chain directly.",
				"",
				"## Component Skill Breakdown",
				"messy raw component dump should be removed before apply.",
				"",
				"## Learned Shortcut",
				"Net effect: input + 20.",
			}, "\n"),
			Status: evolution.DraftStatusCandidate,
		},
	}); err != nil {
		t.Fatalf("SaveDrafts: %v", err)
	}

	rt, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config: config.EvolutionConfig{Enabled: true, Mode: "apply"},
		Now:    func() time.Time { return time.Unix(1700001000, 0).UTC() },
		Store:  store,
		Applier: evolution.NewApplier(
			evolution.NewPaths(root, ""),
			func() time.Time { return time.Unix(1700001000, 0).UTC() },
		),
		DraftGenerator: stubDraftGenerator{},
		Organizer:      evolution.NewOrganizer(evolution.OrganizerOptions{MinCaseCount: 3, MinSuccessRate: 0.7}),
		SkillsRecaller: evolution.NewSkillsRecaller(root),
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	if runErr := rt.RunColdPathOnce(context.Background(), root); runErr != nil {
		t.Fatalf("RunColdPathOnce: %v", runErr)
	}

	data, err := os.ReadFile(filepath.Join(root, "skills", "calculate-100-via-theorems", "SKILL.md"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "## Procedure Details") || !strings.Contains(content, "## Procedure") {
		t.Fatalf("expected clean combined skill sections:\n%s", content)
	}
	if strings.Contains(content, "Learned") || strings.Contains(content, "Source Evidence") {
		t.Fatalf("deployed skill should not expose learning traces:\n%s", content)
	}
	if strings.Contains(content, "messy raw component dump") ||
		strings.Contains(content, "## Component Skill Breakdown") {
		t.Fatalf("expected old verbose draft content to be cleaned:\n%s", content)
	}

	drafts, err := store.LoadDrafts()
	if err != nil {
		t.Fatalf("LoadDrafts: %v", err)
	}
	if len(drafts) != 1 {
		t.Fatalf("len(drafts) = %d, want 1", len(drafts))
	}
	if drafts[0].Status != evolution.DraftStatusAccepted {
		t.Fatalf("draft status = %q, want %q", drafts[0].Status, evolution.DraftStatusAccepted)
	}
	if drafts[0].TargetSkillName != "calculate-100-via-theorems" {
		t.Fatalf("TargetSkillName = %q, want calculate-100-via-theorems", drafts[0].TargetSkillName)
	}
}

func TestRuntime_RunColdPathOnce_ApplyModeRetargetsStableMultiSkillPathIntoCombinedShortcut(t *testing.T) {
	root := t.TempDir()
	store := evolution.NewStore(evolution.NewPaths(root, ""))
	writeSkillForCombinedShortcutTest(t, root, "three-one-theorem", "Add 31 to the input before continuing.")
	writeSkillForCombinedShortcutTest(t, root, "four-two-theorem", "Add 42 to the intermediate result.")
	writeSkillForCombinedShortcutTest(t, root, "five-three-theorem", "Subtract 53 to produce the final result.")

	rule := evolution.LearningRecord{
		ID:          "rule-1",
		Kind:        evolution.RecordKindRule,
		WorkspaceID: root,
		CreatedAt:   time.Unix(1700000000, 0).UTC(),
		Summary:     "calculate 100",
		Status:      evolution.RecordStatus("ready"),
		EventCount:  4,
		SuccessRate: 1,
		WinningPath: []string{"three-one-theorem", "four-two-theorem", "five-three-theorem"},
	}
	if err := store.AppendLearningRecords([]evolution.LearningRecord{rule}); err != nil {
		t.Fatalf("AppendLearningRecords: %v", err)
	}

	rt, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config: config.EvolutionConfig{Enabled: true, Mode: "apply"},
		Now:    func() time.Time { return time.Unix(1700001000, 0).UTC() },
		Store:  store,
		Applier: evolution.NewApplier(evolution.NewPaths(root, ""), func() time.Time {
			return time.Unix(1700001000, 0).UTC()
		}),
		DraftGenerator: stubDraftGenerator{
			draft: evolution.SkillDraft{
				ID:              "draft-1",
				WorkspaceID:     root,
				SourceRecordID:  "rule-1",
				TargetSkillName: "five-three-theorem",
				DraftType:       evolution.DraftTypeShortcut,
				ChangeKind:      evolution.ChangeKindAppend,
				HumanSummary:    "combine the theorem chain into one shortcut skill",
				BodyOrPatch: strings.Join([]string{
					"Prefer the full theorem chain directly.",
					"",
					"## Component Skill Breakdown",
					"messy raw component dump should be removed.",
					"",
					"## Learned Shortcut",
					"Net effect: input + 20.",
				}, "\n"),
			},
		},
		Organizer:      evolution.NewOrganizer(evolution.OrganizerOptions{MinCaseCount: 3, MinSuccessRate: 0.7}),
		SkillsRecaller: evolution.NewSkillsRecaller(root),
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	if runErr := rt.RunColdPathOnce(context.Background(), root); runErr != nil {
		t.Fatalf("RunColdPathOnce: %v", runErr)
	}

	skillPath := filepath.Join(root, "skills", "calculate-100-via-theorems", "SKILL.md")
	data, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "name: calculate-100-via-theorems") {
		t.Fatalf("unexpected content:\n%s", content)
	}
	if !strings.Contains(content, "# Calculate 100 Via Theorems") {
		t.Fatalf("missing synthesized heading:\n%s", content)
	}
	if !strings.Contains(content, "Prefer the full theorem chain directly.") {
		t.Fatalf("missing learned content:\n%s", content)
	}
	if !strings.Contains(content, "## Procedure") {
		t.Fatalf("missing compact procedure:\n%s", content)
	}
	if !strings.Contains(content, "## Procedure Details") {
		t.Fatalf("missing source skill summary:\n%s", content)
	}
	if strings.Contains(content, "Learned") || strings.Contains(content, "Source Evidence") {
		t.Fatalf("deployed skill should not expose learning traces:\n%s", content)
	}
	if !strings.Contains(content, "Add 31 to the input") ||
		!strings.Contains(content, "Subtract 53 to produce the final result") {
		t.Fatalf("missing extracted component skill content:\n%s", content)
	}
	if strings.Contains(content, "Extracted guidance") {
		t.Fatalf("component content should be concise, not raw extracted guidance:\n%s", content)
	}
	if strings.Contains(content, "messy raw component dump") {
		t.Fatalf("learned context should remove verbose component dumps:\n%s", content)
	}
	if !strings.Contains(content, "Use `calculate-100-via-theorems` directly") {
		t.Fatalf("missing direct shortcut guidance:\n%s", content)
	}

	drafts, err := store.LoadDrafts()
	if err != nil {
		t.Fatalf("LoadDrafts: %v", err)
	}
	if len(drafts) != 1 {
		t.Fatalf("len(drafts) = %d, want 1", len(drafts))
	}
	if drafts[0].Status != evolution.DraftStatusAccepted {
		t.Fatalf("draft status = %q, want %q", drafts[0].Status, evolution.DraftStatusAccepted)
	}
	if drafts[0].ChangeKind != evolution.ChangeKindCreate {
		t.Fatalf("ChangeKind = %q, want %q", drafts[0].ChangeKind, evolution.ChangeKindCreate)
	}
	if drafts[0].TargetSkillName != "calculate-100-via-theorems" {
		t.Fatalf("TargetSkillName = %q, want calculate-100-via-theorems", drafts[0].TargetSkillName)
	}
	if len(drafts[0].PreferredEntryPath) != 1 || drafts[0].PreferredEntryPath[0] != "calculate-100-via-theorems" {
		t.Fatalf("PreferredEntryPath = %v, want [calculate-100-via-theorems]", drafts[0].PreferredEntryPath)
	}
	if len(drafts[0].ReviewNotes) == 0 {
		t.Fatal("expected normalization review notes")
	}
}

func TestRuntime_RunColdPathOnce_CombinedShortcutKeepsReadableLongGuidance(t *testing.T) {
	root := t.TempDir()
	store := evolution.NewStore(evolution.NewPaths(root, ""))
	longOperation := strings.Join([]string{
		"Step 1: normalize the incoming number and keep the original input available for reporting.",
		"Step 2: add 31 to the normalized value and record the intermediate value.",
		"Step 3: add 42 to the intermediate value and verify that arithmetic was performed exactly once.",
		"Step 4: subtract 53 from the second intermediate value and return only the final value.",
		"Step 5: if the user asks for explanation, include the compact arithmetic chain without unrelated context.",
	}, " ")
	writeSkillForCombinedShortcutTest(t, root, "three-one-theorem", longOperation)
	writeSkillForCombinedShortcutTest(t, root, "four-two-theorem", "Add 42 to the intermediate result.")

	rule := evolution.LearningRecord{
		ID:          "rule-1",
		Kind:        evolution.RecordKindRule,
		WorkspaceID: root,
		CreatedAt:   time.Unix(1700000000, 0).UTC(),
		Summary:     "calculate with theorem chain",
		Status:      evolution.RecordStatus("ready"),
		EventCount:  4,
		SuccessRate: 1,
		WinningPath: []string{"three-one-theorem", "four-two-theorem"},
	}
	if err := store.AppendLearningRecords([]evolution.LearningRecord{rule}); err != nil {
		t.Fatalf("AppendLearningRecords: %v", err)
	}

	rt, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config: config.EvolutionConfig{Enabled: true, Mode: "apply"},
		Now:    func() time.Time { return time.Unix(1700001000, 0).UTC() },
		Store:  store,
		Applier: evolution.NewApplier(
			evolution.NewPaths(root, ""),
			func() time.Time { return time.Unix(1700001000, 0).UTC() },
		),
		DraftGenerator: stubDraftGenerator{draft: evolution.SkillDraft{
			ID:              "draft-1",
			WorkspaceID:     root,
			SourceRecordID:  "rule-1",
			TargetSkillName: "three-one-theorem",
			DraftType:       evolution.DraftTypeShortcut,
			ChangeKind:      evolution.ChangeKindAppend,
			HumanSummary:    "combine theorem chain",
			BodyOrPatch: strings.Join([]string{
				"Prefer the theorem chain directly.",
				"Include Step A, Step B, Step C, Step D, Step E, Step F, Step G, Step H, Step I, Step J, Step K, Step L, Step M, Step N, Step O, Step P, Step Q, Step R, Step S, Step T, Step U, Step V, Step W, Step X, Step Y, Step Z, and then return the answer.",
				"Finish with a short arithmetic explanation.",
			}, " "),
		}},
		Organizer:      evolution.NewOrganizer(evolution.OrganizerOptions{MinCaseCount: 3, MinSuccessRate: 0.7}),
		SkillsRecaller: evolution.NewSkillsRecaller(root),
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	if runErr := rt.RunColdPathOnce(context.Background(), root); runErr != nil {
		t.Fatalf("RunColdPathOnce: %v", runErr)
	}

	data, err := os.ReadFile(filepath.Join(root, "skills", "calculate-with-theorem-chain-via-theorems", "SKILL.md"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "Step 5: if the user asks for explanation") {
		t.Fatalf("procedure details were cut too aggressively:\n%s", content)
	}
	if !strings.Contains(content, "Step Z, and then return the answer.") {
		t.Fatalf("procedure notes were cut too aggressively:\n%s", content)
	}
}

func writeSkillForCombinedShortcutTest(t *testing.T, root, name, body string) {
	t.Helper()

	skillPath := filepath.Join(root, "skills", name, "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(skillPath), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	content := strings.Join([]string{
		"---",
		"name: " + name,
		"description: test component skill",
		"---",
		"# " + name,
		body,
		"",
	}, "\n")
	if err := os.WriteFile(skillPath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func TestRuntime_RunColdPathOnce_ApplyFailureQuarantinesDraftAndWritesRollbackAudit(t *testing.T) {
	root := t.TempDir()
	store := evolution.NewStore(evolution.NewPaths(root, ""))

	profile := evolution.SkillProfile{
		SkillName:      "weather",
		WorkspaceID:    root,
		CurrentVersion: "v1",
		Status:         evolution.SkillStatusActive,
		Origin:         "evolved",
		HumanSummary:   "weather helper",
		LastUsedAt:     time.Unix(1700000000, 0).UTC(),
		RetentionScore: 1,
		VersionHistory: []evolution.SkillVersionEntry{
			{
				Version:   "v1",
				Action:    "create",
				Timestamp: time.Unix(1700000000, 0).UTC(),
				DraftID:   "draft-old",
				Summary:   "initial",
			},
		},
	}
	if err := store.SaveProfile(profile); err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}

	rule := evolution.LearningRecord{
		ID:          "rule-1",
		Kind:        evolution.RecordKindRule,
		WorkspaceID: root,
		CreatedAt:   time.Unix(1700000000, 0).UTC(),
		Summary:     "weather native-name path",
		Status:      evolution.RecordStatus("ready"),
		EventCount:  4,
	}
	if err := store.AppendLearningRecords([]evolution.LearningRecord{rule}); err != nil {
		t.Fatalf("AppendLearningRecords: %v", err)
	}

	skillDir := filepath.Join(root, "skills", "weather")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	skillPath := filepath.Join(skillDir, "SKILL.md")
	original := "---\nname: weather\ndescription: valid\n---\n# Weather\nold body\n"
	if err := os.WriteFile(skillPath, []byte(original), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	rt, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config: config.EvolutionConfig{Enabled: true, Mode: "apply"},
		Now:    func() time.Time { return time.Unix(1700001000, 0).UTC() },
		Store:  store,
		Applier: evolution.NewApplier(evolution.NewPaths(root, ""), func() time.Time {
			return time.Unix(1700001000, 0).UTC()
		}),
		DraftGenerator: stubDraftGenerator{
			draft: evolution.SkillDraft{
				ID:              "draft-rollback",
				WorkspaceID:     root,
				SourceRecordID:  "rule-1",
				TargetSkillName: "weather",
				DraftType:       evolution.DraftTypeShortcut,
				ChangeKind:      evolution.ChangeKindReplace,
				HumanSummary:    "broken weather helper",
				BodyOrPatch:     "invalid-frontmatter",
			},
		},
		Organizer:      evolution.NewOrganizer(evolution.OrganizerOptions{MinCaseCount: 3, MinSuccessRate: 0.7}),
		SkillsRecaller: evolution.NewSkillsRecaller(root),
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	err = rt.RunColdPathOnce(context.Background(), root)
	if err == nil {
		t.Fatal("expected RunColdPathOnce to fail")
	}
	if !errors.Is(err, evolution.ErrApplyDraftFailed) {
		t.Fatalf("error = %v, want ErrApplyDraftFailed", err)
	}

	drafts, err := store.LoadDrafts()
	if err != nil {
		t.Fatalf("LoadDrafts: %v", err)
	}
	if len(drafts) != 1 {
		t.Fatalf("len(drafts) = %d, want 1", len(drafts))
	}
	if drafts[0].Status != evolution.DraftStatusQuarantined {
		t.Fatalf("draft status = %q, want %q", drafts[0].Status, evolution.DraftStatusQuarantined)
	}
	if len(drafts[0].ScanFindings) == 0 {
		t.Fatal("expected apply error in ScanFindings")
	}

	loadedProfile, err := store.LoadProfile("weather")
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	if len(loadedProfile.VersionHistory) != 2 {
		t.Fatalf("len(VersionHistory) = %d, want 2", len(loadedProfile.VersionHistory))
	}
	last := loadedProfile.VersionHistory[len(loadedProfile.VersionHistory)-1]
	if !last.Rollback {
		t.Fatal("expected rollback audit entry")
	}
	if last.DraftID != "draft-rollback" {
		t.Fatalf("DraftID = %q, want draft-rollback", last.DraftID)
	}

	got, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != original {
		t.Fatalf("skill content changed after runtime rollback:\n%s", string(got))
	}
}

func TestRuntime_RunColdPathOnce_FirstApplyFailureDoesNotCreateGhostProfile(t *testing.T) {
	root := t.TempDir()
	store := evolution.NewStore(evolution.NewPaths(root, ""))

	rule := evolution.LearningRecord{
		ID:          "rule-1",
		Kind:        evolution.RecordKindRule,
		WorkspaceID: root,
		CreatedAt:   time.Unix(1700000000, 0).UTC(),
		Summary:     "weather native-name path",
		Status:      evolution.RecordStatus("ready"),
		EventCount:  4,
	}
	if err := store.AppendLearningRecords([]evolution.LearningRecord{rule}); err != nil {
		t.Fatalf("AppendLearningRecords: %v", err)
	}

	rt, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config: config.EvolutionConfig{Enabled: true, Mode: "apply"},
		Now:    func() time.Time { return time.Unix(1700001000, 0).UTC() },
		Store:  store,
		Applier: evolution.NewApplier(evolution.NewPaths(root, ""), func() time.Time {
			return time.Unix(1700001000, 0).UTC()
		}),
		DraftGenerator: stubDraftGenerator{
			draft: evolution.SkillDraft{
				ID:              "draft-ghost-profile",
				WorkspaceID:     root,
				SourceRecordID:  "rule-1",
				TargetSkillName: "weather",
				DraftType:       evolution.DraftTypeShortcut,
				ChangeKind:      evolution.ChangeKindCreate,
				HumanSummary:    "broken weather helper",
				BodyOrPatch:     "invalid-frontmatter",
			},
		},
		Organizer:      evolution.NewOrganizer(evolution.OrganizerOptions{MinCaseCount: 3, MinSuccessRate: 0.7}),
		SkillsRecaller: evolution.NewSkillsRecaller(root),
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	err = rt.RunColdPathOnce(context.Background(), root)
	if err == nil {
		t.Fatal("expected RunColdPathOnce to fail")
	}
	if !errors.Is(err, evolution.ErrApplyDraftFailed) {
		t.Fatalf("error = %v, want ErrApplyDraftFailed", err)
	}

	if _, loadErr := store.LoadProfile("weather"); !os.IsNotExist(loadErr) {
		t.Fatalf("expected no profile after first apply failure, got err=%v", loadErr)
	}
}

func TestRuntime_RunColdPathOnce_DraftSaveFailureRollsBackAppliedSkill(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory permission behavior differs on Windows")
	}

	root := t.TempDir()
	paths := evolution.NewPaths(root, "")
	store := evolution.NewStore(paths)

	rule := evolution.LearningRecord{
		ID:          "rule-1",
		Kind:        evolution.RecordKindRule,
		WorkspaceID: root,
		CreatedAt:   time.Unix(1700000000, 0).UTC(),
		Summary:     "weather native-name path",
		Status:      evolution.RecordStatus("ready"),
		EventCount:  4,
	}
	if err := store.AppendLearningRecords([]evolution.LearningRecord{rule}); err != nil {
		t.Fatalf("AppendLearningRecords: %v", err)
	}

	if err := os.Chmod(paths.RootDir, 0o555); err != nil {
		t.Fatalf("Chmod(root read-only): %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(paths.RootDir, 0o755)
	})

	rt, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config: config.EvolutionConfig{Enabled: true, Mode: "apply"},
		Now:    func() time.Time { return time.Unix(1700001000, 0).UTC() },
		Store:  store,
		Applier: evolution.NewApplier(paths, func() time.Time {
			return time.Unix(1700001000, 0).UTC()
		}),
		DraftGenerator: stubDraftGenerator{
			draft: evolution.SkillDraft{
				ID:              "draft-save-fail",
				WorkspaceID:     root,
				SourceRecordID:  "rule-1",
				TargetSkillName: "weather",
				DraftType:       evolution.DraftTypeShortcut,
				ChangeKind:      evolution.ChangeKindCreate,
				HumanSummary:    "weather helper",
				BodyOrPatch:     "---\nname: weather\ndescription: weather helper\n---\n# Weather\n## Start Here\nUse native-name query first.\n",
			},
		},
		Organizer:      evolution.NewOrganizer(evolution.OrganizerOptions{MinCaseCount: 3, MinSuccessRate: 0.7}),
		SkillsRecaller: evolution.NewSkillsRecaller(root),
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	err = rt.RunColdPathOnce(context.Background(), root)
	if err == nil {
		t.Fatal("expected RunColdPathOnce to fail")
	}
	if !errors.Is(err, evolution.ErrApplyDraftFailed) {
		t.Fatalf("error = %v, want ErrApplyDraftFailed", err)
	}

	skillPath := filepath.Join(root, "skills", "weather", "SKILL.md")
	if _, statErr := os.Stat(skillPath); !os.IsNotExist(statErr) {
		t.Fatalf("expected applied skill to be rolled back, got err=%v", statErr)
	}
	if _, loadErr := store.LoadProfile("weather"); !os.IsNotExist(loadErr) {
		t.Fatalf("expected no profile after draft save failure, got err=%v", loadErr)
	}
}

func TestRuntime_RunColdPathOnce_AutoRunsLifecycleMaintenance(t *testing.T) {
	root := t.TempDir()
	paths := evolution.NewPaths(root, "")
	store := evolution.NewStore(paths)
	now := time.Unix(1700001000, 0).UTC()

	if err := store.SaveProfile(evolution.SkillProfile{
		SkillName:      "stale-active-skill",
		WorkspaceID:    root,
		Status:         evolution.SkillStatusActive,
		Origin:         "evolved",
		HumanSummary:   "stale active skill",
		LastUsedAt:     now.Add(-91 * 24 * time.Hour),
		RetentionScore: 0.1,
	}); err != nil {
		t.Fatalf("SaveProfile(active): %v", err)
	}

	skillDir := filepath.Join(root, "skills", "stale-archived-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(
		skillPath,
		[]byte("---\nname: stale-archived-skill\ndescription: stale\n---\n# Stale Archived Skill\n"),
		0o644,
	); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := store.SaveProfile(evolution.SkillProfile{
		SkillName:      "stale-archived-skill",
		WorkspaceID:    root,
		Status:         evolution.SkillStatusArchived,
		Origin:         "evolved",
		HumanSummary:   "stale archived skill",
		LastUsedAt:     now.Add(-366 * 24 * time.Hour),
		RetentionScore: 0.05,
	}); err != nil {
		t.Fatalf("SaveProfile(archived): %v", err)
	}

	rt, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config: config.EvolutionConfig{Enabled: true, Mode: "apply"},
		Now:    func() time.Time { return now },
		Store:  store,
		Applier: evolution.NewApplier(paths, func() time.Time {
			return now
		}),
		Organizer:      evolution.NewOrganizer(evolution.OrganizerOptions{MinCaseCount: 3, MinSuccessRate: 0.7}),
		SkillsRecaller: evolution.NewSkillsRecaller(root),
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	if runErr := rt.RunColdPathOnce(context.Background(), root); runErr != nil {
		t.Fatalf("RunColdPathOnce: %v", runErr)
	}

	activeProfile, err := store.LoadProfile("stale-active-skill")
	if err != nil {
		t.Fatalf("LoadProfile(active): %v", err)
	}
	if activeProfile.Status != evolution.SkillStatusCold {
		t.Fatalf("active profile Status = %q, want %q", activeProfile.Status, evolution.SkillStatusCold)
	}
	if len(activeProfile.VersionHistory) != 1 || activeProfile.VersionHistory[0].Action != "lifecycle:cold" {
		t.Fatalf("active profile VersionHistory = %+v, want lifecycle:cold entry", activeProfile.VersionHistory)
	}

	archivedProfile, err := store.LoadProfile("stale-archived-skill")
	if err != nil {
		t.Fatalf("LoadProfile(archived): %v", err)
	}
	if archivedProfile.Status != evolution.SkillStatusDeleted {
		t.Fatalf("archived profile Status = %q, want %q", archivedProfile.Status, evolution.SkillStatusDeleted)
	}
	if len(archivedProfile.VersionHistory) != 1 || archivedProfile.VersionHistory[0].Action != "lifecycle:deleted" {
		t.Fatalf("archived profile VersionHistory = %+v, want lifecycle:deleted entry", archivedProfile.VersionHistory)
	}

	if _, statErr := os.Stat(skillPath); !os.IsNotExist(statErr) {
		t.Fatalf("expected lifecycle delete to remove skill file, stat err = %v", statErr)
	}
}

func TestRuntime_RunColdPathOnce_ProfileSaveFailureRollsBackSkillAndQuarantinesDraft(t *testing.T) {
	root := t.TempDir()
	paths := evolution.NewPaths(root, "")
	store := evolution.NewStore(paths)

	rule := evolution.LearningRecord{
		ID:          "rule-1",
		Kind:        evolution.RecordKindRule,
		WorkspaceID: root,
		CreatedAt:   time.Unix(1700000000, 0).UTC(),
		Summary:     "weather native-name path",
		Status:      evolution.RecordStatus("ready"),
		EventCount:  4,
	}
	if err := store.AppendLearningRecords([]evolution.LearningRecord{rule}); err != nil {
		t.Fatalf("AppendLearningRecords: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(paths.ProfilesDir), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(paths.ProfilesDir, []byte("not-a-directory"), 0o644); err != nil {
		t.Fatalf("WriteFile(profiles): %v", err)
	}

	rt, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config: config.EvolutionConfig{Enabled: true, Mode: "apply"},
		Now:    func() time.Time { return time.Unix(1700001000, 0).UTC() },
		Store:  store,
		Applier: evolution.NewApplier(paths, func() time.Time {
			return time.Unix(1700001000, 0).UTC()
		}),
		DraftGenerator: stubDraftGenerator{
			draft: evolution.SkillDraft{
				ID:              "draft-profile-fail",
				WorkspaceID:     root,
				SourceRecordID:  "rule-1",
				TargetSkillName: "weather",
				DraftType:       evolution.DraftTypeShortcut,
				ChangeKind:      evolution.ChangeKindCreate,
				HumanSummary:    "weather helper",
				BodyOrPatch:     "---\nname: weather\ndescription: weather helper\n---\n# Weather\n## Start Here\nUse native-name query first.\n",
			},
		},
		Organizer:      evolution.NewOrganizer(evolution.OrganizerOptions{MinCaseCount: 3, MinSuccessRate: 0.7}),
		SkillsRecaller: evolution.NewSkillsRecaller(root),
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	err = rt.RunColdPathOnce(context.Background(), root)
	if err == nil {
		t.Fatal("expected RunColdPathOnce to fail")
	}
	if !errors.Is(err, evolution.ErrApplyDraftFailed) {
		t.Fatalf("error = %v, want ErrApplyDraftFailed", err)
	}

	skillPath := filepath.Join(root, "skills", "weather", "SKILL.md")
	if _, statErr := os.Stat(skillPath); !os.IsNotExist(statErr) {
		t.Fatalf("expected rolled back skill file, got err=%v", statErr)
	}

	drafts, err := store.LoadDrafts()
	if err != nil {
		t.Fatalf("LoadDrafts: %v", err)
	}
	if len(drafts) != 1 {
		t.Fatalf("len(drafts) = %d, want 1", len(drafts))
	}
	if drafts[0].Status != evolution.DraftStatusQuarantined {
		t.Fatalf("draft status = %q, want %q", drafts[0].Status, evolution.DraftStatusQuarantined)
	}
	if len(drafts[0].ScanFindings) == 0 {
		t.Fatal("expected scan findings for profile save failure")
	}
}
