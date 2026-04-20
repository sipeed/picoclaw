package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"

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
		"session": {
			"dimensions": ["sender"],
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

	if len(cfg.Session.Dimensions) != 1 || cfg.Session.Dimensions[0] != "sender" {
		t.Errorf("Session.Dimensions = %v", cfg.Session.Dimensions)
	}
	if len(cfg.Session.IdentityLinks) != 1 {
		t.Errorf("Session.IdentityLinks = %v", cfg.Session.IdentityLinks)
	}
	links := cfg.Session.IdentityLinks["john"]
	if len(links) != 2 {
		t.Errorf("john links = %v", links)
	}
}

func TestDefaultConfig_MCPMaxInlineTextChars(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Tools.MCP.GetMaxInlineTextChars() != DefaultMCPMaxInlineTextChars {
		t.Fatalf(
			"DefaultConfig().Tools.MCP.GetMaxInlineTextChars() = %d, want %d",
			cfg.Tools.MCP.GetMaxInlineTextChars(),
			DefaultMCPMaxInlineTextChars,
		)
	}
}

func TestLoadConfig_MCPMaxInlineTextChars(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	raw := `{
		"tools": {
			"mcp": {
				"enabled": true,
				"max_inline_text_chars": 2048
			}
		}
	}`
	if err := os.WriteFile(configPath, []byte(raw), 0o644); err != nil {
		t.Fatalf("WriteFile(configPath): %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}
	if got := cfg.Tools.MCP.GetMaxInlineTextChars(); got != 2048 {
		t.Fatalf("cfg.Tools.MCP.GetMaxInlineTextChars() = %d, want 2048", got)
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
}

func TestAgentConfig_ParsesDispatchRules(t *testing.T) {
	jsonData := `{
		"agents": {
			"defaults": {
				"workspace": "~/.picoclaw/workspace",
				"model": "glm-4.7"
			},
			"list": [
				{ "id": "main", "default": true },
				{ "id": "support" }
			],
			"dispatch": {
				"rules": [
					{
						"name": "support-vip",
						"agent": "support",
						"when": {
							"channel": "telegram",
							"chat": "group:-100123",
							"sender": "12345",
							"mentioned": true
						},
						"session_dimensions": ["chat", "sender"]
					}
				]
			}
		}
	}`

	cfg := DefaultConfig()
	if err := json.Unmarshal([]byte(jsonData), cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cfg.Agents.Dispatch == nil {
		t.Fatal("Agents.Dispatch should not be nil")
	}
	if len(cfg.Agents.Dispatch.Rules) != 1 {
		t.Fatalf("Dispatch.Rules len = %d, want 1", len(cfg.Agents.Dispatch.Rules))
	}
	rule := cfg.Agents.Dispatch.Rules[0]
	if rule.Name != "support-vip" || rule.Agent != "support" {
		t.Fatalf("rule = %+v", rule)
	}
	if rule.When.Channel != "telegram" || rule.When.Chat != "group:-100123" || rule.When.Sender != "12345" {
		t.Fatalf("rule.When = %+v", rule.When)
	}
	if rule.When.Mentioned == nil || !*rule.When.Mentioned {
		t.Fatalf("rule.When.Mentioned = %+v, want true", rule.When.Mentioned)
	}
	if got := rule.SessionDimensions; len(got) != 2 || got[0] != "chat" || got[1] != "sender" {
		t.Fatalf("rule.SessionDimensions = %v, want [chat sender]", got)
	}
}

func TestLoadConfig_MigratesLegacyBindingsToDispatchRules(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	raw := `{
		"version": 2,
		"agents": {
			"defaults": {
				"workspace": "~/.picoclaw/workspace",
				"model": "glm-4.7"
			},
			"list": [
				{ "id": "main", "default": true },
				{ "id": "support" },
				{ "id": "ops" },
				{ "id": "slack" }
			]
		},
		"bindings": [
			{
				"agent_id": "support",
				"match": {
					"channel": "telegram",
					"peer": { "kind": "group", "id": "-100123" }
				}
			},
			{
				"agent_id": "ops",
				"match": {
					"channel": "discord",
					"guild_id": "guild-1"
				}
			},
			{
				"agent_id": "slack",
				"match": {
					"channel": "slack",
					"account_id": "*"
				}
			}
		]
	}`
	if err := os.WriteFile(configPath, []byte(raw), 0o644); err != nil {
		t.Fatalf("WriteFile(configPath): %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}
	if cfg.Agents.Dispatch == nil {
		t.Fatal("Agents.Dispatch should not be nil")
	}
	if len(cfg.Agents.Dispatch.Rules) != 3 {
		t.Fatalf("Dispatch.Rules len = %d, want 3", len(cfg.Agents.Dispatch.Rules))
	}

	first := cfg.Agents.Dispatch.Rules[0]
	if first.Agent != "support" {
		t.Fatalf("first.Agent = %q, want %q", first.Agent, "support")
	}
	if first.When.Channel != "telegram" || first.When.Chat != "group:-100123" {
		t.Fatalf("first.When = %+v", first.When)
	}
	if first.When.Account != legacyDefaultAccountID {
		t.Fatalf("first.When.Account = %q, want %q", first.When.Account, legacyDefaultAccountID)
	}

	second := cfg.Agents.Dispatch.Rules[1]
	if second.Agent != "ops" || second.When.Space != "guild:guild-1" {
		t.Fatalf("second = %+v", second)
	}

	third := cfg.Agents.Dispatch.Rules[2]
	if third.Agent != "slack" {
		t.Fatalf("third.Agent = %q, want %q", third.Agent, "slack")
	}
	if third.When.Channel != "slack" || third.When.Account != "" {
		t.Fatalf("third.When = %+v", third.When)
	}
}

func TestLoadConfig_PrefersDispatchRulesOverLegacyBindings(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	raw := `{
		"version": 2,
		"agents": {
			"defaults": {
				"workspace": "~/.picoclaw/workspace",
				"model": "glm-4.7"
			},
			"list": [
				{ "id": "main", "default": true },
				{ "id": "support" }
			],
			"dispatch": {
				"rules": [
					{
						"name": "explicit",
						"agent": "support",
						"when": {
							"channel": "telegram",
							"chat": "group:-100123"
						}
					}
				]
			}
		},
		"bindings": [
			{
				"agent_id": "main",
				"match": {
					"channel": "telegram",
					"account_id": "*"
				}
			}
		]
	}`
	if err := os.WriteFile(configPath, []byte(raw), 0o644); err != nil {
		t.Fatalf("WriteFile(configPath): %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}
	if cfg.Agents.Dispatch == nil {
		t.Fatal("Agents.Dispatch should not be nil")
	}
	if len(cfg.Agents.Dispatch.Rules) != 1 {
		t.Fatalf("Dispatch.Rules len = %d, want 1", len(cfg.Agents.Dispatch.Rules))
	}
	if cfg.Agents.Dispatch.Rules[0].Name != "explicit" {
		t.Fatalf("Dispatch.Rules[0].Name = %q, want %q", cfg.Agents.Dispatch.Rules[0].Name, "explicit")
	}
}

func TestLoadConfig_MigratesLegacyDirectBindingsWithIdentityLinks(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	raw := `{
		"version": 2,
		"agents": {
			"defaults": {
				"workspace": "~/.picoclaw/workspace",
				"model": "glm-4.7"
			},
			"list": [
				{ "id": "main", "default": true },
				{ "id": "support" }
			]
		},
		"session": {
			"identity_links": {
				"john": ["telegram:123", "123"]
			}
		},
		"bindings": [
			{
				"agent_id": "support",
				"match": {
					"channel": "telegram",
					"peer": { "kind": "direct", "id": "123" }
				}
			}
		]
	}`
	if err := os.WriteFile(configPath, []byte(raw), 0o644); err != nil {
		t.Fatalf("WriteFile(configPath): %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}
	if cfg.Agents.Dispatch == nil || len(cfg.Agents.Dispatch.Rules) != 1 {
		t.Fatalf("Dispatch.Rules = %+v, want 1 migrated rule", cfg.Agents.Dispatch)
	}
	if got := cfg.Agents.Dispatch.Rules[0].When.Sender; got != "john" {
		t.Fatalf("migrated sender selector = %q, want %q", got, "john")
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

	if cfg.Gateway.Host != "localhost" {
		t.Error("Gateway host should have default value")
	}
	if cfg.Gateway.Port == 0 {
		t.Error("Gateway port should have default value")
	}
	if cfg.Gateway.HotReload {
		t.Error("Gateway hot reload should be disabled by default")
	}
}

// TestDefaultConfig_Channels verifies channels are disabled by default
func TestDefaultConfig_Channels(t *testing.T) {
	cfg := DefaultConfig()

	for name, bc := range cfg.Channels {
		if bc.Enabled {
			t.Errorf("Channel %q should be disabled by default", name)
		}
	}
}

func TestValidateSingletonChannels_RejectsMultipleInstances(t *testing.T) {
	channels := ChannelsConfig{
		"pico1": &Channel{Enabled: true, Type: ChannelPico},
		"pico2": &Channel{Enabled: true, Type: ChannelPico},
	}
	err := validateSingletonChannels(channels)
	if err == nil {
		t.Fatal("expected error for multiple pico channels, got nil")
	}
	if !strings.Contains(err.Error(), "singleton") {
		t.Fatalf("expected singleton error, got: %v", err)
	}
}

func TestValidateSingletonChannels_AllowsSingleInstance(t *testing.T) {
	channels := ChannelsConfig{
		"pico1": &Channel{Enabled: true, Type: ChannelPico},
	}
	err := validateSingletonChannels(channels)
	if err != nil {
		t.Fatalf("expected no error for single pico channel, got: %v", err)
	}
}

func TestValidateSingletonChannels_IgnoresDisabledInstances(t *testing.T) {
	channels := ChannelsConfig{
		"pico1": &Channel{Enabled: true, Type: ChannelPico},
		"pico2": &Channel{Enabled: false, Type: ChannelPico},
	}
	err := validateSingletonChannels(channels)
	if err != nil {
		t.Fatalf("expected no error when only one pico channel is enabled, got: %v", err)
	}
}

func TestValidateSingletonChannels_AllowsMultiInstanceTypes(t *testing.T) {
	channels := ChannelsConfig{
		"tg1": &Channel{Enabled: true, Type: ChannelTelegram},
		"tg2": &Channel{Enabled: true, Type: ChannelTelegram},
	}
	err := validateSingletonChannels(channels)
	if err != nil {
		t.Fatalf("telegram should allow multiple instances, got error: %v", err)
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

func TestSaveConfig_PreservesDisabledTelegramPlaceholder(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.json")

	cfg := DefaultConfig()
	if bc := cfg.Channels.Get("telegram"); bc != nil {
		bc.Placeholder.Enabled = false
	}

	if err := SaveConfig(path, cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !strings.Contains(string(data), `"placeholder": {`) {
		t.Fatalf("saved config should include telegram placeholder config, got: %s", string(data))
	}
	if !strings.Contains(string(data), `"enabled": false`) {
		t.Fatalf("saved config should persist placeholder.enabled=false, got: %s", string(data))
	}

	loaded, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	bc := loaded.Channels.Get("telegram")
	if bc != nil && bc.Placeholder.Enabled {
		t.Fatal("telegram placeholder should remain disabled after SaveConfig/LoadConfig round-trip")
	}
}

// TestSaveConfig_FiltersVirtualModels verifies that SaveConfig does not write
// virtual models (generated by expandMultiKeyModels) to the config file.
func TestSaveConfig_FiltersVirtualModels(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.json")

	cfg := DefaultConfig()

	// Manually add a virtual model to ModelList (simulating what expandMultiKeyModels does)
	primaryModel := &ModelConfig{
		ModelName: "gpt-4",
		Model:     "openai/gpt-4o",
		APIKeys:   SimpleSecureStrings("key1"),
	}
	virtualModel := &ModelConfig{
		ModelName: "gpt-4__key_1",
		Model:     "openai/gpt-4o",
		APIKeys:   SimpleSecureStrings("key2"),
		isVirtual: true,
	}
	cfg.ModelList = []*ModelConfig{primaryModel, virtualModel}

	// SaveConfig should filter out virtual models
	if err := SaveConfig(path, cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Reload and verify
	reloaded, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Should only have the primary model, not the virtual one
	if len(reloaded.ModelList) != 1 {
		t.Fatalf("expected 1 model after reload, got %d", len(reloaded.ModelList))
	}

	if reloaded.ModelList[0].ModelName != "gpt-4" {
		t.Errorf("expected model_name 'gpt-4', got %q", reloaded.ModelList[0].ModelName)
	}

	// Verify virtual model was not persisted
	for _, m := range reloaded.ModelList {
		if m.ModelName == "gpt-4__key_1" {
			t.Errorf("virtual model gpt-4__key_1 should not have been saved")
		}
	}

	// Verify the saved file does not contain the virtual model name
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if strings.Contains(string(data), "gpt-4__key_1") {
		t.Errorf("saved config should not contain virtual model name 'gpt-4__key_1'")
	}
}

// TestConfig_Complete verifies all config fields are set
func TestConfig_Complete(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Agents.Defaults.Workspace == "" {
		t.Error("Workspace should not be empty")
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
	if cfg.Gateway.Host != "localhost" {
		t.Error("Gateway host should have default value")
	}
	if cfg.Gateway.Port == 0 {
		t.Error("Gateway port should have default value")
	}
	if !cfg.Heartbeat.Enabled {
		t.Error("Heartbeat should be enabled by default")
	}
	if !cfg.Tools.Exec.AllowRemote {
		t.Error("Exec.AllowRemote should be true by default")
	}
}

func TestDefaultConfig_WebPreferNativeEnabled(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.Tools.Web.PreferNative {
		t.Fatal("DefaultConfig().Tools.Web.PreferNative should be true")
	}
}

func TestDefaultConfig_WebProviderIsAuto(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Tools.Web.Provider != "auto" {
		t.Fatalf("DefaultConfig().Tools.Web.Provider = %q, want auto", cfg.Tools.Web.Provider)
	}
}

func TestConfigExample_WebProviderIsAuto(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "config", "config.example.json"))
	if err != nil {
		t.Fatalf("ReadFile(config.example.json) error: %v", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("Unmarshal(config.example.json) error: %v", err)
	}
	if cfg.Tools.Web.Provider != "auto" {
		t.Fatalf("config.example.json tools.web.provider = %q, want auto", cfg.Tools.Web.Provider)
	}
}

func TestDefaultConfig_ToolFeedbackDisabled(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Agents.Defaults.ToolFeedback.Enabled {
		t.Fatal("DefaultConfig().Agents.Defaults.ToolFeedback.Enabled should be false")
	}
}

func TestLoadConfig_ToolFeedbackDefaultsFalseWhenUnset(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(
		configPath,
		[]byte(`{"version":1,"agents":{"defaults":{"workspace":"./workspace"}}}`),
		0o600,
	); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}
	if cfg.Agents.Defaults.ToolFeedback.Enabled {
		t.Fatal("agents.defaults.tool_feedback.enabled should remain false when unset in config file")
	}
}

func TestLoadConfig_WebPreferNativeDefaultsTrueWhenUnset(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"version":1,"tools":{"web":{"enabled":true}}}`), 0o600); err != nil {
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

func TestDefaultConfig_FilterSensitiveDataEnabled(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.Tools.FilterSensitiveData {
		t.Fatal("DefaultConfig().Tools.FilterSensitiveData should be true")
	}
}

func TestDefaultConfig_FilterMinLength(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Tools.FilterMinLength != 8 {
		t.Fatalf("DefaultConfig().Tools.FilterMinLength = %d, want 8", cfg.Tools.FilterMinLength)
	}
}

func TestToolsConfig_GetFilterMinLength(t *testing.T) {
	tests := []struct {
		name     string
		minLen   int
		expected int
	}{
		{"zero returns default", 0, 8},
		{"negative returns default", -1, 8},
		{"positive returns value", 16, 16},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &ToolsConfig{FilterMinLength: tt.minLen}
			if got := cfg.GetFilterMinLength(); got != tt.expected {
				t.Errorf("GetFilterMinLength() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDefaultConfig_CronAllowCommandEnabled(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.Tools.Cron.AllowCommand {
		t.Fatal("DefaultConfig().Tools.Cron.AllowCommand should be true")
	}
}

func TestDefaultConfig_HooksDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.Hooks.Enabled {
		t.Fatal("DefaultConfig().Hooks.Enabled should be true")
	}
	if cfg.Hooks.Defaults.ObserverTimeoutMS != 500 {
		t.Fatalf("ObserverTimeoutMS = %d, want 500", cfg.Hooks.Defaults.ObserverTimeoutMS)
	}
	if cfg.Hooks.Defaults.InterceptorTimeoutMS != 5000 {
		t.Fatalf("InterceptorTimeoutMS = %d, want 5000", cfg.Hooks.Defaults.InterceptorTimeoutMS)
	}
	if cfg.Hooks.Defaults.ApprovalTimeoutMS != 60000 {
		t.Fatalf("ApprovalTimeoutMS = %d, want 60000", cfg.Hooks.Defaults.ApprovalTimeoutMS)
	}
}

func TestDefaultConfig_LogLevel(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Gateway.LogLevel != "warn" {
		t.Errorf("LogLevel = %q, want \"fatal\"", cfg.Gateway.LogLevel)
	}
}

func TestLoadConfig_ExecAllowRemoteDefaultsTrueWhenUnset(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"version":1,"tools":{"exec":{"enable_deny_patterns":true}}}`),
		0o600); err != nil {
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
	if err := os.WriteFile(
		configPath,
		[]byte(`{"version":1,"tools":{"cron":{"exec_timeout_minutes":5}}}`),
		0o600,
	); err != nil {
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

func TestLoadConfig_HooksProcessConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	configJSON := `{
  "version": 1,
  "hooks": {
    "processes": {
      "review-gate": {
        "enabled": true,
        "transport": "stdio",
        "command": ["uvx", "picoclaw-hook-reviewer"],
        "dir": "/tmp/hooks",
        "env": {
          "HOOK_MODE": "rewrite"
        },
        "observe": ["turn_start", "turn_end"],
        "intercept": ["before_tool", "approve_tool"]
      }
    },
    "builtins": {
      "audit": {
        "enabled": true,
        "priority": 5,
        "config": {
          "label": "audit"
        }
      }
    }
  }
}`
	if err := os.WriteFile(configPath, []byte(configJSON), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}

	processCfg, ok := cfg.Hooks.Processes["review-gate"]
	if !ok {
		t.Fatal("expected review-gate process hook")
	}
	if !processCfg.Enabled {
		t.Fatal("expected review-gate process hook to be enabled")
	}
	if processCfg.Transport != "stdio" {
		t.Fatalf("Transport = %q, want stdio", processCfg.Transport)
	}
	if len(processCfg.Command) != 2 || processCfg.Command[0] != "uvx" {
		t.Fatalf("Command = %v", processCfg.Command)
	}
	if processCfg.Dir != "/tmp/hooks" {
		t.Fatalf("Dir = %q, want /tmp/hooks", processCfg.Dir)
	}
	if processCfg.Env["HOOK_MODE"] != "rewrite" {
		t.Fatalf("HOOK_MODE = %q, want rewrite", processCfg.Env["HOOK_MODE"])
	}
	if len(processCfg.Observe) != 2 || processCfg.Observe[1] != "turn_end" {
		t.Fatalf("Observe = %v", processCfg.Observe)
	}
	if len(processCfg.Intercept) != 2 || processCfg.Intercept[1] != "approve_tool" {
		t.Fatalf("Intercept = %v", processCfg.Intercept)
	}

	builtinCfg, ok := cfg.Hooks.Builtins["audit"]
	if !ok {
		t.Fatal("expected audit builtin hook")
	}
	if !builtinCfg.Enabled {
		t.Fatal("expected audit builtin hook to be enabled")
	}
	if builtinCfg.Priority != 5 {
		t.Fatalf("Priority = %d, want 5", builtinCfg.Priority)
	}
	if !strings.Contains(string(builtinCfg.Config), `"audit"`) {
		t.Fatalf("Config = %s", string(builtinCfg.Config))
	}
	if cfg.Hooks.Defaults.ApprovalTimeoutMS != 60000 {
		t.Fatalf("ApprovalTimeoutMS = %d, want 60000", cfg.Hooks.Defaults.ApprovalTimeoutMS)
	}
}

// TestDefaultConfig_SessionDimensions verifies the default session dimensions
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

func TestDefaultConfig_SessionDimensions(t *testing.T) {
	cfg := DefaultConfig()

	if len(cfg.Session.Dimensions) != 1 || cfg.Session.Dimensions[0] != "chat" {
		t.Errorf("Session.Dimensions = %v, want [chat]", cfg.Session.Dimensions)
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

func TestDefaultConfig_IsolationEnabled(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Isolation.Enabled {
		t.Fatal("DefaultConfig().Isolation.Enabled should be false")
	}
}

func TestConfig_UnmarshalIsolation(t *testing.T) {
	cfg := DefaultConfig()
	raw := []byte(`{
		"isolation": {
			"enabled": false,
			"expose_paths": [
				{"source":"/src","target":"/dst","mode":"ro"}
			]
		}
	}`)
	if err := json.Unmarshal(raw, cfg); err != nil {
		t.Fatalf("json.Unmarshal isolation config: %v", err)
	}
	if cfg.Isolation.Enabled {
		t.Fatal("Isolation.Enabled should be false after unmarshal")
	}
	if len(cfg.Isolation.ExposePaths) != 1 {
		t.Fatalf("ExposePaths len = %d, want 1", len(cfg.Isolation.ExposePaths))
	}
	if got := cfg.Isolation.ExposePaths[0]; got.Source != "/src" || got.Target != "/dst" || got.Mode != "ro" {
		t.Fatalf("ExposePaths[0] = %+v, want source=/src target=/dst mode=ro", got)
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

func TestFlexibleStringSlice_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "null",
			input:    `null`,
			expected: nil,
		},
		{
			name:     "single string",
			input:    `"Thinking..."`,
			expected: []string{"Thinking..."},
		},
		{
			name:     "single number",
			input:    `123`,
			expected: []string{"123"},
		},
		{
			name:     "string array",
			input:    `["Thinking...", "Still working..."]`,
			expected: []string{"Thinking...", "Still working..."},
		},
		{
			name:     "mixed array",
			input:    `["123", 456]`,
			expected: []string{"123", "456"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f FlexibleStringSlice
			if err := json.Unmarshal([]byte(tt.input), &f); err != nil {
				t.Fatalf("json.Unmarshal(%s) error = %v", tt.input, err)
			}
			if tt.expected == nil {
				if f != nil {
					t.Fatalf("json.Unmarshal(%s) = %#v, want nil slice", tt.input, f)
				}
				return
			}
			if len(f) != len(tt.expected) {
				t.Fatalf("json.Unmarshal(%s) len = %d, want %d", tt.input, len(f), len(tt.expected))
			}
			for i, want := range tt.expected {
				if f[i] != want {
					t.Fatalf("json.Unmarshal(%s)[%d] = %q, want %q", tt.input, i, f[i], want)
				}
			}
		})
	}
}

func TestLoadConfig_TelegramPlaceholderTextAcceptsSingleString(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	data := `{
		"version": 1,
		"agents": { "defaults": { "workspace": "", "model": "", "max_tokens": 0, "max_tool_iterations": 0 } },
		"session": {},
		"channels": {
			"telegram": {
				"enabled": true,
				"bot_token": "",
				"allow_from": [],
				"placeholder": {
					"enabled": true,
					"text": "Thinking..."
				}
			}
		},
		"model_list": [],
		"gateway": {},
		"tools": {},
		"heartbeat": {},
		"devices": {},
		"voice": {}
	}`
	if err := os.WriteFile(cfgPath, []byte(data), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	bc := cfg.Channels.Get("telegram")
	if got := []string(bc.Placeholder.Text); len(got) != 1 || got[0] != "Thinking..." {
		t.Fatalf("placeholder.text = %#v, want [\"Thinking...\"]", got)
	}
}

// TestLoadConfig_WarnsForPlaintextAPIKey verifies that LoadConfig resolves a plaintext
// api_key into memory but does NOT rewrite the config file. File writes are the sole
// responsibility of SaveConfig.
func TestLoadConfig_WarnsForPlaintextAPIKey(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	const original = `{"version":1,"model_list":[{"model_name":"test","model":"openai/gpt-4","api_key":"sk-plaintext"}]}`
	if err := os.WriteFile(cfgPath, []byte(original), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	secPath := filepath.Join(dir, SecurityConfigFile)
	const securityConfig = `
model_list:
  test:0:
    api_keys:
      - "sk-plaintext"
`
	if err := os.WriteFile(secPath, []byte(securityConfig), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}
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
	if cfg.ModelList[0].APIKey() != "sk-plaintext" {
		t.Errorf("in-memory api_key = %q, want %q", cfg.ModelList[0].APIKey(), "sk-plaintext")
	}
	// The file on disk must remain unchanged — no need upgrade version
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
	cfg.ModelList = []*ModelConfig{
		{ModelName: "test", Model: "openai/gpt-4", APIKeys: SimpleSecureStrings("")},
	}
	cfg.ModelList[0].APIKeys[0].Set("sk-plaintext")

	if err := SaveConfig(cfgPath, cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	// Disk must contain enc://, not the raw key.
	secPath := filepath.Join(dir, SecurityConfigFile)
	raw, _ := os.ReadFile(secPath)
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
	if cfg2.ModelList[0].APIKey() != "sk-plaintext" {
		t.Errorf("loaded api_key = %q, want %q", cfg2.ModelList[0].APIKey(), "sk-plaintext")
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
	data := `{"version":1,"model_list":[{"model_name":"test","model":"openai/gpt-4"}]}`
	if err := os.WriteFile(cfgPath, []byte(data), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	secPath := filepath.Join(dir, SecurityConfigFile)
	if err := saveSecurityConfig(
		secPath,
		&Config{ModelList: SecureModelList{
			&ModelConfig{ModelName: "test", APIKeys: SimpleSecureStrings("file://openai.key")},
		}}); err != nil {
		t.Fatalf("saveSecurityConfig: %v", err)
	}

	t.Setenv("PICOCLAW_KEY_PASSPHRASE", "test-passphrase")
	t.Setenv("PICOCLAW_SSH_KEY_PATH", "")

	if _, err := LoadConfig(cfgPath); err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	raw, _ := os.ReadFile(secPath)
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
		ModelList: []*ModelConfig{
			{ModelName: "pre", Model: "openai/gpt-4", APIKeys: SimpleSecureStrings("sk-already-plain")},
		},
	}); err != nil {
		t.Fatalf("setup SaveConfig: %v", err)
	}
	raw, _ := os.ReadFile(filepath.Join(dir, SecurityConfigFile))
	// Extract the enc:// value from the saved file.
	var tmp struct {
		ModelList map[string]struct {
			APIKeys []string `yaml:"api_keys"`
		} `yaml:"model_list"`
	}
	if err := yaml.Unmarshal(raw, &tmp); err != nil || len(tmp.ModelList) == 0 {
		t.Fatalf("setup: could not parse saved config: %v", err)
	}
	alreadyEncrypted := tmp.ModelList["pre:0"].APIKeys[0]
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
		Version: CurrentVersion,
		ModelList: []*ModelConfig{
			{ModelName: "plain", Model: "openai/gpt-4", APIKeys: SimpleSecureStrings("sk-new-plaintext")},
			{ModelName: "enc", Model: "openai/gpt-4", APIKeys: SimpleSecureStrings(alreadyEncrypted)},
			{ModelName: "file", Model: "openai/gpt-4", APIKeys: SimpleSecureStrings("file://api.key")},
		},
	}
	if err := SaveConfig(cfgPath, cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	t.Logf("alreadyEncrypted: %s", alreadyEncrypted)
	raw, _ = os.ReadFile(filepath.Join(dir, SecurityConfigFile))
	s := string(raw)

	t.Logf("saved file:\n%s", s)

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
		byName[m.ModelName] = m.APIKey()
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
		Version: CurrentVersion,
		ModelList: []*ModelConfig{
			{ModelName: "m", Model: "openai/gpt-4", APIKeys: SimpleSecureStrings("sk-secret")},
		},
	}); err != nil {
		t.Fatalf("setup SaveConfig: %v", err)
	}
	raw, err := LoadConfig(cfgPath)
	assert.NoError(t, err)
	encValue := raw.ModelList[0].APIKeys[0].raw
	assert.NotEmpty(t, encValue)
	assert.Equal(t, "enc://", encValue[:6])

	// Write a mixed config: enc:// + plaintext + file://
	keyFile := filepath.Join(dir, "api.key")
	if err = os.WriteFile(keyFile, []byte("sk-from-file"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	mixed, _ := json.Marshal(map[string]any{
		"model_list": []map[string]any{
			{"model_name": "enc", "model": "openai/gpt-4", "api_key": encValue},
			{"model_name": "plain", "model": "openai/gpt-4", "api_key": "sk-plain"},
			{"model_name": "file", "model": "openai/gpt-4", "api_key": "file://api.key"},
		},
	})
	if err = os.WriteFile(cfgPath, mixed, 0o600); err != nil {
		t.Fatalf("setup write: %v", err)
	}
	secs, _ := yaml.Marshal(map[string]any{
		"model_list": map[string]map[string]any{
			"enc:0":   {"api_keys": []string{encValue}},
			"plain:0": {"api_keys": []string{"sk-plain"}},
			"file:0":  {"api_keys": []string{"file://api.key"}},
		},
	})
	if err = os.WriteFile(filepath.Join(dir, SecurityConfigFile), secs, 0o600); err != nil {
		t.Fatalf("security write: %v", err)
	}

	// Now clear the passphrase — LoadConfig must fail because enc:// cannot be decrypted.
	t.Setenv("PICOCLAW_KEY_PASSPHRASE", "")

	cfg2, err := LoadConfig(cfgPath)
	if err == nil {
		t.Logf("LoadConfig: %#v", cfg2.ModelList)
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
	cfg.ModelList = []*ModelConfig{
		{ModelName: "test", Model: "openai/gpt-4", APIKeys: SimpleSecureStrings("sk-plaintext")},
	}
	if err := SaveConfig(cfgPath, cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	raw, _ := os.ReadFile(filepath.Join(dir, SecurityConfigFile))
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

	t.Logf("cfgPath: %s", cfgPath)

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.ModelList[0].APIKey() != plainKey {
		t.Errorf("api_key = %q, want %q", cfg.ModelList[0].APIKey(), plainKey)
	}
}

func TestConfigParsesLogLevel(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	data := `{"version":1,"gateway":{"log_level":"debug"}}`
	if err := os.WriteFile(cfgPath, []byte(data), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Gateway.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want \"debug\"", cfg.Gateway.LogLevel)
	}
}

func TestConfigLogLevelEmpty(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	data := `{"version":1}`
	if err := os.WriteFile(cfgPath, []byte(data), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	// When config omits log_level, the DefaultConfig value ("fatal") is preserved.
	if cfg.Gateway.LogLevel != "warn" {
		t.Errorf("LogLevel = %q, want \"fatal\"", cfg.Gateway.LogLevel)
	}
}

func TestResolveGatewayLogLevel(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	data := `{"version":1,"gateway":{"log_level":"debug"}}`
	if err := os.WriteFile(cfgPath, []byte(data), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if got := ResolveGatewayLogLevel(cfgPath); got != "debug" {
		t.Fatalf("ResolveGatewayLogLevel() = %q, want %q", got, "debug")
	}
}

func TestResolveGatewayLogLevel_UsesEnvOverrideAndNormalizesInvalid(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	data := `{"version":1,"gateway":{"log_level":"debug"}}`
	if err := os.WriteFile(cfgPath, []byte(data), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	t.Setenv("PICOCLAW_LOG_LEVEL", "warning")
	if got := ResolveGatewayLogLevel(cfgPath); got != "warn" {
		t.Fatalf("ResolveGatewayLogLevel() with env override = %q, want %q", got, "warn")
	}

	t.Setenv("PICOCLAW_LOG_LEVEL", "garbage")
	if got := ResolveGatewayLogLevel(cfgPath); got != DefaultGatewayLogLevel {
		t.Fatalf("ResolveGatewayLogLevel() with invalid env override = %q, want %q", got, DefaultGatewayLogLevel)
	}
}

func TestLoadConfig_AppliesLegacyClawHubRegistryEnvOverrides(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	data := `{"version":2,"tools":{"skills":{"registries":{"clawhub":{"enabled":true,"base_url":"https://clawhub.ai"}}}}}`
	if err := os.WriteFile(cfgPath, []byte(data), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	t.Setenv(envSkillsClawHubBaseURL, "https://clawhub.example.com")
	t.Setenv(envSkillsClawHubAuthToken, "clawhub-token-from-env")
	t.Setenv(envSkillsClawHubEnabled, "false")
	t.Setenv(envSkillsClawHubSearchPath, "/custom/search")
	t.Setenv(envSkillsClawHubDownloadPath, "/custom/download")
	t.Setenv(envSkillsClawHubTimeout, "17")

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	clawhub, ok := cfg.Tools.Skills.Registries.Get("clawhub")
	if !ok {
		t.Fatal("clawhub registry missing")
	}
	if clawhub.BaseURL != "https://clawhub.example.com" {
		t.Fatalf("BaseURL = %q, want %q", clawhub.BaseURL, "https://clawhub.example.com")
	}
	if clawhub.AuthToken.String() != "clawhub-token-from-env" {
		t.Fatalf("AuthToken = %q, want %q", clawhub.AuthToken.String(), "clawhub-token-from-env")
	}
	if clawhub.Enabled {
		t.Fatal("Enabled = true, want false")
	}
	if got := clawhub.Param["search_path"]; got != "/custom/search" {
		t.Fatalf("search_path = %v, want %q", got, "/custom/search")
	}
	if got := clawhub.Param["download_path"]; got != "/custom/download" {
		t.Fatalf("download_path = %v, want %q", got, "/custom/download")
	}
	if got := clawhub.Param["timeout"]; got != 17 {
		t.Fatalf("timeout = %v, want %d", got, 17)
	}
}

func TestLoadConfig_AppliesGitHubRegistryEnvOverrides(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	data := `{"version":2,"tools":{"skills":{"registries":{"github":{"enabled":true,"base_url":"https://github.com"}}}}}`
	if err := os.WriteFile(cfgPath, []byte(data), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	t.Setenv(envSkillsGitHubBaseURL, "https://ghe.example.com/git")
	t.Setenv(envSkillsGitHubAuthToken, "github-token-from-env")
	t.Setenv(envSkillsGitHubEnabled, "false")
	t.Setenv(envSkillsGitHubProxy, "http://127.0.0.1:7890")

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	github, ok := cfg.Tools.Skills.Registries.Get("github")
	if !ok {
		t.Fatal("github registry missing")
	}
	if github.BaseURL != "https://ghe.example.com/git" {
		t.Fatalf("BaseURL = %q, want %q", github.BaseURL, "https://ghe.example.com/git")
	}
	if github.AuthToken.String() != "github-token-from-env" {
		t.Fatalf("AuthToken = %q, want %q", github.AuthToken.String(), "github-token-from-env")
	}
	if github.Enabled {
		t.Fatal("Enabled = true, want false")
	}
	if got := github.Param["proxy"]; got != "http://127.0.0.1:7890" {
		t.Fatalf("proxy = %v, want %q", got, "http://127.0.0.1:7890")
	}
}

func TestModelConfig_ExtraBodyRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	cfg := &Config{
		Version: CurrentVersion,
		ModelList: []*ModelConfig{
			{
				ModelName: "test-model",
				Model:     "openai/test",
				APIKeys:   SimpleSecureStrings("sk-test"),
				ExtraBody: map[string]any{"custom_field": "value", "num_field": 42},
			},
		},
	}

	if err := SaveConfig(cfgPath, cfg); err != nil {
		t.Fatalf("SaveConfig error: %v", err)
	}

	loaded, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}

	if loaded.ModelList[0].ExtraBody == nil {
		t.Fatal("ExtraBody should not be nil after round-trip")
	}
	if got := loaded.ModelList[0].ExtraBody["custom_field"]; got != "value" {
		t.Errorf("ExtraBody[custom_field] = %v, want value", got)
	}
	if got := loaded.ModelList[0].ExtraBody["num_field"]; got != float64(42) {
		t.Errorf("ExtraBody[num_field] = %v, want 42", got)
	}
}

func TestModelConfig_CustomHeadersRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	cfg := &Config{
		Version: CurrentVersion,
		ModelList: []*ModelConfig{
			{
				ModelName:     "test-model",
				Model:         "openai/test",
				APIKeys:       SimpleSecureStrings("sk-test"),
				CustomHeaders: map[string]string{"X-Source": "coding-plan", "X-Agent": "openclaw"},
			},
		},
	}

	if err := SaveConfig(cfgPath, cfg); err != nil {
		t.Fatalf("SaveConfig error: %v", err)
	}

	loaded, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}

	if loaded.ModelList[0].CustomHeaders == nil {
		t.Fatal("CustomHeaders should not be nil after round-trip")
	}
	if got := loaded.ModelList[0].CustomHeaders["X-Source"]; got != "coding-plan" {
		t.Errorf("CustomHeaders[X-Source] = %q, want coding-plan", got)
	}
	if got := loaded.ModelList[0].CustomHeaders["X-Agent"]; got != "openclaw" {
		t.Errorf("CustomHeaders[X-Agent] = %q, want openclaw", got)
	}
}

func TestDefaultConfig_MinimaxExtraBody(t *testing.T) {
	cfg := DefaultConfig()

	var minimaxCfg *ModelConfig
	for i := range cfg.ModelList {
		if cfg.ModelList[i].Model == "minimax/MiniMax-M2.5" {
			minimaxCfg = cfg.ModelList[i]
			break
		}
	}
	if minimaxCfg == nil {
		t.Fatal("Minimax model not found in ModelList")
	}
	if minimaxCfg.ExtraBody == nil {
		t.Fatal("Minimax ExtraBody should not be nil")
	}
	if got, ok := minimaxCfg.ExtraBody["reasoning_split"]; !ok || got != true {
		t.Fatalf("Minimax ExtraBody[reasoning_split] = %v, want true", got)
	}
}

func TestFilterSensitiveData(t *testing.T) {
	// Test with nil security config
	cfg := &Config{}
	if got := cfg.FilterSensitiveData("hello sk-key123 world"); got != "hello sk-key123 world" {
		t.Errorf("nil security: got %q, want original", got)
	}

	// Test with empty content
	if got := cfg.FilterSensitiveData(""); got != "" {
		t.Errorf("empty content: got %q, want empty", got)
	}

	// Test short content (less than FilterMinLength=8, should skip filtering)
	cfg.ModelList = SecureModelList{
		&ModelConfig{
			ModelName: "test",
			APIKeys:   SimpleSecureStrings("sk-long-key-12345"),
		},
	}
	m, err := cfg.GetModelConfig("test")
	assert.NoError(t, err)
	m.APIKeys = SimpleSecureStrings("sk-long-key-12345")
	cfg.Tools.FilterSensitiveData = true
	cfg.Tools.FilterMinLength = 8

	// Debug: check if sensitive values are collected
	values := cfg.collectSensitiveValues()
	t.Logf("collected %d sensitive values: %v", len(values), values)

	if got := cfg.FilterSensitiveData("sk-key"); got != "sk-key" {
		t.Errorf("short content should not be filtered: got %q", got)
	}

	// Test filtering works
	content := "Your API key is sk-long-key-12345 and token abc123"
	// abc123 is not in sensitive values, only sk-long-key-12345 should be filtered
	expected := "Your API key is [FILTERED] and token abc123"
	if got := cfg.FilterSensitiveData(content); got != expected {
		t.Errorf("filtering failed: got %q, want %q", got, expected)
	}

	// Test disabled filtering
	cfg.Tools.FilterSensitiveData = false
	if got := cfg.FilterSensitiveData(content); got != content {
		t.Errorf("disabled filtering: got %q, want original %q", got, content)
	}
}

func TestFilterSensitiveData_MultipleKeys(t *testing.T) {
	cfg := &Config{
		Tools: ToolsConfig{
			FilterSensitiveData: true,
			FilterMinLength:     8,
		},
		ModelList: SecureModelList{
			&ModelConfig{
				ModelName: "model1",
				Model:     "openai/model1",
				APIKeys:   SecureStrings{NewSecureString("key-one"), NewSecureString("key-two")},
			},
			&ModelConfig{
				ModelName: "model2",
				Model:     "openai/model2",
				APIKeys:   SecureStrings{NewSecureString("key-three")},
			},
		},
	}

	content := "key-one and key-two and key-three should be filtered"
	expected := "[FILTERED] and [FILTERED] and [FILTERED] should be filtered"
	if got := cfg.FilterSensitiveData(content); got != expected {
		t.Errorf("multiple keys: got %q, want %q", got, expected)
	}
}

func TestFilterSensitiveData_AllTokenTypes(t *testing.T) {
	cfg := &Config{
		// Model API keys
		ModelList: SecureModelList{
			&ModelConfig{
				ModelName: "test-model",
				APIKeys:   SecureStrings{NewSecureString("sk-model-key-12345")},
			},
		},
		// Channel tokens
		Channels: testChannelsConfigWithTokens(),
		Tools: ToolsConfig{
			FilterSensitiveData: true,
			FilterMinLength:     8,
			// Web tool API keys
			Web: WebToolsConfig{
				Brave:       BraveConfig{APIKeys: SecureStrings{NewSecureString("brave-api-key")}},
				Tavily:      TavilyConfig{APIKeys: SecureStrings{NewSecureString("tavily-api-key")}},
				Perplexity:  PerplexityConfig{APIKeys: SecureStrings{NewSecureString("perplexity-api-key")}},
				GLMSearch:   GLMSearchConfig{APIKey: *NewSecureString("glm-search-key")},
				BaiduSearch: BaiduSearchConfig{APIKey: *NewSecureString("baidu-search-key")},
			},
			// Skills tokens
			Skills: SkillsToolsConfig{
				Github: SkillsGithubConfig{Token: *NewSecureString("github-token-xyz")},
				Registries: SkillsRegistriesConfig{
					&SkillRegistryConfig{Name: "clawhub", AuthToken: *NewSecureString("clawhub-auth-token")},
				},
			},
		},
	}

	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "model_api_key",
			content: "Using model with key sk-model-key-12345",
			want:    "Using model with key [FILTERED]",
		},
		{
			name:    "telegram_token",
			content: "Telegram token: telegram-bot-token-abcdef",
			want:    "Telegram token: [FILTERED]",
		},
		{
			name:    "discord_token",
			content: "Discord token: discord-bot-token-xyz789",
			want:    "Discord token: [FILTERED]",
		},
		{
			name:    "slack_tokens",
			content: "Slack bot: xoxb-slack-bot-token, app: xapp-slack-app-token",
			want:    "Slack bot: [FILTERED], app: [FILTERED]",
		},
		{
			name:    "matrix_token",
			content: "Matrix access token: matrix-access-token-abc",
			want:    "Matrix access token: [FILTERED]",
		},
		{
			name:    "brave_api_key",
			content: "Brave key: brave-api-key",
			want:    "Brave key: [FILTERED]",
		},
		{
			name:    "tavily_api_key",
			content: "Tavily key: tavily-api-key",
			want:    "Tavily key: [FILTERED]",
		},
		{
			name:    "github_token",
			content: "GitHub token: github-token-xyz",
			want:    "GitHub token: [FILTERED]",
		},
		{
			name:    "irc_passwords",
			content: "IRC password: irc-password, nickserv: nickserv-pass",
			want:    "IRC password: [FILTERED], nickserv: [FILTERED]",
		},
		{
			name:    "mixed_content",
			content: "Model key sk-model-key-12345 and Telegram token telegram-bot-token-abcdef",
			want:    "Model key [FILTERED] and Telegram token [FILTERED]",
		},
		{
			name:    "short_key_not_filtered",
			content: "Key abc not filtered because length < 8",
			want:    "Key abc not filtered because length < 8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cfg.FilterSensitiveData(tt.content); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// makeBackup tests
// ---------------------------------------------------------------------------

// TestMakeBackup_WithDateSuffix verifies backup files include a date suffix.
func TestMakeBackup_WithDateSuffix(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"version":2}`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := makeBackup(configPath); err != nil {
		t.Fatalf("makeBackup: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}

	var hasDatedBackup bool
	for _, e := range entries {
		if matched, _ := filepath.Match("config.json.20*.bak", e.Name()); matched {
			hasDatedBackup = true
			// Verify backup content matches original
			bakPath := filepath.Join(dir, e.Name())
			data, err := os.ReadFile(bakPath)
			if err != nil {
				t.Fatalf("ReadFile backup: %v", err)
			}
			if string(data) != `{"version":2}` {
				t.Errorf("backup content = %q, want original content", string(data))
			}
			break
		}
	}
	if !hasDatedBackup {
		t.Error("expected backup file with date suffix pattern config.json.20*.bak")
	}
}

// TestMakeBackup_AlsoBacksSecurityFile verifies that the security config file
// is also backed up with the same date suffix.
func TestMakeBackup_AlsoBacksSecurityFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	secPath := securityPath(configPath)

	os.WriteFile(configPath, []byte(`{"version":2}`), 0o600)
	os.WriteFile(secPath, []byte(`model_list:\n  test:0:\n    api_keys:\n      - "sk-test"\n`), 0o600)

	if err := makeBackup(configPath); err != nil {
		t.Fatalf("makeBackup: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}

	configBackups := 0
	secBackups := 0
	for _, e := range entries {
		if matched, _ := filepath.Match("config.json.20*.bak", e.Name()); matched {
			configBackups++
		}
		if matched, _ := filepath.Match(".security.yml.20*.bak", e.Name()); matched {
			secBackups++
		}
	}
	if configBackups != 1 {
		t.Errorf("expected 1 config backup, got %d", configBackups)
	}
	if secBackups != 1 {
		t.Errorf("expected 1 security backup, got %d", secBackups)
	}
}

// TestMakeBackup_NonexistentFileSkipsBackup verifies that makeBackup returns nil
// when the config file does not exist (no error, no panic).
func TestMakeBackup_NonexistentFileSkipsBackup(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "nonexistent.json")

	if err := makeBackup(configPath); err != nil {
		t.Fatalf("makeBackup on nonexistent file should return nil, got: %v", err)
	}
}

// TestMakeBackup_OnlyConfigNoSecurity verifies backup succeeds when only
// the config file exists and no security file.
func TestMakeBackup_OnlyConfigNoSecurity(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	os.WriteFile(configPath, []byte(`{"version":2}`), 0o600)

	if err := makeBackup(configPath); err != nil {
		t.Fatalf("makeBackup: %v", err)
	}

	entries, _ := os.ReadDir(dir)
	configBackups := 0
	secBackups := 0
	for _, e := range entries {
		if matched, _ := filepath.Match("config.json.20*.bak", e.Name()); matched {
			configBackups++
		}
		if matched, _ := filepath.Match(".security.yml.20*.bak", e.Name()); matched {
			secBackups++
		}
	}
	if configBackups != 1 {
		t.Errorf("expected 1 config backup, got %d", configBackups)
	}
	if secBackups != 0 {
		t.Errorf("expected 0 security backups when no security file exists, got %d", secBackups)
	}
}

// TestMakeBackup_SameDateSuffix verifies that config and security backups
// share the same date suffix (they are created in the same makeBackup call).
func TestMakeBackup_SameDateSuffix(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	secPath := securityPath(configPath)

	os.WriteFile(configPath, []byte(`{"version":2}`), 0o600)
	os.WriteFile(secPath, []byte(`key: value`), 0o600)

	if err := makeBackup(configPath); err != nil {
		t.Fatalf("makeBackup: %v", err)
	}

	entries, _ := os.ReadDir(dir)
	var configDate, secDate string
	for _, e := range entries {
		name := e.Name()
		// Extract date part: after the last . before .bak
		// e.g. config.json.20260330.bak → 20260330
		if strings.HasPrefix(name, "config.json.") && strings.HasSuffix(name, ".bak") {
			configDate = strings.TrimPrefix(name, "config.json.")
			configDate = strings.TrimSuffix(configDate, ".bak")
		}
		if strings.HasPrefix(name, ".security.yml.") && strings.HasSuffix(name, ".bak") {
			secDate = strings.TrimPrefix(name, ".security.yml.")
			secDate = strings.TrimSuffix(secDate, ".bak")
		}
	}
	if configDate == "" {
		t.Fatal("config backup file not found")
	}
	if secDate == "" {
		t.Fatal("security backup file not found")
	}
	if configDate != secDate {
		t.Errorf("config backup date = %q, security backup date = %q, should match", configDate, secDate)
	}
}

func testChannelsConfigWithTokens() ChannelsConfig {
	channels := make(ChannelsConfig)
	type chDef struct {
		name string
		cfg  any
	}
	defs := []chDef{
		{"telegram", TelegramSettings{Token: *NewSecureString("telegram-bot-token-abcdef")}},
		{"discord", DiscordSettings{Token: *NewSecureString("discord-bot-token-xyz789")}},
		{
			"slack",
			SlackSettings{
				BotToken: *NewSecureString("xoxb-slack-bot-token"),
				AppToken: *NewSecureString("xapp-slack-app-token"),
			},
		},
		{"matrix", MatrixSettings{AccessToken: *NewSecureString("matrix-access-token-abc")}},
		{
			"feishu",
			FeishuSettings{
				AppSecret:  *NewSecureString("feishu-app-secret-123"),
				EncryptKey: *NewSecureString("feishu-encrypt-key"),
			},
		},
		{"dingtalk", DingTalkSettings{ClientSecret: *NewSecureString("dingtalk-client-secret")}},
		{"onebot", OneBotSettings{AccessToken: *NewSecureString("onebot-access-token")}},
		{"wecom", WeComSettings{Secret: *NewSecureString("wecom-secret")}},
		{"pico", PicoSettings{Token: *NewSecureString("pico-token-abc123")}},
		{
			"irc",
			IRCSettings{
				Password:         *NewSecureString("irc-password"),
				NickServPassword: *NewSecureString("nickserv-pass"),
				SASLPassword:     *NewSecureString("sasl-pass"),
			},
		},
	}
	for _, def := range defs {
		// Create Channel directly with settings to preserve SecureString values
		bc := &Channel{Type: def.name}
		bc.Decode(def.cfg)
		channels[def.name] = bc
	}
	return channels
}
