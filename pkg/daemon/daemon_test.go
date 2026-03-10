package daemon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		WorkspaceDir:  tmpDir,
		Executable:    "/usr/bin/false",
		Args:          []string{},
		MaxRestarts:   5,
		RestartWindow: 10 * time.Minute,
	}

	d := New(cfg)

	if d == nil {
		t.Fatal("New() returned nil")
	}

	if d.pidFile != filepath.Join(tmpDir, PIDFileName) {
		t.Errorf("Expected pidFile %s, got %s", filepath.Join(tmpDir, PIDFileName), d.pidFile)
	}

	if d.stateFile != filepath.Join(tmpDir, StateFileName) {
		t.Errorf("Expected stateFile %s, got %s", filepath.Join(tmpDir, StateFileName), d.stateFile)
	}

	if d.logFile != filepath.Join(tmpDir, LogFileName) {
		t.Errorf("Expected logFile %s, got %s", filepath.Join(tmpDir, LogFileName), d.logFile)
	}

	if d.maxRestarts != 5 {
		t.Errorf("Expected maxRestarts 5, got %d", d.maxRestarts)
	}

	if d.restartWindow != 10*time.Minute {
		t.Errorf("Expected restartWindow 10m, got %v", d.restartWindow)
	}
}

func TestNewDefaults(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		WorkspaceDir: tmpDir,
		Executable:   "/usr/bin/false",
		Args:         []string{},
	}

	d := New(cfg)

	if d.maxRestarts != DefaultMaxRestarts {
		t.Errorf("Expected default maxRestarts %d, got %d", DefaultMaxRestarts, d.maxRestarts)
	}

	if d.restartWindow != DefaultRestartWindow {
		t.Errorf("Expected default restartWindow %v, got %v", DefaultRestartWindow, d.restartWindow)
	}

	if d.maxLogSize != DefaultMaxLogSize {
		t.Errorf("Expected default maxLogSize %d, got %d", DefaultMaxLogSize, d.maxLogSize)
	}

	if d.maxLogBackups != DefaultMaxLogBackups {
		t.Errorf("Expected default maxLogBackups %d, got %d", DefaultMaxLogBackups, d.maxLogBackups)
	}
}

func TestSaveAndLoadState(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		WorkspaceDir: tmpDir,
		Executable:   "/usr/bin/false",
		Args:         []string{},
	}

	d := New(cfg)

	state := &State{
		PID:           12345,
		StartTime:     time.Now().UTC().Truncate(time.Second),
		RestartCount:  2,
		LastRestart:   time.Now().UTC().Truncate(time.Second),
		CrashCount:    1,
		FirstCrashTime: time.Now().UTC().Truncate(time.Second),
	}

	if err := d.saveState(state); err != nil {
		t.Fatalf("saveState() failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(d.stateFile); os.IsNotExist(err) {
		t.Fatal("State file was not created")
	}

	// Load state
	loaded, err := d.loadState()
	if err != nil {
		t.Fatalf("loadState() failed: %v", err)
	}

	if loaded.PID != state.PID {
		t.Errorf("Expected PID %d, got %d", state.PID, loaded.PID)
	}

	if loaded.RestartCount != state.RestartCount {
		t.Errorf("Expected RestartCount %d, got %d", state.RestartCount, loaded.RestartCount)
	}

	if loaded.CrashCount != state.CrashCount {
		t.Errorf("Expected CrashCount %d, got %d", state.CrashCount, loaded.CrashCount)
	}

	// Check file is valid JSON
	data, err := os.ReadFile(d.stateFile)
	if err != nil {
		t.Fatalf("Failed to read state file: %v", err)
	}

	var jsonState State
	if err := json.Unmarshal(data, &jsonState); err != nil {
		t.Fatalf("State file is not valid JSON: %v", err)
	}
}

func TestSaveStateAtomic(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		WorkspaceDir: tmpDir,
		Executable:   "/usr/bin/false",
		Args:         []string{},
	}

	d := New(cfg)

	state := &State{
		PID:       12345,
		StartTime: time.Now().UTC(),
	}

	// Save multiple times to ensure atomic rename works
	for i := 0; i < 10; i++ {
		state.PID = 12345 + i
		if err := d.saveState(state); err != nil {
			t.Fatalf("saveState() iteration %d failed: %v", i, err)
		}
	}

	// Verify final state
	loaded, err := d.loadState()
	if err != nil {
		t.Fatalf("loadState() failed: %v", err)
	}

	if loaded.PID != 12354 {
		t.Errorf("Expected final PID 12354, got %d", loaded.PID)
	}

	// Verify no temp file left
	tmpFile := d.stateFile + ".tmp"
	if _, err := os.Stat(tmpFile); !os.IsNotExist(err) {
		t.Error("Temp file was not cleaned up")
	}
}

func TestLoadStateNonexistent(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		WorkspaceDir: tmpDir,
		Executable:   "/usr/bin/false",
		Args:         []string{},
	}

	d := New(cfg)

	_, err := d.loadState()
	if err == nil {
		t.Error("Expected error loading nonexistent state file")
	}
}

func TestIsProcessRunning(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		WorkspaceDir: tmpDir,
		Executable:   "/usr/bin/false",
		Args:         []string{},
	}

	d := New(cfg)

	// Test with current process (should be running)
	if !d.isProcessRunning(os.Getpid()) {
		t.Error("Expected current process to be running")
	}

	// Test with invalid PID
	if d.isProcessRunning(-1) {
		t.Error("Expected invalid PID to not be running")
	}

	if d.isProcessRunning(0) {
		t.Error("Expected PID 0 to not be running")
	}

	// Test with likely unused PID (999999)
	if d.isProcessRunning(999999) {
		t.Error("Expected unlikely PID 999999 to not be running")
	}
}

func TestRecordCrash(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		WorkspaceDir:  tmpDir,
		Executable:    "/usr/bin/false",
		Args:          []string{},
		MaxRestarts:   3,
		RestartWindow: 5 * time.Minute,
	}

	d := New(cfg)

	// Record initial state
	state := &State{
		PID:       os.Getpid(),
		StartTime: time.Now(),
		CrashCount: 0,
	}
	if err := d.saveState(state); err != nil {
		t.Fatalf("Failed to save initial state: %v", err)
	}

	// Record first crash
	if err := d.recordCrash(); err != nil {
		t.Fatalf("recordCrash() failed: %v", err)
	}

	loaded, err := d.loadState()
	if err != nil {
		t.Fatalf("loadState() failed: %v", err)
	}

	if loaded.CrashCount != 1 {
		t.Errorf("Expected CrashCount 1, got %d", loaded.CrashCount)
	}

	if loaded.RestartCount != 1 {
		t.Errorf("Expected RestartCount 1, got %d", loaded.RestartCount)
	}

	if loaded.FirstCrashTime.IsZero() {
		t.Error("Expected FirstCrashTime to be set")
	}

	// Record second crash
	firstCrashTime := loaded.FirstCrashTime
	if err := d.recordCrash(); err != nil {
		t.Fatalf("recordCrash() failed: %v", err)
	}

	loaded, err = d.loadState()
	if err != nil {
		t.Fatalf("loadState() failed: %v", err)
	}

	if loaded.CrashCount != 2 {
		t.Errorf("Expected CrashCount 2, got %d", loaded.CrashCount)
	}

	if loaded.RestartCount != 2 {
		t.Errorf("Expected RestartCount 2, got %d", loaded.RestartCount)
	}

	if !loaded.FirstCrashTime.Equal(firstCrashTime) {
		t.Error("Expected FirstCrashTime to remain unchanged")
	}
}

func TestShouldRestart(t *testing.T) {
	tests := []struct {
		name         string
		crashCount   int
		maxRestarts  int
		windowExpiry bool
		shouldRestart bool
		minDelay     time.Duration
		maxDelay     time.Duration
	}{
		{
			name:         "first crash",
			crashCount:   1,
			maxRestarts:  3,
			windowExpiry: false,
			shouldRestart: true,
			minDelay:     1 * time.Second,
			maxDelay:     2 * time.Second,
		},
		{
			name:         "second crash",
			crashCount:   2,
			maxRestarts:  3,
			windowExpiry: false,
			shouldRestart: true,
			minDelay:     2 * time.Second,
			maxDelay:     4 * time.Second,
		},
		{
			name:         "third crash",
			crashCount:   3,
			maxRestarts:  3,
			windowExpiry: false,
			shouldRestart: false,
		},
		{
			name:         "crashes outside window",
			crashCount:   3,
			maxRestarts:  3,
			windowExpiry: true,
			shouldRestart: true,
			minDelay:     1 * time.Second,
			maxDelay:     2 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			cfg := Config{
				WorkspaceDir:  tmpDir,
				Executable:    "/usr/bin/false",
				Args:          []string{},
				MaxRestarts:   tt.maxRestarts,
				RestartWindow: 5 * time.Minute,
			}

			d := New(cfg)

			state := &State{
				PID:       os.Getpid(),
				StartTime: time.Now(),
				CrashCount: tt.crashCount - 1,
			}

			if tt.crashCount > 0 {
				if tt.windowExpiry {
					// Set first crash time outside window
					state.FirstCrashTime = time.Now().Add(-10 * time.Minute)
				} else {
					state.FirstCrashTime = time.Now().Add(-1 * time.Minute)
				}
			}

			if err := d.saveState(state); err != nil {
				t.Fatalf("Failed to save state: %v", err)
			}

			// Simulate a crash
			if tt.crashCount > 0 {
				if err := d.recordCrash(); err != nil {
					t.Fatalf("recordCrash() failed: %v", err)
				}
			}

			shouldRestart, delay := d.shouldRestart()

			if shouldRestart != tt.shouldRestart {
				t.Errorf("Expected shouldRestart %v, got %v", tt.shouldRestart, shouldRestart)
			}

			if shouldRestart {
				if delay < tt.minDelay || delay > tt.maxDelay {
					t.Errorf("Expected delay between %v and %v, got %v", tt.minDelay, tt.maxDelay, delay)
				}
			}
		})
	}
}

func TestCleanup(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		WorkspaceDir: tmpDir,
		Executable:   "/usr/bin/false",
		Args:         []string{},
	}

	d := New(cfg)

	// Create state files
	state := &State{
		PID:       12345,
		StartTime: time.Now(),
	}
	if err := d.saveState(state); err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	// Create pid file
	if err := os.WriteFile(d.pidFile, []byte("12345\n"), 0o644); err != nil {
		t.Fatalf("Failed to create pid file: %v", err)
	}

	// Verify files exist
	if _, err := os.Stat(d.stateFile); os.IsNotExist(err) {
		t.Error("State file was not created")
	}
	if _, err := os.Stat(d.pidFile); os.IsNotExist(err) {
		t.Error("PID file was not created")
	}

	// Cleanup
	if err := d.cleanup(); err != nil {
		t.Fatalf("cleanup() failed: %v", err)
	}

	// Verify files are removed
	if _, err := os.Stat(d.stateFile); !os.IsNotExist(err) {
		t.Error("State file was not removed")
	}
	if _, err := os.Stat(d.pidFile); !os.IsNotExist(err) {
		t.Error("PID file was not removed")
	}
}

func TestRotateLog(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		WorkspaceDir:  tmpDir,
		Executable:    "/usr/bin/false",
		Args:          []string{},
		MaxLogSize:    1024,
		MaxLogBackups: 3,
	}

	d := New(cfg)

	// Create initial log file with some content
	content := make([]byte, 1024)
	for i := range content {
		content[i] = 'x'
	}
	if err := os.WriteFile(d.logFile, content, 0o644); err != nil {
		t.Fatalf("Failed to create log file: %v", err)
	}

	// Rotate
	d.rotateLog()

	// Check that .1 exists
	if _, err := os.Stat(d.logFile + ".1"); os.IsNotExist(err) {
		t.Error("Log file was not rotated to .1")
	}

	// Create new log and rotate again
	if err := os.WriteFile(d.logFile, content, 0o644); err != nil {
		t.Fatalf("Failed to create log file: %v", err)
	}
	d.rotateLog()

	// Check .1 and .2 exist
	if _, err := os.Stat(d.logFile + ".1"); os.IsNotExist(err) {
		t.Error("Log file .1 does not exist")
	}
	if _, err := os.Stat(d.logFile + ".2"); os.IsNotExist(err) {
		t.Error("Log file was not rotated to .2")
	}
}

func TestStatus(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		WorkspaceDir: tmpDir,
		Executable:   "/usr/bin/false",
		Args:         []string{},
	}

	d := New(cfg)

	// Test with no state
	_, err := d.Status()
	if err == nil {
		t.Error("Expected error when daemon not running")
	}

	// Save state with current process
	state := &State{
		PID:       os.Getpid(),
		StartTime: time.Now(),
	}
	if err := d.saveState(state); err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	// Get status
	status, err := d.Status()
	if err != nil {
		t.Fatalf("Status() failed: %v", err)
	}

	if status.PID != os.Getpid() {
		t.Errorf("Expected PID %d, got %d", os.Getpid(), status.PID)
	}
}
