package plugin

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/config"
)

func TestNewLintSubcommand(t *testing.T) {
	cmd := newLintSubcommand()

	require.NotNil(t, cmd)

	assert.Equal(t, "lint", cmd.Use)
	assert.Equal(t, "Lint plugin configuration", cmd.Short)

	assert.Nil(t, cmd.Run)
	assert.NotNil(t, cmd.RunE)

	assert.False(t, cmd.HasSubCommands())
	assert.True(t, cmd.HasFlags())

	configFlag := cmd.Flags().Lookup("config")
	require.NotNil(t, configFlag)
	assert.Equal(t, internal.GetConfigPath(), configFlag.DefValue)
}

func TestPluginLint_UnknownEnabledExitNonZero(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	cfg := config.DefaultConfig()
	cfg.Plugins = config.PluginsConfig{
		DefaultEnabled: false,
		Enabled:        []string{"missing-plugin"},
		Disabled:       []string{},
	}
	require.NoError(t, config.SaveConfig(configPath, cfg))

	cmd := NewPluginCommand()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"lint", "--config", configPath})

	err := cmd.Execute()
	require.Error(t, err)
	assert.ErrorContains(t, err, "missing-plugin")
}

func TestPluginLint_ValidConfigExitZero(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	cfg := config.DefaultConfig()
	cfg.Plugins = config.PluginsConfig{
		DefaultEnabled: false,
		Enabled:        []string{},
		Disabled:       []string{},
	}
	require.NoError(t, config.SaveConfig(configPath, cfg))

	out := &bytes.Buffer{}
	cmd := NewPluginCommand()
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"lint", "--config", configPath})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, out.String(), "plugin config lint: ok")
}
