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
	if _, err := os.Stat(paths.LearningRecords); !os.IsNotExist(err) {
		t.Fatalf("learning records file should not exist, stat err = %v", err)
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
	if _, err := os.Stat(paths.LearningRecords); !os.IsNotExist(err) {
		t.Fatalf("heartbeat should not create learning records, stat err = %v", err)
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
	data, err := os.ReadFile(paths.LearningRecords)
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
	if first.UserGoal != "summarize the release notes" {
		t.Fatalf("first UserGoal = %q", first.UserGoal)
	}
	if first.FinalOutput != "Here is the summary." {
		t.Fatalf("first FinalOutput = %q", first.FinalOutput)
	}
	if first.Success == nil || !*first.Success {
		t.Fatalf("first Success = %v, want true", first.Success)
	}
	if len(first.ToolKinds) != 2 || first.ToolKinds[0] != "web" || first.ToolKinds[1] != "read_file" {
		t.Fatalf("first ToolKinds = %v", first.ToolKinds)
	}
	if len(first.ToolExecutions) != 2 || !first.ToolExecutions[0].Success || first.ToolExecutions[0].Name != "web" {
		t.Fatalf("first ToolExecutions = %+v", first.ToolExecutions)
	}
	if len(first.InitialSkillNames) != 1 || first.InitialSkillNames[0] != "skill-a" {
		t.Fatalf("first InitialSkillNames = %v", first.InitialSkillNames)
	}
	if len(first.AddedSkillNames) != 0 {
		t.Fatalf("first AddedSkillNames = %v, want empty", first.AddedSkillNames)
	}
	if len(first.UsedSkillNames) != 0 {
		t.Fatalf("first UsedSkillNames = %v, want empty", first.UsedSkillNames)
	}
	if len(first.AllLoadedSkillNames) != 1 || first.AllLoadedSkillNames[0] != "skill-a" {
		t.Fatalf("first AllLoadedSkillNames = %v", first.AllLoadedSkillNames)
	}
	if len(first.ActiveSkillNames) != 1 || first.ActiveSkillNames[0] != "skill-a" {
		t.Fatalf("first ActiveSkillNames = %v", first.ActiveSkillNames)
	}
	if first.AttemptTrail == nil {
		t.Fatal("first AttemptTrail should not be nil")
	}
	if len(first.AttemptTrail.AttemptedSkills) != 1 || first.AttemptTrail.AttemptedSkills[0] != "skill-a" {
		t.Fatalf("first AttemptTrail.AttemptedSkills = %v, want [skill-a]", first.AttemptTrail.AttemptedSkills)
	}
	if len(first.AttemptTrail.FinalSuccessfulPath) != 1 || first.AttemptTrail.FinalSuccessfulPath[0] != "skill-a" {
		t.Fatalf(
			"first AttemptTrail.FinalSuccessfulPath = %v, want [skill-a]",
			first.AttemptTrail.FinalSuccessfulPath,
		)
	}
	if got := first.Source["turn_id"]; got != "turn-1" {
		t.Fatalf("first Source.turn_id = %v", got)
	}
	if got := first.Source["session_key"]; got != "session-1" {
		t.Fatalf("first Source.session_key = %v", got)
	}
	if got := first.Source["agent_id"]; got != "agent-1" {
		t.Fatalf("first Source.agent_id = %v", got)
	}
	if first.TaskHash == "" {
		t.Fatal("first TaskHash should not be empty")
	}
	if len(first.Signals) != 0 {
		t.Fatalf("first Signals = %v, want empty for single-skill success", first.Signals)
	}

	var second evolution.LearningRecord
	if err := json.Unmarshal([]byte(lines[1]), &second); err != nil {
		t.Fatalf("Unmarshal second record: %v", err)
	}
	if second.WorkspaceID != "ws-explicit" {
		t.Fatalf("second WorkspaceID = %q, want %q", second.WorkspaceID, "ws-explicit")
	}
	if second.SessionKey != "session-2" {
		t.Fatalf("second SessionKey = %q, want %q", second.SessionKey, "session-2")
	}
	if second.UserGoal != "run the bash command" {
		t.Fatalf("second UserGoal = %q", second.UserGoal)
	}
	if second.Success == nil || *second.Success {
		t.Fatalf("second Success = %v, want false", second.Success)
	}
	if len(second.ToolExecutions) != 1 || second.ToolExecutions[0].ErrorSummary != "exit status 1" {
		t.Fatalf("second ToolExecutions = %+v", second.ToolExecutions)
	}
	if second.AttemptTrail == nil {
		t.Fatal("second AttemptTrail should not be nil")
	}
	if len(second.AttemptTrail.AttemptedSkills) != 1 || second.AttemptTrail.AttemptedSkills[0] != "skill-b" {
		t.Fatalf("second AttemptTrail.AttemptedSkills = %v, want [skill-b]", second.AttemptTrail.AttemptedSkills)
	}
	if len(second.AttemptTrail.FinalSuccessfulPath) != 0 {
		t.Fatalf(
			"second AttemptTrail.FinalSuccessfulPath = %v, want empty for failed turn",
			second.AttemptTrail.FinalSuccessfulPath,
		)
	}
	if second.TaskHash == "" {
		t.Fatal("second TaskHash should not be empty")
	}
	if second.TaskHash == first.TaskHash {
		t.Fatal("TaskHash should differ for different skill/tool signatures")
	}
	if len(second.Signals) != 0 {
		t.Fatalf("second Signals = %v, want empty for failed turn", second.Signals)
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
		SkillContextSnapshots: []evolution.SkillContextSnapshot{
			{Sequence: 1, Trigger: "initial_build", SkillNames: []string{"geocode"}},
			{Sequence: 2, Trigger: "context_retry_rebuild", SkillNames: []string{"geocode", "weather"}},
		},
	}); err != nil {
		t.Fatalf("FinalizeTurn: %v", err)
	}

	paths := evolution.NewPaths(workspace, "")
	data, err := os.ReadFile(paths.LearningRecords)
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
	if len(record.Signals) != 1 || record.Signals[0] != "potentially_learnable" {
		t.Fatalf("Signals = %v, want [potentially_learnable]", record.Signals)
	}
	if got := record.InitialSkillNames; len(got) != 1 || got[0] != "geocode" {
		t.Fatalf("InitialSkillNames = %v, want [geocode]", got)
	}
	if got := record.AddedSkillNames; len(got) != 1 || got[0] != "weather" {
		t.Fatalf("AddedSkillNames = %v, want [weather]", got)
	}
	if got := record.UsedSkillNames; len(got) != 1 || got[0] != "weather" {
		t.Fatalf("UsedSkillNames = %v, want [weather]", got)
	}
	if got := record.AllLoadedSkillNames; len(got) != 2 || got[0] != "geocode" || got[1] != "weather" {
		t.Fatalf("AllLoadedSkillNames = %v, want [geocode weather]", got)
	}
	if record.AttemptTrail == nil {
		t.Fatal("AttemptTrail should not be nil")
	}
	if got := record.AttemptTrail.FinalSuccessfulPath; len(got) != 1 || got[0] != "weather" {
		t.Fatalf("FinalSuccessfulPath = %v, want [weather]", got)
	}
	if got := record.AttemptTrail.SkillContextSnapshots; len(got) != 0 {
		t.Fatalf("SkillContextSnapshots = %v, want empty", got)
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
	data, err := os.ReadFile(paths.LearningRecords)
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
	if got := record.AddedSkillNames; len(got) != 3 || got[0] != "three-one" || got[1] != "four-two" || got[2] != "five-three" {
		t.Fatalf("AddedSkillNames = %v, want [three-one four-two five-three]", got)
	}
	if got := record.UsedSkillNames; len(got) != 3 || got[0] != "three-one" || got[1] != "four-two" || got[2] != "five-three" {
		t.Fatalf("UsedSkillNames = %v, want [three-one four-two five-three]", got)
	}
	if got := record.AllLoadedSkillNames; len(got) != 3 || got[0] != "three-one" || got[1] != "four-two" || got[2] != "five-three" {
		t.Fatalf("AllLoadedSkillNames = %v, want [three-one four-two five-three]", got)
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

	longChinese := strings.Repeat("中文输出", 80)
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
	data, err := os.ReadFile(paths.LearningRecords)
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
	data, err := os.ReadFile(paths.LearningRecords)
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
	if record.AttemptTrail == nil {
		t.Fatal("AttemptTrail should not be nil")
	}
	if got := record.AttemptTrail.AttemptedSkills; len(got) != 2 || got[0] != "geocode" || got[1] != "weather" {
		t.Fatalf("AttemptedSkills = %v, want [geocode weather]", got)
	}
	if got := record.AttemptTrail.FinalSuccessfulPath; len(got) != 2 || got[0] != "geocode" || got[1] != "weather" {
		t.Fatalf("FinalSuccessfulPath = %v, want [geocode weather]", got)
	}
	if got := record.AttemptTrail.SkillContextSnapshots; len(got) != 0 {
		t.Fatalf("SkillContextSnapshots = %+v, want empty", got)
	}
	if got := record.InitialSkillNames; len(got) != 1 || got[0] != "weather" {
		t.Fatalf("InitialSkillNames = %v, want [weather]", got)
	}
	if got := record.AddedSkillNames; len(got) != 1 || got[0] != "geocode" {
		t.Fatalf("AddedSkillNames = %v, want [geocode]", got)
	}
	if len(record.Signals) != 1 || record.Signals[0] != "potentially_learnable" {
		t.Fatalf("Signals = %v, want [potentially_learnable]", record.Signals)
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
