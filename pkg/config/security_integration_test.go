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

// Test JSON unmarshal of private fields
func TestJSONUnmarshalPrivateFields(t *testing.T) {
	//nolint: govet
	type testStruct struct {
		PublicField  string `json:"public"`
		privateField string `json:"private"`
	}

	data := `{"public": "pub", "private": "priv"}`
	var s testStruct
	if err := json.Unmarshal([]byte(data), &s); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	t.Logf("PublicField: %s", s.PublicField)
	t.Logf("privateField: %s", s.privateField)

	if s.PublicField != "pub" {
		t.Errorf("PublicField = %q, want 'pub'", s.PublicField)
	}
	// This should fail because privateField is unexported
	if s.privateField != "priv" {
		t.Logf("privateField = %q, want 'priv' - THIS IS EXPECTED TO FAIL", s.privateField)
	}
}

func TestSecurityConfigIntegration(t *testing.T) {
	t.Run("Full workflow with security references", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create config.json with references
		configPath := filepath.Join(tmpDir, "config.json")
		configContent := `{
  "version": 1,
  "model_list": [
    {
      "model_name": "test-model",
      "model": "openai/test-model",
      "api_base": "https://api.openai.com/v1",
      "api_key": "ref:model_list.test-model.api_key"
    }
  ],
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "ref:channels.telegram.token"
    }
  },
  "tools": {
    "web": {
      "brave": {
        "enabled": true,
        "api_key": "ref:web.brave.api_key"
      }
    },
    "skills": {
      "github": {
        "token": "ref:skills.github.token"
      }
    }
  }
}`
		err := os.WriteFile(configPath, []byte(configContent), 0o644)
		require.NoError(t, err)

		// Create security.yml with actual values
		securityPath := filepath.Join(tmpDir, "security.yml")
		securityContent := `model_list:
  test-model:
    api_keys:
      - "sk-test-api-key-12345"

channels:
  telegram:
    token: "123456789:ABCdefGHIjklMNOpqrsTUVwxyz"

web:
  brave:
    api_keys:
      - "BSAbrave-api-key-67890"

skills:
  github:
    token: "ghp_github-token-abc123"`
		err = os.WriteFile(securityPath, []byte(securityContent), 0o600)
		require.NoError(t, err)

		// Load config and verify references are resolved
		cfg, err := LoadConfig(configPath)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		// Verify model API key is resolved
		assert.Equal(t, 1, len(cfg.ModelList))
		assert.Equal(t, "test-model", cfg.ModelList[0].ModelName)
		assert.Equal(t, "sk-test-api-key-12345", cfg.ModelList[0].apiKeys[0])

		// Verify channel token is resolved
		assert.Equal(t, "123456789:ABCdefGHIjklMNOpqrsTUVwxyz", cfg.Channels.Telegram.token)

		// Verify web tool API key is resolved
		assert.Equal(t, "BSAbrave-api-key-67890", cfg.Tools.Web.Brave.APIKey())

		// Verify skills token is resolved
		assert.Equal(t, "ghp_github-token-abc123", cfg.Tools.Skills.Github.token)
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

		// Create security.yml
		securityPath := filepath.Join(tmpDir, "security.yml")
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

		// Create config.json without sensitive values (they'll be in security.yml)
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
      "webhook_url": "https://qyapi.weixin.qq.com/cgi-bin/webhook"
    },
    "wecom_app": {
      "enabled": true,
      "corp_id": "test_corp_id",
      "agent_id": 123456
    },
    "wecom_aibot": {
      "enabled": true
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

		// Create security.yml with file:// references and plaintext values
		securityPath := filepath.Join(tmpDir, "security.yml")
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
    token: "wecom_test_webhook_token"
    encoding_aes_key: "wecom_test_aes_key"
  wecom_app:
    corp_secret: "wecom_app_test_corp_secret"
    token: "wecom_app_test_token"
    encoding_aes_key: "wecom_app_test_aes_key"
  wecom_aibot:
    token: "wecom_aibot_test_token"
    encoding_aes_key: "wecom_aibot_test_aes_key"
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
		assert.Equal(t, "123456789:ABCdefGHIjklMNOpqrsTUVwxyz", cfg.Channels.Telegram.Token())
		t.Logf("Telegram Token(): %s", cfg.Channels.Telegram.Token())

		// Feishu
		assert.Equal(t, "feishu_test_app_secret", cfg.Channels.Feishu.AppSecret())
		assert.Equal(t, "feishu_test_encrypt_key", cfg.Channels.Feishu.EncryptKey())
		assert.Equal(t, "feishu_test_verification_token", cfg.Channels.Feishu.VerificationToken())
		t.Logf("Feishu AppSecret(): %s", cfg.Channels.Feishu.AppSecret())
		t.Logf("Feishu EncryptKey(): %s", cfg.Channels.Feishu.EncryptKey())
		t.Logf("Feishu VerificationToken(): %s", cfg.Channels.Feishu.VerificationToken())

		// Discord
		assert.Equal(t, "discord_test_bot_token_xyz", cfg.Channels.Discord.Token())
		t.Logf("Discord Token(): %s", cfg.Channels.Discord.Token())

		// DingTalk
		assert.Equal(t, "dingtalk_test_client_secret", cfg.Channels.DingTalk.ClientSecret())
		t.Logf("DingTalk ClientSecret(): %s", cfg.Channels.DingTalk.ClientSecret())

		// Slack
		assert.Equal(t, "xoxb-slack-bot-token-123", cfg.Channels.Slack.BotToken())
		assert.Equal(t, "xapp-slack-app-token-456", cfg.Channels.Slack.AppToken())
		t.Logf("Slack BotToken(): %s", cfg.Channels.Slack.BotToken())
		t.Logf("Slack AppToken(): %s", cfg.Channels.Slack.AppToken())

		// Matrix
		assert.Equal(t, "matrix_test_access_token", cfg.Channels.Matrix.AccessToken())
		t.Logf("Matrix AccessToken(): %s", cfg.Channels.Matrix.AccessToken())

		// LINE
		assert.Equal(t, "line_test_channel_secret", cfg.Channels.LINE.ChannelSecret())
		assert.Equal(t, "line_test_channel_access_token", cfg.Channels.LINE.ChannelAccessToken())
		t.Logf("LINE ChannelSecret(): %s", cfg.Channels.LINE.ChannelSecret())
		t.Logf("LINE ChannelAccessToken(): %s", cfg.Channels.LINE.ChannelAccessToken())

		// OneBot
		assert.Equal(t, "onebot_test_access_token", cfg.Channels.OneBot.AccessToken())
		t.Logf("OneBot AccessToken(): %s", cfg.Channels.OneBot.AccessToken())

		// WeCom
		assert.Equal(t, "wecom_test_webhook_token", cfg.Channels.WeCom.Token())
		assert.Equal(t, "wecom_test_aes_key", cfg.Channels.WeCom.EncodingAESKey())
		t.Logf("WeCom Token(): %s", cfg.Channels.WeCom.Token())
		t.Logf("WeCom EncodingAESKey(): %s", cfg.Channels.WeCom.EncodingAESKey())

		// WeCom App
		assert.Equal(t, "wecom_app_test_corp_secret", cfg.Channels.WeComApp.CorpSecret())
		assert.Equal(t, "wecom_app_test_token", cfg.Channels.WeComApp.Token())
		assert.Equal(t, "wecom_app_test_aes_key", cfg.Channels.WeComApp.EncodingAESKey())
		t.Logf("WeComApp CorpSecret(): %s", cfg.Channels.WeComApp.CorpSecret())
		t.Logf("WeComApp Token(): %s", cfg.Channels.WeComApp.Token())
		t.Logf("WeComApp EncodingAESKey(): %s", cfg.Channels.WeComApp.EncodingAESKey())

		// WeCom AI Bot
		assert.Equal(t, "wecom_aibot_test_token", cfg.Channels.WeComAIBot.Token())
		assert.Equal(t, "wecom_aibot_test_aes_key", cfg.Channels.WeComAIBot.EncodingAESKey())
		t.Logf("WeComAIBot Token(): %s", cfg.Channels.WeComAIBot.Token())
		t.Logf("WeComAIBot EncodingAESKey(): %s", cfg.Channels.WeComAIBot.EncodingAESKey())

		// Pico
		assert.Equal(t, "pico_test_token", cfg.Channels.Pico.Token())
		t.Logf("Pico Token(): %s", cfg.Channels.Pico.Token())

		// IRC
		assert.Equal(t, "irc_test_password", cfg.Channels.IRC.Password())
		assert.Equal(t, "irc_test_nickserv_password", cfg.Channels.IRC.NickServPassword())
		assert.Equal(t, "irc_test_sasl_password", cfg.Channels.IRC.SASLPassword())
		t.Logf("IRC Password(): %s", cfg.Channels.IRC.Password())
		t.Logf("IRC NickServPassword(): %s", cfg.Channels.IRC.NickServPassword())
		t.Logf("IRC SASLPassword(): %s", cfg.Channels.IRC.SASLPassword())

		// QQ
		assert.Equal(t, "qq_test_app_secret", cfg.Channels.QQ.AppSecret())
		t.Logf("QQ AppSecret(): %s", cfg.Channels.QQ.AppSecret())

		// Verify Web tool API keys
		assert.Equal(t, "BSA-brave-from-file-67890", cfg.Tools.Web.Brave.APIKey())
		t.Logf("Brave APIKey(): %s", cfg.Tools.Web.Brave.APIKey())

		assert.Equal(t, "tvly-tavily-from-file-11111", cfg.Tools.Web.Tavily.APIKey())
		t.Logf("Tavily APIKey(): %s", cfg.Tools.Web.Tavily.APIKey())

		assert.Equal(t, "pplx-perplexity-from-file-22222", cfg.Tools.Web.Perplexity.APIKey())
		t.Logf("Perplexity APIKey(): %s", cfg.Tools.Web.Perplexity.APIKey())

		// GLM Search - Note: GLM uses SetAPIKey (lowercase) internally
		t.Logf("GLMSearch APIKey(): %s", cfg.Tools.Web.GLMSearch.APIKey())
		assert.Equal(t, "glm-test-glm-search-key", cfg.Tools.Web.GLMSearch.APIKey())

		// Verify Skills tokens
		assert.Equal(t, "ghp-github-from-file-abc123", cfg.Tools.Skills.Github.Token())
		t.Logf("Github Token(): %s", cfg.Tools.Skills.Github.Token())

		assert.Equal(t, "clawhub-auth-token-from-file", cfg.Tools.Skills.Registries.ClawHub.AuthToken())
		t.Logf("ClawHub AuthToken(): %s", cfg.Tools.Skills.Registries.ClawHub.AuthToken())

		t.Log("All security keys are successfully accessible via their respective Key() methods")
	})
}
