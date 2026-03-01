package plugin

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal/pluginruntime"
)

func TestNewListSubcommand(t *testing.T) {
	cmd := newListCommand()

	require.NotNil(t, cmd)

	assert.Equal(t, "list", cmd.Use)
	assert.Equal(t, "List configured plugin status", cmd.Short)

	assert.Nil(t, cmd.Run)
	assert.NotNil(t, cmd.RunE)

	assert.False(t, cmd.HasSubCommands())
	assert.True(t, cmd.HasFlags())

	assert.Len(t, cmd.Aliases, 0)

	formatFlag := cmd.Flags().Lookup("format")
	require.NotNil(t, formatFlag)
	assert.Equal(t, formatText, formatFlag.DefValue)
}

func TestNewListSubcommand_RejectsUnknownFormat(t *testing.T) {
	cmd := newListCommand()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--format", "yaml"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.ErrorContains(t, err, `invalid value for --format: "yaml"`)
}

func TestBuildPluginStatuses_DeterministicOrder(t *testing.T) {
	summary := pluginruntime.Summary{
		Enabled:         []string{"beta"},
		Disabled:        []string{"alpha"},
		UnknownEnabled:  []string{"zeta"},
		UnknownDisabled: []string{"eta"},
	}

	got := buildPluginStatuses(summary)

	assert.Equal(t, []pluginStatus{
		{Name: "alpha", Status: "disabled"},
		{Name: "beta", Status: "enabled"},
		{Name: "eta", Status: "unknown-disabled"},
		{Name: "zeta", Status: "unknown-enabled"},
	}, got)
}
