package main

import (
	"bytes"
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/config"
)

// TestRunCommandFatalPrintsToStdout verifies that when a subcommand fails,
// runCommand prints a prominent error to the provided writer (stdout in production),
// so the error is visible in environments where stderr is redirected (e.g. docker).
func TestRunCommandFatalPrintsToStdout(t *testing.T) {
	var buf bytes.Buffer
	// Pass an unknown flag to trigger a cobra error without any real I/O.
	err := runCommand(&buf, []string{"gateway", "--no-such-flag"})
	require.Error(t, err)
	out := buf.String()
	assert.Contains(t, out, "FATAL", "fatal error must be printed to stdout")
	assert.Contains(t, out, err.Error(), "error message must appear in stdout output")
}

func TestNewPicoclawCommand(t *testing.T) {
	cmd := NewPicoclawCommand()

	require.NotNil(t, cmd)

	short := fmt.Sprintf("%s picoclaw - Personal AI Assistant %s\n\n", internal.Logo, config.GetVersion())

	assert.Equal(t, "picoclaw", cmd.Use)
	assert.Equal(t, short, cmd.Short)

	assert.True(t, cmd.HasSubCommands())
	assert.True(t, cmd.HasAvailableSubCommands())

	assert.False(t, cmd.HasFlags())

	assert.Nil(t, cmd.Run)
	assert.Nil(t, cmd.RunE)

	assert.Nil(t, cmd.PersistentPreRun)
	assert.Nil(t, cmd.PersistentPostRun)

	allowedCommands := []string{
		"agent",
		"auth",
		"cron",
		"gateway",
		"migrate",
		"model",
		"onboard",
		"skills",
		"status",
		"update",
		"version",
	}

	subcommands := cmd.Commands()
	assert.Len(t, subcommands, len(allowedCommands))

	for _, subcmd := range subcommands {
		found := slices.Contains(allowedCommands, subcmd.Name())
		assert.True(t, found, "unexpected subcommand %q", subcmd.Name())

		assert.False(t, subcmd.Hidden)
	}
}
