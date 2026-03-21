package internal

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetConfigPath(t *testing.T) {
	t.Setenv("HOME", "/tmp/home")

	got := GetConfigPath()
	want := filepath.Join("/tmp/home", ".piconomous", "config.json")

	assert.Equal(t, want, got)
}

func TestGetConfigPath_WithPICONOMOUS_HOME(t *testing.T) {
	t.Setenv("PICONOMOUS_HOME", "/custom/piconomous")
	t.Setenv("HOME", "/tmp/home")

	got := GetConfigPath()
	want := filepath.Join("/custom/piconomous", "config.json")

	assert.Equal(t, want, got)
}

func TestGetConfigPath_WithPICONOMOUS_CONFIG(t *testing.T) {
	t.Setenv("PICONOMOUS_CONFIG", "/custom/config.json")
	t.Setenv("PICONOMOUS_HOME", "/custom/piconomous")
	t.Setenv("HOME", "/tmp/home")

	got := GetConfigPath()
	want := "/custom/config.json"

	assert.Equal(t, want, got)
}

func TestGetConfigPath_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-specific HOME behavior varies; run on windows")
	}

	testUserProfilePath := `C:\Users\Test`
	t.Setenv("USERPROFILE", testUserProfilePath)

	got := GetConfigPath()
	want := filepath.Join(testUserProfilePath, ".piconomous", "config.json")

	require.True(t, strings.EqualFold(got, want), "GetConfigPath() = %q, want %q", got, want)
}
