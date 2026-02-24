package skills

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInstallbuiltinSubcommand(t *testing.T) {
	cmd := newInstallBuiltinCommand(nil)

	require.NotNil(t, cmd)

	assert.Equal(t, "install-builtin", cmd.Use)
	assert.Equal(t, "Install all builtin skills to workspace", cmd.Short)

	assert.Nil(t, cmd.Run)
	assert.NotNil(t, cmd.RunE)

	assert.True(t, cmd.HasExample())
	assert.False(t, cmd.HasSubCommands())

	assert.False(t, cmd.HasFlags())

	assert.Len(t, cmd.Aliases, 0)
}
