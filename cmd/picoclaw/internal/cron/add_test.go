package cron

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAddSubcommand(t *testing.T) {
	fn := func() string { return "" }
	cmd := newAddCommand(fn)

	require.NotNil(t, cmd)

	assert.Equal(t, "add", cmd.Use)
	assert.Equal(t, "Add a new scheduled job", cmd.Short)

	assert.True(t, cmd.HasFlags())

	assert.NotNil(t, cmd.Flags().Lookup("every"))
	assert.NotNil(t, cmd.Flags().Lookup("cron"))
	assert.NotNil(t, cmd.Flags().Lookup("deliver"))
	assert.NotNil(t, cmd.Flags().Lookup("to"))
	assert.NotNil(t, cmd.Flags().Lookup("channel"))

	nameFlag := cmd.Flags().Lookup("name")
	require.NotNil(t, nameFlag)

	messageFlag := cmd.Flags().Lookup("message")
	require.NotNil(t, messageFlag)

	val, found := nameFlag.Annotations[cobra.BashCompOneRequiredFlag]
	require.True(t, found)
	require.NotEmpty(t, val)
	assert.Equal(t, "true", val[0])

	val, found = messageFlag.Annotations[cobra.BashCompOneRequiredFlag]
	require.True(t, found)
	require.NotEmpty(t, val)
	assert.Equal(t, "true", val[0])
}

func TestNewAddCommandEveryAndCronMutuallyExclusive(t *testing.T) {
	cmd := newAddCommand(func() string { return "testing" })

	cmd.SetArgs([]string{
		"--name", "job",
		"--message", "hello",
		"--every", "10",
		"--cron", "0 9 * * *",
	})

	err := cmd.Execute()
	require.Error(t, err)
}
