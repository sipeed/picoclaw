package skills

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewShowSubcommand(t *testing.T) {
	cmd := newShowCommand(nil)

	require.NotNil(t, cmd)

	assert.Equal(t, "show", cmd.Use)
	assert.Equal(t, "Show skill details", cmd.Short)

	assert.Nil(t, cmd.Run)
	assert.NotNil(t, cmd.RunE)

	assert.True(t, cmd.HasExample())
	assert.False(t, cmd.HasSubCommands())

	assert.False(t, cmd.HasFlags())

	assert.Len(t, cmd.Aliases, 0)
}
