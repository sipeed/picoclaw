package evolution_test

import (
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/evolution"
)

func TestOrganizer_BuildRulesCreatesRuleRecord(t *testing.T) {
	ok := true
	cases := []evolution.LearningRecord{
		{
			ID:               "case-1",
			Kind:             evolution.RecordKindCase,
			WorkspaceID:      "ws-1",
			CreatedAt:        time.Unix(1700000000, 0).UTC(),
			Summary:          "weather shanghai",
			Status:           evolution.RecordStatus("new"),
			Success:          &ok,
			ActiveSkillNames: []string{"weather"},
		},
		{
			ID:               "case-2",
			Kind:             evolution.RecordKindCase,
			WorkspaceID:      "ws-1",
			CreatedAt:        time.Unix(1700000100, 0).UTC(),
			Summary:          "weather beijing",
			Status:           evolution.RecordStatus("new"),
			Success:          &ok,
			ActiveSkillNames: []string{"weather"},
		},
		{
			ID:               "case-3",
			Kind:             evolution.RecordKindCase,
			WorkspaceID:      "ws-1",
			CreatedAt:        time.Unix(1700000200, 0).UTC(),
			Summary:          "weather hangzhou",
			Status:           evolution.RecordStatus("new"),
			Success:          &ok,
			ActiveSkillNames: []string{"weather"},
		},
	}

	org := evolution.NewOrganizer(evolution.OrganizerOptions{
		MinCaseCount:   3,
		MinSuccessRate: 0.7,
		Now:            func() time.Time { return time.Unix(1700001000, 0).UTC() },
	})

	rules, err := org.BuildRules(cases)
	if err != nil {
		t.Fatalf("BuildRules: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("len(rules) = %d, want 1", len(rules))
	}

	rule := rules[0]
	if rule.Kind != evolution.RecordKindRule {
		t.Fatalf("Kind = %q, want %q", rule.Kind, evolution.RecordKindRule)
	}
	if rule.EventCount != 3 {
		t.Fatalf("EventCount = %d, want 3", rule.EventCount)
	}
	if len(rule.SourceRecordIDs) != 3 {
		t.Fatalf("SourceRecordIDs = %v", rule.SourceRecordIDs)
	}
	if rule.MaturityScore <= 0 {
		t.Fatalf("MaturityScore = %v, want > 0", rule.MaturityScore)
	}
	if len(rule.WinningPath) != 1 || rule.WinningPath[0] != "weather" {
		t.Fatalf("WinningPath = %v, want [weather]", rule.WinningPath)
	}
}

func TestOrganizer_BuildRulesSkipsImmatureCluster(t *testing.T) {
	ok := true
	cases := []evolution.LearningRecord{
		{
			ID:          "case-1",
			Kind:        evolution.RecordKindCase,
			WorkspaceID: "ws-1",
			CreatedAt:   time.Unix(1700000000, 0).UTC(),
			Summary:     "release build linux",
			Status:      evolution.RecordStatus("new"),
			Success:     &ok,
		},
	}

	org := evolution.NewOrganizer(evolution.OrganizerOptions{
		MinCaseCount:   3,
		MinSuccessRate: 0.7,
	})

	rules, err := org.BuildRules(cases)
	if err != nil {
		t.Fatalf("BuildRules: %v", err)
	}
	if len(rules) != 0 {
		t.Fatalf("len(rules) = %d, want 0", len(rules))
	}
}

func TestOrganizer_BuildRulesPrefersFinalSuccessfulPathFromAttemptTrail(t *testing.T) {
	ok := true
	cases := []evolution.LearningRecord{
		{
			ID:          "case-1",
			Kind:        evolution.RecordKindCase,
			WorkspaceID: "ws-1",
			CreatedAt:   time.Unix(1700000000, 0).UTC(),
			Summary:     "weather shanghai",
			Status:      evolution.RecordStatus("new"),
			Success:     &ok,
			AttemptTrail: &evolution.AttemptTrail{
				AttemptedSkills:     []string{"geocode", "weather"},
				FinalSuccessfulPath: []string{"geocode", "weather"},
			},
			ActiveSkillNames: []string{"geocode", "weather"},
		},
		{
			ID:          "case-2",
			Kind:        evolution.RecordKindCase,
			WorkspaceID: "ws-1",
			CreatedAt:   time.Unix(1700000100, 0).UTC(),
			Summary:     "weather beijing",
			Status:      evolution.RecordStatus("new"),
			Success:     &ok,
			AttemptTrail: &evolution.AttemptTrail{
				AttemptedSkills:     []string{"browser", "weather"},
				FinalSuccessfulPath: []string{"geocode", "weather"},
			},
			ActiveSkillNames: []string{"browser", "weather"},
		},
		{
			ID:          "case-3",
			Kind:        evolution.RecordKindCase,
			WorkspaceID: "ws-1",
			CreatedAt:   time.Unix(1700000200, 0).UTC(),
			Summary:     "weather hangzhou",
			Status:      evolution.RecordStatus("new"),
			Success:     &ok,
			AttemptTrail: &evolution.AttemptTrail{
				AttemptedSkills:     []string{"maps", "weather"},
				FinalSuccessfulPath: []string{"geocode", "weather"},
			},
			ActiveSkillNames: []string{"maps", "weather"},
		},
	}

	org := evolution.NewOrganizer(evolution.OrganizerOptions{
		MinCaseCount:   3,
		MinSuccessRate: 0.7,
		Now:            func() time.Time { return time.Unix(1700001000, 0).UTC() },
	})

	rules, err := org.BuildRules(cases)
	if err != nil {
		t.Fatalf("BuildRules: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("len(rules) = %d, want 1", len(rules))
	}
	if got := rules[0].WinningPath; len(got) != 2 || got[0] != "geocode" || got[1] != "weather" {
		t.Fatalf("WinningPath = %v, want [geocode weather]", got)
	}
}

func TestOrganizer_BuildRulesCapturesLateAddedSkillHintFromSnapshots(t *testing.T) {
	ok := true
	cases := []evolution.LearningRecord{
		{
			ID:          "case-1",
			Kind:        evolution.RecordKindCase,
			WorkspaceID: "ws-1",
			CreatedAt:   time.Unix(1700000000, 0).UTC(),
			Summary:     "weather shanghai",
			Status:      evolution.RecordStatus("new"),
			Success:     &ok,
			AttemptTrail: &evolution.AttemptTrail{
				AttemptedSkills:     []string{"geocode", "weather"},
				FinalSuccessfulPath: []string{"geocode", "weather"},
				SkillContextSnapshots: []evolution.SkillContextSnapshot{
					{Sequence: 1, Trigger: "initial_build", SkillNames: []string{"geocode"}},
					{Sequence: 2, Trigger: "context_retry_rebuild", SkillNames: []string{"geocode", "weather"}},
				},
			},
		},
		{
			ID:          "case-2",
			Kind:        evolution.RecordKindCase,
			WorkspaceID: "ws-1",
			CreatedAt:   time.Unix(1700000100, 0).UTC(),
			Summary:     "weather beijing",
			Status:      evolution.RecordStatus("new"),
			Success:     &ok,
			AttemptTrail: &evolution.AttemptTrail{
				AttemptedSkills:     []string{"browser", "weather"},
				FinalSuccessfulPath: []string{"geocode", "weather"},
				SkillContextSnapshots: []evolution.SkillContextSnapshot{
					{Sequence: 1, Trigger: "initial_build", SkillNames: []string{"geocode"}},
					{Sequence: 2, Trigger: "context_retry_rebuild", SkillNames: []string{"geocode", "weather"}},
				},
			},
		},
		{
			ID:          "case-3",
			Kind:        evolution.RecordKindCase,
			WorkspaceID: "ws-1",
			CreatedAt:   time.Unix(1700000200, 0).UTC(),
			Summary:     "weather hangzhou",
			Status:      evolution.RecordStatus("new"),
			Success:     &ok,
			AttemptTrail: &evolution.AttemptTrail{
				AttemptedSkills:     []string{"maps", "weather"},
				FinalSuccessfulPath: []string{"geocode", "weather"},
				SkillContextSnapshots: []evolution.SkillContextSnapshot{
					{Sequence: 1, Trigger: "initial_build", SkillNames: []string{"geocode"}},
					{Sequence: 2, Trigger: "context_retry_rebuild", SkillNames: []string{"geocode", "weather"}},
				},
			},
		},
	}

	org := evolution.NewOrganizer(evolution.OrganizerOptions{
		MinCaseCount:   3,
		MinSuccessRate: 0.7,
		Now:            func() time.Time { return time.Unix(1700001000, 0).UTC() },
	})

	rules, err := org.BuildRules(cases)
	if err != nil {
		t.Fatalf("BuildRules: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("len(rules) = %d, want 1", len(rules))
	}
	if got := rules[0].LateAddedSkills; len(got) != 1 || got[0] != "weather" {
		t.Fatalf("LateAddedSkills = %v, want [weather]", got)
	}
	if got := rules[0].FinalSnapshotTrigger; got != "context_retry_rebuild" {
		t.Fatalf("FinalSnapshotTrigger = %q, want context_retry_rebuild", got)
	}
}

func TestOrganizer_BuildRulesUsesAddedSkillNamesWithoutSnapshots(t *testing.T) {
	ok := true
	cases := []evolution.LearningRecord{
		{
			ID:              "case-1",
			Kind:            evolution.RecordKindCase,
			WorkspaceID:     "ws-1",
			CreatedAt:       time.Unix(1700000000, 0).UTC(),
			Summary:         "weather shanghai",
			UserGoal:        "check weather in shanghai",
			Status:          evolution.RecordStatus("new"),
			Success:         &ok,
			UsedSkillNames:  []string{"geocode", "weather"},
			AddedSkillNames: []string{"weather"},
		},
		{
			ID:              "case-2",
			Kind:            evolution.RecordKindCase,
			WorkspaceID:     "ws-1",
			CreatedAt:       time.Unix(1700000100, 0).UTC(),
			Summary:         "weather beijing",
			UserGoal:        "check weather in beijing",
			Status:          evolution.RecordStatus("new"),
			Success:         &ok,
			UsedSkillNames:  []string{"geocode", "weather"},
			AddedSkillNames: []string{"weather"},
		},
		{
			ID:              "case-3",
			Kind:            evolution.RecordKindCase,
			WorkspaceID:     "ws-1",
			CreatedAt:       time.Unix(1700000200, 0).UTC(),
			Summary:         "weather hangzhou",
			UserGoal:        "check weather in hangzhou",
			Status:          evolution.RecordStatus("new"),
			Success:         &ok,
			UsedSkillNames:  []string{"geocode", "weather"},
			AddedSkillNames: []string{"weather"},
		},
	}

	org := evolution.NewOrganizer(evolution.OrganizerOptions{
		MinCaseCount:   3,
		MinSuccessRate: 0.7,
		Now:            func() time.Time { return time.Unix(1700001000, 0).UTC() },
	})

	rules, err := org.BuildRules(cases)
	if err != nil {
		t.Fatalf("BuildRules: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("len(rules) = %d, want 1", len(rules))
	}
	if got := rules[0].WinningPath; len(got) != 2 || got[0] != "geocode" || got[1] != "weather" {
		t.Fatalf("WinningPath = %v, want [geocode weather]", got)
	}
	if got := rules[0].LateAddedSkills; len(got) != 1 || got[0] != "weather" {
		t.Fatalf("LateAddedSkills = %v, want [weather]", got)
	}
	if got := rules[0].FinalSnapshotTrigger; got != "loaded_during_task" {
		t.Fatalf("FinalSnapshotTrigger = %q, want loaded_during_task", got)
	}
}
