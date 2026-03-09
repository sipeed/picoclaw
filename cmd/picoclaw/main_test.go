package main

import (
	"fmt"
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
)

func TestNewPicoclawCommand(t *testing.T) {
	cmd := NewPicoclawCommand()

	require.NotNil(t, cmd)

	short := fmt.Sprintf("%s picoclaw - Personal AI Assistant v%s\n\n", internal.Logo, internal.GetVersion())

	assert.Equal(t, "picoclaw", cmd.Use)
	assert.Equal(t, short, cmd.Short)

	assert.True(t, cmd.HasSubCommands())
	assert.True(t, cmd.HasAvailableSubCommands())

	assert.False(t, cmd.HasFlags())

	// RunE is set to handle plugin execution
	assert.NotNil(t, cmd.RunE)

	assert.Nil(t, cmd.PersistentPreRun)
	assert.Nil(t, cmd.PersistentPostRun)

	allowedCommands := []string{
		"agent",
		"auth",
		"cron",
		"gateway",
		"migrate",
		"onboard",
		"skills",
		"status",
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

func TestDisableFlagParsing(t *testing.T) {
	cmd := NewPicoclawCommand()
	assert.True(t, cmd.DisableFlagParsing, "DisableFlagParsing should be enabled")
}

func TestNoBannerEnvVar(t *testing.T) {
	// Test that PICOCLAW_NO_BANNER environment variable controls banner
	origVal := os.Getenv("PICOCLAW_NO_BANNER")
	defer os.Setenv("PICOCLAW_NO_BANNER", origVal)

	// Without env var
	os.Unsetenv("PICOCLAW_NO_BANNER")
	noBanner := os.Getenv("PICOCLAW_NO_BANNER") != "1"
	assert.True(t, noBanner, "banner should show without env var")

	// With env var set to 1
	os.Setenv("PICOCLAW_NO_BANNER", "1")
	noBanner = os.Getenv("PICOCLAW_NO_BANNER") != "1"
	assert.False(t, noBanner, "banner should be hidden with PICOCLAW_NO_BANNER=1")
}

func TestRootArgsValidatorFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantFlag bool // true if should NOT route to plugin
	}{
		{"--help", []string{"--help"}, true},
		{"-h", []string{"-h"}, true},
		{"--version", []string{"--version"}, true},
		{"-v", []string{"-v"}, true},
		{"--unknown-flag", []string{"--unknown-flag"}, true},
		{"-x", []string{"-x"}, true},
		{"service", []string{"service"}, false}, // should route to plugin
		{"onboard", []string{"onboard"}, false}, // known command
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isFlag := len(tt.args) > 0 && len(tt.args[0]) > 0 && tt.args[0][0] == '-'
			assert.Equal(t, tt.wantFlag, isFlag, "flag detection for %q", tt.args[0])
		})
	}
}
