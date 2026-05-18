package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsToolEnabled_ImageAndReaction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		tool    string
		enabled bool
	}{
		{"load_image enabled by default", "load_image", true},
		{"load_image disabled", "load_image", false},
		{"load_image re-enabled", "load_image", true},
		{"reaction enabled by default", "reaction", true},
		{"reaction disabled", "reaction", false},
		{"reaction re-enabled", "reaction", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := DefaultConfig()
			if tt.enabled {
				assert.True(t, cfg.Tools.IsToolEnabled(tt.tool), "%s should be enabled", tt.tool)
			} else {
				assert.False(t, cfg.Tools.IsToolEnabled(tt.tool), "%s should be disabled", tt.tool)
			}
		})
	}
}

func TestIsToolEnabled_EnabledByDefault(t *testing.T) {
	t.Parallel()

	// Tools that are enabled by default (Enabled: true in defaults.go)
	enabledByDefault := []string{
		"web", "cron", "exec", "skills", "media_cleanup",
		"append_file", "edit_file", "find_skills", "install_skill",
		"list_dir", "message", "read_file",
		"spawn", "subagent",
		"web_fetch", "send_file", "write_file",
		"load_image", "reaction",
	}

	cfg := DefaultConfig()
	for _, tool := range enabledByDefault {
		t.Run(tool+" enabled by default", func(t *testing.T) {
			t.Parallel()
			assert.True(t, cfg.Tools.IsToolEnabled(tool),
				"%s should be enabled by default", tool)
		})
	}
}

func TestIsToolEnabled_DisabledByDefault(t *testing.T) {
	t.Parallel()

	// Tools that are disabled by default (hardware tools, send_tts, spawn_status)
	disabledByDefault := []string{
		"i2c", "serial", "spi", "send_tts", "spawn_status",
	}

	cfg := DefaultConfig()
	for _, tool := range disabledByDefault {
		t.Run(tool+" disabled by default", func(t *testing.T) {
			t.Parallel()
			assert.False(t, cfg.Tools.IsToolEnabled(tool),
				"%s should be disabled by default", tool)
		})
	}
}

func TestIsToolEnabled_CanDisableAndReEnable(t *testing.T) {
	t.Parallel()

	// Verify that default-enabled tools can be disabled and re-enabled
	cfg := DefaultConfig()

	// load_image: disable then re-enable
	cfg.Tools.LoadImage.Enabled = false
	assert.False(t, cfg.Tools.IsToolEnabled("load_image"))
	cfg.Tools.LoadImage.Enabled = true
	assert.True(t, cfg.Tools.IsToolEnabled("load_image"))

	// reaction: disable then re-enable
	cfg.Tools.Reaction.Enabled = false
	assert.False(t, cfg.Tools.IsToolEnabled("reaction"))
	cfg.Tools.Reaction.Enabled = true
	assert.True(t, cfg.Tools.IsToolEnabled("reaction"))
}

func TestIsToolEnabled_UnknownToolDefaultTrue(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()
	assert.True(t, cfg.Tools.IsToolEnabled("nonexistent_tool_xyz"),
		"unknown tools should default to enabled")
}
