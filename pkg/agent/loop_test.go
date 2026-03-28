package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/routing"
	"github.com/sipeed/picoclaw/pkg/tools"
)

type fakeChannel struct{ id string }

func (f *fakeChannel) Name() string                                            { return "fake" }
func (f *fakeChannel) Start(ctx context.Context) error                         { return nil }
func (f *fakeChannel) Stop(ctx context.Context) error                          { return nil }
func (f *fakeChannel) Send(ctx context.Context, msg bus.OutboundMessage) error { return nil }
func (f *fakeChannel) IsRunning() bool                                         { return true }
func (f *fakeChannel) IsAllowed(string) bool                                   { return true }
func (f *fakeChannel) IsAllowedSender(sender bus.SenderInfo) bool              { return true }
func (f *fakeChannel) ReasoningChannelID() string                              { return f.id }

type fakeMediaChannel struct {
	fakeChannel
	sentMedia []bus.OutboundMediaMessage
}

func (f *fakeMediaChannel) SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) error {
	f.sentMedia = append(f.sentMedia, msg)
	return nil
}

func newStartedTestChannelManager(
	t *testing.T,
	msgBus *bus.MessageBus,
	store media.MediaStore,
	name string,
	ch channels.Channel,
) *channels.Manager {
	t.Helper()

	cm, err := channels.NewManager(&config.Config{}, msgBus, store)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	cm.RegisterChannel(name, ch)
	if err := cm.StartAll(context.Background()); err != nil {
		t.Fatalf("StartAll() error = %v", err)
	}
	t.Cleanup(func() {
		if err := cm.StopAll(context.Background()); err != nil {
			t.Fatalf("StopAll() error = %v", err)
		}
	})
	return cm
}

type recordingProvider struct {
	lastMessages []providers.Message
}

func (r *recordingProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	opts map[string]any,
) (*providers.LLMResponse, error) {
	r.lastMessages = append([]providers.Message(nil), messages...)
	return &providers.LLMResponse{
		Content:   "Mock response",
		ToolCalls: []providers.ToolCall{},
	}, nil
}

func (r *recordingProvider) GetDefaultModel() string {
	return "mock-model"
}

func newTestAgentLoop(
	t *testing.T,
) (al *AgentLoop, cfg *config.Config, msgBus *bus.MessageBus, provider *mockProvider, cleanup func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	cfg = &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}
	msgBus = bus.NewMessageBus()
	provider = &mockProvider{}
	al = NewAgentLoop(cfg, msgBus, provider)
	return al, cfg, msgBus, provider, func() { os.RemoveAll(tmpDir) }
}

func TestProcessMessage_IncludesCurrentSenderInDynamicContext(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &recordingProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)

	response, err := al.processMessage(context.Background(), bus.InboundMessage{
		Channel:  "discord",
		SenderID: "discord:123",
		Sender: bus.SenderInfo{
			DisplayName: "Alice",
		},
		ChatID:  "group-1",
		Content: "hello",
	})
	if err != nil {
		t.Fatalf("processMessage() error = %v", err)
	}
	if response != "Mock response" {
		t.Fatalf("processMessage() response = %q, want %q", response, "Mock response")
	}
	if len(provider.lastMessages) == 0 {
		t.Fatal("provider did not receive any messages")
	}

	systemPrompt := provider.lastMessages[0].Content
	wantSender := "## Current Sender\nCurrent sender: Alice (ID: discord:123)"
	if !strings.Contains(systemPrompt, wantSender) {
		t.Fatalf("system prompt missing sender context %q:\n%s", wantSender, systemPrompt)
	}

	lastMessage := provider.lastMessages[len(provider.lastMessages)-1]
	if lastMessage.Role != "user" || lastMessage.Content != "hello" {
		t.Fatalf("last provider message = %+v, want unchanged user message", lastMessage)
	}
}

func TestProcessMessage_UseCommandLoadsRequestedSkill(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "skills", "shell")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(skillDir, "SKILL.md"),
		[]byte("# shell\n\nPrefer concise shell commands and explain them briefly."),
		0o644,
	); err != nil {
		t.Fatalf("write skill file: %v", err)
	}

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}
	msgBus := bus.NewMessageBus()
	provider := &recordingProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)

	response, err := al.processMessage(context.Background(), bus.InboundMessage{
		Channel:  "telegram",
		SenderID: "telegram:123",
		ChatID:   "chat-1",
		Content:  "/use shell explain how to list files",
	})
	if err != nil {
		t.Fatalf("processMessage() error = %v", err)
	}
	if response != "Mock response" {
		t.Fatalf("processMessage() response = %q, want %q", response, "Mock response")
	}
	if len(provider.lastMessages) == 0 {
		t.Fatal("provider did not receive any messages")
	}

	systemPrompt := provider.lastMessages[0].Content
	if !strings.Contains(systemPrompt, "# Active Skills") {
		t.Fatalf("system prompt missing active skills section:\n%s", systemPrompt)
	}
	if !strings.Contains(systemPrompt, "### Skill: shell") {
		t.Fatalf("system prompt missing requested skill content:\n%s", systemPrompt)
	}

	lastMessage := provider.lastMessages[len(provider.lastMessages)-1]
	if lastMessage.Role != "user" || lastMessage.Content != "explain how to list files" {
		t.Fatalf("last provider message = %+v, want rewritten user message", lastMessage)
	}
}

func TestHandleCommand_UseCommandRejectsUnknownSkill(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}
	msgBus := bus.NewMessageBus()
	provider := &recordingProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)
	agent := al.GetRegistry().GetDefaultAgent()

	opts := processOptions{}
	reply, handled := al.handleCommand(context.Background(), bus.InboundMessage{
		Channel:  "telegram",
		SenderID: "telegram:123",
		ChatID:   "chat-1",
		Content:  "/use missing explain how to list files",
	}, agent, &opts)
	if !handled {
		t.Fatal("expected /use with unknown skill to be handled")
	}
	if !strings.Contains(reply, "Unknown skill: missing") {
		t.Fatalf("reply = %q, want unknown skill error", reply)
	}
}

func TestProcessMessage_UseCommandArmsSkillForNextMessage(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "skills", "shell")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(skillDir, "SKILL.md"),
		[]byte("# shell\n\nPrefer concise shell commands and explain them briefly."),
		0o644,
	); err != nil {
		t.Fatalf("write skill file: %v", err)
	}

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}
	msgBus := bus.NewMessageBus()
	provider := &recordingProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)

	response, err := al.processMessage(context.Background(), bus.InboundMessage{
		Channel:  "telegram",
		SenderID: "telegram:123",
		ChatID:   "chat-1",
		Content:  "/use shell",
	})
	if err != nil {
		t.Fatalf("processMessage() arm error = %v", err)
	}
	if !strings.Contains(response, `Skill "shell" is armed for your next message.`) {
		t.Fatalf("arm response = %q, want armed confirmation", response)
	}

	response, err = al.processMessage(context.Background(), bus.InboundMessage{
		Channel:  "telegram",
		SenderID: "telegram:123",
		ChatID:   "chat-1",
		Content:  "explain how to list files",
	})
	if err != nil {
		t.Fatalf("processMessage() follow-up error = %v", err)
	}
	if response != "Mock response" {
		t.Fatalf("follow-up response = %q, want %q", response, "Mock response")
	}
	if len(provider.lastMessages) == 0 {
		t.Fatal("provider did not receive any messages")
	}

	systemPrompt := provider.lastMessages[0].Content
	if !strings.Contains(systemPrompt, "### Skill: shell") {
		t.Fatalf("system prompt missing pending skill content:\n%s", systemPrompt)
	}
	lastMessage := provider.lastMessages[len(provider.lastMessages)-1]
	if lastMessage.Role != "user" || lastMessage.Content != "explain how to list files" {
		t.Fatalf("last provider message = %+v, want unchanged follow-up user message", lastMessage)
	}
}

func TestApplyExplicitSkillCommand_ArmsSkillForNextMessage(t *testing.T) {
	al, cfg, _, _, cleanup := newTestAgentLoop(t)
	defer cleanup()

	if err := os.MkdirAll(filepath.Join(cfg.Agents.Defaults.Workspace, "skills", "finance-news"), 0o755); err != nil {
		t.Fatalf("MkdirAll(skill) error = %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(cfg.Agents.Defaults.Workspace, "skills", "finance-news", "SKILL.md"),
		[]byte("# Finance News\n\nUse web tools for current finance updates.\n"),
		0o644,
	); err != nil {
		t.Fatalf("WriteFile(SKILL.md) error = %v", err)
	}

	agent := al.GetRegistry().GetDefaultAgent()
	if agent == nil {
		t.Fatal("expected default agent")
	}

	opts := &processOptions{SessionKey: "agent:main:test"}
	matched, handled, reply := al.applyExplicitSkillCommand("/use finance-news", agent, opts)
	if !matched {
		t.Fatal("expected /use command to match")
	}
	if !handled {
		t.Fatal("expected /use without inline message to be handled immediately")
	}
	if !strings.Contains(reply, `Skill "finance-news" is armed for your next message`) {
		t.Fatalf("unexpected reply: %q", reply)
	}

	pending := al.takePendingSkills(opts.SessionKey)
	if len(pending) != 1 || pending[0] != "finance-news" {
		t.Fatalf("pending skills = %#v, want [finance-news]", pending)
	}
}

func TestApplyExplicitSkillCommand_InlineMessageMutatesOptions(t *testing.T) {
	al, cfg, _, _, cleanup := newTestAgentLoop(t)
	defer cleanup()

	if err := os.MkdirAll(filepath.Join(cfg.Agents.Defaults.Workspace, "skills", "finance-news"), 0o755); err != nil {
		t.Fatalf("MkdirAll(skill) error = %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(cfg.Agents.Defaults.Workspace, "skills", "finance-news", "SKILL.md"),
		[]byte("# Finance News\n\nUse web tools for current finance updates.\n"),
		0o644,
	); err != nil {
		t.Fatalf("WriteFile(SKILL.md) error = %v", err)
	}

	agent := al.GetRegistry().GetDefaultAgent()
	if agent == nil {
		t.Fatal("expected default agent")
	}

	opts := &processOptions{
		SessionKey:  "agent:main:test",
		UserMessage: "/use finance-news dammi le ultime news",
	}
	matched, handled, reply := al.applyExplicitSkillCommand(opts.UserMessage, agent, opts)
	if !matched {
		t.Fatal("expected /use command to match")
	}
	if handled {
		t.Fatal("expected /use with inline message to fall through into normal agent execution")
	}
	if reply != "" {
		t.Fatalf("unexpected reply: %q", reply)
	}
	if opts.UserMessage != "dammi le ultime news" {
		t.Fatalf("opts.UserMessage = %q, want %q", opts.UserMessage, "dammi le ultime news")
	}
	if len(opts.ForcedSkills) != 1 || opts.ForcedSkills[0] != "finance-news" {
		t.Fatalf("opts.ForcedSkills = %#v, want [finance-news]", opts.ForcedSkills)
	}
}

func TestRecordLastChannel(t *testing.T) {
	al, cfg, msgBus, provider, cleanup := newTestAgentLoop(t)
	defer cleanup()

	testChannel := "test-channel"
	if err := al.RecordLastChannel(testChannel); err != nil {
		t.Fatalf("RecordLastChannel failed: %v", err)
	}
	if got := al.state.GetLastChannel(); got != testChannel {
		t.Errorf("Expected channel '%s', got '%s'", testChannel, got)
	}
	al2 := NewAgentLoop(cfg, msgBus, provider)
	if got := al2.state.GetLastChannel(); got != testChannel {
		t.Errorf("Expected persistent channel '%s', got '%s'", testChannel, got)
	}
}

func TestRecordLastChatID(t *testing.T) {
	al, cfg, msgBus, provider, cleanup := newTestAgentLoop(t)
	defer cleanup()

	testChatID := "test-chat-id-123"
	if err := al.RecordLastChatID(testChatID); err != nil {
		t.Fatalf("RecordLastChatID failed: %v", err)
	}
	if got := al.state.GetLastChatID(); got != testChatID {
		t.Errorf("Expected chat ID '%s', got '%s'", testChatID, got)
	}
	al2 := NewAgentLoop(cfg, msgBus, provider)
	if got := al2.state.GetLastChatID(); got != testChatID {
		t.Errorf("Expected persistent chat ID '%s', got '%s'", testChatID, got)
	}
}

func TestNewAgentLoop_StateInitialized(t *testing.T) {
	// Create temp workspace
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test config
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}

	// Create agent loop
	msgBus := bus.NewMessageBus()
	provider := &mockProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)

	// Verify state manager is initialized
	if al.state == nil {
		t.Error("Expected state manager to be initialized")
	}

	// Verify state directory was created
	stateDir := filepath.Join(tmpDir, "state")
	if _, err := os.Stat(stateDir); os.IsNotExist(err) {
		t.Error("Expected state directory to exist")
	}
}

// TestToolRegistry_ToolRegistration verifies tools can be registered and retrieved
func TestToolRegistry_ToolRegistration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &mockProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)

	// Register a custom tool
	customTool := &mockCustomTool{}
	al.RegisterTool(customTool)

	// Verify tool is registered by checking it doesn't panic on GetStartupInfo
	// (actual tool retrieval is tested in tools package tests)
	info := al.GetStartupInfo()
	toolsInfo := info["tools"].(map[string]any)
	toolsList := toolsInfo["names"].([]string)

	// Check that our custom tool name is in the list
	found := slices.Contains(toolsList, "mock_custom")
	if !found {
		t.Error("Expected custom tool to be registered")
	}
}

// TestToolContext_Updates verifies tool context helpers work correctly
func TestToolContext_Updates(t *testing.T) {
	ctx := tools.WithToolContext(context.Background(), "telegram", "chat-42")

	if got := tools.ToolChannel(ctx); got != "telegram" {
		t.Errorf("expected channel 'telegram', got %q", got)
	}
	if got := tools.ToolChatID(ctx); got != "chat-42" {
		t.Errorf("expected chatID 'chat-42', got %q", got)
	}

	// Empty context returns empty strings
	if got := tools.ToolChannel(context.Background()); got != "" {
		t.Errorf("expected empty channel from bare context, got %q", got)
	}
}

// TestToolRegistry_GetDefinitions verifies tool definitions can be retrieved
func TestToolRegistry_GetDefinitions(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &mockProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)

	// Register a test tool and verify it shows up in startup info
	testTool := &mockCustomTool{}
	al.RegisterTool(testTool)

	info := al.GetStartupInfo()
	toolsInfo := info["tools"].(map[string]any)
	toolsList := toolsInfo["names"].([]string)

	// Check that our custom tool name is in the list
	found := slices.Contains(toolsList, "mock_custom")
	if !found {
		t.Error("Expected custom tool to be registered")
	}
}

func TestProcessMessage_MediaToolHandledSkipsFollowUpLLMAndFinalText(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &handledMediaProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)

	store := media.NewFileMediaStore()
	al.SetMediaStore(store)
	telegramChannel := &fakeMediaChannel{fakeChannel: fakeChannel{id: "rid-telegram"}}
	al.SetChannelManager(newStartedTestChannelManager(t, msgBus, store, "telegram", telegramChannel))

	imagePath := filepath.Join(tmpDir, "screen.png")
	if err := os.WriteFile(imagePath, []byte("fake screenshot"), 0o644); err != nil {
		t.Fatalf("WriteFile(imagePath) error = %v", err)
	}

	al.RegisterTool(&handledMediaTool{
		store: store,
		path:  imagePath,
	})

	response, err := al.processMessage(context.Background(), bus.InboundMessage{
		Channel:  "telegram",
		ChatID:   "chat1",
		SenderID: "user1",
		Content:  "take a screenshot of the screen and send it to me",
	})
	if err != nil {
		t.Fatalf("processMessage() error = %v", err)
	}
	if response != "" {
		t.Fatalf("expected no final response when media tool already handled delivery, got %q", response)
	}
	if provider.calls != 1 {
		t.Fatalf("expected exactly 1 LLM call, got %d", provider.calls)
	}
	if len(provider.toolCounts) != 1 {
		t.Fatalf("expected tool counts for 1 provider call, got %d", len(provider.toolCounts))
	}
	if provider.toolCounts[0] == 0 {
		t.Fatal("expected tools to be available on the first LLM call")
	}

	if len(telegramChannel.sentMedia) != 1 {
		t.Fatalf("expected exactly 1 synchronously sent media message, got %d", len(telegramChannel.sentMedia))
	}
	if telegramChannel.sentMedia[0].Channel != "telegram" || telegramChannel.sentMedia[0].ChatID != "chat1" {
		t.Fatalf("unexpected sent media target: %+v", telegramChannel.sentMedia[0])
	}
	if len(telegramChannel.sentMedia[0].Parts) != 1 {
		t.Fatalf("expected exactly 1 sent media part, got %d", len(telegramChannel.sentMedia[0].Parts))
	}

	select {
	case extra := <-msgBus.OutboundMediaChan():
		t.Fatalf("expected handled media to bypass async queue, got %+v", extra)
	default:
	}

	defaultAgent := al.GetRegistry().GetDefaultAgent()
	if defaultAgent == nil {
		t.Fatal("expected default agent")
	}
	route, _, err := al.resolveMessageRoute(bus.InboundMessage{
		Channel:  "telegram",
		ChatID:   "chat1",
		SenderID: "user1",
		Content:  "take a screenshot of the screen and send it to me",
	})
	if err != nil {
		t.Fatalf("resolveMessageRoute() error = %v", err)
	}
	sessionKey := resolveScopeKey(route, "")
	history := defaultAgent.Sessions.GetHistory(sessionKey)
	if len(history) == 0 {
		t.Fatal("expected session history to be saved")
	}
	last := history[len(history)-1]
	if last.Role != "assistant" || last.Content != "Requested output delivered via tool attachment." {
		t.Fatalf("expected handled assistant summary in history, got %+v", last)
	}
}

func TestProcessMessage_HandledToolProcessesQueuedSteeringBeforeReturning(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &handledMediaWithSteeringProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)

	store := media.NewFileMediaStore()
	al.SetMediaStore(store)
	telegramChannel := &fakeMediaChannel{fakeChannel: fakeChannel{id: "rid-telegram"}}
	al.SetChannelManager(newStartedTestChannelManager(t, msgBus, store, "telegram", telegramChannel))

	imagePath := filepath.Join(tmpDir, "screen-steering.png")
	if err := os.WriteFile(imagePath, []byte("fake screenshot"), 0o644); err != nil {
		t.Fatalf("WriteFile(imagePath) error = %v", err)
	}

	al.RegisterTool(&handledMediaWithSteeringTool{
		store: store,
		path:  imagePath,
		loop:  al,
	})

	response, err := al.processMessage(context.Background(), bus.InboundMessage{
		Channel:  "telegram",
		ChatID:   "chat1",
		SenderID: "user1",
		Content:  "take a screenshot of the screen and send it to me",
	})
	if err != nil {
		t.Fatalf("processMessage() error = %v", err)
	}
	if response != "Handled the queued steering message." {
		t.Fatalf("response = %q, want queued steering response", response)
	}
	if provider.calls != 2 {
		t.Fatalf("expected 2 LLM calls after queued steering, got %d", provider.calls)
	}
	if len(telegramChannel.sentMedia) != 1 {
		t.Fatalf("expected exactly 1 synchronously sent media message, got %d", len(telegramChannel.sentMedia))
	}
}

func TestProcessMessage_MediaArtifactCanBeForwardedBySendFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = tmpDir
	cfg.Agents.Defaults.ModelName = "test-model"
	cfg.Agents.Defaults.MaxTokens = 4096
	cfg.Agents.Defaults.MaxToolIterations = 10

	msgBus := bus.NewMessageBus()
	provider := &artifactThenSendProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)

	store := media.NewFileMediaStore()
	al.SetMediaStore(store)
	telegramChannel := &fakeMediaChannel{fakeChannel: fakeChannel{id: "rid-telegram"}}
	al.SetChannelManager(newStartedTestChannelManager(t, msgBus, store, "telegram", telegramChannel))

	mediaDir := media.TempDir()
	if err := os.MkdirAll(mediaDir, 0o700); err != nil {
		t.Fatalf("MkdirAll(mediaDir) error = %v", err)
	}
	imagePath := filepath.Join(mediaDir, "artifact-screen.png")
	if err := os.WriteFile(imagePath, []byte("fake screenshot"), 0o644); err != nil {
		t.Fatalf("WriteFile(imagePath) error = %v", err)
	}

	al.RegisterTool(&mediaArtifactTool{
		store: store,
		path:  imagePath,
	})

	response, err := al.processMessage(context.Background(), bus.InboundMessage{
		Channel:  "telegram",
		ChatID:   "chat1",
		SenderID: "user1",
		Content:  "take a screenshot of the screen and send it to me",
	})
	if err != nil {
		t.Fatalf("processMessage() error = %v", err)
	}
	if response != "" {
		t.Fatalf("expected no final response after send_file handled delivery, got %q", response)
	}
	if provider.calls != 2 {
		t.Fatalf("expected 2 LLM calls (artifact + send_file), got %d", provider.calls)
	}

	if len(telegramChannel.sentMedia) != 1 {
		t.Fatalf("expected exactly 1 synchronously sent media message, got %d", len(telegramChannel.sentMedia))
	}
	if telegramChannel.sentMedia[0].Channel != "telegram" || telegramChannel.sentMedia[0].ChatID != "chat1" {
		t.Fatalf("unexpected sent media target: %+v", telegramChannel.sentMedia[0])
	}
	if len(telegramChannel.sentMedia[0].Parts) != 1 {
		t.Fatalf("expected exactly 1 sent media part, got %d", len(telegramChannel.sentMedia[0].Parts))
	}

	select {
	case extra := <-msgBus.OutboundMediaChan():
		t.Fatalf("expected synchronous send_file delivery to bypass async queue, got %+v", extra)
	default:
	}
}

// TestAgentLoop_GetStartupInfo verifies startup info contains tools
func TestAgentLoop_GetStartupInfo(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = tmpDir
	cfg.Agents.Defaults.ModelName = "test-model"
	cfg.Agents.Defaults.MaxTokens = 4096
	cfg.Agents.Defaults.MaxToolIterations = 10

	msgBus := bus.NewMessageBus()
	provider := &mockProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)

	info := al.GetStartupInfo()

	// Verify tools info exists
	toolsInfo, ok := info["tools"]
	if !ok {
		t.Fatal("Expected 'tools' key in startup info")
	}

	toolsMap, ok := toolsInfo.(map[string]any)
	if !ok {
		t.Fatal("Expected 'tools' to be a map")
	}

	count, ok := toolsMap["count"]
	if !ok {
		t.Fatal("Expected 'count' in tools info")
	}

	// Should have default tools registered
	if count.(int) == 0 {
		t.Error("Expected at least some tools to be registered")
	}
}

// TestAgentLoop_Stop verifies Stop() sets running to false
func TestAgentLoop_Stop(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &mockProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)

	// Note: running is only set to true when Run() is called
	// We can't test that without starting the event loop
	// Instead, verify the Stop method can be called safely
	al.Stop()

	// Verify running is false (initial state or after Stop)
	if al.running.Load() {
		t.Error("Expected agent to be stopped (or never started)")
	}
}

// Mock implementations for testing

type simpleMockProvider struct {
	response string
}

func (m *simpleMockProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	opts map[string]any,
) (*providers.LLMResponse, error) {
	return &providers.LLMResponse{
		Content:   m.response,
		ToolCalls: []providers.ToolCall{},
	}, nil
}

func (m *simpleMockProvider) GetDefaultModel() string {
	return "mock-model"
}

type reasoningContentProvider struct {
	response         string
	reasoningContent string
}

func (m *reasoningContentProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	opts map[string]any,
) (*providers.LLMResponse, error) {
	return &providers.LLMResponse{
		Content:          m.response,
		ReasoningContent: m.reasoningContent,
		ToolCalls:        []providers.ToolCall{},
	}, nil
}

func (m *reasoningContentProvider) GetDefaultModel() string {
	return "reasoning-content-model"
}

type countingMockProvider struct {
	response string
	calls    int
}

func (m *countingMockProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	opts map[string]any,
) (*providers.LLMResponse, error) {
	m.calls++
	return &providers.LLMResponse{
		Content:   m.response,
		ToolCalls: []providers.ToolCall{},
	}, nil
}

func (m *countingMockProvider) GetDefaultModel() string {
	return "counting-mock-model"
}

type handledMediaProvider struct {
	calls      int
	toolCounts []int
}

func (m *handledMediaProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	opts map[string]any,
) (*providers.LLMResponse, error) {
	m.calls++
	m.toolCounts = append(m.toolCounts, len(tools))
	if m.calls == 1 {
		return &providers.LLMResponse{
			Content: "Taking the screenshot now.",
			ToolCalls: []providers.ToolCall{{
				ID:        "call_handled_media",
				Type:      "function",
				Name:      "handled_media_tool",
				Arguments: map[string]any{},
			}},
		}, nil
	}
	return &providers.LLMResponse{}, nil
}

func (m *handledMediaProvider) GetDefaultModel() string {
	return "handled-media-model"
}

type artifactThenSendProvider struct {
	calls int
}

func (m *artifactThenSendProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	opts map[string]any,
) (*providers.LLMResponse, error) {
	m.calls++
	if m.calls == 1 {
		return &providers.LLMResponse{
			Content: "Taking the screenshot now.",
			ToolCalls: []providers.ToolCall{{
				ID:        "call_artifact_media",
				Type:      "function",
				Name:      "media_artifact_tool",
				Arguments: map[string]any{},
			}},
		}, nil
	}

	var artifactPath string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role != "tool" {
			continue
		}
		start := strings.Index(messages[i].Content, "[file:")
		if start < 0 {
			continue
		}
		rest := messages[i].Content[start+len("[file:"):]
		end := strings.Index(rest, "]")
		if end < 0 {
			continue
		}
		artifactPath = rest[:end]
		break
	}
	if artifactPath == "" {
		return nil, fmt.Errorf("provider did not receive artifact path in tool result")
	}

	return &providers.LLMResponse{
		Content: "",
		ToolCalls: []providers.ToolCall{{
			ID:        "call_send_file",
			Type:      "function",
			Name:      "send_file",
			Arguments: map[string]any{"path": artifactPath},
		}},
	}, nil
}

func (m *artifactThenSendProvider) GetDefaultModel() string {
	return "artifact-then-send-model"
}

type toolFeedbackProvider struct {
	filePath string
	calls    int
}

func (m *toolFeedbackProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	opts map[string]any,
) (*providers.LLMResponse, error) {
	m.calls++
	if m.calls == 1 {
		return &providers.LLMResponse{
			ToolCalls: []providers.ToolCall{{
				ID:        "call_heartbeat_read_file",
				Type:      "function",
				Name:      "read_file",
				Arguments: map[string]any{"path": m.filePath},
			}},
		}, nil
	}

	return &providers.LLMResponse{
		Content:   "HEARTBEAT_OK",
		ToolCalls: []providers.ToolCall{},
	}, nil
}

func (m *toolFeedbackProvider) GetDefaultModel() string {
	return "heartbeat-tool-feedback-model"
}

type toolLimitOnlyProvider struct{}

func (m *toolLimitOnlyProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	opts map[string]any,
) (*providers.LLMResponse, error) {
	return &providers.LLMResponse{
		ToolCalls: []providers.ToolCall{{
			ID:        "call_tool_limit_test",
			Type:      "function",
			Name:      "tool_limit_test_tool",
			Arguments: map[string]any{"value": "x"},
		}},
	}, nil
}

func (m *toolLimitOnlyProvider) GetDefaultModel() string {
	return "tool-limit-only-model"
}

// mockCustomTool is a simple mock tool for registration testing
type mockCustomTool struct{}

func (m *mockCustomTool) Name() string {
	return "mock_custom"
}

func (m *mockCustomTool) Description() string {
	return "Mock custom tool for testing"
}

func (m *mockCustomTool) Parameters() map[string]any {
	return map[string]any{
		"type":                 "object",
		"properties":           map[string]any{},
		"additionalProperties": true,
	}
}

func (m *mockCustomTool) Execute(ctx context.Context, args map[string]any) *tools.ToolResult {
	return tools.SilentResult("Custom tool executed")
}

type handledMediaTool struct {
	store media.MediaStore
	path  string
}

func (m *handledMediaTool) Name() string { return "handled_media_tool" }
func (m *handledMediaTool) Description() string {
	return "Returns a media attachment and fully handles the user response"
}

func (m *handledMediaTool) Parameters() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

func (m *handledMediaTool) Execute(ctx context.Context, args map[string]any) *tools.ToolResult {
	ref, err := m.store.Store(m.path, media.MediaMeta{
		Filename:    filepath.Base(m.path),
		ContentType: "image/png",
		Source:      "test:handled_media_tool",
	}, "test:handled_media")
	if err != nil {
		return tools.ErrorResult(err.Error()).WithError(err)
	}
	return tools.MediaResult("Attachment delivered by tool.", []string{ref}).WithResponseHandled()
}

type handledMediaWithSteeringProvider struct {
	calls int
}

func (m *handledMediaWithSteeringProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	opts map[string]any,
) (*providers.LLMResponse, error) {
	m.calls++
	if m.calls == 1 {
		return &providers.LLMResponse{
			Content: "Taking the screenshot now.",
			ToolCalls: []providers.ToolCall{{
				ID:        "call_handled_media_steering",
				Type:      "function",
				Name:      "handled_media_with_steering_tool",
				Arguments: map[string]any{},
			}},
		}, nil
	}

	for _, msg := range messages {
		if msg.Role == "user" && msg.Content == "what about this instead?" {
			return &providers.LLMResponse{Content: "Handled the queued steering message."}, nil
		}
	}

	return nil, fmt.Errorf("provider did not receive queued steering message")
}

func (m *handledMediaWithSteeringProvider) GetDefaultModel() string {
	return "handled-media-with-steering-model"
}

type handledMediaWithSteeringTool struct {
	store media.MediaStore
	path  string
	loop  *AgentLoop
}

func (m *handledMediaWithSteeringTool) Name() string { return "handled_media_with_steering_tool" }
func (m *handledMediaWithSteeringTool) Description() string {
	return "Returns handled media and enqueues a steering message during execution"
}

func (m *handledMediaWithSteeringTool) Parameters() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

func (m *handledMediaWithSteeringTool) Execute(ctx context.Context, args map[string]any) *tools.ToolResult {
	if err := m.loop.Steer(providers.Message{Role: "user", Content: "what about this instead?"}); err != nil {
		return tools.ErrorResult(err.Error()).WithError(err)
	}

	ref, err := m.store.Store(m.path, media.MediaMeta{
		Filename:    filepath.Base(m.path),
		ContentType: "image/png",
		Source:      "test:handled_media_with_steering_tool",
	}, "test:handled_media_with_steering")
	if err != nil {
		return tools.ErrorResult(err.Error()).WithError(err)
	}
	return tools.MediaResult("Attachment delivered by tool.", []string{ref}).WithResponseHandled()
}

type mediaArtifactTool struct {
	store media.MediaStore
	path  string
}

func (m *mediaArtifactTool) Name() string { return "media_artifact_tool" }
func (m *mediaArtifactTool) Description() string {
	return "Returns a media artifact that the agent can forward or save later"
}

func (m *mediaArtifactTool) Parameters() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

func (m *mediaArtifactTool) Execute(ctx context.Context, args map[string]any) *tools.ToolResult {
	ref, err := m.store.Store(m.path, media.MediaMeta{
		Filename:    filepath.Base(m.path),
		ContentType: "image/png",
		Source:      "test:media_artifact_tool",
	}, "test:media_artifact")
	if err != nil {
		return tools.ErrorResult(err.Error()).WithError(err)
	}
	return tools.MediaResult("Artifact created.", []string{ref})
}

type toolLimitTestTool struct{}

func (m *toolLimitTestTool) Name() string {
	return "tool_limit_test_tool"
}

func (m *toolLimitTestTool) Description() string {
	return "Tool used to exhaust the iteration budget in tests"
}

func (m *toolLimitTestTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"value": map[string]any{"type": "string"},
		},
	}
}

func (m *toolLimitTestTool) Execute(ctx context.Context, args map[string]any) *tools.ToolResult {
	return tools.SilentResult("tool limit test result")
}

// testHelper executes a message and returns the response
type testHelper struct {
	al *AgentLoop
}

func newChatCompletionTestServer(
	t *testing.T,
	label string,
	response string,
	calls *int,
	model *string,
) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("%s server path = %q, want /chat/completions", label, r.URL.Path)
		}
		*calls = *calls + 1
		defer r.Body.Close()

		var req struct {
			Model string `json:"model"`
		}
		decodeErr := json.NewDecoder(r.Body).Decode(&req)
		if decodeErr != nil {
			t.Fatalf("decode %s request: %v", label, decodeErr)
		}
		*model = req.Model

		w.Header().Set("Content-Type", "application/json")
		encodeErr := json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{
					"message":       map[string]any{"content": response},
					"finish_reason": "stop",
				},
			},
		})
		if encodeErr != nil {
			t.Fatalf("encode %s response: %v", label, encodeErr)
		}
	}))
}

func newStrictChatCompletionTestServer(
	t *testing.T,
	label string,
	expectedModel string,
	response string,
	calls *int,
) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("%s server path = %q, want /chat/completions", label, r.URL.Path)
		}
		*calls = *calls + 1
		defer r.Body.Close()

		var req struct {
			Model string `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode %s request: %v", label, err)
		}
		if req.Model != expectedModel {
			t.Fatalf("%s server model = %q, want %q", label, req.Model, expectedModel)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{
					"message":       map[string]any{"content": response},
					"finish_reason": "stop",
				},
			},
		}); err != nil {
			t.Fatalf("encode %s response: %v", label, err)
		}
	}))
}

func (h testHelper) executeAndGetResponse(tb testing.TB, ctx context.Context, msg bus.InboundMessage) string {
	// Use a short timeout to avoid hanging
	timeoutCtx, cancel := context.WithTimeout(ctx, responseTimeout)
	defer cancel()

	response, err := h.al.processMessage(timeoutCtx, msg)
	if err != nil {
		tb.Fatalf("processMessage failed: %v", err)
	}
	return response
}

const responseTimeout = 3 * time.Second

func TestProcessMessage_UsesRouteSessionKey(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &simpleMockProvider{response: "ok"}
	al := NewAgentLoop(cfg, msgBus, provider)

	msg := bus.InboundMessage{
		Channel:  "telegram",
		SenderID: "user1",
		ChatID:   "chat1",
		Content:  "hello",
		Peer: bus.Peer{
			Kind: "direct",
			ID:   "user1",
		},
	}

	route := al.registry.ResolveRoute(routing.RouteInput{
		Channel: msg.Channel,
		Peer:    extractPeer(msg),
	})
	sessionKey := route.SessionKey

	defaultAgent := al.registry.GetDefaultAgent()
	if defaultAgent == nil {
		t.Fatal("No default agent found")
	}

	helper := testHelper{al: al}
	_ = helper.executeAndGetResponse(t, context.Background(), msg)

	history := defaultAgent.Sessions.GetHistory(sessionKey)
	if len(history) != 2 {
		t.Fatalf("expected session history len=2, got %d", len(history))
	}
	if history[0].Role != "user" || history[0].Content != "hello" {
		t.Fatalf("unexpected first message in session: %+v", history[0])
	}
}

func TestProcessMessage_CommandOutcomes(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
		Session: config.SessionConfig{
			DMScope: "per-channel-peer",
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &countingMockProvider{response: "LLM reply"}
	al := NewAgentLoop(cfg, msgBus, provider)
	helper := testHelper{al: al}

	baseMsg := bus.InboundMessage{
		Channel:  "whatsapp",
		SenderID: "user1",
		ChatID:   "chat1",
		Peer: bus.Peer{
			Kind: "direct",
			ID:   "user1",
		},
	}

	showResp := helper.executeAndGetResponse(t, context.Background(), bus.InboundMessage{
		Channel:  baseMsg.Channel,
		SenderID: baseMsg.SenderID,
		ChatID:   baseMsg.ChatID,
		Content:  "/show channel",
		Peer:     baseMsg.Peer,
	})
	if showResp != "Current Channel: whatsapp" {
		t.Fatalf("unexpected /show reply: %q", showResp)
	}
	if provider.calls != 0 {
		t.Fatalf("LLM should not be called for handled command, calls=%d", provider.calls)
	}

	fooResp := helper.executeAndGetResponse(t, context.Background(), bus.InboundMessage{
		Channel:  baseMsg.Channel,
		SenderID: baseMsg.SenderID,
		ChatID:   baseMsg.ChatID,
		Content:  "/foo",
		Peer:     baseMsg.Peer,
	})
	if fooResp != "LLM reply" {
		t.Fatalf("unexpected /foo reply: %q", fooResp)
	}
	if provider.calls != 1 {
		t.Fatalf("LLM should be called exactly once after /foo passthrough, calls=%d", provider.calls)
	}

	newResp := helper.executeAndGetResponse(t, context.Background(), bus.InboundMessage{
		Channel:  baseMsg.Channel,
		SenderID: baseMsg.SenderID,
		ChatID:   baseMsg.ChatID,
		Content:  "/new",
		Peer:     baseMsg.Peer,
	})
	if newResp != "LLM reply" {
		t.Fatalf("unexpected /new reply: %q", newResp)
	}
	if provider.calls != 2 {
		t.Fatalf("LLM should be called for passthrough /new command, calls=%d", provider.calls)
	}
}

func TestProcessMessage_SwitchModelShowModelConsistency(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				Provider:          "openai",
				ModelName:         "local",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
		ModelList: []*config.ModelConfig{
			{
				ModelName: "local",
				Model:     "openai/local-model",
				APIBase:   "https://local.example.invalid/v1",
				APIKeys:   config.SimpleSecureStrings("test-key"),
			},
			{
				ModelName: "deepseek",
				Model:     "openrouter/deepseek/deepseek-v3.2",
				APIBase:   "https://openrouter.ai/api/v1",
				APIKeys:   config.SimpleSecureStrings("test-key"),
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &countingMockProvider{response: "LLM reply"}
	al := NewAgentLoop(cfg, msgBus, provider)
	helper := testHelper{al: al}

	switchResp := helper.executeAndGetResponse(t, context.Background(), bus.InboundMessage{
		Channel:  "telegram",
		SenderID: "user1",
		ChatID:   "chat1",
		Content:  "/switch model to deepseek",
		Peer: bus.Peer{
			Kind: "direct",
			ID:   "user1",
		},
	})
	if !strings.Contains(switchResp, "Switched model from local to deepseek") {
		t.Fatalf("unexpected /switch reply: %q", switchResp)
	}

	showResp := helper.executeAndGetResponse(t, context.Background(), bus.InboundMessage{
		Channel:  "telegram",
		SenderID: "user1",
		ChatID:   "chat1",
		Content:  "/show model",
		Peer: bus.Peer{
			Kind: "direct",
			ID:   "user1",
		},
	})
	if !strings.Contains(showResp, "Current Model: deepseek (Provider: openrouter)") {
		t.Fatalf("unexpected /show model reply after switch: %q", showResp)
	}

	if provider.calls != 0 {
		t.Fatalf("LLM should not be called for /switch and /show, calls=%d", provider.calls)
	}
}

func TestProcessMessage_SwitchModelRejectsUnknownAlias(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				Provider:          "openai",
				ModelName:         "local",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
		ModelList: []*config.ModelConfig{
			{
				ModelName: "local",
				Model:     "openai/local-model",
				APIBase:   "https://local.example.invalid/v1",
				APIKeys:   config.SimpleSecureStrings("test-key"),
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &countingMockProvider{response: "LLM reply"}
	al := NewAgentLoop(cfg, msgBus, provider)
	helper := testHelper{al: al}

	switchResp := helper.executeAndGetResponse(t, context.Background(), bus.InboundMessage{
		Channel:  "telegram",
		SenderID: "user1",
		ChatID:   "chat1",
		Content:  "/switch model to missing",
		Peer: bus.Peer{
			Kind: "direct",
			ID:   "user1",
		},
	})
	if switchResp != `model "missing" not found in model_list or providers` {
		t.Fatalf("unexpected /switch error reply: %q", switchResp)
	}

	showResp := helper.executeAndGetResponse(t, context.Background(), bus.InboundMessage{
		Channel:  "telegram",
		SenderID: "user1",
		ChatID:   "chat1",
		Content:  "/show model",
		Peer: bus.Peer{
			Kind: "direct",
			ID:   "user1",
		},
	})
	if !strings.Contains(showResp, "Current Model: local (Provider: openai)") {
		t.Fatalf("unexpected /show model reply after rejected switch: %q", showResp)
	}

	if provider.calls != 0 {
		t.Fatalf("LLM should not be called for rejected /switch and /show, calls=%d", provider.calls)
	}
}

func TestProcessMessage_SwitchModelRoutesSubsequentRequestsToSelectedProvider(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	localCalls := 0
	localModel := ""
	localServer := newChatCompletionTestServer(t, "local", "local reply", &localCalls, &localModel)
	defer localServer.Close()

	remoteCalls := 0
	remoteModel := ""
	remoteServer := newChatCompletionTestServer(t, "remote", "remote reply", &remoteCalls, &remoteModel)
	defer remoteServer.Close()

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				Provider:          "openai",
				ModelName:         "local",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
		ModelList: []*config.ModelConfig{
			{
				ModelName: "local",
				Model:     "openai/Qwen3.5-35B-A3B",
				APIBase:   localServer.URL,
				APIKeys:   config.SimpleSecureStrings("local-key"),
			},
			{
				ModelName: "deepseek",
				Model:     "openrouter/deepseek/deepseek-v3.2",
				APIBase:   remoteServer.URL,
				APIKeys:   config.SimpleSecureStrings("remote-key"),
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider, _, err := providers.CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider() error = %v", err)
	}
	al := NewAgentLoop(cfg, msgBus, provider)
	helper := testHelper{al: al}

	firstResp := helper.executeAndGetResponse(t, context.Background(), bus.InboundMessage{
		Channel:  "telegram",
		SenderID: "user1",
		ChatID:   "chat1",
		Content:  "hello before switch",
		Peer: bus.Peer{
			Kind: "direct",
			ID:   "user1",
		},
	})
	if firstResp != "local reply" {
		t.Fatalf("unexpected response before switch: %q", firstResp)
	}
	if localCalls != 1 {
		t.Fatalf("local calls before switch = %d, want 1", localCalls)
	}
	if remoteCalls != 0 {
		t.Fatalf("remote calls before switch = %d, want 0", remoteCalls)
	}
	if localModel != "Qwen3.5-35B-A3B" {
		t.Fatalf("local model before switch = %q, want %q", localModel, "Qwen3.5-35B-A3B")
	}

	switchResp := helper.executeAndGetResponse(t, context.Background(), bus.InboundMessage{
		Channel:  "telegram",
		SenderID: "user1",
		ChatID:   "chat1",
		Content:  "/switch model to deepseek",
		Peer: bus.Peer{
			Kind: "direct",
			ID:   "user1",
		},
	})
	if !strings.Contains(switchResp, "Switched model from local to deepseek") {
		t.Fatalf("unexpected /switch reply: %q", switchResp)
	}

	secondResp := helper.executeAndGetResponse(t, context.Background(), bus.InboundMessage{
		Channel:  "telegram",
		SenderID: "user1",
		ChatID:   "chat1",
		Content:  "hello after switch",
		Peer: bus.Peer{
			Kind: "direct",
			ID:   "user1",
		},
	})
	if secondResp != "remote reply" {
		t.Fatalf("unexpected response after switch: %q", secondResp)
	}
	if localCalls != 1 {
		t.Fatalf("local calls after switch = %d, want 1", localCalls)
	}
	if remoteCalls != 1 {
		t.Fatalf("remote calls after switch = %d, want 1", remoteCalls)
	}
	if remoteModel != "deepseek-v3.2" {
		t.Fatalf(
			"remote model after switch = %q, want %q",
			remoteModel,
			"deepseek-v3.2",
		)
	}
}

func TestProcessMessage_ModelRoutingUsesLightProvider(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	heavyCalls := 0
	heavyServer := newStrictChatCompletionTestServer(
		t,
		"heavy",
		"gemini-2.5-flash",
		"heavy reply",
		&heavyCalls,
	)
	defer heavyServer.Close()

	lightCalls := 0
	lightServer := newStrictChatCompletionTestServer(
		t,
		"light",
		"qwen2.5:0.5b",
		"light reply",
		&lightCalls,
	)
	defer lightServer.Close()

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				ModelName:         "gemini-main",
				MaxTokens:         4096,
				MaxToolIterations: 10,
				Routing: &config.RoutingConfig{
					Enabled:    true,
					LightModel: "qwen-light",
					Threshold:  0.99,
				},
			},
		},
		ModelList: []*config.ModelConfig{
			{
				ModelName: "gemini-main",
				Model:     "gemini/gemini-2.5-flash",
				APIBase:   heavyServer.URL,
			},
			{
				ModelName: "qwen-light",
				Model:     "ollama/qwen2.5:0.5b",
				APIBase:   lightServer.URL,
			},
		},
	}
	cfg.WithSecurity(&config.SecurityConfig{
		ModelList: map[string]config.ModelSecurityEntry{
			"gemini-main": {APIKeys: []string{"heavy-key"}},
			"qwen-light":  {APIKeys: []string{"light-key"}},
		},
	})

	msgBus := bus.NewMessageBus()
	provider, _, err := providers.CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider() error = %v", err)
	}
	al := NewAgentLoop(cfg, msgBus, provider)
	helper := testHelper{al: al}

	resp := helper.executeAndGetResponse(t, context.Background(), bus.InboundMessage{
		Channel:  "telegram",
		SenderID: "user1",
		ChatID:   "chat1",
		Content:  "hi",
		Peer: bus.Peer{
			Kind: "direct",
			ID:   "user1",
		},
	})
	if resp != "light reply" {
		t.Fatalf("response = %q, want %q", resp, "light reply")
	}
	if heavyCalls != 0 {
		t.Fatalf("heavy calls = %d, want 0", heavyCalls)
	}
	if lightCalls != 1 {
		t.Fatalf("light calls = %d, want 1", lightCalls)
	}
}

// TestToolResult_SilentToolDoesNotSendUserMessage verifies silent tools don't trigger outbound
func TestToolResult_SilentToolDoesNotSendUserMessage(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &simpleMockProvider{response: "File operation complete"}
	al := NewAgentLoop(cfg, msgBus, provider)
	helper := testHelper{al: al}

	// ReadFileTool returns SilentResult, which should not send user message
	ctx := context.Background()
	msg := bus.InboundMessage{
		Channel:    "test",
		SenderID:   "user1",
		ChatID:     "chat1",
		Content:    "read test.txt",
		SessionKey: "test-session",
	}

	response := helper.executeAndGetResponse(t, ctx, msg)

	// Silent tool should return the LLM's response directly
	if response != "File operation complete" {
		t.Errorf("Expected 'File operation complete', got: %s", response)
	}
}

// TestToolResult_UserFacingToolDoesSendMessage verifies user-facing tools trigger outbound
func TestToolResult_UserFacingToolDoesSendMessage(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &simpleMockProvider{response: "Command output: hello world"}
	al := NewAgentLoop(cfg, msgBus, provider)
	helper := testHelper{al: al}

	// ExecTool returns UserResult, which should send user message
	ctx := context.Background()
	msg := bus.InboundMessage{
		Channel:    "test",
		SenderID:   "user1",
		ChatID:     "chat1",
		Content:    "run hello",
		SessionKey: "test-session",
	}

	response := helper.executeAndGetResponse(t, ctx, msg)

	// User-facing tool should include the output in final response
	if response != "Command output: hello world" {
		t.Errorf("Expected 'Command output: hello world', got: %s", response)
	}
}

// failFirstMockProvider fails on the first N calls with a specific error
type failFirstMockProvider struct {
	failures    int
	currentCall int
	failError   error
	successResp string
}

func (m *failFirstMockProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	opts map[string]any,
) (*providers.LLMResponse, error) {
	m.currentCall++
	if m.currentCall <= m.failures {
		return nil, m.failError
	}
	return &providers.LLMResponse{
		Content:   m.successResp,
		ToolCalls: []providers.ToolCall{},
	}, nil
}

func (m *failFirstMockProvider) GetDefaultModel() string {
	return "mock-fail-model"
}

// TestAgentLoop_ContextExhaustionRetry verify that the agent retries on context errors
func TestAgentLoop_ContextExhaustionRetry(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}

	msgBus := bus.NewMessageBus()

	// Create a provider that fails once with a context error
	contextErr := fmt.Errorf("InvalidParameter: Total tokens of image and text exceed max message tokens")
	provider := &failFirstMockProvider{
		failures:    1,
		failError:   contextErr,
		successResp: "Recovered from context error",
	}

	al := NewAgentLoop(cfg, msgBus, provider)

	// Inject some history to simulate a full context.
	// Session history only stores user/assistant/tool messages — the system
	// prompt is built dynamically by BuildMessages and is NOT stored here.
	sessionKey := "test-session-context"
	history := []providers.Message{
		{Role: "user", Content: "Old message 1"},
		{Role: "assistant", Content: "Old response 1"},
		{Role: "user", Content: "Old message 2"},
		{Role: "assistant", Content: "Old response 2"},
		{Role: "user", Content: "Trigger message"},
	}
	defaultAgent := al.registry.GetDefaultAgent()
	if defaultAgent == nil {
		t.Fatal("No default agent found")
	}
	defaultAgent.Sessions.SetHistory(sessionKey, history)

	// Call ProcessDirectWithChannel
	// Note: ProcessDirectWithChannel calls processMessage which will execute runLLMIteration
	response, err := al.ProcessDirectWithChannel(
		context.Background(),
		"Trigger message",
		sessionKey,
		"test",
		"test-chat",
	)
	if err != nil {
		t.Fatalf("Expected success after retry, got error: %v", err)
	}

	if response != "Recovered from context error" {
		t.Errorf("Expected 'Recovered from context error', got '%s'", response)
	}

	// We expect 2 calls: 1st failed, 2nd succeeded
	if provider.currentCall != 2 {
		t.Errorf("Expected 2 calls (1 fail + 1 success), got %d", provider.currentCall)
	}

	// Check final history length
	finalHistory := defaultAgent.Sessions.GetHistory(sessionKey)
	// We verify that the history has been modified (compressed)
	// Original length: 5
	// Expected behavior: compression drops ~50% of Turns
	// Without compression: 5 + 1 (new user msg) + 1 (assistant msg) = 7
	if len(finalHistory) >= 7 {
		t.Errorf("Expected history to be compressed (len < 7), got %d", len(finalHistory))
	}
}

func TestAgentLoop_EmptyModelResponseUsesAccurateFallback(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 3,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &simpleMockProvider{response: ""}
	al := NewAgentLoop(cfg, msgBus, provider)

	response, err := al.ProcessDirectWithChannel(context.Background(), "hello", "empty-response", "test", "chat1")
	if err != nil {
		t.Fatalf("ProcessDirectWithChannel failed: %v", err)
	}
	if response != defaultResponse {
		t.Fatalf("response = %q, want %q", response, defaultResponse)
	}
}

func TestAgentLoop_ToolLimitUsesDedicatedFallback(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 1,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &toolLimitOnlyProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)
	al.RegisterTool(&toolLimitTestTool{})

	response, err := al.ProcessDirectWithChannel(context.Background(), "hello", "tool-limit", "test", "chat1")
	if err != nil {
		t.Fatalf("ProcessDirectWithChannel failed: %v", err)
	}
	if response != toolLimitResponse {
		t.Fatalf("response = %q, want %q", response, toolLimitResponse)
	}

	defaultAgent := al.registry.GetDefaultAgent()
	if defaultAgent == nil {
		t.Fatal("No default agent found")
	}
	route := al.registry.ResolveRoute(routing.RouteInput{
		Channel: "test",
		Peer: &routing.RoutePeer{
			Kind: "direct",
			ID:   "cron",
		},
	})
	history := defaultAgent.Sessions.GetHistory(route.SessionKey)
	if len(history) != 4 {
		t.Fatalf("history len = %d, want 4", len(history))
	}
	assertRoles(t, history, "user", "assistant", "tool", "assistant")
	if history[3].Content != toolLimitResponse {
		t.Fatalf("final assistant content = %q, want %q", history[3].Content, toolLimitResponse)
	}
}

// TestProcessDirectWithChannel_TriggersMCPInitialization verifies that
// ProcessDirectWithChannel triggers MCP initialization when MCP is enabled.
// Note: Manager is only initialized when at least one MCP server is configured
// and successfully connected.
func TestProcessDirectWithChannel_TriggersMCPInitialization(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test with MCP enabled but no servers - should not initialize manager
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
		Tools: config.ToolsConfig{
			MCP: config.MCPConfig{
				ToolConfig: config.ToolConfig{
					Enabled: true,
				},
				// No servers configured - manager should not be initialized
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &mockProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)
	defer al.Close()

	if al.mcp.hasManager() {
		t.Fatal("expected MCP manager to be nil before first direct processing")
	}

	_, err = al.ProcessDirectWithChannel(
		context.Background(),
		"hello",
		"session-1",
		"cli",
		"direct",
	)
	if err != nil {
		t.Fatalf("ProcessDirectWithChannel failed: %v", err)
	}

	// Manager should not be initialized when no servers are configured
	if al.mcp.hasManager() {
		t.Fatal("expected MCP manager to be nil when no servers are configured")
	}
}

func TestTargetReasoningChannelID_AllChannels(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}

	al := NewAgentLoop(cfg, bus.NewMessageBus(), &mockProvider{})
	chManager, err := channels.NewManager(&config.Config{}, bus.NewMessageBus(), nil)
	if err != nil {
		t.Fatalf("Failed to create channel manager: %v", err)
	}
	for name, id := range map[string]string{
		"whatsapp": "rid-whatsapp",
		"telegram": "rid-telegram",
		"feishu":   "rid-feishu",
		"discord":  "rid-discord",
		"maixcam":  "rid-maixcam",
		"qq":       "rid-qq",
		"dingtalk": "rid-dingtalk",
		"slack":    "rid-slack",
		"line":     "rid-line",
		"onebot":   "rid-onebot",
		"wecom":    "rid-wecom",
	} {
		chManager.RegisterChannel(name, &fakeChannel{id: id})
	}
	al.SetChannelManager(chManager)
	tests := []struct {
		channel string
		wantID  string
	}{
		{channel: "whatsapp", wantID: "rid-whatsapp"},
		{channel: "telegram", wantID: "rid-telegram"},
		{channel: "feishu", wantID: "rid-feishu"},
		{channel: "discord", wantID: "rid-discord"},
		{channel: "maixcam", wantID: "rid-maixcam"},
		{channel: "qq", wantID: "rid-qq"},
		{channel: "dingtalk", wantID: "rid-dingtalk"},
		{channel: "slack", wantID: "rid-slack"},
		{channel: "line", wantID: "rid-line"},
		{channel: "onebot", wantID: "rid-onebot"},
		{channel: "wecom", wantID: "rid-wecom"},
		{channel: "unknown", wantID: ""},
	}

	for _, tt := range tests {
		t.Run(tt.channel, func(t *testing.T) {
			got := al.targetReasoningChannelID(tt.channel)
			if got != tt.wantID {
				t.Fatalf("targetReasoningChannelID(%q) = %q, want %q", tt.channel, got, tt.wantID)
			}
		})
	}
}

func TestHandleReasoning(t *testing.T) {
	newLoop := func(t *testing.T) (*AgentLoop, *bus.MessageBus) {
		t.Helper()
		tmpDir, err := os.MkdirTemp("", "agent-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })
		cfg := &config.Config{
			Agents: config.AgentsConfig{
				Defaults: config.AgentDefaults{
					Workspace:         tmpDir,
					ModelName:         "test-model",
					MaxTokens:         4096,
					MaxToolIterations: 10,
				},
			},
		}
		msgBus := bus.NewMessageBus()
		return NewAgentLoop(cfg, msgBus, &mockProvider{}), msgBus
	}

	t.Run("skips when any required field is empty", func(t *testing.T) {
		al, msgBus := newLoop(t)
		al.handleReasoning(context.Background(), "reasoning", "telegram", "")

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		for {
			select {
			case msg, ok := <-msgBus.OutboundChan():
				if !ok {
					t.Fatalf("expected no outbound message, got %+v", msg)
				}
				if msg.Content == "reasoning" {
					t.Fatalf("expected no message for empty chatID, got %+v", msg)
				}
				return
			case <-ctx.Done():
				t.Log("expected an outbound message, got none within timeout")
				return
			default:
				// Continue to check for message
				time.Sleep(5 * time.Millisecond) // Avoid busy loop
			}
		}
	})

	t.Run("publishes one message for non telegram", func(t *testing.T) {
		al, msgBus := newLoop(t)
		al.handleReasoning(context.Background(), "hello reasoning", "slack", "channel-1")

		msg, ok := <-msgBus.OutboundChan()
		if !ok {
			t.Fatal("expected an outbound message")
		}
		if msg.Channel != "slack" || msg.ChatID != "channel-1" || msg.Content != "hello reasoning" {
			t.Fatalf("unexpected outbound message: %+v", msg)
		}
	})

	t.Run("publishes one message for telegram", func(t *testing.T) {
		al, msgBus := newLoop(t)
		reasoning := "hello telegram reasoning"
		al.handleReasoning(context.Background(), reasoning, "telegram", "tg-chat")

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		for {
			select {
			case <-ctx.Done():
				t.Fatal("expected an outbound message, got none within timeout")
				return
			case msg, ok := <-msgBus.OutboundChan():
				if !ok {
					t.Fatal("expected outbound message")
				}

				if msg.Channel != "telegram" {
					t.Fatalf("expected telegram channel message, got %+v", msg)
				}
				if msg.ChatID != "tg-chat" {
					t.Fatalf("expected chatID tg-chat, got %+v", msg)
				}
				if msg.Content != reasoning {
					t.Fatalf("content mismatch: got %q want %q", msg.Content, reasoning)
				}
				return
			}
		}
	})
	t.Run("expired ctx", func(t *testing.T) {
		al, msgBus := newLoop(t)
		reasoning := "hello telegram reasoning"

		al.handleReasoning(context.Background(), reasoning, "telegram", "tg-chat")

		consumeCtx, consumeCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer consumeCancel()

		for {
			select {
			case msg, ok := <-msgBus.OutboundChan():
				if !ok {
					t.Fatalf("expected no outbound message, but received: %+v", msg)
				}
				t.Logf("Received unexpected outbound message: %+v", msg)
				return
			case <-consumeCtx.Done():
				t.Fatalf("failed: no message received within timeout")
				return
			}
		}
	})

	t.Run("returns promptly when bus is full", func(t *testing.T) {
		al, msgBus := newLoop(t)

		// Fill the outbound bus buffer until a publish would block.
		// Use a short timeout to detect when the buffer is full,
		// rather than hardcoding the buffer size.
		for i := 0; ; i++ {
			fillCtx, fillCancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			err := msgBus.PublishOutbound(fillCtx, bus.OutboundMessage{
				Channel: "filler",
				ChatID:  "filler",
				Content: fmt.Sprintf("filler-%d", i),
			})
			fillCancel()
			if err != nil {
				// Buffer is full (timed out trying to send).
				break
			}
		}

		// Use a short-deadline parent context to bound the test.
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		start := time.Now()
		al.handleReasoning(ctx, "should timeout", "slack", "channel-full")
		elapsed := time.Since(start)

		// handleReasoning uses a 5s internal timeout, but the parent ctx
		// expires in 500ms. It should return within ~500ms, not 5s.
		if elapsed > 2*time.Second {
			t.Fatalf("handleReasoning blocked too long (%v); expected prompt return", elapsed)
		}

		// Drain the bus and verify the reasoning message was NOT published
		// (it should have been dropped due to timeout).
		timeer := time.After(1 * time.Second)
		for {
			select {
			case <-timeer:
				t.Logf(
					"no reasoning message received after draining bus for 1s, as expected,length=%d",
					len(msgBus.OutboundChan()),
				)
				return
			case msg, ok := <-msgBus.OutboundChan():
				if !ok {
					break
				}
				if msg.Content == "should timeout" {
					t.Fatal("expected reasoning message to be dropped when bus is full, but it was published")
				}
			}
		}
	})
}

func TestProcessMessage_PublishesReasoningContentToReasoningChannel(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &reasoningContentProvider{
		response:         "final answer",
		reasoningContent: "thinking trace",
	}
	al := NewAgentLoop(cfg, msgBus, provider)

	chManager, err := channels.NewManager(&config.Config{}, msgBus, nil)
	if err != nil {
		t.Fatalf("Failed to create channel manager: %v", err)
	}
	chManager.RegisterChannel("telegram", &fakeChannel{id: "reason-chat"})
	al.SetChannelManager(chManager)

	response, err := al.processMessage(context.Background(), bus.InboundMessage{
		Channel:  "telegram",
		SenderID: "user1",
		ChatID:   "chat1",
		Content:  "hello",
	})
	if err != nil {
		t.Fatalf("processMessage() error = %v", err)
	}
	if response != "final answer" {
		t.Fatalf("processMessage() response = %q, want %q", response, "final answer")
	}

	select {
	case outbound := <-msgBus.OutboundChan():
		if outbound.Channel != "telegram" {
			t.Fatalf("reasoning channel = %q, want %q", outbound.Channel, "telegram")
		}
		if outbound.ChatID != "reason-chat" {
			t.Fatalf("reasoning chatID = %q, want %q", outbound.ChatID, "reason-chat")
		}
		if outbound.Content != "thinking trace" {
			t.Fatalf("reasoning content = %q, want %q", outbound.Content, "thinking trace")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("expected reasoning content to be published to reasoning channel")
	}
}

func TestProcessHeartbeat_DoesNotPublishToolFeedback(t *testing.T) {
	tmpDir := t.TempDir()
	heartbeatFile := filepath.Join(tmpDir, "heartbeat-task.txt")
	if err := os.WriteFile(heartbeatFile, []byte("heartbeat task"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
				ToolFeedback: config.ToolFeedbackConfig{
					Enabled:       true,
					MaxArgsLength: 300,
				},
			},
		},
		Tools: config.ToolsConfig{
			ReadFile: config.ReadFileToolConfig{
				Enabled: true,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &toolFeedbackProvider{filePath: heartbeatFile}
	al := NewAgentLoop(cfg, msgBus, provider)

	response, err := al.ProcessHeartbeat(context.Background(), "check heartbeat tasks", "telegram", "chat-1")
	if err != nil {
		t.Fatalf("ProcessHeartbeat() error = %v", err)
	}
	if response != "HEARTBEAT_OK" {
		t.Fatalf("ProcessHeartbeat() response = %q, want %q", response, "HEARTBEAT_OK")
	}

	select {
	case outbound := <-msgBus.OutboundChan():
		t.Fatalf("expected no outbound tool feedback during heartbeat, got %+v", outbound)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestProcessMessage_PublishesToolFeedbackWhenEnabled(t *testing.T) {
	tmpDir := t.TempDir()
	heartbeatFile := filepath.Join(tmpDir, "tool-feedback.txt")
	if err := os.WriteFile(heartbeatFile, []byte("tool feedback task"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
				ToolFeedback: config.ToolFeedbackConfig{
					Enabled:       true,
					MaxArgsLength: 300,
				},
			},
		},
		Tools: config.ToolsConfig{
			ReadFile: config.ReadFileToolConfig{
				Enabled: true,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &toolFeedbackProvider{filePath: heartbeatFile}
	al := NewAgentLoop(cfg, msgBus, provider)

	response, err := al.processMessage(context.Background(), bus.InboundMessage{
		Channel:  "telegram",
		SenderID: "user-1",
		ChatID:   "chat-1",
		Content:  "check tool feedback",
	})
	if err != nil {
		t.Fatalf("processMessage() error = %v", err)
	}
	if response != "HEARTBEAT_OK" {
		t.Fatalf("processMessage() response = %q, want %q", response, "HEARTBEAT_OK")
	}

	select {
	case outbound := <-msgBus.OutboundChan():
		if outbound.Channel != "telegram" {
			t.Fatalf("tool feedback channel = %q, want %q", outbound.Channel, "telegram")
		}
		if outbound.ChatID != "chat-1" {
			t.Fatalf("tool feedback chatID = %q, want %q", outbound.ChatID, "chat-1")
		}
		if !strings.Contains(outbound.Content, "`read_file`") {
			t.Fatalf("tool feedback content = %q, want read_file preview", outbound.Content)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected outbound tool feedback for regular messages")
	}
}

func TestResolveMediaRefs_ResolvesToBase64(t *testing.T) {
	store := media.NewFileMediaStore()
	dir := t.TempDir()

	// Create a minimal valid PNG (8-byte header is enough for filetype detection)
	pngPath := filepath.Join(dir, "test.png")
	// PNG magic: 0x89 P N G \r \n 0x1A \n + minimal IHDR
	pngHeader := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
		0x00, 0x00, 0x00, 0x0D, // IHDR length
		0x49, 0x48, 0x44, 0x52, // "IHDR"
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x02, // 1x1 RGB
		0x00, 0x00, 0x00, // no interlace
		0x90, 0x77, 0x53, 0xDE, // CRC
	}
	if err := os.WriteFile(pngPath, pngHeader, 0o644); err != nil {
		t.Fatal(err)
	}
	ref, err := store.Store(pngPath, media.MediaMeta{}, "test")
	if err != nil {
		t.Fatal(err)
	}

	messages := []providers.Message{
		{Role: "user", Content: "describe this", Media: []string{ref}},
	}
	result := resolveMediaRefs(messages, store, config.DefaultMaxMediaSize)

	if len(result[0].Media) != 1 {
		t.Fatalf("expected 1 resolved media, got %d", len(result[0].Media))
	}
	if !strings.HasPrefix(result[0].Media[0], "data:image/png;base64,") {
		t.Fatalf("expected data:image/png;base64, prefix, got %q", result[0].Media[0][:40])
	}
}

func TestResolveMediaRefs_SkipsOversizedFile(t *testing.T) {
	store := media.NewFileMediaStore()
	dir := t.TempDir()

	bigPath := filepath.Join(dir, "big.png")
	// Write PNG header + padding to exceed limit
	data := make([]byte, 1024+1) // 1KB + 1 byte
	copy(data, []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
	if err := os.WriteFile(bigPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	ref, _ := store.Store(bigPath, media.MediaMeta{}, "test")

	messages := []providers.Message{
		{Role: "user", Content: "hi", Media: []string{ref}},
	}
	// Use a tiny limit (1KB) so the file is oversized
	result := resolveMediaRefs(messages, store, 1024)

	if len(result[0].Media) != 0 {
		t.Fatalf("expected 0 media (oversized), got %d", len(result[0].Media))
	}
}

func TestResolveMediaRefs_UnknownTypeInjectsPath(t *testing.T) {
	store := media.NewFileMediaStore()
	dir := t.TempDir()

	txtPath := filepath.Join(dir, "readme.txt")
	if err := os.WriteFile(txtPath, []byte("hello world"), 0o644); err != nil {
		t.Fatal(err)
	}
	ref, _ := store.Store(txtPath, media.MediaMeta{}, "test")

	messages := []providers.Message{
		{Role: "user", Content: "hi", Media: []string{ref}},
	}
	result := resolveMediaRefs(messages, store, config.DefaultMaxMediaSize)

	if len(result[0].Media) != 0 {
		t.Fatalf("expected 0 media entries, got %d", len(result[0].Media))
	}
	expected := "hi [file:" + txtPath + "]"
	if result[0].Content != expected {
		t.Fatalf("expected content %q, got %q", expected, result[0].Content)
	}
}

func TestResolveMediaRefs_PassesThroughNonMediaRefs(t *testing.T) {
	messages := []providers.Message{
		{Role: "user", Content: "hi", Media: []string{"https://example.com/img.png"}},
	}
	result := resolveMediaRefs(messages, nil, config.DefaultMaxMediaSize)

	if len(result[0].Media) != 1 || result[0].Media[0] != "https://example.com/img.png" {
		t.Fatalf("expected passthrough of non-media:// URL, got %v", result[0].Media)
	}
}

func TestResolveMediaRefs_DoesNotMutateOriginal(t *testing.T) {
	store := media.NewFileMediaStore()
	dir := t.TempDir()
	pngPath := filepath.Join(dir, "test.png")
	pngHeader := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x02,
		0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xDE,
	}
	os.WriteFile(pngPath, pngHeader, 0o644)
	ref, _ := store.Store(pngPath, media.MediaMeta{}, "test")

	original := []providers.Message{
		{Role: "user", Content: "hi", Media: []string{ref}},
	}
	originalRef := original[0].Media[0]

	resolveMediaRefs(original, store, config.DefaultMaxMediaSize)

	if original[0].Media[0] != originalRef {
		t.Fatal("resolveMediaRefs mutated original message slice")
	}
}

func TestResolveMediaRefs_UsesMetaContentType(t *testing.T) {
	store := media.NewFileMediaStore()
	dir := t.TempDir()

	// File with JPEG content but stored with explicit content type
	jpegPath := filepath.Join(dir, "photo")
	jpegHeader := []byte{0xFF, 0xD8, 0xFF, 0xE0} // JPEG magic bytes
	os.WriteFile(jpegPath, jpegHeader, 0o644)
	ref, _ := store.Store(jpegPath, media.MediaMeta{ContentType: "image/jpeg"}, "test")

	messages := []providers.Message{
		{Role: "user", Content: "hi", Media: []string{ref}},
	}
	result := resolveMediaRefs(messages, store, config.DefaultMaxMediaSize)

	if len(result[0].Media) != 1 {
		t.Fatalf("expected 1 media, got %d", len(result[0].Media))
	}
	if !strings.HasPrefix(result[0].Media[0], "data:image/jpeg;base64,") {
		t.Fatalf("expected jpeg prefix, got %q", result[0].Media[0][:30])
	}
}

func TestResolveMediaRefs_PDFInjectsFilePath(t *testing.T) {
	store := media.NewFileMediaStore()
	dir := t.TempDir()

	pdfPath := filepath.Join(dir, "report.pdf")
	// PDF magic bytes
	os.WriteFile(pdfPath, []byte("%PDF-1.4 test content"), 0o644)
	ref, _ := store.Store(pdfPath, media.MediaMeta{ContentType: "application/pdf"}, "test")

	messages := []providers.Message{
		{Role: "user", Content: "report.pdf [file]", Media: []string{ref}},
	}
	result := resolveMediaRefs(messages, store, config.DefaultMaxMediaSize)

	if len(result[0].Media) != 0 {
		t.Fatalf("expected 0 media (non-image), got %d", len(result[0].Media))
	}
	expected := "report.pdf [file:" + pdfPath + "]"
	if result[0].Content != expected {
		t.Fatalf("expected content %q, got %q", expected, result[0].Content)
	}
}

func TestResolveMediaRefs_AudioInjectsAudioPath(t *testing.T) {
	store := media.NewFileMediaStore()
	dir := t.TempDir()

	oggPath := filepath.Join(dir, "voice.ogg")
	os.WriteFile(oggPath, []byte("fake audio"), 0o644)
	ref, _ := store.Store(oggPath, media.MediaMeta{ContentType: "audio/ogg"}, "test")

	messages := []providers.Message{
		{Role: "user", Content: "voice.ogg [audio]", Media: []string{ref}},
	}
	result := resolveMediaRefs(messages, store, config.DefaultMaxMediaSize)

	if len(result[0].Media) != 0 {
		t.Fatalf("expected 0 media, got %d", len(result[0].Media))
	}
	expected := "voice.ogg [audio:" + oggPath + "]"
	if result[0].Content != expected {
		t.Fatalf("expected content %q, got %q", expected, result[0].Content)
	}
}

func TestResolveMediaRefs_VideoInjectsVideoPath(t *testing.T) {
	store := media.NewFileMediaStore()
	dir := t.TempDir()

	mp4Path := filepath.Join(dir, "clip.mp4")
	os.WriteFile(mp4Path, []byte("fake video"), 0o644)
	ref, _ := store.Store(mp4Path, media.MediaMeta{ContentType: "video/mp4"}, "test")

	messages := []providers.Message{
		{Role: "user", Content: "clip.mp4 [video]", Media: []string{ref}},
	}
	result := resolveMediaRefs(messages, store, config.DefaultMaxMediaSize)

	if len(result[0].Media) != 0 {
		t.Fatalf("expected 0 media, got %d", len(result[0].Media))
	}
	expected := "clip.mp4 [video:" + mp4Path + "]"
	if result[0].Content != expected {
		t.Fatalf("expected content %q, got %q", expected, result[0].Content)
	}
}

func TestResolveMediaRefs_NoGenericTagAppendsPath(t *testing.T) {
	store := media.NewFileMediaStore()
	dir := t.TempDir()

	csvPath := filepath.Join(dir, "data.csv")
	os.WriteFile(csvPath, []byte("a,b,c"), 0o644)
	ref, _ := store.Store(csvPath, media.MediaMeta{ContentType: "text/csv"}, "test")

	messages := []providers.Message{
		{Role: "user", Content: "here is my data", Media: []string{ref}},
	}
	result := resolveMediaRefs(messages, store, config.DefaultMaxMediaSize)

	expected := "here is my data [file:" + csvPath + "]"
	if result[0].Content != expected {
		t.Fatalf("expected content %q, got %q", expected, result[0].Content)
	}
}

func TestResolveMediaRefs_EmptyContentGetsPathTag(t *testing.T) {
	store := media.NewFileMediaStore()
	dir := t.TempDir()

	docPath := filepath.Join(dir, "doc.docx")
	os.WriteFile(docPath, []byte("fake docx"), 0o644)
	docxMIME := "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	ref, _ := store.Store(docPath, media.MediaMeta{ContentType: docxMIME}, "test")

	messages := []providers.Message{
		{Role: "user", Content: "", Media: []string{ref}},
	}
	result := resolveMediaRefs(messages, store, config.DefaultMaxMediaSize)

	expected := "[file:" + docPath + "]"
	if result[0].Content != expected {
		t.Fatalf("expected content %q, got %q", expected, result[0].Content)
	}
}

func TestResolveMediaRefs_MixedImageAndFile(t *testing.T) {
	store := media.NewFileMediaStore()
	dir := t.TempDir()

	pngPath := filepath.Join(dir, "photo.png")
	pngHeader := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x02,
		0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xDE,
	}
	os.WriteFile(pngPath, pngHeader, 0o644)
	imgRef, _ := store.Store(pngPath, media.MediaMeta{}, "test")

	pdfPath := filepath.Join(dir, "report.pdf")
	os.WriteFile(pdfPath, []byte("%PDF-1.4 test"), 0o644)
	fileRef, _ := store.Store(pdfPath, media.MediaMeta{ContentType: "application/pdf"}, "test")

	messages := []providers.Message{
		{Role: "user", Content: "check these [file]", Media: []string{imgRef, fileRef}},
	}
	result := resolveMediaRefs(messages, store, config.DefaultMaxMediaSize)

	if len(result[0].Media) != 1 {
		t.Fatalf("expected 1 media (image only), got %d", len(result[0].Media))
	}
	if !strings.HasPrefix(result[0].Media[0], "data:image/png;base64,") {
		t.Fatal("expected image to be base64 encoded")
	}
	expectedContent := "check these [file:" + pdfPath + "]"
	if result[0].Content != expectedContent {
		t.Fatalf("expected content %q, got %q", expectedContent, result[0].Content)
	}
}

// --- Native search helper tests ---

type nativeSearchProvider struct {
	supported bool
}

func (p *nativeSearchProvider) Chat(
	ctx context.Context, msgs []providers.Message, tools []providers.ToolDefinition,
	model string, opts map[string]any,
) (*providers.LLMResponse, error) {
	return &providers.LLMResponse{Content: "ok"}, nil
}

func (p *nativeSearchProvider) GetDefaultModel() string { return "test-model" }

func (p *nativeSearchProvider) SupportsNativeSearch() bool { return p.supported }

type plainProvider struct{}

func (p *plainProvider) Chat(
	ctx context.Context, msgs []providers.Message, tools []providers.ToolDefinition,
	model string, opts map[string]any,
) (*providers.LLMResponse, error) {
	return &providers.LLMResponse{Content: "ok"}, nil
}

func (p *plainProvider) GetDefaultModel() string { return "test-model" }

func TestIsNativeSearchProvider_Supported(t *testing.T) {
	if !isNativeSearchProvider(&nativeSearchProvider{supported: true}) {
		t.Fatal("expected true for provider that supports native search")
	}
}

func TestIsNativeSearchProvider_NotSupported(t *testing.T) {
	if isNativeSearchProvider(&nativeSearchProvider{supported: false}) {
		t.Fatal("expected false for provider that does not support native search")
	}
}

func TestIsNativeSearchProvider_NoInterface(t *testing.T) {
	if isNativeSearchProvider(&plainProvider{}) {
		t.Fatal("expected false for provider that does not implement NativeSearchCapable")
	}
}

func TestFilterClientWebSearch_RemovesWebSearch(t *testing.T) {
	defs := []providers.ToolDefinition{
		{Type: "function", Function: providers.ToolFunctionDefinition{Name: "web_search"}},
		{Type: "function", Function: providers.ToolFunctionDefinition{Name: "read_file"}},
		{Type: "function", Function: providers.ToolFunctionDefinition{Name: "exec"}},
	}
	result := filterClientWebSearch(defs)
	if len(result) != 2 {
		t.Fatalf("len(result) = %d, want 2", len(result))
	}
	for _, td := range result {
		if td.Function.Name == "web_search" {
			t.Fatal("web_search should be filtered out")
		}
	}
}

func TestFilterClientWebSearch_NoWebSearch(t *testing.T) {
	defs := []providers.ToolDefinition{
		{Type: "function", Function: providers.ToolFunctionDefinition{Name: "read_file"}},
		{Type: "function", Function: providers.ToolFunctionDefinition{Name: "exec"}},
	}
	result := filterClientWebSearch(defs)
	if len(result) != 2 {
		t.Fatalf("len(result) = %d, want 2", len(result))
	}
}

func TestFilterClientWebSearch_EmptyInput(t *testing.T) {
	result := filterClientWebSearch(nil)
	if len(result) != 0 {
		t.Fatalf("len(result) = %d, want 0", len(result))
	}
}

type overflowProvider struct {
	calls        int
	lastMessages []providers.Message
	chatFunc     func(ctx context.Context, messages []providers.Message, tools []providers.ToolDefinition, model string, opts map[string]any) (*providers.LLMResponse, error)
}

func (p *overflowProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	opts map[string]any,
) (*providers.LLMResponse, error) {
	p.calls++
	p.lastMessages = append([]providers.Message(nil), messages...)

	if p.chatFunc != nil {
		return p.chatFunc(ctx, messages, tools, model, opts)
	}

	if p.calls == 1 {
		return nil, errors.New("context_window_exceeded")
	}

	return &providers.LLMResponse{
		Content: "Recovered from overflow",
	}, nil
}

func (p *overflowProvider) GetDefaultModel() string {
	return "test-model"
}

func TestProcessMessage_ContextOverflowRecovery(t *testing.T) {
	al, cfg, _, _, cleanup := newTestAgentLoop(t)
	defer cleanup()
	_ = cfg

	provider := &overflowProvider{}
	al.registry = NewAgentRegistry(al.cfg, provider)

	sessionKey := "agent:main:test-session"
	agent := al.GetRegistry().GetDefaultAgent()

	for i := 0; i < 5; i++ {
		agent.Sessions.AddFullMessage(sessionKey, providers.Message{Role: "user", Content: "heavy message"})
		agent.Sessions.AddFullMessage(sessionKey, providers.Message{Role: "assistant", Content: "response"})
	}

	response, err := al.processMessage(context.Background(), bus.InboundMessage{
		Channel:    "test",
		ChatID:     "chat1",
		SenderID:   "user1",
		SessionKey: "test-session",
		Content:    "trigger recovery",
	})
	if err != nil {
		t.Fatalf("processMessage() error = %v", err)
	}
	if response != "Recovered from overflow" {
		t.Fatalf("response = %q, want %q", response, "Recovered from overflow")
	}

	if provider.calls != 2 {
		t.Fatalf("expected 2 calls, got %d", provider.calls)
	}
}

func TestProcessMessage_ContextOverflow_AnthropicStyle(t *testing.T) {
	al, cfg, _, _, cleanup := newTestAgentLoop(t)
	defer cleanup()
	_ = cfg

	provider := &overflowProvider{}
	al.registry = NewAgentRegistry(al.cfg, provider)

	recoveryMsg := "error: status 400: context_window_exceeded"

	provider.chatFunc = func(
		ctx context.Context,
		messages []providers.Message,
		tools []providers.ToolDefinition,
		model string,
		opts map[string]any,
	) (*providers.LLMResponse, error) {
		if provider.calls == 1 {
			return nil, errors.New(recoveryMsg)
		}
		return &providers.LLMResponse{Content: "Anthropic recovery success"}, nil
	}

	response, err := al.processMessage(context.Background(), bus.InboundMessage{
		Channel:  "test",
		ChatID:   "chat1",
		SenderID: "user1",
		Content:  "hello",
	})
	if err != nil {
		t.Fatalf("processMessage() error = %v", err)
	}
	if !strings.Contains(response, "Anthropic recovery success") {
		t.Fatalf("response = %q, want success message", response)
	}
	if provider.calls != 2 {
		t.Fatalf("expected 2 calls for retry, got %d", provider.calls)
	}
}
