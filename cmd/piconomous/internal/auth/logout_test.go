package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogoutSubcommand(t *testing.T) {
	cmd := newLogoutCommand()

	require.NotNil(t, cmd)

	assert.Equal(t, "Remove stored credentials", cmd.Short)

	assert.True(t, cmd.HasFlags())

	assert.NotNil(t, cmd.Flags().Lookup("provider"))
}
