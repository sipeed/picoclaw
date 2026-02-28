package auth

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLoginSubCommand(t *testing.T) {
	cmd := newLoginCommand()

	require.NotNil(t, cmd)

	assert.Equal(t, "Login via OAuth or paste token", cmd.Short)

	assert.True(t, cmd.HasFlags())

	assert.NotNil(t, cmd.Flags().Lookup("device-code"))

	providerFlag := cmd.Flags().Lookup("provider")
	require.NotNil(t, providerFlag)

	val, found := providerFlag.Annotations[cobra.BashCompOneRequiredFlag]
	require.True(t, found)
	require.NotEmpty(t, val)
	assert.Equal(t, "true", val[0])
}
