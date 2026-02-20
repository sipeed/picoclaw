package skills

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewListSubcommand(t *testing.T) {
	cmd := newListCommand(nil)

	require.NotNil(t, cmd)

	assert.Equal(t, "list", cmd.Use)
	assert.Equal(t, "List installed skills", cmd.Short)

	assert.NotNil(t, cmd.Run)

	assert.True(t, cmd.HasExample())
	assert.False(t, cmd.HasSubCommands())

	assert.False(t, cmd.HasFlags())

	assert.Len(t, cmd.Aliases, 0)
}
