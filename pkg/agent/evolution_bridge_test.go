package agent

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	runtimeevents "github.com/sipeed/picoclaw/pkg/events"
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

	assertNotExists(t, filepath.Join(tmpDir, "state", "evolution", "task-records.jsonl"))
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

	record := waitForEvolutionRecord(t, filepath.Join(tmpDir, "state", "evolution", "task-records.jsonl"))

	if got := record["kind"]; got != string(evolution.RecordKindCase) {
		t.Fatalf("kind = %v, want %q", got, evolution.RecordKindCase)
	}
	if got := record["workspace_id"]; got != tmpDir {
		t.Fatalf("workspace_id = %v, want %q", got, tmpDir)
	}
	if got := record["status"]; got != "new" {
		t.Fatalf("status = %v, want %q", got, "new")
	}

	for _, field := range []string{"tool_kinds", "tool_executions", "initial_skill_names", "active_skill_names", "attempt_trail", "source"} {
		if _, exists := record[field]; exists {
			t.Fatalf("%s should not be persisted in slim task record: %#v", field, record[field])
		}
	}
}

func TestEvolutionBridge_TurnEndBypassesHookObserverBackpressure(t *testing.T) {
	tmpDir := t.TempDir()
	al := newEvolutionTestLoop(t, tmpDir, config.EvolutionConfig{
		Enabled: true,
		Mode:    "observe",
	}, &simpleMockProvider{response: "ok"})
	defer al.Close()

	blocker := &blockingRuntimeObserver{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	defer close(blocker.release)
	al.hooks.ConfigureTimeouts(5*time.Second, 0, 0)
	if err := al.MountHook(NamedHook("aaa-block-runtime-events", blocker)); err != nil {
		t.Fatalf("MountHook: %v", err)
	}

	al.publishRuntimeEvent(runtimeevents.Event{
		Kind:   runtimeevents.KindAgentTurnStart,
		Source: runtimeevents.Source{Component: "agent", Name: "main"},
	})
	select {
	case <-blocker.started:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for blocking runtime observer")
	}

	for i := 0; i < hookObserverBufferSize+10; i++ {
		al.publishRuntimeEvent(runtimeevents.Event{
			Kind:   runtimeevents.KindAgentLLMDelta,
			Source: runtimeevents.Source{Component: "agent", Name: "main"},
		})
	}

	al.emitEvent(runtimeevents.KindAgentTurnEnd, EventMeta{
		AgentID:    "main",
		TurnID:     "turn-backpressure",
		SessionKey: "session-backpressure",
	}, TurnEndPayload{
		Status:       TurnEndStatusCompleted,
		Workspace:    tmpDir,
		UserMessage:  "hello",
		FinalContent: "ok",
	})

	record := waitForEvolutionRecord(t, filepath.Join(tmpDir, "state", "evolution", "task-records.jsonl"))
	if got := record["session_key"]; got != "session-backpressure" {
		t.Fatalf("session_key = %v, want session-backpressure", got)
	}
	if got := record["summary"]; got != "hello" {
		t.Fatalf("summary = %v, want hello", got)
	}
}

func TestEvolutionBridge_RuntimeBusTurnEndWritesCaseRecord(t *testing.T) {
	tmpDir := t.TempDir()
	al := newEvolutionTestLoop(t, tmpDir, config.EvolutionConfig{
		Enabled: true,
		Mode:    "observe",
	}, &simpleMockProvider{response: "ok"})
	defer al.Close()

	result := al.RuntimeEventBus().Publish(context.Background(), runtimeevents.Event{
		Kind:   runtimeevents.KindAgentTurnEnd,
		Source: runtimeevents.Source{Component: "agent", Name: "main"},
		Scope: runtimeevents.Scope{
			AgentID:    "main",
			TurnID:     "turn-runtime-bus",
			SessionKey: "session-runtime-bus",
		},
		Payload: TurnEndPayload{
			Status:       TurnEndStatusCompleted,
			Workspace:    tmpDir,
			UserMessage:  "runtime bus task",
			FinalContent: "ok",
		},
	})
	if result.Delivered == 0 {
		t.Fatalf("runtime bus publish delivered = %d, want > 0", result.Delivered)
	}

	record := waitForEvolutionRecord(t, filepath.Join(tmpDir, "state", "evolution", "task-records.jsonl"))
	if got := record["session_key"]; got != "session-runtime-bus" {
		t.Fatalf("session_key = %v, want session-runtime-bus", got)
	}
	if got := record["summary"]; got != "runtime bus task" {
		t.Fatalf("summary = %v, want runtime bus task", got)
	}
}

func TestEvolutionBridge_RuntimeBusOnlyCurrentBridgeConsumesTurnEnd(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Evolution: config.EvolutionConfig{
			Enabled: true,
			Mode:    "observe",
		},
	}
	eventBus := runtimeevents.NewBus()
	defer eventBus.Close()

	oldBridge, err := newEvolutionBridge(nil, cfg, nil)
	if err != nil {
		t.Fatalf("newEvolutionBridge(old): %v", err)
	}
	defer oldBridge.Close()
	newBridge, err := newEvolutionBridge(nil, cfg, nil)
	if err != nil {
		t.Fatalf("newEvolutionBridge(new): %v", err)
	}
	defer newBridge.Close()

	current := newBridge
	oldBridge.setCurrentCheck(func(bridge *evolutionBridge) bool {
		return current == bridge
	})
	newBridge.setCurrentCheck(func(bridge *evolutionBridge) bool {
		return current == bridge
	})
	if err := oldBridge.subscribeRuntimeEvents(eventBus.Channel()); err != nil {
		t.Fatalf("old subscribeRuntimeEvents: %v", err)
	}
	if err := newBridge.subscribeRuntimeEvents(eventBus.Channel()); err != nil {
		t.Fatalf("new subscribeRuntimeEvents: %v", err)
	}

	eventBus.Publish(context.Background(), runtimeevents.Event{
		Kind:   runtimeevents.KindAgentTurnEnd,
		Source: runtimeevents.Source{Component: "agent", Name: "main"},
		Scope: runtimeevents.Scope{
			AgentID:    "main",
			TurnID:     "turn-current-bridge",
			SessionKey: "session-current-bridge",
		},
		Payload: TurnEndPayload{
			Status:       TurnEndStatusCompleted,
			Workspace:    tmpDir,
			UserMessage:  "current bridge task",
			FinalContent: "ok",
		},
	})

	recordsPath := filepath.Join(tmpDir, "state", "evolution", "task-records.jsonl")
	waitForEvolutionRecord(t, recordsPath)
	time.Sleep(100 * time.Millisecond)
	if got := countEvolutionTaskRecords(t, recordsPath); got != 1 {
		t.Fatalf("task record count = %d, want 1", got)
	}
}

func TestEvolutionBridge_DirectDeliveryFailureFallsBackToCurrentRuntimeBridge(t *testing.T) {
	tmpDir := t.TempDir()
	al := newEvolutionTestLoop(t, tmpDir, config.EvolutionConfig{
		Enabled: true,
		Mode:    "observe",
	}, &simpleMockProvider{response: "ok"})
	defer al.Close()

	oldBridge := al.evolution
	if oldBridge == nil {
		t.Fatal("expected initial evolution bridge")
	}
	defer oldBridge.Close()

	newBridge, err := newEvolutionBridge(al.registry, al.cfg, &simpleMockProvider{response: "ok"})
	if err != nil {
		t.Fatalf("newEvolutionBridge: %v", err)
	}
	newBridge.setCurrentCheck(al.isCurrentEvolutionBridge)
	if err := newBridge.subscribeRuntimeEvents(al.RuntimeEventBus().Channel()); err != nil {
		t.Fatalf("subscribeRuntimeEvents: %v", err)
	}

	oldBridge.closeMu.Lock()
	done := make(chan struct{})
	go func() {
		defer close(done)
		al.emitEvent(runtimeevents.KindAgentTurnEnd, EventMeta{
			AgentID:    "main",
			TurnID:     "turn-direct-fallback",
			SessionKey: "session-direct-fallback",
		}, TurnEndPayload{
			Status:       TurnEndStatusCompleted,
			Workspace:    tmpDir,
			UserMessage:  "direct fallback task",
			FinalContent: "ok",
		})
	}()

	time.Sleep(20 * time.Millisecond)
	al.mu.Lock()
	al.evolution = newBridge
	al.mu.Unlock()
	oldBridge.closed = true
	oldBridge.closeMu.Unlock()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for emitEvent")
	}

	recordsPath := filepath.Join(tmpDir, "state", "evolution", "task-records.jsonl")
	record := waitForEvolutionRecord(t, recordsPath)
	if got := record["session_key"]; got != "session-direct-fallback" {
		t.Fatalf("session_key = %v, want session-direct-fallback", got)
	}
	if got := countEvolutionTaskRecords(t, recordsPath); got != 1 {
		t.Fatalf("task record count = %d, want 1", got)
	}
}

func TestEvolutionBridge_CloseCancelsPendingTurnEndRecord(t *testing.T) {
	tmpDir := t.TempDir()
	al := newEvolutionTestLoop(t, tmpDir, config.EvolutionConfig{
		Enabled: true,
		Mode:    "observe",
	}, &simpleMockProvider{response: "ok"})

	al.emitEvent(runtimeevents.KindAgentTurnEnd, EventMeta{
		AgentID:    "main",
		TurnID:     "turn-close-flush",
		SessionKey: "session-close-flush",
	}, TurnEndPayload{
		Status:       TurnEndStatusCompleted,
		Workspace:    tmpDir,
		UserMessage:  "close flush task",
		FinalContent: "ok",
	})

	done := make(chan struct{})
	go func() {
		al.Close()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Close timed out")
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

	resp, err := al.ProcessDirectWithChannel(
		context.Background(),
		"hello",
		"session-observe-attempt-trail",
		"cli",
		"direct",
	)
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

	resp, err := al.ProcessDirectWithChannel(
		context.Background(),
		"hello",
		"session-observe-retry-snapshot",
		"cli",
		"direct",
	)
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
		t.Fatalf(
			"SkillContextSnapshots[0].Trigger = %q, want %q",
			turnEndPayload.SkillContextSnapshots[0].Trigger,
			skillContextTriggerInitialBuild,
		)
	}
	if turnEndPayload.SkillContextSnapshots[1].Trigger != skillContextTriggerContextRetryRebuild {
		t.Fatalf(
			"SkillContextSnapshots[1].Trigger = %q, want %q",
			turnEndPayload.SkillContextSnapshots[1].Trigger,
			skillContextTriggerContextRetryRebuild,
		)
	}
	if got := turnEndPayload.SkillContextSnapshots[1].SkillNames; len(got) != 2 || got[0] != "base-skill" ||
		got[1] != "late-skill" {
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

	waitForEvolutionRecord(t, filepath.Join(tmpDir, "state", "evolution", "task-records.jsonl"))
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

	waitForEvolutionRecord(t, filepath.Join(tmpDir, "state", "evolution", "task-records.jsonl"))
	waitForDrafts(t, filepath.Join(tmpDir, "state", "evolution", "skill-drafts.json"), 1)
}

func TestEvolutionBridge_ScheduledModeDoesNotRunColdPathAfterTurn(t *testing.T) {
	tmpDir := t.TempDir()
	seedReadyRule(t, tmpDir)

	al := newEvolutionTestLoop(t, tmpDir, config.EvolutionConfig{
		Enabled:         true,
		Mode:            "draft",
		ColdPathTrigger: "scheduled",
	}, &simpleMockProvider{response: "ok"})
	defer al.Close()

	resp, err := al.ProcessDirectWithChannel(
		context.Background(),
		"hello",
		"session-scheduled-cold-path",
		"cli",
		"direct",
	)
	if err != nil {
		t.Fatalf("ProcessDirectWithChannel failed: %v", err)
	}
	if resp != "ok" {
		t.Fatalf("response = %q, want %q", resp, "ok")
	}

	waitForEvolutionRecord(t, filepath.Join(tmpDir, "state", "evolution", "task-records.jsonl"))
	time.Sleep(150 * time.Millisecond)
	assertNotExists(t, filepath.Join(tmpDir, "state", "evolution", "skill-drafts.json"))
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

	resp, err := al.ProcessDirectWithChannel(
		context.Background(),
		"hello",
		"session-auto-cold-path-llm",
		"cli",
		"direct",
	)
	if err != nil {
		t.Fatalf("ProcessDirectWithChannel failed: %v", err)
	}
	if resp == "" {
		t.Fatal("expected non-empty response")
	}

	waitForEvolutionRecord(t, filepath.Join(tmpDir, "state", "evolution", "task-records.jsonl"))
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

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				ModelName:         "",
				MaxTokens:         4096,
				MaxToolIterations: 3,
			},
		},
		Evolution: config.EvolutionConfig{
			Enabled: true,
			Mode:    "draft",
		},
	}

	al := NewAgentLoop(cfg, bus.NewMessageBus(), provider)
	defer al.Close()

	if _, err := al.ProcessDirectWithChannel(
		context.Background(),
		"hello",
		"session-auto-cold-path-model",
		"cli",
		"direct",
	); err != nil {
		t.Fatalf("ProcessDirectWithChannel failed: %v", err)
	}

	waitForEvolutionRecord(t, filepath.Join(tmpDir, "state", "evolution", "task-records.jsonl"))
	waitForDrafts(t, filepath.Join(tmpDir, "state", "evolution", "skill-drafts.json"), 1)
	if provider.lastModel != "provider-explicit-model" {
		t.Fatalf("lastModel = %q, want provider-explicit-model", provider.lastModel)
	}
}

func TestEvolutionBridge_DraftModePrefersConfigDefaultModelName(t *testing.T) {
	tmpDir := t.TempDir()
	seedReadyRule(t, tmpDir)

	provider := &capturingEvolutionDraftProvider{
		defaultModel: "provider-default-model",
		response:     `{"target_skill_name":"weather","draft_type":"shortcut","change_kind":"append","human_summary":"Prefer native-name path first","body_or_patch":"## Start Here\nUse native-name query first."}`,
	}

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 3,
			},
		},
		Evolution: config.EvolutionConfig{
			Enabled: true,
			Mode:    "draft",
		},
	}
	cfg.Agents.Defaults.ModelName = "resolved-config-model"

	al := NewAgentLoop(cfg, bus.NewMessageBus(), provider)
	defer al.Close()

	if _, err := al.ProcessDirectWithChannel(
		context.Background(),
		"hello",
		"session-auto-cold-path-model-config",
		"cli",
		"direct",
	); err != nil {
		t.Fatalf("ProcessDirectWithChannel failed: %v", err)
	}

	waitForEvolutionRecord(t, filepath.Join(tmpDir, "state", "evolution", "task-records.jsonl"))
	waitForDrafts(t, filepath.Join(tmpDir, "state", "evolution", "skill-drafts.json"), 1)
	if provider.lastModel != "resolved-config-model" {
		t.Fatalf("lastModel = %q, want resolved-config-model", provider.lastModel)
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

	if _, err := al.ProcessDirectWithChannel(
		context.Background(),
		"hello",
		"session-apply-no-auto-apply",
		"cli",
		"direct",
	); err != nil {
		t.Fatalf("ProcessDirectWithChannel failed: %v", err)
	}

	waitForEvolutionRecord(t, filepath.Join(tmpDir, "state", "evolution", "task-records.jsonl"))
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

	if _, err := al.ProcessDirectWithChannel(
		context.Background(),
		"hello",
		"session-apply-merge",
		"cli",
		"direct",
	); err != nil {
		t.Fatalf("ProcessDirectWithChannel failed: %v", err)
	}

	waitForEvolutionRecord(t, filepath.Join(tmpDir, "state", "evolution", "task-records.jsonl"))
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

	resp, err := al.ProcessDirectWithChannel(
		context.Background(),
		"hello",
		"session-no-auto-cold-path",
		"cli",
		"direct",
	)
	if err != nil {
		t.Fatalf("ProcessDirectWithChannel failed: %v", err)
	}
	if resp != "ok" {
		t.Fatalf("response = %q, want %q", resp, "ok")
	}

	waitForEvolutionRecord(t, filepath.Join(tmpDir, "state", "evolution", "task-records.jsonl"))
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

	record := waitForEvolutionRecord(t, filepath.Join(workspace, "state", "evolution", "task-records.jsonl"))
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
				{
					Sequence:   2,
					Trigger:    skillContextTriggerContextRetryRebuild,
					SkillNames: []string{"geocode", "weather"},
				},
			},
			ToolKinds: []string{"echo_text"},
		},
	})
	if err != nil {
		t.Fatalf("OnEvent: %v", err)
	}

	record := waitForEvolutionRecord(t, filepath.Join(workspace, "state", "evolution", "task-records.jsonl"))
	usedSkills, ok := record["used_skill_names"].([]any)
	if !ok || len(usedSkills) != 2 || usedSkills[0] != "geocode" || usedSkills[1] != "weather" {
		t.Fatalf("used_skill_names = %#v, want [geocode weather]", record["used_skill_names"])
	}
	for _, field := range []string{"attempt_trail", "initial_skill_names", "added_skill_names"} {
		if _, exists := record[field]; exists {
			t.Fatalf("%s should not be persisted in slim task record: %#v", field, record[field])
		}
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

func TestEvolutionBridge_CloseRejectsLateTurnEndEvents(t *testing.T) {
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

	if closeErr := bridge.Close(); closeErr != nil {
		t.Fatalf("Close() error = %v", closeErr)
	}

	err = bridge.OnEvent(context.Background(), Event{
		Kind: EventKindTurnEnd,
		Meta: EventMeta{
			TurnID:     "turn-after-close",
			SessionKey: "session-after-close",
			AgentID:    "agent-after-close",
		},
		Payload: TurnEndPayload{
			Status:    TurnEndStatusCompleted,
			Workspace: workspace,
		},
	})
	if err != nil {
		t.Fatalf("OnEvent() error = %v", err)
	}

	assertNotExists(t, filepath.Join(workspace, "state", "evolution", "task-records.jsonl"))
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

func TestEvolutionBridge_ColdPathScheduleParsing(t *testing.T) {
	schedule := parseColdPathSchedule([]string{"18:30", "bad", "03:05", "18:30", "24:00", "09:99"})
	if len(schedule) != 2 {
		t.Fatalf("len(schedule) = %d, want 2: %+v", len(schedule), schedule)
	}
	if schedule[0].hour != 3 || schedule[0].minute != 5 {
		t.Fatalf("schedule[0] = %+v, want 03:05", schedule[0])
	}
	if schedule[1].hour != 18 || schedule[1].minute != 30 {
		t.Fatalf("schedule[1] = %+v, want 18:30", schedule[1])
	}

	now := time.Date(2026, 5, 7, 4, 0, 0, 0, time.Local)
	next := nextColdPathScheduledTime(now, schedule)
	want := time.Date(2026, 5, 7, 18, 30, 0, 0, time.Local)
	if !next.Equal(want) {
		t.Fatalf("next = %v, want %v", next, want)
	}

	now = time.Date(2026, 5, 7, 19, 0, 0, 0, time.Local)
	next = nextColdPathScheduledTime(now, schedule)
	want = time.Date(2026, 5, 8, 3, 5, 0, 0, time.Local)
	if !next.Equal(want) {
		t.Fatalf("next after day end = %v, want %v", next, want)
	}
}

func TestEvolutionBridge_ScheduledColdPathTracksObservedWorkspaces(t *testing.T) {
	bridge := &evolutionBridge{
		cfg: config.EvolutionConfig{
			Enabled:         true,
			Mode:            "draft",
			ColdPathTrigger: "scheduled",
			ColdPathTimes:   []string{"03:00"},
		},
	}

	bridge.rememberScheduledColdPathWorkspace("/tmp/workspace-b")
	bridge.rememberScheduledColdPathWorkspace("/tmp/workspace-a")
	bridge.rememberScheduledColdPathWorkspace("/tmp/workspace-b")
	bridge.rememberScheduledColdPathWorkspace("")

	got := bridge.scheduledColdPathWorkspaces()
	want := []string{"/tmp/workspace-a", "/tmp/workspace-b"}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("scheduled workspaces = %v, want %v", got, want)
	}
}

func TestEvolutionBridge_ScheduledColdPathSeedsConfiguredAgentWorkspaces(t *testing.T) {
	defaultWorkspace := t.TempDir()
	workerWorkspace := t.TempDir()
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         defaultWorkspace,
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 3,
			},
			List: []config.AgentConfig{
				{ID: "main", Default: true},
				{ID: "worker", Workspace: workerWorkspace},
			},
		},
		Evolution: config.EvolutionConfig{
			Enabled:         true,
			Mode:            "draft",
			ColdPathTrigger: "scheduled",
			ColdPathTimes:   []string{"03:00"},
		},
	}
	registry := NewAgentRegistry(cfg, &simpleMockProvider{response: "ok"})
	bridge, err := newEvolutionBridge(registry, cfg, &simpleMockProvider{response: "ok"})
	if err != nil {
		t.Fatalf("newEvolutionBridge: %v", err)
	}
	defer bridge.Close()

	got := bridge.scheduledColdPathWorkspaces()
	want := []string{defaultWorkspace, workerWorkspace}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("scheduled workspaces = %v, want %v", got, want)
	}
}

func seedReadyRule(t *testing.T, workspace string) {
	t.Helper()

	store := evolution.NewStore(evolution.NewPaths(workspace, ""))
	rule := evolution.LearningRecord{
		ID:            "rule-1",
		Kind:          evolution.RecordKindRule,
		WorkspaceID:   workspace,
		CreatedAt:     time.Unix(1700000000, 0).UTC(),
		Label:         "weather-native-name-path",
		Summary:       "weather native-name path",
		Status:        evolution.RecordStatus("ready"),
		TaskRecordIDs: []string{"task-1", "task-2"},
	}
	if err := store.AppendLearningRecords([]evolution.LearningRecord{rule}); err != nil {
		t.Fatalf("AppendLearningRecords: %v", err)
	}
}

func newEvolutionTestLoop(
	t *testing.T,
	workspace string,
	evo config.EvolutionConfig,
	provider providers.LLMProvider,
) *AgentLoop {
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

func countEvolutionTaskRecords(t *testing.T, path string) int {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	count := 0
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("json.Unmarshal(%s): %v", path, err)
		}
		if kind, _ := record["kind"].(string); kind == string(evolution.RecordKindTask) {
			count++
		}
	}
	return count
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
	if _, loadErr := store.LoadProfile(skillName); !os.IsNotExist(loadErr) {
		t.Fatalf("profile %q should not exist, got err = %v", skillName, loadErr)
	}
}

func assertNotExists(t *testing.T, path string) {
	t.Helper()
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatalf("%s should not exist, stat err = %v", path, statErr)
	}
}

func waitForEvent(t *testing.T, ch <-chan Event, timeout time.Duration, match func(Event) bool) Event {
	t.Helper()

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case evt, ok := <-ch:
			if !ok {
				t.Fatal("event channel closed")
			}
			if match == nil || match(evt) {
				return evt
			}
		case <-timer.C:
			t.Fatal("timed out waiting for event")
		}
	}
}

type blockingRuntimeObserver struct {
	once    sync.Once
	started chan struct{}
	release chan struct{}
}

func (o *blockingRuntimeObserver) OnRuntimeEvent(ctx context.Context, _ runtimeevents.Event) error {
	o.once.Do(func() {
		close(o.started)
	})
	select {
	case <-o.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
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
