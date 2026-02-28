package tools

import (
	"testing"
)

// TestAllowedToolsForPreset checks that each preset has appropriate tool access.
func TestAllowedToolsForPreset(t *testing.T) {
	tests := []struct {
		name          string
		preset        Preset
		wantRead      bool
		wantWrite     bool
		wantExec      bool
		wantSpawn     bool
		wantWebSearch bool
	}{
		{
			name:          "scout",
			preset:        PresetScout,
			wantRead:      true,
			wantWrite:     false,
			wantExec:      false,
			wantSpawn:     false,
			wantWebSearch: true,
		},
		{
			name:          "analyst",
			preset:        PresetAnalyst,
			wantRead:      true,
			wantWrite:     false,
			wantExec:      true,
			wantSpawn:     false,
			wantWebSearch: true,
		},
		{
			name:          "coder",
			preset:        PresetCoder,
			wantRead:      true,
			wantWrite:     true,
			wantExec:      true,
			wantSpawn:     false,
			wantWebSearch: true,
		},
		{
			name:          "worker",
			preset:        PresetWorker,
			wantRead:      true,
			wantWrite:     true,
			wantExec:      true,
			wantSpawn:     false,
			wantWebSearch: true,
		},
		{
			name:          "coordinator",
			preset:        PresetCoordinator,
			wantRead:      true,
			wantWrite:     true,
			wantExec:      true,
			wantSpawn:     true,
			wantWebSearch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed := AllowedToolsForPreset(tt.preset)

			// Check read tools
			if got := allowed["read_file"]; got != tt.wantRead {
				t.Errorf("read_file: got %v, want %v", got, tt.wantRead)
			}
			if got := allowed["list_dir"]; got != tt.wantRead {
				t.Errorf("list_dir: got %v, want %v", got, tt.wantRead)
			}

			// Check write tools
			if got := allowed["write_file"]; got != tt.wantWrite {
				t.Errorf("write_file: got %v, want %v", got, tt.wantWrite)
			}

			// Check exec
			if got := allowed["exec"]; got != tt.wantExec {
				t.Errorf("exec: got %v, want %v", got, tt.wantExec)
			}

			// Check spawn
			if got := allowed["spawn"]; got != tt.wantSpawn {
				t.Errorf("spawn: got %v, want %v", got, tt.wantSpawn)
			}

			// Check web_search (all presets should have it)
			if got := allowed["web_search"]; got != tt.wantWebSearch {
				t.Errorf("web_search: got %v, want %v", got, tt.wantWebSearch)
			}
		})
	}
}

// TestSandboxConfigForPreset_Scout checks scout preset isolation.
func TestSandboxConfigForPreset_Scout(t *testing.T) {
	config := SandboxConfigForPreset(PresetScout, "/tmp/scout")

	if config.Preset != PresetScout {
		t.Errorf("Preset: got %v, want %v", config.Preset, PresetScout)
	}
	if config.WriteRoot != "" {
		t.Errorf("WriteRoot: got %q, want empty", config.WriteRoot)
	}
	if config.ExecPolicy != nil {
		t.Errorf("ExecPolicy: got non-nil, want nil")
	}
	if config.SpawnablePresets != nil {
		t.Errorf("SpawnablePresets: got non-nil, want nil")
	}
}

// TestSandboxConfigForPreset_Coder checks coder preset isolation.
func TestSandboxConfigForPreset_Coder(t *testing.T) {
	config := SandboxConfigForPreset(PresetCoder, "/tmp/coder")

	if config.Preset != PresetCoder {
		t.Errorf("Preset: got %v, want %v", config.Preset, PresetCoder)
	}
	if config.WriteRoot != "/tmp/coder" {
		t.Errorf("WriteRoot: got %q, want /tmp/coder", config.WriteRoot)
	}
	if config.ExecPolicy == nil {
		t.Errorf("ExecPolicy: got nil, want non-nil")
	} else if !config.ExecPolicy.LocalNetOnly {
		t.Errorf("ExecPolicy.LocalNetOnly: got false, want true")
	}
	if config.SpawnablePresets != nil {
		t.Errorf("SpawnablePresets: got non-nil, want nil")
	}
}

// TestSandboxConfigForPreset_Coordinator checks coordinator preset isolation.
func TestSandboxConfigForPreset_Coordinator(t *testing.T) {
	config := SandboxConfigForPreset(PresetCoordinator, "/tmp/coord")

	if config.Preset != PresetCoordinator {
		t.Errorf("Preset: got %v, want %v", config.Preset, PresetCoordinator)
	}
	if config.WriteRoot != "/tmp/coord" {
		t.Errorf("WriteRoot: got %q, want /tmp/coord", config.WriteRoot)
	}
	if config.ExecPolicy == nil {
		t.Errorf("ExecPolicy: got nil, want non-nil")
	} else if !config.ExecPolicy.LocalNetOnly {
		t.Errorf("ExecPolicy.LocalNetOnly: got false, want true")
	}
	if config.SpawnablePresets == nil {
		t.Errorf("SpawnablePresets: got nil, want non-nil")
	}
	// Verify coordinator cannot spawn itself
	canSpawnCoordinator := false
	for _, p := range config.SpawnablePresets {
		if p == "coordinator" {
			canSpawnCoordinator = true
			break
		}
	}
	if canSpawnCoordinator {
		t.Errorf("Coordinator should not be in SpawnablePresets")
	}
}

// TestPresetAllowRules_Coder validates coder exec allowlist.
func TestPresetAllowRules_Coder(t *testing.T) {
	rules := presetAllowRules[PresetCoder]
	if len(rules) == 0 {
		t.Fatalf("coder rules missing or empty")
	}

	exec, err := NewExecTool(t.TempDir(), true)
	if err != nil {
		t.Fatalf("NewExecTool: %v", err)
	}
	exec.SetAllowRules(rules)

	tests := []struct {
		cmd    string
		wantOK bool
	}{
		{"go test ./...", true},
		{"go vet ./...", true},
		{"go fmt ./...", true},
		{"gofmt -w file.go", true},
		{"golangci-lint run", true},
		{"cargo test", true},
		{"cargo fmt", true},
		{"cargo clippy", true},
		{"pnpm test", true},
		{"pnpm run test", true},
		{"pnpm run lint", true},
		{"pnpm run format", true},
		{"bun test", true},
		{"bun run test", true},
		{"uv run pytest", true},
		// curl/wget are in the allowlist; LocalNetOnly enforcement is at runtime
		{"curl http://localhost:3000/health", true},
		{"wget http://127.0.0.1:8080/status", true},
		// blocked
		{"go build ./...", false},
		{"npm install", false},
		{"pnpm install", false},
		{"pnpm run build", false},
		{"cargo build", false},
		{"pwd", false},
		{"ls", false},
	}

	for _, tt := range tests {
		result := exec.guardCommand(tt.cmd, t.TempDir())
		gotOK := result == ""
		if gotOK != tt.wantOK {
			t.Errorf("cmd %q: got allowed=%v, want %v (guard: %q)", tt.cmd, gotOK, tt.wantOK, result)
		}
	}
}

// TestPresetAllowRules_Analyst validates analyst exec allowlist.
func TestPresetAllowRules_Analyst(t *testing.T) {
	rules := presetAllowRules[PresetAnalyst]
	if len(rules) == 0 {
		t.Fatalf("analyst rules missing or empty")
	}

	exec, err := NewExecTool(t.TempDir(), true)
	if err != nil {
		t.Fatalf("NewExecTool: %v", err)
	}
	exec.SetAllowRules(rules)

	tests := []struct {
		cmd    string
		wantOK bool
	}{
		{"go test ./...", true},
		{"go vet ./...", true},
		{"git log --oneline", true},
		{"git diff HEAD", true},
		{"git status", true},
		{"grep pattern file", true},
		{"find . -name '*.go'", true},
		{"curl http://example.com", true},
		// blocked
		{"go build ./...", false},
		{"npm install", false},
		{"git push", false},
		{"git checkout", false},
		{"pwd", false},
		{"ls", false},
	}

	for _, tt := range tests {
		result := exec.guardCommand(tt.cmd, t.TempDir())
		gotOK := result == ""
		if gotOK != tt.wantOK {
			t.Errorf("cmd %q: got allowed=%v, want %v (guard: %q)", tt.cmd, gotOK, tt.wantOK, result)
		}
	}
}

// TestPresetAllowRules_Worker validates worker exec allowlist.
func TestPresetAllowRules_Worker(t *testing.T) {
	rules := presetAllowRules[PresetWorker]
	if len(rules) == 0 {
		t.Fatalf("worker rules missing or empty")
	}

	exec, err := NewExecTool(t.TempDir(), true)
	if err != nil {
		t.Fatalf("NewExecTool: %v", err)
	}
	exec.SetAllowRules(rules)

	tests := []struct {
		cmd    string
		wantOK bool
	}{
		{"go build ./...", true},
		{"go test ./...", true},
		{"pnpm install", true},
		{"pnpm add lodash", true},
		{"pnpm run dev", true},
		{"bun install", true},
		{"bun build", true},
		{"uv run pytest", true},
		{"uv sync", true},
		{"uv pip install flask", true},
		{"pip install flask", true},
		{"cargo build", true},
		{"cargo test", true},
		{"curl http://localhost:8080", true},
		// blocked
		{"npm install", false},
		{"pwd", false},
	}

	for _, tt := range tests {
		result := exec.guardCommand(tt.cmd, t.TempDir())
		gotOK := result == ""
		if gotOK != tt.wantOK {
			t.Errorf("cmd %q: got allowed=%v, want %v (guard: %q)", tt.cmd, gotOK, tt.wantOK, result)
		}
	}
}

// TestPresetAllowRules_Coordinator validates coordinator exec allowlist.
func TestPresetAllowRules_Coordinator(t *testing.T) {
	rules := presetAllowRules[PresetCoordinator]
	if len(rules) == 0 {
		t.Fatalf("coordinator rules missing or empty")
	}

	exec, err := NewExecTool(t.TempDir(), true)
	if err != nil {
		t.Fatalf("NewExecTool: %v", err)
	}
	exec.SetAllowRules(rules)

	tests := []struct {
		cmd    string
		wantOK bool
	}{
		{"go build ./...", true},
		{"go test ./...", true},
		{"pnpm install", true},
		{"pnpm run dev", true},
		{"bun install", true},
		{"bun run dev", true},
		{"curl http://localhost:8080", true},
		{"wget http://127.0.0.1:3000", true},
		// blocked
		{"npm install", false},
		{"cargo build", false},
		{"pwd", false},
	}

	for _, tt := range tests {
		result := exec.guardCommand(tt.cmd, t.TempDir())
		gotOK := result == ""
		if gotOK != tt.wantOK {
			t.Errorf("cmd %q: got allowed=%v, want %v (guard: %q)", tt.cmd, gotOK, tt.wantOK, result)
		}
	}
}

// TestIsValidPreset checks preset validation.
func TestIsValidPreset(t *testing.T) {
	tests := []struct {
		preset Preset
		valid  bool
	}{
		{PresetScout, true},
		{PresetAnalyst, true},
		{PresetCoder, true},
		{PresetWorker, true},
		{PresetCoordinator, true},
		{Preset("invalid"), false},
		{Preset(""), false},
	}

	for _, tt := range tests {
		gotValid := IsValidPreset(tt.preset)
		if gotValid != tt.valid {
			t.Errorf("preset %q: got %v, want %v", tt.preset, gotValid, tt.valid)
		}
	}
}
