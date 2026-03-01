package audit

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestNew_Disabled(t *testing.T) {
	cfg := config.AuditConfig{Enabled: false}
	logger, err := New(cfg, "/tmp/test")
	if err != nil {
		t.Errorf("expected no error when disabled, got %v", err)
	}
	if logger != nil {
		t.Error("expected nil logger when disabled")
	}
}

func TestLogger_Log(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.AuditConfig{
		Enabled:  true,
		Location: tmpDir,
		Format:   "json",
		Rotation: config.RotationConfig{},
		Events: config.AuditEvents{
			ToolCalls: true,
			Messages:  true,
			Errors:    true,
			System:    true,
		},
	}

	logger, err := New(cfg, tmpDir)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Close()

	// Log a test entry
	entry := &Entry{
		Timestamp: time.Now().UTC(),
		Level:     LevelInfo,
		Component: "test",
		EventType: EventSystem,
		RequestID: "test-request-123",
		System: &SystemData{
			Operation: "test_operation",
			Details:   map[string]interface{}{"key": "value"},
		},
	}

	logger.Log(entry)

	// Give the worker time to write
	time.Sleep(100 * time.Millisecond)

	// Verify file was created
	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("failed to read temp dir: %v", err)
	}

	if len(files) == 0 {
		t.Fatal("expected log file to be created")
	}

	// Read and verify the log entry
	logFile := filepath.Join(tmpDir, files[0].Name())
	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	var loggedEntry Entry
	if err := json.Unmarshal(data, &loggedEntry); err != nil {
		t.Fatalf("failed to unmarshal log entry: %v", err)
	}

	if loggedEntry.RequestID != "test-request-123" {
		t.Errorf("expected request_id=test-request-123, got %s", loggedEntry.RequestID)
	}
	if loggedEntry.Component != "test" {
		t.Errorf("expected component=test, got %s", loggedEntry.Component)
	}
}

func TestLogger_Filtering(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.AuditConfig{
		Enabled:  true,
		Location: tmpDir,
		Format:   "json",
		Rotation: config.RotationConfig{},
		Events: config.AuditEvents{
			ToolCalls: true,
			Messages:  false, // Disabled
			Errors:    true,
			System:    true,
		},
	}

	logger, err := New(cfg, tmpDir)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Close()

	// Log a message (should be filtered out)
	logger.Log(&Entry{
		Timestamp: time.Now().UTC(),
		Level:     LevelInfo,
		Component: "test",
		EventType: EventMessage,
		RequestID: "filtered-request-xyz",
	})

	// Log a system event (should pass through)
	logger.Log(&Entry{
		Timestamp: time.Now().UTC(),
		Level:     LevelInfo,
		Component: "test",
		EventType: EventSystem,
		RequestID: "allowed-request-xyz",
	})

	// Close to flush
	logger.Close()

	// Verify only system event was logged
	files, _ := os.ReadDir(tmpDir)
	if len(files) == 0 {
		t.Fatal("expected log file to be created")
	}

	logFile := filepath.Join(tmpDir, files[0].Name())
	data, _ := os.ReadFile(logFile)

	content := string(data)
	
	// Should not contain "filtered-request"
	if strings.Contains(content, "filtered-request-xyz") {
		t.Error("message entry should have been filtered out")
	}

	// Should contain "allowed-request"
	if !strings.Contains(content, "allowed-request-xyz") {
		t.Error("system entry should have been logged")
	}
}

func TestContextPropagation(t *testing.T) {
	ctx := context.Background()

	// Test request ID
	ctx = WithRequestID(ctx, "req-123")
	if id := RequestIDFromContext(ctx); id != "req-123" {
		t.Errorf("expected request_id=req-123, got %s", id)
	}

	// Test session ID
	ctx = WithSessionID(ctx, "sess-456")
	if id := SessionIDFromContext(ctx); id != "sess-456" {
		t.Errorf("expected session_id=sess-456, got %s", id)
	}

	// Test agent ID
	ctx = WithAgentID(ctx, "agent-789")
	if id := AgentIDFromContext(ctx); id != "agent-789" {
		t.Errorf("expected agent_id=agent-789, got %s", id)
	}
}

func TestContextPropagation_Empty(t *testing.T) {
	ctx := context.Background()

	// Test empty context returns empty strings
	if id := RequestIDFromContext(ctx); id != "" {
		t.Errorf("expected empty request_id, got %s", id)
	}
	if id := SessionIDFromContext(ctx); id != "" {
		t.Errorf("expected empty session_id, got %s", id)
	}
	if id := AgentIDFromContext(ctx); id != "" {
		t.Errorf("expected empty agent_id, got %s", id)
	}
}

func TestLogger_NilSafety(t *testing.T) {
	// All methods should be safe to call on nil logger
	var logger *Logger

	// These should not panic
	logger.Log(&Entry{})
	logger.LogToolCall(context.Background(), &ToolCallData{}, 100)
	logger.LogMessage(context.Background(), "inbound", "text", "test", "")
	logger.LogError(context.Background(), "test", "message", true)
	logger.LogSystem(context.Background(), "test", nil)

	err := logger.Close()
	if err != nil {
		t.Errorf("expected no error on close of nil logger, got %v", err)
	}
}

func TestGlobalLogger(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.AuditConfig{
		Enabled:  true,
		Location: tmpDir,
		Format:   "json",
		Events: config.AuditEvents{
			System: true,
		},
	}

	// Initialize global logger
	err := InitGlobal(cfg, tmpDir)
	if err != nil {
		t.Fatalf("failed to init global logger: %v", err)
	}
	defer CloseGlobal()

	// Log via global functions
	ctx := WithRequestID(context.Background(), "global-test")
	LogSystem(ctx, "test_operation", map[string]interface{}{"test": true})

	// Give time to write
	time.Sleep(100 * time.Millisecond)

	// Verify global logger is set
	if Global() == nil {
		t.Error("expected global logger to be set")
	}
}
