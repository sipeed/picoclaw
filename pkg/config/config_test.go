package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestAgentModelConfig_StringFormatViaLoadConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
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
				}
			]
		}
	}`

	if err := os.WriteFile(configPath, []byte(jsonData), 0o600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}

	sales := cfg.Agents.List[0]
	if sales.Model == nil || sales.Model.Primary != "gpt-4" {
		t.Errorf("sales.Model = %+v", sales.Model)
	}
}

func TestAgentModelConfig_ObjectFormatViaLoadConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
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
					"id": "support",
					"name": "Support Bot",
					"model": {
						"primary": "claude-opus",
						"fallbacks": ["gpt-4o-mini", "haiku"]
					}
				}
			]
		}
	}`

	if err := os.WriteFile(configPath, []byte(jsonData), 0o600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}

	support := cfg.Agents.List[0]
	if support.Model == nil || support.Model.Primary != "claude-opus" {
		t.Errorf("support.Model = %+v", support.Model)
	}
	if len(support.Model.Fallbacks) != 2 || support.Model.Fallbacks[0] != "gpt-4o-mini" || support.Model.Fallbacks[1] != "haiku" {
		t.Errorf("support.Model.Fallbacks = %v", support.Model.Fallbacks)
	}
}

func TestAgentConfig_FullParseViaLoadConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
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

	if err := os.WriteFile(configPath, []byte(jsonData), 0o600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}

	if len(cfg.Agents.List) != 2 {
		t.Fatalf("agents.list len = %d, want 2", len(cfg.Agents.List))
	}
}

func TestConfig_BackwardCompat_NoAgentsListViaLoadConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
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

	if err := os.WriteFile(configPath, []byte(jsonData), 0o600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
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

	if cfg.Agents.Defaults.Model == "" {
		t.Error("Model should not be empty")
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

	if cfg.Gateway.Host != "0.0.0.0" {
		t.Error("Gateway host should have default value")
	}
	if cfg.Gateway.Port == 0 {
		t.Error("Gateway port should have default value")
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
}

// TestDefaultConfig_WebTools verifies web tools config
func TestDefaultConfig_WebTools(t *testing.T) {
	cfg := DefaultConfig()

	// Verify web tools defaults
	if cfg.Tools.Web.Brave.MaxResults != 5 {
		t.Error("Expected Brave MaxResults 5, got ", cfg.Tools.Web.Brave.MaxResults)
	}
	if cfg.Tools.Web.Brave.APIKey != "" {
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
	if perm != 0600 {
		t.Errorf("config file has permission %04o, want 0600", perm)
	}
}

// TestConfig_Complete verifies all config fields are set
func TestConfig_Complete(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Agents.Defaults.Workspace == "" {
		t.Error("Workspace should not be empty")
	}
	if cfg.Agents.Defaults.Model == "" {
		t.Error("Model should not be empty")
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
	if cfg.Gateway.Host != "0.0.0.0" {
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
