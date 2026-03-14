package main

import (
	"fmt"
	"slices"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/config"
)

func TestNewPicoclawCommand(t *testing.T) {
	cmd := NewPicoclawCommand()

	require.NotNil(t, cmd)

	short := fmt.Sprintf("%s picoclaw - Personal AI Assistant v%s\n\n", internal.Logo, config.GetVersion())

	assert.Equal(t, "picoclaw", cmd.Use)
	assert.Equal(t, short, cmd.Short)

	assert.True(t, cmd.HasSubCommands())
	assert.True(t, cmd.HasAvailableSubCommands())

	assert.False(t, cmd.HasFlags())

	assert.Nil(t, cmd.Run)
	assert.Nil(t, cmd.RunE)

	assert.Nil(t, cmd.PersistentPreRun)
	assert.Nil(t, cmd.PersistentPostRun)

	allowedCommands := []string{
		"agent",
		"auth",
		"cron",
		"gateway",
		"migrate",
		"model",
		"onboard",
		"skills",
		"status",
		"version",
	}

	subcommands := cmd.Commands()
	assert.Len(t, subcommands, len(allowedCommands))

	for _, subcmd := range subcommands {
		found := slices.Contains(allowedCommands, subcmd.Name())
		assert.True(t, found, "unexpected subcommand %q", subcmd.Name())

		assert.False(t, subcmd.Hidden)
	}
}

func TestShouldPrintBanner(t *testing.T) {
	t.Run("interactive command prints banner", func(t *testing.T) {
		t.Setenv(noBannerEnv, "")
		assert.True(t, shouldPrintBanner([]string{"picoclaw", "agent"}, true))
	})

	t.Run("redirected stdout suppresses banner", func(t *testing.T) {
		t.Setenv(noBannerEnv, "")
		assert.False(t, shouldPrintBanner([]string{"picoclaw", "agent"}, false))
	})

	t.Run("completion command suppresses banner", func(t *testing.T) {
		t.Setenv(noBannerEnv, "")
		assert.False(t, shouldPrintBanner([]string{"picoclaw", "completion", "zsh"}, true))
		assert.False(t, shouldPrintBanner([]string{"picoclaw", cobra.ShellCompRequestCmd}, true))
		assert.False(t, shouldPrintBanner([]string{"picoclaw", cobra.ShellCompNoDescRequestCmd}, true))
	})

	t.Run("env disables banner", func(t *testing.T) {
		t.Setenv(noBannerEnv, "1")
		assert.False(t, shouldPrintBanner([]string{"picoclaw", "agent"}, true))
	})

	t.Run("truthy env values disable banner after normalization", func(t *testing.T) {
		for _, value := range []string{"true", "yes", "on", " TrUe ", "\tON\n"} {
			t.Run(fmt.Sprintf("%q", value), func(t *testing.T) {
				t.Setenv(noBannerEnv, value)
				assert.False(t, shouldPrintBanner([]string{"picoclaw", "agent"}, true))
			})
		}
	})

	t.Run("root command still prints banner in terminal", func(t *testing.T) {
		t.Setenv(noBannerEnv, "")
		assert.True(t, shouldPrintBanner([]string{"picoclaw"}, true))
	})

	t.Run("unknown subcommand still prints banner in terminal", func(t *testing.T) {
		t.Setenv(noBannerEnv, "")
		assert.True(t, shouldPrintBanner([]string{"picoclaw", "unknown"}, true))
	})
}
