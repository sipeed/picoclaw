package gateway

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGatewayCommand(t *testing.T) {
	cmd := NewGatewayCommand()

	require.NotNil(t, cmd)

	assert.Equal(t, "gateway", cmd.Use)
	assert.Equal(t, "Manage picoclaw gateway daemon", cmd.Short)

	assert.Len(t, cmd.Aliases, 1)
	assert.True(t, cmd.HasAlias("g"))

	assert.Nil(t, cmd.Run)
	assert.NotNil(t, cmd.RunE)

	assert.Nil(t, cmd.PersistentPreRun)
	assert.Nil(t, cmd.PersistentPostRun)

	// Should now have subcommands
	assert.True(t, cmd.HasSubCommands())

	// Check for daemon subcommands
	assert.True(t, cmd.HasSubCommands())
	subcommands := cmd.Commands()
	subcommandUses := make([]string, len(subcommands))
	for i, sub := range subcommands {
		subcommandUses[i] = sub.Use
	}

	assert.Contains(t, subcommandUses, "start")
	assert.Contains(t, subcommandUses, "stop")
	assert.Contains(t, subcommandUses, "restart")
	assert.Contains(t, subcommandUses, "status")

	assert.True(t, cmd.HasFlags())
	assert.NotNil(t, cmd.Flags().Lookup("debug"))
	assert.NotNil(t, cmd.Flags().Lookup("run-daemon"))
}
