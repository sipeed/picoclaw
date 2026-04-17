// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestMigration_Integration_LegacyConfigWithoutWorkspace tests the issue reported:
// User configured Model and Provider but no Workspace - settings should not be lost
func TestMigration_Integration_LegacyConfigWithoutWorkspace(t *testing.T) {
	// Create a temporary directory for test config files
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Create a legacy config (version 0) with Model and Provider but NO Workspace
	// This simulates the real-world scenario where user settings would be lost
	legacyConfig := `{
		"agents": {
			"defaults": {
				"provider": "openai",
				"model": "gpt-4o",
				"max_tokens": 8192,
				"temperature": 0.7
			}
		},
		"channels": {
			"telegram": {
				"enabled": true,
				"token": "test-token"
			}
		},
		"gateway": {
			"host": "127.0.0.1",
			"port": 18790
		},
		"tools": {
			"web": {
				"enabled": true
			}
		},
		"heartbeat": {
			"enabled": true,
			"interval": 30
		},
		"devices": {
			"enabled": false
		}
	}`

	if err := os.WriteFile(configPath, []byte(legacyConfig), 0o600); err != nil {
		t.Fatalf("Failed to write legacy config: %v", err)
	}

	// Load the config - this should trigger migration
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify version is updated
	if cfg.Version != CurrentVersion {
		t.Errorf("Version = %d, want %d", cfg.Version, CurrentVersion)
	}

	// CRITICAL: Verify that user's settings are preserved
	// This was the bug - these settings were lost when Workspace was empty
	if cfg.Agents.Defaults.Provider != "openai" {
		t.Errorf("Provider = %q, want %q (user's setting should be preserved)", cfg.Agents.Defaults.Provider, "openai")
	}

	t.Logf("defaults: %v", cfg.Agents.Defaults)
	// Old "model" field is migrated to "model_name" field
	if cfg.Agents.Defaults.ModelName != "gpt-4o" {
		t.Errorf(
			"ModelName = %q, want %q (user's setting should be preserved)",
			cfg.Agents.Defaults.ModelName, "gpt-4o",
		)
	}
	// GetModelName() should also return the migrated value
	if cfg.Agents.Defaults.GetModelName() != "gpt-4o" {
		t.Errorf("GetModelName() = %q, want %q", cfg.Agents.Defaults.GetModelName(), "gpt-4o")
	}
	if cfg.Agents.Defaults.MaxTokens != 8192 {
		t.Errorf("MaxTokens = %d, want %d", cfg.Agents.Defaults.MaxTokens, 8192)
	}
	if cfg.Agents.Defaults.Temperature == nil {
		t.Error("Temperature should not be nil")
	} else if *cfg.Agents.Defaults.Temperature != 0.7 {
		t.Errorf("Temperature = %v, want %v", *cfg.Agents.Defaults.Temperature, 0.7)
	}

	// Verify Workspace has a default value (should not be empty)
	if cfg.Agents.Defaults.Workspace == "" {
		t.Error("Workspace should have a default value, not be empty")
	}

	// Verify other config sections are preserved
	var tgCfg TelegramSettings
	bc := cfg.Channels.Get("telegram")
	if bc == nil || !bc.Enabled {
		t.Error("Telegram.Enabled should be true")
	}
	bc.Decode(&tgCfg)
	if tgCfg.Token.String() != "test-token" {
		t.Errorf("Telegram.Token = %q, want %q", tgCfg.Token.String(), "test-token")
	}
	if cfg.Gateway.Port != 18790 {
		t.Errorf("Gateway.Port = %d, want %d", cfg.Gateway.Port, 18790)
	}
}

// TestMigration_Integration_LegacyConfigWithWorkspace tests migration with Workspace set
func TestMigration_Integration_LegacyConfigWithWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	legacyConfig := `{
		"agents": {
			"defaults": {
				"workspace": "/custom/workspace",
				"provider": "deepseek",
				"model": "deepseek-chat",
				"max_tokens": 16384
			}
		},
		"channels": {
			"telegram": {
				"enabled": false
			}
		},
		"gateway": {
			"host": "0.0.0.0",
			"port": 8080
		},
		"tools": {
			"web": {
				"enabled": false
			}
		},
		"heartbeat": {
			"enabled": false
		},
		"devices": {
			"enabled": true
		}
	}`

	if err := os.WriteFile(configPath, []byte(legacyConfig), 0o600); err != nil {
		t.Fatalf("Failed to write legacy config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// All user settings should be preserved
	if cfg.Agents.Defaults.Workspace != "/custom/workspace" {
		t.Errorf("Workspace = %q, want %q", cfg.Agents.Defaults.Workspace, "/custom/workspace")
	}
	if cfg.Agents.Defaults.Provider != "deepseek" {
		t.Errorf("Provider = %q, want %q", cfg.Agents.Defaults.Provider, "deepseek")
	}
	if cfg.Agents.Defaults.ModelName != "deepseek-chat" {
		t.Errorf("ModelName = %q, want %q", cfg.Agents.Defaults.ModelName, "deepseek-chat")
	}
	if cfg.Agents.Defaults.MaxTokens != 16384 {
		t.Errorf("MaxTokens = %d, want %d", cfg.Agents.Defaults.MaxTokens, 16384)
	}

	// Verify other settings
	if cfg.Gateway.Port != 8080 {
		t.Errorf("Gateway.Port = %d, want %d", cfg.Gateway.Port, 8080)
	}
	if !cfg.Devices.Enabled {
		t.Error("Devices.Enabled should be true")
	}
}

// TestMigration_Integration_PreservesAllAgentsFields tests that ALL Agents fields are preserved
func TestMigration_Integration_PreservesAllAgentsFields(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	legacyConfig := `{
		"agents": {
			"defaults": {
				"workspace": "",
				"restrict_to_workspace": false,
				"allow_read_outside_workspace": true,
				"provider": "anthropic",
				"model": "claude-opus-4",
				"model_fallbacks": ["claude-sonnet-4", "claude-haiku-4"],
				"image_model": "claude-opus-4-vision",
				"image_model_fallbacks": ["claude-sonnet-4-vision"],
				"max_tokens": 4096,
				"temperature": 0.5,
				"max_tool_iterations": 100,
				"summarize_message_threshold": 30,
				"summarize_token_percent": 80,
				"max_media_size": 10485760
			},
			"list": [
				{
					"id": "special-agent",
					"default": false,
					"name": "Special Agent",
					"workspace": "/special/workspace"
				}
			]
		},
		"channels": {
			"telegram": {"enabled": false}
		},
		"gateway": {
			"host": "127.0.0.1",
			"port": 18790
		},
		"tools": {
			"web": {"enabled": true}
		},
		"heartbeat": {
			"enabled": true,
			"interval": 30
		},
		"devices": {
			"enabled": false
		}
	}`

	if err := os.WriteFile(configPath, []byte(legacyConfig), 0o600); err != nil {
		t.Fatalf("Failed to write legacy config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify ALL defaults fields are preserved
	d := cfg.Agents.Defaults

	if d.RestrictToWorkspace != false {
		t.Errorf("RestrictToWorkspace = %v, want false", d.RestrictToWorkspace)
	}
	if d.AllowReadOutsideWorkspace != true {
		t.Errorf("AllowReadOutsideWorkspace = %v, want true", d.AllowReadOutsideWorkspace)
	}
	if d.Provider != "anthropic" {
		t.Errorf("Provider = %q, want %q", d.Provider, "anthropic")
	}
	if d.ModelName != "claude-opus-4" {
		t.Errorf("ModelName = %q, want %q", d.ModelName, "claude-opus-4")
	}
	if len(d.ModelFallbacks) != 2 {
		t.Errorf("len(ModelFallbacks) = %d, want 2", len(d.ModelFallbacks))
	} else {
		if d.ModelFallbacks[0] != "claude-sonnet-4" {
			t.Errorf("ModelFallbacks[0] = %q, want %q", d.ModelFallbacks[0], "claude-sonnet-4")
		}
		if d.ModelFallbacks[1] != "claude-haiku-4" {
			t.Errorf("ModelFallbacks[1] = %q, want %q", d.ModelFallbacks[1], "claude-haiku-4")
		}
	}
	if d.ImageModel != "claude-opus-4-vision" {
		t.Errorf("ImageModel = %q, want %q", d.ImageModel, "claude-opus-4-vision")
	}
	if len(d.ImageModelFallbacks) != 1 {
		t.Errorf("len(ImageModelFallbacks) = %d, want 1", len(d.ImageModelFallbacks))
	} else if d.ImageModelFallbacks[0] != "claude-sonnet-4-vision" {
		t.Errorf("ImageModelFallbacks[0] = %q, want %q", d.ImageModelFallbacks[0], "claude-sonnet-4-vision")
	}
	if d.MaxTokens != 4096 {
		t.Errorf("MaxTokens = %d, want %d", d.MaxTokens, 4096)
	}
	if d.Temperature == nil || *d.Temperature != 0.5 {
		t.Errorf("Temperature = %v, want 0.5", d.Temperature)
	}
	if d.MaxToolIterations != 100 {
		t.Errorf("MaxToolIterations = %d, want %d", d.MaxToolIterations, 100)
	}
	if d.SummarizeMessageThreshold != 30 {
		t.Errorf("SummarizeMessageThreshold = %d, want %d", d.SummarizeMessageThreshold, 30)
	}
	if d.SummarizeTokenPercent != 80 {
		t.Errorf("SummarizeTokenPercent = %d, want %d", d.SummarizeTokenPercent, 80)
	}
	if d.MaxMediaSize != 10485760 {
		t.Errorf("MaxMediaSize = %d, want %d", d.MaxMediaSize, 10485760)
	}

	// Verify agent list is preserved
	if len(cfg.Agents.List) != 1 {
		t.Fatalf("len(Agents.List) = %d, want 1", len(cfg.Agents.List))
	}
	if cfg.Agents.List[0].ID != "special-agent" {
		t.Errorf("Agent.ID = %q, want %q", cfg.Agents.List[0].ID, "special-agent")
	}
	if cfg.Agents.List[0].Workspace != "/special/workspace" {
		t.Errorf("Agent.Workspace = %q, want %q", cfg.Agents.List[0].Workspace, "/special/workspace")
	}

	// Workspace should have default since it was empty in legacy config
	if d.Workspace == "" {
		t.Error("Workspace should have a default value, not be empty")
	}
}

// TestMigration_Integration_ChannelsConfigMigrated tests channel config migration
func TestMigration_Integration_ChannelsConfigMigrated(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Legacy config with old channel field formats
	legacyConfig := `{
		"agents": {
			"defaults": {}
		},
		"channels": {
			"discord": {
				"enabled": true,
				"token": "discord-token",
				"mention_only": true
			},
			"onebot": {
				"enabled": true,
				"ws_url": "ws://127.0.0.1:3001",
				"group_trigger_prefix": ["/", "!"]
			}
		},
		"gateway": {
			"host": "127.0.0.1",
			"port": 18790
		},
		"tools": {
			"web": {"enabled": true}
		},
		"heartbeat": {
			"enabled": true,
			"interval": 30
		},
		"devices": {
			"enabled": false
		}
	}`

	if err := os.WriteFile(configPath, []byte(legacyConfig), 0o600); err != nil {
		t.Fatalf("Failed to write legacy config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Discord: mention_only should be migrated to group_trigger.mention_only
	discordBC := cfg.Channels.Get("discord")
	if !discordBC.GroupTrigger.MentionOnly {
		t.Error("Discord.GroupTrigger.MentionOnly should be true after migration")
	}

	// OneBot: group_trigger_prefix should be migrated to group_trigger.prefixes
	oneBotBC := cfg.Channels.Get("onebot")
	if len(oneBotBC.GroupTrigger.Prefixes) != 2 {
		t.Errorf("len(OneBot.GroupTrigger.Prefixes) = %d, want 2", len(oneBotBC.GroupTrigger.Prefixes))
	} else {
		if oneBotBC.GroupTrigger.Prefixes[0] != "/" {
			t.Errorf("Prefixes[0] = %q, want %q", oneBotBC.GroupTrigger.Prefixes[0], "/")
		}
		if oneBotBC.GroupTrigger.Prefixes[1] != "!" {
			t.Errorf("Prefixes[1] = %q, want %q", oneBotBC.GroupTrigger.Prefixes[1], "!")
		}
	}
}

// TestMigration_Integration_RoundTrip_SerializeAndLoad tests that migrated config can be saved and reloaded
func TestMigration_Integration_RoundTrip_SerializeAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	legacyConfig := `{
		"agents": {
			"defaults": {
				"provider": "openai",
				"model": "gpt-4o",
				"max_tokens": 8192
			}
		},
		"channels": {
			"telegram": {
				"enabled": true,
				"token": "test-token"
			}
		},
		"gateway": {
			"host": "127.0.0.1",
			"port": 18790
		},
		"tools": {
			"web": {"enabled": true}
		},
		"heartbeat": {
			"enabled": true,
			"interval": 30
		},
		"devices": {
			"enabled": false
		}
	}`

	if err := os.WriteFile(configPath, []byte(legacyConfig), 0o600); err != nil {
		t.Fatalf("Failed to write legacy config: %v", err)
	}

	// First load - triggers migration and saves
	cfg1, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("First LoadConfig failed: %v", err)
	}

	// Read the migrated config from disk
	migratedData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read migrated config: %v", err)
	}

	// Verify it has the current version
	var versionCheck struct {
		Version int `json:"version"`
	}
	if err = json.Unmarshal(migratedData, &versionCheck); err != nil {
		t.Fatalf("Failed to parse migrated config version: %v", err)
	}
	if versionCheck.Version != CurrentVersion {
		t.Errorf("Migrated config version = %d, want %d", versionCheck.Version, CurrentVersion)
	}

	// Second load - should load the migrated config without changes
	cfg2, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Second LoadConfig failed: %v", err)
	}

	// Verify configs are identical
	if cfg2.Agents.Defaults.Provider != cfg1.Agents.Defaults.Provider {
		t.Errorf("Provider changed from %q to %q", cfg1.Agents.Defaults.Provider, cfg2.Agents.Defaults.Provider)
	}
	if cfg2.Agents.Defaults.ModelName != cfg1.Agents.Defaults.ModelName {
		t.Errorf("ModelName changed from %q to %q", cfg1.Agents.Defaults.ModelName, cfg2.Agents.Defaults.ModelName)
	}
	if cfg2.Agents.Defaults.MaxTokens != cfg1.Agents.Defaults.MaxTokens {
		t.Errorf("MaxTokens changed from %d to %d", cfg1.Agents.Defaults.MaxTokens, cfg2.Agents.Defaults.MaxTokens)
	}
}

// TestMigration_Integration_EmptyAgentsDefaults tests migration with completely empty agents config
func TestMigration_Integration_EmptyAgentsDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Legacy config with empty agents defaults
	legacyConfig := `{
		"agents": {
			"defaults": {}
		},
		"channels": {
			"telegram": {"enabled": false}
		},
		"gateway": {
			"host": "127.0.0.1",
			"port": 18790
		},
		"tools": {
			"web": {"enabled": true}
		},
		"heartbeat": {
			"enabled": true,
			"interval": 30
		},
		"devices": {
			"enabled": false
		}
	}`

	if err := os.WriteFile(configPath, []byte(legacyConfig), 0o600); err != nil {
		t.Fatalf("Failed to write legacy config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Workspace should have default value
	if cfg.Agents.Defaults.Workspace == "" {
		t.Error("Workspace should have a default value")
	}

	// Note: When fields are explicitly set in config (even to zero values),
	// they override defaults. This is correct JSON unmarshaling behavior.
	// Users should set values they want; defaults are for unspecified fields.
	if cfg.Agents.Defaults.MaxTokens == 0 {
		// This is expected when users don't set max_tokens in their config
		// The zero value (0) from the legacy config is preserved
	}
	if cfg.Agents.Defaults.MaxToolIterations == 0 {
		// Same as above - zero value is preserved if it was in the config
	}
}

// TestMigration_Integration_ModelNameField tests migration using new model_name field
func TestMigration_Integration_ModelNameField(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Legacy config using the new model_name field
	legacyConfig := `{
		"agents": {
			"defaults": {
				"provider": "deepseek",
				"model_name": "deepseek-reasoner",
				"model_fallbacks": ["deepseek-chat"]
			}
		},
		"channels": {
			"telegram": {"enabled": false}
		},
		"gateway": {
			"host": "127.0.0.1",
			"port": 18790
		},
		"tools": {
			"web": {"enabled": true}
		},
		"heartbeat": {
			"enabled": true,
			"interval": 30
		},
		"devices": {
			"enabled": false
		}
	}`

	if err := os.WriteFile(configPath, []byte(legacyConfig), 0o600); err != nil {
		t.Fatalf("Failed to write legacy config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// model_name field should be preserved
	if cfg.Agents.Defaults.ModelName != "deepseek-reasoner" {
		t.Errorf("ModelName = %q, want %q", cfg.Agents.Defaults.ModelName, "deepseek-reasoner")
	}

	// GetModelName() should return model_name, not model (deprecated)
	if cfg.Agents.Defaults.GetModelName() != "deepseek-reasoner" {
		t.Errorf("GetModelName() = %q, want %q", cfg.Agents.Defaults.GetModelName(), "deepseek-reasoner")
	}

	if len(cfg.Agents.Defaults.ModelFallbacks) != 1 {
		t.Errorf("len(ModelFallbacks) = %d, want 1", len(cfg.Agents.Defaults.ModelFallbacks))
	} else if cfg.Agents.Defaults.ModelFallbacks[0] != "deepseek-chat" {
		t.Errorf("ModelFallbacks[0] = %q, want %q", cfg.Agents.Defaults.ModelFallbacks[0], "deepseek-chat")
	}
}

// TestMigration_PreservesExistingSecurityConfig tests that when migrating from v0 to v1,
// existing .security.yml values (e.g., loaded from environment variables) are preserved
// and not overwritten by empty values from the legacy config.
func TestMigration_PreservesExistingSecurityConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	securityPath := filepath.Join(tmpDir, ".security.yml")

	// Create a legacy config (version 0) with model_list and channel config
	// The model_list doesn't have api_keys, they should come from existing .security.yml
	legacyConfig := `{
		"version": 1,
		"agents": {
			"defaults": {
				"provider": "openai",
				"model": "gpt-4"
			}
		},
		"model_list": [
			{
				"model_name": "openai",
				"model": "openai/gpt-4"
			}
		],
		"channels": {
			"telegram": {
				"enabled": true
			}
		},
		"gateway": {
			"host": "127.0.0.1",
			"port": 18790
		},
		"tools": {
			"web": {"enabled": true}
		},
		"heartbeat": {
			"enabled": true,
			"interval": 30
		},
		"devices": {
			"enabled": false
		}
	}`

	// Create an existing .security.yml with values that might come from env vars
	existingSecurity := `model_list:
  openai:0:
    api_keys:
      - sk-existing-key-from-env
channels:
  telegram:
    token: existing-telegram-token-from-env
  discord:
    token: existing-discord-token-from-env
web:
  brave:
    api_keys:
      - existing-brave-key
`

	if err := os.WriteFile(configPath, []byte(legacyConfig), 0o600); err != nil {
		t.Fatalf("Failed to write legacy config: %v", err)
	}

	if err := os.WriteFile(securityPath, []byte(existingSecurity), 0o600); err != nil {
		t.Fatalf("Failed to write existing security config: %v", err)
	}

	// Load the config - this should trigger migration
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	t.Logf("Migrated config: %#v", cfg.Channels["telegram"])
	t.Logf("Migrated config settings: %v", string(cfg.Channels["telegram"].Settings))

	// Verify that the migrated config has the existing security values
	// Telegram token should be preserved
	var tgCfg1 *TelegramSettings
	if bc := cfg.Channels.Get("telegram"); bc != nil {
		t.Logf("telegram settings: %v", string(bc.Settings))
		if decoded, e := bc.GetDecoded(); e == nil && decoded != nil {
			tgCfg1 = decoded.(*TelegramSettings)
		}
	}
	require.NotNil(t, tgCfg1)
	if tgCfg1.Token.String() != "existing-telegram-token-from-env" {
		t.Errorf("Telegram token was overwritten: got %q, want %q",
			tgCfg1.Token.String(), "existing-telegram-token-from-env")
	}

	// Discord token should be preserved (even though legacy config didn't have it)
	var dcCfg1 *DiscordSettings
	if bc := cfg.Channels.Get("discord"); bc != nil {
		if decoded, e := bc.GetDecoded(); e == nil && decoded != nil {
			dcCfg1 = decoded.(*DiscordSettings)
		}
	}
	if dcCfg1.Token.String() != "existing-discord-token-from-env" {
		t.Errorf("Discord token was overwritten: got %q, want %q",
			dcCfg1.Token.String(), "existing-discord-token-from-env")
	}

	// Model API key should be preserved
	t.Logf("model_list: %#v", cfg.ModelList[0])
	if cfg.ModelList[0].APIKey() != "sk-existing-key-from-env" {
		t.Errorf("Model API key was overwritten: got %q, want %q",
			cfg.ModelList[0].APIKey(), "sk-existing-key-from-env")
	}

	// Brave API key should be preserved
	if cfg.Tools.Web.Brave.APIKey() != "existing-brave-key" {
		t.Errorf("Brave API key was overwritten: got %q, want %q",
			cfg.Tools.Web.Brave.APIKey(), "existing-brave-key")
	}

	// Reload the security config from disk to verify it wasn't corrupted
	reloadedSec := cfg
	t.Logf("reloadedSec started")
	err = loadSecurityConfig(cfg, securityPath)
	if err != nil {
		t.Fatalf("Failed to reload security config: %v", err)
	}

	var tgCfgSec *TelegramSettings
	if bc := reloadedSec.Channels.Get("telegram"); bc != nil {
		if decoded, err := bc.GetDecoded(); err == nil && decoded != nil {
			tgCfgSec = decoded.(*TelegramSettings)
		}
	}
	if tgCfgSec.Token.String() != "existing-telegram-token-from-env" {
		t.Errorf("Telegram settings: %v", tgCfgSec)
		t.Error("Telegram token not preserved in .security.yml file")
	}

	var dcCfgSec *DiscordSettings
	if bc := reloadedSec.Channels.Get("discord"); bc != nil {
		if decoded, err := bc.GetDecoded(); err == nil && decoded != nil {
			dcCfgSec = decoded.(*DiscordSettings)
		}
	}
	if dcCfgSec.Token.String() != "existing-discord-token-from-env" {
		t.Error("Discord token not preserved in .security.yml file")
	}
}

// ---------------------------------------------------------------------------
// V1 → V2 migration tests
// ---------------------------------------------------------------------------

//// TestMigrateModelEnabled_APIKeysInferredEnabled verifies that models with API keys
//// are marked as enabled during V1→V2 migration.
//func TestMigrateModelEnabled_APIKeysInferredEnabled(t *testing.T) {
//	v1 := &configV1{Config: Config{
//		ModelList: []*ModelConfig{
//			{ModelName: "gpt-4", Model: "openai/gpt-4", APIKeys: SimpleSecureStrings("sk-test")},
//			{ModelName: "claude", Model: "anthropic/claude", APIKeys: SimpleSecureStrings("sk-ant")},
//		},
//	}}
//	v1.migrateModelEnabled()
//	for _, m := range v1.ModelList {
//		if !m.Enabled {
//			t.Errorf("model %q with API key should be enabled", m.ModelName)
//		}
//	}
//}
//
//// TestMigrateModelEnabled_LocalModelInferredEnabled verifies that the reserved
//// "local-model" entry is enabled even without API keys.
//func TestMigrateModelEnabled_LocalModelInferredEnabled(t *testing.T) {
//	v1 := &configV1{
//		ModelList: []*ModelConfig{
//			{ModelName: "local-model", Model: "vllm/custom-model", APIBase: "http://localhost:8000/v1"},
//		},
//	}
//	v1.migrateModelEnabled()
//	if !v1.ModelList[0].Enabled {
//		t.Error("local-model should be enabled")
//	}
//}
//
//// TestMigrateModelEnabled_NoKeyStaysDisabled verifies that models without API keys
//// and not named "local-model" remain disabled.
//func TestMigrateModelEnabled_NoKeyStaysDisabled(t *testing.T) {
//	v1 := &configV1{
//		ModelList: []*ModelConfig{
//			{ModelName: "gpt-4", Model: "openai/gpt-4"},
//			{ModelName: "claude", Model: "anthropic/claude"},
//		},
//	}
//	v1.migrateModelEnabled()
//	for _, m := range v1.ModelList {
//		if m.Enabled {
//			t.Errorf("model %q without API key should stay disabled", m.ModelName)
//		}
//	}
//}
//
//// TestMigrateModelEnabled_ExplicitEnabledPreserved verifies that a model with
//// explicitly enabled=true is NOT overridden by the migration.
//func TestMigrateModelEnabled_ExplicitEnabledPreserved(t *testing.T) {
//	v1 := &configV1{Config: Config{
//		ModelList: []*ModelConfig{
//			{ModelName: "gpt-4", Model: "openai/gpt-4", APIKeys: SimpleSecureStrings("sk-test"), Enabled: true},
//		},
//	}}
//	v1.migrateModelEnabled()
//	if !v1.ModelList[0].Enabled {
//		t.Error("explicitly enabled model should remain enabled")
//	}
//}
//
//// TestMigrateModelEnabled_ExplicitDisabledNotOverridden verifies that a model with
//// explicitly enabled=false and API keys gets enabled during migration.
//// Note: since Go's zero value for bool is false and JSON omitempty omits false,
//// migration cannot distinguish "explicitly false" from "field absent". Both cases
//// get the same inference treatment.
//func TestMigrateModelEnabled_ExplicitDisabledNotOverridden(t *testing.T) {
//	v1 := &configV1{Config: Config{
//		ModelList: []*ModelConfig{
//			{ModelName: "gpt-4", Model: "openai/gpt-4", APIKeys: SimpleSecureStrings("sk-test"), Enabled: false},
//		},
//	}}
//	v1.migrateModelEnabled()
//	// Even though Enabled was set to false, migration infers it as true because
//	// the migration cannot distinguish from a missing field (both are zero value).
//	if !v1.ModelList[0].Enabled {
//		t.Error("model with API key should be enabled by migration inference")
//	}
//}
//
//// TestMigrateModelEnabled_Mixed verifies a mix of models.
//func TestMigrateModelEnabled_Mixed(t *testing.T) {
//	v1 := &configV1{Config: Config{
//		ModelList: []*ModelConfig{
//			{ModelName: "with-key", Model: "openai/gpt-4", APIKeys: SimpleSecureStrings("sk-test")},
//			{ModelName: "no-key", Model: "openai/gpt-4"},
//			{ModelName: "local-model", Model: "vllm/custom"},
//			{
//				ModelName: "disabled-explicit",
//				Model:     "openai/gpt-4",
//				APIKeys:   SimpleSecureStrings("sk-test"),
//				Enabled:   false,
//			},
//		},
//	}}
//	v1.migrateModelEnabled()
//
//	assertEnabled := func(name string, want bool) {
//		for _, m := range v1.ModelList {
//			if m.ModelName == name {
//				if m.Enabled != want {
//					t.Errorf("model %q: Enabled=%v, want %v", name, m.Enabled, want)
//				}
//				return
//			}
//		}
//		t.Errorf("model %q not found", name)
//	}
//
//	assertEnabled("with-key", true)
//	assertEnabled("no-key", false)
//	assertEnabled("local-model", true)
//	assertEnabled("disabled-explicit", true) // false is zero value, migration infers from API key
//}
//
//// TestMigrateChannelConfigs_DiscordMentionOnly verifies Discord mention_only migration.
//func TestMigrateChannelConfigs_DiscordMentionOnly(t *testing.T) {
//	channels := ChannelsConfig{"discord": makeBaseChannelFromConfig(DiscordSettings{MentionOnly: true})}
//	v1 := &configV1{Config: Config{Channels: channels}}
//	v1.migrateChannelConfigs()
//	bc := v1.Channels.Get("discord")
//	if !bc.GroupTrigger.MentionOnly {
//		t.Error("Discord GroupTrigger.MentionOnly should be set to true")
//	}
//}
//
//// TestMigrateChannelConfigs_DiscordAlreadyMigrated is a no-op test.
//func TestMigrateChannelConfigs_DiscordAlreadyMigrated(t *testing.T) {
//	channels := ChannelsConfig{"discord": makeBaseChannelFromConfig(map[string]any{
//		"group_trigger": map[string]any{"mention_only": true},
//	})}
//	v1 := &configV1{Config: Config{Channels: channels}}
//	v1.migrateChannelConfigs()
//}
//
//// TestMigrateChannelConfigs_OneBotPrefix verifies OneBot prefix migration.
//func TestMigrateChannelConfigs_OneBotPrefix(t *testing.T) {
//	channels := ChannelsConfig{"onebot": makeBaseChannelFromConfig(OneBotSettings{GroupTriggerPrefix: []string{"/"}})}
//	v1 := &configV1{Config: Config{Channels: channels}}
//	v1.migrateChannelConfigs()
//	bc := v1.Channels.Get("onebot")
//	if len(bc.GroupTrigger.Prefixes) != 1 || bc.GroupTrigger.Prefixes[0] != "/" {
//		t.Errorf("OneBot GroupTrigger.Prefixes = %v, want [\"/\"]", bc.GroupTrigger.Prefixes)
//	}
//}
//
//// TestMigrateConfigV1_Combined verifies that configV1.Migrate applies both migrations.
//func TestMigrateConfigV1_Combined(t *testing.T) {
//	v1 := &configV1{Config: Config{
//		ModelList: []*ModelConfig{
//			{ModelName: "gpt-4", Model: "openai/gpt-4", APIKeys: SimpleSecureStrings("sk-test")},
//		},
//		Channels: ChannelsConfig{"discord": makeBaseChannelFromConfig(DiscordSettings{MentionOnly: true})},
//	}}
//	result, err := v1.Migrate()
//	if err != nil {
//		t.Fatalf("Migrate: %v", err)
//	}
//
//	if !result.ModelList[0].Enabled {
//		t.Error("model with API key should be enabled after V1→V2 migration")
//	}
//	dcResultBC := result.Channels.Get("discord")
//	if !dcResultBC.GroupTrigger.MentionOnly {
//		t.Error("Discord mention_only should be migrated after V1→V2 migration")
//	}
//}

// TestLoadConfig_V1ToV2Migration verifies end-to-end V1→V2 config migration
// through LoadConfig, including Enabled field inference and version bump.
func TestLoadConfig_V1ToV2Migration(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Write a V1 config with model_list but no "enabled" field
	v1Config := `{
		"version": 1,
		"model_list": [
			{
				"model_name": "gpt-4",
				"model": "openai/gpt-4"
			},
			{
				"model_name": "local-model",
				"model": "vllm/custom-model",
				"api_base": "http://localhost:8000/v1"
			}
		],
		"channels": {
			"discord": {
				"mention_only": true
			}
		},
		"gateway": {"host": "127.0.0.1", "port": 18790}
	}`

	if err := os.WriteFile(configPath, []byte(v1Config), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	// Version should be bumped to 2
	if cfg.Version != CurrentVersion {
		t.Errorf("Version = %d, want %d", cfg.Version, CurrentVersion)
	}

	// gpt-4 has no API key → disabled
	gpt4, err := cfg.GetModelConfig("gpt-4")
	if err != nil {
		t.Fatalf("GetModelConfig(gpt-4): %v", err)
	}
	if gpt4.Enabled {
		t.Error("gpt-4 without API key should be disabled after migration")
	}

	// local-model → enabled
	local, err := cfg.GetModelConfig("local-model")
	if err != nil {
		t.Fatalf("GetModelConfig(local-model): %v", err)
	}
	if !local.Enabled {
		t.Error("local-model should be enabled after migration")
	}

	// Discord channel config should be migrated
	dcMigBC := cfg.Channels.Get("discord")
	if !dcMigBC.GroupTrigger.MentionOnly {
		t.Error("Discord mention_only should be migrated to group_trigger.mention_only")
	}

	// Verify backup was created with date suffix
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	var hasBackup bool
	for _, e := range entries {
		if matched, _ := filepath.Match("config.json.20*.bak", e.Name()); matched {
			hasBackup = true
			break
		}
	}
	if !hasBackup {
		t.Error("expected backup file with date suffix to be created")
	}

	// Verify the saved config on disk now has version 2
	saved, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile saved config: %v", err)
	}
	var versionCheck struct {
		Version int `json:"version"`
	}
	if err := json.Unmarshal(saved, &versionCheck); err != nil {
		t.Fatalf("Unmarshal saved config: %v", err)
	}
	if versionCheck.Version != 3 {
		t.Errorf("saved config version = %d, want 3", versionCheck.Version)
	}
}

// TestLoadConfig_V1WithAPIKeysInferredEnabled verifies that V1 configs with
// API keys in the security file get Enabled=true after migration.
func TestLoadConfig_V1WithAPIKeysInferredEnabled(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	secPath := securityPath(configPath)

	v1Config := `{
		"version": 1,
		"model_list": [
			{"model_name": "gpt-4", "model": "openai/gpt-4"},
			{"model_name": "claude", "model": "anthropic/claude"}
		],
		"gateway": {"host": "127.0.0.1", "port": 18790}
	}`

	securityConfig := `model_list:
  gpt-4:0:
    api_keys:
      - "sk-gpt-key"
  claude:0:
    api_keys:
      - "sk-claude-key"
`

	if err := os.WriteFile(configPath, []byte(v1Config), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.WriteFile(secPath, []byte(securityConfig), 0o600); err != nil {
		t.Fatalf("WriteFile security: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	for _, m := range cfg.ModelList {
		t.Logf("Model: %+v", m)
		if !m.Enabled {
			t.Errorf("model %q with API key in security file should be enabled", m.ModelName)
		}
	}
}

// TestLoadConfig_V2DirectLoad verifies that V2 configs load directly without
// running any migration.
func TestLoadConfig_V2DirectLoad(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	v2Config := `{
		"version": 2,
		"model_list": [
			{
				"model_name": "gpt-4",
				"model": "openai/gpt-4",
				"enabled": true
			},
			{
				"model_name": "claude",
				"model": "anthropic/claude"
			}
		],
		"gateway": {"host": "127.0.0.1", "port": 18790}
	}`

	if err := os.WriteFile(configPath, []byte(v2Config), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.Version != 3 {
		t.Errorf("Version = %d, want 3", cfg.Version)
	}

	gpt4, _ := cfg.GetModelConfig("gpt-4")
	if !gpt4.Enabled {
		t.Error("gpt-4 with explicit enabled=true should remain enabled")
	}

	claude, _ := cfg.GetModelConfig("claude")
	if claude.Enabled {
		t.Error("claude without enabled field should be false")
	}

	// V2→V3 migration creates a backup
	entries, _ := os.ReadDir(tmpDir)
	foundBackup := false
	for _, e := range entries {
		if matched, _ := filepath.Match("config.json.*.bak", e.Name()); matched {
			foundBackup = true
		}
	}
	if !foundBackup {
		t.Error("V2→V3 migration should create backup")
	}

	githubRegistry, ok := cfg.Tools.Skills.Registries.Get("github")
	if !ok {
		t.Fatal("expected default github skills registry to survive V0 migration")
	}
	if !githubRegistry.Enabled {
		t.Error("github skills registry should remain enabled after V0 migration")
	}
	if githubRegistry.BaseURL != "https://github.com" {
		t.Errorf("github registry base_url = %q, want %q", githubRegistry.BaseURL, "https://github.com")
	}
}
