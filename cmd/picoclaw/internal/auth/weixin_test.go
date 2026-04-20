package auth

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/config"
)

func TestNewWeixinCommandHasChannelFlag(t *testing.T) {
	cmd := newWeixinCommand()

	flag := cmd.Flags().Lookup("channel")
	require.NotNil(t, flag)
	assert.Equal(t, config.ChannelWeixin, flag.DefValue)
}

func TestSaveWeixinConfigCreatesNamedChannel(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	t.Setenv(config.EnvHome, tmpDir)
	t.Setenv(config.EnvConfig, configPath)

	err := saveWeixinConfig(
		"weixin_personal",
		"token-personal",
		"account-personal",
		"https://region.example.com/",
		"http://127.0.0.1:7890",
	)
	require.NoError(t, err)

	cfg, err := config.LoadConfig(internal.GetConfigPath())
	require.NoError(t, err)
	bc := cfg.Channels.Get("weixin_personal")
	require.NotNil(t, bc)
	assert.True(t, bc.Enabled)
	assert.Equal(t, config.ChannelWeixin, bc.Type)

	decoded, err := bc.GetDecoded()
	require.NoError(t, err)
	wxCfg := decoded.(*config.WeixinSettings)
	assert.Equal(t, "token-personal", wxCfg.Token.String())
	assert.Equal(t, "account-personal", wxCfg.AccountID)
	assert.Equal(t, "https://region.example.com/", wxCfg.BaseURL)
	assert.Equal(t, "http://127.0.0.1:7890", wxCfg.Proxy)
}

func TestSaveWeixinConfigDoesNotOverwriteOtherWeixinChannels(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	t.Setenv(config.EnvHome, tmpDir)
	t.Setenv(config.EnvConfig, configPath)

	require.NoError(t, saveWeixinConfig("weixin_a", "token-a", "account-a", "", ""))
	require.NoError(t, saveWeixinConfig("weixin_b", "token-b", "account-b", "", ""))

	cfg, err := config.LoadConfig(internal.GetConfigPath())
	require.NoError(t, err)

	a := cfg.Channels.Get("weixin_a")
	require.NotNil(t, a)
	aDecoded, err := a.GetDecoded()
	require.NoError(t, err)
	aCfg := aDecoded.(*config.WeixinSettings)
	assert.Equal(t, "token-a", aCfg.Token.String())
	assert.Equal(t, "account-a", aCfg.AccountID)

	b := cfg.Channels.Get("weixin_b")
	require.NotNil(t, b)
	bDecoded, err := b.GetDecoded()
	require.NoError(t, err)
	bCfg := bDecoded.(*config.WeixinSettings)
	assert.Equal(t, "token-b", bCfg.Token.String())
	assert.Equal(t, "account-b", bCfg.AccountID)
}

func TestSaveWeixinConfigRejectsExistingDifferentType(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	t.Setenv(config.EnvHome, tmpDir)
	t.Setenv(config.EnvConfig, configPath)

	cfg := config.DefaultConfig()
	cfg.Channels["telegram_alias"] = &config.Channel{
		Enabled: true,
		Type:    config.ChannelTelegram,
	}
	require.NoError(t, config.SaveConfig(configPath, cfg))

	err := saveWeixinConfig("telegram_alias", "token", "account", "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists with type")
}
