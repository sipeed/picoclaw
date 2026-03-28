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

	"github.com/sipeed/picoclaw/pkg/credential"
)

func TestSecurityConfig(t *testing.T) {
	t.Run("LoadNonExistent", func(t *testing.T) {
		sec := &Config{}
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
		Channels: ChannelsConfig{
			Telegram: TelegramConfig{
				Enabled: true,
				Token:   *NewSecureString("telegram_token"),
			},
			Feishu: FeishuConfig{
				Enabled:   true,
				AppID:     "feishu_app_id",
				AppSecret: *NewSecureString("feishu_app_secret"),
			},
			Discord: DiscordConfig{
				Enabled: true,
				Token:   *NewSecureString("discord_token"),
			},
			QQ: QQConfig{
				Enabled:   true,
				AppSecret: *NewSecureString("qq_app_secret"),
			},
			PicoClient: PicoClientConfig{
				Enabled: true,
				Token:   *NewSecureString("pico_client_token"),
			},
		},
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
		assert.Contains(t, string(marshal), "\"api_keys\"")
		assert.Contains(t, string(marshal), notHere)

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
		yamlOutput := `channels:
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
		assert.Equal(t, yamlOutput, string(file))

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
		t.Setenv("PICOCLAW_CHANNELS_QQ_APP_SECRET", "qq_app_secret_env")
		t.Setenv("PICOCLAW_TOOLS_WEB_BRAVE_API_KEYS", "brave_key_env,abc")
		err2 := env.Parse(cfg2)
		require.NoError(t, err2)
		assert.Equal(t, "qq_app_secret_env", cfg2.Channels.QQ.AppSecret.raw)
		assert.Equal(t, "brave_key_env", cfg2.Tools.Web.Brave.APIKeys[0].raw)
		assert.Equal(t, "abc", cfg2.Tools.Web.Brave.APIKeys[1].raw)
	})
}

func TestLoadSecurityValue(t *testing.T) {
	type valueStruct struct {
		Url     string        `json:"url,omitempty"      yaml:"-"`
		Token   *SecureString `json:"token,omitempty"    yaml:"token,omitempty"    env:"PICO_TOKEN"`
		ApiKeys SecureStrings `json:"api_keys,omitempty" yaml:"api_keys,omitempty" env:"PICO_API_KEYS"`
	}

	type testStruct struct {
		Pico *valueStruct `json:"pico,omitempty" yaml:"pico,omitempty"`
	}

	v1 := &testStruct{
		Pico: &valueStruct{
			Url:     "https://example.com",
			Token:   NewSecureString("token1"),
			ApiKeys: SecureStrings{NewSecureString("api-key1"), NewSecureString("api-key2")},
		},
	}
	bytes, err := yaml.Marshal(v1)
	assert.NoError(t, err)
	jsonBytes, err := json.Marshal(v1)
	assert.NoError(t, err)
	const want = `pico:
    token: token1
    api_keys:
        - api-key1
        - api-key2
`
	const jsonPost = `{"pico":{"url":"https://example.com","token":"token0"}}`
	v0 := &testStruct{}
	err = json.Unmarshal([]byte(jsonPost), v0)
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com", v0.Pico.Url)
	assert.Equal(t, "token0", v0.Pico.Token.String())

	const jsonWant = `{"pico":{"url":"https://example.com","token":"[NOT_HERE]","api_keys":"[NOT_HERE]"}}`
	assert.Equal(t, want, string(bytes))
	assert.Equal(t, jsonWant, string(jsonBytes))

	v2 := &testStruct{}
	err = json.Unmarshal(jsonBytes, v2)
	assert.NoError(t, err)
	err = yaml.Unmarshal(bytes, v2)
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com", v2.Pico.Url)
	if v2.Pico.Token != nil {
		assert.Equal(t, "token1", v2.Pico.Token.String())
		assert.Equal(t, "token1", v2.Pico.Token.raw)
	}

	v2.Pico.Token = NewSecureString("token1")
	v2.Pico.Token.raw = "abc"
	err = yaml.Unmarshal(bytes, v2)
	assert.NoError(t, err)
	assert.Equal(t, "token1", v2.Pico.Token.raw)

	os.Setenv("PICO_TOKEN", "token_env")
	err = env.Parse(v2)
	assert.NoError(t, err)
	assert.NotNil(t, v2.Pico.Token)
	assert.Equal(t, "token1", v2.Pico.Token.String())

	v3 := &testStruct{Pico: &valueStruct{}}
	err = env.Parse(v3)
	assert.NoError(t, err)
	if v3.Pico.Token != nil {
		assert.Equal(t, "token_env", v3.Pico.Token.String())
	}

	type toolsStruct struct {
		Pico valueStruct `json:"pico,omitempty" yaml:"pico,omitempty"`
	}

	type testStruct2 struct {
		Tools toolsStruct `json:"tools,omitempty" yaml:",inline"`
	}

	v4 := &testStruct2{
		Tools: toolsStruct{
			Pico: valueStruct{
				Url:     "https://example.com",
				Token:   NewSecureString("token1"),
				ApiKeys: SecureStrings{NewSecureString("api-key1"), NewSecureString("api-key2")},
			},
		},
	}
	bytes, err = yaml.Marshal(v4)
	assert.NoError(t, err)
	assert.Equal(t, want, string(bytes))
	jsonBytes, err = json.Marshal(v4)
	assert.NoError(t, err)
	assert.Equal(
		t,
		`{"tools":{"pico":{"url":"https://example.com","token":"[NOT_HERE]","api_keys":"[NOT_HERE]"}}}`,
		string(jsonBytes),
	)

	v5 := &testStruct2{}
	err = json.Unmarshal(jsonBytes, v5)
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com", v5.Tools.Pico.Url)
	err = yaml.Unmarshal(bytes, v5)
	assert.NoError(t, err)
	assert.NotNil(t, v5.Tools.Pico.Token)
	assert.Equal(t, "token1", v5.Tools.Pico.Token.raw)

	dir := t.TempDir()
	sshKeyPath := filepath.Join(dir, "picoclaw_ed25519.key")
	if err = os.WriteFile(sshKeyPath, []byte("fake-ssh-key-material\n"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	const passphrase = "test-passphrase-32bytes-long-ok!"

	t.Setenv(credential.SSHKeyPathEnvVar, sshKeyPath)

	t.Setenv(credential.PassphraseEnvVar, passphrase)

	v5.Tools.Pico.Token.Set("newtoken1")
	v5.Tools.Pico.ApiKeys[0].Set("newapi-key1")
	bytes, err = yaml.Marshal(v5)
	assert.NoError(t, err)
	t.Logf("yaml: %s", string(bytes))

	v6 := &testStruct2{}
	err = yaml.Unmarshal(bytes, v6)
	assert.NoError(t, err)
	assert.NotNil(t, v6.Tools.Pico.Token)
	assert.Equal(t, "newtoken1", v6.Tools.Pico.Token.String())
}
