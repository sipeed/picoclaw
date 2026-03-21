// Piconomous - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 Piconomous contributors

package config

// Runtime environment variable keys for the piconomous process.
// These control the location of files and binaries at runtime and are read
// directly via os.Getenv / os.LookupEnv. All piconomous-specific keys use the
// PICONOMOUS_ prefix. Reference these constants instead of inline string
// literals to keep all supported knobs visible in one place and to prevent
// typos.
const (
	// EnvHome overrides the base directory for all piconomous data
	// (config, workspace, skills, auth store, …).
	// Default: ~/.piconomous
	EnvHome = "PICONOMOUS_HOME"

	// EnvConfig overrides the full path to the JSON config file.
	// Default: $PICONOMOUS_HOME/config.json
	EnvConfig = "PICONOMOUS_CONFIG"

	// EnvBuiltinSkills overrides the directory from which built-in
	// skills are loaded.
	// Default: <cwd>/skills
	EnvBuiltinSkills = "PICONOMOUS_BUILTIN_SKILLS"

	// EnvBinary overrides the path to the piconomous executable.
	// Used by the web launcher when spawning the gateway subprocess.
	// Default: resolved from the same directory as the current executable.
	EnvBinary = "PICONOMOUS_BINARY"

	// EnvGatewayHost overrides the host address for the gateway server.
	// Default: "127.0.0.1"
	EnvGatewayHost = "PICONOMOUS_GATEWAY_HOST"

	// EnvAutonomousEnabled enables fully autonomous goal-driven operation.
	// When set to "true", the agent proactively pursues goals without human prompting.
	// Default: "false"
	EnvAutonomousEnabled = "PICONOMOUS_AUTONOMOUS_ENABLED"
)
