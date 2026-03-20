package main

import (
	"fmt"
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/config"
)

func TestNewPicoclawCommand(t *testing.T) {
	cmd := NewPicoclawCommand()

	require.NotNil(t, cmd)

	short := fmt.Sprintf("%s picoclaw - Personal AI Assistant v%s\n\n", internal.Logo, config.GetVersion())

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

func TestShouldShowBanner(t *testing.T) {
	// Save original args to restore later
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	tests := []struct {
		name     string
		args     []string
		envVar   string
		expected bool
	}{
		{
			name:     "default shows banner",
			args:     []string{"picoclaw"},
			envVar:   "",
			expected: true,
		},
		{
			name:     "normal command shows banner",
			args:     []string{"picoclaw", "version"},
			envVar:   "",
			expected: true,
		},
		{
			name:     "__complete suppresses banner",
			args:     []string{"picoclaw", "__complete", "ver"},
			envVar:   "",
			expected: false,
		},
		{
			name:     "__completeNoDesc suppresses banner",
			args:     []string{"picoclaw", "__completeNoDesc", "ver"},
			envVar:   "",
			expected: false,
		},
		{
			name:     "completion command suppresses banner",
			args:     []string{"picoclaw", "completion", "bash"},
			envVar:   "",
			expected: false,
		},
		{
			name:     "PICOCLAW_NO_BANNER=1 suppresses banner",
			args:     []string{"picoclaw", "version"},
			envVar:   "1",
			expected: false,
		},
		{
			name:     "PICOCLAW_NO_BANNER=true suppresses banner",
			args:     []string{"picoclaw", "version"},
			envVar:   "true",
			expected: false,
		},
		{
			name:     "PICOCLAW_NO_BANNER=TRUE suppresses banner",
			args:     []string{"picoclaw", "version"},
			envVar:   "TRUE",
			expected: false,
		},
		{
			name:     "PICOCLAW_NO_BANNER=0 does not suppress banner",
			args:     []string{"picoclaw", "version"},
			envVar:   "0",
			expected: true,
		},
		{
			name:     "PICOCLAW_NO_BANNER=false does not suppress banner",
			args:     []string{"picoclaw", "version"},
			envVar:   "false",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Args = tt.args

			if tt.envVar != "" {
				t.Setenv("PICOCLAW_NO_BANNER", tt.envVar)
			} else {
				// Clear env var if not set in test
				os.Unsetenv("PICOCLAW_NO_BANNER")
			}

			result := shouldShowBanner()
			assert.Equal(t, tt.expected, result)
		})
	}
}
