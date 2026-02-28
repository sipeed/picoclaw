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
	AllowPattern string // Prefix-match regex; matched commands are allowed
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

// presetExecPatterns maps presets to command allowlist regexes.
var presetExecPatterns = map[Preset]string{
	PresetScout:       ``, // No exec allowed
	PresetAnalyst:     `^(go\s+(test|vet)|git\s+(log|diff|status)|curl|wget|grep|find)\b`,
	PresetCoder:       `^(go\s+(test|vet|fmt)|gofmt|goimports|golangci-lint|prettier|eslint|black|ruff|cargo\s+(test|fmt|clippy)|pnpm\s+(test|run\s+(test|lint|format))|bun\s+(test|run\s+(test|lint|format))|uv\s+run\s+)\b`,
	PresetWorker:      `^(go\s+|pnpm\s+(install|add|run|test|build)|bun\s+(install|add|run|test|build)|uv\s+(run|sync|add|pip\s+install)|pip\s+install|cargo\s+)\b`,
	PresetCoordinator: `^(go\s+|pnpm\s+|bun\s+|curl|wget)\b`,
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

	// Set ExecPolicy if exec is allowed and pattern is non-empty
	if allowed["exec"] {
		if pattern := presetExecPatterns[p]; pattern != "" {
			config.ExecPolicy = &ExecPolicy{
				AllowPattern: pattern,
			}
		}
	}

	// Set SpawnablePresets if spawn is allowed
	if allowed["spawn"] {
		config.SpawnablePresets = presetSpawnablePresets[p]
	}

	return config
}
