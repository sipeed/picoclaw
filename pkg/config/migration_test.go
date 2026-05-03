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

// Tests for buildModelWithProtocol helper function.

func TestBuildModelWithProtocol_NoPrefix(t *testing.T) {
	result := buildModelWithProtocol("openai", "gpt-5.4")
	if result != "openai/gpt-5.4" {
		t.Errorf("buildModelWithProtocol(openai, gpt-5.4) = %q, want %q", result, "openai/gpt-5.4")
	}
}

func TestBuildModelWithProtocol_AlreadyHasPrefix(t *testing.T) {
	result := buildModelWithProtocol("openrouter", "openrouter/auto")
	if result != "openrouter/auto" {
		t.Errorf("buildModelWithProtocol(openrouter, openrouter/auto) = %q, want %q", result, "openrouter/auto")
	}
}

func TestBuildModelWithProtocol_DifferentPrefix(t *testing.T) {
	result := buildModelWithProtocol("anthropic", "openrouter/claude-sonnet-4.6")
	if result != "openrouter/claude-sonnet-4.6" {
		t.Errorf(
			"buildModelWithProtocol(anthropic, openrouter/claude-sonnet-4.6) = %q, want %q",
			result,
			"openrouter/claude-sonnet-4.6",
		)
	}
}

// ---------------------------------------------------------------------------
// V0/V1/V2 → V3 migration tests
// ---------------------------------------------------------------------------

// TestLoadConfig_V0MigrateProducesV2 verifies that V0→V3 migration produces
// correct Enabled fields and version.
func TestLoadConfig_V0MigrateProducesV2(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	v0Config := `{
		"model_list": [
			{
				"model_name": "gpt-4",
				"model": "openai/gpt-4",
				"api_key": "sk-test"
			},
			{
				"model_name": "claude",
				"model": "anthropic/claude"
			},
			{
				"model_name": "local-model",
				"model": "vllm/custom-model"
			}
		],
		"gateway": {"host": "127.0.0.1", "port": 18790}
	}`

	if err := os.WriteFile(configPath, []byte(v0Config), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.Version != CurrentVersion {
		t.Errorf("Version = %d, want %d", cfg.Version, CurrentVersion)
	}

	// Check enabled status
	modelEnabled := func(name string) bool {
		m, err := cfg.GetModelConfig(name)
		if err != nil {
			return false
		}
		return m.Enabled
	}

	if !modelEnabled("gpt-4") {
		t.Error("gpt-4 with API key from V0 should be enabled")
	}
	if modelEnabled("claude") {
		t.Error("claude without API key from V0 should be disabled")
	}
	if !modelEnabled("local-model") {
		t.Error("local-model from V0 should be enabled")
	}
}

// TestLoadConfig_UnsupportedVersion verifies that unsupported versions return an error.
func TestLoadConfig_UnsupportedVersion(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	badConfig := `{"version": 99, "gateway": {"host": "127.0.0.1", "port": 18790}}`
	if err := os.WriteFile(configPath, []byte(badConfig), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Fatal("LoadConfig should return error for unsupported version")
	}
	if !containsString(err.Error(), "unsupported config version") {
		t.Errorf("error = %q, want 'unsupported config version'", err.Error())
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestMigrateV0ToV3 verifies V0 (legacy, no version) → V3 migration.
// V0 configs use the old providers format without model_list.
func TestMigrateV0ToV3(t *testing.T) {
	// V0 config: no version field, uses legacy providers
	v0Config := `{
		"agents": {
			"defaults": {
				"provider": "openai",
				"model": "gpt-4"
			}
		},
		"providers": {
			"openai": {
				"api_key": "sk-test123",
				"api_base": "https://api.openai.com/v1"
			}
		},
		"channels": {
			"telegram": {
				"token": "bot-token"
			},
			"discord": {
				"mention_only": true
			}
		}
	}`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	require.NoError(t, os.WriteFile(configPath, []byte(v0Config), 0o600))
	m, err := loadConfigMap(configPath)
	require.NoError(t, err)

	err = migrateV0ToV1(m)
	require.NoError(t, err)
	err = migrateV1ToV2(m)
	require.NoError(t, err)
	err = migrateV2ToV3(m)
	require.NoError(t, err)

	// Version should be set to CurrentVersion
	require.Equal(t, CurrentVersion, m["version"])

	// Providers should be converted to model_list
	modelList, ok := m["model_list"].([]any)
	require.True(t, ok, "model_list should exist")
	require.NotEmpty(t, modelList, "model_list should not be empty")

	t.Logf("modelList: %+v", modelList)
	// First model should be the user's configured provider with user's model
	firstModel := modelList[0].(map[string]any)
	require.Equal(t, "openai", firstModel["model_name"])
	require.Equal(t, "openai/gpt-4", firstModel["model"])
	// api_key is converted to api_keys during migration
	require.Contains(t, firstModel, "api_keys", "api_keys should exist")

	// Channels should be converted to nested format with channel_list
	channelList, ok := m["channel_list"].(map[string]any)
	require.True(t, ok, "channel_list should exist")
	require.NotContains(t, m, "channels", "old 'channels' key should be removed")

	// telegram channel should have settings
	telegram := channelList["telegram"].(map[string]any)
	require.Equal(t, "telegram", telegram["type"])
	require.Contains(t, telegram, "settings", "telegram should have settings")
	settings := telegram["settings"].(map[string]any)
	require.Equal(t, "bot-token", settings["token"])

	// discord channel should have group_trigger and mention_only in group_trigger
	discord := channelList["discord"].(map[string]any)
	require.Equal(t, "discord", discord["type"])
	discordGroupTrigger := discord["group_trigger"].(map[string]any)
	require.Equal(t, true, discordGroupTrigger["mention_only"])
}

// TestMigrateV0ToV3_WithExistingModelList preserves existing model_list when present.
func TestMigrateV0ToV3_WithExistingModelList(t *testing.T) {
	v0Config := `{
		"model_list": [
			{"model_name": "custom", "model": "openai/custom-model", "api_key": "sk-existing"}
		],
		"channels": {
			"telegram": {"token": "bot123"}
		}
	}`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	require.NoError(t, os.WriteFile(configPath, []byte(v0Config), 0o600))
	m, err := loadConfigMap(configPath)
	require.NoError(t, err)

	err = migrateV0ToV1(m)
	require.NoError(t, err)
	err = migrateV1ToV2(m)
	require.NoError(t, err)
	err = migrateV2ToV3(m)
	require.NoError(t, err)

	// Existing model_list should be preserved (not overridden by providers)
	modelList := m["model_list"].([]any)
	require.Len(t, modelList, 1)
	firstModel := modelList[0].(map[string]any)
	require.Equal(t, "custom", firstModel["model_name"])
}

// TestMigrateV1ToV3 verifies V1 → V3 migration.
// V1 uses flat channel format without "settings" wrapper.
func TestMigrateV1ToV3(t *testing.T) {
	v1Config := `{
		"version": 1,
		"model_list": [
			{"model_name": "gpt-4", "model": "openai/gpt-4", "api_key": "sk-test"}
		],
		"channels": {
			"telegram": {
				"token": "bot-token",
				"base_url": "https://custom.api.com"
			},
			"discord": {
				"mention_only": true,
				"proxy": "socks5://localhost:1080"
			},
			"onebot": {
				"ws_url": "ws://localhost:3001",
				"group_trigger_prefix": ["/"]
			}
		}
	}`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	require.NoError(t, os.WriteFile(configPath, []byte(v1Config), 0o600))
	m, err := loadConfigMap(configPath)
	require.NoError(t, err)

	err = migrateV1ToV2(m)
	require.NoError(t, err)
	err = migrateV2ToV3(m)
	require.NoError(t, err)

	// Version should be set to CurrentVersion
	require.Equal(t, CurrentVersion, m["version"])

	// Channels should be converted to nested format
	channelList, ok := m["channel_list"].(map[string]any)
	require.True(t, ok, "channel_list should exist")
	require.NotContains(t, m, "channels", "old 'channels' key should be removed")

	// telegram: flat fields moved to settings
	telegram := channelList["telegram"].(map[string]any)
	require.Equal(t, "telegram", telegram["type"])
	tgSettings := telegram["settings"].(map[string]any)
	require.Equal(t, "bot-token", tgSettings["token"])
	require.Equal(t, "https://custom.api.com", tgSettings["base_url"])

	// discord: mention_only should be moved to group_trigger
	discord := channelList["discord"].(map[string]any)
	require.Equal(t, "discord", discord["type"])
	require.Contains(t, discord, "group_trigger", "mention_only should be migrated to group_trigger")
	gt := discord["group_trigger"].(map[string]any)
	require.Equal(t, true, gt["mention_only"])
	discordSettings := discord["settings"].(map[string]any)
	require.Equal(t, "socks5://localhost:1080", discordSettings["proxy"])

	// onebot: group_trigger_prefix should be moved to group_trigger.prefixes
	onebot := channelList["onebot"].(map[string]any)
	require.Equal(t, "onebot", onebot["type"])
	obGroupTrigger := onebot["group_trigger"].(map[string]any)
	require.Equal(
		t,
		[]any{"/"},
		obGroupTrigger["prefixes"],
		"group_trigger_prefix should be moved to group_trigger.prefixes",
	)
	obSettings := onebot["settings"].(map[string]any)
	require.Equal(t, "ws://localhost:3001", obSettings["ws_url"])
}

// TestMigrateV1ToV3_ApiKeyConversion verifies api_key → api_keys conversion.
func TestMigrateV1ToV3_ApiKeyConversion(t *testing.T) {
	v1Config := `{
		"version": 1,
		"model_list": [
			{"model_name": "gpt-4", "model": "openai/gpt-4", "api_key": "sk-single"},
			{"model_name": "no-key", "model": "openai/no-key"}
		],
		"channels": {
			"telegram": {"token": "bot"}
		}
	}`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	require.NoError(t, os.WriteFile(configPath, []byte(v1Config), 0o600))
	m, err := loadConfigMap(configPath)
	require.NoError(t, err)

	err = migrateV1ToV2(m)
	require.NoError(t, err)
	err = migrateV2ToV3(m)
	require.NoError(t, err)

	// api_key should be converted to api_keys array
	modelList := m["model_list"].([]any)
	firstModel := modelList[0].(map[string]any)
	require.NotContains(t, firstModel, "api_key", "api_key should be removed")
	require.Contains(t, firstModel, "api_keys", "api_keys should exist")
	// api_keys can be []string or []any depending on how it was set
	if apiKeys, ok := firstModel["api_keys"].([]string); ok {
		require.Len(t, apiKeys, 1)
		require.Equal(t, "sk-single", apiKeys[0])
	} else if apiKeys, ok := firstModel["api_keys"].([]any); ok {
		require.Len(t, apiKeys, 1)
		require.Equal(t, "sk-single", apiKeys[0])
	} else {
		t.Fatalf("api_keys has unexpected type: %T", firstModel["api_keys"])
	}

	// Model without api_key should not have api_keys added
	secondModel := modelList[1].(map[string]any)
	require.NotContains(t, secondModel, "api_key")
	require.NotContains(t, secondModel, "api_keys")
}

// TestMigrateV1ToV3_AlreadyNestedFormat leaves already-nested channels unchanged.
func TestMigrateV1ToV3_AlreadyNestedFormat(t *testing.T) {
	v1Config := `{
		"version": 1,
		"model_list": [
			{"model_name": "gpt-4", "model": "openai/gpt-4"}
		],
		"channels": {
			"telegram": {
				"type": "telegram",
				"settings": {
					"token": "bot-token"
				}
			}
		}
	}`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	require.NoError(t, os.WriteFile(configPath, []byte(v1Config), 0o600))
	m, err := loadConfigMap(configPath)
	require.NoError(t, err)

	err = migrateV1ToV2(m)
	require.NoError(t, err)
	err = migrateV2ToV3(m)
	require.NoError(t, err)

	channelList := m["channel_list"].(map[string]any)
	telegram := channelList["telegram"].(map[string]any)
	// Should not be double-wrapped
	require.Equal(t, "telegram", telegram["type"])
	settings := telegram["settings"].(map[string]any)
	require.Equal(t, "bot-token", settings["token"])
	// Should NOT have nested settings inside settings
	require.NotContains(t, settings, "settings")
}

func TestSanitizeDeprecatedFields_MigratesDMScope(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantDims []string
	}{
		{"per-channel-peer", `{"session":{"dm_scope":"per-channel-peer"}}`, []string{"chat", "sender"}},
		{"per-channel", `{"session":{"dm_scope":"per-channel"}}`, []string{"chat"}},
		{"per-peer", `{"session":{"dm_scope":"per-peer"}}`, []string{"sender"}},
		{"global", `{"session":{"dm_scope":"global"}}`, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := sanitizeDeprecatedFields([]byte(tt.input))
			require.NoError(t, err)

			var m map[string]any
			require.NoError(t, json.Unmarshal(out, &m))

			session := m["session"].(map[string]any)
			require.NotContains(t, session, "dm_scope", "dm_scope should be removed")

			dims, ok := session["dimensions"].([]any)
			require.True(t, ok, "dimensions should be a slice")
			got := make([]string, len(dims))
			for i, d := range dims {
				got[i] = d.(string)
			}
			require.Equal(t, tt.wantDims, got)
		})
	}
}

func TestSanitizeDeprecatedFields_PreservesExistingDimensions(t *testing.T) {
	input := `{"session":{"dm_scope":"global","dimensions":["chat","sender"]}}`
	out, err := sanitizeDeprecatedFields([]byte(input))
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(out, &m))

	session := m["session"].(map[string]any)
	require.NotContains(t, session, "dm_scope")
	dims := session["dimensions"].([]any)
	require.Equal(t, 2, len(dims), "existing dimensions should be preserved")
}

func TestSanitizeDeprecatedFields_NoSessionUnchanged(t *testing.T) {
	input := `{"version":3,"gateway":{"host":"localhost"}}`
	out, err := sanitizeDeprecatedFields([]byte(input))
	require.NoError(t, err)
	require.JSONEq(t, input, string(out))
}

func TestSanitizeDeprecatedFields_MigratesChannelsToChannelList(t *testing.T) {
	input := `{"version":3,"channels":{"telegram":{"type":"telegram","enabled":true}}}`
	out, err := sanitizeDeprecatedFields([]byte(input))
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(out, &m))
	require.NotContains(t, m, "channels", "channels should be removed")
	require.Contains(t, m, "channel_list", "channel_list should be present")
}

func TestSanitizeDeprecatedFields_ChannelsDoesNotOverwriteExistingChannelList(t *testing.T) {
	input := `{"version":3,"channels":{"old":{"type":"old"}},"channel_list":{"telegram":{"type":"telegram"}}}`
	out, err := sanitizeDeprecatedFields([]byte(input))
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(out, &m))
	require.NotContains(t, m, "channels")
	cl := m["channel_list"].(map[string]any)
	require.Contains(t, cl, "telegram", "existing channel_list should be preserved")
	require.NotContains(t, cl, "old", "old channels should not overwrite channel_list")
}

func TestSanitizeDeprecatedFields_RemovesBindings(t *testing.T) {
	input := `{"version":3,"bindings":[{"agent":"main"}]}`
	out, err := sanitizeDeprecatedFields([]byte(input))
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(out, &m))
	require.NotContains(t, m, "bindings")
}

func TestSanitizeDeprecatedFields_RemovesProviders(t *testing.T) {
	input := `{"version":3,"providers":{"openai":{"api_key":"sk-test"}}}`
	out, err := sanitizeDeprecatedFields([]byte(input))
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(out, &m))
	require.NotContains(t, m, "providers")
}

func TestLoadConfig_WithLegacyDMScope(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	raw := `{
		"version": 3,
		"session": {
			"dm_scope": "per-channel-peer"
		},
		"model_list": []
	}`
	require.NoError(t, os.WriteFile(configPath, []byte(raw), 0o600))

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err, "LoadConfig should not fail with legacy dm_scope")
	require.Equal(t, []string{"chat", "sender"}, cfg.Session.Dimensions)
}

func TestLoadConfig_WithLegacyChannelsField(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	raw := `{
		"version": 3,
		"channels": {
			"telegram": {
				"type": "telegram",
				"enabled": true,
				"settings": {"token": "test-token"}
			}
		},
		"model_list": []
	}`
	require.NoError(t, os.WriteFile(configPath, []byte(raw), 0o600))

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err, "LoadConfig should not fail with legacy channels field")
	require.NotNil(t, cfg)
}
