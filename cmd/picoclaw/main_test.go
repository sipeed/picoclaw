package main

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/security"
)

func TestNewPicoclawCommand(t *testing.T) {
	cmd := NewPicoclawCommand()

	require.NotNil(t, cmd)

	short := fmt.Sprintf("%s PicoClaw — personal AI assistant", internal.Logo)
	longHas := strings.Contains(cmd.Long, config.FormatVersion())

	assert.Equal(t, "picoclaw", cmd.Use)
	assert.Equal(t, short, cmd.Short)
	assert.True(t, longHas)

	assert.True(t, cmd.HasSubCommands())
	assert.True(t, cmd.HasAvailableSubCommands())

	assert.True(t, cmd.PersistentFlags().Lookup("no-color") != nil)

	assert.Nil(t, cmd.Run)
	assert.Nil(t, cmd.RunE)

	assert.NotNil(t, cmd.PersistentPreRun)
	assert.Nil(t, cmd.PersistentPostRun)

	allowedCommands := []string{
		"agent",
		"auth",
		"cron",
		"freeride",
		"gateway",
		"migrate",
		"model",
		"onboard",
		"skills",
		"status",
		"update",
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

func TestSecurityShieldRegistration_Regression(t *testing.T) {
	// Initialize security hooks (registers them in pkg/agent)
	security.Init()

	expectedHooks := []string{
		"security_policy",
		"security_canary",
		"security_behavior",
		"security_pii",
		"security_ipia",
	}

	for _, name := range expectedHooks {
		t.Run(name, func(t *testing.T) {
			assert.True(t, agent.IsBuiltinHookRegistered(name),
				"Builtin hook %q is not registered. This usually means pkg/security was deleted "+
					"or security.Init() is no longer called in main().", name)
		})
	}
}
