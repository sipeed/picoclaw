package evolution_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/evolution"
)

func TestRuntime_FinalizeTurnDisabledDoesNothing(t *testing.T) {
	rt, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config: config.EvolutionConfig{Enabled: false, Mode: "observe"},
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	workspace := t.TempDir()
	err = rt.FinalizeTurn(context.Background(), evolution.TurnCaseInput{
		Workspace: workspace,
		TurnID:    "turn-1",
		Status:    "completed",
	})
	if err != nil {
		t.Fatalf("FinalizeTurn: %v", err)
	}

	paths := evolution.NewPaths(workspace, "")
	if _, err := os.Stat(paths.TaskRecords); !os.IsNotExist(err) {
		t.Fatalf("task records file should not exist, stat err = %v", err)
	}
}

func TestRuntime_FinalizeTurnWithEmptyWorkspaceDoesNothing(t *testing.T) {
	rt, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config: config.EvolutionConfig{Enabled: true, Mode: "observe"},
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	if err := rt.FinalizeTurn(context.Background(), evolution.TurnCaseInput{
		TurnID: "turn-1",
		Status: "completed",
	}); err != nil {
		t.Fatalf("FinalizeTurn: %v", err)
	}
}

func TestRuntime_FinalizeTurnSkipsHeartbeat(t *testing.T) {
	workspace := t.TempDir()
	rt, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config: config.EvolutionConfig{Enabled: true, Mode: "apply"},
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	if err := rt.FinalizeTurn(context.Background(), evolution.TurnCaseInput{
		Workspace:    workspace,
		TurnID:       "heartbeat-turn",
		SessionKey:   "heartbeat",
		Status:       "completed",
		UserMessage:  "# Heartbeat Check",
		FinalContent: "HEARTBEAT_OK",
	}); err != nil {
		t.Fatalf("FinalizeTurn: %v", err)
	}

	paths := evolution.NewPaths(workspace, "")
	if _, err := os.Stat(paths.TaskRecords); !os.IsNotExist(err) {
		t.Fatalf("heartbeat should not create task records, stat err = %v", err)
	}
}

func TestRuntime_FinalizeTurnWritesRecordWithOverride(t *testing.T) {
	workspace := t.TempDir()
	override := filepath.Join(t.TempDir(), "custom-state")
	now := time.Unix(1700000000, 0).UTC()

	rt, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config: config.EvolutionConfig{
			Enabled:  true,
			Mode:     "observe",
			StateDir: override,
		},
		Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	if err := rt.FinalizeTurn(context.Background(), evolution.TurnCaseInput{
		Workspace:    workspace,
		TurnID:       "turn-1",
		SessionKey:   "session-1",
		AgentID:      "agent-1",
		Status:       "completed",
		UserMessage:  "summarize the release notes",
		FinalContent: "Here is the summary.",
		ToolKinds:    []string{"web", "read_file"},
		ToolExecutions: []evolution.ToolExecutionRecord{
			{Name: "web", Success: true},
			{Name: "read_file", Success: true},
		},
		ActiveSkillNames: []string{"skill-a"},
	}); err != nil {
		t.Fatalf("FinalizeTurn first call: %v", err)
	}

	if err := rt.FinalizeTurn(context.Background(), evolution.TurnCaseInput{
		Workspace:    workspace,
		WorkspaceID:  "ws-explicit",
		TurnID:       "turn-2",
		SessionKey:   "session-2",
		AgentID:      "agent-2",
		Status:       "error",
		UserMessage:  "run the bash command",
		FinalContent: "bash failed",
		ToolKinds:    []string{"bash"},
		ToolExecutions: []evolution.ToolExecutionRecord{
			{Name: "bash", Success: false, ErrorSummary: "exit status 1"},
		},
		ActiveSkillNames: []string{"skill-b"},
	}); err != nil {
		t.Fatalf("FinalizeTurn second call: %v", err)
	}

	paths := evolution.NewPaths(workspace, override)
	data, err := os.ReadFile(paths.TaskRecords)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("record file line count = %d, want 2", len(lines))
	}

	var first evolution.LearningRecord
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("Unmarshal first record: %v", err)
	}
	if first.WorkspaceID != workspace {
		t.Fatalf("first WorkspaceID = %q, want %q", first.WorkspaceID, workspace)
	}
	if first.CreatedAt != now {
		t.Fatalf("first CreatedAt = %v, want %v", first.CreatedAt, now)
	}
	if first.SessionKey != "session-1" {
		t.Fatalf("first SessionKey = %q, want %q", first.SessionKey, "session-1")
	}
	if first.Summary != "summarize the release notes" {
		t.Fatalf("first Summary = %q", first.Summary)
	}
	if first.FinalOutput != "Here is the summary." {
		t.Fatalf("first FinalOutput = %q", first.FinalOutput)
	}
	if first.Success == nil || !*first.Success {
		t.Fatalf("first Success = %v, want true", first.Success)
	}
	if len(first.AddedSkillNames) != 0 {
		t.Fatalf("first AddedSkillNames = %v, want empty", first.AddedSkillNames)
	}
	if len(first.UsedSkillNames) != 0 {
		t.Fatalf("first UsedSkillNames = %v, want empty", first.UsedSkillNames)
	}
	if len(first.ToolKinds) != 0 || len(first.ToolExecutions) != 0 || first.Source != nil || first.AttemptTrail != nil {
		t.Fatalf("first record should be slimmed: %+v", first)
	}
	if first.TaskHash != "" || len(first.Signals) != 0 {
		t.Fatalf("first record should not persist task_hash/signals: %+v", first)
	}

	var second evolution.LearningRecord
	if err := json.Unmarshal([]byte(lines[1]), &second); err != nil {
		t.Fatalf("Unmarshal second record: %v", err)
	}
	if second.WorkspaceID != workspace {
		t.Fatalf("second WorkspaceID = %q, want %q", second.WorkspaceID, workspace)
	}
	if second.SessionKey != "session-2" {
		t.Fatalf("second SessionKey = %q, want %q", second.SessionKey, "session-2")
	}
	if second.Summary != "run the bash command" {
		t.Fatalf("second Summary = %q", second.Summary)
	}
	if second.Success == nil || *second.Success {
		t.Fatalf("second Success = %v, want false", second.Success)
	}
	if len(second.ToolExecutions) != 0 || second.Source != nil || second.AttemptTrail != nil {
		t.Fatalf("second record should be slimmed: %+v", second)
	}
	if second.TaskHash != "" || len(second.Signals) != 0 {
		t.Fatalf("second record should not persist task_hash/signals: %+v", second)
	}
}

func TestRuntime_FinalizeTurnGeneratesUniqueTaskRecordIDsAcrossRestartedTurnSequence(t *testing.T) {
	workspace := t.TempDir()
	createdAt := time.Unix(1700000000, 0).UTC()
	rt, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config: config.EvolutionConfig{Enabled: true, Mode: "observe"},
		Now: func() time.Time {
			createdAt = createdAt.Add(time.Second)
			return createdAt
		},
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	input := evolution.TurnCaseInput{
		Workspace:    workspace,
		TurnID:       "main-turn-1",
		SessionKey:   "session-a",
		AgentID:      "main",
		Status:       "completed",
		UserMessage:  "summarize release notes",
		FinalContent: "done",
	}
	if err := rt.FinalizeTurn(context.Background(), input); err != nil {
		t.Fatalf("FinalizeTurn first: %v", err)
	}
	input.SessionKey = "session-b"
	if err := rt.FinalizeTurn(context.Background(), input); err != nil {
		t.Fatalf("FinalizeTurn second: %v", err)
	}

	store := evolution.NewStore(evolution.NewPaths(workspace, ""))
	records, err := store.LoadTaskRecords()
	if err != nil {
		t.Fatalf("LoadTaskRecords: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("len(records) = %d, want 2: %#v", len(records), records)
	}
	if records[0].ID == records[1].ID {
		t.Fatalf("record IDs should be unique across repeated turn IDs: %#v", records)
	}
	for _, record := range records {
		if !strings.HasPrefix(record.ID, "main-turn-1-") {
			t.Fatalf("record ID = %q, want main-turn-1-*", record.ID)
		}
	}
}

func TestRuntime_FinalizeTurnSharedStateKeepsSkillProfilesScoped(t *testing.T) {
	sharedState := t.TempDir()
	workspaceA := t.TempDir()
	workspaceB := t.TempDir()
	now := time.Unix(1700000000, 0).UTC()

	storeA := evolution.NewStore(evolution.NewPaths(workspaceA, sharedState))
	if err := storeA.SaveProfile(evolution.SkillProfile{
		SkillName:      "weather",
		WorkspaceID:    workspaceA,
		CurrentVersion: "draft-a",
		Status:         evolution.SkillStatusActive,
		Origin:         "evolved",
		HumanSummary:   "workspace A weather helper",
		LastUsedAt:     now,
		UseCount:       7,
		RetentionScore: 0.9,
	}); err != nil {
		t.Fatalf("storeA.SaveProfile: %v", err)
	}

	rt, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config: config.EvolutionConfig{
			Enabled:  true,
			Mode:     "observe",
			StateDir: sharedState,
		},
		Now: func() time.Time { return now.Add(time.Minute) },
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	if err := rt.FinalizeTurn(context.Background(), evolution.TurnCaseInput{
		Workspace:        workspaceA,
		TurnID:           "turn-a",
		SessionKey:       "session-a",
		Status:           "completed",
		ActiveSkillNames: []string{"weather"},
	}); err != nil {
		t.Fatalf("FinalizeTurn(workspaceA): %v", err)
	}

	if err := rt.FinalizeTurn(context.Background(), evolution.TurnCaseInput{
		Workspace:        workspaceB,
		TurnID:           "turn-b",
		SessionKey:       "session-b",
		Status:           "completed",
		ActiveSkillNames: []string{"weather"},
	}); err != nil {
		t.Fatalf("FinalizeTurn(workspaceB): %v", err)
	}

	loadedA, err := storeA.LoadProfile("weather")
	if err != nil {
		t.Fatalf("storeA.LoadProfile: %v", err)
	}
	if loadedA.WorkspaceID != workspaceA {
		t.Fatalf("workspace A profile WorkspaceID = %q, want %q", loadedA.WorkspaceID, workspaceA)
	}
	if loadedA.UseCount != 8 {
		t.Fatalf("workspace A profile UseCount = %d, want 8", loadedA.UseCount)
	}

	storeB := evolution.NewStore(evolution.NewPaths(workspaceB, sharedState))
	loadedB, err := storeB.LoadProfile("weather")
	if err != nil {
		t.Fatalf("storeB.LoadProfile: %v", err)
	}
	if loadedB.WorkspaceID != workspaceB {
		t.Fatalf("workspace B profile WorkspaceID = %q, want %q", loadedB.WorkspaceID, workspaceB)
	}
	if loadedB.UseCount != 1 {
		t.Fatalf("workspace B profile UseCount = %d, want 1", loadedB.UseCount)
	}
	if loadedB.Origin != "manual" {
		t.Fatalf("workspace B profile Origin = %q, want manual", loadedB.Origin)
	}
	if loadedB.CurrentVersion != "" {
		t.Fatalf("workspace B profile CurrentVersion = %q, want empty", loadedB.CurrentVersion)
	}
}

func TestRuntime_FinalizeTurnWritesPotentiallyLearnableSignal(t *testing.T) {
	workspace := t.TempDir()
	now := time.Unix(1700003000, 0).UTC()

	rt, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config: config.EvolutionConfig{Enabled: true, Mode: "observe"},
		Now:    func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	if err := rt.FinalizeTurn(context.Background(), evolution.TurnCaseInput{
		Workspace:        workspace,
		TurnID:           "turn-learnable",
		SessionKey:       "session-learnable",
		AgentID:          "agent-1",
		Status:           "completed",
		ToolKinds:        []string{"web", "bash"},
		ActiveSkillNames: []string{"geocode", "weather"},
		FinalContent:     "weather workflow completed",
		FinalSuccessfulPath: []string{
			"weather",
		},
		SkillContextSnapshots: []evolution.SkillContextSnapshot{
			{Sequence: 1, Trigger: "initial_build", SkillNames: []string{"geocode"}},
			{Sequence: 2, Trigger: "context_retry_rebuild", SkillNames: []string{"geocode", "weather"}},
		},
	}); err != nil {
		t.Fatalf("FinalizeTurn: %v", err)
	}

	paths := evolution.NewPaths(workspace, "")
	data, err := os.ReadFile(paths.TaskRecords)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Fatalf("record file line count = %d, want 1", len(lines))
	}

	var record evolution.LearningRecord
	if err := json.Unmarshal([]byte(lines[0]), &record); err != nil {
		t.Fatalf("Unmarshal record: %v", err)
	}
	if len(record.Signals) != 0 {
		t.Fatalf("Signals = %v, want empty", record.Signals)
	}
	if got := record.InitialSkillNames; len(got) != 0 {
		t.Fatalf("InitialSkillNames = %v, want empty", got)
	}
	if got := record.AddedSkillNames; len(got) != 0 {
		t.Fatalf("AddedSkillNames = %v, want empty", got)
	}
	if got := record.UsedSkillNames; len(got) != 1 || got[0] != "weather" {
		t.Fatalf("UsedSkillNames = %v, want [weather]", got)
	}
	if got := record.AllLoadedSkillNames; len(got) != 0 {
		t.Fatalf("AllLoadedSkillNames = %v, want empty", got)
	}
	if record.AttemptTrail != nil {
		t.Fatalf("AttemptTrail = %+v, want nil", record.AttemptTrail)
	}
}

func TestRuntime_FinalizeTurnUsesSkillNamesFromToolExecutions(t *testing.T) {
	workspace := t.TempDir()
	rt, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config: config.EvolutionConfig{Enabled: true, Mode: "apply"},
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	if err := rt.FinalizeTurn(context.Background(), evolution.TurnCaseInput{
		Workspace:    workspace,
		TurnID:       "turn-skill-chain",
		SessionKey:   "session-skill-chain",
		AgentID:      "main",
		Status:       "completed",
		UserMessage:  "调用三一定理计算100",
		FinalContent: "done",
		ToolExecutions: []evolution.ToolExecutionRecord{
			{Name: "read_file", Success: true, SkillNames: []string{"three-one"}},
			{Name: "read_file", Success: true, SkillNames: []string{"four-two"}},
			{Name: "read_file", Success: true, SkillNames: []string{"five-three"}},
		},
	}); err != nil {
		t.Fatalf("FinalizeTurn: %v", err)
	}

	paths := evolution.NewPaths(workspace, "")
	data, err := os.ReadFile(paths.TaskRecords)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Fatalf("record file line count = %d, want 1", len(lines))
	}

	var record evolution.LearningRecord
	if err := json.Unmarshal([]byte(lines[0]), &record); err != nil {
		t.Fatalf("Unmarshal record: %v", err)
	}
	if got := record.AddedSkillNames; len(got) != 0 {
		t.Fatalf("AddedSkillNames = %v, want empty", got)
	}
	if got := record.UsedSkillNames; len(got) != 3 || got[0] != "three-one" || got[1] != "four-two" || got[2] != "five-three" {
		t.Fatalf("UsedSkillNames = %v, want [three-one four-two five-three]", got)
	}
	if got := record.AllLoadedSkillNames; len(got) != 0 {
		t.Fatalf("AllLoadedSkillNames = %v, want empty", got)
	}
}

func TestRuntime_FinalizeTurnPreservesUTF8WhenTruncatingChineseOutput(t *testing.T) {
	workspace := t.TempDir()
	rt, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config: config.EvolutionConfig{Enabled: true, Mode: "apply"},
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	longChinese := strings.Repeat("中文输出", 500)
	if err := rt.FinalizeTurn(context.Background(), evolution.TurnCaseInput{
		Workspace:    workspace,
		TurnID:       "turn-utf8",
		SessionKey:   "session-utf8",
		AgentID:      "main",
		Status:       "completed",
		UserMessage:  "请处理这段中文输出",
		FinalContent: longChinese,
	}); err != nil {
		t.Fatalf("FinalizeTurn: %v", err)
	}

	paths := evolution.NewPaths(workspace, "")
	data, err := os.ReadFile(paths.TaskRecords)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Fatalf("record file line count = %d, want 1", len(lines))
	}

	var record evolution.LearningRecord
	if err := json.Unmarshal([]byte(lines[0]), &record); err != nil {
		t.Fatalf("Unmarshal record: %v", err)
	}
	if !utf8.ValidString(record.FinalOutput) {
		t.Fatalf("FinalOutput is not valid UTF-8: %q", record.FinalOutput)
	}
	if strings.ContainsRune(record.FinalOutput, '\uFFFD') {
		t.Fatalf("FinalOutput contains replacement rune: %q", record.FinalOutput)
	}
	if !strings.HasSuffix(record.FinalOutput, "...") {
		t.Fatalf("FinalOutput = %q, want truncated suffix ...", record.FinalOutput)
	}
}

func TestRuntime_FinalizeTurnPrefersExplicitAttemptTrail(t *testing.T) {
	workspace := t.TempDir()
	now := time.Unix(1700003500, 0).UTC()

	rt, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config: config.EvolutionConfig{Enabled: true, Mode: "observe"},
		Now:    func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	if err := rt.FinalizeTurn(context.Background(), evolution.TurnCaseInput{
		Workspace:           workspace,
		TurnID:              "turn-explicit-trail",
		SessionKey:          "session-explicit-trail",
		AgentID:             "agent-1",
		Status:              "completed",
		ToolKinds:           []string{"web"},
		ActiveSkillNames:    []string{"weather"},
		AttemptedSkillNames: []string{"geocode", "weather"},
		FinalSuccessfulPath: []string{"geocode", "weather"},
		SkillContextSnapshots: []evolution.SkillContextSnapshot{
			{Sequence: 1, Trigger: "initial_build", SkillNames: []string{"weather"}},
			{Sequence: 2, Trigger: "context_retry_rebuild", SkillNames: []string{"geocode", "weather"}},
		},
	}); err != nil {
		t.Fatalf("FinalizeTurn: %v", err)
	}

	paths := evolution.NewPaths(workspace, "")
	data, err := os.ReadFile(paths.TaskRecords)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Fatalf("record file line count = %d, want 1", len(lines))
	}

	var record evolution.LearningRecord
	if err := json.Unmarshal([]byte(lines[0]), &record); err != nil {
		t.Fatalf("Unmarshal record: %v", err)
	}
	if record.AttemptTrail != nil {
		t.Fatalf("AttemptTrail = %+v, want nil", record.AttemptTrail)
	}
	if got := record.UsedSkillNames; len(got) != 2 || got[0] != "geocode" || got[1] != "weather" {
		t.Fatalf("UsedSkillNames = %v, want [geocode weather]", got)
	}
	if got := record.InitialSkillNames; len(got) != 0 {
		t.Fatalf("InitialSkillNames = %v, want empty", got)
	}
	if got := record.AddedSkillNames; len(got) != 0 {
		t.Fatalf("AddedSkillNames = %v, want empty", got)
	}
	if len(record.Signals) != 0 {
		t.Fatalf("Signals = %v, want empty", record.Signals)
	}
}

func TestRuntime_FinalizeTurnUpdatesSkillProfileUsage(t *testing.T) {
	workspace := t.TempDir()
	now := time.Unix(1700000000, 0).UTC()

	rt, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config: config.EvolutionConfig{
			Enabled: true,
			Mode:    "observe",
		},
		Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	if err := rt.FinalizeTurn(context.Background(), evolution.TurnCaseInput{
		Workspace:        workspace,
		TurnID:           "turn-1",
		SessionKey:       "session-1",
		AgentID:          "agent-1",
		Status:           "completed",
		ActiveSkillNames: []string{"skill-a", "skill-a"},
	}); err != nil {
		t.Fatalf("FinalizeTurn: %v", err)
	}

	store := evolution.NewStore(evolution.NewPaths(workspace, ""))
	profile, err := store.LoadProfile("skill-a")
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	if profile.Origin != "manual" {
		t.Fatalf("Origin = %q, want manual", profile.Origin)
	}
	if profile.UseCount != 1 {
		t.Fatalf("UseCount = %d, want 1", profile.UseCount)
	}
	if profile.LastUsedAt != now {
		t.Fatalf("LastUsedAt = %v, want %v", profile.LastUsedAt, now)
	}
	if profile.RetentionScore <= 0.2 {
		t.Fatalf("RetentionScore = %v, want > 0.2", profile.RetentionScore)
	}
}

func TestRuntime_FinalizeTurnReactivatesColdSkill(t *testing.T) {
	workspace := t.TempDir()
	now := time.Unix(1700001000, 0).UTC()
	store := evolution.NewStore(evolution.NewPaths(workspace, ""))

	if err := store.SaveProfile(evolution.SkillProfile{
		SkillName:      "skill-cold",
		WorkspaceID:    workspace,
		Status:         evolution.SkillStatusCold,
		Origin:         "evolved",
		HumanSummary:   "cold skill",
		LastUsedAt:     now.Add(-24 * time.Hour),
		UseCount:       2,
		RetentionScore: 0.2,
	}); err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}

	rt, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config: config.EvolutionConfig{Enabled: true, Mode: "observe"},
		Now:    func() time.Time { return now },
		Store:  store,
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	if err := rt.FinalizeTurn(context.Background(), evolution.TurnCaseInput{
		Workspace:        workspace,
		TurnID:           "turn-cold",
		Status:           "completed",
		ActiveSkillNames: []string{"skill-cold"},
	}); err != nil {
		t.Fatalf("FinalizeTurn: %v", err)
	}

	profile, err := store.LoadProfile("skill-cold")
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	if profile.Status != evolution.SkillStatusActive {
		t.Fatalf("Status = %q, want %q", profile.Status, evolution.SkillStatusActive)
	}
}

func TestRuntime_FinalizeTurnReactivatesArchivedSkill(t *testing.T) {
	workspace := t.TempDir()
	now := time.Unix(1700002000, 0).UTC()
	store := evolution.NewStore(evolution.NewPaths(workspace, ""))

	if err := store.SaveProfile(evolution.SkillProfile{
		SkillName:      "skill-archived",
		WorkspaceID:    workspace,
		Status:         evolution.SkillStatusArchived,
		Origin:         "evolved",
		HumanSummary:   "archived skill",
		LastUsedAt:     now.Add(-48 * time.Hour),
		UseCount:       5,
		RetentionScore: 0.1,
	}); err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}

	rt, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config: config.EvolutionConfig{Enabled: true, Mode: "observe"},
		Now:    func() time.Time { return now },
		Store:  store,
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	if err := rt.FinalizeTurn(context.Background(), evolution.TurnCaseInput{
		Workspace:        workspace,
		TurnID:           "turn-archived",
		Status:           "completed",
		ActiveSkillNames: []string{"skill-archived"},
	}); err != nil {
		t.Fatalf("FinalizeTurn: %v", err)
	}

	profile, err := store.LoadProfile("skill-archived")
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	if profile.Status != evolution.SkillStatusActive {
		t.Fatalf("Status = %q, want %q", profile.Status, evolution.SkillStatusActive)
	}
}
