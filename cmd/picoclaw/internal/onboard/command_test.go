package onboard

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOnboardCommand(t *testing.T) {
	cmd := NewOnboardCommand()

	require.NotNil(t, cmd)

	assert.Equal(t, "onboard", cmd.Use)
	assert.Equal(t, "Initialize picoclaw configuration and workspace", cmd.Short)

	assert.Len(t, cmd.Aliases, 1)
	assert.True(t, cmd.HasAlias("o"))

	assert.NotNil(t, cmd.Run)
	assert.Nil(t, cmd.RunE)

	assert.Nil(t, cmd.PersistentPreRun)
	assert.Nil(t, cmd.PersistentPostRun)

	assert.True(t, cmd.HasFlags())
	encFlag := cmd.Flags().Lookup("enc")
	require.NotNil(t, encFlag, "expected --enc flag to be registered")
	assert.Equal(t, "false", encFlag.DefValue, "--enc should default to false")
	yesFlag := cmd.Flags().Lookup("yes")
	require.NotNil(t, yesFlag, "expected --yes flag to be registered")
	assert.Equal(t, "false", yesFlag.DefValue, "--yes should default to false")
	assert.True(t, cmd.HasSubCommands())
	assert.Len(t, cmd.Commands(), 1)
	assert.Equal(t, "purge", cmd.Commands()[0].Name())
}
