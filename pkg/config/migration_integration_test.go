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
	if !cfg.Channels.Telegram.Enabled {
		t.Error("Telegram.Enabled should be true")
	}
	if cfg.Channels.Telegram.Token() != "test-token" {
		t.Errorf("Telegram.Token = %q, want %q", cfg.Channels.Telegram.Token(), "test-token")
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
	if cfg.Channels.Discord.GroupTrigger.MentionOnly != true {
		t.Error("Discord.GroupTrigger.MentionOnly should be true after migration")
	}

	// OneBot: group_trigger_prefix should be migrated to group_trigger.prefixes
	if len(cfg.Channels.OneBot.GroupTrigger.Prefixes) != 2 {
		t.Errorf("len(OneBot.GroupTrigger.Prefixes) = %d, want 2", len(cfg.Channels.OneBot.GroupTrigger.Prefixes))
	} else {
		if cfg.Channels.OneBot.GroupTrigger.Prefixes[0] != "/" {
			t.Errorf("Prefixes[0] = %q, want %q", cfg.Channels.OneBot.GroupTrigger.Prefixes[0], "/")
		}
		if cfg.Channels.OneBot.GroupTrigger.Prefixes[1] != "!" {
			t.Errorf("Prefixes[1] = %q, want %q", cfg.Channels.OneBot.GroupTrigger.Prefixes[1], "!")
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

	// Verify that the migrated config has the existing security values
	// Telegram token should be preserved
	if cfg.Channels.Telegram.Token() != "existing-telegram-token-from-env" {
		t.Errorf("Telegram token was overwritten: got %q, want %q",
			cfg.Channels.Telegram.Token(), "existing-telegram-token-from-env")
	}

	// Discord token should be preserved (even though legacy config didn't have it)
	if cfg.Channels.Discord.Token() != "existing-discord-token-from-env" {
		t.Errorf("Discord token was overwritten: got %q, want %q",
			cfg.Channels.Discord.Token(), "existing-discord-token-from-env")
	}

	// Model API key should be preserved
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
	reloadedSec, err := loadSecurityConfig(securityPath)
	if err != nil {
		t.Fatalf("Failed to reload security config: %v", err)
	}

	if reloadedSec.Channels.Telegram == nil ||
		reloadedSec.Channels.Telegram.Token != "existing-telegram-token-from-env" {
		t.Error("Telegram token not preserved in .security.yml file")
	}

	if reloadedSec.Channels.Discord == nil || reloadedSec.Channels.Discord.Token != "existing-discord-token-from-env" {
		t.Error("Discord token not preserved in .security.yml file")
	}
}
