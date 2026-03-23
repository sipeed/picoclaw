// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/caarlos0/env/v11"
	"github.com/tencent-connect/botgo/log"
	"gopkg.in/yaml.v3"

	"github.com/sipeed/picoclaw/pkg/fileutil"
)

const (
	SecurityConfigFile = ".security.yml"
)

// SecurityConfig stores all sensitive data (API keys, tokens, secrets, passwords)
// This data is loaded from security.yml and kept separate from the main config
type SecurityConfig struct {
	// Model API keys. Map key is model_name, can include suffix like "abc:0", "abc:1"
	// for load balancing with same model_name. The suffix ":N" is used to distinguish
	// multiple configs that share the same base model_name.
	ModelList map[string]ModelSecurityEntry `yaml:"model_list,omitempty"`

	// Channel tokens/secrets
	Channels ChannelsSecurity `yaml:"channels,omitempty"`

	Web    WebToolsSecurity `yaml:"web,omitempty"`
	Skills SkillsSecurity   `yaml:"skills,omitempty"`
}

// ModelSecurityEntry stores security data for a model
type ModelSecurityEntry struct {
	APIKeys []string `yaml:"api_keys,omitempty"` // API authentication keys (multiple keys for failover)
}

// ChannelsSecurity stores channel-related security data
type ChannelsSecurity struct {
	Telegram   *TelegramSecurity   `yaml:"telegram,omitempty"`
	Feishu     *FeishuSecurity     `yaml:"feishu,omitempty"`
	Discord    *DiscordSecurity    `yaml:"discord,omitempty"`
	Weixin     *WeixinSecurity     `yaml:"weixin,omitempty"`
	QQ         *QQSecurity         `yaml:"qq,omitempty"`
	DingTalk   *DingTalkSecurity   `yaml:"dingtalk,omitempty"`
	Slack      *SlackSecurity      `yaml:"slack,omitempty"`
	Matrix     *MatrixSecurity     `yaml:"matrix,omitempty"`
	LINE       *LINESecurity       `yaml:"line,omitempty"`
	OneBot     *OneBotSecurity     `yaml:"onebot,omitempty"`
	WeCom      *WeComSecurity      `yaml:"wecom,omitempty"`
	WeComApp   *WeComAppSecurity   `yaml:"wecom_app,omitempty"`
	WeComAIBot *WeComAIBotSecurity `yaml:"wecom_aibot,omitempty"`
	Pico       *PicoSecurity       `yaml:"pico,omitempty"`
	IRC        *IRCSecurity        `yaml:"irc,omitempty"`
}

type TelegramSecurity struct {
	Token string `yaml:"token,omitempty" env:"PICOCLAW_CHANNELS_TELEGRAM_TOKEN"`
}

type FeishuSecurity struct {
	AppSecret         string `yaml:"app_secret,omitempty"         env:"PICOCLAW_CHANNELS_FEISHU_APP_SECRET"`
	EncryptKey        string `yaml:"encrypt_key,omitempty"        env:"PICOCLAW_CHANNELS_FEISHU_ENCRYPT_KEY"`
	VerificationToken string `yaml:"verification_token,omitempty" env:"PICOCLAW_CHANNELS_FEISHU_VERIFICATION_TOKEN"`
}

type DiscordSecurity struct {
	Token string `yaml:"token,omitempty" env:"PICOCLAW_CHANNELS_DISCORD_TOKEN"`
}

type WeixinSecurity struct {
	Token string `yaml:"token,omitempty" env:"PICOCLAW_CHANNELS_WEIXIN_TOKEN"`
}

type QQSecurity struct {
	AppSecret string `yaml:"app_secret,omitempty" env:"PICOCLAW_CHANNELS_QQ_APP_SECRET"`
}

type DingTalkSecurity struct {
	ClientSecret string `yaml:"client_secret,omitempty" env:"PICOCLAW_CHANNELS_DINGTALK_CLIENT_SECRET"`
}

type SlackSecurity struct {
	BotToken string `yaml:"bot_token,omitempty" env:"PICOCLAW_CHANNELS_SLACK_BOT_TOKEN"`
	AppToken string `yaml:"app_token,omitempty" env:"PICOCLAW_CHANNELS_SLACK_APP_TOKEN"`
}

type MatrixSecurity struct {
	AccessToken string `yaml:"access_token,omitempty" env:"PICOCLAW_CHANNELS_MATRIX_ACCESS_TOKEN"`
}

type LINESecurity struct {
	ChannelSecret      string `yaml:"channel_secret,omitempty"       env:"PICOCLAW_CHANNELS_LINE_CHANNEL_SECRET"`
	ChannelAccessToken string `yaml:"channel_access_token,omitempty" env:"PICOCLAW_CHANNELS_LINE_CHANNEL_ACCESS_TOKEN"`
}

type OneBotSecurity struct {
	AccessToken string `yaml:"access_token,omitempty" env:"PICOCLAW_CHANNELS_ONEBOT_ACCESS_TOKEN"`
}

type WeComSecurity struct {
	Token          string `yaml:"token,omitempty"            env:"PICOCLAW_CHANNELS_WECOM_TOKEN"`
	EncodingAESKey string `yaml:"encoding_aes_key,omitempty" env:"PICOCLAW_CHANNELS_WECOM_ENCODING_AES_KEY"`
}

type WeComAppSecurity struct {
	CorpSecret     string `yaml:"corp_secret,omitempty"      env:"PICOCLAW_CHANNELS_WECOM_APP_CORP_SECRET"`
	Token          string `yaml:"token,omitempty"            env:"PICOCLAW_CHANNELS_WECOM_APP_TOKEN"`
	EncodingAESKey string `yaml:"encoding_aes_key,omitempty" env:"PICOCLAW_CHANNELS_WECOM_APP_ENCODING_AES_KEY"`
}

type WeComAIBotSecurity struct {
	Secret         string `yaml:"secret,omitempty"           env:"PICOCLAW_CHANNELS_WECOM_AIBOT_SECRET"`
	Token          string `yaml:"token,omitempty"            env:"PICOCLAW_CHANNELS_WECOM_AIBOT_TOKEN"`
	EncodingAESKey string `yaml:"encoding_aes_key,omitempty" env:"PICOCLAW_CHANNELS_WECOM_AIBOT_ENCODING_AES_KEY"`
}

type PicoSecurity struct {
	Token string `yaml:"token,omitempty" env:"PICOCLAW_CHANNELS_PICO_TOKEN"`
}

type IRCSecurity struct {
	Password         string `yaml:"password,omitempty"          env:"PICOCLAW_CHANNELS_IRC_PASSWORD"`
	NickServPassword string `yaml:"nickserv_password,omitempty" env:"PICOCLAW_CHANNELS_IRC_NICKSERV_PASSWORD"`
	SASLPassword     string `yaml:"sasl_password,omitempty"     env:"PICOCLAW_CHANNELS_IRC_SASL_PASSWORD"`
}

type WebToolsSecurity struct {
	Brave       *BraveSecurity       `yaml:"brave,omitempty"`
	Tavily      *TavilySecurity      `yaml:"tavily,omitempty"`
	Perplexity  *PerplexitySecurity  `yaml:"perplexity,omitempty"`
	GLMSearch   *GLMSearchSecurity   `yaml:"glm_search,omitempty"`
	BaiduSearch *BaiduSearchSecurity `yaml:"baidu_search,omitempty"`
}

type BraveSecurity struct {
	APIKeys []string `yaml:"api_keys,omitempty"`
}

type TavilySecurity struct {
	APIKeys []string `yaml:"api_keys,omitempty"`
}

type PerplexitySecurity struct {
	APIKeys []string `yaml:"api_keys,omitempty"`
}

type GLMSearchSecurity struct {
	APIKey string `yaml:"api_key,omitempty"`
}

type BaiduSearchSecurity struct {
	APIKey string `yaml:"api_key,omitempty" env:"PICOCLAW_TOOLS_WEB_BAIDU_API_KEY"`
}

type SkillsSecurity struct {
	Github  *GithubSecurity  `yaml:"github,omitempty"`
	ClawHub *ClawHubSecurity `yaml:"clawhub,omitempty"`
}

type GithubSecurity struct {
	Token string `yaml:"token,omitempty"`
}

type ClawHubSecurity struct {
	AuthToken string `yaml:"auth_token,omitempty"`
}

// securityPath returns the path to security.yml relative to the config file
func securityPath(configPath string) string {
	configDir := filepath.Dir(configPath)
	return filepath.Join(configDir, SecurityConfigFile)
}

// loadSecurityConfig loads the security configuration from security.yml
// Returns an empty SecurityConfig if the file doesn't exist
func loadSecurityConfig(securityPath string) (*SecurityConfig, error) {
	data, err := os.ReadFile(securityPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &SecurityConfig{}, nil
		}
		return nil, fmt.Errorf("failed to read security config: %w", err)
	}

	var sec SecurityConfig
	if err := yaml.Unmarshal(data, &sec); err != nil {
		return nil, fmt.Errorf("failed to parse security config: %w", err)
	}

	// No need to validate model_name format here - both formats are supported:
	// - "model-name:0" (with index for multiple entries)
	// - "model-name" (without index for single entry or default to index 0)

	if err := env.Parse(&sec); err != nil {
		log.Errorf("failed to parse environment variables: %v", err)
		return nil, err
	}

	return &sec, nil
}

// saveSecurityConfig saves the security configuration to security.yml
func saveSecurityConfig(securityPath string, sec *SecurityConfig) error {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	err := enc.Encode(sec)
	if err != nil {
		return fmt.Errorf("failed to marshal security config: %w", err)
	}
	return fileutil.WriteFileAtomic(securityPath, buf.Bytes(), 0o600)
}
