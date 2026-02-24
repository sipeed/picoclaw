package tools

import (
	"regexp"
	"testing"
)

// TestAllowedToolsForPreset checks that each preset has appropriate tool access.
func TestAllowedToolsForPreset(t *testing.T) {
	tests := []struct {
		name               string
		preset             Preset
		wantRead           bool
		wantWrite          bool
		wantExec           bool
		wantSpawn          bool
		wantWebSearch      bool
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

// TestPresetExecPatterns_Coder validates coder exec allowlist.
func TestPresetExecPatterns_Coder(t *testing.T) {
	pattern, ok := presetExecPatterns[PresetCoder]
	if !ok || pattern == "" {
		t.Fatalf("coder pattern missing or empty")
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		t.Fatalf("failed to compile pattern: %v", err)
	}

	tests := []struct {
		cmd    string
		wantOK bool
	}{
		{"go test ./...", true},
		{"go vet ./...", true},
		{"gofmt -w file.go", true},
		{"golangci-lint run", true},
		{"go build ./...", false},
		{"npm install", false},
		{"pnpm test", true},
	}

	for _, tt := range tests {
		gotOK := re.MatchString(tt.cmd)
		if gotOK != tt.wantOK {
			t.Errorf("cmd %q: got %v, want %v", tt.cmd, gotOK, tt.wantOK)
		}
	}
}

// TestPresetExecPatterns_Analyst validates analyst exec allowlist.
func TestPresetExecPatterns_Analyst(t *testing.T) {
	pattern, ok := presetExecPatterns[PresetAnalyst]
	if !ok || pattern == "" {
		t.Fatalf("analyst pattern missing or empty")
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		t.Fatalf("failed to compile pattern: %v", err)
	}

	tests := []struct {
		cmd    string
		wantOK bool
	}{
		{"go test ./...", true},
		{"go vet ./...", true},
		{"git log --oneline", true},
		{"git diff HEAD", true},
		{"grep pattern file", true},
		{"curl http://example.com", true},
		{"go build ./...", false},
		{"npm install", false},
	}

	for _, tt := range tests {
		gotOK := re.MatchString(tt.cmd)
		if gotOK != tt.wantOK {
			t.Errorf("cmd %q: got %v, want %v", tt.cmd, gotOK, tt.wantOK)
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
