package configstore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigPath_Default(t *testing.T) {
	// Clear environment variables
	os.Unsetenv("PICOCLAW_CONFIG")
	os.Unsetenv("PICOCLAW_HOME")

	got, err := ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath() error = %v", err)
	}

	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".picoclaw", "config.json")
	if got != want {
		t.Errorf("ConfigPath() = %q, want %q", got, want)
	}
}

func TestConfigPath_WithPICOCLAW_CONFIG(t *testing.T) {
	// Set PICOCLAW_CONFIG
	customPath := "/custom/path/config.json"
	os.Setenv("PICOCLAW_CONFIG", customPath)
	defer os.Unsetenv("PICOCLAW_CONFIG")

	os.Unsetenv("PICOCLAW_HOME")

	got, err := ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath() error = %v", err)
	}

	if got != customPath {
		t.Errorf("ConfigPath() = %q, want %q", got, customPath)
	}
}

func TestConfigPath_WithPICOCLAW_HOME(t *testing.T) {
	// Clear PICOCLAW_CONFIG, set PICOCLAW_HOME
	os.Unsetenv("PICOCLAW_CONFIG")
	customHome := "/custom/home"
	os.Setenv("PICOCLAW_HOME", customHome)
	defer os.Unsetenv("PICOCLAW_HOME")

	got, err := ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath() error = %v", err)
	}

	want := filepath.Join(customHome, "config.json")
	if got != want {
		t.Errorf("ConfigPath() = %q, want %q", got, want)
	}
}

func TestConfigDir_Default(t *testing.T) {
	os.Unsetenv("PICOCLAW_HOME")

	got, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir() error = %v", err)
	}

	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".picoclaw")
	if got != want {
		t.Errorf("ConfigDir() = %q, want %q", got, want)
	}
}

func TestConfigDir_WithPICOCLAW_HOME(t *testing.T) {
	customHome := "/custom/picoclaw/home"
	os.Setenv("PICOCLAW_HOME", customHome)
	defer os.Unsetenv("PICOCLAW_HOME")

	got, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir() error = %v", err)
	}

	if got != customHome {
		t.Errorf("ConfigDir() = %q, want %q", got, customHome)
	}
}

func TestConfigPath_Priority(t *testing.T) {
	// PICOCLAW_CONFIG should take priority over PICOCLAW_HOME
	configPath := "/priority/config.json"
	homePath := "/priority/home"

	os.Setenv("PICOCLAW_CONFIG", configPath)
	os.Setenv("PICOCLAW_HOME", homePath)
	defer func() {
		os.Unsetenv("PICOCLAW_CONFIG")
		os.Unsetenv("PICOCLAW_HOME")
	}()

	got, err := ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath() error = %v", err)
	}

	if got != configPath {
		t.Errorf("ConfigPath() = %q, want %q (PICOCLAW_CONFIG should have priority)", got, configPath)
	}
}

// Windows-specific test for case-insensitive comparison
func TestConfigPath_Windows(t *testing.T) {
	if os.PathSeparator != '\\' {
		t.Skip("Skipping Windows-specific test on non-Windows platform")
	}

	os.Unsetenv("PICOCLAW_CONFIG")
	os.Unsetenv("PICOCLAW_HOME")

	got, err := ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath() error = %v", err)
	}

	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".picoclaw", "config.json")
	if !strings.EqualFold(got, want) {
		t.Errorf("ConfigPath() = %q, want %q", got, want)
	}
}
