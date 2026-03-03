package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/sipeed/picoclaw/pkg/config"
)

const Logo = "🦞"

var (
	version   = "dev"
	gitCommit string
	buildTime string
	goVersion string
)

func GetConfigPath() string {
	if configPath := os.Getenv("PICOCLAW_CONFIG"); configPath != "" {
		return configPath
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".picoclaw", "config.json")
}

func LoadConfig() (*config.Config, error) {
	return config.LoadConfig(GetConfigPath())
}

// FormatVersion returns the version string with optional git commit
func FormatVersion() string {
	v := version
	if gitCommit != "" {
		v += fmt.Sprintf(" (git: %s)", gitCommit)
	}
	return v
}

// FormatBuildInfo returns build time and go version info
func FormatBuildInfo() (string, string) {
	build := buildTime
	goVer := goVersion
	if goVer == "" {
		goVer = runtime.Version()
	}
	return build, goVer
}

// GetVersion returns the version string
func GetVersion() string {
	return version
}

// WarnMissingBootstrap checks workspace bootstrap files (SOUL.md, IDENTITY.md, USER.md)
// and warns the user if any are missing or unmodified.
func WarnMissingBootstrap(workspace string) {
	files := []struct {
		name string
		desc string
	}{
		{"SOUL.md", "personality & behavior"},
		{"IDENTITY.md", "agent name & description"},
		{"USER.md", "your preferences & info"},
	}

	var missing []string
	for _, f := range files {
		path := filepath.Join(workspace, f.name)
		info, err := os.Stat(path)
		if os.IsNotExist(err) {
			missing = append(missing, fmt.Sprintf("    %s  — %s", f.name, f.desc))
		} else if err == nil && info.Size() < 50 {
			// File exists but appears to be empty/placeholder
			missing = append(missing, fmt.Sprintf("    %s  — %s (empty)", f.name, f.desc))
		}
	}

	if len(missing) > 0 {
		fmt.Println("  Customize your agent:")
		for _, m := range missing {
			fmt.Println(m)
		}
		fmt.Printf("  Edit files in: %s\n\n", workspace)
	}
}
