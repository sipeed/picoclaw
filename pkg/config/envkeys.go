// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package config

import (
	"os"
	"path/filepath"

	"github.com/sipeed/picoclaw/pkg"
)

// Runtime environment variable keys for the picoclaw process.
// These control the location of files and binaries at runtime and are read
// directly via os.Getenv / os.LookupEnv. All picoclaw-specific keys use the
// PICOCLAW_ prefix. Reference these constants instead of inline string
// literals to keep all supported knobs visible in one place and to prevent
// typos.
const (
	// EnvHome overrides the base directory for all picoclaw data
	// (config, workspace, skills, auth store, …).
	// Default: ~/.picoclaw
	EnvHome = "PICOCLAW_HOME"

	// EnvConfig overrides the full path to the JSON config file.
	// Default: $PICOCLAW_HOME/config.json
	EnvConfig = "PICOCLAW_CONFIG"

	// EnvBuiltinSkills overrides the directory from which built-in
	// skills are loaded.
	// Default: <cwd>/skills
	EnvBuiltinSkills = "PICOCLAW_BUILTIN_SKILLS"

	// EnvBinary overrides the path to the picoclaw executable.
	// Used by the web launcher when spawning the gateway subprocess.
	// Default: resolved from the same directory as the current executable.
	EnvBinary = "PICOCLAW_BINARY"

	// EnvGatewayHost overrides the host address for the gateway server.
	// Default: "localhost"
	EnvGatewayHost = "PICOCLAW_GATEWAY_HOST"
)

func GetHome() string {
	homePath, _ := os.UserHomeDir()
	if picoclawHome := os.Getenv(EnvHome); picoclawHome != "" {
		homePath = picoclawHome
	} else if homePath != "" {
		homePath = filepath.Join(homePath, pkg.DefaultPicoClawHome)
	}
	if homePath == "" {
		homePath = "."
	}
	return homePath
}
