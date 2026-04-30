package agent

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/evolution"
	"github.com/sipeed/picoclaw/pkg/providers"
)

func TestEvolutionBridge_DisabledWritesNothing(t *testing.T) {
	tmpDir := t.TempDir()
	al := newEvolutionTestLoop(t, tmpDir, config.EvolutionConfig{
		Enabled: false,
		Mode:    "observe",
	}, &simpleMockProvider{response: "ok"})
	defer al.Close()

	resp, err := al.ProcessDirectWithChannel(context.Background(), "hello", "session-disabled", "cli", "direct")
	if err != nil {
		t.Fatalf("ProcessDirectWithChannel failed: %v", err)
	}
	if resp != "ok" {
		t.Fatalf("response = %q, want %q", resp, "ok")
	}

	assertNotExists(t, filepath.Join(tmpDir, "state", "evolution", "learning-records.jsonl"))
	assertNotExists(t, filepath.Join(tmpDir, "state", "evolution", "skill-drafts.json"))
}

func TestEvolutionBridge_ObserveWritesCaseRecord(t *testing.T) {
	tmpDir := t.TempDir()
	provider := &toolCallRespProvider{
		toolName: "echo_text",
		toolArgs: map[string]any{"text": "bridge"},
		response: "done",
	}
	al := newEvolutionTestLoop(t, tmpDir, config.EvolutionConfig{
		Enabled: true,
		Mode:    "observe",
	}, provider)
	defer al.Close()

	defaultAgent := al.registry.GetDefaultAgent()
	if defaultAgent == nil {
		t.Fatal("expected default agent")
	}
	defaultAgent.SkillsFilter = []string{"observe-skill"}
	al.RegisterTool(&echoTextTool{})

	resp, err := al.ProcessDirectWithChannel(context.Background(), "hello", "session-observe", "cli", "direct")
	if err != nil {
		t.Fatalf("ProcessDirectWithChannel failed: %v", err)
	}
	if resp != "done" {
		t.Fatalf("response = %q, want %q", resp, "done")
	}

	record := waitForEvolutionRecord(t, filepath.Join(tmpDir, "state", "evolution", "learning-records.jsonl"))

	if got := record["kind"]; got != string(evolution.RecordKindCase) {
		t.Fatalf("kind = %v, want %q", got, evolution.RecordKindCase)
	}
	if got := record["workspace_id"]; got != tmpDir {
		t.Fatalf("workspace_id = %v, want %q", got, tmpDir)
	}
	if got := record["status"]; got != "new" {
		t.Fatalf("status = %v, want %q", got, "new")
	}

	toolKinds, ok := record["tool_kinds"].([]any)
	if !ok {
		t.Fatalf("tool_kinds missing or wrong type: %#v", record["tool_kinds"])
	}
	if len(toolKinds) != 1 || toolKinds[0] != "echo_text" {
		t.Fatalf("tool_kinds = %#v, want [echo_text]", toolKinds)
	}

	activeSkillsRaw, exists := record["initial_skill_names"]
	if !exists {
		t.Fatal("initial_skill_names field missing")
	}
	activeSkills, ok := activeSkillsRaw.([]any)
	if !ok {
		t.Fatalf("initial_skill_names wrong type: %#v", activeSkillsRaw)
	}
	if len(activeSkills) != 1 || activeSkills[0] != "observe-skill" {
		t.Fatalf("initial_skill_names = %#v, want [observe-skill]", activeSkills)
	}
	toolExecsRaw, exists := record["tool_executions"]
	if !exists {
		t.Fatal("tool_executions field missing")
	}
	toolExecs, ok := toolExecsRaw.([]any)
	if !ok || len(toolExecs) != 1 {
		t.Fatalf("tool_executions wrong type: %#v", toolExecsRaw)
	}
}

func TestEvolutionBridge_ObserveTurnEndPayloadIncludesResolvedAttemptTrail(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "skills", "observe-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(skillDir, "SKILL.md"),
		[]byte("---\nname: observe-skill\ndescription: observe test skill\n---\n# Observe Skill\n"),
		0o644,
	); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	al := newEvolutionTestLoop(t, tmpDir, config.EvolutionConfig{
		Enabled: true,
		Mode:    "observe",
	}, &simpleMockProvider{response: "ok"})
	defer al.Close()

	defaultAgent := al.registry.GetDefaultAgent()
	if defaultAgent == nil {
		t.Fatal("expected default agent")
	}
	defaultAgent.SkillsFilter = []string{"missing-skill", "observe-skill", "observe-skill"}

	sub := al.SubscribeEvents(16)
	defer al.UnsubscribeEvents(sub.ID)

	resp, err := al.ProcessDirectWithChannel(context.Background(), "hello", "session-observe-attempt-trail", "cli", "direct")
	if err != nil {
		t.Fatalf("ProcessDirectWithChannel failed: %v", err)
	}
	if resp != "ok" {
		t.Fatalf("response = %q, want %q", resp, "ok")
	}

	turnEndEvt := waitForEvent(t, sub.C, 2*time.Second, func(evt Event) bool {
		return evt.Kind == EventKindTurnEnd
	})
	turnEndPayload, ok := turnEndEvt.Payload.(TurnEndPayload)
	if !ok {
		t.Fatalf("expected TurnEndPayload, got %T", turnEndEvt.Payload)
	}
	if got := turnEndPayload.AttemptedSkills; len(got) != 1 || got[0] != "observe-skill" {
		t.Fatalf("AttemptedSkills = %v, want [observe-skill]", got)
	}
	if got := turnEndPayload.FinalSuccessfulPath; len(got) != 1 || got[0] != "observe-skill" {
		t.Fatalf("FinalSuccessfulPath = %v, want [observe-skill]", got)
	}
	if got := turnEndPayload.SkillContextSnapshots; len(got) != 1 || got[0].Trigger != skillContextTriggerInitialBuild {
		t.Fatalf("SkillContextSnapshots = %+v, want single initial_build snapshot", got)
	}
}

func TestEvolutionBridge_ObserveTurnEndUsesLatestSkillSnapshotAfterRetry(t *testing.T) {
	tmpDir := t.TempDir()
	baseSkillDir := filepath.Join(tmpDir, "skills", "base-skill")
	if err := os.MkdirAll(baseSkillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(baseSkillDir): %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(baseSkillDir, "SKILL.md"),
		[]byte("---\nname: base-skill\ndescription: base test skill\n---\n# Base Skill\n"),
		0o644,
	); err != nil {
		t.Fatalf("WriteFile(base-skill): %v", err)
	}

	lateSkillPath := filepath.Join(tmpDir, "skills", "late-skill", "SKILL.md")
	provider := &lateSkillOnRetryProvider{lateSkillPath: lateSkillPath}
	al := newEvolutionTestLoop(t, tmpDir, config.EvolutionConfig{
		Enabled: true,
		Mode:    "observe",
	}, provider)
	defer al.Close()

	defaultAgent := al.registry.GetDefaultAgent()
	if defaultAgent == nil {
		t.Fatal("expected default agent")
	}
	defaultAgent.SkillsFilter = []string{"base-skill", "late-skill"}

	sub := al.SubscribeEvents(16)
	defer al.UnsubscribeEvents(sub.ID)

	resp, err := al.ProcessDirectWithChannel(context.Background(), "hello", "session-observe-retry-snapshot", "cli", "direct")
	if err != nil {
		t.Fatalf("ProcessDirectWithChannel failed: %v", err)
	}
	if resp != "Recovered after retry" {
		t.Fatalf("response = %q, want %q", resp, "Recovered after retry")
	}

	turnEndEvt := waitForEvent(t, sub.C, 2*time.Second, func(evt Event) bool {
		return evt.Kind == EventKindTurnEnd
	})
	turnEndPayload, ok := turnEndEvt.Payload.(TurnEndPayload)
	if !ok {
		t.Fatalf("expected TurnEndPayload, got %T", turnEndEvt.Payload)
	}
	if got := turnEndPayload.AttemptedSkills; len(got) != 2 || got[0] != "base-skill" || got[1] != "late-skill" {
		t.Fatalf("AttemptedSkills = %v, want [base-skill late-skill]", got)
	}
	if got := turnEndPayload.FinalSuccessfulPath; len(got) != 2 || got[0] != "base-skill" || got[1] != "late-skill" {
		t.Fatalf("FinalSuccessfulPath = %v, want [base-skill late-skill]", got)
	}
	if got := turnEndPayload.SkillContextSnapshots; len(got) != 2 {
		t.Fatalf("len(SkillContextSnapshots) = %d, want 2", len(got))
	}
	if turnEndPayload.SkillContextSnapshots[0].Trigger != skillContextTriggerInitialBuild {
		t.Fatalf("SkillContextSnapshots[0].Trigger = %q, want %q", turnEndPayload.SkillContextSnapshots[0].Trigger, skillContextTriggerInitialBuild)
	}
	if turnEndPayload.SkillContextSnapshots[1].Trigger != skillContextTriggerContextRetryRebuild {
		t.Fatalf("SkillContextSnapshots[1].Trigger = %q, want %q", turnEndPayload.SkillContextSnapshots[1].Trigger, skillContextTriggerContextRetryRebuild)
	}
	if got := turnEndPayload.SkillContextSnapshots[1].SkillNames; len(got) != 2 || got[0] != "base-skill" || got[1] != "late-skill" {
		t.Fatalf("SkillContextSnapshots[1].SkillNames = %v, want [base-skill late-skill]", got)
	}
}

func TestEvolutionBridge_ObserveDoesNotCreateDraftFile(t *testing.T) {
	tmpDir := t.TempDir()
	al := newEvolutionTestLoop(t, tmpDir, config.EvolutionConfig{
		Enabled: true,
		Mode:    "observe",
	}, &simpleMockProvider{response: "ok"})
	defer al.Close()

	resp, err := al.ProcessDirectWithChannel(context.Background(), "hello", "session-observe-no-draft", "cli", "direct")
	if err != nil {
		t.Fatalf("ProcessDirectWithChannel failed: %v", err)
	}
	if resp != "ok" {
		t.Fatalf("response = %q, want %q", resp, "ok")
	}

	waitForEvolutionRecord(t, filepath.Join(tmpDir, "state", "evolution", "learning-records.jsonl"))
	assertNotExists(t, filepath.Join(tmpDir, "state", "evolution", "skill-drafts.json"))
}

func TestEvolutionBridge_DraftModeAutomaticallyRunsColdPathAndCreatesDraftFile(t *testing.T) {
	tmpDir := t.TempDir()
	seedReadyRule(t, tmpDir)

	al := newEvolutionTestLoop(t, tmpDir, config.EvolutionConfig{
		Enabled: true,
		Mode:    "draft",
	}, &simpleMockProvider{response: "ok"})
	defer al.Close()

	resp, err := al.ProcessDirectWithChannel(context.Background(), "hello", "session-auto-cold-path", "cli", "direct")
	if err != nil {
		t.Fatalf("ProcessDirectWithChannel failed: %v", err)
	}
	if resp != "ok" {
		t.Fatalf("response = %q, want %q", resp, "ok")
	}

	waitForEvolutionRecord(t, filepath.Join(tmpDir, "state", "evolution", "learning-records.jsonl"))
	waitForDrafts(t, filepath.Join(tmpDir, "state", "evolution", "skill-drafts.json"), 1)
}

func TestEvolutionBridge_DraftModeUsesProviderBackedDraftGenerator(t *testing.T) {
	tmpDir := t.TempDir()
	seedReadyRule(t, tmpDir)

	al := newEvolutionTestLoop(t, tmpDir, config.EvolutionConfig{
		Enabled: true,
		Mode:    "draft",
	}, &simpleMockProvider{
		response: `{"target_skill_name":"weather","draft_type":"shortcut","change_kind":"append","human_summary":"Prefer native-name path first","body_or_patch":"## Start Here\nUse native-name query first."}`,
	})
	defer al.Close()

	resp, err := al.ProcessDirectWithChannel(context.Background(), "hello", "session-auto-cold-path-llm", "cli", "direct")
	if err != nil {
		t.Fatalf("ProcessDirectWithChannel failed: %v", err)
	}
	if resp == "" {
		t.Fatal("expected non-empty response")
	}

	waitForEvolutionRecord(t, filepath.Join(tmpDir, "state", "evolution", "learning-records.jsonl"))
	drafts := waitForDrafts(t, filepath.Join(tmpDir, "state", "evolution", "skill-drafts.json"), 1)
	if drafts[0].HumanSummary != "Prefer native-name path first" {
		t.Fatalf("HumanSummary = %q, want %q", drafts[0].HumanSummary, "Prefer native-name path first")
	}
}

func TestEvolutionBridge_DraftModeUsesProviderDefaultModel(t *testing.T) {
	tmpDir := t.TempDir()
	seedReadyRule(t, tmpDir)

	provider := &capturingEvolutionDraftProvider{
		defaultModel: "provider-explicit-model",
		response:     `{"target_skill_name":"weather","draft_type":"shortcut","change_kind":"append","human_summary":"Prefer native-name path first","body_or_patch":"## Start Here\nUse native-name query first."}`,
	}

	al := newEvolutionTestLoop(t, tmpDir, config.EvolutionConfig{
		Enabled: true,
		Mode:    "draft",
	}, provider)
	defer al.Close()

	if _, err := al.ProcessDirectWithChannel(context.Background(), "hello", "session-auto-cold-path-model", "cli", "direct"); err != nil {
		t.Fatalf("ProcessDirectWithChannel failed: %v", err)
	}

	waitForEvolutionRecord(t, filepath.Join(tmpDir, "state", "evolution", "learning-records.jsonl"))
	waitForDrafts(t, filepath.Join(tmpDir, "state", "evolution", "skill-drafts.json"), 1)
	if provider.lastModel != "provider-explicit-model" {
		t.Fatalf("lastModel = %q, want provider-explicit-model", provider.lastModel)
	}
}

func TestEvolutionBridge_DraftModeKeepsCandidateDraft(t *testing.T) {
	tmpDir := t.TempDir()
	seedReadyRule(t, tmpDir)

	al := newEvolutionTestLoop(t, tmpDir, config.EvolutionConfig{
		Enabled: true,
		Mode:    "draft",
	}, &simpleMockProvider{
		response: `{"target_skill_name":"weather","draft_type":"shortcut","change_kind":"create","human_summary":"Create weather helper","body_or_patch":"---\nname: weather\ndescription: weather helper\n---\n# Weather\n## Start Here\nUse native-name query first.\n"}`,
	})
	defer al.Close()

	if _, err := al.ProcessDirectWithChannel(context.Background(), "hello", "session-apply-no-auto-apply", "cli", "direct"); err != nil {
		t.Fatalf("ProcessDirectWithChannel failed: %v", err)
	}

	waitForEvolutionRecord(t, filepath.Join(tmpDir, "state", "evolution", "learning-records.jsonl"))
	drafts := waitForDrafts(t, filepath.Join(tmpDir, "state", "evolution", "skill-drafts.json"), 1)
	if drafts[0].Status != evolution.DraftStatusCandidate {
		t.Fatalf("draft status = %q, want %q", drafts[0].Status, evolution.DraftStatusCandidate)
	}

	assertNotExists(t, filepath.Join(tmpDir, "skills", "weather", "SKILL.md"))
	assertProfileNotExists(t, tmpDir, "weather")
}

func TestEvolutionBridge_ApplyModeAutomaticallyRunsColdPathAndAppliesMergeDraft(t *testing.T) {
	tmpDir := t.TempDir()
	seedReadyRule(t, tmpDir)

	skillDir := filepath.Join(tmpDir, "skills", "weather")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	skillPath := filepath.Join(skillDir, "SKILL.md")
	original := "---\nname: weather\ndescription: weather helper\n---\n# Weather\n## Start Here\nUse city names.\n"
	if err := os.WriteFile(skillPath, []byte(original), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	al := newEvolutionTestLoop(t, tmpDir, config.EvolutionConfig{
		Enabled: true,
		Mode:    "apply",
	}, &simpleMockProvider{
		response: `{"target_skill_name":"weather","draft_type":"shortcut","change_kind":"merge","human_summary":"Merge native-name path","body_or_patch":"Prefer native-name query first."}`,
	})
	defer al.Close()

	if _, err := al.ProcessDirectWithChannel(context.Background(), "hello", "session-apply-merge", "cli", "direct"); err != nil {
		t.Fatalf("ProcessDirectWithChannel failed: %v", err)
	}

	waitForEvolutionRecord(t, filepath.Join(tmpDir, "state", "evolution", "learning-records.jsonl"))
	drafts := waitForDrafts(t, filepath.Join(tmpDir, "state", "evolution", "skill-drafts.json"), 1)
	if drafts[0].Status != evolution.DraftStatusAccepted {
		t.Fatalf("draft status = %q, want %q", drafts[0].Status, evolution.DraftStatusAccepted)
	}

	merged := waitForSkillBody(t, skillPath)
	if !strings.Contains(merged, "Use city names.") {
		t.Fatalf("merged skill lost original content:\n%s", merged)
	}
	if !strings.Contains(merged, "## Merged Knowledge") {
		t.Fatalf("merged skill missing merged section:\n%s", merged)
	}
	if !strings.Contains(merged, "Prefer native-name query first.") {
		t.Fatalf("merged skill missing learned knowledge:\n%s", merged)
	}

	profile := waitForProfile(t, tmpDir, "weather")
	if profile.Status != evolution.SkillStatusActive {
		t.Fatalf("profile status = %q, want %q", profile.Status, evolution.SkillStatusActive)
	}
	if profile.CurrentVersion == "" {
		t.Fatal("expected applied profile current version")
	}
}

func TestEvolutionBridge_ObserveModeDoesNotRunColdPathOrCreateDraftFile(t *testing.T) {
	tmpDir := t.TempDir()
	seedReadyRule(t, tmpDir)

	al := newEvolutionTestLoop(t, tmpDir, config.EvolutionConfig{
		Enabled: true,
		Mode:    "observe",
	}, &simpleMockProvider{response: "ok"})
	defer al.Close()

	resp, err := al.ProcessDirectWithChannel(context.Background(), "hello", "session-no-auto-cold-path", "cli", "direct")
	if err != nil {
		t.Fatalf("ProcessDirectWithChannel failed: %v", err)
	}
	if resp != "ok" {
		t.Fatalf("response = %q, want %q", resp, "ok")
	}

	waitForEvolutionRecord(t, filepath.Join(tmpDir, "state", "evolution", "learning-records.jsonl"))
	assertNotExists(t, filepath.Join(tmpDir, "state", "evolution", "skill-drafts.json"))
}

func TestEvolutionBridge_TurnEndUsesPayloadWorkspace(t *testing.T) {
	workspace := t.TempDir()
	cfg := &config.Config{
		Evolution: config.EvolutionConfig{
			Enabled: true,
			Mode:    "observe",
		},
	}

	bridge, err := newEvolutionBridge(nil, cfg, nil)
	if err != nil {
		t.Fatalf("newEvolutionBridge: %v", err)
	}

	err = bridge.OnEvent(context.Background(), Event{
		Kind: EventKindTurnEnd,
		Meta: EventMeta{
			AgentID:    "main",
			TurnID:     "turn-1",
			SessionKey: "session-1",
		},
		Payload: TurnEndPayload{
			Status:       TurnEndStatusCompleted,
			Workspace:    workspace,
			ActiveSkills: []string{"observe-skill"},
			ToolKinds:    []string{"echo_text"},
		},
	})
	if err != nil {
		t.Fatalf("OnEvent: %v", err)
	}

	record := waitForEvolutionRecord(t, filepath.Join(workspace, "state", "evolution", "learning-records.jsonl"))
	if got := record["workspace_id"]; got != workspace {
		t.Fatalf("workspace_id = %v, want %q", got, workspace)
	}
}

func TestEvolutionBridge_TurnEndUsesExplicitAttemptTrail(t *testing.T) {
	workspace := t.TempDir()
	cfg := &config.Config{
		Evolution: config.EvolutionConfig{
			Enabled: true,
			Mode:    "observe",
		},
	}

	bridge, err := newEvolutionBridge(nil, cfg, nil)
	if err != nil {
		t.Fatalf("newEvolutionBridge: %v", err)
	}

	err = bridge.OnEvent(context.Background(), Event{
		Kind: EventKindTurnEnd,
		Meta: EventMeta{
			AgentID:    "main",
			TurnID:     "turn-1",
			SessionKey: "session-1",
		},
		Payload: TurnEndPayload{
			Status:              TurnEndStatusCompleted,
			Workspace:           workspace,
			ActiveSkills:        []string{"weather"},
			AttemptedSkills:     []string{"geocode", "weather"},
			FinalSuccessfulPath: []string{"geocode", "weather"},
			SkillContextSnapshots: []SkillContextSnapshot{
				{Sequence: 1, Trigger: skillContextTriggerInitialBuild, SkillNames: []string{"weather"}},
				{Sequence: 2, Trigger: skillContextTriggerContextRetryRebuild, SkillNames: []string{"geocode", "weather"}},
			},
			ToolKinds: []string{"echo_text"},
		},
	})
	if err != nil {
		t.Fatalf("OnEvent: %v", err)
	}

	record := waitForEvolutionRecord(t, filepath.Join(workspace, "state", "evolution", "learning-records.jsonl"))
	attemptTrailRaw, ok := record["attempt_trail"].(map[string]any)
	if !ok {
		t.Fatalf("attempt_trail missing or wrong type: %#v", record["attempt_trail"])
	}
	attemptedSkills, ok := attemptTrailRaw["attempted_skills"].([]any)
	if !ok {
		t.Fatalf("attempted_skills wrong type: %#v", attemptTrailRaw["attempted_skills"])
	}
	if len(attemptedSkills) != 2 || attemptedSkills[0] != "geocode" || attemptedSkills[1] != "weather" {
		t.Fatalf("attempted_skills = %#v, want [geocode weather]", attemptedSkills)
	}
	finalPath, ok := attemptTrailRaw["final_successful_path"].([]any)
	if !ok {
		t.Fatalf("final_successful_path wrong type: %#v", attemptTrailRaw["final_successful_path"])
	}
	if len(finalPath) != 2 || finalPath[0] != "geocode" || finalPath[1] != "weather" {
		t.Fatalf("final_successful_path = %#v, want [geocode weather]", finalPath)
	}
	if _, exists := attemptTrailRaw["skill_context_snapshots"]; exists {
		t.Fatalf("skill_context_snapshots should not be persisted: %#v", attemptTrailRaw["skill_context_snapshots"])
	}
	initialSkills, ok := record["initial_skill_names"].([]any)
	if !ok || len(initialSkills) != 1 || initialSkills[0] != "weather" {
		t.Fatalf("initial_skill_names = %#v, want [weather]", record["initial_skill_names"])
	}
	addedSkills, ok := record["added_skill_names"].([]any)
	if !ok || len(addedSkills) != 1 || addedSkills[0] != "geocode" {
		t.Fatalf("added_skill_names = %#v, want [geocode]", record["added_skill_names"])
	}
}

func TestEvolutionBridge_CloseStopsColdPathRunnerIdempotently(t *testing.T) {
	cfg := &config.Config{
		Evolution: config.EvolutionConfig{
			Enabled: true,
			Mode:    "draft",
		},
	}

	bridge, err := newEvolutionBridge(nil, cfg, nil)
	if err != nil {
		t.Fatalf("newEvolutionBridge: %v", err)
	}
	if bridge.coldPathRunner == nil {
		t.Fatal("expected cold path runner")
	}

	if err := bridge.Close(); err != nil {
		t.Fatalf("first Close() error = %v", err)
	}
	if err := bridge.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
	if bridge.coldPathRunner.Trigger(t.TempDir()) {
		t.Fatal("expected closed bridge runner to reject new work")
	}
}

func TestAgentLoop_ReloadProviderAndConfig_RebuildsEvolutionBridge(t *testing.T) {
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         t.TempDir(),
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 3,
			},
		},
		Evolution: config.EvolutionConfig{
			Enabled: false,
			Mode:    "observe",
		},
	}

	al := NewAgentLoop(cfg, bus.NewMessageBus(), &mockProvider{})
	defer al.Close()

	oldBridge := al.evolution
	if oldBridge == nil {
		t.Fatal("expected initial evolution bridge")
	}

	reloadCfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         t.TempDir(),
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 3,
			},
		},
		Evolution: config.EvolutionConfig{
			Enabled:  true,
			Mode:     "apply",
			StateDir: filepath.Join(t.TempDir(), "evolution-state"),
		},
	}

	if err := al.ReloadProviderAndConfig(context.Background(), &mockProvider{}, reloadCfg); err != nil {
		t.Fatalf("ReloadProviderAndConfig failed: %v", err)
	}

	if al.evolution == nil {
		t.Fatal("expected evolution bridge after reload")
	}
	if al.evolution == oldBridge {
		t.Fatal("expected evolution bridge to be rebuilt on reload")
	}
	if al.evolution.cfg.Enabled != reloadCfg.Evolution.Enabled {
		t.Fatalf("reloaded evolution enabled = %v, want %v", al.evolution.cfg.Enabled, reloadCfg.Evolution.Enabled)
	}
	if al.evolution.cfg.Mode != reloadCfg.Evolution.Mode {
		t.Fatalf("reloaded evolution mode = %q, want %q", al.evolution.cfg.Mode, reloadCfg.Evolution.Mode)
	}
	if al.evolution.cfg.StateDir != reloadCfg.Evolution.StateDir {
		t.Fatalf("reloaded evolution state_dir = %q, want %q", al.evolution.cfg.StateDir, reloadCfg.Evolution.StateDir)
	}
}

func seedReadyRule(t *testing.T, workspace string) {
	t.Helper()

	store := evolution.NewStore(evolution.NewPaths(workspace, ""))
	rule := evolution.LearningRecord{
		ID:          "rule-1",
		Kind:        evolution.RecordKindRule,
		WorkspaceID: workspace,
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
}

func newEvolutionTestLoop(t *testing.T, workspace string, evo config.EvolutionConfig, provider providers.LLMProvider) *AgentLoop {
	t.Helper()

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         workspace,
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 3,
			},
		},
		Evolution: evo,
	}

	return NewAgentLoop(cfg, bus.NewMessageBus(), provider)
}

func waitForEvolutionRecord(t *testing.T, path string) map[string]any {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(path)
		if err == nil {
			lines := strings.Split(strings.TrimSpace(string(data)), "\n")
			for i := len(lines) - 1; i >= 0; i-- {
				if strings.TrimSpace(lines[i]) == "" {
					continue
				}
				var record map[string]any
				if err := json.Unmarshal([]byte(lines[i]), &record); err != nil {
					t.Fatalf("json.Unmarshal(%s): %v", path, err)
				}
				if kind, _ := record["kind"].(string); kind == string(evolution.RecordKindTask) {
					return record
				}
			}
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for evolution record at %s", path)
	return nil
}

func waitForDrafts(t *testing.T, path string, want int) []evolution.SkillDraft {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(path)
		if err == nil {
			var drafts []evolution.SkillDraft
			if err := json.Unmarshal(data, &drafts); err != nil {
				t.Fatalf("json.Unmarshal(%s): %v", path, err)
			}
			if len(drafts) == want {
				return drafts
			}
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for %d drafts at %s", want, path)
	return nil
}

func waitForSkillBody(t *testing.T, path string) string {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(path)
		if err == nil {
			return string(data)
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for skill file at %s", path)
	return ""
}

func waitForProfile(t *testing.T, workspace, skillName string) evolution.SkillProfile {
	t.Helper()

	store := evolution.NewStore(evolution.NewPaths(workspace, ""))
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		profile, err := store.LoadProfile(skillName)
		if err == nil {
			return profile
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for profile %q in %s", skillName, workspace)
	return evolution.SkillProfile{}
}

func assertProfileNotExists(t *testing.T, workspace, skillName string) {
	t.Helper()

	store := evolution.NewStore(evolution.NewPaths(workspace, ""))
	if _, err := store.LoadProfile(skillName); !os.IsNotExist(err) {
		t.Fatalf("profile %q should not exist, got err = %v", skillName, err)
	}
}

func assertNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("%s should not exist, stat err = %v", path, err)
	}
}

type capturingEvolutionDraftProvider struct {
	response     string
	defaultModel string
	lastModel    string
}

type lateSkillOnRetryProvider struct {
	calls         int
	lateSkillPath string
}

func (p *lateSkillOnRetryProvider) Chat(
	_ context.Context,
	_ []providers.Message,
	_ []providers.ToolDefinition,
	_ string,
	_ map[string]any,
) (*providers.LLMResponse, error) {
	p.calls++
	if p.calls == 1 {
		if err := os.MkdirAll(filepath.Dir(p.lateSkillPath), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(
			p.lateSkillPath,
			[]byte("---\nname: late-skill\ndescription: late test skill\n---\n# Late Skill\n"),
			0o644,
		); err != nil {
			return nil, err
		}
		return nil, errors.New("context_window_exceeded")
	}

	return &providers.LLMResponse{Content: "Recovered after retry"}, nil
}

func (p *lateSkillOnRetryProvider) GetDefaultModel() string {
	return "mock-model"
}

func (p *capturingEvolutionDraftProvider) Chat(
	_ context.Context,
	_ []providers.Message,
	_ []providers.ToolDefinition,
	model string,
	_ map[string]any,
) (*providers.LLMResponse, error) {
	p.lastModel = model
	return &providers.LLMResponse{Content: p.response}, nil
}

func (p *capturingEvolutionDraftProvider) GetDefaultModel() string {
	return p.defaultModel
}
