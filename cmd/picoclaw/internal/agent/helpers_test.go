package agent

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustAbs(t *testing.T, path string) string {
	t.Helper()
	abs, err := filepath.Abs(path)
	require.NoError(t, err)
	return abs
}

func TestValidateWorkspacePaths(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspaces")

	absRoot, err := filepath.Abs(root)
	require.NoError(t, err)

	tests := []struct {
		name      string
		root      string
		workspace string
		configDir string
		wantWS    string // expected resolved workspace; empty if error expected
		wantCD    string // expected resolved configDir; empty if error expected
		wantErr   string // substring of expected error; empty means success
	}{
		{
			name:      "valid workspace subdirectory",
			root:      root,
			workspace: "tenant1",
			wantWS:    filepath.Join(absRoot, "tenant1"),
		},
		{
			name:      "valid config-dir subdirectory",
			root:      root,
			configDir: "tenant1/config",
			wantCD:    filepath.Join(absRoot, "tenant1", "config"),
		},
		{
			name:      "both valid",
			root:      root,
			workspace: "tenant1",
			configDir: "tenant1/config",
			wantWS:    filepath.Join(absRoot, "tenant1"),
			wantCD:    filepath.Join(absRoot, "tenant1", "config"),
		},
		{
			name:      "empty flags are fine",
			root:      root,
			workspace: "",
			configDir: "",
		},

		// --- workspace traversal ---
		{
			name:      "workspace traversal rejected",
			root:      root,
			workspace: "../escape",
			wantErr:   "invalid --workspace",
		},
		{
			name:      "workspace bare dotdot rejected",
			root:      root,
			workspace: "..",
			wantErr:   "invalid --workspace",
		},
		{
			name:      "workspace mid-path traversal rejected",
			root:      root,
			workspace: "a/../../../etc",
			wantErr:   "invalid --workspace",
		},

		// --- config-dir traversal ---
		{
			name:      "config-dir traversal rejected",
			root:      root,
			configDir: "../escape",
			wantErr:   "invalid --config-dir",
		},
		{
			name:      "config-dir absolute path rejected",
			root:      root,
			configDir: "/etc/passwd",
			wantErr:   "invalid --config-dir",
		},

		// --- no workspace_root configured (backward compat: falls back to filepath.Abs) ---
		{
			name:      "workspace without root uses Abs fallback",
			root:      "",
			workspace: "anything",
			wantWS:    mustAbs(t, "anything"),
		},
		{
			name:      "config-dir without root uses Abs fallback",
			root:      "",
			configDir: "anything",
			wantCD:    mustAbs(t, "anything"),
		},
		{
			name:      "workspace traversal without root still rejected",
			root:      "",
			workspace: "../escape",
			wantErr:   "invalid --workspace",
		},

		// --- workspace resolves to root itself ---
		{
			name:      "workspace dot rejected",
			root:      root,
			workspace: ".",
			wantErr:   "invalid --workspace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws, cd, err := validateWorkspacePaths(tt.root, tt.workspace, tt.configDir)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.True(t, strings.Contains(err.Error(), tt.wantErr),
					"error %q should contain %q", err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantWS, ws)
			assert.Equal(t, tt.wantCD, cd)
		})
	}
}
