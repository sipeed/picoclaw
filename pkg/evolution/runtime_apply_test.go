package evolution_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
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
		Config: config.EvolutionConfig{Enabled: true, Mode: "apply", AutoApply: true},
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

func TestRuntime_RunColdPathOnce_ApplyModeWithoutAutoApplyKeepsCandidateDraft(t *testing.T) {
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
		Config: config.EvolutionConfig{Enabled: true, Mode: "apply", AutoApply: false},
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
		Config: config.EvolutionConfig{Enabled: true, Mode: "apply", AutoApply: true},
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
		Config: config.EvolutionConfig{Enabled: true, Mode: "apply", AutoApply: true},
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
