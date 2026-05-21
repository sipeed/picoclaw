package agent

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

type turnProfileCaptureProvider struct {
	messages []providers.Message
	tools    []providers.ToolDefinition
}

func (p *turnProfileCaptureProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	opts map[string]any,
) (*providers.LLMResponse, error) {
	p.messages = append([]providers.Message(nil), messages...)
	p.tools = append([]providers.ToolDefinition(nil), tools...)
	return &providers.LLMResponse{Content: "profile response"}, nil
}

func (p *turnProfileCaptureProvider) GetDefaultModel() string {
	return "test-model"
}

func newTurnProfileAgentLoop(
	t *testing.T,
	cfg *config.Config,
	provider *turnProfileCaptureProvider,
) *AgentLoop {
	t.Helper()
	t.Setenv("PICOCLAW_BUILTIN_SKILLS", t.TempDir())
	if cfg.Agents.Defaults.Workspace == "" {
		cfg.Agents.Defaults.Workspace = t.TempDir()
	}
	if cfg.Agents.Defaults.ModelName == "" {
		cfg.Agents.Defaults.ModelName = "test-model"
	}
	if cfg.Agents.Defaults.MaxTokens == 0 {
		cfg.Agents.Defaults.MaxTokens = 4096
	}
	if cfg.Agents.Defaults.MaxToolIterations == 0 {
		cfg.Agents.Defaults.MaxToolIterations = 10
	}
	return NewAgentLoop(cfg, bus.NewMessageBus(), provider)
}

func writeTurnProfileSkill(t *testing.T, workspace, name, body string) {
	t.Helper()
	dir := filepath.Join(workspace, "skills", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}
}

func TestTurnProfile_DisabledPreservesDefaultHistoryAndPrompt(t *testing.T) {
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				TurnProfile: config.TurnProfileConfig{
					Enabled: false,
					History: config.TurnProfileBlock{
						Mode: config.TurnProfileModeOff,
					},
				},
			},
		},
	}
	provider := &turnProfileCaptureProvider{}
	al := newTurnProfileAgentLoop(t, cfg, provider)
	agent := al.GetRegistry().GetDefaultAgent()
	sessionKey := "agent:default:test-default"
	initialHistory := []providers.Message{
		{Role: "user", Content: "old user"},
		{Role: "assistant", Content: "old assistant"},
	}
	agent.Sessions.SetHistory(sessionKey, initialHistory)
	agent.Sessions.SetSummary(sessionKey, "old summary")

	got, err := al.runAgentLoop(context.Background(), agent, processOptions{
		SessionKey:      sessionKey,
		UserMessage:     "new user",
		DefaultResponse: defaultResponse,
		EnableSummary:   false,
	})
	if err != nil {
		t.Fatalf("runAgentLoop() error = %v", err)
	}
	if got != "profile response" {
		t.Fatalf("runAgentLoop() = %q, want profile response", got)
	}
	if len(provider.messages) != 4 {
		t.Fatalf("provider messages len = %d, want system + history + user", len(provider.messages))
	}
	if !reflect.DeepEqual(provider.messages[1:3], initialHistory) {
		t.Fatalf("provider history = %#v, want %#v", provider.messages[1:3], initialHistory)
	}
	if !strings.Contains(provider.messages[0].Content, "CONTEXT_SUMMARY") {
		t.Fatalf("system prompt missing summary in default mode:\n%s", provider.messages[0].Content)
	}
	history := agent.Sessions.GetHistory(sessionKey)
	if len(history) != len(initialHistory)+2 {
		t.Fatalf("history len = %d, want initial + user + assistant", len(history))
	}
}

func TestTurnProfile_HistoryOffSuppressesHistoryAndPersistence(t *testing.T) {
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				TurnProfile: config.TurnProfileConfig{
					Enabled: true,
					History: config.TurnProfileBlock{Mode: config.TurnProfileModeOff},
				},
			},
		},
	}
	provider := &turnProfileCaptureProvider{}
	al := newTurnProfileAgentLoop(t, cfg, provider)
	agent := al.GetRegistry().GetDefaultAgent()
	sessionKey := "agent:default:test-history-off"
	initialHistory := []providers.Message{
		{Role: "user", Content: "old user"},
		{Role: "assistant", Content: "old assistant"},
	}
	agent.Sessions.SetHistory(sessionKey, initialHistory)
	agent.Sessions.SetSummary(sessionKey, "old summary")

	_, err := al.runAgentLoop(context.Background(), agent, processOptions{
		SessionKey:      sessionKey,
		UserMessage:     "new user",
		DefaultResponse: defaultResponse,
		EnableSummary:   true,
	})
	if err != nil {
		t.Fatalf("runAgentLoop() error = %v", err)
	}
	if len(provider.messages) != 2 {
		t.Fatalf("provider messages len = %d, want system + current user", len(provider.messages))
	}
	if provider.messages[1].Content != "new user" {
		t.Fatalf("current message = %q, want new user", provider.messages[1].Content)
	}
	if strings.Contains(provider.messages[0].Content, "old summary") {
		t.Fatalf("system prompt includes suppressed summary:\n%s", provider.messages[0].Content)
	}
	history := agent.Sessions.GetHistory(sessionKey)
	if !reflect.DeepEqual(history, initialHistory) {
		t.Fatalf("history = %#v, want unchanged %#v", history, initialHistory)
	}
}

func TestTurnProfile_ProcessMessageUsesEnabledTurnProfile(t *testing.T) {
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				TurnProfile: config.TurnProfileConfig{
					Enabled: true,
					History: config.TurnProfileBlock{Mode: config.TurnProfileModeOff},
				},
			},
		},
	}
	provider := &turnProfileCaptureProvider{}
	al := newTurnProfileAgentLoop(t, cfg, provider)

	_, err := al.processMessage(context.Background(), bus.InboundMessage{
		Context: bus.InboundContext{
			Channel:  "pico",
			ChatID:   "pico:sess-1",
			ChatType: "direct",
			SenderID: "pico-user",
		},
		Content: "hello from pico",
	})
	if err != nil {
		t.Fatalf("processMessage() error = %v", err)
	}
	if len(provider.messages) != 2 {
		t.Fatalf("provider messages len = %d, want system + current user", len(provider.messages))
	}
	if provider.messages[1].Content != "hello from pico" {
		t.Fatalf("current message = %q, want hello from pico", provider.messages[1].Content)
	}
}

func TestTurnProfile_SystemPromptOffUsesExternalPromptOnly(t *testing.T) {
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				TurnProfile: config.TurnProfileConfig{
					Enabled:      true,
					History:      config.TurnProfileBlock{Mode: config.TurnProfileModeOff},
					SystemPrompt: config.TurnProfileBlock{Mode: config.TurnProfileModeOff},
					Skills:       config.TurnProfileBlock{Mode: config.TurnProfileModeOff},
					Tools:        config.TurnProfileBlock{Mode: config.TurnProfileModeOff},
				},
			},
		},
	}
	provider := &turnProfileCaptureProvider{}
	al := newTurnProfileAgentLoop(t, cfg, provider)
	agent := al.GetRegistry().GetDefaultAgent()

	_, err := al.runAgentLoop(context.Background(), agent, processOptions{
		SessionKey:           "agent:default:test-external-prompt",
		UserMessage:          "hello",
		DefaultResponse:      defaultResponse,
		SystemPromptOverride: "External prompt only.",
	})
	if err != nil {
		t.Fatalf("runAgentLoop() error = %v", err)
	}
	if len(provider.messages) != 2 {
		t.Fatalf("messages len = %d, want system + user", len(provider.messages))
	}
	if strings.TrimSpace(provider.messages[0].Content) != "External prompt only." {
		t.Fatalf("system prompt = %q, want external only", provider.messages[0].Content)
	}
}

func TestTurnProfile_SystemPromptOffAddsToolFallbackWhenToolsVisible(t *testing.T) {
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				TurnProfile: config.TurnProfileConfig{
					Enabled:      true,
					History:      config.TurnProfileBlock{Mode: config.TurnProfileModeOff},
					SystemPrompt: config.TurnProfileBlock{Mode: config.TurnProfileModeOff},
					Skills:       config.TurnProfileBlock{Mode: config.TurnProfileModeOff},
					Tools: config.TurnProfileBlock{
						Mode:  config.TurnProfileModeCustom,
						Allow: []string{"web_search"},
					},
				},
			},
		},
	}
	provider := &turnProfileCaptureProvider{}
	al := newTurnProfileAgentLoop(t, cfg, provider)
	agent := al.GetRegistry().GetDefaultAgent()

	_, err := al.runAgentLoop(context.Background(), agent, processOptions{
		SessionKey:      "agent:default:test-tool-fallback",
		UserMessage:     "hello",
		DefaultResponse: defaultResponse,
	})
	if err != nil {
		t.Fatalf("runAgentLoop() error = %v", err)
	}
	if got := strings.TrimSpace(provider.messages[0].Content); got != toolUseSystemPromptRule() {
		t.Fatalf("fallback prompt = %q, want existing tool rule %q", got, toolUseSystemPromptRule())
	}
}

func TestTurnProfile_SkillsOffAndCustomControlCatalogAndActiveSkills(t *testing.T) {
	workspace := t.TempDir()
	writeTurnProfileSkill(
		t,
		workspace,
		"shell",
		"---\ndescription: shell skill\n---\n# shell\n\nUse shell carefully.",
	)
	writeTurnProfileSkill(
		t,
		workspace,
		"paint",
		"---\ndescription: paint skill\n---\n# paint\n\nUse paint vividly.",
	)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace: workspace,
				TurnProfile: config.TurnProfileConfig{
					Enabled: true,
					History: config.TurnProfileBlock{
						Mode: config.TurnProfileModeOff,
					},
					Skills: config.TurnProfileBlock{
						Mode: config.TurnProfileModeOff,
					},
				},
			},
		},
	}
	provider := &turnProfileCaptureProvider{}
	al := newTurnProfileAgentLoop(t, cfg, provider)
	agent := al.GetRegistry().GetDefaultAgent()

	_, err := al.runAgentLoop(context.Background(), agent, processOptions{
		SessionKey:      "agent:default:test-skills-off",
		UserMessage:     "hello",
		DefaultResponse: defaultResponse,
		ForcedSkills:    []string{"shell"},
	})
	if err != nil {
		t.Fatalf("runAgentLoop(no-skills) error = %v", err)
	}
	noSkillsPrompt := provider.messages[0].Content
	if strings.Contains(noSkillsPrompt, "# Skills") ||
		strings.Contains(noSkillsPrompt, "# Active Skills") {
		t.Fatalf("skills-off prompt includes skill context:\n%s", noSkillsPrompt)
	}

	cfg.Agents.Defaults.TurnProfile.Skills = config.TurnProfileBlock{
		Mode:  config.TurnProfileModeCustom,
		Allow: []string{"shell"},
	}
	provider = &turnProfileCaptureProvider{}
	al = newTurnProfileAgentLoop(t, cfg, provider)
	agent = al.GetRegistry().GetDefaultAgent()

	_, err = al.runAgentLoop(context.Background(), agent, processOptions{
		SessionKey:      "agent:default:test-skills-custom",
		UserMessage:     "hello",
		DefaultResponse: defaultResponse,
		ForcedSkills:    []string{"shell", "paint"},
	})
	if err != nil {
		t.Fatalf("runAgentLoop(shell-only) error = %v", err)
	}
	customPrompt := provider.messages[0].Content
	if !strings.Contains(customPrompt, "<name>shell</name>") ||
		!strings.Contains(customPrompt, "### Skill: shell") {
		t.Fatalf("custom skills prompt missing allowed shell context:\n%s", customPrompt)
	}
	if strings.Contains(customPrompt, "<name>paint</name>") ||
		strings.Contains(customPrompt, "### Skill: paint") {
		t.Fatalf("custom skills prompt includes disallowed paint context:\n%s", customPrompt)
	}
}

type turnProfileAddToolHook struct{}

func (h turnProfileAddToolHook) BeforeLLM(
	ctx context.Context,
	req *LLMHookRequest,
) (*LLMHookRequest, HookDecision, error) {
	next := req.Clone()
	next.Tools = append(next.Tools, providers.ToolDefinition{
		Type: "function",
		Function: providers.ToolFunctionDefinition{
			Name:        "echo_text_rewritten",
			Description: "hook-added tool",
			Parameters:  map[string]any{"type": "object"},
		},
	})
	return next, HookDecision{Action: HookActionModify}, nil
}

type turnProfileToolCallProvider struct {
	calls    int
	messages []providers.Message
}

func (p *turnProfileToolCallProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	opts map[string]any,
) (*providers.LLMResponse, error) {
	p.calls++
	p.messages = append([]providers.Message(nil), messages...)
	if p.calls == 1 {
		return &providers.LLMResponse{
			Content: "calling disallowed",
			ToolCalls: []providers.ToolCall{{
				ID:        "call_1",
				Name:      "echo_text_rewritten",
				Arguments: map[string]any{"text": "blocked"},
			}},
			FinishReason: "tool_calls",
		}, nil
	}
	return &providers.LLMResponse{Content: "done"}, nil
}

func (p *turnProfileToolCallProvider) GetDefaultModel() string {
	return "test-model"
}

func TestTurnProfile_ToolsCustomFiltersProviderToolsAndHookAdditions(t *testing.T) {
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				TurnProfile: config.TurnProfileConfig{
					Enabled: true,
					History: config.TurnProfileBlock{Mode: config.TurnProfileModeOff},
					Tools: config.TurnProfileBlock{
						Mode:  config.TurnProfileModeCustom,
						Allow: []string{"echo_text"},
					},
				},
			},
		},
	}
	provider := &turnProfileCaptureProvider{}
	al := newTurnProfileAgentLoop(t, cfg, provider)
	al.RegisterTool(&echoTextTool{})
	al.RegisterTool(&echoTextRewrittenTool{})
	if err := al.MountHook(NamedHook("add-disallowed-tool", turnProfileAddToolHook{})); err != nil {
		t.Fatalf("MountHook() error = %v", err)
	}

	_, err := al.runAgentLoop(
		context.Background(),
		al.GetRegistry().GetDefaultAgent(),
		processOptions{
			SessionKey:      "agent:default:test-tools-filter",
			UserMessage:     "hello",
			DefaultResponse: defaultResponse,
		},
	)
	if err != nil {
		t.Fatalf("runAgentLoop() error = %v", err)
	}
	if len(provider.tools) != 1 {
		t.Fatalf("provider tools len = %d, want 1: %#v", len(provider.tools), provider.tools)
	}
	if provider.tools[0].Function.Name != "echo_text" {
		t.Fatalf("provider tool = %q, want echo_text", provider.tools[0].Function.Name)
	}
}

func TestTurnProfile_ToolsOffDisablesProviderAndNativeSearchTools(t *testing.T) {
	cfg := &config.Config{
		Tools: config.ToolsConfig{
			Web: config.WebToolsConfig{
				ToolConfig:   config.ToolConfig{Enabled: true},
				PreferNative: true,
			},
		},
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				TurnProfile: config.TurnProfileConfig{
					Enabled: true,
					History: config.TurnProfileBlock{Mode: config.TurnProfileModeOff},
					Tools:   config.TurnProfileBlock{Mode: config.TurnProfileModeOff},
				},
			},
		},
	}
	cfg.Agents.Defaults.Workspace = t.TempDir()
	cfg.Agents.Defaults.ModelName = "test-model"
	cfg.Agents.Defaults.MaxTokens = 4096
	cfg.Agents.Defaults.MaxToolIterations = 10
	provider := &nativeSearchCaptureProvider{}
	al := NewAgentLoop(cfg, bus.NewMessageBus(), provider)

	_, err := al.runAgentLoop(
		context.Background(),
		al.GetRegistry().GetDefaultAgent(),
		processOptions{
			SessionKey:      "agent:default:test-tools-off",
			UserMessage:     "hello",
			DefaultResponse: defaultResponse,
		},
	)
	if err != nil {
		t.Fatalf("runAgentLoop() error = %v", err)
	}
	if provider.lastOpts["native_search"] == true {
		t.Fatalf("native_search option enabled despite tools.mode=off: %#v", provider.lastOpts)
	}
}

func TestTurnProfile_ToolsCustomAllowsNativeWebSearch(t *testing.T) {
	cfg := &config.Config{
		Tools: config.ToolsConfig{
			Web: config.WebToolsConfig{
				ToolConfig:   config.ToolConfig{Enabled: true},
				PreferNative: true,
			},
		},
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         t.TempDir(),
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
				TurnProfile: config.TurnProfileConfig{
					Enabled: true,
					History: config.TurnProfileBlock{Mode: config.TurnProfileModeOff},
					Tools: config.TurnProfileBlock{
						Mode:  config.TurnProfileModeCustom,
						Allow: []string{"web_search"},
					},
				},
			},
		},
	}
	provider := &nativeSearchCaptureProvider{}
	al := NewAgentLoop(cfg, bus.NewMessageBus(), provider)

	_, err := al.runAgentLoop(
		context.Background(),
		al.GetRegistry().GetDefaultAgent(),
		processOptions{
			SessionKey:      "agent:default:test-native-web-allowed",
			UserMessage:     "search",
			DefaultResponse: defaultResponse,
		},
	)
	if err != nil {
		t.Fatalf("runAgentLoop() error = %v", err)
	}
	if got, _ := provider.lastOpts["native_search"].(bool); !got {
		t.Fatalf("native_search = %#v, want true", provider.lastOpts["native_search"])
	}
}

func TestTurnProfile_ToolExecutionRejectsDisallowedToolCalls(t *testing.T) {
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				TurnProfile: config.TurnProfileConfig{
					Enabled: true,
					History: config.TurnProfileBlock{Mode: config.TurnProfileModeOff},
					Tools: config.TurnProfileBlock{
						Mode:  config.TurnProfileModeCustom,
						Allow: []string{"echo_text"},
					},
				},
			},
		},
	}
	provider := &turnProfileToolCallProvider{}
	al := NewAgentLoop(cfg, bus.NewMessageBus(), provider)
	al.RegisterTool(&echoTextTool{})
	al.RegisterTool(&echoTextRewrittenTool{})

	response, err := al.runAgentLoop(
		context.Background(),
		al.GetRegistry().GetDefaultAgent(),
		processOptions{
			SessionKey:      "agent:default:test-tool-exec-deny",
			UserMessage:     "run tool",
			DefaultResponse: defaultResponse,
		},
	)
	if err != nil {
		t.Fatalf("runAgentLoop() error = %v", err)
	}
	if response != "done" {
		t.Fatalf("response = %q, want done", response)
	}
	if provider.calls != 2 {
		t.Fatalf("provider calls = %d, want 2", provider.calls)
	}
	var foundDeniedResult bool
	for _, msg := range provider.messages {
		if msg.Role == "tool" &&
			strings.Contains(msg.Content, "not allowed by the active turn profile") {
			foundDeniedResult = true
			break
		}
	}
	if !foundDeniedResult {
		t.Fatalf("second provider call did not include denied tool result: %#v", provider.messages)
	}
}
