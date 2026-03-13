package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
	"github.com/sipeed/picoclaw/pkg/tools"
)

func TestRecordLastHeartbeatTarget(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace: tmpDir,

				Model: "test-model",

				MaxTokens: 4096,

				MaxToolIterations: 10,
			},
		},
	}

	msgBus := bus.NewMessageBus()

	provider := &mockProvider{}

	al := NewAgentLoop(cfg, msgBus, provider)

	target := "telegram:-100123/42"

	if err := al.RecordLastHeartbeatTarget(target); err != nil {
		t.Fatalf("RecordLastHeartbeatTarget failed: %v", err)
	}

	if got := al.state.GetLastHeartbeatTarget(); got != target {
		t.Fatalf("GetLastHeartbeatTarget = %q, want %q", got, target)
	}
}

func newTestAgentLoopSimple(t *testing.T) (*AgentLoop, func()) {
	t.Helper()
	al, _, _, _, cleanup := newTestAgentLoop(t) //nolint:dogsled
	return al, cleanup
}

func TestShouldInjectReminder(t *testing.T) {
	tests := []struct {
		name string

		iteration int

		interval int

		want bool
	}{
		{"first iteration skipped", 1, 5, false},

		{"iteration 5 interval 5", 5, 5, true},

		{"iteration 10 interval 5", 10, 5, true},

		{"iteration 3 interval 5", 3, 5, false},

		{"interval zero disabled", 5, 0, false},

		{"interval negative disabled", 5, -1, false},

		{"iteration 2 interval 1", 2, 1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldInjectReminder(tt.iteration, tt.interval)

			if got != tt.want {
				t.Errorf("shouldInjectReminder(%d, %d) = %v, want %v", tt.iteration, tt.interval, got, tt.want)
			}
		})
	}
}

func TestBuildTaskReminder_WithoutBlocker(t *testing.T) {
	msg := buildTaskReminder("implement feature X", "")

	if msg.Role != "user" {
		t.Errorf("expected role 'user', got %q", msg.Role)
	}

	if !strings.Contains(msg.Content, "[TASK REMINDER]") {
		t.Error("expected content to contain '[TASK REMINDER]'")
	}

	if !strings.Contains(msg.Content, "implement feature X") {
		t.Error("expected content to contain original message")
	}

	if strings.Contains(msg.Content, "blocker") {
		t.Error("expected content NOT to contain 'blocker' when no blocker provided")
	}

	if !strings.Contains(msg.Content, "move on") {
		t.Error("expected content to contain completion prompt")
	}
}

func TestBuildTaskReminder_WithBlocker(t *testing.T) {
	msg := buildTaskReminder("implement feature X", "ModuleNotFoundError: No module named 'foo'")

	if msg.Role != "user" {
		t.Errorf("expected role 'user', got %q", msg.Role)
	}

	if !strings.Contains(msg.Content, "[TASK REMINDER]") {
		t.Error("expected content to contain '[TASK REMINDER]'")
	}

	if !strings.Contains(msg.Content, "implement feature X") {
		t.Error("expected content to contain original message")
	}

	if !strings.Contains(msg.Content, "Last blocker") {
		t.Error("expected content to contain 'Last blocker'")
	}

	if !strings.Contains(msg.Content, "ModuleNotFoundError") {
		t.Error("expected content to contain blocker text")
	}
}

func TestResolveProvider_CachesProviders(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace: tmpDir,

				Model: "test-model",

				Provider: "vllm",

				MaxTokens: 4096,

				MaxToolIterations: 10,
			},
		},

		Providers: config.ProvidersConfig{
			VLLM: config.ProviderConfig{
				APIKey: "test-key",

				APIBase: "https://example.com/v1",
			},
		},
	}

	msgBus := bus.NewMessageBus()

	primary := &mockProvider{}

	al := NewAgentLoop(cfg, msgBus, primary)

	p1 := al.resolveProvider("vllm", "test-model", primary)

	if p1 == primary {
		t.Fatal("expected a new provider from legacy providers config, not the fallback")
	}

	p2 := al.resolveProvider("vllm", "test-model", primary)

	if p1 != p2 {
		t.Fatal("expected same cached instance on second call")
	}
}

func TestResolveProvider_FallsBackOnError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace: tmpDir,

				Model: "test-model",

				Provider: "vllm",

				MaxTokens: 4096,

				MaxToolIterations: 10,
			},
		},
	}

	msgBus := bus.NewMessageBus()

	primary := &mockProvider{}

	al := NewAgentLoop(cfg, msgBus, primary)

	p := al.resolveProvider("nonexistent", "unknown-model", primary)

	if p != primary {
		t.Fatal("expected fallback to primary provider on creation error")
	}

	if _, ok := al.providerCache["nonexistent"]; ok {
		t.Fatal("failed provider should not be cached")
	}
}

func TestResolveProvider_EmptyNameReturnsFallback(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace: tmpDir,

				Model: "test-model",

				MaxTokens: 4096,

				MaxToolIterations: 10,
			},
		},
	}

	msgBus := bus.NewMessageBus()

	primary := &mockProvider{}

	al := NewAgentLoop(cfg, msgBus, primary)

	p := al.resolveProvider("", "", primary)

	if p != primary {
		t.Fatal("expected fallback provider for empty name")
	}
}

func TestSlashCommandResponseSkipsPlaceholder(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace: tmpDir,

				Model: "test-model",

				MaxTokens: 4096,

				MaxToolIterations: 10,
			},
		},
	}

	msgBus := bus.NewMessageBus()

	provider := &mockProvider{}

	al := NewAgentLoop(cfg, msgBus, provider)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)

	defer cancel()

	go func() {
		_ = al.Run(ctx)
	}()

	msgBus.PublishInbound(context.Background(), bus.InboundMessage{
		Channel: "telegram",

		SenderID: "user1",

		ChatID: "chat1",

		Content: "/skills",
	})

	outMsg, ok := msgBus.SubscribeOutbound(ctx)

	if !ok {
		t.Fatal("expected outbound message from slash command")
	}

	if !outMsg.SkipPlaceholder {
		t.Errorf("expected SkipPlaceholder=true for slash command response, got false")
	}
}

func TestBuildTaskReminder_Truncation(t *testing.T) {
	longMsg := strings.Repeat("あ", 1000)

	longBlocker := strings.Repeat("X", 500)

	msg := buildTaskReminder(longMsg, longBlocker)

	runeCount := strings.Count(msg.Content, "あ")

	if runeCount >= 1000 {
		t.Errorf("expected task message to be truncated, got %d 'あ' runes", runeCount)
	}

	if runeCount > taskReminderMaxChars {
		t.Errorf("expected at most %d task runes, got %d", taskReminderMaxChars, runeCount)
	}

	xCount := strings.Count(msg.Content, "X")

	if xCount >= 500 {
		t.Errorf("expected blocker to be truncated, got %d 'X' chars", xCount)
	}

	if xCount > blockerMaxChars {
		t.Errorf("expected at most %d blocker chars, got %d", blockerMaxChars, xCount)
	}
}

func TestBuildPlanReminder(t *testing.T) {
	tests := []struct {
		name string

		status string

		wantOK bool

		wantSubstr string
	}{
		{"interviewing", "interviewing", true, "interviewing the user"},

		{"review", "review", true, "under review"},

		{"executing returns false", "executing", false, ""},

		{"empty returns false", "", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, ok := buildPlanReminder(tt.status)

			if ok != tt.wantOK {
				t.Fatalf("buildPlanReminder(%q) ok = %v, want %v", tt.status, ok, tt.wantOK)
			}

			if !ok {
				return
			}

			if msg.Role != "user" {
				t.Errorf("expected role 'user', got %q", msg.Role)
			}

			if !strings.Contains(msg.Content, tt.wantSubstr) {
				t.Errorf("expected content to contain %q, got %q", tt.wantSubstr, msg.Content)
			}
		})
	}
}

func TestPlanCommand_ShowNoPlan(t *testing.T) {
	al, cleanup := newTestAgentLoopSimple(t)

	defer cleanup()

	response, handled := al.handleCommand(context.Background(), bus.InboundMessage{Content: "/plan"})

	if !handled {
		t.Fatal("expected /plan to be handled")
	}

	if !strings.Contains(response, "No active plan") {
		t.Errorf("expected 'No active plan', got %q", response)
	}
}

func TestSplitChatAndThread(t *testing.T) {
	tests := []struct {
		name string

		chatID string

		wantChatID string

		wantThread int
	}{
		{name: "plain chat", chatID: "-100123", wantChatID: "-100123", wantThread: 0},

		{name: "chat with thread", chatID: "-100123/77", wantChatID: "-100123", wantThread: 77},

		{name: "invalid thread", chatID: "-100123/abc", wantChatID: "-100123", wantThread: 0},

		{name: "empty", chatID: "", wantChatID: "", wantThread: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotChatID, gotThread := splitChatAndThread(tt.chatID)

			if gotChatID != tt.wantChatID || gotThread != tt.wantThread {
				t.Fatalf(

					"splitChatAndThread(%q) = (%q, %d), want (%q, %d)",

					tt.chatID,

					gotChatID,

					gotThread,

					tt.wantChatID,

					tt.wantThread,
				)
			}
		})
	}
}

func TestHeartbeatCommandThreadHerePersistsConfig(t *testing.T) {
	al, cleanup := newTestAgentLoopSimple(t)

	defer cleanup()

	var saved bool

	var updatedThread int

	al.SetConfigSaver(func(cfg *config.Config) error {
		saved = true

		if cfg.Channels.Telegram.HeartbeatThreadID != 42 {
			t.Fatalf("HeartbeatThreadID in saver = %d, want 42", cfg.Channels.Telegram.HeartbeatThreadID)
		}

		return nil
	})

	al.SetHeartbeatThreadUpdater(func(threadID int) { updatedThread = threadID })

	msg := bus.InboundMessage{
		Content: "/heartbeat thread here",

		Channel: "telegram",

		ChatID: "-100500/42",
	}

	resp, handled := al.handleCommand(context.Background(), msg)

	if !handled {
		t.Fatal("expected /heartbeat command to be handled")
	}

	if !strings.Contains(resp, "Heartbeat thread set to 42") {
		t.Fatalf("unexpected response: %q", resp)
	}

	if !saved {
		t.Fatal("expected config saver to be called")
	}

	if updatedThread != 42 {
		t.Fatalf("updatedThread = %d, want 42", updatedThread)
	}

	if got := al.cfg.Channels.Telegram.HeartbeatThreadID; got != 42 {
		t.Fatalf("cfg heartbeat thread = %d, want 42", got)
	}

	if got := al.state.GetHeartbeatTarget(); got != "telegram:-100500" {
		t.Fatalf("state heartbeat target = %q, want %q", got, "telegram:-100500")
	}
}

func TestHeartbeatCommandThreadOff(t *testing.T) {
	al, cleanup := newTestAgentLoopSimple(t)

	defer cleanup()

	al.cfg.Channels.Telegram.HeartbeatThreadID = 99

	resp, handled := al.handleCommand(context.Background(), bus.InboundMessage{
		Content: "/heartbeat thread off",

		Channel: "telegram",

		ChatID: "-100500/42",
	})

	if !handled {
		t.Fatal("expected /heartbeat command to be handled")
	}

	if !strings.Contains(resp, "disabled") {
		t.Fatalf("unexpected response: %q", resp)
	}

	if got := al.cfg.Channels.Telegram.HeartbeatThreadID; got != 0 {
		t.Fatalf("cfg heartbeat thread = %d, want 0", got)
	}
}

func TestPlanCommand_StartNewPlan(t *testing.T) {
	al, cleanup := newTestAgentLoopSimple(t)

	defer cleanup()

	_, handled := al.handleCommand(context.Background(), bus.InboundMessage{Content: "/plan Set up monitoring"})

	if handled {
		t.Fatal("expected /plan <task> NOT to be handled (should fall through to LLM)")
	}

	msg := bus.InboundMessage{Content: "/plan Set up monitoring"}

	expanded, compact, ok := al.expandPlanCommand(msg)

	if !ok {
		t.Fatal("expected expandPlanCommand to succeed")
	}

	if expanded != "Set up monitoring" {
		t.Errorf("expected expanded = 'Set up monitoring', got %q", expanded)
	}

	if !strings.Contains(compact, "Set up monitoring") {
		t.Errorf("expected compact to contain task, got %q", compact)
	}

	agent := al.registry.GetDefaultAgent()

	if !agent.ContextBuilder.HasActivePlan() {
		t.Error("expected active plan after expandPlanCommand")
	}

	if status := agent.ContextBuilder.GetPlanStatus(); status != "interviewing" {
		t.Errorf("expected 'interviewing', got %q", status)
	}
}

func TestPlanCommand_StartBlockedByExisting(t *testing.T) {
	al, cleanup := newTestAgentLoopSimple(t)

	defer cleanup()

	al.expandPlanCommand(bus.InboundMessage{Content: "/plan First task"})

	response, handled := al.handleCommand(context.Background(), bus.InboundMessage{Content: "/plan Second task"})

	if !handled {
		t.Fatal("expected second /plan to be handled (blocked)")
	}

	if !strings.Contains(response, "already active") {
		t.Errorf("expected 'already active', got %q", response)
	}
}

func TestPlanCommand_Clear(t *testing.T) {
	al, cleanup := newTestAgentLoopSimple(t)

	defer cleanup()

	al.expandPlanCommand(bus.InboundMessage{Content: "/plan Test task"})

	response, _ := al.handleCommand(context.Background(), bus.InboundMessage{Content: "/plan clear"})

	if !strings.Contains(response, "Plan cleared") {
		t.Errorf("expected 'Plan cleared', got %q", response)
	}

	agent := al.registry.GetDefaultAgent()

	if agent.ContextBuilder.HasActivePlan() {
		t.Error("expected no plan after clear")
	}
}

func TestPlanCommand_ClearNoPlan(t *testing.T) {
	al, cleanup := newTestAgentLoopSimple(t)

	defer cleanup()

	response, _ := al.handleCommand(context.Background(), bus.InboundMessage{Content: "/plan clear"})

	if !strings.Contains(response, "No active plan") {
		t.Errorf("expected 'No active plan', got %q", response)
	}
}

func TestPlanCommand_Start(t *testing.T) {
	al, cleanup := newTestAgentLoopSimple(t)

	defer cleanup()

	agent := al.registry.GetDefaultAgent()

	plan := "# Active Plan\n\n> Task: Test task\n> Status: interviewing\n> Phase: 1\n\n## Phase 1: Setup\n- [ ] Step one\n\n## Context\n"

	_ = agent.ContextBuilder.WriteMemory(plan)

	response, _ := al.handleCommand(context.Background(), bus.InboundMessage{Content: "/plan start"})

	if !strings.Contains(response, "approved") {
		t.Errorf("expected 'approved', got %q", response)
	}

	if status := agent.ContextBuilder.GetPlanStatus(); status != "executing" {
		t.Errorf("expected 'executing', got %q", status)
	}

	if !al.planStartPending {
		t.Error("expected planStartPending to be true after /plan start")
	}
}

func TestPlanCommand_StartFromReview(t *testing.T) {
	al, cleanup := newTestAgentLoopSimple(t)

	defer cleanup()

	agent := al.registry.GetDefaultAgent()

	plan := "# Active Plan\n\n> Task: Test task\n> Status: review\n> Phase: 1\n\n## Phase 1: Setup\n- [ ] Step one\n\n## Context\n"

	_ = agent.ContextBuilder.WriteMemory(plan)

	response, _ := al.handleCommand(context.Background(), bus.InboundMessage{Content: "/plan start"})

	if !strings.Contains(response, "approved") {
		t.Errorf("expected 'approved', got %q", response)
	}

	if status := agent.ContextBuilder.GetPlanStatus(); status != "executing" {
		t.Errorf("expected 'executing', got %q", status)
	}

	if !al.planStartPending {
		t.Error("expected planStartPending to be true after /plan start from review")
	}
}

func TestPlanCommand_StartNoPhases(t *testing.T) {
	al, cleanup := newTestAgentLoopSimple(t)

	defer cleanup()

	al.expandPlanCommand(bus.InboundMessage{Content: "/plan Test task"})

	response, _ := al.handleCommand(context.Background(), bus.InboundMessage{Content: "/plan start"})

	if !strings.Contains(response, "no phases") {
		t.Errorf("expected 'no phases' error, got %q", response)
	}

	agent := al.registry.GetDefaultAgent()

	if status := agent.ContextBuilder.GetPlanStatus(); status != "interviewing" {
		t.Errorf("expected status to remain 'interviewing', got %q", status)
	}

	if al.planStartPending {
		t.Error("planStartPending must not be set when start is rejected (no phases)")
	}
}

func TestPlanCommand_StartAlreadyExecuting(t *testing.T) {
	al, cleanup := newTestAgentLoopSimple(t)

	defer cleanup()

	agent := al.registry.GetDefaultAgent()

	plan := "# Active Plan\n\n> Task: Test task\n> Status: interviewing\n> Phase: 1\n\n## Phase 1: Setup\n- [ ] Step one\n\n## Context\n"

	_ = agent.ContextBuilder.WriteMemory(plan)

	al.handleCommand(context.Background(), bus.InboundMessage{Content: "/plan start"})

	al.planStartPending = false

	response, _ := al.handleCommand(context.Background(), bus.InboundMessage{Content: "/plan start"})

	if !strings.Contains(response, "already executing") {
		t.Errorf("expected 'already executing', got %q", response)
	}

	if al.planStartPending {
		t.Error("planStartPending must not be set when plan is already executing")
	}
}

func TestPlanCommand_Done(t *testing.T) {
	al, cleanup := newTestAgentLoopSimple(t)

	defer cleanup()

	agent := al.registry.GetDefaultAgent()

	plan := `# Active Plan



> Task: Test task

> Status: executing

> Phase: 1



## Phase 1: Setup

- [ ] Step one

- [ ] Step two



## Context

Test context

`

	agent.ContextBuilder.WriteMemory(plan)

	response, _ := al.handleCommand(context.Background(), bus.InboundMessage{Content: "/plan done 1"})

	if !strings.Contains(response, "Marked step 1") {
		t.Errorf("expected confirmation, got %q", response)
	}
}

func TestPlanCommand_DoneInvalidStep(t *testing.T) {
	al, cleanup := newTestAgentLoopSimple(t)

	defer cleanup()

	al.expandPlanCommand(bus.InboundMessage{Content: "/plan Test task"})

	response, _ := al.handleCommand(context.Background(), bus.InboundMessage{Content: "/plan done abc"})

	if !strings.Contains(response, "positive integer") {
		t.Errorf("expected step validation error, got %q", response)
	}
}

func TestPlanCommand_Add(t *testing.T) {
	al, cleanup := newTestAgentLoopSimple(t)

	defer cleanup()

	agent := al.registry.GetDefaultAgent()

	plan := `# Active Plan



> Task: Test task

> Status: executing

> Phase: 1



## Phase 1: Setup

- [ ] Step one



## Context

Test context

`

	agent.ContextBuilder.WriteMemory(plan)

	response, _ := al.handleCommand(context.Background(), bus.InboundMessage{Content: "/plan add New step here"})

	if !strings.Contains(response, "Added step") {
		t.Errorf("expected 'Added step', got %q", response)
	}

	content := agent.ContextBuilder.ReadMemory()

	if !strings.Contains(content, "New step here") {
		t.Error("expected new step in plan content")
	}
}

func TestPlanCommand_Next(t *testing.T) {
	al, cleanup := newTestAgentLoopSimple(t)

	defer cleanup()

	agent := al.registry.GetDefaultAgent()

	plan := `# Active Plan



> Task: Test task

> Status: executing

> Phase: 1



## Phase 1: Setup

- [x] Step one



## Phase 2: Deploy

- [ ] Step two



## Context

Test

`

	agent.ContextBuilder.WriteMemory(plan)

	response, _ := al.handleCommand(context.Background(), bus.InboundMessage{Content: "/plan next"})

	if !strings.Contains(response, "phase 2") {
		t.Errorf("expected 'phase 2', got %q", response)
	}

	if phase := agent.ContextBuilder.GetCurrentPhase(); phase != 2 {
		t.Errorf("expected phase 2, got %d", phase)
	}
}

func TestPlanCommand_ShowActivePlan(t *testing.T) {
	al, cleanup := newTestAgentLoopSimple(t)

	defer cleanup()

	agent := al.registry.GetDefaultAgent()

	plan := `# Active Plan



> Task: Deploy app

> Status: executing

> Phase: 1



## Phase 1: Build

- [x] Compile code

- [ ] Run tests



## Context

Production server

`

	agent.ContextBuilder.WriteMemory(plan)

	response, _ := al.handleCommand(context.Background(), bus.InboundMessage{Content: "/plan"})

	if !strings.Contains(response, "Deploy app") {
		t.Errorf("expected task name in display, got %q", response)
	}

	if !strings.Contains(response, "Phase 1") {
		t.Errorf("expected phase info in display, got %q", response)
	}
}

func TestAutoPhaseAdvance(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-auto-advance-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace: tmpDir,

				Model: "test-model",

				MaxTokens: 4096,

				MaxToolIterations: 10,
			},
		},
	}

	msgBus := bus.NewMessageBus()

	provider := &simpleMockProvider{response: "OK"}

	al := NewAgentLoop(cfg, msgBus, provider)

	agent := al.registry.GetDefaultAgent()

	if agent == nil {
		t.Fatal("No default agent")
	}

	plan := `# Active Plan



> Task: Test auto advance

> Status: executing

> Phase: 1



## Phase 1: Setup

- [x] Step one

- [x] Step two



## Phase 2: Deploy

- [ ] Step three



## Context

Test

`

	agent.ContextBuilder.WriteMemory(plan)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()

	_, err = al.ProcessDirectWithChannel(ctx, "continue", "auto-advance-test", "test", "chat1")
	if err != nil {
		t.Fatalf("ProcessDirectWithChannel failed: %v", err)
	}

	if phase := agent.ContextBuilder.GetCurrentPhase(); phase != 2 {
		t.Errorf("expected phase auto-advanced to 2, got %d", phase)
	}
}

func TestAutoCompleteClears(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-auto-complete-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace: tmpDir,

				Model: "test-model",

				MaxTokens: 4096,

				MaxToolIterations: 10,
			},
		},
	}

	msgBus := bus.NewMessageBus()

	provider := &simpleMockProvider{response: "All done"}

	al := NewAgentLoop(cfg, msgBus, provider)

	agent := al.registry.GetDefaultAgent()

	if agent == nil {
		t.Fatal("No default agent")
	}

	plan := `# Active Plan



> Task: Test auto complete

> Status: executing

> Phase: 1



## Phase 1: Setup

- [x] Step one

- [x] Step two



## Context

Test

`

	agent.ContextBuilder.WriteMemory(plan)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()

	_, err = al.ProcessDirectWithChannel(ctx, "finish up", "auto-complete-test", "test", "chat1")
	if err != nil {
		t.Fatalf("ProcessDirectWithChannel failed: %v", err)
	}

	if !agent.ContextBuilder.HasActivePlan() {
		t.Error("expected plan to be retained after completion")
	}

	if status := agent.ContextBuilder.GetPlanStatus(); status != "completed" {
		t.Errorf("expected plan status 'completed', got %q", status)
	}

	if phase := agent.ContextBuilder.GetCurrentPhase(); phase != 1 {
		t.Errorf("expected phase 1 (total phases), got %d", phase)
	}
}

func TestIsToolAllowedDuringInterview_FuzzyNames(t *testing.T) {
	tests := []struct {
		name string

		args map[string]any

		want bool
	}{
		{"read_file", nil, true},

		{"list_dir", nil, true},

		{"web_search", nil, true},

		{"web_fetch", nil, true},

		{"readfile", nil, true},

		{"ReadFile", nil, true},

		{"listdir", nil, true},

		{"websearch", nil, true},

		{"webfetch", nil, true},

		{"message", nil, true},

		{"Message", nil, true},

		{"edit_file", map[string]any{"path": "/ws/memory/MEMORY.md"}, true},

		{"editfile", map[string]any{"path": "/ws/memory/MEMORY.md"}, true},

		{"EditFile", map[string]any{"path": "/ws/memory/MEMORY.md"}, true},

		{"edit_file", map[string]any{"path": "/ws/main.go"}, false},

		{"editfile", map[string]any{"path": "/ws/main.go"}, false},

		{"exec", map[string]any{"command": "find . -name '*.py'"}, true},

		{"exec", map[string]any{"command": "ls -la"}, true},

		{"exec", map[string]any{"command": "grep -r TODO ."}, true},

		{"exec", map[string]any{"command": "cat README.md"}, true},

		{"exec", map[string]any{"command": "cd /home/user/project && find . -type f"}, true},

		{"exec", map[string]any{"command": "cd /tmp && rm -rf *"}, false},

		{"exec", map[string]any{"command": "find . > output.txt"}, false},

		{"exec", map[string]any{"command": "ls -la >> log.txt"}, false},

		{"exec", map[string]any{"command": "cat foo | tee bar.txt"}, false},

		{"exec", map[string]any{"command": "cat ../../etc/passwd"}, false},

		{"exec", map[string]any{"command": "find ../../"}, false},

		{"exec", map[string]any{"command": "ls ../secret"}, false},

		{"exec", map[string]any{"command": "cat /etc/passwd"}, false},

		{"exec", map[string]any{"command": "find /etc -name '*.conf'"}, false},

		{"exec", map[string]any{"command": "ls /root"}, false},

		{"exec", map[string]any{"command": "rm -rf /"}, false},

		{"exec", map[string]any{"command": "mv a b"}, false},

		{"exec", nil, false},

		{"exec", map[string]any{"command": ""}, false},

		{"Exec", nil, false},
	}

	for _, tt := range tests {
		got := isToolAllowedDuringInterview(tt.name, tt.args)

		if got != tt.want {
			t.Errorf("isToolAllowedDuringInterview(%q, %v) = %v, want %v", tt.name, tt.args, got, tt.want)
		}
	}
}

func TestBuildArgsSnippet_ExecStripsCD(t *testing.T) {
	tests := []struct {
		name string

		tool string

		args map[string]any

		workspace string

		wantSnip string
	}{
		{
			name: "exec strips cd prefix",

			tool: "exec",

			args: map[string]any{
				"command": "cd /home/user/workspace/project/my-projects && pytest tests/test_integration.py",
			},

			workspace: "/home/user/workspace",

			wantSnip: "pytest tests/test_integration.py",
		},

		{
			name: "exec no cd prefix, flags stripped",

			tool: "exec",

			args: map[string]any{"command": "ls -la"},

			workspace: "/ws",

			wantSnip: "ls",
		},

		{
			name: "exec empty command",

			tool: "exec",

			args: map[string]any{},

			workspace: "/ws",

			wantSnip: "{}",
		},

		{
			name: "read_file strips workspace",

			tool: "read_file",

			args: map[string]any{"path": "/home/user/workspace/src/main.go"},

			workspace: "/home/user/workspace",

			wantSnip: "src/main.go",
		},

		{
			name: "edit_file shows path",

			tool: "edit_file",

			args: map[string]any{"path": "/ws/config.json", "old_text": "old value here"},

			workspace: "/ws",

			wantSnip: "config.json",
		},

		{
			name: "file tool long path prioritizes filename",

			tool: "read_file",

			args: map[string]any{
				"path": "/ws/projects/terra-py-form/src/terra_py_form/hot/state/backend.py",
			},

			workspace: "/ws",

			wantSnip: "projects/terra-py-form/src/terra_py_form/hot/sta\u2026/backend.py",
		},

		{
			name: "unknown tool shows raw JSON",

			tool: "web_search",

			args: map[string]any{"query": "hello"},

			workspace: "/ws",

			wantSnip: `{"query":"hello"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildArgsSnippet(tt.tool, tt.args, tt.workspace)

			if got != tt.wantSnip {
				t.Errorf("buildArgsSnippet(%q) = %q, want %q", tt.tool, got, tt.wantSnip)
			}
		})
	}
}

func TestFormatCompactEntry(t *testing.T) {
	tests := []struct {
		name string

		entry toolLogEntry

		wantSub string // must be a substring

		wantMark string // result marker must appear

		noTime bool // if true, duration should NOT appear
	}{
		{
			name: "exec short entry",

			entry: toolLogEntry{Name: "[1] exec", ArgsSnip: "ls", Result: "✓ 1.0s"},

			wantSub: "exec ls",

			wantMark: "✓ 1.0s",
		},

		{
			name: "exec long entry truncated from end",

			entry: toolLogEntry{
				Name: "[2] exec",

				ArgsSnip: "pytest tests/integration/test_very_long_name.py",

				Result: "✗ 3.0s",
			},

			wantMark: "✗",
		},

		{
			name: "file tool omits duration, shows filename",

			entry: toolLogEntry{
				Name: "[3] edit_file",

				ArgsSnip: "projects/terra/src/deep/nested/backend.py",

				Result: "✓ 0.0s",
			},

			wantSub: "backend.py",

			wantMark: "✓",

			noTime: true,
		},

		{
			name: "file tool path truncates from start",

			entry: toolLogEntry{
				Name: "[4] read_file",

				ArgsSnip: "projects/terra-py-form/src/terra_py_form/hot/state/backend.py",

				Result: "✓ 0.1s",
			},

			wantSub: "backend.py",

			wantMark: "✓",

			noTime: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatCompactEntry(tt.entry)

			if tt.wantSub != "" && !strings.Contains(got, tt.wantSub) {
				t.Errorf("expected to contain %q, got: %q", tt.wantSub, got)
			}

			if !strings.Contains(got, tt.wantMark) {
				t.Errorf("result marker %q missing from: %q", tt.wantMark, got)
			}

			if tt.noTime && strings.Contains(got, "0s") {
				t.Errorf("file tool should omit duration, got: %q", got)
			}

			if runeLen := len([]rune(got)); runeLen > maxEntryLineWidth {
				t.Errorf("entry too wide: %d runes (max %d): %q", runeLen, maxEntryLineWidth, got)
			}
		})
	}
}

func TestBuildRichStatus(t *testing.T) {
	task := &activeTask{
		Iteration: 3,

		MaxIter: 20,

		toolLog: []toolLogEntry{
			{Name: "exec", ArgsSnip: "ls -la", Result: "✓ 1.2s"},

			{Name: "exec", ArgsSnip: "pytest tests/", Result: "✓ 5.0s"},

			{Name: "read_file", ArgsSnip: "src/main.go", Result: "⏳"},
		},
	}

	got := buildRichStatus(task, false, "/home/user/my-projects")

	mustContain := []string{
		"Task in progress (3/20)",

		"my-projects",

		"read_file",

		"No errors",
	}

	for _, s := range mustContain {
		if !strings.Contains(got, s) {
			t.Errorf("expected output to contain %q, got:\n%s", s, got)
		}
	}

	if strings.Contains(got, "Reply to intervene") {
		t.Error("non-background task should not have reply prompt")
	}

	bgGot := buildRichStatus(task, true, "/home/user/my-projects")

	if !strings.Contains(bgGot, "Reply to intervene") {
		t.Error("background task should have reply prompt")
	}
}

func TestBuildRichStatus_ProjectDir(t *testing.T) {
	task := &activeTask{
		Iteration: 1,

		MaxIter: 10,

		projectDir: "terra-py-form",

		toolLog: []toolLogEntry{
			{Name: "exec", ArgsSnip: "ls", Result: "✓ 0.1s"},
		},
	}

	got := buildRichStatus(task, false, "/home/user/.picoclaw/workspace")

	if !strings.Contains(got, "terra-py-form") {
		t.Errorf("expected projectDir in output, got:\n%s", got)
	}

	task2 := &activeTask{
		Iteration: 1,

		MaxIter: 10,

		fileCommonDir: "projects/terra-py-form",

		toolLog: []toolLogEntry{
			{Name: "read_file", ArgsSnip: "src/main.py", Result: "✓ 0.1s"},
		},
	}

	got2 := buildRichStatus(task2, false, "/home/user/.picoclaw/workspace")

	if !strings.Contains(got2, "terra-py-form") {
		t.Errorf("expected fileCommonDir basename in output, got:\n%s", got2)
	}

	task3 := &activeTask{
		Iteration: 1,

		MaxIter: 10,

		toolLog: []toolLogEntry{
			{Name: "exec", ArgsSnip: "ls", Result: "✓ 0.1s"},
		},
	}

	for _, ws := range []string{"/home/user/my-project/", "/home/user/my-project"} {
		got := buildRichStatus(task3, false, ws)

		if !strings.Contains(got, "my-project") {
			t.Errorf("workspace %q: expected 'my-project' in output, got:\n%s", ws, got)
		}
	}
}

func TestExtractExecProjectDir(t *testing.T) {
	tests := []struct {
		name string

		cmd string

		want string
	}{
		{"cd deep path", "cd /ws/projects/terra-py-form && pytest", "terra-py-form"},

		{"cd direct subdir", "cd /ws/my-app && make build", "my-app"},

		{"cd trailing slash", "cd /ws/my-app/ && ls", "my-app"},

		{"cd to workspace", "cd /ws && ls", "ws"},

		{"no cd prefix", "pytest tests/", ""},

		{"empty command", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := map[string]any{"command": tt.cmd}

			got := extractExecProjectDir(args)

			if got != tt.want {
				t.Errorf("extractExecProjectDir(%q) = %q, want %q", tt.cmd, got, tt.want)
			}
		})
	}
}

func TestFileParentRelDir(t *testing.T) {
	ws := "/home/user/.picoclaw/workspace"

	tests := []struct {
		name string

		path string

		want string
	}{
		{"deep path", ws + "/projects/terra/src/main.py", "projects/terra/src"},

		{"direct subdir", ws + "/my-app/README.md", "my-app"},

		{"workspace root file", ws + "/notes.txt", ""},

		{"outside workspace", "/tmp/foo.txt", ""},

		{"trailing slash ws", ws + "/projects/terra/src/main.py", "projects/terra/src"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fileParentRelDir(tt.path, ws)

			if got != tt.want {
				t.Errorf("fileParentRelDir(%q, ws) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestCommonDirPrefix(t *testing.T) {
	tests := []struct {
		name string

		a, b string

		want string
	}{
		{"same dir", "projects/terra/src", "projects/terra/src", "projects/terra/src"},

		{"converge to project", "projects/terra/src", "projects/terra/tests", "projects/terra"},

		{"converge to top", "projects/terra/src", "projects/other/tests", "projects"},

		{"no common", "aaa/bbb", "ccc/ddd", ""},

		{"one is prefix", "projects/terra", "projects/terra/src", "projects/terra"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := commonDirPrefix(tt.a, tt.b)

			if got != tt.want {
				t.Errorf("commonDirPrefix(%q, %q) = %q, want %q", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestDisplayProjectDir(t *testing.T) {
	task1 := &activeTask{projectDir: "my-app", fileCommonDir: "projects/other"}

	if got := displayProjectDir(task1); got != "my-app" {
		t.Errorf("expected 'my-app', got %q", got)
	}

	task2 := &activeTask{fileCommonDir: "projects/terra-py-form"}

	if got := displayProjectDir(task2); got != "terra-py-form" {
		t.Errorf("expected 'terra-py-form', got %q", got)
	}

	task3 := &activeTask{fileCommonDir: "my-app"}

	if got := displayProjectDir(task3); got != "my-app" {
		t.Errorf("expected 'my-app', got %q", got)
	}

	task4 := &activeTask{}

	if got := displayProjectDir(task4); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestBuildRichStatus_FixedHeight(t *testing.T) {
	countLines := func(s string) int {
		return strings.Count(s, "\n")
	}

	task0 := &activeTask{Iteration: 1, MaxIter: 10}

	lines0 := countLines(buildRichStatus(task0, true, "/ws/p"))

	task1 := &activeTask{
		Iteration: 1, MaxIter: 10,

		toolLog: []toolLogEntry{{Name: "exec", ArgsSnip: "ls", Result: "⏳"}},
	}

	lines1 := countLines(buildRichStatus(task1, true, "/ws/p"))

	task5 := &activeTask{Iteration: 5, MaxIter: 10}

	for i := 0; i < 5; i++ {
		task5.toolLog = append(task5.toolLog, toolLogEntry{
			Name: fmt.Sprintf("[%d] exec", i), ArgsSnip: "cmd", Result: "✓ 1.0s",
		})
	}

	lines5 := countLines(buildRichStatus(task5, true, "/ws/p"))

	task5err := &activeTask{Iteration: 5, MaxIter: 10}

	for i := 0; i < 5; i++ {
		task5err.toolLog = append(task5err.toolLog, toolLogEntry{
			Name: fmt.Sprintf("[%d] exec", i), ArgsSnip: "cmd", Result: "✓ 1.0s",
		})
	}

	errEntry := toolLogEntry{
		Name: "[3] exec", ArgsSnip: "pytest", Result: "✗ 2.0s",

		ErrDetail: "FAILED test\nExit code: 1",
	}

	task5err.lastError = &errEntry

	lines5err := countLines(buildRichStatus(task5err, true, "/ws/p"))

	if lines0 != lines1 || lines1 != lines5 || lines5 != lines5err {
		t.Errorf("line counts should be equal: 0=%d, 1=%d, 5=%d, 5+err=%d",

			lines0, lines1, lines5, lines5err)
	}
}

func TestBuildRichStatus_StickyError(t *testing.T) {
	errEntry := toolLogEntry{
		Name: "[2] exec", ArgsSnip: "pytest", Result: "✗ 3.2s",

		ErrDetail: "FAILED test_login\nExit code: 1",
	}

	task := &activeTask{
		Iteration: 5,

		MaxIter: 10,

		toolLog: []toolLogEntry{
			{Name: "[3] read_file", ArgsSnip: "src/auth.py", Result: "✓ 0.1s"},

			{Name: "[4] edit_file", ArgsSnip: "src/auth.py", Result: "✓ 0.2s"},

			{Name: "[5] exec", ArgsSnip: "pytest --retry", Result: "⏳"},
		},

		lastError: &errEntry,
	}

	got := buildRichStatus(task, false, "/ws/p")

	if !strings.Contains(got, "FAILED test_login") {
		t.Errorf("expected sticky error detail in error section, got:\n%s", got)
	}

	if !strings.Contains(got, "\u274C") {
		t.Errorf("expected ❌ error header, got:\n%s", got)
	}

	if !strings.Contains(got, "pytest --retry") {
		t.Errorf("expected latest entry command, got:\n%s", got)
	}
}

func TestBuildRichStatus_LatestEntryNoInlineResult(t *testing.T) {
	longCmd := "uv run pytest tests/hot/test_state_backend_integration.py"

	task := &activeTask{
		Iteration: 2,

		MaxIter: 10,

		toolLog: []toolLogEntry{
			{Name: "exec", ArgsSnip: "ls -la", Result: "\u2713 0.5s"},

			{Name: "exec", ArgsSnip: longCmd, Result: "\u23F3"},
		},
	}

	got := buildRichStatus(task, false, "/ws/my-project")

	if !strings.Contains(got, "integration.py") {
		t.Errorf("latest entry should show filename, got:\n%s", got)
	}

	if !strings.Contains(got, "  \u23F3") {
		t.Errorf("latest entry result should be on indented line, got:\n%s", got)
	}

	if !strings.Contains(got, "my-project") {
		t.Errorf("should show project name, got:\n%s", got)
	}

	lines := strings.Split(got, "\n")

	sepCount := 0

	for _, l := range lines {
		if strings.HasPrefix(l, "\u2501") {
			sepCount++
		}
	}

	if sepCount != 1 {
		t.Errorf("expected exactly 1 separator, got %d in:\n%s", sepCount, got)
	}
}

func TestSanitizeHistoryForProvider_MultiToolCall(t *testing.T) {
	history := []providers.Message{
		{Role: "user", Content: "hello"},

		{Role: "assistant", Content: "", ToolCalls: []providers.ToolCall{
			{ID: "a", Function: &providers.FunctionCall{Name: "exec"}},

			{ID: "b", Function: &providers.FunctionCall{Name: "read_file"}},
		}},

		{Role: "tool", Content: "ok", ToolCallID: "a"},

		{Role: "tool", Content: "ok", ToolCallID: "b"},

		{Role: "assistant", Content: "done"},
	}

	got := sanitizeHistoryForProvider(history)

	if len(got) != 5 {
		roles := make([]string, len(got))

		for i, m := range got {
			roles[i] = m.Role
		}

		t.Fatalf("expected 5 messages, got %d: %v", len(got), roles)
	}

	toolCount := 0

	for _, m := range got {
		if m.Role == "tool" {
			toolCount++
		}
	}

	if toolCount != 2 {
		t.Errorf("expected 2 tool results, got %d", toolCount)
	}
}

func setupPlanNudgeTest(
	t *testing.T,
	plan, content, sessionKey string,
) (*countingMockProvider, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "agent-nudge-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace: tmpDir,

				Model: "test-model",

				MaxTokens: 4096,

				MaxToolIterations: 10,
			},
		},
	}

	provider := &countingMockProvider{}

	msgBus := bus.NewMessageBus()

	al := NewAgentLoop(cfg, msgBus, provider)

	agent := al.registry.GetDefaultAgent()

	if agent == nil {
		t.Fatal("no default agent")
	}

	agent.ContextBuilder.WriteMemory(plan)

	ctx, cancel := context.WithTimeout(
		context.Background(), 5*time.Second,
	)

	msg := bus.InboundMessage{
		Channel: "test",

		SenderID: "user1",

		ChatID: "chat1",

		Content: content,

		SessionKey: sessionKey,
	}

	_, err = al.processMessage(ctx, msg)

	cancel()

	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("processMessage failed: %v", err)
	}

	return provider, func() { os.RemoveAll(tmpDir) }
}

func TestPlanNudge_ForegroundExecution(t *testing.T) {
	plan := "# Active Plan\n\n> Task: Test\n" +
		"> Status: executing\n> Phase: 1\n\n" +
		"## Phase 1: Setup\n- [ ] Step one\n" +
		"- [ ] Step two\n\n## Context\n"

	provider, cleanup := setupPlanNudgeTest(
		t, plan, "continue working", "nudge-test",
	)

	defer cleanup()

	if provider.calls < 2 {
		t.Errorf(
			"expected at least 2 provider calls"+
				" (nudge should trigger continuation),"+
				" got %d", provider.calls,
		)
	}
}

func TestPlanNudge_NoNudgeWhenAllStepsComplete(t *testing.T) {
	plan := "# Active Plan\n\n> Task: Test\n" +
		"> Status: executing\n> Phase: 1\n\n" +
		"## Phase 1: Setup\n- [x] Step one\n" +
		"- [x] Step two\n\n## Context\n"

	provider, cleanup := setupPlanNudgeTest(
		t, plan, "all done", "nudge-test-complete",
	)

	defer cleanup()

	if provider.calls != 1 {
		t.Errorf(
			"expected exactly 1 provider call"+
				" (no nudge needed), got %d",
			provider.calls,
		)
	}
}

func TestPlanNudge_ProgressMessage(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-nudge-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace: tmpDir,

				Model: "test-model",

				MaxTokens: 4096,

				MaxToolIterations: 10,
			},
		},
	}

	var nudgeContent string

	provider := &nudgeCaptureMockProvider{onSecondCall: func(msgs []providers.Message) {
		for i := len(msgs) - 1; i >= 0; i-- {
			if msgs[i].Role == "user" {
				nudgeContent = msgs[i].Content

				break
			}
		}
	}}

	msgBus := bus.NewMessageBus()

	al := NewAgentLoop(cfg, msgBus, provider)

	agent := al.registry.GetDefaultAgent()

	if agent == nil {
		t.Fatal("no default agent")
	}

	plan := "# Active Plan\n\n> Task: Test\n> Status: executing\n> Phase: 1\n\n## Phase 1: Setup\n- [ ] Step one\n- [ ] Step two\n- [ ] Step three\n\n## Context\n"

	agent.ContextBuilder.WriteMemory(plan)

	provider.onFirstCall = func() {
		updated := strings.Replace(agent.ContextBuilder.ReadMemory(), "- [ ] Step one", "- [x] Step one", 1)

		agent.ContextBuilder.WriteMemory(updated)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()

	msg := bus.InboundMessage{
		Channel: "test",

		SenderID: "user1",

		ChatID: "chat1",

		Content: "work on the plan",

		SessionKey: "nudge-progress-test",
	}

	_, err = al.processMessage(ctx, msg)
	if err != nil {
		t.Fatalf("processMessage failed: %v", err)
	}

	if !strings.Contains(nudgeContent, "Progress recorded") {
		t.Errorf("expected 'Progress recorded' nudge, got %q", nudgeContent)
	}

	if !strings.Contains(nudgeContent, "2 unchecked steps remain") {
		t.Errorf("expected '2 unchecked steps remain' in nudge, got %q", nudgeContent)
	}
}

type nudgeCaptureMockProvider struct {
	calls int

	onFirstCall func()

	onSecondCall func([]providers.Message)
}

func (m *nudgeCaptureMockProvider) Chat(
	_ context.Context,
	messages []providers.Message,
	_ []providers.ToolDefinition,
	_ string,
	_ map[string]any,
) (*providers.LLMResponse, error) {
	m.calls++
	if m.calls == 1 && m.onFirstCall != nil {
		m.onFirstCall()
	}
	if m.calls == 2 && m.onSecondCall != nil {
		m.onSecondCall(messages)
	}
	return &providers.LLMResponse{Content: "ok"}, nil
}

func (m *nudgeCaptureMockProvider) GetDefaultModel() string {
	return "nudge-mock"
}

func TestConsumeStream_NormalCompletion(t *testing.T) {
	ch := make(chan protocoltypes.StreamEvent, 8)

	go func() {
		ch <- protocoltypes.StreamEvent{ContentDelta: "Hello "}

		ch <- protocoltypes.StreamEvent{ContentDelta: "world!"}

		ch <- protocoltypes.StreamEvent{
			FinishReason: "stop",

			Usage: &providers.UsageInfo{PromptTokens: 5, CompletionTokens: 2, TotalTokens: 7},
		}

		close(ch)
	}()

	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	resp, detected, err := consumeStreamWithRepetitionDetection(ch, cancel, 1000, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if detected {
		t.Fatal("expected detected=false for normal content")
	}

	if resp.Content != "Hello world!" {
		t.Errorf("Content = %q, want %q", resp.Content, "Hello world!")
	}

	if resp.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want %q", resp.FinishReason, "stop")
	}

	if resp.Usage == nil || resp.Usage.TotalTokens != 7 {
		t.Errorf("Usage.TotalTokens = %v, want 7", resp.Usage)
	}

	_ = ctx
}

func TestConsumeStream_DetectsRepetition(t *testing.T) {
	ch := make(chan protocoltypes.StreamEvent, 64)

	cancelCalled := false

	ctx, cancel := context.WithCancel(context.Background())

	wrappedCancel := func() {
		cancelCalled = true

		cancel()
	}

	repeatedChunk := strings.Repeat("abcdefghij", 50)

	go func() {
		for i := 0; i < 6; i++ {
			ch <- protocoltypes.StreamEvent{ContentDelta: repeatedChunk}
		}

		for i := 0; i < 10; i++ {
			ch <- protocoltypes.StreamEvent{ContentDelta: "more data"}
		}

		close(ch)
	}()

	resp, detected, err := consumeStreamWithRepetitionDetection(ch, wrappedCancel, 1000, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !detected {
		t.Fatal("expected repetition detection to trigger")
	}

	if !cancelCalled {
		t.Error("expected cancelFn to be called")
	}

	if len(resp.Content) >= 3000+10*len("more data") {
		t.Errorf("Content length = %d, expected less than full output", len(resp.Content))
	}

	_ = ctx
}

func TestConsumeStream_ToolCallAccumulation(t *testing.T) {
	ch := make(chan protocoltypes.StreamEvent, 8)

	go func() {
		ch <- protocoltypes.StreamEvent{
			ToolCallDeltas: []protocoltypes.StreamToolCallDelta{
				{Index: 0, ID: "call_1", Name: "test_fn", ArgumentsDelta: `{"ke`},
			},
		}

		ch <- protocoltypes.StreamEvent{
			ToolCallDeltas: []protocoltypes.StreamToolCallDelta{
				{Index: 0, ArgumentsDelta: `y":"val"}`},
			},
		}

		ch <- protocoltypes.StreamEvent{FinishReason: "tool_calls"}

		close(ch)
	}()

	_, cancel := context.WithCancel(context.Background())

	defer cancel()

	resp, detected, err := consumeStreamWithRepetitionDetection(ch, cancel, 1000, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if detected {
		t.Fatal("expected no repetition detection for tool calls")
	}

	if len(resp.ToolCalls) != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1", len(resp.ToolCalls))
	}

	if resp.ToolCalls[0].Name != "test_fn" {
		t.Errorf("ToolCalls[0].Name = %q, want %q", resp.ToolCalls[0].Name, "test_fn")
	}

	if resp.ToolCalls[0].Arguments["key"] != "val" {
		t.Errorf("ToolCalls[0].Arguments[key] = %v, want %q", resp.ToolCalls[0].Arguments["key"], "val")
	}
}

func TestConsumeStream_StreamError(t *testing.T) {
	ch := make(chan protocoltypes.StreamEvent, 4)

	go func() {
		ch <- protocoltypes.StreamEvent{ContentDelta: "partial"}

		ch <- protocoltypes.StreamEvent{Err: fmt.Errorf("read error")}

		close(ch)
	}()

	_, cancel := context.WithCancel(context.Background())

	defer cancel()

	_, _, err := consumeStreamWithRepetitionDetection(ch, cancel, 1000, nil)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "read error") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "read error")
	}
}

func TestConsumeStream_OnChunkCallback(t *testing.T) {
	ch := make(chan protocoltypes.StreamEvent, 8)

	go func() {
		ch <- protocoltypes.StreamEvent{ContentDelta: "Hello "}

		ch <- protocoltypes.StreamEvent{ContentDelta: "world"}

		ch <- protocoltypes.StreamEvent{ContentDelta: "!"}

		ch <- protocoltypes.StreamEvent{FinishReason: "stop"}

		close(ch)
	}()

	_, cancel := context.WithCancel(context.Background())

	defer cancel()

	var chunks []string

	onChunk := func(accumulated, _ string) {
		chunks = append(chunks, accumulated)
	}

	resp, detected, err := consumeStreamWithRepetitionDetection(ch, cancel, 1000, onChunk)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if detected {
		t.Fatal("expected detected=false")
	}

	if resp.Content != "Hello world!" {
		t.Errorf("Content = %q, want %q", resp.Content, "Hello world!")
	}

	if len(chunks) != 3 {
		t.Fatalf("onChunk called %d times, want 3", len(chunks))
	}

	if chunks[0] != "Hello " {
		t.Errorf("chunks[0] = %q, want %q", chunks[0], "Hello ")
	}

	if chunks[1] != "Hello world" {
		t.Errorf("chunks[1] = %q, want %q", chunks[1], "Hello world")
	}

	if chunks[2] != "Hello world!" {
		t.Errorf("chunks[2] = %q, want %q", chunks[2], "Hello world!")
	}
}

func TestConsumeStream_OnChunkWithRepetitionDetection(t *testing.T) {
	ch := make(chan protocoltypes.StreamEvent, 64)

	cancelCalled := false

	ctx, cancel := context.WithCancel(context.Background())

	wrappedCancel := func() {
		cancelCalled = true

		cancel()
	}

	repeatedChunk := strings.Repeat("abcdefghij", 50)

	go func() {
		for i := 0; i < 6; i++ {
			ch <- protocoltypes.StreamEvent{ContentDelta: repeatedChunk}
		}

		for i := 0; i < 10; i++ {
			ch <- protocoltypes.StreamEvent{ContentDelta: "more data"}
		}

		close(ch)
	}()

	var chunkCount int

	onChunk := func(_, _ string) {
		chunkCount++
	}

	_, detected, err := consumeStreamWithRepetitionDetection(ch, wrappedCancel, 1000, onChunk)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !detected {
		t.Fatal("expected repetition detection to trigger")
	}

	if !cancelCalled {
		t.Error("expected cancelFn to be called")
	}

	if chunkCount == 0 {
		t.Error("expected onChunk to be called at least once")
	}

	_ = ctx
}

type modelCapturingMockProvider struct {
	mu sync.Mutex

	models []string

	response string
}

func (m *modelCapturingMockProvider) Chat(
	_ context.Context,
	_ []providers.Message,
	_ []providers.ToolDefinition,
	model string,
	_ map[string]any,
) (*providers.LLMResponse, error) {
	m.mu.Lock()
	m.models = append(m.models, model)
	m.mu.Unlock()
	resp := m.response
	if resp == "" {
		resp = "ok"
	}
	return &providers.LLMResponse{Content: resp}, nil
}

func (m *modelCapturingMockProvider) GetDefaultModel() string {
	return "model-capturing-mock"
}

func setupPlanModelTest(
	t *testing.T,
	response, memoryContent, userMsg, sessionKey string,
) (*modelCapturingMockProvider, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "agent-test-planmodel-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace: tmpDir,

				Model: "normal-model",

				PlanModel: "plan-model",

				MaxTokens: 4096,

				MaxToolIterations: 2,
			},
		},
	}

	msgBus := bus.NewMessageBus()

	provider := &modelCapturingMockProvider{response: response}

	al := NewAgentLoop(cfg, msgBus, provider)

	defaultAgent := al.registry.GetDefaultAgent()

	if defaultAgent == nil {
		t.Fatal("No default agent found")
	}

	memoryDir := filepath.Join(tmpDir, "memory")

	os.MkdirAll(memoryDir, 0o755)

	memoryPath := filepath.Join(memoryDir, "MEMORY.md")

	if wErr := os.WriteFile(
		memoryPath, []byte(memoryContent), 0o644,
	); wErr != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to write MEMORY.md: %v", wErr)
	}

	_, err = al.ProcessDirectWithChannel(
		context.Background(),
		userMsg,
		sessionKey,
		"test",
		"test-chat",
	)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("ProcessDirectWithChannel failed: %v", err)
	}

	return provider, func() { os.RemoveAll(tmpDir) }
}

func TestAgentLoop_PlanModel_UsedDuringInterviewing(t *testing.T) {
	mem := "# Active Plan\n\n" +
		"> Task: Test plan model\n" +
		"> Status: interviewing\n> Phase: 1\n"

	provider, cleanup := setupPlanModelTest(
		t, "Plan interview response", mem,
		"Hello, plan model test", "test-plan-session",
	)

	defer cleanup()

	provider.mu.Lock()

	defer provider.mu.Unlock()

	if len(provider.models) == 0 {
		t.Fatal("Expected at least one Chat call")
	}

	if provider.models[0] != "plan-model" {
		t.Errorf(
			"Expected plan model 'plan-model'"+
				" during interviewing, got %q",
			provider.models[0],
		)
	}
}

func TestAgentLoop_PlanModel_NotUsedDuringExecuting(t *testing.T) {
	mem := "# Active Plan\n\n\n\n" +
		"> Task: Test plan model\n\n" +
		"> Status: executing\n\n> Phase: 1\n\n\n\n" +
		"## Phase 1: Build\n\n- [ ] Run build\n\n"

	provider, cleanup := setupPlanModelTest(
		t, "Executing response", mem,
		"Hello, executing test", "test-exec-session",
	)

	defer cleanup()

	provider.mu.Lock()

	defer provider.mu.Unlock()

	if len(provider.models) == 0 {
		t.Fatal("Expected at least one Chat call")
	}

	if provider.models[0] != "normal-model" {
		t.Errorf(
			"Expected normal model 'normal-model'"+
				" during executing, got %q",
			provider.models[0],
		)
	}
}

func TestAgentLoop_PlanModel_ResolvesProviderForSingleCandidate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-planmodel-resolve-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace: tmpDir,

				Model: "MiniMax-M2.5",

				PlanModel: "openai/gpt-5.2",

				MaxTokens: 4096,

				MaxToolIterations: 2,
			},
		},
	}

	msgBus := bus.NewMessageBus()

	mainProvider := &modelCapturingMockProvider{response: "wrong provider response"}

	al := NewAgentLoop(cfg, msgBus, mainProvider)

	resolvedProvider := &modelCapturingMockProvider{response: "correct provider response"}

	al.providerCache["openai/gpt-5.2"] = resolvedProvider

	memoryDir := filepath.Join(tmpDir, "memory")

	os.MkdirAll(memoryDir, 0o755)

	memoryPath := filepath.Join(memoryDir, "MEMORY.md")

	memoryContent := "# Active Plan\n\n> Task: Test provider resolution\n> Status: interviewing\n> Phase: 1\n"

	if wErr := os.WriteFile(memoryPath, []byte(memoryContent), 0o644); wErr != nil {
		t.Fatalf("Failed to write MEMORY.md: %v", wErr)
	}

	_, err = al.ProcessDirectWithChannel(

		context.Background(),

		"Hello, resolve provider test",

		"test-resolve-session",

		"test",

		"test-chat",
	)
	if err != nil {
		t.Fatalf("ProcessDirectWithChannel failed: %v", err)
	}

	resolvedProvider.mu.Lock()

	defer resolvedProvider.mu.Unlock()

	mainProvider.mu.Lock()

	defer mainProvider.mu.Unlock()

	if len(resolvedProvider.models) == 0 {
		t.Fatal("Expected resolved provider to receive Chat call, but it got none")
	}

	if resolvedProvider.models[0] != "gpt-5.2" {
		t.Errorf("Expected resolved provider to receive model 'gpt-5.2', got %q", resolvedProvider.models[0])
	}

	if len(mainProvider.models) > 0 {
		t.Errorf("Expected main provider to receive no Chat calls during plan model phase, got %d calls with models %v",

			len(mainProvider.models), mainProvider.models)
	}
}

func TestPlanCommand_StartClear(t *testing.T) {
	al, cleanup := newTestAgentLoopSimple(t)

	defer cleanup()

	agent := al.registry.GetDefaultAgent()

	plan := "# Active Plan\n\n> Task: Test task\n> Status: review\n> Phase: 1\n\n## Phase 1: Setup\n- [ ] Step one\n\n## Context\n"

	_ = agent.ContextBuilder.WriteMemory(plan)

	agent.Sessions.AddMessage("test-session", "user", "hello")

	agent.Sessions.AddMessage("test-session", "assistant", "world")

	agent.Sessions.SetSummary("test-session", "some summary")

	response, handled := al.handleCommand(context.Background(), bus.InboundMessage{
		Content: "/plan start clear",

		SessionKey: "test-session",
	})

	if !handled {
		t.Fatal("expected /plan start clear to be handled")
	}

	if !strings.Contains(response, "clean history") {
		t.Errorf("expected 'clean history' in response, got %q", response)
	}

	if !al.planStartPending {
		t.Error("expected planStartPending to be true")
	}

	if !al.planClearHistory {
		t.Error("expected planClearHistory to be true")
	}

	al.planStartPending = false

	clearHistory := al.planClearHistory

	al.planClearHistory = false

	if clearHistory {
		agent.Sessions.SetHistory("test-session", nil)

		agent.Sessions.SetSummary("test-session", "")

		_ = agent.Sessions.Save("test-session")
	}

	history := agent.Sessions.GetHistory("test-session")

	if len(history) != 0 {
		t.Errorf("expected empty history after clear, got %d messages", len(history))
	}

	summary := agent.Sessions.GetSummary("test-session")

	if summary != "" {
		t.Errorf("expected empty summary after clear, got %q", summary)
	}
}

func TestPlanCommand_StartWithoutClear_PreservesHistory(t *testing.T) {
	al, cleanup := newTestAgentLoopSimple(t)

	defer cleanup()

	agent := al.registry.GetDefaultAgent()

	plan := "# Active Plan\n\n> Task: Test task\n> Status: review\n> Phase: 1\n\n## Phase 1: Setup\n- [ ] Step one\n\n## Context\n"

	_ = agent.ContextBuilder.WriteMemory(plan)

	agent.Sessions.AddMessage("test-session", "user", "hello")

	agent.Sessions.AddMessage("test-session", "assistant", "world")

	agent.Sessions.SetSummary("test-session", "some summary")

	response, _ := al.handleCommand(context.Background(), bus.InboundMessage{
		Content: "/plan start",

		SessionKey: "test-session",
	})

	if strings.Contains(response, "clean history") {
		t.Errorf("did not expect 'clean history' in response, got %q", response)
	}

	if al.planClearHistory {
		t.Error("planClearHistory should be false for /plan start without clear")
	}

	history := agent.Sessions.GetHistory("test-session")

	if len(history) != 2 {
		t.Errorf("expected 2 history messages preserved, got %d", len(history))
	}

	summary := agent.Sessions.GetSummary("test-session")

	if summary != "some summary" {
		t.Errorf("expected summary preserved, got %q", summary)
	}
}

func TestFilterInterviewTools(t *testing.T) {
	allDefs := []providers.ToolDefinition{
		{Function: protocoltypes.ToolFunctionDefinition{Name: "read_file"}},

		{Function: protocoltypes.ToolFunctionDefinition{Name: "list_dir"}},

		{Function: protocoltypes.ToolFunctionDefinition{Name: "web_search"}},

		{Function: protocoltypes.ToolFunctionDefinition{Name: "web_fetch"}},

		{Function: protocoltypes.ToolFunctionDefinition{Name: "message"}},

		{Function: protocoltypes.ToolFunctionDefinition{Name: "edit_file"}},

		{Function: protocoltypes.ToolFunctionDefinition{Name: "append_file"}},

		{Function: protocoltypes.ToolFunctionDefinition{Name: "write_file"}},

		{Function: protocoltypes.ToolFunctionDefinition{Name: "exec"}},

		{Function: protocoltypes.ToolFunctionDefinition{Name: "logs"}},

		{Function: protocoltypes.ToolFunctionDefinition{Name: "spawn_subagent"}},

		{Function: protocoltypes.ToolFunctionDefinition{Name: "skills_search"}},

		{Function: protocoltypes.ToolFunctionDefinition{Name: "skills_install"}},

		{Function: protocoltypes.ToolFunctionDefinition{Name: "bg_monitor"}},

		{Function: protocoltypes.ToolFunctionDefinition{Name: "i2c_transfer"}},
	}

	filtered := filterInterviewTools(allDefs)

	if len(filtered) != 10 {
		names := make([]string, len(filtered))

		for i, d := range filtered {
			names[i] = d.Function.Name
		}

		t.Errorf("expected 10 allowed tools, got %d: %v", len(filtered), names)
	}

	disallowed := map[string]bool{
		"spawnsubagent": true, "skillssearch": true,

		"skillsinstall": true, "bgmonitor": true, "ictransfer": true,
	}

	for _, d := range filtered {
		norm := tools.NormalizeToolName(d.Function.Name)

		if disallowed[norm] {
			t.Errorf("disallowed tool %q should have been filtered out", d.Function.Name)
		}
	}
}

func TestBuildStreamingDisplay_ContentOnly(t *testing.T) {
	display := buildStreamingDisplay("hello world", "")

	if !strings.HasSuffix(display, " \u2589") {
		t.Error("expected cursor suffix")
	}

	if strings.Contains(display, "\U0001f9e0") {
		t.Error("should not contain brain emoji when no reasoning")
	}

	lines := strings.Count(display, "\n") + 1

	if lines != streamingDisplayLines+1 {
		t.Logf("display:\n%s", display)
	}
}

func TestBuildStreamingDisplay_ReasoningOnly(t *testing.T) {
	display := buildStreamingDisplay("", "let me think about this")

	if !strings.Contains(display, "\U0001f9e0") {
		t.Error("expected brain emoji for reasoning phase")
	}

	if !strings.Contains(display, "Thinking...") {
		t.Error("expected Thinking... header")
	}

	if !strings.HasSuffix(display, " \u2589") {
		t.Error("expected cursor suffix")
	}
}

func TestBuildStreamingDisplay_Both(t *testing.T) {
	display := buildStreamingDisplay("the answer is 42", "first I considered...")

	if !strings.Contains(display, "\U0001f9e0") {
		t.Error("expected brain emoji")
	}

	if !strings.Contains(display, "responding") {
		t.Error("expected responding header when both present")
	}

	if !strings.Contains(display, "the answer is 42") {
		t.Error("expected content in display")
	}
}

func TestFormatDurationMs(t *testing.T) {
	tests := []struct {
		ms int64

		want string
	}{
		{0, "0ms"},

		{500, "500ms"},

		{999, "999ms"},

		{1000, "1.0s"},

		{1200, "1.2s"},

		{3500, "3.5s"},

		{59900, "59.9s"},

		{60000, "1m"},

		{61000, "1m1s"},

		{65000, "1m5s"},

		{120000, "2m"},

		{3661000, "61m1s"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%dms", tt.ms), func(t *testing.T) {
			got := formatDurationMs(tt.ms)

			if got != tt.want {
				t.Errorf("formatDurationMs(%d) = %q, want %q", tt.ms, got, tt.want)
			}
		})
	}
}

func TestFormatSubagentCompletion(t *testing.T) {
	tests := []struct {
		name string

		label string

		metadata map[string]string

		want string
	}{
		{
			"no metadata",

			"scout-1",

			nil,

			"📋 scout-1 completed.",
		},

		{
			"empty metadata",

			"scout-1",

			map[string]string{},

			"📋 scout-1 completed.",
		},

		{
			"duration and tool calls",

			"scout-1",

			map[string]string{"duration_ms": "3200", "tool_calls": "5"},

			"📋 scout-1 completed (3.2s, 5 tool calls).",
		},

		{
			"single tool call",

			"coder-1",

			map[string]string{"duration_ms": "1200", "tool_calls": "1"},

			"📋 coder-1 completed (1.2s, 1 tool call).",
		},

		{
			"duration only",

			"scout-2",

			map[string]string{"duration_ms": "65000", "tool_calls": "0"},

			"📋 scout-2 completed (1m5s).",
		},

		{
			"tool calls only",

			"scout-3",

			map[string]string{"duration_ms": "0", "tool_calls": "10"},

			"📋 scout-3 completed (10 tool calls).",
		},

		{
			"zero everything",

			"scout-4",

			map[string]string{"duration_ms": "0", "tool_calls": "0"},

			"📋 scout-4 completed.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatSubagentCompletion(tt.label, tt.metadata)

			if got != tt.want {
				t.Errorf("formatSubagentCompletion(%q, %v) = %q, want %q", tt.label, tt.metadata, got, tt.want)
			}
		})
	}
}
