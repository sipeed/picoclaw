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

	"github.com/caarlos0/env/v11"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestSecurityConfig(t *testing.T) {
	t.Run("LoadNonExistent", func(t *testing.T) {
		sec := &Config{Channels: make(ChannelsConfig)}
		err := loadSecurityConfig(sec, "/nonexistent/.security.yml")
		require.NoError(t, err)
		assert.NotNil(t, sec)
		assert.Empty(t, sec.ModelList)
		assert.NotNil(t, sec.Channels)
		assert.NotNil(t, sec.Tools.Web)
		assert.NotNil(t, sec.Tools.Skills)
	})
}

func TestSecurityPath(t *testing.T) {
	tests := []struct {
		name      string
		configDir string
		want      string
	}{
		{
			name:      "standard path",
			configDir: "/home/user/.picoclaw/config.json",
			want:      "/home/user/.picoclaw/.security.yml",
		},
		{
			name:      "nested path",
			configDir: "/path/to/config/myconfig.json",
			want:      "/path/to/config/.security.yml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := securityPath(tt.configDir)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSaveAndLoadSecurityConfig(t *testing.T) {
	t.Run("test for securestring", func(t *testing.T) {
		type testStruct struct {
			Secret SecureString `json:"secret,omitzero" yaml:"secret,omitempty" env:"TEST_SECURE_STRING"`
		}
		s := testStruct{Secret: *NewSecureString("test")}
		out, err := yaml.Marshal(s) // 直接对 SecureString 进行序列化
		require.NoError(t, err)
		t.Logf("output: %v", string(out))
		assert.Equal(t, "secret: test\n", string(out))
		out, err = json.Marshal(s)
		require.NoError(t, err)
		t.Logf("output: %v", string(out))
		assert.Equal(t, "{}", string(out))
	})
	tmpDir := t.TempDir()
	secPath := filepath.Join(tmpDir, SecurityConfigFile)

	original := &Config{
		Version: CurrentVersion,
		ModelList: SecureModelList{
			{
				ModelName: "model1",
				Model:     "test/model",
				APIBase:   "api.example.com",
				APIKeys:   SecureStrings{NewSecureString("key1"), NewSecureString("key2")},
			},
			{
				ModelName: "model2",
				Model:     "test/model2",
				APIBase:   "api2.example.com",
				APIKeys:   SecureStrings{NewSecureString("model2_key")},
			},
		},
		Tools: ToolsConfig{
			Web: WebToolsConfig{
				Brave: BraveConfig{
					Enabled: true,
					APIKeys: SecureStrings{NewSecureString("brave_key")},
				},
			},
			Skills: SkillsToolsConfig{
				Github: SkillsGithubConfig{
					Token: *NewSecureString("github_token"),
					Proxy: "test proxy",
				},
			},
		},
		Channels: func() ChannelsConfig {
			chs := make(ChannelsConfig)
			type def struct {
				name string
				raw  string // raw JSON with actual secure values (bypasses SecureString.MarshalJSON)
			}
			for _, d := range []def{
				{"telegram", `{"enabled":true,"settings":{"token":"telegram_token"}}`},
				{"feishu", `{"enabled":true,"settings":{"app_id":"feishu_app_id","app_secret":"feishu_app_secret"}}`},
				{"discord", `{"enabled":true,"settings":{"token":"discord_token"}}`},
				{"qq", `{"enabled":true,"settings":{"app_secret":"qq_app_secret"}}`},
				{"pico_client", `{"enabled":true,"settings":{"token":"pico_client_token"}}`},
			} {
				bc := &Channel{}
				json.Unmarshal([]byte(d.raw), bc)
				bc.Type = d.name
				switch bc.Type {
				case "qq":
					bc.Decode(&QQSettings{})
				case "telegram":
					bc.Decode(&TelegramSettings{})
				case "discord":
					bc.Decode(&DiscordSettings{})
				case "feishu":
					bc.Decode(&FeishuSettings{})
				case "pico_client":
					bc.Decode(&PicoClientSettings{})
				}
				chs[d.name] = bc
			}
			return chs
		}(),
	}

	t.Run("test for original", func(t *testing.T) {
		assert.Equal(t, 2, len(original.ModelList[0].APIKeys))
		assert.Equal(t, "key1", original.ModelList[0].APIKeys[0].String())
	})

	cfg2 := &Config{}
	t.Run("test for json", func(t *testing.T) {
		marshal, err := json.Marshal(original)
		require.NoError(t, err)
		t.Logf("json: %s", string(marshal))
		assert.NotContains(t, string(marshal), "\"api_keys\"")
		assert.NotContains(t, string(marshal), notHere)

		err = json.Unmarshal(marshal, cfg2)
		require.NoError(t, err)
		require.Equal(t, 2, len(cfg2.ModelList))
		assert.Empty(t, cfg2.ModelList[0].APIKeys)
		assert.Empty(t, cfg2.ModelList[1].APIKeys)
	})

	t.Run("test for save yaml", func(t *testing.T) {
		// Save
		err := saveSecurityConfig(secPath, original)
		require.NoError(t, err)

		// Verify file was created with correct permissions
		info, err := os.Stat(secPath)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), info.Mode())

		file, err := os.ReadFile(secPath)
		assert.NoError(t, err)
		t.Logf("%s", string(file))

		// Parse saved YAML and verify channelTestSaveConfig_EncryptsPlaintextAPIKey secure fields are present
		var saved struct {
			ChannelList map[string]map[string]any `yaml:"channel_list"`
		}
		require.NoError(t, yaml.Unmarshal(file, &saved))
		channels := saved.ChannelList
		getSetting := func(name string) map[string]any {
			return channels[name]["settings"].(map[string]any)
		}
		assert.Contains(t, getSetting("telegram")["token"], "telegram_token")
		assert.Contains(t, getSetting("feishu")["app_secret"], "feishu_app_secret")
		assert.Contains(t, getSetting("discord")["token"], "discord_token")
		assert.Contains(t, getSetting("qq")["app_secret"], "qq_app_secret")
		assert.Contains(t, getSetting("pico_client")["token"], "pico_client_token")

		// Rewrite file with deterministic content for load test (use channel_list)
		yamlOutput := `channel_list:
  telegram:
    token: telegram_token
  feishu:
    app_secret: feishu_app_secret
  discord:
    token: discord_token
  qq:
    app_secret: qq_app_secret
  pico_client:
    token: pico_client_token
model_list:
  model1:0:
    api_keys:
      - key1
      - key2
  model2:0:
    api_keys:
      - model2_key
web:
  brave:
    api_keys:
      - brave_key
skills:
  github:
    token: github_token
`
		err = os.WriteFile(secPath, []byte(yamlOutput), 0o600)
		require.NoError(t, err)
	})

	t.Run("test for load yaml", func(t *testing.T) {
		// Load
		cfg := cfg2
		err := loadSecurityConfig(cfg, secPath)
		require.NoError(t, err)

		t.Logf("%+v", cfg)
		t.Logf("%+v", cfg.Tools.Web.Brave.APIKeys)
		t.Logf("%+v", cfg.Tools.Skills.Github.Token)
		require.EqualValues(t, 2, len(cfg.ModelList))
		assert.Equal(t, "key1", cfg.ModelList[0].APIKeys[0].String())
		assert.Equal(t, "key2", cfg.ModelList[0].APIKeys[1].String())
		assert.Equal(t, "model2_key", cfg.ModelList[1].APIKeys[0].String())
		assert.EqualValues(t, original.Tools.Web.Brave.APIKeys, cfg.Tools.Web.Brave.APIKeys)
	})

	t.Run("test for env overwrite", func(t *testing.T) {
		// This will throw a COMPILER ERROR if SecureString doesn't
		// correctly implement the yaml.Marshaler interface.
		var _ yaml.Marshaler = (*SecureString)(nil)
		// If you are using Value types in your config, also check:
		var _ yaml.Marshaler = SecureString{}

		// Set up a fresh config with a qq channel
		envCfg := &Config{
			Channels: ChannelsConfig{
				"qq": {
					Enabled:  true,
					Type:     "qq",
					Settings: RawNode(`{"enabled":true,"app_secret":"qq_app_secret"}`),
				},
			},
			Tools: original.Tools,
		}

		t.Setenv("PICOCLAW_CHANNELS_QQ_APP_SECRET", "qq_app_secret_env")
		t.Setenv("PICOCLAW_TOOLS_WEB_BRAVE_API_KEYS", "brave_key_env,abc")

		require.NoError(t, env.Parse(envCfg))
		// Channel env overrides need explicit handling since ChannelsConfig is map-based
		require.NoError(t, InitChannelList(envCfg.Channels))

		bc := envCfg.Channels.Get("qq")
		decoded, err := bc.GetDecoded()
		require.NoError(t, err)
		qqCfg := decoded.(*QQSettings)
		assert.Equal(t, "qq_app_secret_env", qqCfg.AppSecret.raw)
		assert.Equal(t, "brave_key_env", envCfg.Tools.Web.Brave.APIKeys[0].raw)
		assert.Equal(t, "abc", envCfg.Tools.Web.Brave.APIKeys[1].raw)
	})
}
