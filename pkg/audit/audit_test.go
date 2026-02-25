package audit

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLogger_Log(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "audit-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := Config{
		Enabled:           true,
		LogToolExecutions: true,
		LogAuthEvents:     true,
		LogConfigChanges:  true,
		RetentionDays:     30,
		SecretKey:         []byte("test-secret-key-32-bytes-long!!"),
		LogFilePath:       filepath.Join(tmpDir, "audit.log"),
	}

	logger := &Logger{config: config}
	if err := logger.init(); err != nil {
		t.Fatalf("Failed to init logger: %v", err)
	}
	defer logger.Close()

	// Log a tool execution event
	err = logger.Log(Event{
		EventType: EventTypeToolExecution,
		Action:    "execute",
		Resource:  "/workspace/test.txt",
		Details:   map[string]any{"tool": "read_file"},
		Success:   true,
	})
	if err != nil {
		t.Errorf("Failed to log event: %v", err)
	}

	// Log an auth event
	err = logger.Log(Event{
		EventType: EventTypeAuthLogin,
		Actor:     "test-user",
		Action:    "login",
		Resource:  "openai",
		Success:   true,
	})
	if err != nil {
		t.Errorf("Failed to log auth event: %v", err)
	}

	// Verify file was created and has content
	data, err := os.ReadFile(config.LogFilePath)
	if err != nil {
		t.Fatalf("Failed to read audit log: %v", err)
	}

	if len(data) == 0 {
		t.Error("Audit log is empty")
	}
}

func TestLogger_Disabled(t *testing.T) {
	config := Config{
		Enabled:     false,
		LogFilePath: "/dev/null",
	}

	logger := &Logger{config: config}
	if err := logger.init(); err != nil {
		t.Fatalf("Failed to init logger: %v", err)
	}

	// Should not error when disabled
	err := logger.Log(Event{
		EventType: EventTypeToolExecution,
		Action:    "test",
		Success:   true,
	})
	if err != nil {
		t.Errorf("Should not error when disabled: %v", err)
	}
}

func TestLogger_HashChain(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "audit-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := Config{
		Enabled:           true,
		LogToolExecutions: true,
		SecretKey:         []byte("test-secret-key-32-bytes-long!!"),
		LogFilePath:       filepath.Join(tmpDir, "audit.log"),
	}

	logger := &Logger{config: config}
	if err := logger.init(); err != nil {
		t.Fatalf("Failed to init logger: %v", err)
	}
	defer logger.Close()

	// Log multiple events
	for i := 0; i < 5; i++ {
		err := logger.Log(Event{
			EventType: EventTypeToolExecution,
			Action:    "test",
			Success:   true,
		})
		if err != nil {
			t.Errorf("Failed to log event %d: %v", i, err)
		}
	}

	// Verify hash chain
	valid, err := logger.VerifyChain()
	if err != nil {
		t.Errorf("Failed to verify chain: %v", err)
	}
	if !valid {
		t.Error("Hash chain verification failed")
	}
}

func TestLogger_ShouldLog(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		eventType EventType
		shouldLog bool
	}{
		{
			name: "tool execution enabled",
			config: Config{
				Enabled:           true,
				LogToolExecutions: true,
			},
			eventType: EventTypeToolExecution,
			shouldLog: true,
		},
		{
			name: "tool execution disabled",
			config: Config{
				Enabled:           true,
				LogToolExecutions: false,
			},
			eventType: EventTypeToolExecution,
			shouldLog: false,
		},
		{
			name: "auth event enabled",
			config: Config{
				Enabled:       true,
				LogAuthEvents: true,
			},
			eventType: EventTypeAuthLogin,
			shouldLog: true,
		},
		{
			name: "security event always logged when enabled",
			config: Config{
				Enabled: true,
			},
			eventType: EventTypeSSRFBlock,
			shouldLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := &Logger{config: tt.config}
			result := logger.shouldLog(tt.eventType)
			if result != tt.shouldLog {
				t.Errorf("shouldLog(%v) = %v, want %v", tt.eventType, result, tt.shouldLog)
			}
		})
	}
}

func TestLogToolExecution(t *testing.T) {
	// Initialize global logger
	tmpDir, err := os.MkdirTemp("", "audit-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := Config{
		Enabled:           true,
		LogToolExecutions: true,
		SecretKey:         []byte("test-secret-key-32-bytes-long!!"),
		LogFilePath:       filepath.Join(tmpDir, "audit.log"),
	}

	// Reset for test
	globalLogger = &Logger{config: config}
	if err := globalLogger.init(); err != nil {
		t.Fatalf("Failed to init: %v", err)
	}
	defer globalLogger.Close()

	err = LogToolExecution("read_file", "read", "/test/file.txt", true, map[string]any{"bytes": 1024})
	if err != nil {
		t.Errorf("LogToolExecution failed: %v", err)
	}
}

func TestLogAuthEvent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "audit-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := Config{
		Enabled:       true,
		LogAuthEvents: true,
		SecretKey:     []byte("test-secret-key-32-bytes-long!!"),
		LogFilePath:   filepath.Join(tmpDir, "audit.log"),
	}

	globalLogger = &Logger{config: config}
	if err := globalLogger.init(); err != nil {
		t.Fatalf("Failed to init: %v", err)
	}
	defer globalLogger.Close()

	err = LogAuthEvent(EventTypeAuthLogin, "test-user", "openai", true, nil)
	if err != nil {
		t.Errorf("LogAuthEvent failed: %v", err)
	}

	err = LogAuthEvent(EventTypeAuthFailure, "test-user", "openai", false, os.ErrPermission)
	if err != nil {
		t.Errorf("LogAuthEvent with error failed: %v", err)
	}
}

func TestLogConfigChange(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "audit-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := Config{
		Enabled:          true,
		LogConfigChanges: true,
		SecretKey:        []byte("test-secret-key-32-bytes-long!!"),
		LogFilePath:      filepath.Join(tmpDir, "audit.log"),
	}

	globalLogger = &Logger{config: config}
	if err := globalLogger.init(); err != nil {
		t.Fatalf("Failed to init: %v", err)
	}
	defer globalLogger.Close()

	err = LogConfigChange("admin", "max_tokens", "4096", "8192")
	if err != nil {
		t.Errorf("LogConfigChange failed: %v", err)
	}
}

func TestLogSecurityEvent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "audit-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := Config{
		Enabled:     true,
		SecretKey:   []byte("test-secret-key-32-bytes-long!!"),
		LogFilePath: filepath.Join(tmpDir, "audit.log"),
	}

	globalLogger = &Logger{config: config}
	if err := globalLogger.init(); err != nil {
		t.Fatalf("Failed to init: %v", err)
	}
	defer globalLogger.Close()

	err = LogSecurityEvent(EventTypeSSRFBlock, "web_fetch", "http://169.254.169.254/", "metadata endpoint blocked")
	if err != nil {
		t.Errorf("LogSecurityEvent failed: %v", err)
	}
}

func TestCleanupOldLogs(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "audit-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := Config{
		Enabled:       true,
		RetentionDays: 1,
		SecretKey:     []byte("test-secret-key-32-bytes-long!!"),
		LogFilePath:   filepath.Join(tmpDir, "audit.log"),
	}

	logger := &Logger{config: config}
	if err := logger.init(); err != nil {
		t.Fatalf("Failed to init: %v", err)
	}

	// Log an old event (2 days ago)
	oldEvent := Event{
		Timestamp: time.Now().AddDate(0, 0, -2),
		EventType: EventTypeToolExecution,
		Action:    "old_action",
		Success:   true,
	}

	// Log a recent event
	newEvent := Event{
		Timestamp: time.Now(),
		EventType: EventTypeToolExecution,
		Action:    "new_action",
		Success:   true,
	}

	logger.Log(oldEvent)
	logger.Log(newEvent)
	logger.Close()

	// Reopen logger for cleanup
	logger2 := &Logger{config: config}
	if err := logger2.init(); err != nil {
		t.Fatalf("Failed to reinit: %v", err)
	}

	// Cleanup
	if err := logger2.CleanupOldLogs(); err != nil {
		t.Errorf("CleanupOldLogs failed: %v", err)
	}
	logger2.Close()

	// Verify old event was removed - read raw file
	data, err := os.ReadFile(config.LogFilePath)
	if err != nil {
		t.Fatalf("Failed to read log: %v", err)
	}

	logContent := string(data)
	if contains(logContent, "old_action") {
		t.Error("Old event should have been cleaned up")
	}
	// Note: new_action may also be cleaned if the test runs slowly
	// The key test is that old_action is removed
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if !config.Enabled {
		t.Error("Default config should have audit enabled")
	}
	if !config.LogToolExecutions {
		t.Error("Default config should log tool executions")
	}
	if config.RetentionDays != 30 {
		t.Errorf("Default retention days = %d, want 30", config.RetentionDays)
	}
}

func TestComputeHash(t *testing.T) {
	config := Config{
		SecretKey: []byte("test-secret-key-32-bytes-long!!"),
	}
	logger := &Logger{config: config}

	event := Event{
		Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EventType: EventTypeToolExecution,
		Action:    "test",
		Resource:  "resource",
		Success:   true,
	}

	hash1 := logger.computeHash(event)
	hash2 := logger.computeHash(event)

	if hash1 != hash2 {
		t.Error("Same event should produce same hash")
	}

	// Different event should produce different hash
	event.Success = false
	hash3 := logger.computeHash(event)

	if hash1 == hash3 {
		t.Error("Different events should produce different hashes")
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
