package gateway

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGatewayCommand(t *testing.T) {
	cmd := NewGatewayCommand()

	require.NotNil(t, cmd)

	assert.Equal(t, "gateway", cmd.Use)
	assert.Equal(t, "Start picoclaw gateway", cmd.Short)

	assert.Len(t, cmd.Aliases, 1)
	assert.True(t, cmd.HasAlias("g"))

	assert.Nil(t, cmd.Run)
	assert.NotNil(t, cmd.RunE)

	assert.Nil(t, cmd.PersistentPreRun)
	assert.Nil(t, cmd.PersistentPostRun)

	// Gateway command now has subcommands: start, stop, status
	assert.True(t, cmd.HasSubCommands())
	assert.Len(t, cmd.Commands(), 3)

	assert.True(t, cmd.HasFlags())
	assert.NotNil(t, cmd.Flags().Lookup("debug"))
}

func TestGatewayStartCommand(t *testing.T) {
	cmd := NewGatewayCommand()

	startCmd, _, err := cmd.Find([]string{"start"})
	require.NoError(t, err)
	require.NotNil(t, startCmd)

	assert.Equal(t, "start", startCmd.Use)
	assert.Equal(t, "Start picoclaw gateway in background", startCmd.Short)
	assert.NotNil(t, startCmd.RunE)
}

func TestGatewayStopCommand(t *testing.T) {
	cmd := NewGatewayCommand()

	stopCmd, _, err := cmd.Find([]string{"stop"})
	require.NoError(t, err)
	require.NotNil(t, stopCmd)

	assert.Equal(t, "stop", stopCmd.Use)
	assert.Equal(t, "Stop picoclaw gateway background service", stopCmd.Short)
	assert.NotNil(t, stopCmd.RunE)
}

func TestGatewayStatusCommand(t *testing.T) {
	cmd := NewGatewayCommand()

	statusCmd, _, err := cmd.Find([]string{"status"})
	require.NoError(t, err)
	require.NotNil(t, statusCmd)

	assert.Equal(t, "status", statusCmd.Use)
	assert.Equal(t, "Show picoclaw gateway status", statusCmd.Short)
	assert.NotNil(t, statusCmd.Run)
}
