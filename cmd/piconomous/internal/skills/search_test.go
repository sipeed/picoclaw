package skills

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSearchSubcommand(t *testing.T) {
	cmd := newSearchCommand()

	require.NotNil(t, cmd)

	assert.Equal(t, "search [query]", cmd.Use)
	assert.Equal(t, "Search available skills", cmd.Short)

	assert.Nil(t, cmd.Run)
	assert.NotNil(t, cmd.RunE)

	assert.False(t, cmd.HasSubCommands())
	assert.False(t, cmd.HasFlags())

	assert.Len(t, cmd.Aliases, 0)
}
