package migrate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMigrateCommand(t *testing.T) {
	cmd := NewMigrateCommand()

	require.NotNil(t, cmd)

	assert.Equal(t, "migrate", cmd.Use)
	assert.Equal(t, "Migrate from OpenClaw to PicoClaw", cmd.Short)

	assert.Len(t, cmd.Aliases, 0)

	assert.True(t, cmd.HasExample())
	assert.False(t, cmd.HasSubCommands())

	assert.Nil(t, cmd.Run)
	assert.NotNil(t, cmd.RunE)

	assert.Nil(t, cmd.PersistentPreRun)
	assert.Nil(t, cmd.PersistentPostRun)

	assert.True(t, cmd.HasFlags())

	assert.NotNil(t, cmd.Flags().Lookup("dry-run"))
	assert.NotNil(t, cmd.Flags().Lookup("refresh"))
	assert.NotNil(t, cmd.Flags().Lookup("config-only"))
	assert.NotNil(t, cmd.Flags().Lookup("workspace-only"))
	assert.NotNil(t, cmd.Flags().Lookup("force"))
	assert.NotNil(t, cmd.Flags().Lookup("openclaw-home"))
	assert.NotNil(t, cmd.Flags().Lookup("picoclaw-home"))
}
