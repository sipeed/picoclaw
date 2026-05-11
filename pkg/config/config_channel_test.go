package config

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/sipeed/picoclaw/pkg/credential"
)

// ─── Test extend structs (simplified, settings + secure in one struct) ───

type testTelegramConfig struct {
	BaseURL                string          `json:"base_url"            yaml:"-"`
	Proxy                  string          `json:"proxy"               yaml:"-"`
	BusinessMode           bool            `json:"business_mode"       yaml:"-"`
	BusinessOwner          string          `json:"business_owner"      yaml:"-"`
	BusinessCommandsEnable bool            `json:"business_commands_enable" yaml:"-"`
	UseMarkdownV2          bool            `json:"use_markdown_v2"     yaml:"-"`
	Streaming              StreamingConfig `json:"streaming,omitempty" yaml:"-"`
	Token                  SecureString    `json:"token,omitzero"      yaml:"token,omitempty"`
}

type testDiscordConfig struct {
	MentionOnly bool          `json:"mention_only"      yaml:"-"`
	Token       SecureString  `json:"token,omitzero"    yaml:"token,omitempty"`
	ApiKeys     SecureStrings `json:"api_keys,omitzero" yaml:"api_keys,omitempty"`
}

// ═══════════════════════════════════════════════════
//  RawNode JSON/YAML round-trip
// ═══════════════════════════════════════════════════

func TestRawNode_JSON_RoundTrip(t *testing.T) {
	t.Run("unmarshal and decode", func(t *testing.T) {
		var r RawNode
		require.NoError(t, json.Unmarshal([]byte(`{"key":"value","num":42}`), &r))
		assert.False(t, r.IsEmpty())

		var m map[string]any
		require.NoError(t, r.Decode(&m))
		assert.Equal(t, "value", m["key"])
		assert.Equal(t, float64(42), m["num"])
	})

	t.Run("marshal round-trip", func(t *testing.T) {
		r := RawNode(`{"a":1}`)
		data, err := json.Marshal(r)
		require.NoError(t, err)
		assert.JSONEq(t, `{"a":1}`, string(data))
	})

	t.Run("null input", func(t *testing.T) {
		var r RawNode
		require.NoError(t, json.Unmarshal([]byte("null"), &r))
		assert.True(t, r.IsEmpty())

		data, err := json.Marshal(r)
		require.NoError(t, err)
		assert.Equal(t, "null", string(data))
	})

	t.Run("empty node decode", func(t *testing.T) {
		var r RawNode
		var m map[string]any
		require.NoError(t, r.Decode(&m))
		assert.Nil(t, m)
	})
}

func TestRawNode_YAML_RoundTrip(t *testing.T) {
	t.Run("unmarshal and decode", func(t *testing.T) {
		var r RawNode
		require.NoError(t, yaml.Unmarshal([]byte("key: value\nnum: 42"), &r))
		assert.False(t, r.IsEmpty())

		var m map[string]any
		require.NoError(t, r.Decode(&m))
		assert.Equal(t, "value", m["key"])
	})

	t.Run("marshal round-trip", func(t *testing.T) {
		r := RawNode(`{"name":"test"}`)
		data, err := yaml.Marshal(r)
		require.NoError(t, err)
		assert.Contains(t, string(data), "name: test")
	})

	t.Run("empty node marshal", func(t *testing.T) {
		var r RawNode
		v, err := yaml.Marshal(r)
		require.NoError(t, err)
		assert.Equal(t, "null\n", string(v))
	})
}

// ═══════════════════════════════════════════════════
//  JSON unmarshal: extend.json
// ═══════════════════════════════════════════════════

func TestChannel_JSON_Unmarshal(t *testing.T) {
	jsonData := `{
		"enabled": true,
		"type": "telegram",
		"allow_from": ["user1", "user2"],
		"reasoning_channel_id": "-100xxx",
		"settings": {
			"base_url": "https://custom-api.example.com",
			"business_mode": true,
			"business_owner": "42",
			"business_commands_enable": true,
			"use_markdown_v2": true,
			"streaming": {"enabled": true, "throttle_seconds": 2},
			"token": "[NOT_HERE]"
		}
	}`

	var ch Channel
	require.NoError(t, json.Unmarshal([]byte(jsonData), &ch))

	assert.True(t, ch.Enabled)
	assert.Equal(t, "telegram", ch.Type)
	assert.Equal(t, FlexibleStringSlice{"user1", "user2"}, ch.AllowFrom)
	assert.Equal(t, "-100xxx", ch.ReasoningChannelID)
	assert.False(t, ch.SettingsIsEmpty())

	// Decode into combined struct
	var cfg testTelegramConfig
	require.NoError(t, ch.Decode(&cfg))
	assert.Equal(t, "https://custom-api.example.com", cfg.BaseURL)
	assert.True(t, cfg.BusinessMode)
	assert.Equal(t, "42", cfg.BusinessOwner)
	assert.True(t, cfg.BusinessCommandsEnable)
	assert.True(t, cfg.UseMarkdownV2)
	assert.True(t, cfg.Streaming.Enabled)
	assert.Equal(t, 2, cfg.Streaming.ThrottleSeconds)
	// SecureString.UnmarshalJSON("[NOT_HERE]") → no-op → empty
	assert.Equal(t, "", cfg.Token.String())
}

// ═══════════════════════════════════════════════════
//  JSON marshal: secure fields masked as [NOT_HERE]
// ═══════════════════════════════════════════════════

func TestChannel_JSON_Marshal_SecureMasked(t *testing.T) {
	ch := Channel{
		Enabled: true,
		Type:    ChannelTelegram,
		name:    "my_telegram",
		Settings: mustParseRawNode(
			`{"base_url": "https://api.telegram.org", "proxy": "socks5://127.0.0.1:1080", "token": "123456:SECRET"}`,
		),
	}
	// Decode to register secure field names
	var cfg testTelegramConfig
	require.NoError(t, ch.Decode(&cfg))

	data, err := json.MarshalIndent(ch, "", "  ")
	require.NoError(t, err)
	t.Logf("JSON output:\n%s", string(data))

	assert.NotContains(t, string(data), "token")
	assert.NotContains(t, string(data), "123456:SECRET")
	assert.NotContains(t, string(data), "SECRET")
	assert.Contains(t, string(data), "base_url")
	assert.Contains(t, string(data), "proxy")
}

// ═══════════════════════════════════════════════════
//  YAML unmarshal: security.yml — only secure data
// ═══════════════════════════════════════════════════

func TestChannel_YAML_Unmarshal(t *testing.T) {
	yamlData := `
settings:
  token: "789012:XYZ-TOKEN"
`

	var ch Channel
	require.NoError(t, yaml.Unmarshal([]byte(yamlData), &ch))
	assert.False(t, ch.SettingsIsEmpty())

	var cfg testTelegramConfig
	require.NoError(t, ch.Decode(&cfg))
	assert.Equal(t, "789012:XYZ-TOKEN", cfg.Token.String())
	assert.Equal(t, "", cfg.BaseURL)
}

// ═══════════════════════════════════════════════════
//  YAML marshal: only secure fields
// ═══════════════════════════════════════════════════

func TestChannel_YAML_Marshal_OnlySecureFields(t *testing.T) {
	ch := Channel{
		Enabled:  true,
		Type:     ChannelTelegram,
		name:     "my_telegram",
		Settings: mustParseRawNode(`{"base_url": "https://api.telegram.org", "token": "123456:SECRET"}`),
	}
	var cfg testTelegramConfig
	require.NoError(t, ch.Decode(&cfg))

	data, err := yaml.Marshal(ch)
	require.NoError(t, err)
	t.Logf("YAML output:\n%s", string(data))

	assert.NotContains(t, string(data), "NOT_HERE")
	assert.Contains(t, string(data), "token")
	assert.Contains(t, string(data), "123456:SECRET")
	// Non-secure fields must NOT appear in YAML output
	assert.NotContains(t, string(data), "base_url")
	assert.NotContains(t, string(data), "proxy")
}

// ═══════════════════════════════════════════════════
//  extractSecureFieldNames
// ═══════════════════════════════════════════════════

func TestExtractSecureFieldNames(t *testing.T) {
	t.Run("telegram extend", func(t *testing.T) {
		names := extractSecureFieldNames(&testTelegramConfig{})
		assert.Equal(t, map[string]struct{}{"token": {}}, names)
	})

	t.Run("discord extend", func(t *testing.T) {
		names := extractSecureFieldNames(&testDiscordConfig{})
		assert.Equal(t, map[string]struct{}{"token": {}, "api_keys": {}}, names)
	})

	t.Run("non-struct target", func(t *testing.T) {
		names := extractSecureFieldNames("not a struct")
		assert.Nil(t, names)
	})

	t.Run("struct without secure fields", func(t *testing.T) {
		type NoSecure struct {
			Name  string `json:"name"`
			Count int    `json:"count"`
		}
		names := extractSecureFieldNames(&NoSecure{})
		assert.Empty(t, names)
	})
}

// ═══════════════════════════════════════════════════
//  mergeRawJSON
// ═══════════════════════════════════════════════════

func TestMergeRawJSON(t *testing.T) {
	t.Run("overlay overrides base", func(t *testing.T) {
		base := RawNode(`{"base_url": "old", "token": "[NOT_HERE]"}`)
		overlay := RawNode(`{"token": "REAL_TOKEN"}`)
		merged, err := mergeRawJSON(base, overlay)
		require.NoError(t, err)

		var m map[string]any
		json.Unmarshal(merged, &m)
		assert.Equal(t, "old", m["base_url"])
		assert.Equal(t, "REAL_TOKEN", m["token"])
	})

	t.Run("empty overlay", func(t *testing.T) {
		base := RawNode(`{"base_url": "https://api.telegram.org"}`)
		merged, err := mergeRawJSON(base, nil)
		require.NoError(t, err)
		// mergeRawJSON normalizes JSON through unmarshal→marshal, so compare parsed values
		var orig, result map[string]any
		json.Unmarshal(base, &orig)
		json.Unmarshal(merged, &result)
		assert.Equal(t, orig, result)
	})

	t.Run("empty base", func(t *testing.T) {
		overlay := RawNode(`{"token": "NEW"}`)
		merged, err := mergeRawJSON(nil, overlay)
		require.NoError(t, err)
		assert.Contains(t, string(merged), `"token":"NEW"`)
	})
}

// ═══════════════════════════════════════════════════
//  Full flow: extend.json + security.yml merge
// ═══════════════════════════════════════════════════

func TestChannel_FullFlow_JSON_YAML_Merge(t *testing.T) {
	// Step 1: Load from extend.json
	jsonData := `{
		"enabled": true,
		"type": "telegram",
		"allow_from": ["admin"],
		"settings": {
			"base_url": "https://custom-api.example.com",
			"use_markdown_v2": true,
			"streaming": {"enabled": true},
			"token": "[NOT_HERE]"
		}
	}`

	var ch Channel
	require.NoError(t, json.Unmarshal([]byte(jsonData), &ch))
	assert.True(t, ch.Enabled)

	// Step 2: Load secure from security.yml
	yamlData := `
settings:
  token: "123456:REAL-TOKEN"
`
	//var yamlOverlay struct {
	//	Settings RawNode `yaml:"settings"`
	//}
	require.NoError(t, yaml.Unmarshal([]byte(yamlData), &ch))

	// Step 3: Merge
	// require.NoError(t, ch.MergeSecure(yamlOverlay.Settings))

	// Step 4: Decode merged result
	var cfg testTelegramConfig
	require.NoError(t, ch.Decode(&cfg))
	assert.Equal(t, "https://custom-api.example.com", cfg.BaseURL)
	assert.True(t, cfg.UseMarkdownV2)
	assert.Equal(t, "123456:REAL-TOKEN", cfg.Token.String())

	// Step 5: Save extend.json → token masked as [NOT_HERE]
	outJSON, err := json.MarshalIndent(ch, "", "  ")
	require.NoError(t, err)
	t.Logf("Saved extend.json:\n%s", string(outJSON))
	assert.NotContains(t, string(outJSON), "token")
	assert.NotContains(t, string(outJSON), "REAL-TOKEN")
	assert.Contains(t, string(outJSON), "base_url")

	// Step 6: Save security.yml → only token
	outYAML, err := yaml.Marshal(ch)
	require.NoError(t, err)
	t.Logf("Saved security.yml:\n%s", string(outYAML))
	assert.Contains(t, string(outYAML), "123456:REAL-TOKEN")
	assert.NotContains(t, string(outYAML), "NOT_HERE")
	assert.NotContains(t, string(outYAML), "base_url")
}

// ═══════════════════════════════════════════════════
//  Multiple channels in a list
// ═══════════════════════════════════════════════════

func TestChannel_MultipleChannels(t *testing.T) {
	type ChannelsWrapper struct {
		Channels ChannelsConfig `json:"channels" yaml:"channels"`
	}

	jsonData := `{
		"channels": {
			"tg1": {
				"enabled": true,
				"type": "telegram",
				"settings": {"base_url": "https://api.telegram.org", "token": "[NOT_HERE]"}
			},
			"tg2": {
				"enabled": true,
				"type": "telegram",
				"settings": {"base_url": "https://custom-api.example.com", "proxy": "socks5://proxy:1080", "token": "[NOT_HERE]"}
			},
			"discord1": {
				"enabled": true,
				"type": "discord",
				"settings": {"mention_only": true, "token": "[NOT_HERE]"}
			}
		}
	}`

	var wrapper ChannelsWrapper
	require.NoError(t, json.Unmarshal([]byte(jsonData), &wrapper))
	require.Len(t, wrapper.Channels, 3)

	// Decode each channel to register secure field names
	for name, ch := range wrapper.Channels {
		ch.SetName(name) // Set channel name
		switch ch.Type {
		case "telegram":
			var tc testTelegramConfig
			require.NoError(t, ch.Decode(&tc))
		case "discord":
			var dc testDiscordConfig
			require.NoError(t, ch.Decode(&dc))
		default:
			t.Logf("Unknown channel type: %s for channel %s", ch.Type, name)
		}
	}

	// Load secrets from YAML
	yamlData := `
channels:
  tg1:
    settings:
      token: "TOKEN_1"
  tg2:
    settings:
      token: "TOKEN_2"
  discord1:
    settings:
      token: "DISCORD_TOKEN"
`
	require.NoError(t, yaml.Unmarshal([]byte(yamlData), &wrapper))

	// Verify first telegram
	var tg1 testTelegramConfig
	require.NoError(t, wrapper.Channels["tg1"].Decode(&tg1))
	assert.Equal(t, "https://api.telegram.org", tg1.BaseURL)
	assert.Equal(t, "TOKEN_1", tg1.Token.String())

	// Verify second telegram
	var tg2 testTelegramConfig
	require.NoError(t, wrapper.Channels["tg2"].Decode(&tg2))
	assert.Equal(t, "https://custom-api.example.com", tg2.BaseURL)
	assert.Equal(t, "socks5://proxy:1080", tg2.Proxy)
	assert.Equal(t, "TOKEN_2", tg2.Token.String())

	// Verify discord
	var disc testDiscordConfig
	require.NoError(t, wrapper.Channels["discord1"].Decode(&disc))
	assert.True(t, disc.MentionOnly)
	assert.Equal(t, "DISCORD_TOKEN", disc.Token.String())

	// Save JSON → all tokens removed
	outJSON, err := json.MarshalIndent(wrapper, "", "  ")
	require.NoError(t, err)
	t.Logf("Saved extend.json:\n%s", string(outJSON))
	assert.NotContains(t, string(outJSON), "token")
	assert.NotContains(t, string(outJSON), "TOKEN_1")
	assert.NotContains(t, string(outJSON), "DISCORD_TOKEN")

	// Save YAML → only tokens
	outYAML, err := yaml.Marshal(wrapper)
	require.NoError(t, err)
	t.Logf("Saved security.yml:\n%s", string(outYAML))
	assert.Contains(t, string(outYAML), "TOKEN_1")
	assert.Contains(t, string(outYAML), "DISCORD_TOKEN")
	assert.NotContains(t, string(outYAML), "base_url")
	assert.NotContains(t, string(outYAML), "NOT_HERE")
}

// ═══════════════════════════════════════════════════
//  Empty/missing settings
// ═══════════════════════════════════════════════════

func TestChannel_EmptySettings(t *testing.T) {
	// Flat format with only common fields: enabled and type are extracted to Channel,
	// Settings should be empty (no channel-specific fields)
	jsonData := `{
		"enabled": true,
		"type": "telegram"
	}`

	var ch Channel
	require.NoError(t, json.Unmarshal([]byte(jsonData), &ch))
	// All fields are common fields — Settings should be empty
	assert.True(t, ch.SettingsIsEmpty())

	// Decode into typed config — common fields like enabled/type are extracted,
	// channel-specific fields should be empty
	var cfg testTelegramConfig
	require.NoError(t, ch.Decode(&cfg))
	assert.Equal(t, "", cfg.BaseURL)
	assert.Equal(t, "", cfg.Token.String())
}

func TestChannel_NestedEmptySettings(t *testing.T) {
	// Nested format with empty settings
	jsonData := `{
		"enabled": true,
		"type": "telegram",
		"settings": {}
	}`

	var ch Channel
	require.NoError(t, json.Unmarshal([]byte(jsonData), &ch))
	assert.True(t, ch.SettingsIsEmpty())

	var cfg testTelegramConfig
	require.NoError(t, ch.Decode(&cfg))
	assert.Equal(t, "", cfg.BaseURL)
	assert.Equal(t, "", cfg.Token.String())
}

// ═══════════════════════════════════════════════════
//  YAML merge with fewer channels than JSON
// ═══════════════════════════════════════════════════

func TestChannel_MultipleChannels_PartialYAMLMerge(t *testing.T) {
	type ChannelsWrapper struct {
		Channels ChannelsConfig `json:"channels" yaml:"channels"`
	}

	// JSON has 3 channels
	jsonData := `{
		"channels": {
			"tg1": {"enabled": true, "type": "telegram", "settings": {"base_url": "https://api.telegram.org", "token": "[NOT_HERE]"}},
			"tg2": {"enabled": true, "type": "telegram", "settings": {"base_url": "https://custom-api.example.com", "token": "[NOT_HERE]"}},
			"discord1": {"enabled": true, "type": "discord", "settings": {"mention_only": true, "token": "[NOT_HERE]"}}
		}
	}`
	var wrapper ChannelsWrapper
	require.NoError(t, json.Unmarshal([]byte(jsonData), &wrapper))
	require.Len(t, wrapper.Channels, 3)
	t.Logf("wrapper: %v", wrapper)

	// YAML has only 2 secrets (missing tg2)
	yamlData := `
channels:
  tg1:
    settings:
      token: "TOKEN_1"
  discord1:
    settings:
      token: "DISCORD_TOKEN"
`
	//var yamlWrapper struct {
	//	Channels map[string]struct {
	//		Settings RawNode `yaml:"settings"`
	//	} `yaml:"channels"`
	//}
	assert.True(t, wrapper.Channels["tg1"].Enabled)
	assert.Equal(t, "telegram", wrapper.Channels["tg1"].Type)

	require.NoError(t, yaml.Unmarshal([]byte(yamlData), &wrapper))
	t.Logf("yamlWrapper: %v", wrapper)
	require.Len(t, wrapper.Channels, 3)

	assert.True(t, wrapper.Channels["tg1"].Enabled)

	t.Logf("wrapper: %v", string(wrapper.Channels["tg1"].Settings))
	//// Merge by name; missing keys are simply absent from the YAML map (no-op)
	//for name, ch := range wrapper.Channels {
	//	if overlay, ok := yamlWrapper.Channels[name]; ok {
	//		require.NoError(t, ch.MergeSecure(overlay.Settings))
	//	}
	//}

	// tg1: merged from YAML
	var tg1 TelegramSettings
	require.NoError(t, wrapper.Channels["tg1"].Decode(&tg1))
	assert.Equal(t, "TOKEN_1", tg1.Token.String())

	// tg2: no YAML entry → MergeSecure not called → token stays [NOT_HERE] → empty
	var tg2 TelegramSettings
	require.NoError(t, wrapper.Channels["tg2"].Decode(&tg2))
	assert.Equal(t, "", tg2.Token.String())
	assert.Equal(t, "https://custom-api.example.com", tg2.BaseURL)

	// discord1: merged from YAML
	var disc DiscordSettings
	require.NoError(t, wrapper.Channels["discord1"].Decode(&disc))
	assert.Equal(t, "DISCORD_TOKEN", disc.Token.String())
	assert.True(t, disc.MentionOnly)
}

// ═══════════════════════════════════════════════════
//  YAML list: channels with secure data
// ═══════════════════════════════════════════════════

func TestChannel_YAML_ListWithSecure(t *testing.T) {
	yamlData := `
channels:
  tg_bot:
    enabled: true
    type: telegram
    settings:
      token: "TG_TOKEN_FROM_YAML"
  discord_bot:
    enabled: true
    type: discord
    settings:
      token: "DISCORD_TOKEN_FROM_YAML"
`

	type ChannelsWrapper struct {
		Channels map[string]*Channel `yaml:"channels"`
	}

	var wrapper ChannelsWrapper
	require.NoError(t, yaml.Unmarshal([]byte(yamlData), &wrapper))
	require.Len(t, wrapper.Channels, 2)

	var tg testTelegramConfig
	require.NoError(t, wrapper.Channels["tg_bot"].Decode(&tg))
	assert.Equal(t, "TG_TOKEN_FROM_YAML", tg.Token.String())

	var disc testDiscordConfig
	require.NoError(t, wrapper.Channels["discord_bot"].Decode(&disc))
	assert.Equal(t, "DISCORD_TOKEN_FROM_YAML", disc.Token.String())
}

// ═══════════════════════════════════════════════════
//  removeSecureFields / filterSecureFields unit tests
// ═══════════════════════════════════════════════════

func TestRemoveSecureFields(t *testing.T) {
	t.Run("removes known secure fields", func(t *testing.T) {
		r := RawNode(`{"base_url": "https://api.telegram.org", "token": "SECRET"}`)
		names := map[string]struct{}{"token": {}}
		cleaned := removeSecureFields(r, names)

		var m map[string]any
		json.Unmarshal(cleaned, &m)
		assert.Equal(t, "https://api.telegram.org", m["base_url"])
		assert.NotContains(t, m, "token")
	})

	t.Run("nil secureFields returns as-is", func(t *testing.T) {
		r := RawNode(`{"token": "SECRET"}`)
		cleaned := removeSecureFields(r, nil)
		assert.Equal(t, string(r), string(cleaned))
	})

	t.Run("empty raw returns as-is", func(t *testing.T) {
		cleaned := removeSecureFields(nil, map[string]struct{}{"token": {}})
		assert.Nil(t, cleaned)
	})
}

func TestFilterSecureFields(t *testing.T) {
	t.Run("keeps only secure fields", func(t *testing.T) {
		r := RawNode(`{"base_url": "https://api.telegram.org", "token": "SECRET"}`)
		names := map[string]struct{}{"token": {}}
		filtered := filterSecureFields(r, names)

		var m map[string]any
		json.Unmarshal(filtered, &m)
		assert.NotContains(t, m, "base_url")
		assert.Equal(t, "SECRET", m["token"])
	})

	t.Run("nil secureFields returns nil", func(t *testing.T) {
		r := RawNode(`{"token": "SECRET"}`)
		filtered := filterSecureFields(r, nil)
		assert.Nil(t, filtered)
	})

	t.Run("empty raw returns nil", func(t *testing.T) {
		filtered := filterSecureFields(nil, map[string]struct{}{"token": {}})
		assert.Nil(t, filtered)
	})
}

// ═══════════════════════════════════════════════════
//  SecureStrings (ApiKeys) full flow
// ═══════════════════════════════════════════════════

func TestChannel_SecureStrings_ApiKeys(t *testing.T) {
	// Step 1: Load from extend.json
	jsonData := `{
		"enabled": true,
		"type": "discord",
		"settings": {
			"mention_only": true,
			"token": "[NOT_HERE]",
			"api_keys": ["[NOT_HERE]"]
		}
	}`
	var ch Channel
	require.NoError(t, json.Unmarshal([]byte(jsonData), &ch))

	// Step 2: Merge secure from security.yml
	yamlData := `
settings:
  token: "DISCORD_BOT_TOKEN"
  api_keys:
    - "KEY_1"
    - "KEY_2"
`
	require.NoError(t, yaml.Unmarshal([]byte(yamlData), &ch))

	// Step 3: Decode — both SecureString and SecureStrings should be populated
	var cfg testDiscordConfig
	require.NoError(t, ch.Decode(&cfg))
	assert.True(t, cfg.MentionOnly)
	assert.Equal(t, "DISCORD_BOT_TOKEN", cfg.Token.String())
	require.Len(t, cfg.ApiKeys, 2)
	assert.Equal(t, "KEY_1", cfg.ApiKeys[0].String())
	assert.Equal(t, "KEY_2", cfg.ApiKeys[1].String())

	// Step 4: Save extend.json — both secure fields removed
	outJSON, err := json.MarshalIndent(ch, "", "  ")
	require.NoError(t, err)
	t.Logf("Saved extend.json:\n%s", string(outJSON))
	assert.NotContains(t, string(outJSON), "token")
	assert.NotContains(t, string(outJSON), "api_keys")
	assert.NotContains(t, string(outJSON), "DISCORD_BOT_TOKEN")
	assert.NotContains(t, string(outJSON), "KEY")
	assert.Contains(t, string(outJSON), "mention_only")

	// Step 5: Save security.yml — only secure fields
	outYAML, err := yaml.Marshal(ch)
	require.NoError(t, err)
	t.Logf("Saved security.yml:\n%s", string(outYAML))
	assert.Contains(t, string(outYAML), "DISCORD_BOT_TOKEN")
	assert.Contains(t, string(outYAML), "KEY_1")
	assert.Contains(t, string(outYAML), "KEY_2")
	assert.NotContains(t, string(outYAML), "mention_only")
	assert.NotContains(t, string(outYAML), "NOT_HERE")
}

func TestChannel_SecureStrings_ApiKeys_EmptyInJSON(t *testing.T) {
	// JSON has no api_keys field
	jsonData := `{
		"enabled": true,
		"type": "discord",
		"settings": {
			"mention_only": true,
			"token": "[NOT_HERE]"
		}
	}`
	var ch Channel
	require.NoError(t, json.Unmarshal([]byte(jsonData), &ch))

	// Merge with api_keys from YAML
	yamlData := `
settings:
  token: "MY_TOKEN"
  api_keys:
    - "KEY_A"
`
	require.NoError(t, yaml.Unmarshal([]byte(yamlData), &ch))

	var cfg testDiscordConfig
	require.NoError(t, ch.Decode(&cfg))
	assert.Equal(t, "MY_TOKEN", cfg.Token.String())
	require.Len(t, cfg.ApiKeys, 1)
	assert.Equal(t, "KEY_A", cfg.ApiKeys[0].String())
}

func TestChannel_SecureStrings_ApiKeys_NoMerge(t *testing.T) {
	// JSON only, no merge — SecureStrings should be empty
	jsonData := `{
		"enabled": true,
		"type": "discord",
		"settings": {
			"mention_only": true,
			"token": "[NOT_HERE]",
			"api_keys": ["[NOT_HERE]"]
		}
	}`
	var ch Channel
	require.NoError(t, json.Unmarshal([]byte(jsonData), &ch))

	var cfg testDiscordConfig
	require.NoError(t, ch.Decode(&cfg))
	assert.True(t, cfg.MentionOnly)
	assert.Equal(t, "", cfg.Token.String())
	// ["[NOT_HERE]"] entries are filtered out → nil
	assert.Nil(t, cfg.ApiKeys)
}

// ═══════════════════════════════════════════════════
//  enc:// token: encrypt → store → merge → decrypt
// ═══════════════════════════════════════════════════

func TestChannel_EncryptedToken(t *testing.T) {
	mustSetupSSHKey(t)

	const testPassphrase = "test-passphrase-123"
	const plainToken = "123456:MY-SECRET-TOKEN"

	// Encrypt the token to get an enc:// string
	encrypted, err := credential.Encrypt(testPassphrase, "", plainToken)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(encrypted, "enc://"), "expected enc:// prefix, got: %s", encrypted)
	t.Logf("encrypted token: %s", encrypted)

	// Replace PassphraseProvider so SecureString.fromRaw can decrypt
	orig := credential.PassphraseProvider
	credential.PassphraseProvider = func() string { return testPassphrase }
	t.Cleanup(func() { credential.PassphraseProvider = orig })

	// Step 1: Load from extend.json (token is [NOT_HERE])
	jsonData := `{
		"enabled": true,
		"type": "telegram",
		"settings": {
			"base_url": "https://api.telegram.org",
			"use_markdown_v2": true,
			"token": "[NOT_HERE]"
		}
	}`
	var ch Channel
	require.NoError(t, json.Unmarshal([]byte(jsonData), &ch))

	// ── Scenario: security.yml stores enc:// token ──
	yamlData := `
settings:
  token: ` + encrypted + `
`
	// Step 2: Merge enc:// token from security.yml
	require.NoError(t, yaml.Unmarshal([]byte(yamlData), &ch))

	// Step 3: Decode — SecureString.fromRaw resolves enc:// → plaintext
	var cfg testTelegramConfig
	require.NoError(t, ch.Decode(&cfg))
	assert.Equal(t, "https://api.telegram.org", cfg.BaseURL)
	assert.True(t, cfg.UseMarkdownV2)
	// The key assertion: enc:// is decrypted to the original plaintext
	assert.Equal(t, plainToken, cfg.Token.String(),
		"SecureString should resolve enc:// to the original plaintext token")

	// Step 4: Save extend.json → token masked as [NOT_HERE]
	outJSON, err := json.MarshalIndent(ch, "", "  ")
	require.NoError(t, err)
	assert.NotContains(t, string(outJSON), "token")
	assert.NotContains(t, string(outJSON), plainToken)
	assert.NotContains(t, string(outJSON), "enc://")

	// Step 5: Save security.yml → token preserved as enc://
	outYAML, err := yaml.Marshal(ch)
	require.NoError(t, err)
	t.Logf("Saved security.yml:\n%s", string(outYAML))
	assert.Contains(t, string(outYAML), encrypted)
	assert.NotContains(t, string(outYAML), plainToken)
	assert.NotContains(t, string(outYAML), "NOT_HERE")
	assert.NotContains(t, string(outYAML), "base_url")
}

// ═══════════════════════════════════════════════════
//  enc:// token directly in extend.json (edge case)
// ═══════════════════════════════════════════════════

func TestChannel_EncryptedTokenInJSON(t *testing.T) {
	mustSetupSSHKey(t)

	const testPassphrase = "json-enc-passphrase"
	const plainToken = "BOT-TOKEN-FROM-JSON"
	const plainToken2 = "new token2"

	encrypted, err := credential.Encrypt(testPassphrase, "", plainToken)
	require.NoError(t, err)

	orig := credential.PassphraseProvider
	credential.PassphraseProvider = func() string { return testPassphrase }
	t.Cleanup(func() { credential.PassphraseProvider = orig })

	// extend.json with enc:// token directly (no merge needed)
	jsonData := `{
		"enabled": true,
		"type": "telegram",
		"settings": {
			"base_url": "https://api.telegram.org",
			"token": ` + `"` + encrypted + `"` + `
		}
	}`
	t.Logf("JSON data:\n%s", jsonData)
	var ch Channel
	require.NoError(t, json.Unmarshal([]byte(jsonData), &ch))

	var cfg testTelegramConfig
	require.NoError(t, ch.Decode(&cfg))
	assert.Equal(t, plainToken, cfg.Token.String(),
		"enc:// token in JSON should be decrypted correctly")

	cfg.Token.Set(plainToken2)
	// No explicit Encode needed — Decode stored &cfg, so modifications are
	// automatically reflected in MarshalJSON/MarshalYAML.

	// Save JSON → masked as [NOT_HERE]
	outJSON, err := json.MarshalIndent(ch, "", "  ")
	require.NoError(t, err)
	t.Logf("Saved extend.json:\n%s", string(outJSON))
	assert.NotContains(t, string(outJSON), "token")
	assert.NotContains(t, string(outJSON), plainToken2)
	assert.NotContains(t, string(outJSON), "enc://")

	// Save YAML → only token, re-encrypted
	outYAML, err := yaml.Marshal(ch)
	require.NoError(t, err)
	t.Logf("Saved security.yml:\n%s", string(outYAML))
	// MarshalYAML re-encrypts with a new random salt/nonce, so verify via round-trip
	assert.Contains(t, string(outYAML), "enc://")

	// Round-trip: unmarshal YAML output through Channel and verify decryption
	var ch2 Channel
	require.NoError(t, yaml.Unmarshal(outYAML, &ch2))
	var cfg2 testTelegramConfig
	require.NoError(t, ch2.Decode(&cfg2))
	assert.Equal(t, plainToken2, cfg2.Token.String())
}

// ═══════════════════════════════════════════════════
//  enc:// token with missing passphrase → error
// ═══════════════════════════════════════════════════

func TestChannel_EncryptedToken_NoPassphrase(t *testing.T) {
	mustSetupSSHKey(t)

	const testPassphrase = "will-be-removed"
	encrypted, err := credential.Encrypt(testPassphrase, "", "secret-token")
	require.NoError(t, err)

	// Ensure no passphrase is available
	orig := credential.PassphraseProvider
	credential.PassphraseProvider = func() string { return "" }
	t.Cleanup(func() { credential.PassphraseProvider = orig })

	jsonData := `{
		"enabled": true,
		"type": "telegram",
		"settings": {
			"base_url": "https://api.telegram.org",
			"token": ` + `"` + encrypted + `"` + `
		}
	}`
	var ch Channel
	require.NoError(t, json.Unmarshal([]byte(jsonData), &ch))

	var cfg testTelegramConfig
	// Decode should fail because enc:// cannot be decrypted without passphrase
	err = ch.Decode(&cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "passphrase required")
}

// ─── helper ───

func mustParseRawNode(s string) RawNode {
	return RawNode(s)
}
