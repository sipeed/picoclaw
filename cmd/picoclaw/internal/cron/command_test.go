package cron

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCronCommand(t *testing.T) {
	cmd := NewCronCommand()

	require.NotNil(t, cmd)

	assert.Equal(t, "Manage scheduled tasks", cmd.Short)

	assert.Len(t, cmd.Aliases, 1)
	assert.True(t, cmd.HasAlias("c"))

	assert.False(t, cmd.HasFlags())

	assert.Nil(t, cmd.Run)
	assert.NotNil(t, cmd.RunE)

	assert.NotNil(t, cmd.PersistentPreRunE)
	assert.Nil(t, cmd.PersistentPreRun)
	assert.Nil(t, cmd.PersistentPostRun)

	assert.True(t, cmd.HasSubCommands())

	allowedCommands := []string{
		"list",
		"add",
		"remove",
		"enable",
		"disable",
	}

	subcommands := cmd.Commands()
	assert.Len(t, subcommands, len(allowedCommands))

	for _, subcmd := range subcommands {
		found := slices.Contains(allowedCommands, subcmd.Name())
		assert.True(t, found, "unexpected subcommand %q", subcmd.Name())

		assert.Len(t, subcmd.Aliases, 0)
		assert.False(t, subcmd.Hidden)

		assert.False(t, subcmd.HasSubCommands())

		assert.Nil(t, subcmd.Run)
		assert.NotNil(t, subcmd.RunE)

		assert.Nil(t, subcmd.PersistentPreRun)
		assert.Nil(t, subcmd.PersistentPostRun)
	}
}
