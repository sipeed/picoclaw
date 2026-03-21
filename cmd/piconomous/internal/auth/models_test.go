package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewModelsCommand(t *testing.T) {
	cmd := newModelsCommand()

	require.NotNil(t, cmd)

	assert.Equal(t, "models", cmd.Use)
	assert.Equal(t, "Show available models", cmd.Short)

	assert.False(t, cmd.HasFlags())
}
