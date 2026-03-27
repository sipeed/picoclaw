package channel

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStartCommand(t *testing.T) {
	cmd := newStartCommand()

	require.NotNil(t, cmd)

	assert.Equal(t, "start", cmd.Use)
	assert.Equal(t, "Start channels without gateway side services", cmd.Short)
	assert.NotNil(t, cmd.RunE)
	assert.NotNil(t, cmd.PreRunE)

	assert.NotNil(t, cmd.Flags().Lookup("debug"))
	assert.NotNil(t, cmd.Flags().Lookup("allow-empty"))
	assert.NotNil(t, cmd.Flags().Lookup("no-truncate"))
}

func TestStartCommandPreRunE_NoTruncateRequiresDebug(t *testing.T) {
	cmd := newStartCommand()

	err := cmd.ParseFlags([]string{"--no-truncate"})
	require.NoError(t, err)

	err = cmd.PreRunE(cmd, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--no-truncate")
}

func TestStartCommandPreRunE_NoTruncateWithDebug(t *testing.T) {
	cmd := newStartCommand()

	err := cmd.ParseFlags([]string{"--debug", "--no-truncate"})
	require.NoError(t, err)

	err = cmd.PreRunE(cmd, nil)
	assert.NoError(t, err)
}
