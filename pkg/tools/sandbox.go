package tools

// Preset defines the capability tier for a subagent.
type Preset string

const (
	PresetScout       Preset = "scout"
	PresetAnalyst     Preset = "analyst"
	PresetCoder       Preset = "coder"
	PresetWorker      Preset = "worker"
	PresetCoordinator Preset = "coordinator"
)

// IsValidPreset checks if the given preset is a valid capability tier.
func IsValidPreset(p Preset) bool {
	switch p {
	case PresetScout, PresetAnalyst, PresetCoder, PresetWorker, PresetCoordinator:
		return true
	}
	return false
}

// ExecPolicy defines which commands are allowed for execution.
type ExecPolicy struct {
	AllowRules   []string // Command prefix allowlist (e.g., "go test", "pnpm run test")
	LocalNetOnly bool     // Restrict curl/wget to localhost and RFC 1918 private addresses
}

// SandboxConfig describes the sandbox isolation policy for a preset.
type SandboxConfig struct {
	Preset           Preset
	WriteRoot        string          // Path restriction for write tools; empty = no writes allowed
	AllowedTools     map[string]bool // Tools that can be used
	ExecPolicy       *ExecPolicy     // nil = exec not allowed
	SpawnablePresets []string        // Presets that can be spawned; nil = spawn not allowed
}

// SubagentEnvironment provides context for subagent execution.
type SubagentEnvironment struct {
	Workspace    string   // Absolute path to workspace
	WorktreeDir  string   // Absolute path to worktree (write root)
	Background   string   // Additional context/intent
	Constraints  string   // Execution constraints
	ContextFiles []string // Files to provide as context
}

// presetAllowRules maps presets to command prefix allowlists.
// Each entry is a command prefix: the first N words of the executed command
// must match exactly. e.g. "go test" allows "go test ./..." but not "go build".
// A single word like "curl" allows any arguments.
// curl/wget are included where exec is allowed; LocalNetOnly in ExecPolicy
// ensures all curl/wget requests are restricted to localhost and RFC 1918 addresses.
var presetAllowRules = map[Preset][]string{
	PresetScout: nil, // No exec allowed
	PresetAnalyst: {
		"go test", "go vet",
		"git log", "git diff", "git status",
		"curl", "wget", "grep", "find",
	},
	PresetCoder: {
		"go test", "go vet", "go fmt",
		"gofmt", "goimports", "golangci-lint",
		"prettier", "eslint", "black", "ruff",
		"cargo test", "cargo fmt", "cargo clippy",
		"pnpm test", "pnpm run test", "pnpm run lint", "pnpm run format",
		"bun test", "bun run test", "bun run lint", "bun run format",
		"uv run",
		"curl", "wget",
	},
	PresetWorker: {
		"go",
		"pnpm install", "pnpm add", "pnpm run", "pnpm test", "pnpm build",
		"bun install", "bun add", "bun run", "bun test", "bun build",
		"uv run", "uv sync", "uv add", "uv pip install",
		"pip install",
		"cargo",
		"curl", "wget",
	},
	PresetCoordinator: {
		"go",
		"pnpm", "bun",
		"curl", "wget",
	},
}

// presetSpawnablePresets maps presets to which presets they can spawn.
var presetSpawnablePresets = map[Preset][]string{
	PresetScout:       nil,
	PresetAnalyst:     nil,
	PresetCoder:       nil,
	PresetWorker:      nil,
	PresetCoordinator: {"scout", "analyst", "coder", "worker"},
}

// AllowedToolsForPreset returns the set of allowed tools for a given preset.
func AllowedToolsForPreset(p Preset) map[string]bool {
	// Base tools available to all presets
	allowed := map[string]bool{
		"read_file":  true,
		"list_dir":   true,
		"web_search": true,
		"web_fetch":  true,
		"message":    true,
	}

	// Add analyst+ tools (exec, git, etc.)
	if p == PresetAnalyst || p == PresetCoder || p == PresetWorker || p == PresetCoordinator {
		allowed["exec"] = true
	}

	// Add coder/worker/coordinator tools (write, bg_monitor)
	if p == PresetCoder || p == PresetWorker || p == PresetCoordinator {
		allowed["write_file"] = true
		allowed["edit_file"] = true
		allowed["append_file"] = true
		allowed["bg_monitor"] = true
	}

	// Add coordinator-only tools (spawn)
	if p == PresetCoordinator {
		allowed["spawn"] = true
	}

	return allowed
}

// SandboxConfigForPreset creates a SandboxConfig for the given preset.
func SandboxConfigForPreset(p Preset, writeRoot string) SandboxConfig {
	allowed := AllowedToolsForPreset(p)

	config := SandboxConfig{
		Preset:       p,
		AllowedTools: allowed,
	}

	// Only set WriteRoot for presets that have write permissions
	if allowed["write_file"] {
		config.WriteRoot = writeRoot
	}

	// Set ExecPolicy if exec is allowed and rules are defined.
	// LocalNetOnly is always true: curl/wget in subagents is for local server
	// testing only; external HTTP access goes through the web_fetch tool.
	if allowed["exec"] {
		if rules := presetAllowRules[p]; len(rules) > 0 {
			config.ExecPolicy = &ExecPolicy{
				AllowRules:   rules,
				LocalNetOnly: true,
			}
		}
	}

	// Set SpawnablePresets if spawn is allowed
	if allowed["spawn"] {
		config.SpawnablePresets = presetSpawnablePresets[p]
	}

	return config
}
