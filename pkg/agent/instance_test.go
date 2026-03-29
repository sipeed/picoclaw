package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/providers"
)

func TestNewAgentInstance_UsesDefaultsTemperatureAndMaxTokens(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-instance-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				ModelName:         "test-model",
				MaxTokens:         1234,
				MaxToolIterations: 5,
			},
		},
	}

	configuredTemp := 1.0
	cfg.Agents.Defaults.Temperature = &configuredTemp

	provider := &mockProvider{}
	agent := NewAgentInstance(nil, &cfg.Agents.Defaults, cfg, provider)

	if agent.MaxTokens != 1234 {
		t.Fatalf("MaxTokens = %d, want %d", agent.MaxTokens, 1234)
	}
	if agent.Temperature != 1.0 {
		t.Fatalf("Temperature = %f, want %f", agent.Temperature, 1.0)
	}
}

func TestNewAgentInstance_DefaultsTemperatureWhenZero(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-instance-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				ModelName:         "test-model",
				MaxTokens:         1234,
				MaxToolIterations: 5,
			},
		},
	}

	configuredTemp := 0.0
	cfg.Agents.Defaults.Temperature = &configuredTemp

	provider := &mockProvider{}
	agent := NewAgentInstance(nil, &cfg.Agents.Defaults, cfg, provider)

	if agent.Temperature != 0.0 {
		t.Fatalf("Temperature = %f, want %f", agent.Temperature, 0.0)
	}
}

func TestNewAgentInstance_DefaultsTemperatureWhenUnset(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-instance-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				ModelName:         "test-model",
				MaxTokens:         1234,
				MaxToolIterations: 5,
			},
		},
	}

	provider := &mockProvider{}
	agent := NewAgentInstance(nil, &cfg.Agents.Defaults, cfg, provider)

	if agent.Temperature != 0.7 {
		t.Fatalf("Temperature = %f, want %f", agent.Temperature, 0.7)
	}
}

func TestNewAgentInstance_ResolveCandidatesFromModelListAlias(t *testing.T) {
	tests := []struct {
		name         string
		aliasName    string
		modelName    string
		apiBase      string
		wantProvider string
		wantModel    string
	}{
		{
			name:         "alias with provider prefix",
			aliasName:    "step-3.5-flash",
			modelName:    "openrouter/stepfun/step-3.5-flash:free",
			apiBase:      "https://openrouter.ai/api/v1",
			wantProvider: "openrouter",
			wantModel:    "stepfun/step-3.5-flash:free",
		},
		{
			name:         "alias without provider prefix",
			aliasName:    "glm-5",
			modelName:    "glm-5",
			apiBase:      "https://api.z.ai/api/coding/paas/v4",
			wantProvider: "openai",
			wantModel:    "glm-5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "agent-instance-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			cfg := &config.Config{
				Agents: config.AgentsConfig{
					Defaults: config.AgentDefaults{
						Workspace: tmpDir,
						ModelName: tt.aliasName,
					},
				},
				ModelList: []*config.ModelConfig{
					{
						ModelName: tt.aliasName,
						Model:     tt.modelName,
						APIBase:   tt.apiBase,
					},
				},
			}

			provider := &mockProvider{}
			agent := NewAgentInstance(nil, &cfg.Agents.Defaults, cfg, provider)

			if len(agent.Candidates) != 1 {
				t.Fatalf("len(Candidates) = %d, want 1", len(agent.Candidates))
			}
			if agent.Candidates[0].Provider != tt.wantProvider {
				t.Fatalf("candidate provider = %q, want %q", agent.Candidates[0].Provider, tt.wantProvider)
			}
			if agent.Candidates[0].Model != tt.wantModel {
				t.Fatalf("candidate model = %q, want %q", agent.Candidates[0].Model, tt.wantModel)
			}
		})
	}
}

func TestNewAgentInstance_AllowsMediaTempDirForReadListAndExec(t *testing.T) {
	workspace := t.TempDir()
	mediaDir := media.TempDir()
	if err := os.MkdirAll(mediaDir, 0o700); err != nil {
		t.Fatalf("MkdirAll(mediaDir) error = %v", err)
	}

	mediaFile, err := os.CreateTemp(mediaDir, "instance-tool-*.txt")
	if err != nil {
		t.Fatalf("CreateTemp(mediaDir) error = %v", err)
	}
	mediaPath := mediaFile.Name()
	if _, err := mediaFile.WriteString("attachment content"); err != nil {
		mediaFile.Close()
		t.Fatalf("WriteString(mediaFile) error = %v", err)
	}
	if err := mediaFile.Close(); err != nil {
		t.Fatalf("Close(mediaFile) error = %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(mediaPath) })

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:           workspace,
				ModelName:           "test-model",
				RestrictToWorkspace: true,
			},
		},
		Tools: config.ToolsConfig{
			ReadFile: config.ReadFileToolConfig{Enabled: true},
			ListDir:  config.ToolConfig{Enabled: true},
			Exec: config.ExecConfig{
				ToolConfig:         config.ToolConfig{Enabled: true},
				EnableDenyPatterns: true,
				AllowRemote:        true,
			},
		},
	}

	agent := NewAgentInstance(nil, &cfg.Agents.Defaults, cfg, &mockProvider{})

	readTool, ok := agent.Tools.Get("read_file")
	if !ok {
		t.Fatal("read_file tool not registered")
	}
	readResult := readTool.Execute(context.Background(), map[string]any{"path": mediaPath})
	if readResult.IsError {
		t.Fatalf("read_file should allow media temp dir, got: %s", readResult.ForLLM)
	}
	if !strings.Contains(readResult.ForLLM, "attachment content") {
		t.Fatalf("read_file output missing media content: %s", readResult.ForLLM)
	}

	listTool, ok := agent.Tools.Get("list_dir")
	if !ok {
		t.Fatal("list_dir tool not registered")
	}
	listResult := listTool.Execute(context.Background(), map[string]any{"path": mediaDir})
	if listResult.IsError {
		t.Fatalf("list_dir should allow media temp dir, got: %s", listResult.ForLLM)
	}
	if !strings.Contains(listResult.ForLLM, filepath.Base(mediaPath)) {
		t.Fatalf("list_dir output missing media file: %s", listResult.ForLLM)
	}

	execTool, ok := agent.Tools.Get("exec")
	if !ok {
		t.Fatal("exec tool not registered")
	}
	execResult := execTool.Execute(context.Background(), map[string]any{
		"action":  "run",
		"command": "cat " + filepath.Base(mediaPath),
		"cwd":     mediaDir,
	})
	if execResult.IsError {
		t.Fatalf("exec should allow media temp dir, got: %s", execResult.ForLLM)
	}
	if !strings.Contains(execResult.ForLLM, "attachment content") {
		t.Fatalf("exec output missing media content: %s", execResult.ForLLM)
	}
}

// TestRegisterCandidateProviders_NilCfgIsNoop verifies that passing a nil
// config does not panic and leaves the output map empty.
func TestRegisterCandidateProviders_NilCfgIsNoop(t *testing.T) {
	out := map[string]providers.LLMProvider{}
	registerCandidateProviders(nil, []providers.FallbackCandidate{{Provider: "openai", Model: "gpt-4o"}}, out)
	if len(out) != 0 {
		t.Fatalf("expected empty map, got %d entries", len(out))
	}
}

// TestRegisterCandidateProviders_SkipsExistingKeys verifies that a key already
// present in the output map is not overwritten.
func TestRegisterCandidateProviders_SkipsExistingKeys(t *testing.T) {
	existing := &mockProvider{}
	key := providers.ModelKey("openai", "gpt-4o")
	out := map[string]providers.LLMProvider{key: existing}

	cfg := &config.Config{
		ModelList: []*config.ModelConfig{
			{Model: "openai/gpt-4o", APIKeys: config.SimpleSecureStrings("test-key")},
		},
	}
	registerCandidateProviders(cfg, []providers.FallbackCandidate{{Provider: "openai", Model: "gpt-4o"}}, out)

	if out[key] != existing {
		t.Fatal("existing provider entry was overwritten; expected it to be preserved")
	}
}

// TestRegisterCandidateProviders_MatchesBareModelName verifies that a
// model_list entry without a provider prefix (e.g. "gpt-4o") still matches a
// candidate whose provider is "openai" — the default protocol that
// ExtractProtocol assigns to bare names.
func TestRegisterCandidateProviders_MatchesBareModelName(t *testing.T) {
	workspace := t.TempDir()
	out := map[string]providers.LLMProvider{}

	cfg := &config.Config{
		ModelList: []*config.ModelConfig{
			// Bare name — no "openai/" prefix. ExtractProtocol should
			// assign "openai" as the default protocol.
			{Model: "gpt-4o", APIBase: "https://api.openai.com/v1", Workspace: workspace},
		},
	}
	registerCandidateProviders(cfg, []providers.FallbackCandidate{{Provider: "openai", Model: "gpt-4o"}}, out)

	key := providers.ModelKey("openai", "gpt-4o")
	if out[key] == nil {
		t.Fatalf("expected CandidateProviders[%q] to be populated for bare model name", key)
	}
}

// TestRegisterCandidateProviders_MatchesWithProtocolPrefix verifies that a
// model_list entry using full "provider/model" notation (e.g.
// "gemini/gemma-3-27b-it") is matched correctly.
func TestRegisterCandidateProviders_MatchesWithProtocolPrefix(t *testing.T) {
	workspace := t.TempDir()
	out := map[string]providers.LLMProvider{}

	cfg := &config.Config{
		ModelList: []*config.ModelConfig{
			{
				Model:     "gemini/gemma-3-27b-it",
				APIKeys:   config.SimpleSecureStrings("gemini-test-key"),
				Workspace: workspace,
			},
		},
	}
	registerCandidateProviders(cfg, []providers.FallbackCandidate{{Provider: "gemini", Model: "gemma-3-27b-it"}}, out)

	key := providers.ModelKey("gemini", "gemma-3-27b-it")
	if out[key] == nil {
		t.Fatalf("expected CandidateProviders[%q] to be populated for protocol-prefixed model name", key)
	}
}

// TestRegisterCandidateProviders_EmptyCandidatesIsNoop verifies the early-exit
// path when the candidates slice is empty — no index is built and the map
// remains unchanged.
func TestRegisterCandidateProviders_EmptyCandidatesIsNoop(t *testing.T) {
	out := map[string]providers.LLMProvider{}
	cfg := &config.Config{
		ModelList: []*config.ModelConfig{
			{Model: "openai/gpt-4o", APIKeys: config.SimpleSecureStrings("key")},
		},
	}
	registerCandidateProviders(cfg, nil, out)
	if len(out) != 0 {
		t.Fatalf("expected empty map, got %d entries", len(out))
	}
}

// TestRegisterCandidateProviders_EmptyModelListIsNoop verifies the early-exit
// path when model_list is empty — no provider can be created.
func TestRegisterCandidateProviders_EmptyModelListIsNoop(t *testing.T) {
	out := map[string]providers.LLMProvider{}
	cfg := &config.Config{}
	registerCandidateProviders(cfg, []providers.FallbackCandidate{{Provider: "openai", Model: "gpt-4o"}}, out)
	if len(out) != 0 {
		t.Fatalf("expected empty map, got %d entries", len(out))
	}
}

// TestRegisterCandidateProviders_FirstModelListEntryWinsForDuplicates verifies
// that when model_list contains two entries with the same normalised
// provider/model key, the first one is used (mirrors model_list precedence).
func TestRegisterCandidateProviders_FirstModelListEntryWinsForDuplicates(t *testing.T) {
	workspace := t.TempDir()
	out := map[string]providers.LLMProvider{}

	cfg := &config.Config{
		ModelList: []*config.ModelConfig{
			{Model: "openai/gpt-4o", APIBase: "https://first.example.com/v1", Workspace: workspace},
			{Model: "openai/gpt-4o", APIBase: "https://second.example.com/v1", Workspace: workspace},
		},
	}
	registerCandidateProviders(cfg, []providers.FallbackCandidate{{Provider: "openai", Model: "gpt-4o"}}, out)

	key := providers.ModelKey("openai", "gpt-4o")
	if out[key] == nil {
		t.Fatalf("expected CandidateProviders[%q] to be populated", key)
	}
	// Only one entry should be registered despite two model_list entries.
	if len(out) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(out))
	}
}

// TestRegisterCandidateProviders_UnmatchedCandidateIsSkipped verifies that a
// candidate with no matching model_list entry is silently skipped and does not
// cause a panic or leave a nil entry in the map.
func TestRegisterCandidateProviders_UnmatchedCandidateIsSkipped(t *testing.T) {
	out := map[string]providers.LLMProvider{}
	cfg := &config.Config{
		ModelList: []*config.ModelConfig{
			{Model: "openai/gpt-4o", APIKeys: config.SimpleSecureStrings("key")},
		},
	}
	// "anthropic/claude-3-opus" has no matching model_list entry.
	registerCandidateProviders(cfg, []providers.FallbackCandidate{{Provider: "anthropic", Model: "claude-3-opus"}}, out)

	if len(out) != 0 {
		t.Fatalf("expected empty map for unmatched candidate, got %d entries", len(out))
	}
}

// TestNewAgentInstance_CandidateProvidersPopulatedForCrossProviderFallbacks
// mirrors the exact scenario from bug #2140: primary model on OpenRouter with
// Gemini fallbacks. Each entry must get its own provider instance so that
// fallback requests go to the correct API endpoint, not the primary's.
func TestNewAgentInstance_CandidateProvidersPopulatedForCrossProviderFallbacks(t *testing.T) {
	workspace := t.TempDir()

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:      workspace,
				ModelName:      "mistral-small-3.1",
				ModelFallbacks: []string{"gemma-3-27b", "gemini-images"},
			},
		},
		ModelList: []*config.ModelConfig{
			{
				ModelName: "mistral-small-3.1",
				Model:     "openrouter/mistralai/mistral-small-3.1-24b-instruct:free",
				APIBase:   "https://openrouter.ai/api/v1",
				APIKeys:   config.SimpleSecureStrings("sk-or-test"),
				Workspace: workspace,
			},
			{
				ModelName: "gemma-3-27b",
				Model:     "gemini/gemma-3-27b-it",
				APIKeys:   config.SimpleSecureStrings("AIzaSy-test"),
				Workspace: workspace,
			},
			{
				ModelName: "gemini-images",
				Model:     "gemini/gemini-2.5-flash-lite",
				APIKeys:   config.SimpleSecureStrings("AIzaSy-test"),
				Workspace: workspace,
			},
		},
	}

	primaryProvider := &mockProvider{}
	agent := NewAgentInstance(nil, &cfg.Agents.Defaults, cfg, primaryProvider)

	wantKeys := []string{
		providers.ModelKey("openrouter", "mistralai/mistral-small-3.1-24b-instruct:free"),
		providers.ModelKey("gemini", "gemma-3-27b-it"),
		providers.ModelKey("gemini", "gemini-2.5-flash-lite"),
	}

	for _, key := range wantKeys {
		p, ok := agent.CandidateProviders[key]
		if !ok {
			t.Errorf("CandidateProviders missing key %q", key)
			continue
		}
		if p == nil {
			t.Errorf("CandidateProviders[%q] is nil", key)
		}
		// Each fallback must use its own provider, not the injected primary.
		if p == primaryProvider {
			t.Errorf(
				"CandidateProviders[%q] is the same instance as the primary provider; fallback would inherit primary credentials",
				key,
			)
		}
	}

	if t.Failed() {
		t.Logf("CandidateProviders keys present: %v", func() []string {
			keys := make([]string, 0, len(agent.CandidateProviders))
			for k := range agent.CandidateProviders {
				keys = append(keys, k)
			}
			return keys
		}())
	}
}

func TestNewAgentInstance_InvalidExecConfigDoesNotExit(t *testing.T) {
	workspace := t.TempDir()

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace: workspace,
				ModelName: "test-model",
			},
		},
		Tools: config.ToolsConfig{
			ReadFile: config.ReadFileToolConfig{Enabled: true},
			Exec: config.ExecConfig{
				ToolConfig:         config.ToolConfig{Enabled: true},
				EnableDenyPatterns: true,
				CustomDenyPatterns: []string{"[invalid-regex"},
			},
		},
	}

	agent := NewAgentInstance(nil, &cfg.Agents.Defaults, cfg, &mockProvider{})
	if agent == nil {
		t.Fatal("expected agent instance, got nil")
	}

	if _, ok := agent.Tools.Get("exec"); ok {
		t.Fatal("exec tool should not be registered when exec config is invalid")
	}

	if _, ok := agent.Tools.Get("read_file"); !ok {
		t.Fatal("read_file tool should still be registered")
	}
}
