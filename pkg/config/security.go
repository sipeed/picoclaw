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
	"reflect"
	"strings"
	"sync"

	"github.com/caarlos0/env/v11"
	"github.com/tencent-connect/botgo/log"
	"gopkg.in/yaml.v3"

	"github.com/sipeed/picoclaw/pkg/fileutil"
)

const (
	SecurityConfigFile = ".security.yml"
)

func normalizeSecurityConfig(sec *SecurityConfig) *SecurityConfig {
	if sec == nil {
		sec = &SecurityConfig{}
	}
	if sec.ModelList == nil {
		sec.ModelList = map[string]ModelSecurityEntry{}
	}
	if sec.Channels == nil {
		sec.Channels = &ChannelsSecurity{}
	}
	if sec.Web == nil {
		sec.Web = &WebToolsSecurity{}
	}
	if sec.Skills == nil {
		sec.Skills = &SkillsSecurity{}
	}
	return sec
}

// SecurityConfig stores all sensitive data (API keys, tokens, secrets, passwords)
// This data is loaded from security.yml and kept separate from the main config
type SecurityConfig struct {
	// Model API keys. Map key is model_name, can include suffix like "abc:0", "abc:1"
	// for load balancing with same model_name. The suffix ":N" is used to distinguish
	// multiple configs that share the same base model_name.
	ModelList map[string]ModelSecurityEntry `yaml:"model_list"`

	// Channel tokens/secrets
	Channels *ChannelsSecurity `yaml:"channels,omitempty"`

	Web    *WebToolsSecurity `yaml:"web,omitempty"`
	Skills *SkillsSecurity   `yaml:"skills,omitempty"`

	// cache for sensitive values and compiled regex (computed once)
	sensitiveCache *SensitiveDataCache
}

// ModelSecurityEntry stores security data for a model
type ModelSecurityEntry struct {
	APIKeys []string `yaml:"api_keys,omitempty"` // API authentication keys (multiple keys for failover)
}

// ChannelsSecurity stores channel-related security data
type ChannelsSecurity struct {
	Telegram *TelegramSecurity `yaml:"telegram,omitempty"`
	Feishu   *FeishuSecurity   `yaml:"feishu,omitempty"`
	Discord  *DiscordSecurity  `yaml:"discord,omitempty"`
	Weixin   *WeixinSecurity   `yaml:"weixin,omitempty"`
	QQ       *QQSecurity       `yaml:"qq,omitempty"`
	DingTalk *DingTalkSecurity `yaml:"dingtalk,omitempty"`
	Slack    *SlackSecurity    `yaml:"slack,omitempty"`
	Matrix   *MatrixSecurity   `yaml:"matrix,omitempty"`
	LINE     *LINESecurity     `yaml:"line,omitempty"`
	OneBot   *OneBotSecurity   `yaml:"onebot,omitempty"`
	WeCom    *WeComSecurity    `yaml:"wecom,omitempty"`
	Pico     *PicoSecurity     `yaml:"pico,omitempty"`
	IRC      *IRCSecurity      `yaml:"irc,omitempty"`
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
	Secret string `yaml:"secret,omitempty" env:"PICOCLAW_CHANNELS_WECOM_SECRET"`
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
			return normalizeSecurityConfig(nil), nil
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

	return normalizeSecurityConfig(&sec), nil
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

// mergeSecurityConfig merges two SecurityConfig instances, preferring non-empty values from 'newer'.
// This is used during config migration to preserve existing security data while adding new entries.
func mergeSecurityConfig(existing, newer *SecurityConfig) *SecurityConfig {
	if existing == nil {
		return normalizeSecurityConfig(newer)
	}
	if newer == nil {
		return normalizeSecurityConfig(existing)
	}

	result := normalizeSecurityConfig(nil)

	// Merge ModelList: prefer newer if it has keys, otherwise use existing
	for k, v := range existing.ModelList {
		result.ModelList[k] = v
	}
	for k, v := range newer.ModelList {
		if len(v.APIKeys) > 0 {
			result.ModelList[k] = v
		}
	}

	// Merge Channels
	if existing.Channels != nil {
		result.Channels = existing.Channels
	}
	if newer.Channels != nil {
		if result.Channels == nil {
			result.Channels = &ChannelsSecurity{}
		}
		mergeChannelsSecurity(result.Channels, newer.Channels)
	}

	// Merge Web
	if existing.Web != nil {
		result.Web = existing.Web
	}
	if newer.Web != nil {
		if result.Web == nil {
			result.Web = &WebToolsSecurity{}
		}
		mergeWebToolsSecurity(result.Web, newer.Web)
	}

	// Merge Skills
	if existing.Skills != nil {
		result.Skills = existing.Skills
	}
	if newer.Skills != nil {
		if result.Skills == nil {
			result.Skills = &SkillsSecurity{}
		}
		mergeSkillsSecurity(result.Skills, newer.Skills)
	}

	return result
}

func mergeChannelsSecurity(dst, src *ChannelsSecurity) {
	if src.Telegram != nil && src.Telegram.Token != "" {
		dst.Telegram = src.Telegram
	}
	if src.Feishu != nil &&
		(src.Feishu.AppSecret != "" || src.Feishu.EncryptKey != "" || src.Feishu.VerificationToken != "") {
		dst.Feishu = src.Feishu
	}
	if src.Discord != nil && src.Discord.Token != "" {
		dst.Discord = src.Discord
	}
	if src.Weixin != nil && src.Weixin.Token != "" {
		dst.Weixin = src.Weixin
	}
	if src.QQ != nil && src.QQ.AppSecret != "" {
		dst.QQ = src.QQ
	}
	if src.DingTalk != nil && src.DingTalk.ClientSecret != "" {
		dst.DingTalk = src.DingTalk
	}
	if src.Slack != nil && (src.Slack.BotToken != "" || src.Slack.AppToken != "") {
		dst.Slack = src.Slack
	}
	if src.Matrix != nil && src.Matrix.AccessToken != "" {
		dst.Matrix = src.Matrix
	}
	if src.LINE != nil && (src.LINE.ChannelSecret != "" || src.LINE.ChannelAccessToken != "") {
		dst.LINE = src.LINE
	}
	if src.OneBot != nil && src.OneBot.AccessToken != "" {
		dst.OneBot = src.OneBot
	}
	if src.WeCom != nil && src.WeCom.Secret != "" {
		dst.WeCom = src.WeCom
	}
	if src.Pico != nil && src.Pico.Token != "" {
		dst.Pico = src.Pico
	}
	if src.IRC != nil && (src.IRC.Password != "" || src.IRC.NickServPassword != "" || src.IRC.SASLPassword != "") {
		dst.IRC = src.IRC
	}
}

func mergeWebToolsSecurity(dst, src *WebToolsSecurity) {
	if src.Brave != nil && len(src.Brave.APIKeys) > 0 {
		dst.Brave = src.Brave
	}
	if src.Tavily != nil && len(src.Tavily.APIKeys) > 0 {
		dst.Tavily = src.Tavily
	}
	if src.Perplexity != nil && len(src.Perplexity.APIKeys) > 0 {
		dst.Perplexity = src.Perplexity
	}
	if src.GLMSearch != nil && src.GLMSearch.APIKey != "" {
		dst.GLMSearch = src.GLMSearch
	}
	if src.BaiduSearch != nil && src.BaiduSearch.APIKey != "" {
		dst.BaiduSearch = src.BaiduSearch
	}
}

func mergeSkillsSecurity(dst, src *SkillsSecurity) {
	if src.Github != nil && src.Github.Token != "" {
		dst.Github = src.Github
	}
	if src.ClawHub != nil && src.ClawHub.AuthToken != "" {
		dst.ClawHub = src.ClawHub
	}
}

// SensitiveDataCache caches the compiled regex for filtering sensitive data.
// SensitiveDataCache caches the strings.Replacer for filtering sensitive data.
// Computed once on first access via sync.Once.
type SensitiveDataCache struct {
	replacer *strings.Replacer
	once     sync.Once
}

// SensitiveDataReplacer returns the strings.Replacer for filtering sensitive data.
// It is computed once on first access via sync.Once.
func (sec *SecurityConfig) SensitiveDataReplacer() *strings.Replacer {
	sec.initSensitiveCache()
	return sec.sensitiveCache.replacer
}

// initSensitiveCache initializes the sensitive data cache if not already done.
func (sec *SecurityConfig) initSensitiveCache() {
	if sec.sensitiveCache == nil {
		sec.sensitiveCache = &SensitiveDataCache{}
	}
	sec.sensitiveCache.once.Do(func() {
		values := sec.collectSensitiveValues()
		if len(values) == 0 {
			sec.sensitiveCache.replacer = strings.NewReplacer()
			return
		}

		// Build old/new pairs for strings.Replacer
		var pairs []string
		for _, v := range values {
			if len(v) > 3 {
				pairs = append(pairs, v, "[FILTERED]")
			}
		}
		if len(pairs) == 0 {
			sec.sensitiveCache.replacer = strings.NewReplacer()
			return
		}
		sec.sensitiveCache.replacer = strings.NewReplacer(pairs...)
	})
}

// collectSensitiveValues collects all sensitive strings from SecurityConfig using reflection.
func (sec *SecurityConfig) collectSensitiveValues() []string {
	var values []string
	collectSensitive(reflect.ValueOf(sec), &values)
	return values
}

// collectSensitive recursively traverses the value and collects all non-empty string fields.
func collectSensitive(v reflect.Value, values *[]string) {
	// Dereference pointers/interfaces to get the underlying value
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			fieldType := v.Type().Field(i)
			if !fieldType.IsExported() {
				continue
			}
			collectSensitive(field, values)
		}
	case reflect.String:
		if v.String() != "" {
			*values = append(*values, v.String())
		}
	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.String {
			for i := 0; i < v.Len(); i++ {
				if s := v.Index(i).String(); s != "" {
					*values = append(*values, s)
				}
			}
		}
	case reflect.Map:
		for _, key := range v.MapKeys() {
			collectSensitive(v.MapIndex(key), values)
		}
	}
}
