package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewVersionCommand(t *testing.T) {
	cmd := NewVersionCommand()

	require.NotNil(t, cmd)

	assert.Equal(t, "version", cmd.Use)

	assert.Len(t, cmd.Aliases, 1)
	assert.True(t, cmd.HasAlias("v"))

	assert.False(t, cmd.HasFlags())

	assert.Equal(t, "Show version information", cmd.Short)

	assert.False(t, cmd.HasSubCommands())

	assert.NotNil(t, cmd.Run)
	assert.Nil(t, cmd.RunE)

	assert.Nil(t, cmd.PersistentPreRun)
	assert.Nil(t, cmd.PersistentPostRun)
}
