package agent

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

type builtinAutoHookConfig struct {
	Model  string `json:"model"`
	Suffix string `json:"suffix"`
}

type builtinAutoHook struct {
	model  string
	suffix string
}

func (h *builtinAutoHook) BeforeLLM(
	ctx context.Context,
	req *LLMHookRequest,
) (*LLMHookRequest, HookDecision, error) {
	next := req.Clone()
	next.Model = h.model
	return next, HookDecision{Action: HookActionModify}, nil
}

func (h *builtinAutoHook) AfterLLM(
	ctx context.Context,
	resp *LLMHookResponse,
) (*LLMHookResponse, HookDecision, error) {
	next := resp.Clone()
	if next.Response != nil {
		next.Response.Content += h.suffix
	}
	return next, HookDecision{Action: HookActionModify}, nil
}

func newConfiguredHookLoop(t *testing.T, provider *llmHookTestProvider, hooks config.HooksConfig) *AgentLoop {
	t.Helper()

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         t.TempDir(),
				Model:             "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
		Hooks: hooks,
	}

	return NewAgentLoop(cfg, bus.NewMessageBus(), provider)
}

func TestAgentLoop_ProcessDirectWithChannel_AutoMountsBuiltinHook(t *testing.T) {
	const hookName = "test-auto-builtin-hook"

	if err := RegisterBuiltinHook(hookName, func(
		ctx context.Context,
		spec config.BuiltinHookConfig,
	) (any, error) {
		var hookCfg builtinAutoHookConfig
		if len(spec.Config) > 0 {
			if err := json.Unmarshal(spec.Config, &hookCfg); err != nil {
				return nil, err
			}
		}
		return &builtinAutoHook{
			model:  hookCfg.Model,
			suffix: hookCfg.Suffix,
		}, nil
	}); err != nil {
		t.Fatalf("RegisterBuiltinHook failed: %v", err)
	}
	t.Cleanup(func() {
		unregisterBuiltinHook(hookName)
	})

	rawCfg, err := json.Marshal(builtinAutoHookConfig{
		Model:  "builtin-model",
		Suffix: "|builtin",
	})
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	provider := &llmHookTestProvider{}
	al := newConfiguredHookLoop(t, provider, config.HooksConfig{
		Enabled: true,
		Builtins: map[string]config.BuiltinHookConfig{
			hookName: {
				Enabled: true,
				Config:  rawCfg,
			},
		},
	})
	defer al.Close()

	resp, err := al.ProcessDirectWithChannel(context.Background(), "hello", "session-1", "cli", "direct")
	if err != nil {
		t.Fatalf("ProcessDirectWithChannel failed: %v", err)
	}
	if resp != "provider content|builtin" {
		t.Fatalf("expected builtin-hooked content, got %q", resp)
	}

	provider.mu.Lock()
	lastModel := provider.lastModel
	provider.mu.Unlock()
	if lastModel != "builtin-model" {
		t.Fatalf("expected builtin model, got %q", lastModel)
	}
}

func TestAgentLoop_ProcessDirectWithChannel_AutoMountsProcessHook(t *testing.T) {
	provider := &llmHookTestProvider{}
	eventLog := filepath.Join(t.TempDir(), "events.log")

	al := newConfiguredHookLoop(t, provider, config.HooksConfig{
		Enabled: true,
		Processes: map[string]config.ProcessHookConfig{
			"ipc-auto": {
				Enabled: true,
				Command: processHookHelperCommand(),
				Env: map[string]string{
					"PICOCLAW_HOOK_HELPER":    "1",
					"PICOCLAW_HOOK_MODE":      "rewrite",
					"PICOCLAW_HOOK_EVENT_LOG": eventLog,
				},
				Observe:   []string{"turn_end"},
				Intercept: []string{"before_llm", "after_llm"},
			},
		},
	})
	defer al.Close()

	resp, err := al.ProcessDirectWithChannel(context.Background(), "hello", "session-1", "cli", "direct")
	if err != nil {
		t.Fatalf("ProcessDirectWithChannel failed: %v", err)
	}
	if resp != "provider content|ipc" {
		t.Fatalf("expected process-hooked content, got %q", resp)
	}

	provider.mu.Lock()
	lastModel := provider.lastModel
	provider.mu.Unlock()
	if lastModel != "process-model" {
		t.Fatalf("expected process model, got %q", lastModel)
	}

	waitForFileContains(t, eventLog, "turn_end")
}

func TestAgentLoop_ProcessDirectWithChannel_InvalidConfiguredHookFails(t *testing.T) {
	provider := &llmHookTestProvider{}
	al := newConfiguredHookLoop(t, provider, config.HooksConfig{
		Enabled: true,
		Processes: map[string]config.ProcessHookConfig{
			"bad-hook": {
				Enabled:   true,
				Command:   processHookHelperCommand(),
				Intercept: []string{"not_supported"},
			},
		},
	})
	defer al.Close()

	_, err := al.ProcessDirectWithChannel(context.Background(), "hello", "session-1", "cli", "direct")
	if err == nil {
		t.Fatal("expected invalid configured hook error")
	}
}
