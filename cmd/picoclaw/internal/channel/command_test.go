package channel

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewChannelCommand(t *testing.T) {
	cmd := NewChannelCommand()

	require.NotNil(t, cmd)

	assert.Equal(t, "channel", cmd.Use)
	assert.Equal(t, "Manage channel runtimes", cmd.Short)

	assert.Len(t, cmd.Aliases, 1)
	assert.True(t, cmd.HasAlias("ch"))

	assert.Nil(t, cmd.Run)
	assert.NotNil(t, cmd.RunE)

	assert.True(t, cmd.HasSubCommands())
	startCmd, _, err := cmd.Find([]string{"start"})
	require.NoError(t, err)
	require.NotNil(t, startCmd)
	assert.Equal(t, "start", startCmd.Name())
}
