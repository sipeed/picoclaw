package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/credential"
)

// mustSetupSSHKey generates a temporary Ed25519 SSH key in t.TempDir() and sets
// PICOCLAW_SSH_KEY_PATH to its path for the duration of the test. This is required
// whenever a test exercises encryption/decryption via credential.Encrypt or SaveConfig.
func mustSetupSSHKey(t *testing.T) {
	t.Helper()
	keyPath := filepath.Join(t.TempDir(), "picoclaw_ed25519.key")
	if err := credential.GenerateSSHKey(keyPath); err != nil {
		t.Fatalf("mustSetupSSHKey: %v", err)
	}
	t.Setenv("PICOCLAW_SSH_KEY_PATH", keyPath)
}

func TestAgentModelConfig_UnmarshalString(t *testing.T) {
	var m AgentModelConfig
	if err := json.Unmarshal([]byte(`"gpt-4"`), &m); err != nil {
		t.Fatalf("unmarshal string: %v", err)
	}
	if m.Primary != "gpt-4" {
		t.Errorf("Primary = %q, want 'gpt-4'", m.Primary)
	}
	if m.Fallbacks != nil {
		t.Errorf("Fallbacks = %v, want nil", m.Fallbacks)
	}
}

func TestAgentModelConfig_UnmarshalObject(t *testing.T) {
	var m AgentModelConfig
	data := `{"primary": "claude-opus", "fallbacks": ["gpt-4o-mini", "haiku"]}`
	if err := json.Unmarshal([]byte(data), &m); err != nil {
		t.Fatalf("unmarshal object: %v", err)
	}
	if m.Primary != "claude-opus" {
		t.Errorf("Primary = %q, want 'claude-opus'", m.Primary)
	}
	if len(m.Fallbacks) != 2 {
		t.Fatalf("Fallbacks len = %d, want 2", len(m.Fallbacks))
	}
	if m.Fallbacks[0] != "gpt-4o-mini" || m.Fallbacks[1] != "haiku" {
		t.Errorf("Fallbacks = %v", m.Fallbacks)
	}
}

func TestAgentModelConfig_MarshalString(t *testing.T) {
	m := AgentModelConfig{Primary: "gpt-4"}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(data) != `"gpt-4"` {
		t.Errorf("marshal = %s, want '\"gpt-4\"'", string(data))
	}
}

func TestAgentModelConfig_MarshalObject(t *testing.T) {
	m := AgentModelConfig{Primary: "claude-opus", Fallbacks: []string{"haiku"}}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var result map[string]any
	json.Unmarshal(data, &result)
	if result["primary"] != "claude-opus" {
		t.Errorf("primary = %v", result["primary"])
	}
}

func TestProvidersConfig_IsEmpty(t *testing.T) {
	var empty ProvidersConfig
	if !empty.IsEmpty() {
		t.Fatal("empty ProvidersConfig should report empty")
	}

	novita := ProvidersConfig{
		Novita: ProviderConfig{
			APIKey: "test-key",
		},
	}
	if novita.IsEmpty() {
		t.Fatal("ProvidersConfig with novita settings should not report empty")
	}
}

func TestAgentConfig_FullParse(t *testing.T) {
	jsonData := `{
		"agents": {
			"defaults": {
				"workspace": "~/.picoclaw/workspace",
				"model": "glm-4.7",
				"max_tokens": 8192,
				"max_tool_iterations": 20
			},
			"list": [
				{
					"id": "sales",
					"default": true,
					"name": "Sales Bot",
					"model": "gpt-4"
				},
				{
					"id": "support",
					"name": "Support Bot",
					"model": {
						"primary": "claude-opus",
						"fallbacks": ["haiku"]
					},
					"subagents": {
						"allow_agents": ["sales"]
					}
				}
			]
		},
		"bindings": [
			{
				"agent_id": "support",
				"match": {
					"channel": "telegram",
					"account_id": "*",
					"peer": {"kind": "direct", "id": "user123"}
				}
			}
		],
		"session": {
			"dm_scope": "per-peer",
			"identity_links": {
				"john": ["telegram:123", "discord:john#1234"]
			}
		}
	}`

	cfg := DefaultConfig()
	if err := json.Unmarshal([]byte(jsonData), cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(cfg.Agents.List) != 2 {
		t.Fatalf("agents.list len = %d, want 2", len(cfg.Agents.List))
	}

	sales := cfg.Agents.List[0]
	if sales.ID != "sales" || !sales.Default || sales.Name != "Sales Bot" {
		t.Errorf("sales = %+v", sales)
	}
	if sales.Model == nil || sales.Model.Primary != "gpt-4" {
		t.Errorf("sales.Model = %+v", sales.Model)
	}

	support := cfg.Agents.List[1]
	if support.ID != "support" || support.Name != "Support Bot" {
		t.Errorf("support = %+v", support)
	}
	if support.Model == nil || support.Model.Primary != "claude-opus" {
		t.Errorf("support.Model = %+v", support.Model)
	}
	if len(support.Model.Fallbacks) != 1 || support.Model.Fallbacks[0] != "haiku" {
		t.Errorf("support.Model.Fallbacks = %v", support.Model.Fallbacks)
	}
	if support.Subagents == nil || len(support.Subagents.AllowAgents) != 1 {
		t.Errorf("support.Subagents = %+v", support.Subagents)
	}

	if len(cfg.Bindings) != 1 {
		t.Fatalf("bindings len = %d, want 1", len(cfg.Bindings))
	}
	binding := cfg.Bindings[0]
	if binding.AgentID != "support" || binding.Match.Channel != "telegram" {
		t.Errorf("binding = %+v", binding)
	}
	if binding.Match.Peer == nil || binding.Match.Peer.Kind != "direct" || binding.Match.Peer.ID != "user123" {
		t.Errorf("binding.Match.Peer = %+v", binding.Match.Peer)
	}

	if cfg.Session.DMScope != "per-peer" {
		t.Errorf("Session.DMScope = %q", cfg.Session.DMScope)
	}
	if len(cfg.Session.IdentityLinks) != 1 {
		t.Errorf("Session.IdentityLinks = %v", cfg.Session.IdentityLinks)
	}
	links := cfg.Session.IdentityLinks["john"]
	if len(links) != 2 {
		t.Errorf("john links = %v", links)
	}
}

func TestConfig_BackwardCompat_NoAgentsList(t *testing.T) {
	jsonData := `{
		"agents": {
			"defaults": {
				"workspace": "~/.picoclaw/workspace",
				"model": "glm-4.7",
				"max_tokens": 8192,
				"max_tool_iterations": 20
			}
		}
	}`

	cfg := DefaultConfig()
	if err := json.Unmarshal([]byte(jsonData), cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(cfg.Agents.List) != 0 {
		t.Errorf("agents.list should be empty for backward compat, got %d", len(cfg.Agents.List))
	}
	if len(cfg.Bindings) != 0 {
		t.Errorf("bindings should be empty, got %d", len(cfg.Bindings))
	}
}

// TestDefaultConfig_HeartbeatEnabled verifies heartbeat is enabled by default
func TestDefaultConfig_HeartbeatEnabled(t *testing.T) {
	cfg := DefaultConfig()

	if !cfg.Heartbeat.Enabled {
		t.Error("Heartbeat should be enabled by default")
	}
}

// TestDefaultConfig_WorkspacePath verifies workspace path is correctly set
func TestDefaultConfig_WorkspacePath(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Agents.Defaults.Workspace == "" {
		t.Error("Workspace should not be empty")
	}
}

// TestDefaultConfig_Model verifies model is set
func TestDefaultConfig_Model(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Agents.Defaults.Model != "" {
		t.Error("Model should be empty")
	}
}

// TestDefaultConfig_MaxTokens verifies max tokens has default value
func TestDefaultConfig_MaxTokens(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Agents.Defaults.MaxTokens == 0 {
		t.Error("MaxTokens should not be zero")
	}
}

// TestDefaultConfig_MaxToolIterations verifies max tool iterations has default value
func TestDefaultConfig_MaxToolIterations(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Agents.Defaults.MaxToolIterations == 0 {
		t.Error("MaxToolIterations should not be zero")
	}
}

// TestDefaultConfig_Temperature verifies temperature has default value
func TestDefaultConfig_Temperature(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Agents.Defaults.Temperature != nil {
		t.Error("Temperature should be nil when not provided")
	}
}

// TestDefaultConfig_Gateway verifies gateway defaults
func TestDefaultConfig_Gateway(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Gateway.Host != "127.0.0.1" {
		t.Error("Gateway host should have default value")
	}
	if cfg.Gateway.Port == 0 {
		t.Error("Gateway port should have default value")
	}
	if cfg.Gateway.HotReload {
		t.Error("Gateway hot reload should be disabled by default")
	}
}

// TestDefaultConfig_Providers verifies provider structure
func TestDefaultConfig_Providers(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Providers.Anthropic.APIKey != "" {
		t.Error("Anthropic API key should be empty by default")
	}
	if cfg.Providers.OpenAI.APIKey != "" {
		t.Error("OpenAI API key should be empty by default")
	}
	if cfg.Providers.OpenRouter.APIKey != "" {
		t.Error("OpenRouter API key should be empty by default")
	}
}

// TestDefaultConfig_Channels verifies channels are disabled by default
func TestDefaultConfig_Channels(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Channels.Telegram.Enabled {
		t.Error("Telegram should be disabled by default")
	}
	if cfg.Channels.Discord.Enabled {
		t.Error("Discord should be disabled by default")
	}
	if cfg.Channels.Slack.Enabled {
		t.Error("Slack should be disabled by default")
	}
	if cfg.Channels.Matrix.Enabled {
		t.Error("Matrix should be disabled by default")
	}
}

// TestDefaultConfig_WebTools verifies web tools config
func TestDefaultConfig_WebTools(t *testing.T) {
	cfg := DefaultConfig()

	// Verify web tools defaults
	if cfg.Tools.Web.Brave.MaxResults != 5 {
		t.Error("Expected Brave MaxResults 5, got ", cfg.Tools.Web.Brave.MaxResults)
	}
	if len(cfg.Tools.Web.Brave.APIKeys) != 0 {
		t.Error("Brave API key should be empty by default")
	}
	if cfg.Tools.Web.DuckDuckGo.MaxResults != 5 {
		t.Error("Expected DuckDuckGo MaxResults 5, got ", cfg.Tools.Web.DuckDuckGo.MaxResults)
	}
}

func TestSaveConfig_FilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission bits are not enforced on Windows")
	}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.json")

	cfg := DefaultConfig()
	if err := SaveConfig(path, cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	perm := info.Mode().Perm()
	if perm != 0o600 {
		t.Errorf("config file has permission %04o, want 0600", perm)
	}
}

func TestSaveConfig_IncludesEmptyLegacyModelField(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.json")

	cfg := DefaultConfig()
	if err := SaveConfig(path, cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if !strings.Contains(string(data), `"model_name": ""`) {
		t.Fatalf("saved config should include empty legacy model_name field, got: %s", string(data))
	}
}

// TestConfig_Complete verifies all config fields are set
func TestConfig_Complete(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Agents.Defaults.Workspace == "" {
		t.Error("Workspace should not be empty")
	}
	if cfg.Agents.Defaults.Model != "" {
		t.Error("Model should be empty")
	}
	if cfg.Agents.Defaults.Temperature != nil {
		t.Error("Temperature should be nil when not provided")
	}
	if cfg.Agents.Defaults.MaxTokens == 0 {
		t.Error("MaxTokens should not be zero")
	}
	if cfg.Agents.Defaults.MaxToolIterations == 0 {
		t.Error("MaxToolIterations should not be zero")
	}
	if cfg.Gateway.Host != "127.0.0.1" {
		t.Error("Gateway host should have default value")
	}
	if cfg.Gateway.Port == 0 {
		t.Error("Gateway port should have default value")
	}
	if !cfg.Heartbeat.Enabled {
		t.Error("Heartbeat should be enabled by default")
	}
}

func TestDefaultConfig_OpenAIWebSearchEnabled(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.Providers.OpenAI.WebSearch {
		t.Fatal("DefaultConfig().Providers.OpenAI.WebSearch should be true")
	}
}

func TestDefaultConfig_WebPreferNativeEnabled(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.Tools.Web.PreferNative {
		t.Fatal("DefaultConfig().Tools.Web.PreferNative should be true")
	}
}

func TestLoadConfig_WebPreferNativeDefaultsTrueWhenUnset(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"tools":{"web":{"enabled":true}}}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}
	if !cfg.Tools.Web.PreferNative {
		t.Fatal("PreferNative should remain true when unset in config file")
	}
}

func TestLoadConfig_WebPreferNativeCanBeDisabled(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"tools":{"web":{"prefer_native":false}}}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}
	if cfg.Tools.Web.PreferNative {
		t.Fatal("PreferNative should be false when disabled in config file")
	}
}

func TestDefaultConfig_ExecAllowRemoteEnabled(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.Tools.Exec.AllowRemote {
		t.Fatal("DefaultConfig().Tools.Exec.AllowRemote should be true")
	}
}

func TestDefaultConfig_CronAllowCommandEnabled(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.Tools.Cron.AllowCommand {
		t.Fatal("DefaultConfig().Tools.Cron.AllowCommand should be true")
	}
}

func TestDefaultConfig_LogLevel(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Agents.Defaults.LogLevel != "fatal" {
		t.Errorf("LogLevel = %q, want \"fatal\"", cfg.Agents.Defaults.LogLevel)
	}
}

func TestLoadConfig_OpenAIWebSearchDefaultsTrueWhenUnset(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"providers":{"openai":{"api_base":""}}}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}
	if !cfg.Providers.OpenAI.WebSearch {
		t.Fatal("OpenAI codex web search should remain true when unset in config file")
	}
}

func TestLoadConfig_ExecAllowRemoteDefaultsTrueWhenUnset(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"tools":{"exec":{"enable_deny_patterns":true}}}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}
	if !cfg.Tools.Exec.AllowRemote {
		t.Fatal("tools.exec.allow_remote should remain true when unset in config file")
	}
}

func TestLoadConfig_CronAllowCommandDefaultsTrueWhenUnset(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"tools":{"cron":{"exec_timeout_minutes":5}}}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}
	if !cfg.Tools.Cron.AllowCommand {
		t.Fatal("tools.cron.allow_command should remain true when unset in config file")
	}
}

func TestLoadConfig_OpenAIWebSearchCanBeDisabled(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"providers":{"openai":{"web_search":false}}}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}
	if cfg.Providers.OpenAI.WebSearch {
		t.Fatal("OpenAI codex web search should be false when disabled in config file")
	}
}

func TestLoadConfig_WebToolsProxy(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	configJSON := `{
  "agents": {"defaults":{"workspace":"./workspace","model":"gpt4","max_tokens":8192,"max_tool_iterations":20}},
  "model_list": [{"model_name":"gpt4","model":"openai/gpt-5.4","api_key":"x"}],
  "tools": {"web":{"proxy":"http://127.0.0.1:7890"}}
}`
	if err := os.WriteFile(configPath, []byte(configJSON), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}
	if cfg.Tools.Web.Proxy != "http://127.0.0.1:7890" {
		t.Fatalf("Tools.Web.Proxy = %q, want %q", cfg.Tools.Web.Proxy, "http://127.0.0.1:7890")
	}
}

// TestDefaultConfig_DMScope verifies the default dm_scope value
// TestDefaultConfig_SummarizationThresholds verifies summarization defaults
func TestDefaultConfig_SummarizationThresholds(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Agents.Defaults.SummarizeMessageThreshold != 20 {
		t.Errorf("SummarizeMessageThreshold = %d, want 20", cfg.Agents.Defaults.SummarizeMessageThreshold)
	}
	if cfg.Agents.Defaults.SummarizeTokenPercent != 75 {
		t.Errorf("SummarizeTokenPercent = %d, want 75", cfg.Agents.Defaults.SummarizeTokenPercent)
	}
}

func TestDefaultConfig_DMScope(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Session.DMScope != "per-channel-peer" {
		t.Errorf("Session.DMScope = %q, want 'per-channel-peer'", cfg.Session.DMScope)
	}
}

func TestDefaultConfig_WorkspacePath_Default(t *testing.T) {
	t.Setenv("PICOCLAW_HOME", "")

	var fakeHome string
	if runtime.GOOS == "windows" {
		fakeHome = `C:\tmp\home`
		t.Setenv("USERPROFILE", fakeHome)
	} else {
		fakeHome = "/tmp/home"
		t.Setenv("HOME", fakeHome)
	}

	cfg := DefaultConfig()
	want := filepath.Join(fakeHome, ".picoclaw", "workspace")

	if cfg.Agents.Defaults.Workspace != want {
		t.Errorf("Default workspace path = %q, want %q", cfg.Agents.Defaults.Workspace, want)
	}
}

func TestDefaultConfig_WorkspacePath_WithPicoclawHome(t *testing.T) {
	t.Setenv("PICOCLAW_HOME", "/custom/picoclaw/home")

	cfg := DefaultConfig()
	want := filepath.Join("/custom/picoclaw/home", "workspace")

	if cfg.Agents.Defaults.Workspace != want {
		t.Errorf("Workspace path with PICOCLAW_HOME = %q, want %q", cfg.Agents.Defaults.Workspace, want)
	}
}

// TestFlexibleStringSlice_UnmarshalText tests UnmarshalText with various comma separators
func TestFlexibleStringSlice_UnmarshalText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "English commas only",
			input:    "123,456,789",
			expected: []string{"123", "456", "789"},
		},
		{
			name:     "Chinese commas only",
			input:    "123，456，789",
			expected: []string{"123", "456", "789"},
		},
		{
			name:     "Mixed English and Chinese commas",
			input:    "123,456，789",
			expected: []string{"123", "456", "789"},
		},
		{
			name:     "Single value",
			input:    "123",
			expected: []string{"123"},
		},
		{
			name:     "Values with whitespace",
			input:    " 123 , 456 , 789 ",
			expected: []string{"123", "456", "789"},
		},
		{
			name:     "Empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "Only commas - English",
			input:    ",,",
			expected: []string{},
		},
		{
			name:     "Only commas - Chinese",
			input:    "，，",
			expected: []string{},
		},
		{
			name:     "Mixed commas with empty parts",
			input:    "123,,456，，789",
			expected: []string{"123", "456", "789"},
		},
		{
			name:     "Complex mixed values",
			input:    "user1@example.com，user2@test.com, admin@domain.org",
			expected: []string{"user1@example.com", "user2@test.com", "admin@domain.org"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f FlexibleStringSlice
			err := f.UnmarshalText([]byte(tt.input))
			if err != nil {
				t.Fatalf("UnmarshalText(%q) error = %v", tt.input, err)
			}

			if tt.expected == nil {
				if f != nil {
					t.Errorf("UnmarshalText(%q) = %v, want nil", tt.input, f)
				}
				return
			}

			if len(f) != len(tt.expected) {
				t.Errorf("UnmarshalText(%q) length = %d, want %d", tt.input, len(f), len(tt.expected))
				return
			}

			for i, v := range tt.expected {
				if f[i] != v {
					t.Errorf("UnmarshalText(%q)[%d] = %q, want %q", tt.input, i, f[i], v)
				}
			}
		})
	}
}

// TestFlexibleStringSlice_UnmarshalText_EmptySliceConsistency tests nil vs empty slice behavior
func TestFlexibleStringSlice_UnmarshalText_EmptySliceConsistency(t *testing.T) {
	t.Run("Empty string returns nil", func(t *testing.T) {
		var f FlexibleStringSlice
		err := f.UnmarshalText([]byte(""))
		if err != nil {
			t.Fatalf("UnmarshalText error = %v", err)
		}
		if f != nil {
			t.Errorf("Empty string should return nil, got %v", f)
		}
	})

	t.Run("Commas only returns empty slice", func(t *testing.T) {
		var f FlexibleStringSlice
		err := f.UnmarshalText([]byte(",,,"))
		if err != nil {
			t.Fatalf("UnmarshalText error = %v", err)
		}
		if f == nil {
			t.Error("Commas only should return empty slice, not nil")
		}
		if len(f) != 0 {
			t.Errorf("Expected empty slice, got %v", f)
		}
	})
}

// TestLoadConfig_WarnsForPlaintextAPIKey verifies that LoadConfig resolves a plaintext
// api_key into memory but does NOT rewrite the config file. File writes are the sole
// responsibility of SaveConfig.
func TestLoadConfig_WarnsForPlaintextAPIKey(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	const original = `{"model_list":[{"model_name":"test","model":"openai/gpt-4","api_key":"sk-plaintext"}]}`
	if err := os.WriteFile(cfgPath, []byte(original), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	t.Setenv("PICOCLAW_KEY_PASSPHRASE", "test-passphrase")
	t.Setenv("PICOCLAW_SSH_KEY_PATH", "")

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	// In-memory value must be the resolved plaintext.
	if cfg.ModelList[0].APIKey != "sk-plaintext" {
		t.Errorf("in-memory api_key = %q, want %q", cfg.ModelList[0].APIKey, "sk-plaintext")
	}
	// The file on disk must remain unchanged — LoadConfig must not write anything.
	raw, _ := os.ReadFile(cfgPath)
	if string(raw) != original {
		t.Errorf("LoadConfig must not modify the config file; got:\n%s", string(raw))
	}
}

// TestSaveConfig_EncryptsPlaintextAPIKey verifies that SaveConfig writes enc:// ciphertext
// to disk and that a subsequent LoadConfig decrypts it back to the original plaintext.
func TestSaveConfig_EncryptsPlaintextAPIKey(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	t.Setenv("PICOCLAW_KEY_PASSPHRASE", "test-passphrase")
	mustSetupSSHKey(t)

	cfg := DefaultConfig()
	cfg.ModelList = []ModelConfig{
		{ModelName: "test", Model: "openai/gpt-4", APIKey: "sk-plaintext"},
	}
	if err := SaveConfig(cfgPath, cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	// Disk must contain enc://, not the raw key.
	raw, _ := os.ReadFile(cfgPath)
	if !strings.Contains(string(raw), "enc://") {
		t.Errorf("saved file should contain enc://, got:\n%s", string(raw))
	}
	if strings.Contains(string(raw), "sk-plaintext") {
		t.Errorf("saved file must not contain the plaintext key")
	}

	// A fresh load must decrypt back to the original plaintext.
	cfg2, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig after SaveConfig: %v", err)
	}
	if cfg2.ModelList[0].APIKey != "sk-plaintext" {
		t.Errorf("loaded api_key = %q, want %q", cfg2.ModelList[0].APIKey, "sk-plaintext")
	}
}

// TestLoadConfig_NoSealWithoutPassphrase verifies that api_key values are left
// unchanged when PICOCLAW_KEY_PASSPHRASE is not set.
func TestLoadConfig_NoSealWithoutPassphrase(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	data := `{"model_list":[{"model_name":"test","model":"openai/gpt-4","api_key":"sk-plaintext"}]}`
	if err := os.WriteFile(cfgPath, []byte(data), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	t.Setenv("PICOCLAW_KEY_PASSPHRASE", "")
	t.Setenv("PICOCLAW_SSH_KEY_PATH", "")

	if _, err := LoadConfig(cfgPath); err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	raw, _ := os.ReadFile(cfgPath)
	if strings.Contains(string(raw), "enc://") {
		t.Error("config file must not be modified when no passphrase is set")
	}
}

// TestLoadConfig_FileRefNotSealed verifies that file:// api_key references are not
// converted to enc:// values (they are resolved at runtime by the Resolver).
func TestLoadConfig_FileRefNotSealed(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	keyFile := filepath.Join(dir, "openai.key")
	if err := os.WriteFile(keyFile, []byte("sk-from-file"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	data := `{"model_list":[{"model_name":"test","model":"openai/gpt-4","api_key":"file://openai.key"}]}`
	if err := os.WriteFile(cfgPath, []byte(data), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	t.Setenv("PICOCLAW_KEY_PASSPHRASE", "test-passphrase")
	t.Setenv("PICOCLAW_SSH_KEY_PATH", "")

	if _, err := LoadConfig(cfgPath); err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	raw, _ := os.ReadFile(cfgPath)
	if !strings.Contains(string(raw), "file://openai.key") {
		t.Error("file:// reference should be preserved unchanged in the config file")
	}
	if strings.Contains(string(raw), "enc://") {
		t.Error("file:// reference must not be converted to enc://")
	}
}

// TestSaveConfig_MixedKeys verifies that SaveConfig encrypts only plaintext api_keys
// and leaves already-encrypted (enc://) and file:// entries unchanged.
func TestSaveConfig_MixedKeys(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	t.Setenv("PICOCLAW_KEY_PASSPHRASE", "test-passphrase")
	mustSetupSSHKey(t)

	// Pre-encrypt one key so we have a genuine enc:// value to put in the config.
	if err := SaveConfig(cfgPath, &Config{
		ModelList: []ModelConfig{
			{ModelName: "pre", Model: "openai/gpt-4", APIKey: "sk-already-plain"},
		},
	}); err != nil {
		t.Fatalf("setup SaveConfig: %v", err)
	}
	raw, _ := os.ReadFile(cfgPath)
	// Extract the enc:// value from the saved file.
	var tmp struct {
		ModelList []struct {
			APIKey string `json:"api_key"`
		} `json:"model_list"`
	}
	if err := json.Unmarshal(raw, &tmp); err != nil || len(tmp.ModelList) == 0 {
		t.Fatalf("setup: could not parse saved config: %v", err)
	}
	alreadyEncrypted := tmp.ModelList[0].APIKey
	if !strings.HasPrefix(alreadyEncrypted, "enc://") {
		t.Fatalf("setup: expected enc:// key, got %q", alreadyEncrypted)
	}

	// Build a config with three models:
	//   1. plaintext   → must be encrypted by SaveConfig
	//   2. enc://      → must be left unchanged (already encrypted)
	//   3. file://     → must be left unchanged (file reference)
	keyFile := filepath.Join(dir, "api.key")
	if err := os.WriteFile(keyFile, []byte("sk-from-file"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	cfg := &Config{
		ModelList: []ModelConfig{
			{ModelName: "plain", Model: "openai/gpt-4", APIKey: "sk-new-plaintext"},
			{ModelName: "enc", Model: "openai/gpt-4", APIKey: alreadyEncrypted},
			{ModelName: "file", Model: "openai/gpt-4", APIKey: "file://api.key"},
		},
	}
	if err := SaveConfig(cfgPath, cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	raw, _ = os.ReadFile(cfgPath)
	s := string(raw)

	// 1. Plaintext must be encrypted.
	if strings.Contains(s, "sk-new-plaintext") {
		t.Error("plaintext key must not appear in saved file")
	}
	// 2. The pre-existing enc:// value must still be present (byte-for-byte unchanged).
	if !strings.Contains(s, alreadyEncrypted) {
		t.Error("pre-existing enc:// entry must be preserved unchanged")
	}
	// 3. file:// must be preserved.
	if !strings.Contains(s, "file://api.key") {
		t.Error("file:// reference must be preserved unchanged")
	}

	// Now load and verify all three decrypt/resolve correctly.
	cfg2, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig after SaveConfig: %v", err)
	}
	byName := make(map[string]string)
	for _, m := range cfg2.ModelList {
		byName[m.ModelName] = m.APIKey
	}
	if byName["plain"] != "sk-new-plaintext" {
		t.Errorf("plain model api_key = %q, want %q", byName["plain"], "sk-new-plaintext")
	}
	if byName["enc"] != "sk-already-plain" {
		t.Errorf("enc model api_key = %q, want %q", byName["enc"], "sk-already-plain")
	}
	if byName["file"] != "sk-from-file" {
		t.Errorf("file model api_key = %q, want %q", byName["file"], "sk-from-file")
	}
}

// TestLoadConfig_MixedKeys_NoPassphrase verifies that when PICOCLAW_KEY_PASSPHRASE
// is not set, enc:// entries cause LoadConfig to return an error, while plaintext
// and file:// entries in the same config are not affected.
func TestLoadConfig_MixedKeys_NoPassphrase(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	// First encrypt a key so we have a real enc:// value.
	t.Setenv("PICOCLAW_KEY_PASSPHRASE", "test-passphrase")
	mustSetupSSHKey(t)
	if err := SaveConfig(cfgPath, &Config{
		ModelList: []ModelConfig{
			{ModelName: "m", Model: "openai/gpt-4", APIKey: "sk-secret"},
		},
	}); err != nil {
		t.Fatalf("setup SaveConfig: %v", err)
	}
	raw, _ := os.ReadFile(cfgPath)
	var tmp struct {
		ModelList []struct {
			APIKey string `json:"api_key"`
		} `json:"model_list"`
	}
	if err := json.Unmarshal(raw, &tmp); err != nil {
		t.Fatalf("setup parse: %v", err)
	}
	encValue := tmp.ModelList[0].APIKey

	// Write a mixed config: enc:// + plaintext + file://
	keyFile := filepath.Join(dir, "api.key")
	if err := os.WriteFile(keyFile, []byte("sk-from-file"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	mixed, _ := json.Marshal(map[string]any{
		"model_list": []map[string]any{
			{"model_name": "enc", "model": "openai/gpt-4", "api_key": encValue},
			{"model_name": "plain", "model": "openai/gpt-4", "api_key": "sk-plain"},
			{"model_name": "file", "model": "openai/gpt-4", "api_key": "file://api.key"},
		},
	})
	if err := os.WriteFile(cfgPath, mixed, 0o600); err != nil {
		t.Fatalf("setup write: %v", err)
	}

	// Now clear the passphrase — LoadConfig must fail because enc:// cannot be decrypted.
	t.Setenv("PICOCLAW_KEY_PASSPHRASE", "")

	_, err := LoadConfig(cfgPath)
	if err == nil {
		t.Fatal("LoadConfig should fail when enc:// key is present and no passphrase is set")
	}
	if !strings.Contains(err.Error(), "passphrase required") {
		t.Errorf("error should mention passphrase required, got: %v", err)
	}
}

// TestSaveConfig_UsesPassphraseProvider verifies that SaveConfig encrypts plaintext
// api_keys using credential.PassphraseProvider() rather than os.Getenv directly.
// This matters for the launcher, which clears the environment variable and redirects
// PassphraseProvider to an in-memory SecureStore.
func TestSaveConfig_UsesPassphraseProvider(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	// Ensure the env var is empty — passphrase must come from PassphraseProvider only.
	t.Setenv("PICOCLAW_KEY_PASSPHRASE", "")
	mustSetupSSHKey(t)

	// Replace PassphraseProvider with an in-memory function (simulating SecureStore).
	const testPassphrase = "provider-passphrase"
	orig := credential.PassphraseProvider
	credential.PassphraseProvider = func() string { return testPassphrase }
	t.Cleanup(func() { credential.PassphraseProvider = orig })

	cfg := DefaultConfig()
	cfg.ModelList = []ModelConfig{
		{ModelName: "test", Model: "openai/gpt-4", APIKey: "sk-plaintext"},
	}
	if err := SaveConfig(cfgPath, cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	raw, _ := os.ReadFile(cfgPath)
	if !strings.Contains(string(raw), "enc://") {
		t.Errorf("SaveConfig should have encrypted plaintext key via PassphraseProvider; got:\n%s", raw)
	}
}

// TestLoadConfig_UsesPassphraseProvider verifies that LoadConfig decrypts enc:// keys
// using credential.PassphraseProvider() rather than os.Getenv directly.
func TestLoadConfig_UsesPassphraseProvider(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	// Ensure the env var is empty throughout.
	t.Setenv("PICOCLAW_KEY_PASSPHRASE", "")
	mustSetupSSHKey(t)

	const testPassphrase = "provider-passphrase"
	const plainKey = "sk-secret"

	// First, encrypt the key using the same passphrase.
	encrypted, err := credential.Encrypt(testPassphrase, "", plainKey)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	raw, _ := json.Marshal(map[string]any{
		"model_list": []map[string]any{
			{"model_name": "test", "model": "openai/gpt-4", "api_key": encrypted},
		},
	})
	if err = os.WriteFile(cfgPath, raw, 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Redirect PassphraseProvider — env var is empty, so without this the load would fail.
	orig := credential.PassphraseProvider
	credential.PassphraseProvider = func() string { return testPassphrase }
	t.Cleanup(func() { credential.PassphraseProvider = orig })

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.ModelList[0].APIKey != plainKey {
		t.Errorf("api_key = %q, want %q", cfg.ModelList[0].APIKey, plainKey)
	}
}

func TestConfigParsesLogLevel(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	data := `{"agents":{"defaults":{"log_level":"debug"}}}`
	if err := os.WriteFile(cfgPath, []byte(data), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Agents.Defaults.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want \"debug\"", cfg.Agents.Defaults.LogLevel)
	}
}

func TestConfigLogLevelEmpty(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	data := `{}`
	if err := os.WriteFile(cfgPath, []byte(data), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	// When config omits log_level, the DefaultConfig value ("fatal") is preserved.
	if cfg.Agents.Defaults.LogLevel != "fatal" {
		t.Errorf("LogLevel = %q, want \"fatal\"", cfg.Agents.Defaults.LogLevel)
	}
}
