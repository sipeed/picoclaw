package evolution_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
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

	if err := rt.RunColdPathOnce(context.Background(), root); err != nil {
		t.Fatalf("RunColdPathOnce: %v", err)
	}

	skillPath := filepath.Join(root, "skills", "weather", "SKILL.md")
	if _, err := os.Stat(skillPath); err != nil {
		t.Fatalf("expected skill file: %v", err)
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
	if len(profile.AvoidPatterns) != 1 || profile.AvoidPatterns[0] != "avoid translating city names before querying weather" {
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

	if err := rt.RunColdPathOnce(context.Background(), root); err != nil {
		t.Fatalf("RunColdPathOnce: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, "skills", "weather", "SKILL.md")); !os.IsNotExist(err) {
		t.Fatalf("expected no applied skill file, got err=%v", err)
	}
	if _, err := store.LoadProfile("weather"); !os.IsNotExist(err) {
		t.Fatalf("expected no profile, got err=%v", err)
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

func TestRuntime_RunColdPathOnce_ApplyModeRetargetsStableMultiSkillPathIntoCombinedShortcut(t *testing.T) {
	root := t.TempDir()
	store := evolution.NewStore(evolution.NewPaths(root, ""))

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
				BodyOrPatch:     "Prefer the full theorem chain directly.",
			},
		},
		Organizer:      evolution.NewOrganizer(evolution.OrganizerOptions{MinCaseCount: 3, MinSuccessRate: 0.7}),
		SkillsRecaller: evolution.NewSkillsRecaller(root),
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	if err := rt.RunColdPathOnce(context.Background(), root); err != nil {
		t.Fatalf("RunColdPathOnce: %v", err)
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
	if err := os.WriteFile(skillPath, []byte("---\nname: stale-archived-skill\ndescription: stale\n---\n# Stale Archived Skill\n"), 0o644); err != nil {
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

	if err := rt.RunColdPathOnce(context.Background(), root); err != nil {
		t.Fatalf("RunColdPathOnce: %v", err)
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

	if _, err := os.Stat(skillPath); !os.IsNotExist(err) {
		t.Fatalf("expected lifecycle delete to remove skill file, stat err = %v", err)
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
