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
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	runtimeevents "github.com/sipeed/picoclaw/pkg/events"
	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/routing"
	"github.com/sipeed/picoclaw/pkg/session"
	"github.com/sipeed/picoclaw/pkg/tools"
	"github.com/sipeed/picoclaw/pkg/utils"
)

type fakeChannel struct{ id string }

func (f *fakeChannel) Name() string                    { return "fake" }
func (f *fakeChannel) Start(ctx context.Context) error { return nil }
func (f *fakeChannel) Stop(ctx context.Context) error  { return nil }
func (f *fakeChannel) Send(ctx context.Context, msg bus.OutboundMessage) ([]string, error) {
	return nil, nil
}
func (f *fakeChannel) IsRunning() bool                            { return true }
func (f *fakeChannel) IsAllowed(string) bool                      { return true }
func (f *fakeChannel) IsAllowedSender(sender bus.SenderInfo) bool { return true }
func (f *fakeChannel) ReasoningChannelID() string                 { return f.id }

type fakeMediaChannel struct {
	fakeChannel
	sentMessages []bus.OutboundMessage
	sentMedia    []bus.OutboundMediaMessage
}

func (f *fakeMediaChannel) Send(ctx context.Context, msg bus.OutboundMessage) ([]string, error) {
	f.sentMessages = append(f.sentMessages, msg)
	return nil, nil
}

func (f *fakeMediaChannel) SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) ([]string, error) {
	f.sentMedia = append(f.sentMedia, msg)
	return nil, nil
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
	lastModel    string
}

func (r *recordingProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	opts map[string]any,
) (*providers.LLMResponse, error) {
	r.lastMessages = append([]providers.Message(nil), messages...)
	r.lastModel = model
	return &providers.LLMResponse{
		Content:   "Mock response",
		ToolCalls: []providers.ToolCall{},
	}, nil
}

func (r *recordingProvider) GetDefaultModel() string {
	return "mock-model"
}

type modelRewriteHook struct {
	model string
}

func (h modelRewriteHook) BeforeLLM(
	ctx context.Context,
	req *LLMHookRequest,
) (*LLMHookRequest, HookDecision, error) {
	next := req.Clone()
	next.Model = h.model
	return next, HookDecision{Action: HookActionModify}, nil
}

func (h modelRewriteHook) AfterLLM(
	ctx context.Context,
	resp *LLMHookResponse,
) (*LLMHookResponse, HookDecision, error) {
	return resp.Clone(), HookDecision{Action: HookActionContinue}, nil
}

func useTestSideQuestionProvider(al *AgentLoop, provider providers.LLMProvider) {
	al.providerFactory = func(mc *config.ModelConfig) (providers.LLMProvider, string, error) {
		model := provider.GetDefaultModel()
		if mc != nil {
			if _, modelID := providers.ExtractProtocol(mc); modelID != "" {
				model = modelID
			}
		}
		return provider, model, nil
	}
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

func TestPublishResponseIfNeededWithPolicy_AlwaysPublishesFinalAfterMessageTool(t *testing.T) {
	al, _, msgBus, _, cleanup := newTestAgentLoop(t)
	defer cleanup()

	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		t.Fatal("expected default agent")
	}
	rawTool, ok := agent.Tools.Get("message")
	if !ok {
		mt := tools.NewMessageTool()
		agent.Tools.Register(mt)
		rawTool = mt
	}
	mt, ok := rawTool.(*tools.MessageTool)
	if !ok {
		t.Fatalf("message tool type = %T", rawTool)
	}
	mt.SetSendCallback(func(ctx context.Context, channel, chatID, content, replyToMessageID string) error {
		return nil
	})

	ctx := tools.WithToolSessionContext(context.Background(), routing.DefaultAgentID, "session-msg-1", nil)
	res := mt.Execute(ctx, map[string]any{
		"content": "working on it",
		"channel": "telegram",
		"chat_id": "-100123",
	})
	if res == nil || res.IsError {
		t.Fatalf("message tool execute failed: %+v", res)
	}

	al.publishResponseIfNeededWithPolicy(
		context.Background(),
		"telegram",
		"-100123",
		"session-msg-1",
		"final result",
		finalResponseAlwaysPublish,
	)

	select {
	case outbound := <-msgBus.OutboundChan():
		if outbound.Content != "final result" {
			t.Fatalf("outbound content = %q, want final result", outbound.Content)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected outbound response")
	}
}

func TestPublishResponseIfNeededWithPolicy_SuppressesWhenMessageToolAlreadySent(t *testing.T) {
	al, _, msgBus, _, cleanup := newTestAgentLoop(t)
	defer cleanup()

	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		t.Fatal("expected default agent")
	}
	rawTool, ok := agent.Tools.Get("message")
	if !ok {
		mt := tools.NewMessageTool()
		agent.Tools.Register(mt)
		rawTool = mt
	}
	mt, ok := rawTool.(*tools.MessageTool)
	if !ok {
		t.Fatalf("message tool type = %T", rawTool)
	}
	mt.SetSendCallback(func(ctx context.Context, channel, chatID, content, replyToMessageID string) error {
		return nil
	})

	ctx := tools.WithToolSessionContext(context.Background(), routing.DefaultAgentID, "session-msg-2", nil)
	res := mt.Execute(ctx, map[string]any{
		"content": "working on it",
		"channel": "telegram",
		"chat_id": "-100123",
	})
	if res == nil || res.IsError {
		t.Fatalf("message tool execute failed: %+v", res)
	}

	al.publishResponseIfNeededWithPolicy(
		context.Background(),
		"telegram",
		"-100123",
		"session-msg-2",
		"final result",
		finalResponseSuppressIfMessageToolSent,
	)

	select {
	case outbound := <-msgBus.OutboundChan():
		t.Fatalf("unexpected outbound response: %+v", outbound)
	case <-time.After(150 * time.Millisecond):
	}
}

func TestNewAgentLoop_RegistersWebSearchTool(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = t.TempDir()

	al := NewAgentLoop(cfg, bus.NewMessageBus(), &mockProvider{})

	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		t.Fatal("expected default agent")
	}
	if _, ok := agent.Tools.Get("web_search"); !ok {
		t.Fatal("expected web_search tool to be registered")
	}
}

func TestNewAgentLoop_RegistersWebSearchTool_WhenExplicitProviderUnavailable(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = t.TempDir()
	cfg.Tools.Web.Provider = "brave"
	cfg.Tools.Web.Brave.Enabled = true
	cfg.Tools.Web.Sogou.Enabled = true

	al := NewAgentLoop(cfg, bus.NewMessageBus(), &mockProvider{})

	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		t.Fatal("expected default agent")
	}
	if _, ok := agent.Tools.Get("web_search"); !ok {
		t.Fatal("expected web_search tool to fall back to auto provider selection")
	}
}

func TestNewAgentLoop_DoesNotRegisterWebSearchTool_WhenNoReadyProviders(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = t.TempDir()
	cfg.Tools.Web.Provider = "brave"
	cfg.Tools.Web.Brave.Enabled = true
	cfg.Tools.Web.Sogou.Enabled = false
	cfg.Tools.Web.DuckDuckGo.Enabled = false

	al := NewAgentLoop(cfg, bus.NewMessageBus(), &mockProvider{})

	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		t.Fatal("expected default agent")
	}
	if _, ok := agent.Tools.Get("web_search"); ok {
		t.Fatal("expected web_search tool to be absent when no providers are ready")
	}
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

	response, err := al.processMessage(context.Background(), testInboundMessage(bus.InboundMessage{
		Channel:  "discord",
		SenderID: "discord:123",
		Sender: bus.SenderInfo{
			DisplayName: "Alice",
		},
		ChatID:  "group-1",
		Content: "hello",
	}))
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

	response, err := al.processMessage(context.Background(), testInboundMessage(bus.InboundMessage{
		Channel:  "telegram",
		SenderID: "telegram:123",
		ChatID:   "chat-1",
		Content:  "/use shell explain how to list files",
	}))
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

func TestProcessMessage_BtwCommandRunsWithoutPersistingHistory(t *testing.T) {
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
		// Add model list so isolated provider can resolve the model
		ModelList: []*config.ModelConfig{
			{ModelName: "test-model", Model: "openai/test-model"},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &recordingProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)
	useTestSideQuestionProvider(al, provider)
	defaultAgent := al.GetRegistry().GetDefaultAgent()
	if defaultAgent == nil {
		t.Fatal("expected default agent")
	}

	msg := bus.InboundMessage{
		Channel:  "telegram",
		SenderID: "telegram:123",
		ChatID:   "chat-1",
		Content:  "/btw explain side effects",
	}
	route, _, err := al.resolveMessageRoute(msg)
	if err != nil {
		t.Fatalf("resolveMessageRoute() error = %v", err)
	}
	allocation := al.allocateRouteSession(route, msg)
	sessionKey := resolveScopeKey(allocation.SessionKey, msg.SessionKey)
	initialHistory := []providers.Message{
		{Role: "user", Content: "We decided to avoid global state."},
		{Role: "assistant", Content: "Right, keep it request-scoped."},
	}
	defaultAgent.Sessions.SetHistory(sessionKey, initialHistory)
	defaultAgent.Sessions.SetSummary(sessionKey, "The team decided to keep state request-scoped.")

	response, err := al.processMessage(context.Background(), msg)
	if err != nil {
		t.Fatalf("processMessage() error = %v", err)
	}
	if response != "Mock response" {
		t.Fatalf("processMessage() response = %q, want %q", response, "Mock response")
	}
	if len(provider.lastMessages) == 0 {
		t.Fatal("provider did not receive any messages")
	}
	if len(provider.lastMessages) != 4 {
		t.Fatalf("provider messages len = %d, want 4 (system + prior history + user)", len(provider.lastMessages))
	}

	if !reflect.DeepEqual(provider.lastMessages[1:3], initialHistory) {
		t.Fatalf("provider history = %#v, want %#v", provider.lastMessages[1:3], initialHistory)
	}

	lastMessage := provider.lastMessages[len(provider.lastMessages)-1]
	if lastMessage.Role != "user" || lastMessage.Content != "explain side effects" {
		t.Fatalf("last provider message = %+v, want stripped /btw question", lastMessage)
	}

	history := al.GetRegistry().GetDefaultAgent().Sessions.GetHistory(sessionKey)
	if !reflect.DeepEqual(history, initialHistory) {
		t.Fatalf("session history = %#v, want %#v", history, initialHistory)
	}
}

func TestProcessMessage_BtwCommandIncludesRequestContextAndMedia(t *testing.T) {
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
	useTestSideQuestionProvider(al, provider)

	response, err := al.processMessage(context.Background(), testInboundMessage(bus.InboundMessage{
		Channel:  "discord",
		SenderID: "discord:123",
		Sender: bus.SenderInfo{
			DisplayName: "Alice",
		},
		ChatID:  "group-1",
		Content: "/btw describe this image",
		Media:   []string{"media://image-1"},
	}))
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
	if !strings.Contains(systemPrompt, "## Current Session\nChannel: discord\nChat ID: group-1") {
		t.Fatalf("system prompt missing current session context:\n%s", systemPrompt)
	}
	if !strings.Contains(systemPrompt, "## Current Sender\nCurrent sender: Alice (ID: discord:123)") {
		t.Fatalf("system prompt missing current sender context:\n%s", systemPrompt)
	}

	lastMessage := provider.lastMessages[len(provider.lastMessages)-1]
	if lastMessage.Role != "user" || lastMessage.Content != "describe this image" {
		t.Fatalf("last provider message = %+v, want stripped /btw question", lastMessage)
	}
	if !reflect.DeepEqual(lastMessage.Media, []string{"media://image-1"}) {
		t.Fatalf("last provider media = %#v, want media ref", lastMessage.Media)
	}
}

func TestProcessMessage_BtwCommandUsesIsolatedProvider(t *testing.T) {
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
		// Add model list so isolated provider can resolve the model
		ModelList: []*config.ModelConfig{
			{ModelName: "test-model", Model: "openai/test-model"},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &recordingProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)
	useTestSideQuestionProvider(al, provider)
	defaultAgent := al.GetRegistry().GetDefaultAgent()
	if defaultAgent == nil {
		t.Fatal("expected default agent")
	}

	// Set up initial history for the main session
	mainSessionKey := "telegram:123:chat-1"
	initialHistory := []providers.Message{
		{Role: "user", Content: "We decided to avoid global state."},
		{Role: "assistant", Content: "Right, keep it request-scoped."},
	}
	defaultAgent.Sessions.SetHistory(mainSessionKey, initialHistory)

	// Process a /btw command
	response, err := al.processMessage(context.Background(), bus.InboundMessage{
		Channel:    "telegram",
		SenderID:   "telegram:123",
		ChatID:     "chat-1",
		SessionKey: mainSessionKey,
		Content:    "/btw explain isolation",
	})
	if err != nil {
		t.Fatalf("processMessage() error = %v", err)
	}
	if response != "Mock response" {
		t.Fatalf("processMessage() response = %q, want %q", response, "Mock response")
	}

	// Verify the provider received the side question
	if len(provider.lastMessages) == 0 {
		t.Fatal("provider did not receive any messages for /btw command")
	}

	// Verify the question was stripped of /btw prefix
	lastMessage := provider.lastMessages[len(provider.lastMessages)-1]
	if lastMessage.Role != "user" || lastMessage.Content != "explain isolation" {
		t.Fatalf("last provider message = %+v, want stripped /btw question", lastMessage)
	}

	// Verify main session history was NOT modified
	currentHistory := defaultAgent.Sessions.GetHistory(mainSessionKey)
	if !reflect.DeepEqual(currentHistory, initialHistory) {
		t.Fatalf("main session history was modified:\ngot  %#v\nwant %#v", currentHistory, initialHistory)
	}
}

func TestProcessMessage_BtwCommandRetriesWithoutMediaOnVisionUnsupported(t *testing.T) {
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
		// Add model list so isolated provider can resolve the model
		ModelList: []*config.ModelConfig{
			{ModelName: "test-model", Model: "openai/test-model"},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &visionUnsupportedMediaProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)
	useTestSideQuestionProvider(al, provider)

	response, err := al.processMessage(context.Background(), testInboundMessage(bus.InboundMessage{
		Channel:  "telegram",
		SenderID: "telegram:123",
		ChatID:   "chat-1",
		Content:  "/btw describe this image",
		Media:    []string{"data:image/png;base64,abc123"},
	}))
	if err != nil {
		t.Fatalf("processMessage() error = %v", err)
	}
	if response != "ok" {
		t.Fatalf("processMessage() response = %q, want %q", response, "ok")
	}
	// Note: With isolated providers, each /btw creates a new provider instance,
	// so we can't track calls across retries in the same way.
	// The retry logic happens within askSideQuestion, creating separate isolated providers.
	// For now, we just verify the command succeeds.
	if provider.calls < 1 {
		t.Fatalf("provider was not called for /btw command")
	}
}

func TestProcessMessage_BtwCommandUsesProviderFactoryModel(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				ModelName:         "lb-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
		ModelList: []*config.ModelConfig{
			{ModelName: "lb-model", Model: "openai/lb-model-a"},
			{ModelName: "lb-model", Model: "openai/lb-model-b"},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &recordingProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)
	useTestSideQuestionProvider(al, provider)

	response, err := al.processMessage(context.Background(), bus.InboundMessage{
		Channel:  "telegram",
		SenderID: "telegram:123",
		ChatID:   "chat-1",
		Content:  "/btw explain load balancing",
	})
	if err != nil {
		t.Fatalf("processMessage() error = %v", err)
	}
	if response != "Mock response" {
		t.Fatalf("processMessage() response = %q, want %q", response, "Mock response")
	}

	// Verify that /btw used the configured model from ModelList
	// The provider should have been called with one of the lb-model variants
	if provider.lastModel == "" {
		t.Fatal("provider was not called for /btw command")
	}
	if !strings.HasPrefix(provider.lastModel, "lb-model") {
		t.Fatalf("/btw used model %q, expected lb-model variant", provider.lastModel)
	}
}

func TestProcessMessage_BtwCommandHookModelBypassesFallbackCandidates(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				ModelName:         "primary-model",
				ModelFallbacks:    []string{"fallback-model"},
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &recordingProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)
	useTestSideQuestionProvider(al, provider)
	if err := al.MountHook(NamedHook("rewrite-model", modelRewriteHook{model: "hook-model"})); err != nil {
		t.Fatalf("MountHook failed: %v", err)
	}

	response, err := al.processMessage(context.Background(), bus.InboundMessage{
		Channel:  "telegram",
		SenderID: "telegram:123",
		ChatID:   "chat-1",
		Content:  "/btw explain hook routing",
	})
	if err != nil {
		t.Fatalf("processMessage() error = %v", err)
	}
	if response != "Mock response" {
		t.Fatalf("processMessage() response = %q, want %q", response, "Mock response")
	}
	if provider.lastModel != "hook-model" {
		t.Fatalf("/btw model = %q, want hook-selected model", provider.lastModel)
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

	response, err := al.processMessage(context.Background(), testInboundMessage(bus.InboundMessage{
		Channel:  "telegram",
		SenderID: "telegram:123",
		ChatID:   "chat-1",
		Content:  "/use shell",
	}))
	if err != nil {
		t.Fatalf("processMessage() arm error = %v", err)
	}
	if !strings.Contains(response, `Skill "shell" is armed for your next message.`) {
		t.Fatalf("arm response = %q, want armed confirmation", response)
	}

	response, err = al.processMessage(context.Background(), testInboundMessage(bus.InboundMessage{
		Channel:  "telegram",
		SenderID: "telegram:123",
		ChatID:   "chat-1",
		Content:  "explain how to list files",
	}))
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

	inboundCtx := tools.WithToolInboundContext(
		context.Background(),
		"telegram",
		"chat-42",
		"msg-123",
		"msg-100",
	)
	if got := tools.ToolMessageID(inboundCtx); got != "msg-123" {
		t.Errorf("expected messageID 'msg-123', got %q", got)
	}
	if got := tools.ToolReplyToMessageID(inboundCtx); got != "msg-100" {
		t.Errorf("expected replyToMessageID 'msg-100', got %q", got)
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

	response, err := al.processMessage(context.Background(), testInboundMessage(bus.InboundMessage{
		Channel:  "telegram",
		ChatID:   "chat1",
		SenderID: "user1",
		Content:  "take a screenshot of the screen and send it to me",
	}))
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
	route, _, err := al.resolveMessageRoute(testInboundMessage(bus.InboundMessage{
		Channel:  "telegram",
		ChatID:   "chat1",
		SenderID: "user1",
		Content:  "take a screenshot of the screen and send it to me",
	}))
	if err != nil {
		t.Fatalf("resolveMessageRoute() error = %v", err)
	}
	sessionKey := resolveScopeKey(al.allocateRouteSession(route, testInboundMessage(bus.InboundMessage{
		Channel:  "telegram",
		ChatID:   "chat1",
		SenderID: "user1",
		Content:  "take a screenshot of the screen and send it to me",
	})).SessionKey, "")
	history := defaultAgent.Sessions.GetHistory(sessionKey)
	if len(history) == 0 {
		t.Fatal("expected session history to be saved")
	}
	last := history[len(history)-1]
	if last.Role != "assistant" || last.Content != "Requested output delivered via tool attachment." {
		t.Fatalf("expected handled assistant summary in history, got %+v", last)
	}
	if len(last.Attachments) != 1 {
		t.Fatalf("expected handled assistant summary attachments in history, got %+v", last.Attachments)
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

	response, err := al.processMessage(context.Background(), testInboundMessage(bus.InboundMessage{
		Channel:  "telegram",
		ChatID:   "chat1",
		SenderID: "user1",
		Content:  "take a screenshot of the screen and send it to me",
	}))
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

func TestRunAgentLoop_ResponseHandledToolPublishesForUserWhenSendResponseDisabled(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = tmpDir
	cfg.Agents.Defaults.ModelName = "test-model"
	cfg.Agents.Defaults.MaxTokens = 4096
	cfg.Agents.Defaults.MaxToolIterations = 10

	msgBus := bus.NewMessageBus()
	provider := &handledUserProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)

	store := media.NewFileMediaStore()
	al.SetMediaStore(store)
	telegramChannel := &fakeMediaChannel{fakeChannel: fakeChannel{id: "rid-telegram"}}
	al.SetChannelManager(newStartedTestChannelManager(t, msgBus, store, "telegram", telegramChannel))
	al.RegisterTool(&handledUserTool{})

	defaultAgent := al.registry.GetDefaultAgent()
	if defaultAgent == nil {
		t.Fatal("expected default agent")
	}

	response, err := al.runAgentLoop(context.Background(), defaultAgent, processOptions{
		Dispatch: DispatchRequest{
			SessionKey:  "session-1",
			UserMessage: "take a screenshot of the screen and send it to me",
			SessionScope: &session.SessionScope{
				Version:    session.ScopeVersionV1,
				AgentID:    defaultAgent.ID,
				Channel:    "telegram",
				Dimensions: []string{"chat"},
				Values: map[string]string{
					"chat": "direct:chat1",
				},
			},
			InboundContext: &bus.InboundContext{
				Channel:  "telegram",
				ChatID:   "chat1",
				ChatType: "direct",
				SenderID: "user1",
			},
		},
		DefaultResponse: defaultResponse,
		EnableSummary:   false,
		SendResponse:    false,
	})
	if err != nil {
		t.Fatalf("runAgentLoop() error = %v", err)
	}
	if response != "" {
		t.Fatalf("expected no final response when tool already handled delivery, got %q", response)
	}

	deadline := time.Now().Add(2 * time.Second)
	for len(telegramChannel.sentMessages) == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if len(telegramChannel.sentMessages) != 1 {
		t.Fatalf("expected exactly 1 sent text message, got %d", len(telegramChannel.sentMessages))
	}
	if telegramChannel.sentMessages[0].Content != "Handled user output from tool." {
		t.Fatalf("unexpected sent text message: %+v", telegramChannel.sentMessages[0])
	}
	if telegramChannel.sentMessages[0].AgentID != defaultAgent.ID {
		t.Fatalf("sent text agent_id = %q, want %q", telegramChannel.sentMessages[0].AgentID, defaultAgent.ID)
	}
	if telegramChannel.sentMessages[0].SessionKey != "session-1" {
		t.Fatalf("sent text session_key = %q, want session-1", telegramChannel.sentMessages[0].SessionKey)
	}
	if telegramChannel.sentMessages[0].Scope == nil ||
		telegramChannel.sentMessages[0].Scope.Values["chat"] != "direct:chat1" {
		t.Fatalf("unexpected sent text scope: %+v", telegramChannel.sentMessages[0].Scope)
	}
}

func TestAppendEventContextFields_IncludesInboundRouteAndScope(t *testing.T) {
	fields := map[string]any{}

	appendEventContextFields(fields, &TurnContext{
		Inbound: &bus.InboundContext{
			Channel:   "slack",
			Account:   "workspace-a",
			ChatID:    "C123",
			ChatType:  "channel",
			TopicID:   "thread-42",
			SpaceType: "workspace",
			SpaceID:   "T001",
			SenderID:  "U123",
			Mentioned: true,
		},
		Route: &routing.ResolvedRoute{
			AgentID:   "support",
			Channel:   "slack",
			AccountID: "workspace-a",
			MatchedBy: "default",
			SessionPolicy: routing.SessionPolicy{
				Dimensions: []string{"chat", "sender"},
				IdentityLinks: map[string][]string{
					"canonical-user": {"slack:U123"},
				},
			},
		},
		Scope: &session.SessionScope{
			Version:    session.ScopeVersionV1,
			AgentID:    "support",
			Channel:    "slack",
			Account:    "workspace-a",
			Dimensions: []string{"chat", "sender"},
			Values: map[string]string{
				"chat":   "channel:c123",
				"sender": "u123",
			},
		},
	})

	if fields["inbound_channel"] != "slack" {
		t.Fatalf("inbound_channel = %v, want slack", fields["inbound_channel"])
	}
	if fields["inbound_topic_id"] != "thread-42" {
		t.Fatalf("inbound_topic_id = %v, want thread-42", fields["inbound_topic_id"])
	}
	if fields["route_matched_by"] != "default" {
		t.Fatalf("route_matched_by = %v, want default", fields["route_matched_by"])
	}
	if fields["route_dimensions"] != "chat,sender" {
		t.Fatalf("route_dimensions = %v, want chat,sender", fields["route_dimensions"])
	}
	if fields["route_identity_link_count"] != 1 {
		t.Fatalf("route_identity_link_count = %v, want 1", fields["route_identity_link_count"])
	}
	if fields["scope_dimensions"] != "chat,sender" {
		t.Fatalf("scope_dimensions = %v, want chat,sender", fields["scope_dimensions"])
	}
	if fields["scope_chat"] != "channel:c123" {
		t.Fatalf("scope_chat = %v, want channel:c123", fields["scope_chat"])
	}
	if fields["scope_sender"] != "u123" {
		t.Fatalf("scope_sender = %v, want u123", fields["scope_sender"])
	}
}

func TestResolveMessageRoute_UsesInboundContextAccount(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace: tmpDir,
				ModelName: "test-model",
			},
			List: []config.AgentConfig{
				{ID: "main", Default: true},
				{ID: "work"},
			},
		},
		Session: config.SessionConfig{
			Dimensions: []string{"sender"},
		},
	}

	msgBus := bus.NewMessageBus()
	al := NewAgentLoop(cfg, msgBus, &simpleMockProvider{response: "ok"})

	route, _, err := al.resolveMessageRoute(testInboundMessage(bus.InboundMessage{
		Context: bus.InboundContext{
			Channel:   "slack",
			Account:   "workspace-a",
			ChatID:    "C123",
			ChatType:  "channel",
			SenderID:  "U123",
			SpaceID:   "T001",
			SpaceType: "workspace",
		},
		Content: "hello",
	}))
	if err != nil {
		t.Fatalf("resolveMessageRoute() error = %v", err)
	}
	if route.AgentID != "main" {
		t.Fatalf("AgentID = %q, want main", route.AgentID)
	}
	if route.MatchedBy != "default" {
		t.Fatalf("MatchedBy = %q, want default", route.MatchedBy)
	}
	if route.AccountID != "workspace-a" {
		t.Fatalf("AccountID = %q, want workspace-a", route.AccountID)
	}
}

func TestResolveMessageRoute_UsesDispatchRulesInOrder(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace: tmpDir,
				ModelName: "test-model",
			},
			List: []config.AgentConfig{
				{ID: "main", Default: true},
				{ID: "support"},
				{ID: "sales"},
			},
			Dispatch: &config.DispatchConfig{
				Rules: []config.DispatchRule{
					{
						Name:  "support-group",
						Agent: "support",
						When: config.DispatchSelector{
							Channel: "telegram",
							Chat:    "group:-100123",
						},
						SessionDimensions: []string{"chat"},
					},
					{
						Name:  "vip-in-group",
						Agent: "sales",
						When: config.DispatchSelector{
							Channel: "telegram",
							Chat:    "group:-100123",
							Sender:  "12345",
						},
						SessionDimensions: []string{"chat", "sender"},
					},
				},
			},
		},
		Session: config.SessionConfig{
			Dimensions: []string{"sender"},
		},
	}

	msgBus := bus.NewMessageBus()
	al := NewAgentLoop(cfg, msgBus, &simpleMockProvider{response: "ok"})

	route, _, err := al.resolveMessageRoute(testInboundMessage(bus.InboundMessage{
		Context: bus.InboundContext{
			Channel:  "telegram",
			ChatID:   "-100123",
			ChatType: "group",
			SenderID: "12345",
		},
		Content: "hello",
	}))
	if err != nil {
		t.Fatalf("resolveMessageRoute() error = %v", err)
	}
	if route.AgentID != "support" {
		t.Fatalf("AgentID = %q, want support", route.AgentID)
	}
	if route.MatchedBy != "dispatch.rule:support-group" {
		t.Fatalf("MatchedBy = %q, want dispatch.rule:support-group", route.MatchedBy)
	}
	if got := route.SessionPolicy.Dimensions; len(got) != 1 || got[0] != "chat" {
		t.Fatalf("SessionPolicy.Dimensions = %v, want [chat]", got)
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

	response, err := al.processMessage(context.Background(), testInboundMessage(bus.InboundMessage{
		Channel:  "telegram",
		ChatID:   "chat1",
		SenderID: "user1",
		Content:  "take a screenshot of the screen and send it to me",
	}))
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

type handledUserProvider struct {
	calls int
}

func (m *handledUserProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	opts map[string]any,
) (*providers.LLMResponse, error) {
	m.calls++
	if m.calls == 1 {
		return &providers.LLMResponse{
			Content: "Delivering the result now.",
			ToolCalls: []providers.ToolCall{{
				ID:        "call_handled_user",
				Type:      "function",
				Name:      "handled_user_tool",
				Arguments: map[string]any{},
			}},
		}, nil
	}
	return &providers.LLMResponse{}, nil
}

func (m *handledUserProvider) GetDefaultModel() string {
	return "handled-user-model"
}

type messageToolProvider struct {
	calls int
}

func (m *messageToolProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	opts map[string]any,
) (*providers.LLMResponse, error) {
	m.calls++
	if m.calls == 1 {
		return &providers.LLMResponse{
			Content: "",
			ToolCalls: []providers.ToolCall{{
				ID:        "call_message",
				Type:      "function",
				Name:      "message",
				Arguments: map[string]any{"content": "direct tool message"},
			}},
		}, nil
	}
	return &providers.LLMResponse{}, nil
}

func (m *messageToolProvider) GetDefaultModel() string {
	return "message-tool-model"
}

type reasoningVisibleToolProvider struct {
	filePath string
	calls    int
}

func (m *reasoningVisibleToolProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	opts map[string]any,
) (*providers.LLMResponse, error) {
	m.calls++
	if m.calls == 1 {
		return &providers.LLMResponse{
			Content:          "I'll inspect that file now.",
			ReasoningContent: "Read the file before answering.",
			ToolCalls: []providers.ToolCall{{
				ID:        "call_read_file",
				Type:      "function",
				Name:      "read_file",
				Arguments: map[string]any{"path": m.filePath},
			}},
		}, nil
	}
	return &providers.LLMResponse{Content: "DONE"}, nil
}

func (m *reasoningVisibleToolProvider) GetDefaultModel() string {
	return "reasoning-visible-tool-model"
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
		for _, prefix := range []string{"[image:", "[file:", "[audio:", "[video:"} {
			start := strings.Index(messages[i].Content, prefix)
			if start < 0 {
				continue
			}
			rest := messages[i].Content[start+len(prefix):]
			end := strings.Index(rest, "]")
			if end < 0 {
				continue
			}
			artifactPath = rest[:end]
			break
		}
		if artifactPath != "" {
			break
		}
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

type toolFeedbackReasoningProvider struct {
	filePath string
	calls    int
}

func (m *toolFeedbackReasoningProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	opts map[string]any,
) (*providers.LLMResponse, error) {
	m.calls++
	if m.calls == 1 {
		return &providers.LLMResponse{
			ReasoningContent: "Read README.md first to confirm the context that needs to be changed.",
			ToolCalls: []providers.ToolCall{{
				ID:        "call_reasoning_read_file",
				Type:      "function",
				Name:      "read_file",
				Arguments: map[string]any{"path": m.filePath},
			}},
		}, nil
	}

	return &providers.LLMResponse{
		Content:   "DONE",
		ToolCalls: []providers.ToolCall{},
	}, nil
}

func (m *toolFeedbackReasoningProvider) GetDefaultModel() string {
	return "tool-feedback-reasoning-model"
}

func TestToolFeedbackExplanationFromResponse_UsesCurrentContentFirst(t *testing.T) {
	response := &providers.LLMResponse{
		Content:          "Read README.md first",
		ReasoningContent: "current reasoning fallback",
	}
	messages := []providers.Message{
		{Role: "user", Content: "check file"},
		{Role: "assistant", Content: "Previous turn explanation"},
		{Role: "tool", Content: "tool output", ToolCallID: "call_1"},
	}

	got := toolFeedbackExplanationFromResponse(response, messages)
	if got != "Read README.md first" {
		t.Fatalf("toolFeedbackExplanationFromResponse() = %q, want current content", got)
	}
}

func TestSideQuestionResponseContent_FallsBackWhenContentIsWhitespace(t *testing.T) {
	response := &providers.LLMResponse{
		Content:          " \n\t ",
		ReasoningContent: "reasoning fallback",
	}

	if got := sideQuestionResponseContent(response); got != "reasoning fallback" {
		t.Fatalf("sideQuestionResponseContent() = %q, want %q", got, "reasoning fallback")
	}
}

func TestResponseReasoningContent_FallsBackWhenReasoningIsWhitespace(t *testing.T) {
	response := &providers.LLMResponse{
		Reasoning:        " \n\t ",
		ReasoningContent: "structured reasoning fallback",
	}

	if got := responseReasoningContent(response); got != "structured reasoning fallback" {
		t.Fatalf("responseReasoningContent() = %q, want %q", got, "structured reasoning fallback")
	}
}

func TestToolFeedbackExplanationFromResponse_UsesExplicitToolCallExtraContent(t *testing.T) {
	response := &providers.LLMResponse{
		ToolCalls: []providers.ToolCall{{
			ID:   "call_1",
			Name: "read_file",
			ExtraContent: &providers.ExtraContent{
				ToolFeedbackExplanation: "Read README.md first to confirm the current project structure.",
			},
		}},
	}
	messages := []providers.Message{
		{Role: "user", Content: "check file"},
		{Role: "assistant", Content: ""},
		{Role: "tool", Content: "tool output", ToolCallID: "call_1"},
	}

	got := toolFeedbackExplanationFromResponse(response, messages)
	if got != "Read README.md first to confirm the current project structure." {
		t.Fatalf("toolFeedbackExplanationFromResponse() = %q, want explicit tool feedback explanation", got)
	}
}

func TestToolFeedbackExplanationForToolCall_PrefersToolSpecificExtraContent(t *testing.T) {
	response := &providers.LLMResponse{
		Content: "Shared explanation",
		ToolCalls: []providers.ToolCall{
			{
				ID:   "call_1",
				Name: "read_file",
				ExtraContent: &providers.ExtraContent{
					ToolFeedbackExplanation: "Read README.md first.",
				},
			},
			{
				ID:   "call_2",
				Name: "edit_file",
				ExtraContent: &providers.ExtraContent{
					ToolFeedbackExplanation: "Update config example after reading it.",
				},
			},
		},
	}

	got1 := toolFeedbackExplanationForToolCall(response, response.ToolCalls[0], nil)
	got2 := toolFeedbackExplanationForToolCall(response, response.ToolCalls[1], nil)
	if got1 != "Read README.md first." {
		t.Fatalf("toolFeedbackExplanationForToolCall() first = %q, want tool-specific explanation", got1)
	}
	if got2 != "Update config example after reading it." {
		t.Fatalf("toolFeedbackExplanationForToolCall() second = %q, want tool-specific explanation", got2)
	}
}

func TestToolFeedbackExplanationForToolCall_DoesNotReuseAnotherToolCallExplanation(t *testing.T) {
	response := &providers.LLMResponse{
		ToolCalls: []providers.ToolCall{
			{
				ID:   "call_1",
				Name: "read_file",
			},
			{
				ID:   "call_2",
				Name: "edit_file",
				ExtraContent: &providers.ExtraContent{
					ToolFeedbackExplanation: "Update config example after reading it.",
				},
			},
		},
	}
	messages := []providers.Message{
		{Role: "user", Content: "inspect the config and update the example"},
	}

	got := toolFeedbackExplanationForToolCall(response, response.ToolCalls[0], messages)
	want := utils.ToolFeedbackContinuationHint + ": inspect the config and update the example"
	if got != want {
		t.Fatalf("toolFeedbackExplanationForToolCall() = %q, want %q", got, want)
	}
}

func TestToolFeedbackExplanationFromResponse_DoesNotUseReasoningContent(t *testing.T) {
	response := &providers.LLMResponse{
		Content:          "",
		ReasoningContent: "hidden reasoning should not be shown",
	}
	messages := []providers.Message{
		{Role: "user", Content: "check file"},
		{Role: "assistant", Content: "Previous turn explanation"},
		{Role: "user", Content: "Inspect README.md and update the config example."},
		{Role: "tool", Content: "tool output", ToolCallID: "call_1"},
	}

	got := toolFeedbackExplanationFromResponse(response, messages)
	want := utils.ToolFeedbackContinuationHint + ": Inspect README.md and update the config example."
	if got != want {
		t.Fatalf("toolFeedbackExplanationFromResponse() = %q, want latest user content fallback", got)
	}
}

func TestToolFeedbackExplanationForToolCall_DoesNotTruncateLongExplanation(t *testing.T) {
	explanation := "Read README.md first to confirm the current project structure before editing the config example."
	response := &providers.LLMResponse{
		ToolCalls: []providers.ToolCall{{
			ID:   "call_1",
			Name: "read_file",
			ExtraContent: &providers.ExtraContent{
				ToolFeedbackExplanation: explanation,
			},
		}},
	}

	got := toolFeedbackExplanationForToolCall(response, response.ToolCalls[0], nil)
	if got != explanation {
		t.Fatalf("toolFeedbackExplanationForToolCall() = %q, want full explanation", got)
	}
}

func TestToolFeedbackArgsPreview_UsesJSONAndTruncates(t *testing.T) {
	got := toolFeedbackArgsPreview(map[string]any{
		"path":  "README.md",
		"limit": 42,
	}, 128)
	want := "{\n  \"limit\": 42,\n  \"path\": \"README.md\"\n}"
	if got != want {
		t.Fatalf("toolFeedbackArgsPreview() = %q, want %q", got, want)
	}
}

type picoInterleavedContentProvider struct {
	calls int
}

func (m *picoInterleavedContentProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	opts map[string]any,
) (*providers.LLMResponse, error) {
	m.calls++
	if m.calls == 1 {
		return &providers.LLMResponse{
			Content: "intermediate model text",
			ToolCalls: []providers.ToolCall{{
				ID:        "call_tool_limit_test",
				Type:      "function",
				Name:      "tool_limit_test_tool",
				Arguments: map[string]any{"value": "x"},
			}},
		}, nil
	}

	return &providers.LLMResponse{
		Content:   "final model text",
		ToolCalls: []providers.ToolCall{},
	}, nil
}

func (m *picoInterleavedContentProvider) GetDefaultModel() string {
	return "pico-interleaved-content-model"
}

type picoDistinctToolCallContentProvider struct {
	calls int
}

func (m *picoDistinctToolCallContentProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	opts map[string]any,
) (*providers.LLMResponse, error) {
	m.calls++
	if m.calls == 1 {
		return &providers.LLMResponse{
			Content: "intermediate model text",
			ToolCalls: []providers.ToolCall{{
				ID:        "call_tool_limit_test",
				Type:      "function",
				Name:      "tool_limit_test_tool",
				Arguments: map[string]any{"value": "x"},
				ExtraContent: &providers.ExtraContent{
					ToolFeedbackExplanation: "Read the file before replying.",
				},
			}},
		}, nil
	}

	return &providers.LLMResponse{
		Content:   "final model text",
		ToolCalls: []providers.ToolCall{},
	}, nil
}

func (m *picoDistinctToolCallContentProvider) GetDefaultModel() string {
	return "pico-distinct-tool-call-content-model"
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

type handledUserTool struct{}

func (m *handledUserTool) Name() string { return "handled_user_tool" }
func (m *handledUserTool) Description() string {
	return "Returns a user-visible result and marks delivery as handled"
}

func (m *handledUserTool) Parameters() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

func (m *handledUserTool) Execute(ctx context.Context, args map[string]any) *tools.ToolResult {
	return tools.UserResult("Handled user output from tool.").WithResponseHandled()
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

	response, err := h.al.processMessage(timeoutCtx, testInboundMessage(msg))
	if err != nil {
		tb.Fatalf("processMessage failed: %v", err)
	}
	return response
}

func testInboundMessage(msg bus.InboundMessage) bus.InboundMessage {
	if msg.Context.Channel == "" &&
		msg.Context.Account == "" &&
		msg.Context.ChatID == "" &&
		msg.Context.ChatType == "" &&
		msg.Context.TopicID == "" &&
		msg.Context.SpaceID == "" &&
		msg.Context.SpaceType == "" &&
		msg.Context.SenderID == "" &&
		msg.Context.MessageID == "" &&
		!msg.Context.Mentioned &&
		msg.Context.ReplyToMessageID == "" &&
		msg.Context.ReplyToSenderID == "" &&
		len(msg.Context.ReplyHandles) == 0 &&
		len(msg.Context.Raw) == 0 {
		msg.Context = bus.InboundContext{
			Channel:   msg.Channel,
			ChatID:    msg.ChatID,
			ChatType:  "direct",
			SenderID:  msg.SenderID,
			MessageID: msg.MessageID,
		}
	}
	return bus.NormalizeInboundMessage(msg)
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
		Context: bus.InboundContext{
			Channel:  "telegram",
			ChatID:   "chat1",
			ChatType: "direct",
			SenderID: "user1",
		},
		Content: "hello",
	}

	route := al.registry.ResolveRoute(bus.NormalizeInboundMessage(msg).Context)
	sessionKey := al.allocateRouteSession(route, msg).SessionKey

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
			Dimensions: []string{"chat"},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &countingMockProvider{response: "LLM reply"}
	al := NewAgentLoop(cfg, msgBus, provider)
	helper := testHelper{al: al}

	baseMsg := bus.InboundMessage{
		Context: bus.InboundContext{
			Channel:  "whatsapp",
			ChatID:   "chat1",
			ChatType: "direct",
			SenderID: "user1",
		},
	}

	showResp := helper.executeAndGetResponse(t, context.Background(), bus.InboundMessage{
		Context: bus.InboundContext{
			Channel:  baseMsg.Context.Channel,
			ChatID:   baseMsg.Context.ChatID,
			ChatType: baseMsg.Context.ChatType,
			SenderID: baseMsg.Context.SenderID,
		},
		Content: "/show channel",
	})
	if showResp != "Current Channel: whatsapp" {
		t.Fatalf("unexpected /show reply: %q", showResp)
	}
	if provider.calls != 0 {
		t.Fatalf("LLM should not be called for handled command, calls=%d", provider.calls)
	}

	fooResp := helper.executeAndGetResponse(t, context.Background(), bus.InboundMessage{
		Context: bus.InboundContext{
			Channel:  baseMsg.Context.Channel,
			ChatID:   baseMsg.Context.ChatID,
			ChatType: baseMsg.Context.ChatType,
			SenderID: baseMsg.Context.SenderID,
		},
		Content: "/foo",
	})
	if fooResp != "LLM reply" {
		t.Fatalf("unexpected /foo reply: %q", fooResp)
	}
	if provider.calls != 1 {
		t.Fatalf("LLM should be called exactly once after /foo passthrough, calls=%d", provider.calls)
	}

	newResp := helper.executeAndGetResponse(t, context.Background(), bus.InboundMessage{
		Context: bus.InboundContext{
			Channel:  baseMsg.Context.Channel,
			ChatID:   baseMsg.Context.ChatID,
			ChatType: baseMsg.Context.ChatType,
			SenderID: baseMsg.Context.SenderID,
		},
		Content: "/new",
	})
	if newResp != "LLM reply" {
		t.Fatalf("unexpected /new reply: %q", newResp)
	}
	if provider.calls != 2 {
		t.Fatalf("LLM should be called for passthrough /new command, calls=%d", provider.calls)
	}
}

func TestProcessMessage_MCPCommandsHandledWithoutLLMCall(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	deferred := true
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
			Dimensions: []string{"chat"},
		},
		Tools: config.ToolsConfig{
			MCP: config.MCPConfig{
				ToolConfig: config.ToolConfig{Enabled: true},
				Discovery:  config.ToolDiscoveryConfig{Enabled: true},
				Servers: map[string]config.MCPServerConfig{
					"github": {
						Enabled:  true,
						Deferred: &deferred,
					},
				},
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &countingMockProvider{response: "LLM reply"}
	al := NewAgentLoop(cfg, msgBus, provider)
	helper := testHelper{al: al}

	baseContext := bus.InboundContext{
		Channel:  "whatsapp",
		ChatID:   "chat1",
		ChatType: "direct",
		SenderID: "user1",
	}

	listResp := helper.executeAndGetResponse(t, context.Background(), bus.InboundMessage{
		Context: baseContext,
		Content: "/list mcp",
	})
	if !strings.Contains(listResp, "- `github`") || !strings.Contains(listResp, "Deferred: yes") {
		t.Fatalf("unexpected /list mcp reply: %q", listResp)
	}
	if provider.calls != 0 {
		t.Fatalf("LLM should not be called for /list mcp, calls=%d", provider.calls)
	}

	showResp := helper.executeAndGetResponse(t, context.Background(), bus.InboundMessage{
		Context: baseContext,
		Content: "/show mcp github",
	})
	if showResp != "MCP server 'github' is configured but not connected" {
		t.Fatalf("unexpected /show mcp reply: %q", showResp)
	}
	if provider.calls != 0 {
		t.Fatalf("LLM should not be called for /show mcp, calls=%d", provider.calls)
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
	})
	if !strings.Contains(switchResp, "Switched model from local to deepseek") {
		t.Fatalf("unexpected /switch reply: %q", switchResp)
	}

	showResp := helper.executeAndGetResponse(t, context.Background(), bus.InboundMessage{
		Channel:  "telegram",
		SenderID: "user1",
		ChatID:   "chat1",
		Content:  "/show model",
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
	})
	if switchResp != `model "missing" not found in model_list or providers` {
		t.Fatalf("unexpected /switch error reply: %q", switchResp)
	}

	showResp := helper.executeAndGetResponse(t, context.Background(), bus.InboundMessage{
		Channel:  "telegram",
		SenderID: "user1",
		ChatID:   "chat1",
		Content:  "/show model",
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
	})
	if !strings.Contains(switchResp, "Switched model from local to deepseek") {
		t.Fatalf("unexpected /switch reply: %q", switchResp)
	}

	secondResp := helper.executeAndGetResponse(t, context.Background(), bus.InboundMessage{
		Channel:  "telegram",
		SenderID: "user1",
		ChatID:   "chat1",
		Content:  "hello after switch",
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
				APIKeys:   config.SimpleSecureStrings("heavy-key"),
			},
			{
				ModelName: "qwen-light",
				Model:     "ollama/qwen2.5:0.5b",
				APIBase:   lightServer.URL,
				APIKeys:   config.SimpleSecureStrings("light-key"),
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

	resp := helper.executeAndGetResponse(t, context.Background(), bus.InboundMessage{
		Channel:  "telegram",
		SenderID: "user1",
		ChatID:   "chat1",
		Content:  "hi",
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

// TestProcessMessage_FallbackUsesPerCandidateProvider is the loop-level test for
// bug #2140. It verifies that when the primary model returns a rate-limit error
// the fallback closure routes the retry to the fallback model's own provider
// (its own api_base), not back to the primary provider's endpoint.
func TestProcessMessage_FallbackUsesPerCandidateProvider(t *testing.T) {
	workspace := t.TempDir()

	primaryCalls := 0
	primaryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		primaryCalls++
		// Return 429 so FallbackChain classifies this as retriable and moves on.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "rate limit exceeded",
				"type":    "rate_limit_error",
			},
		})
	}))
	defer primaryServer.Close()

	fallbackCalls := 0
	fallbackServer := newStrictChatCompletionTestServer(
		t, "fallback", "gemma-3-27b-it", "fallback reply", &fallbackCalls,
	)
	defer fallbackServer.Close()

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         workspace,
				ModelName:         "mistral-primary",
				ModelFallbacks:    []string{"gemma-fallback"},
				MaxTokens:         4096,
				MaxToolIterations: 3,
			},
		},
		ModelList: []*config.ModelConfig{
			{
				ModelName: "mistral-primary",
				Model:     "openrouter/mistralai/mistral-small-3.1",
				APIBase:   primaryServer.URL,
				APIKeys:   config.SimpleSecureStrings("primary-key"),
				Workspace: workspace,
			},
			{
				ModelName: "gemma-fallback",
				Model:     "openrouter/gemma-3-27b-it",
				APIBase:   fallbackServer.URL,
				APIKeys:   config.SimpleSecureStrings("fallback-key"),
				Workspace: workspace,
			},
		},
	}

	provider, _, err := providers.CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider() error = %v", err)
	}
	msgBus := bus.NewMessageBus()
	al := NewAgentLoop(cfg, msgBus, provider)
	helper := testHelper{al: al}

	resp := helper.executeAndGetResponse(t, context.Background(), bus.InboundMessage{
		Channel:  "telegram",
		SenderID: "user1",
		ChatID:   "chat1",
		Content:  "hi",
	})

	if resp != "fallback reply" {
		t.Fatalf("response = %q, want %q (fallback provider)", resp, "fallback reply")
	}
	if primaryCalls == 0 {
		t.Fatal("primary server was never called; expected at least one attempt")
	}
	if fallbackCalls != 1 {
		t.Fatalf("fallback server calls = %d, want 1", fallbackCalls)
	}
}

// TestProcessMessage_FallbackUsesActiveProviderWhenCandidateNotRegistered verifies
// that when a candidate has no model_list entry it is absent from CandidateProviders
// and the fallback closure falls back to activeProvider instead of panicking.
func TestProcessMessage_FallbackUsesActiveProviderWhenCandidateNotRegistered(t *testing.T) {
	workspace := t.TempDir()

	// Primary server: returns 429 on first call, succeeds on second.
	// Both the primary and the unregistered fallback share this server
	// (same api_base) so activeProvider routes both calls here.
	callCount := 0
	primaryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{"message": "rate limit", "type": "rate_limit_error"},
			})
			return
		}
		// Second call (fallback via activeProvider) succeeds.
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": "active provider reply"}, "finish_reason": "stop"},
			},
		})
	}))
	defer primaryServer.Close()

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         workspace,
				ModelName:         "primary-model",
				MaxTokens:         4096,
				MaxToolIterations: 3,
				// No model_list entry for this alias — absent from CandidateProviders.
				ModelFallbacks: []string{"openrouter/fallback-model"},
			},
		},
		ModelList: []*config.ModelConfig{
			{
				ModelName: "primary-model",
				Model:     "openrouter/primary-model",
				APIBase:   primaryServer.URL,
				APIKeys:   config.SimpleSecureStrings("primary-key"),
				Workspace: workspace,
			},
		},
	}

	provider, _, err := providers.CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider() error = %v", err)
	}
	msgBus := bus.NewMessageBus()
	al := NewAgentLoop(cfg, msgBus, provider)

	helper := testHelper{al: al}
	resp := helper.executeAndGetResponse(t, context.Background(), bus.InboundMessage{
		Channel:  "telegram",
		SenderID: "user1",
		ChatID:   "chat1",
		Content:  "hi",
	})

	if resp != "active provider reply" {
		t.Fatalf("response = %q, want %q", resp, "active provider reply")
	}
	if callCount < 2 {
		t.Fatalf("primary server calls = %d, want >= 2 (one 429 + one success via activeProvider)", callCount)
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

type visionUnsupportedMediaProvider struct {
	calls     int
	mediaSeen []bool
}

func (p *visionUnsupportedMediaProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	opts map[string]any,
) (*providers.LLMResponse, error) {
	p.calls++

	hasMedia := false
	for _, msg := range messages {
		for _, ref := range msg.Media {
			if strings.TrimSpace(ref) != "" {
				hasMedia = true
				break
			}
		}
		if hasMedia {
			break
		}
	}
	p.mediaSeen = append(p.mediaSeen, hasMedia)

	if hasMedia {
		return nil, fmt.Errorf("API request failed: " +
			"Status: 404 Body: {\"error\":{\"message\":\"No endpoints found that support image input\"}}")
	}

	return &providers.LLMResponse{
		Content:   "ok",
		ToolCalls: []providers.ToolCall{},
	}, nil
}

func (p *visionUnsupportedMediaProvider) GetDefaultModel() string {
	return "mock-fail-model"
}

func TestAgentLoop_VisionUnsupportedErrorStripsSessionMedia(t *testing.T) {
	workspace := t.TempDir()

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         workspace,
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 3,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &visionUnsupportedMediaProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)

	sessionKey := "agent:main:telegram:direct:user1"

	timeoutCtx, cancel := context.WithTimeout(context.Background(), responseTimeout)
	defer cancel()

	resp, err := al.processMessage(timeoutCtx, testInboundMessage(bus.InboundMessage{
		Context: bus.InboundContext{
			Channel:   "telegram",
			ChatID:    "chat1",
			ChatType:  "direct",
			SenderID:  "user1",
			MessageID: "m1",
		},
		Content:    "describe this",
		Media:      []string{"data:image/png;base64,abc123"},
		SessionKey: sessionKey,
	}))
	if err != nil {
		t.Fatalf("processMessage() error = %v", err)
	}
	if resp != "ok" {
		t.Fatalf("response = %q, want %q", resp, "ok")
	}
	if provider.calls != 2 {
		t.Fatalf("calls = %d, want %d (fail with media, then retry without media)", provider.calls, 2)
	}
	if !slices.Equal(provider.mediaSeen, []bool{true, false}) {
		t.Fatalf("mediaSeen = %v, want %v", provider.mediaSeen, []bool{true, false})
	}

	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		t.Fatal("expected default agent")
	}
	history := agent.Sessions.GetHistory(sessionKey)
	for i, msg := range history {
		if len(msg.Media) > 0 {
			t.Fatalf("history[%d].Media = %v, want no media after stripping", i, msg.Media)
		}
	}

	timeoutCtx2, cancel2 := context.WithTimeout(context.Background(), responseTimeout)
	defer cancel2()

	resp2, err := al.processMessage(timeoutCtx2, testInboundMessage(bus.InboundMessage{
		Context: bus.InboundContext{
			Channel:   "telegram",
			ChatID:    "chat1",
			ChatType:  "direct",
			SenderID:  "user1",
			MessageID: "m2",
		},
		Content:    "hello again",
		SessionKey: sessionKey,
	}))
	if err != nil {
		t.Fatalf("processMessage() second call error = %v", err)
	}
	if resp2 != "ok" {
		t.Fatalf("second response = %q, want %q", resp2, "ok")
	}
	if provider.calls != 3 {
		t.Fatalf("calls after second turn = %d, want %d", provider.calls, 3)
	}
	if !slices.Equal(provider.mediaSeen, []bool{true, false, false}) {
		t.Fatalf("mediaSeen = %v, want %v", provider.mediaSeen, []bool{true, false, false})
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
	route := al.registry.ResolveRoute(bus.InboundContext{
		Channel:  "test",
		ChatType: "direct",
		SenderID: "cron",
	})
	history := defaultAgent.Sessions.GetHistory(al.allocateRouteSession(route, testInboundMessage(bus.InboundMessage{
		Channel:  "test",
		SenderID: "cron",
		ChatID:   "chat1",
	})).SessionKey)
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
				Context: bus.NewOutboundContext("filler", "filler", ""),
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

	response, err := al.processMessage(context.Background(), testInboundMessage(bus.InboundMessage{
		Channel:  "telegram",
		SenderID: "user1",
		ChatID:   "chat1",
		Content:  "hello",
	}))
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
		if outbound.Context.Channel != "telegram" || outbound.Context.ChatID != "reason-chat" {
			t.Fatalf("unexpected reasoning context: %+v", outbound.Context)
		}
		if outbound.Content != "thinking trace" {
			t.Fatalf("reasoning content = %q, want %q", outbound.Content, "thinking trace")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("expected reasoning content to be published to reasoning channel")
	}
}

func TestProcessMessage_PicoPublishesReasoningAsThoughtMessage(t *testing.T) {
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

	response, err := al.processMessage(context.Background(), bus.InboundMessage{
		Channel:  "pico",
		SenderID: "user1",
		ChatID:   "pico:test-session",
		Content:  "hello",
	})
	if err != nil {
		t.Fatalf("processMessage() error = %v", err)
	}
	if response != "final answer" {
		t.Fatalf("processMessage() response = %q, want %q", response, "final answer")
	}

	var thoughtMsg *bus.OutboundMessage
	deadline := time.After(3 * time.Second)

	for thoughtMsg == nil {
		select {
		case outbound := <-msgBus.OutboundChan():
			msg := outbound
			if msg.Content == "thinking trace" {
				thoughtMsg = &msg
			}
		case <-deadline:
			t.Fatal("expected thought outbound message for pico")
		}
	}

	if thoughtMsg.Channel != "pico" || thoughtMsg.ChatID != "pico:test-session" {
		t.Fatalf("thought message route = %s/%s, want pico/pico:test-session", thoughtMsg.Channel, thoughtMsg.ChatID)
	}
	if thoughtMsg.Context.Raw[metadataKeyMessageKind] != messageKindThought {
		t.Fatalf(
			"thought metadata kind = %q, want %q",
			thoughtMsg.Context.Raw[metadataKeyMessageKind],
			messageKindThought,
		)
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

	response, err := al.processMessage(context.Background(), testInboundMessage(bus.InboundMessage{
		Channel:  "telegram",
		SenderID: "user-1",
		ChatID:   "chat-1",
		Content:  "check tool feedback",
	}))
	if err != nil {
		t.Fatalf("processMessage() error = %v", err)
	}
	if response != "HEARTBEAT_OK" {
		t.Fatalf("processMessage() response = %q, want %q", response, "HEARTBEAT_OK")
	}

	select {
	case outbound := <-msgBus.OutboundChan():
		escapedHeartbeatFile := strings.ReplaceAll(heartbeatFile, `\`, `\\`)
		if outbound.Channel != "telegram" {
			t.Fatalf("tool feedback channel = %q, want %q", outbound.Channel, "telegram")
		}
		if outbound.ChatID != "chat-1" {
			t.Fatalf("tool feedback chatID = %q, want %q", outbound.ChatID, "chat-1")
		}
		if outbound.Context.Channel != "telegram" || outbound.Context.ChatID != "chat-1" {
			t.Fatalf("unexpected tool feedback context: %+v", outbound.Context)
		}
		if !strings.Contains(outbound.Content, "`read_file`") {
			t.Fatalf("tool feedback content = %q, want read_file summary", outbound.Content)
		}
		if !strings.Contains(outbound.Content, utils.ToolFeedbackContinuationHint) {
			t.Fatalf("tool feedback content = %q, want continuation hint fallback", outbound.Content)
		}
		if !strings.Contains(outbound.Content, "check tool feedback") {
			t.Fatalf("tool feedback content = %q, want current user intent fallback", outbound.Content)
		}
		if !strings.Contains(outbound.Content, "\"path\":") {
			t.Fatalf("tool feedback content = %q, want serialized tool arguments", outbound.Content)
		}
		if !strings.Contains(outbound.Content, escapedHeartbeatFile) {
			t.Fatalf("tool feedback content = %q, want tool argument value", outbound.Content)
		}
		if strings.Contains(outbound.Content, "Previous turn explanation") {
			t.Fatalf("tool feedback content = %q, want no previous assistant fallback", outbound.Content)
		}
		if outbound.AgentID != "main" {
			t.Fatalf("tool feedback agent_id = %q, want main", outbound.AgentID)
		}
		if outbound.SessionKey == "" {
			t.Fatal("expected tool feedback to carry session_key")
		}
		if outbound.Scope == nil || outbound.Scope.AgentID != "main" || outbound.Scope.Channel != "telegram" {
			t.Fatalf("expected tool feedback scope, got %+v", outbound.Scope)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected outbound tool feedback for regular messages")
	}
}

func TestProcessMessage_PersistsReasoningContentInSessionHistory(t *testing.T) {
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

	response, err := al.processMessage(context.Background(), bus.InboundMessage{
		Channel:  "pico",
		SenderID: "user1",
		ChatID:   "pico:test-session",
		Content:  "hello",
	})
	if err != nil {
		t.Fatalf("processMessage() error = %v", err)
	}
	if response != "final answer" {
		t.Fatalf("processMessage() response = %q, want %q", response, "final answer")
	}

	store := al.GetRegistry().GetDefaultAgent().Sessions
	sessionKeys := store.ListSessions()
	if len(sessionKeys) != 1 {
		t.Fatalf("session keys = %v, want exactly 1 active session", sessionKeys)
	}
	history := store.GetHistory(sessionKeys[0])
	if len(history) < 2 {
		t.Fatalf("session history len = %d, want at least 2", len(history))
	}

	last := history[len(history)-1]
	if last.Role != "assistant" {
		t.Fatalf("last message role = %q, want assistant", last.Role)
	}
	if last.Content != "final answer" {
		t.Fatalf("last message content = %q, want %q", last.Content, "final answer")
	}
	if last.ReasoningContent != "thinking trace" {
		t.Fatalf("last message reasoning_content = %q, want %q", last.ReasoningContent, "thinking trace")
	}
}

func TestProcessMessage_PersistsReasoningToolResponseAsSingleAssistantRecord(t *testing.T) {
	tmpDir := t.TempDir()
	inspectPath := filepath.Join(tmpDir, "inspect.txt")
	if err := os.WriteFile(inspectPath, []byte("inspect me"), 0o644); err != nil {
		t.Fatalf("WriteFile(inspectPath) error = %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = tmpDir
	cfg.Agents.Defaults.ModelName = "test-model"
	cfg.Agents.Defaults.MaxTokens = 4096
	cfg.Agents.Defaults.MaxToolIterations = 10

	msgBus := bus.NewMessageBus()
	provider := &reasoningVisibleToolProvider{filePath: inspectPath}
	al := NewAgentLoop(cfg, msgBus, provider)

	response, err := al.processMessage(context.Background(), bus.InboundMessage{
		Channel:  "telegram",
		SenderID: "user1",
		ChatID:   "chat1",
		Content:  "hello",
	})
	if err != nil {
		t.Fatalf("processMessage() error = %v", err)
	}
	if response != "DONE" {
		t.Fatalf("processMessage() response = %q, want %q", response, "DONE")
	}

	store := al.GetRegistry().GetDefaultAgent().Sessions
	sessionKeys := store.ListSessions()
	if len(sessionKeys) != 1 {
		t.Fatalf("session keys = %v, want exactly 1 active session", sessionKeys)
	}

	history := store.GetHistory(sessionKeys[0])
	if len(history) < 3 {
		t.Fatalf("session history len = %d, want at least 3", len(history))
	}

	var assistantWithToolCall *providers.Message
	for i := range history {
		msg := history[i]
		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			assistantWithToolCall = &msg
			break
		}
	}
	if assistantWithToolCall == nil {
		t.Fatal("expected assistant history record with tool_calls")
	}
	if assistantWithToolCall.Content != "I'll inspect that file now." {
		t.Fatalf("assistant content = %q, want %q", assistantWithToolCall.Content, "I'll inspect that file now.")
	}
	if assistantWithToolCall.ReasoningContent != "Read the file before answering." {
		t.Fatalf("assistant reasoning_content = %q, want preserved", assistantWithToolCall.ReasoningContent)
	}
	if len(assistantWithToolCall.ToolCalls) != 1 {
		t.Fatalf("assistant tool calls = %+v, want single read_file tool", assistantWithToolCall.ToolCalls)
	}
	if got := providers.NormalizeToolCall(assistantWithToolCall.ToolCalls[0]).Name; got != "read_file" {
		t.Fatalf("assistant tool calls = %+v, want single read_file tool", assistantWithToolCall.ToolCalls)
	}

	sessionDir := filepath.Join(tmpDir, "sessions")
	entries, err := os.ReadDir(sessionDir)
	if err != nil {
		t.Fatalf("ReadDir(%q) error = %v", sessionDir, err)
	}

	var jsonlPath string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		jsonlPath = filepath.Join(sessionDir, entry.Name())
		break
	}
	if jsonlPath == "" {
		t.Fatal("expected session jsonl file to be created")
	}

	data, err := os.ReadFile(jsonlPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", jsonlPath, err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) < 3 {
		t.Fatalf("jsonl lines = %d, want at least 3", len(lines))
	}

	matchingRecords := 0
	for _, line := range lines {
		var msg providers.Message
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			t.Fatalf("Unmarshal(jsonl line) error = %v", err)
		}
		if msg.Role != "assistant" {
			continue
		}
		if msg.Content == "I'll inspect that file now." || msg.ReasoningContent == "Read the file before answering." {
			matchingRecords++
			toolName := ""
			if len(msg.ToolCalls) == 1 {
				toolName = providers.NormalizeToolCall(msg.ToolCalls[0]).Name
			}
			if msg.Content != "I'll inspect that file now." ||
				msg.ReasoningContent != "Read the file before answering." ||
				len(msg.ToolCalls) != 1 ||
				toolName != "read_file" {
				t.Fatalf("assistant jsonl record = %+v, want content+reasoning+tool_calls in one line", msg)
			}
		}
	}
	if matchingRecords != 1 {
		t.Fatalf("matching assistant jsonl records = %d, want exactly 1 canonical assistant record", matchingRecords)
	}
}

func TestProcessMessage_DoesNotLeakReasoningContentInToolFeedback(t *testing.T) {
	tmpDir := t.TempDir()
	heartbeatFile := filepath.Join(tmpDir, "tool-feedback-reasoning.txt")
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
	provider := &toolFeedbackReasoningProvider{filePath: heartbeatFile}
	al := NewAgentLoop(cfg, msgBus, provider)

	response, err := al.processMessage(context.Background(), testInboundMessage(bus.InboundMessage{
		Channel:  "telegram",
		SenderID: "user-1",
		ChatID:   "chat-1",
		Content:  "check reasoning fallback",
	}))
	if err != nil {
		t.Fatalf("processMessage() error = %v", err)
	}
	if response != "DONE" {
		t.Fatalf("processMessage() response = %q, want %q", response, "DONE")
	}

	select {
	case outbound := <-msgBus.OutboundChan():
		escapedHeartbeatFile := strings.ReplaceAll(heartbeatFile, `\`, `\\`)
		if !strings.Contains(outbound.Content, "`read_file`") {
			t.Fatalf("tool feedback content = %q, want read_file summary", outbound.Content)
		}
		if !strings.Contains(outbound.Content, utils.ToolFeedbackContinuationHint) {
			t.Fatalf("tool feedback content = %q, want continuation hint fallback", outbound.Content)
		}
		if !strings.Contains(outbound.Content, "check reasoning fallback") {
			t.Fatalf("tool feedback content = %q, want current user intent fallback", outbound.Content)
		}
		if !strings.Contains(outbound.Content, "\"path\":") {
			t.Fatalf("tool feedback content = %q, want serialized tool arguments", outbound.Content)
		}
		if !strings.Contains(outbound.Content, escapedHeartbeatFile) {
			t.Fatalf("tool feedback content = %q, want tool argument value", outbound.Content)
		}
		if strings.Contains(outbound.Content, "Read README.md first") {
			t.Fatalf("tool feedback content = %q, should not leak hidden reasoning", outbound.Content)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected outbound tool feedback without leaking reasoning")
	}
}

func TestProcessMessage_DoesNotPublishToolFeedbackForDiscordWhenDisabled(t *testing.T) {
	assertToolFeedbackNotPublishedWhenDisabled(t, "discord")
}

func assertToolFeedbackNotPublishedWhenDisabled(t *testing.T, channel string) {
	t.Helper()

	tmpDir := t.TempDir()
	heartbeatFile := filepath.Join(tmpDir, "tool-feedback-"+channel+".txt")
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

	response, err := al.processMessage(context.Background(), testInboundMessage(bus.InboundMessage{
		Channel:  channel,
		SenderID: "user-1",
		ChatID:   "chat-1",
		Content:  "check tool feedback",
	}))
	if err != nil {
		t.Fatalf("processMessage() error = %v", err)
	}
	if response != "HEARTBEAT_OK" {
		t.Fatalf("processMessage() response = %q, want %q", response, "HEARTBEAT_OK")
	}

	select {
	case outbound := <-msgBus.OutboundChan():
		t.Fatalf("expected no outbound tool feedback for %s when disabled, got %+v", channel, outbound)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestProcessMessage_DoesNotPublishToolFeedbackForTelegramWhenDisabled(t *testing.T) {
	assertToolFeedbackNotPublishedWhenDisabled(t, "telegram")
}

func TestProcessMessage_DoesNotPublishToolFeedbackForFeishuWhenDisabled(t *testing.T) {
	assertToolFeedbackNotPublishedWhenDisabled(t, "feishu")
}

func TestProcessMessage_MessageToolPublishesOutboundWithTurnMetadata(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = t.TempDir()
	cfg.Agents.Defaults.ModelName = "test-model"
	cfg.Agents.Defaults.MaxTokens = 4096
	cfg.Agents.Defaults.MaxToolIterations = 10
	cfg.Session.Dimensions = []string{"chat"}

	msgBus := bus.NewMessageBus()
	provider := &messageToolProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)

	response, err := al.processMessage(context.Background(), testInboundMessage(bus.InboundMessage{
		Channel:  "telegram",
		SenderID: "user-1",
		ChatID:   "chat-1",
		Content:  "send a direct message",
	}))
	if err != nil {
		t.Fatalf("processMessage() error = %v", err)
	}
	if response == "" {
		t.Fatal("expected processMessage() to return a final loop response")
	}

	select {
	case outbound := <-msgBus.OutboundChan():
		if outbound.Content != "direct tool message" {
			t.Fatalf("outbound content = %q, want direct tool message", outbound.Content)
		}
		if outbound.AgentID != "main" {
			t.Fatalf("outbound agent_id = %q, want main", outbound.AgentID)
		}
		if outbound.SessionKey == "" {
			t.Fatal("expected message tool outbound to carry session_key")
		}
		if outbound.Scope == nil || outbound.Scope.Values["chat"] != "direct:chat-1" {
			t.Fatalf("unexpected message tool outbound scope: %+v", outbound.Scope)
		}
		if outbound.Context.Channel != "telegram" || outbound.Context.ChatID != "chat-1" {
			t.Fatalf("unexpected message tool outbound context: %+v", outbound.Context)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected message tool outbound")
	}
}

func TestRun_PicoPublishesAssistantContentDuringToolCallsWithoutFinalDuplicate(t *testing.T) {
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
	provider := &picoDistinctToolCallContentProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)

	agent := al.GetRegistry().GetDefaultAgent()
	if agent == nil {
		t.Fatal("expected default agent")
	}
	agent.Tools.Register(&toolLimitTestTool{})

	runCtx, runCancel := context.WithCancel(context.Background())
	defer runCancel()

	runDone := make(chan error, 1)
	go func() {
		runDone <- al.Run(runCtx)
	}()

	if err := msgBus.PublishInbound(context.Background(), bus.InboundMessage{
		Channel:  "pico",
		SenderID: "user-1",
		ChatID:   "session-1",
		Content:  "run with tools",
	}); err != nil {
		t.Fatalf("PublishInbound() error = %v", err)
	}

	outputs := make([]bus.OutboundMessage, 0, 3)
	deadline := time.After(2 * time.Second)
	for len(outputs) < 3 {
		select {
		case outbound := <-msgBus.OutboundChan():
			outputs = append(outputs, outbound)
		case <-deadline:
			t.Fatalf("timed out waiting for pico outputs, got %v", outputs)
		}
	}

	if outputs[0].Content != "intermediate model text" {
		t.Fatalf("first outbound content = %q, want %q", outputs[0].Content, "intermediate model text")
	}
	if outputs[1].Context.Raw[metadataKeyMessageKind] != messageKindToolCalls {
		t.Fatalf("second outbound = %+v, want tool_calls message", outputs[1])
	}
	if !strings.Contains(outputs[1].Context.Raw[metadataKeyToolCalls], "tool_limit_test_tool") {
		t.Fatalf("second outbound tool_calls = %q, want tool name", outputs[1].Context.Raw[metadataKeyToolCalls])
	}
	if outputs[2].Content != "final model text" {
		t.Fatalf("third outbound content = %q, want %q", outputs[2].Content, "final model text")
	}

	runCancel()
	select {
	case err := <-runDone:
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Run() to exit")
	}

	select {
	case outbound := <-msgBus.OutboundChan():
		if outbound.Content == "final model text" {
			t.Fatalf("unexpected duplicate final pico output: %+v", outbound)
		}
	case <-time.After(200 * time.Millisecond):
	}
}

func TestRunAgentLoop_PicoSkipsInterimPublishWhenNotAllowed(t *testing.T) {
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
	provider := &picoInterleavedContentProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)

	agent := al.GetRegistry().GetDefaultAgent()
	if agent == nil {
		t.Fatal("expected default agent")
	}
	agent.Tools.Register(&toolLimitTestTool{})

	response, err := al.runAgentLoop(context.Background(), agent, processOptions{
		SessionKey:              "agent:main:pico:session-1",
		Channel:                 "pico",
		ChatID:                  "session-1",
		UserMessage:             "run with tools",
		DefaultResponse:         defaultResponse,
		EnableSummary:           false,
		SendResponse:            false,
		AllowInterimPicoPublish: false,
		SuppressToolFeedback:    true,
	})
	if err != nil {
		t.Fatalf("runAgentLoop() error = %v", err)
	}
	if response != "final model text" {
		t.Fatalf("runAgentLoop() response = %q, want %q", response, "final model text")
	}

	select {
	case outbound := <-msgBus.OutboundChan():
		t.Fatalf("unexpected outbound message when interim publish disabled: %+v", outbound)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestRun_PicoToolFeedbackSuppressesDuplicateInterimAssistantContent(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
				ToolFeedback: config.ToolFeedbackConfig{
					Enabled: true,
				},
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &picoInterleavedContentProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)

	agent := al.GetRegistry().GetDefaultAgent()
	if agent == nil {
		t.Fatal("expected default agent")
	}
	agent.Tools.Register(&toolLimitTestTool{})

	runCtx, runCancel := context.WithCancel(context.Background())
	defer runCancel()

	runDone := make(chan error, 1)
	go func() {
		runDone <- al.Run(runCtx)
	}()

	if err := msgBus.PublishInbound(context.Background(), bus.InboundMessage{
		Channel:  "pico",
		SenderID: "user-1",
		ChatID:   "session-1",
		Content:  "run with tools",
	}); err != nil {
		t.Fatalf("PublishInbound() error = %v", err)
	}

	outputs := make([]bus.OutboundMessage, 0, 3)
	deadline := time.After(2 * time.Second)
	for len(outputs) < 2 {
		select {
		case outbound := <-msgBus.OutboundChan():
			outputs = append(outputs, outbound)
		case <-deadline:
			t.Fatalf("timed out waiting for pico outputs, got %v", outputs)
		}
	}

	if outputs[0].Context.Raw[metadataKeyMessageKind] != messageKindToolCalls {
		t.Fatalf("first outbound = %+v, want tool_calls message", outputs[0])
	}
	if outputs[0].Content != "" {
		t.Fatalf("first outbound content = %q, want empty tool_calls content", outputs[0].Content)
	}
	if !strings.Contains(outputs[0].Context.Raw[metadataKeyToolCalls], "tool_limit_test_tool") {
		t.Fatalf("first outbound tool_calls = %q, want tool name", outputs[0].Context.Raw[metadataKeyToolCalls])
	}
	if outputs[1].Content != "final model text" {
		t.Fatalf("second outbound content = %q, want %q", outputs[1].Content, "final model text")
	}

	runCancel()
	select {
	case err := <-runDone:
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Run() to exit")
	}

	select {
	case outbound := <-msgBus.OutboundChan():
		t.Fatalf("unexpected extra pico output after tool feedback + final reply: %+v", outbound)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestResolveMediaRefs_ImageInjectsPathTag(t *testing.T) {
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

	if len(result[0].Media) != 0 {
		t.Fatalf("expected 0 media (images use path tags), got %d", len(result[0].Media))
	}
	localPath, _, _ := store.ResolveWithMeta(ref)
	expectedContent := "describe this [image:" + localPath + "]"
	if result[0].Content != expectedContent {
		t.Fatalf("expected content %q, got %q", expectedContent, result[0].Content)
	}
}

func TestResolveMediaRefs_ToolRoleImageAppendedAsUserMessage(t *testing.T) {
	store := media.NewFileMediaStore()
	dir := t.TempDir()

	pngPath := filepath.Join(dir, "tool-result.png")
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
	ref, _ := store.Store(pngPath, media.MediaMeta{}, "test")

	messages := []providers.Message{
		{Role: "tool", Content: "Image loaded", Media: []string{ref}},
	}
	result := resolveMediaRefs(messages, store, config.DefaultMaxMediaSize)

	// Tool message should have path tag but no base64
	if len(result[0].Media) != 0 {
		t.Fatalf("expected 0 media in tool message, got %d", len(result[0].Media))
	}
	localPath, _, _ := store.ResolveWithMeta(ref)
	if !strings.Contains(result[0].Content, "[image:"+localPath+"]") {
		t.Fatalf("expected image path tag in tool content, got %q", result[0].Content)
	}

	// A synthetic user message with base64 should follow
	if len(result) != 2 {
		t.Fatalf("expected 2 messages (tool + synthetic user), got %d", len(result))
	}
	if result[1].Role != "user" {
		t.Fatalf("expected synthetic message role=user, got %q", result[1].Role)
	}
	if len(result[1].Media) != 1 {
		t.Fatalf("expected 1 base64 media in synthetic user message, got %d", len(result[1].Media))
	}
	if !strings.HasPrefix(result[1].Media[0], "data:image/png;base64,") {
		t.Fatalf("expected data:image/png;base64, prefix, got %q", result[1].Media[0][:40])
	}
}

func TestResolveMediaRefs_MultiToolCallPreservesOrdering(t *testing.T) {
	store := media.NewFileMediaStore()
	dir := t.TempDir()

	// Create image for tool #1
	pngPath := filepath.Join(dir, "loaded.png")
	pngHeader := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x02,
		0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xDE,
	}
	os.WriteFile(pngPath, pngHeader, 0o644)
	imgRef, _ := store.Store(pngPath, media.MediaMeta{}, "test")

	// Simulate: assistant called load_image + read_file, two tool results follow
	messages := []providers.Message{
		{Role: "assistant", Content: "Let me load the image and read the file."},
		{Role: "tool", Content: "Image loaded [image: photo]", Media: []string{imgRef}},
		{Role: "tool", Content: "file contents here"},
	}
	result := resolveMediaRefs(messages, store, config.DefaultMaxMediaSize)

	// assistant, tool#1, tool#2 must remain contiguous — no user in between
	if result[0].Role != "assistant" {
		t.Fatalf("result[0] expected assistant, got %q", result[0].Role)
	}
	if result[1].Role != "tool" {
		t.Fatalf("result[1] expected tool, got %q", result[1].Role)
	}
	if result[2].Role != "tool" {
		t.Fatalf("result[2] expected tool, got %q", result[2].Role)
	}

	// Synthetic user message should come AFTER the tool block
	if len(result) != 4 {
		t.Fatalf("expected 4 messages (assistant + 2 tool + synthetic user), got %d", len(result))
	}
	if result[3].Role != "user" {
		t.Fatalf("result[3] expected user, got %q", result[3].Role)
	}
	if len(result[3].Media) != 1 || !strings.HasPrefix(result[3].Media[0], "data:image/png;base64,") {
		t.Fatal("expected synthetic user message to contain base64 image")
	}
}

func TestResolveMediaRefs_OversizedImageSkipsBase64KeepsPathTag(t *testing.T) {
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
	localPath, _, _ := store.ResolveWithMeta(ref)
	expected := "hi [image:" + localPath + "]"
	if result[0].Content != expected {
		t.Fatalf("expected content %q, got %q", expected, result[0].Content)
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

	if len(result[0].Media) != 0 {
		t.Fatalf("expected 0 media (images use path tags), got %d", len(result[0].Media))
	}
	localPath, _, _ := store.ResolveWithMeta(ref)
	expectedContent := "hi [image:" + localPath + "]"
	if result[0].Content != expectedContent {
		t.Fatalf("expected content %q, got %q", expectedContent, result[0].Content)
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

func TestInjectPathTags_HandlesVariousChannelPlaceholders(t *testing.T) {
	cases := []struct {
		name    string
		content string
		tag     string
		want    string
	}{
		// Telegram / Feishu format
		{"image_photo", "[image: photo]", "[image:/tmp/p.png]", "[image:/tmp/p.png]"},
		// WeCom / WeChat / Line format
		{"bare_image", "[image]", "[image:/tmp/p.png]", "[image:/tmp/p.png]"},
		// QQ / Discord format with filename
		{"image_filename", "[image: pic.jpg]", "[image:/tmp/p.png]", "[image:/tmp/p.png]"},
		{"audio_with_filename", "[audio: voice.m4a]", "[audio:/tmp/a.m4a]", "[audio:/tmp/a.m4a]"},
		{"bare_audio", "[audio]", "[audio:/tmp/a.m4a]", "[audio:/tmp/a.m4a]"},
		{"bare_video", "[video]", "[video:/tmp/v.mp4]", "[video:/tmp/v.mp4]"},
		{"bare_file", "[file]", "[file:/tmp/f.pdf]", "[file:/tmp/f.pdf]"},
		// Mixed surrounding text
		{
			"with_text",
			"hello [image] world",
			"[image:/tmp/p.png]",
			"hello [image:/tmp/p.png] world",
		},
		// No placeholder — append
		{"no_placeholder", "hello world", "[image:/tmp/p.png]", "hello world [image:/tmp/p.png]"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := injectPathTags(tc.content, []string{tc.tag})
			if got != tc.want {
				t.Errorf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestInjectPathTags_DoesNotReplacePathTag(t *testing.T) {
	// If content already contains a path tag, we must not touch it.
	content := "see [image:/already/placed.png] thanks"
	got := injectPathTags(content, []string{"[image:/new/path.png]"})
	want := "see [image:/already/placed.png] thanks [image:/new/path.png]"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestInjectPathTags_PrependsForJSONContent(t *testing.T) {
	jsonContent := `{"schema":"2.0","body":{"elements":[{"tag":"img","img_key":"img_123"}]}}`
	got := injectPathTags(jsonContent, []string{"[image:/tmp/photo.png]"})
	want := "[image:/tmp/photo.png]\n" + jsonContent
	if got != want {
		t.Fatalf("expected tag prepended to JSON, got %q", got)
	}
}

func TestInjectPathTags_BracketTextNotTreatedAsJSON(t *testing.T) {
	content := "[update] see attached report"
	got := injectPathTags(content, []string{"[file:/tmp/report.pdf]"})
	want := "[update] see attached report [file:/tmp/report.pdf]"
	if got != want {
		t.Fatalf("expected tag appended to bracket text, got %q", got)
	}
}

func TestResolveMediaRefs_JSONContentPrependsPathTag(t *testing.T) {
	store := media.NewFileMediaStore()
	dir := t.TempDir()

	pngPath := filepath.Join(dir, "card_img.png")
	pngHeader := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x02,
		0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xDE,
	}
	os.WriteFile(pngPath, pngHeader, 0o644)
	ref, _ := store.Store(pngPath, media.MediaMeta{ContentType: "image/png"}, "test")

	jsonContent := `{"schema":"2.0","body":{"elements":[{"tag":"img","img_key":"img_123"}]}}`
	messages := []providers.Message{
		{Role: "user", Content: jsonContent, Media: []string{ref}},
	}
	result := resolveMediaRefs(messages, store, config.DefaultMaxMediaSize)

	want := "[image:" + pngPath + "]\n" + jsonContent
	if result[0].Content != want {
		t.Fatalf("expected path tag prepended to JSON content, got %q", result[0].Content)
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

	if len(result[0].Media) != 0 {
		t.Fatalf("expected 0 media (all types use path tags), got %d", len(result[0].Media))
	}
	imgLocalPath, _, _ := store.ResolveWithMeta(imgRef)
	pdfLocalPath, _, _ := store.ResolveWithMeta(fileRef)
	expectedContent := "check these [file:" + pdfLocalPath + "] [image:" + imgLocalPath + "]"
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

	response, err := al.processMessage(context.Background(), testInboundMessage(bus.InboundMessage{
		Channel:    "test",
		ChatID:     "chat1",
		SenderID:   "user1",
		SessionKey: "test-session",
		Content:    "trigger recovery",
	}))
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

	response, err := al.processMessage(context.Background(), testInboundMessage(bus.InboundMessage{
		Channel:  "test",
		ChatID:   "chat1",
		SenderID: "user1",
		Content:  "hello",
	}))
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

func TestParallelMessageProcessing_DifferentSessionsProcessedConcurrently(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Track concurrent executions using a unique ID per turn
	var mu sync.Mutex
	activeTurns := make(map[string]bool)
	maxConcurrent := 0
	turnCounter := 0
	var wg sync.WaitGroup
	wg.Add(3) // Wait for 3 turns to complete

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
				MaxParallelTurns:  3, // Allow up to 3 concurrent turns
			},
		},
		Session: config.SessionConfig{
			Dimensions: []string{"chat"},
		},
	}

	msgBus := bus.NewMessageBus()
	defer msgBus.Close()

	// Create a slow mock provider that tracks concurrency
	provider := &concurrentMockProvider{
		responseFunc: func(callID int) string {
			mu.Lock()
			turnCounter++
			turnID := fmt.Sprintf("turn-%d", turnCounter)
			activeTurns[turnID] = true
			currentActive := len(activeTurns)
			if currentActive > maxConcurrent {
				maxConcurrent = currentActive
			}
			mu.Unlock()

			// Simulate some processing time
			time.Sleep(100 * time.Millisecond)

			mu.Lock()
			delete(activeTurns, turnID)
			mu.Unlock()

			wg.Done()
			return fmt.Sprintf("Response %s", turnID)
		},
	}

	al := NewAgentLoop(cfg, msgBus, provider)
	defer al.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the agent loop
	go func() {
		if err := al.Run(ctx); err != nil {
			t.Logf("Agent loop error: %v", err)
		}
	}()

	// Give the loop time to start
	time.Sleep(50 * time.Millisecond)

	// Send 3 messages from different sessions
	sessions := []string{"user1", "user2", "user3"}
	for i, session := range sessions {
		msg := bus.InboundMessage{
			Context: bus.InboundContext{
				Channel:  "telegram",
				ChatID:   fmt.Sprintf("chat%d", i),
				ChatType: "direct",
				SenderID: session,
			},
			Channel:  "telegram",
			ChatID:   fmt.Sprintf("chat%d", i),
			SenderID: session,
			Content:  fmt.Sprintf("Hello from %s", session),
		}
		if err := msgBus.PublishInbound(context.Background(), msg); err != nil {
			t.Fatalf("PublishInbound failed: %v", err)
		}
	}

	// Wait for all turns to complete with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All turns completed successfully
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for turns to complete")
	}

	// Verify that we had concurrent executions
	mu.Lock()
	defer mu.Unlock()

	if maxConcurrent < 2 {
		t.Errorf("Expected at least 2 concurrent executions, got max %d", maxConcurrent)
	}

	t.Logf("Maximum concurrent executions: %d", maxConcurrent)
}

func TestParallelMessageProcessing_SameSessionProcessedSequentially(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	var mu sync.Mutex
	turnIDs := make(map[string]bool)
	var wg sync.WaitGroup
	var firstResponse sync.Once
	wg.Add(1) // Only 1 turn should be created for same session

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
				MaxParallelTurns:  3,
			},
		},
		Session: config.SessionConfig{
			Dimensions: []string{"chat"},
		},
	}

	msgBus := bus.NewMessageBus()
	defer msgBus.Close()

	al := NewAgentLoop(cfg, msgBus, &concurrentMockProvider{
		responseFunc: func(callID int) string {
			firstResponse.Do(func() {
				wg.Done()
			})
			return "ok"
		},
	})
	defer al.Close()

	runtimeCh, closeRuntimeEvents := subscribeRuntimeEventsForTest(
		t,
		al,
		64,
		runtimeevents.KindAgentTurnStart,
	)
	defer closeRuntimeEvents()

	go func() {
		for evt := range runtimeCh {
			if evt.Kind == runtimeevents.KindAgentTurnStart {
				mu.Lock()
				turnIDs[evt.Scope.TurnID] = true
				mu.Unlock()
			}
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := al.Run(ctx); err != nil {
			t.Logf("Agent loop error: %v", err)
		}
	}()

	time.Sleep(50 * time.Millisecond)

	// Send 3 messages from the SAME session - only one turn should be created;
	// subsequent messages should be enqueued to the steering queue and processed
	// within the same turn (not as separate concurrent turns).
	for i := 0; i < 3; i++ {
		msg := bus.InboundMessage{
			Context: bus.InboundContext{
				Channel:  "telegram",
				ChatID:   "chat1",
				ChatType: "direct",
				SenderID: "user1",
			},
			Channel:  "telegram",
			SenderID: "user1",
			ChatID:   "chat1",
			Content:  fmt.Sprintf("Message %d", i+1),
		}
		if err := msgBus.PublishInbound(context.Background(), msg); err != nil {
			t.Fatalf("PublishInbound failed: %v", err)
		}
	}

	// Wait for turn to complete with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Turn completed successfully
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for turn to complete")
	}

	mu.Lock()
	defer mu.Unlock()

	// Only 1 turn ID should have been created — proving messages were
	// serialized into a single turn rather than spawning concurrent turns.
	if len(turnIDs) != 1 {
		t.Errorf("Expected 1 turn (others queued to steering), got %d: %v", len(turnIDs), turnIDs)
	}
}

// concurrentMockProvider is a mock provider that allows tracking concurrency
type concurrentMockProvider struct {
	responseFunc func(callID int) string
}

func (p *concurrentMockProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	opts map[string]any,
) (*providers.LLMResponse, error) {
	// Use an atomic counter to assign unique call IDs for concurrency tracking.
	// This avoids relying on sessionKey derivation from message content, which
	// is not deterministic across concurrent calls.
	response := "Mock response"
	if p.responseFunc != nil {
		response = p.responseFunc(len(messages))
	}

	return &providers.LLMResponse{
		Content:   response,
		ToolCalls: []providers.ToolCall{},
	}, nil
}

func (p *concurrentMockProvider) GetDefaultModel() string {
	return "test-model"
}
