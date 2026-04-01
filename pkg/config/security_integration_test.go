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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test JSON unmarshal of private fields (unexported fields are never filled, with or without json tag).
func TestJSONUnmarshalPrivateFields(t *testing.T) {
	type testStruct struct {
		PublicField  string `json:"public"`
		privateField string
	}

	data := `{"public": "pub", "privateField": "priv"}`
	var s testStruct
	if err := json.Unmarshal([]byte(data), &s); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	t.Logf("PublicField: %s", s.PublicField)
	t.Logf("privateField: %s", s.privateField)

	if s.PublicField != "pub" {
		t.Errorf("PublicField = %q, want 'pub'", s.PublicField)
	}
	if s.privateField != "" {
		t.Errorf("privateField = %q, want empty because unexported fields are ignored", s.privateField)
	}
}

func TestSecurityConfigIntegration(t *testing.T) {
	t.Run("Full workflow with security references", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create config.json with direct security values (not ref: references)
		// These values should take precedence over .security.yml
		configPath := filepath.Join(tmpDir, "config.json")
		configContent := `{
  "version": 1,
  "model_list": [
    {
      "model_name": "test-model",
      "model": "openai/test-model",
      "api_base": "https://api.openai.com/v1",
      "api_key": "sk-from-config-json-direct"
    }
  ],
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "token-from-config-json-direct"
    }
  },
  "tools": {
    "web": {
      "brave": {
        "enabled": true,
        "api_keys": ["BSA-from-config-json-direct"]
      }
    },
    "skills": {
      "github": {
        "token": "ghp-from-config-json-direct"
      }
    }
  }
}`
		err := os.WriteFile(configPath, []byte(configContent), 0o644)
		require.NoError(t, err)

		// Create .security.yml with different values
		// These should be overridden by config.json values
		securityPath := filepath.Join(tmpDir, SecurityConfigFile)
		securityContent := `model_list:
  test-model:
    api_keys:
      - "sk-from-security-yml"

channels:
  telegram:
    token: "token-from-security-yml"

skills:
  github:
    token: "ghp-from-security-yml"`
		err = os.WriteFile(securityPath, []byte(securityContent), 0o600)
		require.NoError(t, err)

		// Load config and verify config.json values take precedence
		cfg, err := LoadConfig(configPath)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		// Verify model API key from config.json takes precedence
		assert.Equal(t, 1, len(cfg.ModelList))
		assert.Equal(t, "test-model", cfg.ModelList[0].ModelName)
		assert.Equal(t, "sk-from-security-yml", cfg.ModelList[0].APIKey())

		// Verify channel token from config.json takes precedence
		assert.Equal(t, "token-from-security-yml", cfg.Channels.Telegram.Token.String())

		assert.Equal(t, "sk-from-security-yml", cfg.ModelList[0].APIKeys[0].String())

		// Verify web tool API key from config.json takes precedence
		assert.Equal(t, "BSA-from-config-json-direct", cfg.Tools.Web.Brave.APIKey())

		// Verify skills token is resolved
		assert.Equal(t, "ghp-from-security-yml", cfg.Tools.Skills.Github.Token.String())
	})
}

func TestSecurityConfigWithAPIKeysArray(t *testing.T) {
	t.Run("Multiple API keys via security", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create config with APIKeys array
		configPath := filepath.Join(tmpDir, "config.json")
		configContent := `{
  "version": 1,
  "model_list": [
    {
      "model_name": "multi-key-model",
      "model": "openai/multi-key-model"
    }
  ]
}`
		err := os.WriteFile(configPath, []byte(configContent), 0o644)
		require.NoError(t, err)

		// Create .security.yml
		securityPath := filepath.Join(tmpDir, SecurityConfigFile)
		securityContent := `model_list:
  multi-key-model:0:
    api_key: "sk-key-1"
    api_keys:
      - "sk-key-1"
      - "sk-key-2"
      - "sk-key-3"
`
		err = os.WriteFile(securityPath, []byte(securityContent), 0o600)
		require.NoError(t, err)

		// Load config
		cfg, err := LoadConfig(configPath)
		require.NoError(t, err)

		t.Logf("Config: %+v", cfg.ModelList)
		for _, m := range cfg.ModelList {
			t.Logf("Model: %+v", m)
		}
		// Verify multi-key expansion works
		assert.Equal(t, 3, len(cfg.ModelList))
		assert.Equal(t, "multi-key-model", cfg.ModelList[2].ModelName)
	})
}

func TestAllSecurityKeysAccessible(t *testing.T) {
	t.Run("All security keys accessible via Key() methods including file://", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create test files for file:// references
		modelAPIKeyFile := filepath.Join(tmpDir, "model_api_key.txt")
		err := os.WriteFile(modelAPIKeyFile, []byte("sk-model-from-file-12345"), 0o600)
		require.NoError(t, err)

		braveAPIKeyFile := filepath.Join(tmpDir, "brave_api_key.txt")
		err = os.WriteFile(braveAPIKeyFile, []byte("BSA-brave-from-file-67890"), 0o600)
		require.NoError(t, err)

		tavilyAPIKeyFile := filepath.Join(tmpDir, "tavily_api_key.txt")
		err = os.WriteFile(tavilyAPIKeyFile, []byte("tvly-tavily-from-file-11111"), 0o600)
		require.NoError(t, err)

		perplexityAPIKeyFile := filepath.Join(tmpDir, "perplexity_api_key.txt")
		err = os.WriteFile(perplexityAPIKeyFile, []byte("pplx-perplexity-from-file-22222"), 0o600)
		require.NoError(t, err)

		githubTokenFile := filepath.Join(tmpDir, "github_token.txt")
		err = os.WriteFile(githubTokenFile, []byte("ghp-github-from-file-abc123"), 0o600)
		require.NoError(t, err)

		clawhubAuthTokenFile := filepath.Join(tmpDir, "clawhub_auth_token.txt")
		err = os.WriteFile(clawhubAuthTokenFile, []byte("clawhub-auth-token-from-file"), 0o600)
		require.NoError(t, err)

		// Create config.json without sensitive values (they'll be in .security.yml)
		configPath := filepath.Join(tmpDir, "config.json")
		configContent := `{
  "version": 1,
  "model_list": [
    {
      "model_name": "test-model-1",
      "model": "openai/test-model-1"
    }
  ],
  "channels": {
    "telegram": {
      "enabled": true
    },
    "feishu": {
      "enabled": true,
      "app_id": "test_app_id"
    },
    "discord": {
      "enabled": true
    },
    "dingtalk": {
      "enabled": true,
      "client_id": "test_client_id"
    },
    "slack": {
      "enabled": true
    },
    "matrix": {
      "enabled": true,
      "homeserver": "https://matrix.org",
      "user_id": "@test:matrix.org"
    },
    "line": {
      "enabled": true,
      "webhook_host": "localhost",
      "webhook_port": 8080,
      "webhook_path": "/webhook"
    },
    "onebot": {
      "enabled": true,
      "ws_url": "ws://localhost:8080"
    },
    "wecom": {
      "enabled": true,
      "bot_id": "test_wecom_bot_id"
    },
    "pico": {
      "enabled": true
    },
    "irc": {
      "enabled": true,
      "server": "irc.example.com",
      "nick": "testbot"
    },
    "qq": {
      "enabled": true,
      "app_id": "test_qq_app_id"
    }
  },
  "tools": {
    "web": {
      "brave": {
        "enabled": true
      },
      "tavily": {
        "enabled": true
      },
      "perplexity": {
        "enabled": true
      },
      "glm_search": {
        "enabled": true
      }
    },
    "skills": {
      "github": {}
    }
  }
}`
		err = os.WriteFile(configPath, []byte(configContent), 0o644)
		require.NoError(t, err)

		// Create .security.yml with file:// references and plaintext values
		securityPath := filepath.Join(tmpDir, SecurityConfigFile)
		securityContent := `model_list:
  test-model-1:
    api_keys:
      - "file://model_api_key.txt"

channels:
  telegram:
    token: "123456789:ABCdefGHIjklMNOpqrsTUVwxyz"
  feishu:
    app_secret: "feishu_test_app_secret"
    encrypt_key: "feishu_test_encrypt_key"
    verification_token: "feishu_test_verification_token"
  discord:
    token: "discord_test_bot_token_xyz"
  dingtalk:
    client_secret: "dingtalk_test_client_secret"
  slack:
    bot_token: "xoxb-slack-bot-token-123"
    app_token: "xapp-slack-app-token-456"
  matrix:
    access_token: "matrix_test_access_token"
  line:
    channel_secret: "line_test_channel_secret"
    channel_access_token: "line_test_channel_access_token"
  onebot:
    access_token: "onebot_test_access_token"
  wecom:
    secret: "wecom_test_secret"
  pico:
    token: "pico_test_token"
  irc:
    password: "irc_test_password"
    nickserv_password: "irc_test_nickserv_password"
    sasl_password: "irc_test_sasl_password"
  qq:
    app_secret: "qq_test_app_secret"

web:
  brave:
    api_keys:
      - "file://brave_api_key.txt"
  tavily:
    api_keys:
      - "file://tavily_api_key.txt"
  perplexity:
    api_keys:
      - "file://perplexity_api_key.txt"
  glm_search:
    api_key: "glm-test-glm-search-key"

skills:
  github:
    token: "file://github_token.txt"
  clawhub:
    auth_token: "file://clawhub_auth_token.txt"
`
		err = os.WriteFile(securityPath, []byte(securityContent), 0o600)
		require.NoError(t, err)

		// Load config and verify all security keys are accessible
		cfg, err := LoadConfig(configPath)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		// Verify Model API keys
		assert.Equal(t, 1, len(cfg.ModelList))
		assert.Equal(t, "test-model-1", cfg.ModelList[0].ModelName)
		// file:// reference should be resolved
		assert.Equal(t, "sk-model-from-file-12345", cfg.ModelList[0].APIKey())
		t.Logf("Model APIKey(): %s", cfg.ModelList[0].APIKey())

		// Verify Channel tokens via Key() methods
		// Telegram
		assert.Equal(t, "123456789:ABCdefGHIjklMNOpqrsTUVwxyz", cfg.Channels.Telegram.Token.String())
		t.Logf("Telegram Token(): %s", cfg.Channels.Telegram.Token.String())

		// Feishu
		assert.Equal(t, "feishu_test_app_secret", cfg.Channels.Feishu.AppSecret.String())
		assert.Equal(t, "feishu_test_encrypt_key", cfg.Channels.Feishu.EncryptKey.String())
		assert.Equal(t, "feishu_test_verification_token", cfg.Channels.Feishu.VerificationToken.String())
		t.Logf("Feishu AppSecret(): %s", cfg.Channels.Feishu.AppSecret.String())
		t.Logf("Feishu EncryptKey(): %s", cfg.Channels.Feishu.EncryptKey.String())
		t.Logf("Feishu VerificationToken(): %s", cfg.Channels.Feishu.VerificationToken.String())

		// Discord
		assert.Equal(t, "discord_test_bot_token_xyz", cfg.Channels.Discord.Token.String())
		t.Logf("Discord Token(): %s", cfg.Channels.Discord.Token.String())

		// DingTalk
		assert.Equal(t, "dingtalk_test_client_secret", cfg.Channels.DingTalk.ClientSecret.String())
		t.Logf("DingTalk ClientSecret(): %s", cfg.Channels.DingTalk.ClientSecret.String())

		// Slack
		assert.Equal(t, "xoxb-slack-bot-token-123", cfg.Channels.Slack.BotToken.String())
		assert.Equal(t, "xapp-slack-app-token-456", cfg.Channels.Slack.AppToken.String())
		t.Logf("Slack BotToken(): %s", cfg.Channels.Slack.BotToken.String())
		t.Logf("Slack AppToken(): %s", cfg.Channels.Slack.AppToken.String())

		// Matrix
		assert.Equal(t, "matrix_test_access_token", cfg.Channels.Matrix.AccessToken.String())
		t.Logf("Matrix AccessToken(): %s", cfg.Channels.Matrix.AccessToken.String())

		// LINE
		assert.Equal(t, "line_test_channel_secret", cfg.Channels.LINE.ChannelSecret.String())
		assert.Equal(t, "line_test_channel_access_token", cfg.Channels.LINE.ChannelAccessToken.String())
		t.Logf("LINE ChannelSecret(): %s", cfg.Channels.LINE.ChannelSecret.String())
		t.Logf("LINE ChannelAccessToken(): %s", cfg.Channels.LINE.ChannelAccessToken.String())

		// OneBot
		assert.Equal(t, "onebot_test_access_token", cfg.Channels.OneBot.AccessToken.String())
		t.Logf("OneBot AccessToken(): %s", cfg.Channels.OneBot.AccessToken.String())

		// WeCom
		assert.Equal(t, "test_wecom_bot_id", cfg.Channels.WeCom.BotID)
		assert.Equal(t, "wecom_test_secret", cfg.Channels.WeCom.Secret.String())
		t.Logf("WeCom BotID: %s", cfg.Channels.WeCom.BotID)
		t.Logf("WeCom Secret(): %s", cfg.Channels.WeCom.Secret.String())

		// Pico
		assert.Equal(t, "pico_test_token", cfg.Channels.Pico.Token.String())
		t.Logf("Pico Token(): %s", cfg.Channels.Pico.Token.String())

		// IRC
		assert.Equal(t, "irc_test_password", cfg.Channels.IRC.Password.String())
		assert.Equal(t, "irc_test_nickserv_password", cfg.Channels.IRC.NickServPassword.String())
		assert.Equal(t, "irc_test_sasl_password", cfg.Channels.IRC.SASLPassword.String())
		t.Logf("IRC Password(): %s", cfg.Channels.IRC.Password.String())
		t.Logf("IRC NickServPassword(): %s", cfg.Channels.IRC.NickServPassword.String())
		t.Logf("IRC SASLPassword(): %s", cfg.Channels.IRC.SASLPassword.String())

		// QQ
		assert.Equal(t, "qq_test_app_secret", cfg.Channels.QQ.AppSecret.String())
		t.Logf("QQ AppSecret(): %s", cfg.Channels.QQ.AppSecret.String())

		// Verify Web tool API keys
		assert.Equal(t, "BSA-brave-from-file-67890", cfg.Tools.Web.Brave.APIKey())
		t.Logf("Brave APIKey(): %s", cfg.Tools.Web.Brave.APIKey())

		assert.Equal(t, "tvly-tavily-from-file-11111", cfg.Tools.Web.Tavily.APIKey())
		t.Logf("Tavily APIKey(): %s", cfg.Tools.Web.Tavily.APIKey())

		assert.Equal(t, "pplx-perplexity-from-file-22222", cfg.Tools.Web.Perplexity.APIKey())
		t.Logf("Perplexity APIKey(): %s", cfg.Tools.Web.Perplexity.APIKey())

		// GLM Search - Note: GLM uses SetAPIKey (lowercase) internally
		t.Logf("GLMSearch APIKey(): %s", cfg.Tools.Web.GLMSearch.APIKey.String())
		assert.Equal(t, "glm-test-glm-search-key", cfg.Tools.Web.GLMSearch.APIKey.String())

		// Verify Skills tokens
		assert.Equal(t, "ghp-github-from-file-abc123", cfg.Tools.Skills.Github.Token.String())
		t.Logf("Github Token(): %s", cfg.Tools.Skills.Github.Token.String())

		assert.Equal(t, "clawhub-auth-token-from-file", cfg.Tools.Skills.Registries.ClawHub.AuthToken.String())
		t.Logf("ClawHub AuthToken(): %s", cfg.Tools.Skills.Registries.ClawHub.AuthToken.String())

		t.Log("All security keys are successfully accessible via their respective Key() methods")
	})
}
