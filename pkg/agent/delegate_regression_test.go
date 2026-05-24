package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/session"
	toolspkg "github.com/sipeed/picoclaw/pkg/tools"
)

func TestDelegateTool_SubagentsOnlyConfig_UsesSubTurnPath(t *testing.T) {
	tmpDir := t.TempDir()
	plannerDir := filepath.Join(tmpDir, "planner")
	researchDir := filepath.Join(tmpDir, "research")
	if err := os.MkdirAll(plannerDir, 0o755); err != nil {
		t.Fatalf("mkdir planner workspace: %v", err)
	}
	if err := os.MkdirAll(researchDir, 0o755); err != nil {
		t.Fatalf("mkdir research workspace: %v", err)
	}

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 4,
			},
			List: []config.AgentConfig{
				{
					ID:        "planner",
					Default:   true,
					Workspace: plannerDir,
					Subagents: &config.SubagentsConfig{
						AllowAgents: []string{"research"},
					},
				},
				{
					ID:        "research",
					Workspace: researchDir,
				},
			},
		},
	}

	al := NewAgentLoop(cfg, bus.NewMessageBus(), &plainProvider{})
	defer al.Close()

	planner, ok := al.registry.GetAgent("planner")
	if !ok || planner == nil {
		t.Fatal("expected planner agent")
	}
	delegateTool, ok := planner.Tools.Get("delegate")
	if !ok {
		t.Fatal("expected delegate tool")
	}

	parent := &turnState{
		ctx:            context.Background(),
		turnID:         "parent-planner",
		agent:          planner,
		agentID:        planner.ID,
		session:        newEphemeralSession(nil),
		pendingResults: make(chan *toolspkg.ToolResult, 4),
		concurrencySem: make(chan struct{}, testMaxConcurrentSubTurns),
	}

	result := delegateTool.Execute(withTurnState(context.Background(), parent), map[string]any{
		"agent_id": "research",
		"task":     "Summarize the logs",
	})

	if result.IsError {
		t.Fatalf("expected delegate to use sub-turn path, got error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, `[Response from agent "research"]`) {
		t.Fatalf("result.ForLLM = %q", result.ForLLM)
	}
}

func TestDelegateTool_CommunicationOnlyConfig_UsesCollaborationPath(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := collaborationTestConfig(tmpDir)
	al := NewAgentLoop(cfg, bus.NewMessageBus(), &collaborationReplyProvider{})
	defer al.Close()

	planner, ok := al.registry.GetAgent("planner")
	if !ok || planner == nil {
		t.Fatal("expected planner agent")
	}
	delegateTool, ok := planner.Tools.Get("delegate")
	if !ok {
		t.Fatal("expected delegate tool")
	}

	plannerScope := &session.SessionScope{
		Version:    session.ScopeVersionV1,
		AgentID:    "planner",
		Channel:    "cli",
		Account:    "default",
		Dimensions: []string{"chat"},
		Values:     map[string]string{"chat": "direct:tester"},
	}
	plannerSessionKey := session.BuildSessionKey(*plannerScope)
	ctx := toolspkg.WithToolSessionContext(context.Background(), "planner", plannerSessionKey, plannerScope)

	result := delegateTool.Execute(ctx, map[string]any{
		"agent_id": "research",
		"task":     "Summarize the collaboration tradeoffs",
	})

	if result.IsError {
		t.Fatalf("expected delegate to use collaboration path, got error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, `[Response from agent "research"]`) {
		t.Fatalf("result.ForLLM = %q", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "Tradeoffs: faster indexing") {
		t.Fatalf("result.ForLLM = %q", result.ForLLM)
	}

	inbox, err := al.collaboration.Inbox(ctx, toolspkg.AgentInboxParams{})
	if err != nil {
		t.Fatalf("Inbox() error = %v", err)
	}
	if len(inbox.Messages) == 0 {
		t.Fatal("expected collaboration reply in planner inbox")
	}
	waitForCollaborationIdle(t, al, inbox.Messages[0].ThreadID, "research")
}
