package plugins

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestGetPluginsDir(t *testing.T) {
	dir := getPluginsDir()
	expected := filepath.Join(os.Getenv("HOME"), ".picoclaw", "plugins")
	if dir != expected {
		t.Errorf("expected %s, got %s", expected, dir)
	}
}

func TestListPlugins(t *testing.T) {
	// Test when plugins directory doesn't exist
	// We can't easily test this since ~/.picoclaw/plugins may already exist
	// Just verify it doesn't error
	plugins, err := listPlugins()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// plugins may be nil or empty or contain existing plugins
	_ = plugins
}

func TestFindPlugin(t *testing.T) {
	// Test with non-existent plugin
	_, err := findPlugin("nonexistent-plugin-12345")
	if err == nil {
		t.Error("expected error for non-existent plugin")
	}
}

func TestNewPluginsCommand(t *testing.T) {
	cmd := NewPluginsCommand()
	if cmd == nil {
		t.Fatal("NewPluginsCommand returned nil")
	}

	if cmd.Use != "plugins" {
		t.Errorf("expected use 'plugins', got %s", cmd.Use)
	}

	// Check subcommands
	subcommands := cmd.Commands()
	if len(subcommands) != 1 {
		t.Errorf("expected 1 subcommand, got %d", len(subcommands))
	}

	// Find list command
	var listCmd *cobra.Command
	for _, sub := range subcommands {
		if sub.Use == "list" {
			listCmd = sub
			break
		}
	}
	if listCmd == nil {
		t.Error("list subcommand not found")
	}
}

func TestRunPluginNotFound(t *testing.T) {
	_, err := findPlugin("nonexistent-plugin-xyz")
	if err == nil {
		t.Error("expected error for non-existent plugin")
	}
}

func TestListPluginsIntegration(t *testing.T) {
	// This is an integration test that checks the actual plugins directory
	pluginsDir := getPluginsDir()

	// Create a temporary test plugin
	tmpDir := pluginsDir
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		// Directory creation failed, skip integration test
		t.Skip("cannot create plugins directory")
	}

	// Create a temporary test script
	testPlugin := filepath.Join(tmpDir, "test-plugin-temp")
	if err := os.WriteFile(testPlugin, []byte("#!/bin/bash\necho hello"), 0o755); err != nil {
		t.Skip("cannot write test plugin")
	}
	defer os.Remove(testPlugin)

	// Test listing
	plugins, err := listPlugins()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	found := false
	for _, p := range plugins {
		if p == "test-plugin-temp" {
			found = true
			break
		}
	}
	if !found {
		t.Error("test plugin not found in listing")
	}

	// Test finding (verifies plugin is found - execution tested via CLI)
	foundPath, err := findPlugin("test-plugin-temp")
	if err != nil {
		t.Errorf("unexpected error finding plugin: %v", err)
	}
	if foundPath != testPlugin {
		t.Errorf("expected %s, got %s", testPlugin, foundPath)
	}
}
