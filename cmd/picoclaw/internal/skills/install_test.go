package skills

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInstallSubcommand(t *testing.T) {
	cmd := newInstallCommand(nil)

	require.NotNil(t, cmd)

	assert.Equal(t, "install", cmd.Use)
	assert.Equal(t, "Install skill from GitHub", cmd.Short)

	assert.Nil(t, cmd.Run)
	assert.NotNil(t, cmd.RunE)

	assert.True(t, cmd.HasExample())
	assert.False(t, cmd.HasSubCommands())

	assert.True(t, cmd.HasFlags())
	assert.NotNil(t, cmd.Flags().Lookup("registry"))

	assert.Len(t, cmd.Aliases, 0)
}
