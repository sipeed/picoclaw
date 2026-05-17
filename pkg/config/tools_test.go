package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsToolEnabled_LoadImage(t *testing.T) {
	t.Parallel()

	t.Run("default enables load_image", func(t *testing.T) {
		cfg := DefaultConfig()
		assert.True(t, cfg.Tools.IsToolEnabled("load_image"),
			"load_image should be enabled by default")
	})

	t.Run("explicitly disabling load_image works", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Tools.LoadImage.Enabled = false
		assert.False(t, cfg.Tools.IsToolEnabled("load_image"),
			"load_image should be disabled when explicitly set to false")
	})

	t.Run("explicitly enabling load_image works", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Tools.LoadImage.Enabled = true
		assert.True(t, cfg.Tools.IsToolEnabled("load_image"),
			"load_image should be enabled when explicitly set to true")
	})
}

func TestIsToolEnabled_Reaction(t *testing.T) {
	t.Parallel()

	t.Run("default enables reaction", func(t *testing.T) {
		cfg := DefaultConfig()
		assert.True(t, cfg.Tools.IsToolEnabled("reaction"),
			"reaction should be enabled by default")
	})

	t.Run("explicitly disabling reaction works", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Tools.Reaction.Enabled = false
		assert.False(t, cfg.Tools.IsToolEnabled("reaction"),
			"reaction should be disabled when explicitly set to false")
	})

	t.Run("explicitly enabling reaction works", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Tools.Reaction.Enabled = true
		assert.True(t, cfg.Tools.IsToolEnabled("reaction"),
			"reaction should be enabled when explicitly set to true")
	})
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
