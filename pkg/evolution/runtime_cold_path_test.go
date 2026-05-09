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
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/skills"
)

type stubDraftGenerator struct {
	draft evolution.SkillDraft
	err   error
}

func (g stubDraftGenerator) GenerateDraft(
	_ context.Context,
	_ evolution.LearningRecord,
	_ []skills.SkillInfo,
) (evolution.SkillDraft, error) {
	return g.draft, g.err
}

type sequenceDraftGenerator struct {
	results []draftGenerationResult
	index   int
}

type draftGenerationResult struct {
	draft evolution.SkillDraft
	err   error
}

type evidenceCaptureDraftGenerator struct {
	evidence evolution.DraftEvidence
}

func (g *evidenceCaptureDraftGenerator) GenerateDraft(
	_ context.Context,
	_ evolution.LearningRecord,
	_ []skills.SkillInfo,
) (evolution.SkillDraft, error) {
	return evolution.SkillDraft{}, nil
}

func (g *evidenceCaptureDraftGenerator) GenerateDraftWithEvidence(
	_ context.Context,
	_ evolution.LearningRecord,
	_ []skills.SkillInfo,
	evidence evolution.DraftEvidence,
) (evolution.SkillDraft, error) {
	g.evidence = evidence
	return evolution.SkillDraft{
		ID:              "draft-evidence",
		TargetSkillName: "weather",
		DraftType:       evolution.DraftTypeShortcut,
		ChangeKind:      evolution.ChangeKindCreate,
		HumanSummary:    "weather helper",
		BodyOrPatch:     "---\nname: weather\ndescription: weather helper\n---\n# Weather\nUse current workspace evidence.\n",
	}, nil
}

type stubSuccessJudge struct {
	decisions map[string]evolution.TaskSuccessDecision
	calls     []string
}

func (j *stubSuccessJudge) JudgeTaskRecord(
	_ context.Context,
	record evolution.LearningRecord,
) (evolution.TaskSuccessDecision, error) {
	j.calls = append(j.calls, record.ID)
	if decision, ok := j.decisions[record.ID]; ok {
		return decision, nil
	}
	return evolution.TaskSuccessDecision{Success: true, Reason: "default success"}, nil
}

func (g *sequenceDraftGenerator) GenerateDraft(
	_ context.Context,
	_ evolution.LearningRecord,
	_ []skills.SkillInfo,
) (evolution.SkillDraft, error) {
	if g.index >= len(g.results) {
		return evolution.SkillDraft{}, nil
	}
	result := g.results[g.index]
	g.index++
	return result.draft, result.err
}

func TestRuntime_RunColdPathOnce_GeneratesCandidateDraft(t *testing.T) {
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

	rt, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config: config.EvolutionConfig{Enabled: true, Mode: "draft"},
		Now:    func() time.Time { return time.Unix(1700001000, 0).UTC() },
		DraftGenerator: stubDraftGenerator{
			draft: evolution.SkillDraft{
				ID:              "draft-1",
				WorkspaceID:     root,
				SourceRecordID:  "rule-1",
				TargetSkillName: "weather",
				DraftType:       evolution.DraftTypeShortcut,
				ChangeKind:      evolution.ChangeKindAppend,
				HumanSummary:    "prefer native-name path first",
				BodyOrPatch:     "## Start Here\nUse native-name query first.",
			},
		},
		Store:          store,
		SkillsRecaller: evolution.NewSkillsRecaller(root),
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	if err := rt.RunColdPathOnce(context.Background(), root); err != nil {
		t.Fatalf("RunColdPathOnce: %v", err)
	}

	drafts, err := store.LoadDrafts()
	if err != nil {
		t.Fatalf("LoadDrafts: %v", err)
	}
	if len(drafts) != 1 {
		t.Fatalf("len(drafts) = %d, want 1", len(drafts))
	}
	if drafts[0].Status != evolution.DraftStatusCandidate {
		t.Fatalf("Status = %q, want %q", drafts[0].Status, evolution.DraftStatusCandidate)
	}
}

func TestRuntime_RunColdPathOnce_AdmitsOnlyRecordsApprovedBySuccessJudge(t *testing.T) {
	root := t.TempDir()
	store := evolution.NewStore(evolution.NewPaths(root, ""))
	ok := true
	failed := false

	records := []evolution.LearningRecord{
		{
			ID:             "task-failed",
			Kind:           evolution.RecordKindTask,
			WorkspaceID:    root,
			CreatedAt:      time.Unix(1700000000, 0).UTC(),
			Summary:        "failed weather attempt",
			UserGoal:       "check weather in shanghai",
			FinalOutput:    "tool failed",
			Status:         evolution.RecordStatus("new"),
			Success:        &failed,
			UsedSkillNames: []string{"weather", "native-name"},
			ToolKinds:      []string{"read_file"},
		},
		{
			ID:             "task-rejected",
			Kind:           evolution.RecordKindTask,
			WorkspaceID:    root,
			CreatedAt:      time.Unix(1700000100, 0).UTC(),
			Summary:        "partial weather answer",
			UserGoal:       "check weather in shanghai",
			FinalOutput:    "I will check it next",
			Status:         evolution.RecordStatus("new"),
			Success:        &ok,
			UsedSkillNames: []string{"weather", "native-name"},
			ToolKinds:      []string{"read_file"},
			ToolExecutions: []evolution.ToolExecutionRecord{
				{Name: "read_file", Success: true},
				{Name: "read_file", Success: true},
			},
		},
		{
			ID:              "task-admitted",
			Kind:            evolution.RecordKindTask,
			WorkspaceID:     root,
			CreatedAt:       time.Unix(1700000200, 0).UTC(),
			Summary:         "weather answer delivered",
			UserGoal:        "check weather in shanghai",
			FinalOutput:     "sunny, 26C",
			Status:          evolution.RecordStatus("new"),
			Success:         &ok,
			UsedSkillNames:  []string{"weather", "native-name"},
			AddedSkillNames: []string{"native-name"},
			ToolKinds:       []string{"read_file"},
			ToolExecutions: []evolution.ToolExecutionRecord{
				{Name: "read_file", Success: true},
				{Name: "read_file", Success: true},
			},
			AttemptTrail: &evolution.AttemptTrail{
				AttemptedSkills:     []string{"weather"},
				FinalSuccessfulPath: []string{"weather"},
			},
		},
	}
	if err := store.AppendLearningRecords(records); err != nil {
		t.Fatalf("AppendLearningRecords: %v", err)
	}

	judge := &stubSuccessJudge{
		decisions: map[string]evolution.TaskSuccessDecision{
			"task-rejected": {Success: false, Reason: "only partial reasoning"},
			"task-admitted": {Success: true, Reason: "goal achieved"},
		},
	}

	rt, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config:         config.EvolutionConfig{Enabled: true, Mode: "draft", MinTaskCount: 1},
		Store:          store,
		SuccessJudge:   judge,
		Organizer:      evolution.NewOrganizer(evolution.OrganizerOptions{MinCaseCount: 1, MinSuccessRate: 1}),
		SkillsRecaller: evolution.NewSkillsRecaller(root),
		DraftGenerator: stubDraftGenerator{
			draft: evolution.SkillDraft{
				ID:              "draft-weather",
				TargetSkillName: "weather",
				DraftType:       evolution.DraftTypeShortcut,
				ChangeKind:      evolution.ChangeKindAppend,
				HumanSummary:    "prefer the proven weather path",
				BodyOrPatch:     "## Start Here\nUse the weather path directly.",
			},
		},
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	if err := rt.RunColdPathOnce(context.Background(), root); err != nil {
		t.Fatalf("RunColdPathOnce: %v", err)
	}

	if len(judge.calls) != 2 || judge.calls[0] != "task-rejected" || judge.calls[1] != "task-admitted" {
		t.Fatalf("judge calls = %v, want [task-rejected task-admitted]", judge.calls)
	}

	allRecords, err := store.LoadLearningRecords()
	if err != nil {
		t.Fatalf("LoadLearningRecords: %v", err)
	}

	var pattern evolution.LearningRecord
	foundPattern := false
	for _, record := range allRecords {
		if record.Kind != evolution.RecordKindPattern {
			continue
		}
		pattern = record
		foundPattern = true
		break
	}
	if !foundPattern {
		t.Fatal("expected generated pattern record")
	}
	if len(pattern.TaskRecordIDs) != 1 || pattern.TaskRecordIDs[0] != "task-admitted" {
		t.Fatalf("TaskRecordIDs = %v, want [task-admitted]", pattern.TaskRecordIDs)
	}
	if pattern.Label == "" {
		t.Fatal("pattern Label should not be empty")
	}

	drafts, err := store.LoadDrafts()
	if err != nil {
		t.Fatalf("LoadDrafts: %v", err)
	}
	if len(drafts) != 1 {
		t.Fatalf("len(drafts) = %d, want 1", len(drafts))
	}
	if drafts[0].SourceRecordID != pattern.ID {
		t.Fatalf("draft SourceRecordID = %q, want %q", drafts[0].SourceRecordID, pattern.ID)
	}
}

func TestRuntime_RunColdPathOnce_RejectsClusterBelowMinSuccessRatio(t *testing.T) {
	root := t.TempDir()
	store := evolution.NewStore(evolution.NewPaths(root, ""))
	ok := true
	failed := false

	records := []evolution.LearningRecord{
		{
			ID:             "task-success",
			Kind:           evolution.RecordKindTask,
			WorkspaceID:    root,
			CreatedAt:      time.Unix(1700000200, 0).UTC(),
			Summary:        "weather lookup 100",
			FinalOutput:    "sunny",
			Status:         evolution.RecordStatus("new"),
			Success:        &ok,
			UsedSkillNames: []string{"weather"},
		},
		{
			ID:             "task-failed-1",
			Kind:           evolution.RecordKindTask,
			WorkspaceID:    root,
			CreatedAt:      time.Unix(1700000100, 0).UTC(),
			Summary:        "weather lookup 200",
			FinalOutput:    "failed",
			Status:         evolution.RecordStatus("new"),
			Success:        &failed,
			UsedSkillNames: []string{"weather"},
		},
		{
			ID:             "task-failed-2",
			Kind:           evolution.RecordKindTask,
			WorkspaceID:    root,
			CreatedAt:      time.Unix(1700000000, 0).UTC(),
			Summary:        "weather lookup 300",
			FinalOutput:    "failed",
			Status:         evolution.RecordStatus("new"),
			Success:        &failed,
			UsedSkillNames: []string{"weather"},
		},
	}
	if err := store.AppendLearningRecords(records); err != nil {
		t.Fatalf("AppendLearningRecords: %v", err)
	}

	rt, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config:         config.EvolutionConfig{Enabled: true, Mode: "draft", MinTaskCount: 1, MinSuccessRatio: 0.8},
		Store:          store,
		SuccessJudge:   &stubSuccessJudge{},
		SkillsRecaller: evolution.NewSkillsRecaller(root),
		DraftGenerator: stubDraftGenerator{
			draft: evolution.SkillDraft{
				ID:              "draft-weather",
				TargetSkillName: "weather",
				DraftType:       evolution.DraftTypeShortcut,
				ChangeKind:      evolution.ChangeKindAppend,
				HumanSummary:    "prefer the proven weather path",
				BodyOrPatch:     "## Start Here\nUse the weather path directly.",
			},
		},
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	if err := rt.RunColdPathOnce(context.Background(), root); err != nil {
		t.Fatalf("RunColdPathOnce: %v", err)
	}

	patterns, err := store.LoadPatternRecords()
	if err != nil {
		t.Fatalf("LoadPatternRecords: %v", err)
	}
	if len(patterns) != 0 {
		t.Fatalf("len(patterns) = %d, want 0", len(patterns))
	}
	drafts, err := store.LoadDrafts()
	if err != nil {
		t.Fatalf("LoadDrafts: %v", err)
	}
	if len(drafts) != 0 {
		t.Fatalf("len(drafts) = %d, want 0", len(drafts))
	}
}

func TestRuntime_RunColdPathOnce_FallbackUsesJudgeAdjustedSuccessRatio(t *testing.T) {
	root := t.TempDir()
	store := evolution.NewStore(evolution.NewPaths(root, ""))
	ok := true

	records := []evolution.LearningRecord{
		{
			ID:             "task-success",
			Kind:           evolution.RecordKindTask,
			WorkspaceID:    root,
			CreatedAt:      time.Unix(1700000200, 0).UTC(),
			Summary:        "weather lookup 100",
			FinalOutput:    "sunny",
			Status:         evolution.RecordStatus("new"),
			Success:        &ok,
			UsedSkillNames: []string{"weather"},
		},
		{
			ID:             "task-judge-rejected",
			Kind:           evolution.RecordKindTask,
			WorkspaceID:    root,
			CreatedAt:      time.Unix(1700000100, 0).UTC(),
			Summary:        "weather lookup 200",
			FinalOutput:    "partial answer",
			Status:         evolution.RecordStatus("new"),
			Success:        &ok,
			UsedSkillNames: []string{"weather"},
		},
	}
	if err := store.AppendLearningRecords(records); err != nil {
		t.Fatalf("AppendLearningRecords: %v", err)
	}

	judge := &stubSuccessJudge{
		decisions: map[string]evolution.TaskSuccessDecision{
			"task-success":        {Success: true, Reason: "goal achieved"},
			"task-judge-rejected": {Success: false, Reason: "partial result"},
		},
	}
	clusterer := evolution.NewLLMPatternClusterer(
		&llmClusterTestProvider{content: `not-json`, defaultModel: "test-model"},
		"test-model",
		evolution.NewHeuristicPatternClusterer(1, nil),
		1,
		func() time.Time { return time.Unix(1700000000, 0).UTC() },
	)

	rt, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config:           config.EvolutionConfig{Enabled: true, Mode: "draft", MinTaskCount: 1, MinSuccessRatio: 0.8},
		Store:            store,
		PatternClusterer: clusterer,
		SuccessJudge:     judge,
		SkillsRecaller:   evolution.NewSkillsRecaller(root),
		DraftGenerator: stubDraftGenerator{
			draft: evolution.SkillDraft{
				ID:              "draft-weather",
				TargetSkillName: "weather",
				DraftType:       evolution.DraftTypeShortcut,
				ChangeKind:      evolution.ChangeKindAppend,
				HumanSummary:    "prefer the proven weather path",
				BodyOrPatch:     "## Start Here\nUse the weather path directly.",
			},
		},
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	if err := rt.RunColdPathOnce(context.Background(), root); err != nil {
		t.Fatalf("RunColdPathOnce: %v", err)
	}

	patterns, err := store.LoadPatternRecords()
	if err != nil {
		t.Fatalf("LoadPatternRecords: %v", err)
	}
	if len(patterns) != 0 {
		t.Fatalf("len(patterns) = %d, want 0", len(patterns))
	}
	drafts, err := store.LoadDrafts()
	if err != nil {
		t.Fatalf("LoadDrafts: %v", err)
	}
	if len(drafts) != 0 {
		t.Fatalf("len(drafts) = %d, want 0", len(drafts))
	}
}

func TestRuntime_RunColdPathOnce_FallbackMarksAcceptedFailureEvidenceClustered(t *testing.T) {
	root := t.TempDir()
	store := evolution.NewStore(evolution.NewPaths(root, ""))
	ok := true

	records := []evolution.LearningRecord{
		{
			ID:             "task-success",
			Kind:           evolution.RecordKindTask,
			WorkspaceID:    root,
			CreatedAt:      time.Unix(1700000200, 0).UTC(),
			Summary:        "weather lookup 100",
			FinalOutput:    "sunny",
			Status:         evolution.RecordStatus("new"),
			Success:        &ok,
			UsedSkillNames: []string{"weather"},
		},
		{
			ID:             "task-judge-rejected",
			Kind:           evolution.RecordKindTask,
			WorkspaceID:    root,
			CreatedAt:      time.Unix(1700000100, 0).UTC(),
			Summary:        "weather lookup 200",
			FinalOutput:    "partial answer",
			Status:         evolution.RecordStatus("new"),
			Success:        &ok,
			UsedSkillNames: []string{"weather"},
		},
	}
	if err := store.AppendLearningRecords(records); err != nil {
		t.Fatalf("AppendLearningRecords: %v", err)
	}

	judge := &stubSuccessJudge{
		decisions: map[string]evolution.TaskSuccessDecision{
			"task-success":        {Success: true, Reason: "goal achieved"},
			"task-judge-rejected": {Success: false, Reason: "partial result"},
		},
	}
	clusterer := evolution.NewLLMPatternClusterer(
		&llmClusterTestProvider{content: `not-json`, defaultModel: "test-model"},
		"test-model",
		evolution.NewHeuristicPatternClusterer(1, nil),
		1,
		func() time.Time { return time.Unix(1700000000, 0).UTC() },
	)

	rt, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config:           config.EvolutionConfig{Enabled: true, Mode: "draft", MinTaskCount: 1, MinSuccessRatio: 0.5},
		Store:            store,
		PatternClusterer: clusterer,
		SuccessJudge:     judge,
		SkillsRecaller:   evolution.NewSkillsRecaller(root),
		DraftGenerator: stubDraftGenerator{
			draft: evolution.SkillDraft{
				ID:              "draft-weather",
				TargetSkillName: "weather",
				DraftType:       evolution.DraftTypeShortcut,
				ChangeKind:      evolution.ChangeKindAppend,
				HumanSummary:    "prefer the proven weather path",
				BodyOrPatch:     "## Start Here\nUse the weather path directly.",
			},
		},
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	if err := rt.RunColdPathOnce(context.Background(), root); err != nil {
		t.Fatalf("RunColdPathOnce: %v", err)
	}

	patterns, err := store.LoadPatternRecords()
	if err != nil {
		t.Fatalf("LoadPatternRecords: %v", err)
	}
	if len(patterns) != 1 {
		t.Fatalf("len(patterns) = %d, want 1", len(patterns))
	}
	if got := strings.Join(patterns[0].TaskRecordIDs, ","); got != "task-success" {
		t.Fatalf("pattern TaskRecordIDs = %v, want only successful task", patterns[0].TaskRecordIDs)
	}
	taskRecords, err := store.LoadTaskRecords()
	if err != nil {
		t.Fatalf("LoadTaskRecords: %v", err)
	}
	statusByID := make(map[string]evolution.RecordStatus)
	for _, record := range taskRecords {
		statusByID[record.ID] = record.Status
	}
	for _, id := range []string{"task-success", "task-judge-rejected"} {
		if statusByID[id] != evolution.RecordStatus("clustered") {
			t.Fatalf("statusByID[%s] = %q, want clustered", id, statusByID[id])
		}
	}
}

func TestRuntime_RunColdPathOnce_DraftEvidenceDoesNotCrossWorkspaceWithDuplicateTaskID(t *testing.T) {
	sharedState := t.TempDir()
	workspaceA := t.TempDir()
	workspaceB := t.TempDir()
	store := evolution.NewStore(evolution.NewPaths(workspaceA, sharedState))
	ok := true

	if err := store.AppendTaskRecords(context.Background(), []evolution.LearningRecord{
		{
			ID:             "main-turn-1",
			Kind:           evolution.RecordKindTask,
			WorkspaceID:    workspaceB,
			CreatedAt:      time.Unix(1700000000, 0).UTC(),
			Summary:        "other workspace weather",
			FinalOutput:    "foreign workspace output",
			Status:         evolution.RecordStatus("clustered"),
			Success:        &ok,
			UsedSkillNames: []string{"foreign-skill"},
		},
		{
			ID:             "main-turn-1",
			Kind:           evolution.RecordKindTask,
			WorkspaceID:    workspaceA,
			CreatedAt:      time.Unix(1700000001, 0).UTC(),
			Summary:        "current workspace weather",
			FinalOutput:    "current workspace output",
			Status:         evolution.RecordStatus("clustered"),
			Success:        &ok,
			UsedSkillNames: []string{"current-skill"},
		},
	}); err != nil {
		t.Fatalf("AppendTaskRecords: %v", err)
	}
	if err := store.AppendPatternRecords([]evolution.LearningRecord{{
		ID:            "pattern-workspace-a",
		Kind:          evolution.RecordKindPattern,
		WorkspaceID:   workspaceA,
		CreatedAt:     time.Unix(1700000002, 0).UTC(),
		Summary:       "current workspace weather",
		Status:        evolution.RecordStatus("ready"),
		TaskRecordIDs: []string{"main-turn-1"},
	}}); err != nil {
		t.Fatalf("AppendPatternRecords: %v", err)
	}

	generator := &evidenceCaptureDraftGenerator{}
	rt, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config:         config.EvolutionConfig{Enabled: true, Mode: "draft", StateDir: sharedState},
		Store:          store,
		SkillsRecaller: evolution.NewSkillsRecaller(workspaceA),
		DraftGenerator: generator,
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	if err := rt.RunColdPathOnce(context.Background(), workspaceA); err != nil {
		t.Fatalf("RunColdPathOnce: %v", err)
	}
	if len(generator.evidence.TaskRecords) != 1 {
		t.Fatalf("evidence task count = %d, want 1: %#v", len(generator.evidence.TaskRecords), generator.evidence.TaskRecords)
	}
	task := generator.evidence.TaskRecords[0]
	if task.WorkspaceID != workspaceA {
		t.Fatalf("evidence workspace = %q, want %q", task.WorkspaceID, workspaceA)
	}
	if task.FinalOutput != "current workspace output" {
		t.Fatalf("evidence FinalOutput = %q, want current workspace output", task.FinalOutput)
	}
	if len(task.UsedSkillNames) != 1 || task.UsedSkillNames[0] != "current-skill" {
		t.Fatalf("evidence UsedSkillNames = %v, want [current-skill]", task.UsedSkillNames)
	}
}

func TestRuntime_RunColdPathOnce_AdmitsSingleSkillTaskButWaitsForMinTaskCount(t *testing.T) {
	root := t.TempDir()
	store := evolution.NewStore(evolution.NewPaths(root, ""))
	ok := true

	record := evolution.LearningRecord{
		ID:              "task-simple",
		Kind:            evolution.RecordKindTask,
		WorkspaceID:     root,
		CreatedAt:       time.Unix(1700000250, 0).UTC(),
		Summary:         "simple weather lookup",
		UserGoal:        "check weather",
		FinalOutput:     "sunny",
		Status:          evolution.RecordStatus("new"),
		Success:         &ok,
		UsedSkillNames:  []string{"weather"},
		AddedSkillNames: []string{"weather"},
		ToolKinds:       []string{"read_file"},
		ToolExecutions:  []evolution.ToolExecutionRecord{{Name: "read_file", Success: true, SkillNames: []string{"weather"}}},
		AttemptTrail: &evolution.AttemptTrail{
			AttemptedSkills:     []string{"weather"},
			FinalSuccessfulPath: []string{"weather"},
		},
	}
	if err := store.AppendLearningRecords([]evolution.LearningRecord{record}); err != nil {
		t.Fatalf("AppendLearningRecords: %v", err)
	}

	judge := &stubSuccessJudge{}
	rt, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config:         config.EvolutionConfig{Enabled: true, Mode: "draft"},
		Store:          store,
		SuccessJudge:   judge,
		Organizer:      evolution.NewOrganizer(evolution.OrganizerOptions{MinCaseCount: 1, MinSuccessRate: 1}),
		SkillsRecaller: evolution.NewSkillsRecaller(root),
		DraftGenerator: stubDraftGenerator{
			draft: evolution.SkillDraft{
				ID:              "draft-simple",
				TargetSkillName: "weather",
				DraftType:       evolution.DraftTypeShortcut,
				ChangeKind:      evolution.ChangeKindAppend,
				HumanSummary:    "simple draft",
				BodyOrPatch:     "## Start Here\nUse weather.",
			},
		},
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	if err := rt.RunColdPathOnce(context.Background(), root); err != nil {
		t.Fatalf("RunColdPathOnce: %v", err)
	}
	if len(judge.calls) != 1 || judge.calls[0] != "task-simple" {
		t.Fatalf("judge calls = %v, want [task-simple]", judge.calls)
	}
	drafts, err := store.LoadDrafts()
	if err != nil {
		t.Fatalf("LoadDrafts: %v", err)
	}
	if len(drafts) != 0 {
		t.Fatalf("len(drafts) = %d, want 0", len(drafts))
	}
}

func TestRuntime_RunColdPathOnce_RejectsTaskWhenSuccessJudgeRejects(t *testing.T) {
	root := t.TempDir()
	store := evolution.NewStore(evolution.NewPaths(root, ""))
	ok := true

	record := evolution.LearningRecord{
		ID:              "task-detailed-path",
		Kind:            evolution.RecordKindTask,
		WorkspaceID:     root,
		CreatedAt:       time.Unix(1700000300, 0).UTC(),
		Summary:         "computed theorem chain",
		UserGoal:        "调用三一定理计算100",
		FinalOutput:     "最终结果：100 通过三一定理计算得到 120",
		Status:          evolution.RecordStatus("new"),
		Success:         &ok,
		UsedSkillNames:  []string{"three-one-theorem", "four-two-theorem", "five-three-theorem"},
		AddedSkillNames: []string{"three-one-theorem", "four-two-theorem", "five-three-theorem"},
		ToolKinds:       []string{"read_file"},
		ToolExecutions: []evolution.ToolExecutionRecord{
			{Name: "read_file", Success: true, SkillNames: []string{"three-one-theorem"}},
			{Name: "read_file", Success: true, SkillNames: []string{"four-two-theorem"}},
			{Name: "read_file", Success: true, SkillNames: []string{"five-three-theorem"}},
		},
		AttemptTrail: &evolution.AttemptTrail{
			AttemptedSkills:     []string{"three-one-theorem", "four-two-theorem", "five-three-theorem"},
			FinalSuccessfulPath: []string{"three-one-theorem", "four-two-theorem", "five-three-theorem"},
		},
	}
	if err := store.AppendLearningRecords([]evolution.LearningRecord{record}); err != nil {
		t.Fatalf("AppendLearningRecords: %v", err)
	}

	judge := &stubSuccessJudge{
		decisions: map[string]evolution.TaskSuccessDecision{
			"task-detailed-path": {Success: false, Reason: "llm false negative"},
		},
	}

	rt, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config:         config.EvolutionConfig{Enabled: true, Mode: "draft"},
		Store:          store,
		SuccessJudge:   judge,
		Organizer:      evolution.NewOrganizer(evolution.OrganizerOptions{MinCaseCount: 1, MinSuccessRate: 1}),
		SkillsRecaller: evolution.NewSkillsRecaller(root),
		DraftGenerator: stubDraftGenerator{
			draft: evolution.SkillDraft{
				ID:              "draft-detailed-path",
				TargetSkillName: "three-one-theorem",
				DraftType:       evolution.DraftTypeShortcut,
				ChangeKind:      evolution.ChangeKindAppend,
				HumanSummary:    "prefer the full theorem chain",
				BodyOrPatch:     "## Start Here\nUse the full three-one, four-two, five-three theorem chain.",
			},
		},
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	if err := rt.RunColdPathOnce(context.Background(), root); err != nil {
		t.Fatalf("RunColdPathOnce: %v", err)
	}

	allRecords, err := store.LoadLearningRecords()
	if err != nil {
		t.Fatalf("LoadLearningRecords: %v", err)
	}

	foundPattern := false
	for _, record := range allRecords {
		if record.Kind != evolution.RecordKindPattern {
			continue
		}
		foundPattern = true
		break
	}
	if foundPattern {
		t.Fatal("unexpected pattern record for rejected task")
	}
}

func TestRuntime_RunColdPathOnce_QuarantinesInvalidDraft(t *testing.T) {
	root := t.TempDir()
	store := evolution.NewStore(evolution.NewPaths(root, ""))

	rule := evolution.LearningRecord{
		ID:          "rule-1",
		Kind:        evolution.RecordKindRule,
		WorkspaceID: root,
		CreatedAt:   time.Unix(1700000000, 0).UTC(),
		Summary:     "release path",
		Status:      evolution.RecordStatus("ready"),
		EventCount:  4,
	}
	if err := store.AppendLearningRecords([]evolution.LearningRecord{rule}); err != nil {
		t.Fatalf("AppendLearningRecords: %v", err)
	}

	rt, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config: config.EvolutionConfig{Enabled: true, Mode: "draft"},
		DraftGenerator: stubDraftGenerator{
			draft: evolution.SkillDraft{
				ID:              "draft-1",
				WorkspaceID:     root,
				SourceRecordID:  "rule-1",
				TargetSkillName: "",
				DraftType:       evolution.DraftTypeShortcut,
				ChangeKind:      evolution.ChangeKindAppend,
				HumanSummary:    "broken",
				BodyOrPatch:     "",
			},
		},
		Store:          store,
		SkillsRecaller: evolution.NewSkillsRecaller(root),
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	if err := rt.RunColdPathOnce(context.Background(), root); err != nil {
		t.Fatalf("RunColdPathOnce: %v", err)
	}

	drafts, err := store.LoadDrafts()
	if err != nil {
		t.Fatalf("LoadDrafts: %v", err)
	}
	if len(drafts) != 1 {
		t.Fatalf("len(drafts) = %d, want 1", len(drafts))
	}
	if drafts[0].Status != evolution.DraftStatusQuarantined {
		t.Fatalf("Status = %q, want %q", drafts[0].Status, evolution.DraftStatusQuarantined)
	}
	if len(drafts[0].ScanFindings) == 0 {
		t.Fatal("expected scan findings for invalid draft")
	}
}

func TestRuntime_RunColdPathOnce_DoesNotWriteSkillFile(t *testing.T) {
	root := t.TempDir()
	skillPath := filepath.Join(root, "skills", "weather", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(skillPath), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(skillPath, []byte("---\nname: weather\ndescription: test\n---\n# Weather"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

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

	original, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("ReadFile(original): %v", err)
	}

	rt, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config: config.EvolutionConfig{Enabled: true, Mode: "apply"},
		DraftGenerator: stubDraftGenerator{
			draft: evolution.SkillDraft{
				ID:              "draft-1",
				WorkspaceID:     root,
				SourceRecordID:  "rule-1",
				TargetSkillName: "weather",
				DraftType:       evolution.DraftTypeShortcut,
				ChangeKind:      evolution.ChangeKindAppend,
				HumanSummary:    "prefer native-name path first",
				BodyOrPatch:     "## Start Here\nUse native-name query first.",
			},
		},
		Store:          store,
		SkillsRecaller: evolution.NewSkillsRecaller(root),
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	if err := rt.RunColdPathOnce(context.Background(), root); err != nil {
		t.Fatalf("RunColdPathOnce: %v", err)
	}

	got, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("ReadFile(after): %v", err)
	}
	if string(got) != string(original) {
		t.Fatalf("skill file changed unexpectedly:\n%s", string(got))
	}
}

func TestRuntime_RunColdPathOnce_UsesDefaultDraftGenerator(t *testing.T) {
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
		SuccessRate: 1,
		WinningPath: []string{"weather"},
	}
	if err := store.AppendLearningRecords([]evolution.LearningRecord{rule}); err != nil {
		t.Fatalf("AppendLearningRecords: %v", err)
	}

	rt, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config: config.EvolutionConfig{Enabled: true, Mode: "draft"},
		Store:  store,
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	if err := rt.RunColdPathOnce(context.Background(), root); err != nil {
		t.Fatalf("RunColdPathOnce: %v", err)
	}

	drafts, err := store.LoadDrafts()
	if err != nil {
		t.Fatalf("LoadDrafts: %v", err)
	}
	if len(drafts) != 1 {
		t.Fatalf("len(drafts) = %d, want 1", len(drafts))
	}
	if drafts[0].TargetSkillName != "weather" {
		t.Fatalf("TargetSkillName = %q, want weather", drafts[0].TargetSkillName)
	}
	if drafts[0].Status != evolution.DraftStatusCandidate {
		t.Fatalf("Status = %q, want %q", drafts[0].Status, evolution.DraftStatusCandidate)
	}
	if drafts[0].BodyOrPatch == "" {
		t.Fatal("expected generated draft body")
	}
}

func TestRuntime_RunColdPathOnce_UsesLLMDraftGeneratorWhenProviderAvailable(t *testing.T) {
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
		SuccessRate: 1,
		WinningPath: []string{"weather"},
	}
	if err := store.AppendLearningRecords([]evolution.LearningRecord{rule}); err != nil {
		t.Fatalf("AppendLearningRecords: %v", err)
	}

	provider := &llmDraftRuntimeProvider{
		response: &providers.LLMResponse{
			Content: `{"target_skill_name":"weather","draft_type":"shortcut","change_kind":"append","human_summary":"Prefer native-name path first","body_or_patch":"## Start Here\nUse native-name query first."}`,
		},
	}
	rt, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config:         config.EvolutionConfig{Enabled: true, Mode: "draft"},
		Store:          store,
		DraftGenerator: evolution.NewDraftGeneratorForWorkspace(root, provider, "runtime-explicit-model"),
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	if err := rt.RunColdPathOnce(context.Background(), root); err != nil {
		t.Fatalf("RunColdPathOnce: %v", err)
	}

	drafts, err := store.LoadDrafts()
	if err != nil {
		t.Fatalf("LoadDrafts: %v", err)
	}
	if len(drafts) != 1 {
		t.Fatalf("len(drafts) = %d, want 1", len(drafts))
	}
	if provider.calls != 1 {
		t.Fatalf("provider.calls = %d, want 1", provider.calls)
	}
	if drafts[0].HumanSummary != "Prefer native-name path first" {
		t.Fatalf("HumanSummary = %q, want %q", drafts[0].HumanSummary, "Prefer native-name path first")
	}
}

func TestRuntime_RunColdPathOnce_UsesDefaultDraftGeneratorWhenFactoryHasNoProvider(t *testing.T) {
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
		SuccessRate: 1,
		WinningPath: []string{"weather"},
	}
	if err := store.AppendLearningRecords([]evolution.LearningRecord{rule}); err != nil {
		t.Fatalf("AppendLearningRecords: %v", err)
	}

	rt, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config:         config.EvolutionConfig{Enabled: true, Mode: "draft"},
		Store:          store,
		DraftGenerator: evolution.NewDraftGeneratorForWorkspace(root, nil, ""),
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	if err := rt.RunColdPathOnce(context.Background(), root); err != nil {
		t.Fatalf("RunColdPathOnce: %v", err)
	}

	drafts, err := store.LoadDrafts()
	if err != nil {
		t.Fatalf("LoadDrafts: %v", err)
	}
	if len(drafts) != 1 {
		t.Fatalf("len(drafts) = %d, want 1", len(drafts))
	}
	if drafts[0].TargetSkillName != "weather" {
		t.Fatalf("TargetSkillName = %q, want weather", drafts[0].TargetSkillName)
	}
	if drafts[0].BodyOrPatch == "" {
		t.Fatal("expected generated draft body")
	}
}

func TestRuntime_RunColdPathOnce_UsesGeneratorFactoryWorkspaceForFallback(t *testing.T) {
	root := t.TempDir()
	store := evolution.NewStore(evolution.NewPaths(root, ""))

	if err := os.MkdirAll(filepath.Join(root, "skills", "weather"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	skillBody := "---\nname: weather\ndescription: workspace weather helper\n---\n# Weather\n## Start Here\nUse the workspace-specific path.\n"
	if err := os.WriteFile(filepath.Join(root, "skills", "weather", "SKILL.md"), []byte(skillBody), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	rule := evolution.LearningRecord{
		ID:          "rule-1",
		Kind:        evolution.RecordKindRule,
		WorkspaceID: root,
		CreatedAt:   time.Unix(1700000000, 0).UTC(),
		Summary:     "weather native-name path",
		Status:      evolution.RecordStatus("ready"),
		EventCount:  4,
		SuccessRate: 1,
		WinningPath: []string{"weather"},
	}
	if err := store.AppendLearningRecords([]evolution.LearningRecord{rule}); err != nil {
		t.Fatalf("AppendLearningRecords: %v", err)
	}

	provider := &llmDraftRuntimeProvider{
		response:     &providers.LLMResponse{Content: `not-json`},
		defaultModel: "runtime-test-model",
	}

	rt, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config: config.EvolutionConfig{Enabled: true, Mode: "draft"},
		Store:  store,
		GeneratorFactory: func(workspace string) evolution.DraftGenerator {
			return evolution.NewDraftGeneratorForWorkspace(workspace, provider, "runtime-explicit-model")
		},
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	if err := rt.RunColdPathOnce(context.Background(), root); err != nil {
		t.Fatalf("RunColdPathOnce: %v", err)
	}

	drafts, err := store.LoadDrafts()
	if err != nil {
		t.Fatalf("LoadDrafts: %v", err)
	}
	if len(drafts) != 1 {
		t.Fatalf("len(drafts) = %d, want 1", len(drafts))
	}
	if drafts[0].ChangeKind != evolution.ChangeKindAppend {
		t.Fatalf("ChangeKind = %q, want %q", drafts[0].ChangeKind, evolution.ChangeKindAppend)
	}
	if !strings.Contains(drafts[0].BodyOrPatch, "## Learned Evolution") {
		t.Fatalf("BodyOrPatch = %q, want appended learned evolution section", drafts[0].BodyOrPatch)
	}
}

func TestRuntime_RunColdPathOnce_PersistsEarlierDraftWhenLaterRuleFails(t *testing.T) {
	root := t.TempDir()
	store := evolution.NewStore(evolution.NewPaths(root, ""))

	rules := []evolution.LearningRecord{
		{
			ID:          "rule-1",
			Kind:        evolution.RecordKindRule,
			WorkspaceID: root,
			CreatedAt:   time.Unix(1700000000, 0).UTC(),
			Summary:     "weather native-name path",
			Status:      evolution.RecordStatus("ready"),
			EventCount:  4,
		},
		{
			ID:          "rule-2",
			Kind:        evolution.RecordKindRule,
			WorkspaceID: root,
			CreatedAt:   time.Unix(1700000100, 0).UTC(),
			Summary:     "release path",
			Status:      evolution.RecordStatus("ready"),
			EventCount:  4,
		},
	}
	if err := store.AppendLearningRecords(rules); err != nil {
		t.Fatalf("AppendLearningRecords: %v", err)
	}

	generator := &sequenceDraftGenerator{
		results: []draftGenerationResult{
			{
				draft: evolution.SkillDraft{
					ID:              "draft-1",
					TargetSkillName: "weather",
					DraftType:       evolution.DraftTypeShortcut,
					ChangeKind:      evolution.ChangeKindAppend,
					HumanSummary:    "prefer native-name path first",
					BodyOrPatch:     "## Start Here\nUse native-name query first.",
				},
			},
			{
				err: context.DeadlineExceeded,
			},
		},
	}

	rt, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config:         config.EvolutionConfig{Enabled: true, Mode: "draft"},
		Store:          store,
		DraftGenerator: generator,
		SkillsRecaller: evolution.NewSkillsRecaller(root),
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	err = rt.RunColdPathOnce(context.Background(), root)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("RunColdPathOnce error = %v, want %v", err, context.DeadlineExceeded)
	}

	drafts, loadErr := store.LoadDrafts()
	if loadErr != nil {
		t.Fatalf("LoadDrafts: %v", loadErr)
	}
	if len(drafts) != 1 {
		t.Fatalf("len(drafts) = %d, want 1", len(drafts))
	}
	if drafts[0].SourceRecordID != "rule-1" {
		t.Fatalf("SourceRecordID = %q, want rule-1", drafts[0].SourceRecordID)
	}
}

func TestRuntime_RunColdPathOnce_RegeneratesAfterQuarantinedDraft(t *testing.T) {
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
	if err := store.SaveDrafts([]evolution.SkillDraft{{
		ID:              "draft-old",
		WorkspaceID:     root,
		CreatedAt:       time.Unix(1700000100, 0).UTC(),
		SourceRecordID:  "rule-1",
		TargetSkillName: "weather",
		DraftType:       evolution.DraftTypeShortcut,
		ChangeKind:      evolution.ChangeKindAppend,
		HumanSummary:    "broken attempt",
		BodyOrPatch:     "## Start Here\nBroken content.",
		Status:          evolution.DraftStatusQuarantined,
		ScanFindings:    []string{"apply failed"},
	}}); err != nil {
		t.Fatalf("SaveDrafts: %v", err)
	}

	rt, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config: config.EvolutionConfig{Enabled: true, Mode: "draft"},
		Store:  store,
		DraftGenerator: stubDraftGenerator{
			draft: evolution.SkillDraft{
				ID:              "draft-new",
				TargetSkillName: "weather",
				DraftType:       evolution.DraftTypeShortcut,
				ChangeKind:      evolution.ChangeKindAppend,
				HumanSummary:    "fixed attempt",
				BodyOrPatch:     "## Start Here\nUse native-name query first.",
			},
		},
		SkillsRecaller: evolution.NewSkillsRecaller(root),
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	if err := rt.RunColdPathOnce(context.Background(), root); err != nil {
		t.Fatalf("RunColdPathOnce: %v", err)
	}

	drafts, err := store.LoadDrafts()
	if err != nil {
		t.Fatalf("LoadDrafts: %v", err)
	}
	if len(drafts) != 2 {
		t.Fatalf("len(drafts) = %d, want 2", len(drafts))
	}
	if drafts[1].ID != "draft-new" {
		t.Fatalf("drafts[1].ID = %q, want draft-new", drafts[1].ID)
	}
}

type llmDraftRuntimeProvider struct {
	response     *providers.LLMResponse
	err          error
	calls        int
	defaultModel string
}

func (p *llmDraftRuntimeProvider) Chat(
	_ context.Context,
	_ []providers.Message,
	_ []providers.ToolDefinition,
	_ string,
	_ map[string]any,
) (*providers.LLMResponse, error) {
	p.calls++
	return p.response, p.err
}

func (p *llmDraftRuntimeProvider) GetDefaultModel() string {
	if p.defaultModel != "" {
		return p.defaultModel
	}
	return "runtime-test-model"
}
