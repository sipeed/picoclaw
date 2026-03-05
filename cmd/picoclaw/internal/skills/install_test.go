package skills

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInstallSubcommand(t *testing.T) {
	cmd := newInstallCommand(nil)

	require.NotNil(t, cmd)

	assert.Equal(t, "install", cmd.Use)
	assert.Equal(t, "Install skill from GitHub", cmd.Short)

	assert.Nil(t, cmd.Run)
	assert.NotNil(t, cmd.RunE)

	assert.True(t, cmd.HasExample())
	assert.False(t, cmd.HasSubCommands())

	assert.True(t, cmd.HasFlags())
	assert.NotNil(t, cmd.Flags().Lookup("registry"))

	assert.Len(t, cmd.Aliases, 0)
}

func TestInstallCommandArgs(t *testing.T) {
	tests := []struct {
		name        string
		registry    string
		args        []string
		wantErr     bool
		errContains string
	}{
		{
			name:     "no registry with one arg passes",
			registry: "",
			args:     []string{"sioeed/picoclaw-skills/weather"},
			wantErr:  false,
		},
		{
			name:     "no registry with no args fails",
			registry: "",
			args:     []string{},
			wantErr:  true,
			errContains: "exactly 1 argument",
		},
		{
			name:     "with registry one arg passes",
			registry: "clawhub",
			args:     []string{"github"},
			wantErr:  false,
		},
		{
			name:        "with registry zero args fails",
			registry:    "clawhub",
			args:        []string{},
			wantErr:     true,
			errContains: "exactly 1 argument",
		},
		{
			name:        "with registry two args fails",
			registry:    "clawhub",
			args:        []string{"arg1", "arg2"},
			wantErr:     true,
			errContains: "exactly 1 argument",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newInstallCommand(nil)
			// Set the registry flag
			err := cmd.Flags().Set("registry", tt.registry)
			require.NoError(t, err)

			err = cmd.Args(cmd, tt.args)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
