package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

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

	// Just verify the workspace is set, don't compare exact paths
	// since expandHome behavior may differ based on environment
	if cfg.Agents.Defaults.Workspace == "" {
		t.Error("Workspace should not be empty")
	}
}

// TestDefaultConfig_Model verifies model is set
func TestDefaultConfig_Model(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.LLM.Model == "" {
		t.Error("LLM.Model should not be empty")
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

	if cfg.Agents.Defaults.Temperature == 0 {
		t.Error("Temperature should not be zero")
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
}

// TestDefaultConfig_LLM verifies LLM config defaults
func TestDefaultConfig_LLM(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.LLM.APIKey != "" {
		t.Error("LLM API key should be empty by default")
	}
	if cfg.LLM.BaseURL != "" {
		t.Error("LLM BaseURL should be empty by default")
	}
	if cfg.LLM.Model != "zhipu/glm-4.7" {
		t.Errorf("LLM Model = %q, want %q", cfg.LLM.Model, "zhipu/glm-4.7")
	}
}

// TestDefaultConfig_Channels verifies channels are disabled by default
func TestDefaultConfig_Channels(t *testing.T) {
	cfg := DefaultConfig()

	// Verify all channels are disabled by default
	if cfg.Channels.WhatsApp.Enabled {
		t.Error("WhatsApp should be disabled by default")
	}
	if cfg.Channels.Telegram.Enabled {
		t.Error("Telegram should be disabled by default")
	}
	if cfg.Channels.Feishu.Enabled {
		t.Error("Feishu should be disabled by default")
	}
	if cfg.Channels.Discord.Enabled {
		t.Error("Discord should be disabled by default")
	}
	if cfg.Channels.MaixCam.Enabled {
		t.Error("MaixCam should be disabled by default")
	}
	if cfg.Channels.QQ.Enabled {
		t.Error("QQ should be disabled by default")
	}
	if cfg.Channels.DingTalk.Enabled {
		t.Error("DingTalk should be disabled by default")
	}
	if cfg.Channels.Slack.Enabled {
		t.Error("Slack should be disabled by default")
	}
}

// TestDefaultConfig_ExecToolDisabled verifies exec tool is disabled by default
func TestDefaultConfig_ExecToolDisabled(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Tools.Exec.Enabled {
		t.Error("Exec tool should be disabled by default")
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

// TestDefaultConfig_DataDir verifies data dir default value
func TestDefaultConfig_DataDir(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Agents.Defaults.DataDir == "" {
		t.Error("DataDir should not be empty")
	}
	if cfg.Agents.Defaults.DataDir != "~/.clawdroid/data" {
		t.Errorf("DataDir should be '~/.clawdroid/data', got '%s'", cfg.Agents.Defaults.DataDir)
	}
}

// TestConfig_DataPath verifies DataPath expands home directory
func TestConfig_DataPath(t *testing.T) {
	cfg := DefaultConfig()

	path := cfg.DataPath()
	if path == "" {
		t.Error("DataPath should not be empty")
	}
	if path == "~/.clawdroid/data" {
		t.Error("DataPath should expand ~ to home directory")
	}
	if path[0] == '~' {
		t.Error("DataPath should not start with ~")
	}
}

// TestConfig_Complete verifies all config fields are set
func TestConfig_Complete(t *testing.T) {
	cfg := DefaultConfig()

	// Verify complete config structure
	if cfg.Agents.Defaults.Workspace == "" {
		t.Error("Workspace should not be empty")
	}
	if cfg.LLM.Model == "" {
		t.Error("LLM.Model should not be empty")
	}
	if cfg.Agents.Defaults.Temperature == 0 {
		t.Error("Temperature should have default value")
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
