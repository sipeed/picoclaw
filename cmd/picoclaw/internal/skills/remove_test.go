package skills

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRemoveSubcommand(t *testing.T) {
	cmd := newRemoveCommand(nil)

	require.NotNil(t, cmd)

	assert.Equal(t, "remove", cmd.Use)
	assert.Equal(t, "Remove installed skill", cmd.Short)

	assert.Nil(t, cmd.Run)
	assert.NotNil(t, cmd.RunE)

	assert.True(t, cmd.HasExample())
	assert.False(t, cmd.HasSubCommands())

	assert.False(t, cmd.HasFlags())

	assert.Len(t, cmd.Aliases, 2)
	assert.True(t, cmd.HasAlias("rm"))
	assert.True(t, cmd.HasAlias("uninstall"))
}
