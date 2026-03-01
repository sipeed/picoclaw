package audit

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
)

// global provides a package-level default logger for convenience.
// It is safe to use even if not initialized - all methods become no-ops.
var global = &globalLogger{}

// globalLogger wraps a *Logger with nil safety.
type globalLogger struct {
	logger *Logger
	mu     sync.RWMutex
}

func (g *globalLogger) set(l *Logger) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.logger = l
}

func (g *globalLogger) get() *Logger {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.logger
}

// InitGlobal initializes the global audit logger.
// This should be called once during application startup.
func InitGlobal(cfg config.AuditConfig, workspace string) error {
	if !cfg.Enabled {
		return nil
	}

	logger, err := New(cfg, workspace)
	if err != nil {
		return err
	}

	global.set(logger)
	return nil
}

// CloseGlobal closes the global audit logger.
// This should be called during graceful shutdown.
func CloseGlobal() error {
	if l := global.get(); l != nil {
		return l.Close()
	}
	return nil
}

// Global returns the global logger instance.
// Returns nil if audit logging is disabled.
func Global() *Logger {
	return global.get()
}

// Convenience functions that delegate to the global logger.
// These are no-ops if audit logging is disabled.

// Log writes a single audit entry to the global logger.
func Log(entry *Entry) {
	if l := global.get(); l != nil {
		l.Log(entry)
	}
}

// LogToolCall logs a tool execution to the global logger.
func LogToolCall(ctx context.Context, data *ToolCallData, duration int64) {
	if l := global.get(); l != nil {
		l.Log(&Entry{
			Level:      LevelInfo,
			Component:  "tool",
			EventType:  EventToolCall,
			RequestID:  RequestIDFromContext(ctx),
			SessionID:  SessionIDFromContext(ctx),
			AgentID:    AgentIDFromContext(ctx),
			ToolCall:   data,
			DurationMs: duration,
		})
	}
}

// LogMessage logs a message event to the global logger.
func LogMessage(ctx context.Context, direction, contentType, content, messageID string) {
	if l := global.get(); l != nil {
		// Truncate content
		const maxLen = 10000
		if len(content) > maxLen {
			content = content[:maxLen] + "... [truncated]"
		}

		l.Log(&Entry{
			Level:     LevelInfo,
			Component: "channel",
			EventType: EventMessage,
			RequestID: RequestIDFromContext(ctx),
			SessionID: SessionIDFromContext(ctx),
			AgentID:   AgentIDFromContext(ctx),
			Message: &MessageData{
				Direction:   direction,
				ContentType: contentType,
				Content:     content,
				MessageID:   messageID,
			},
		})
	}
}

// LogError logs an error event to the global logger.
func LogError(ctx context.Context, errorType, message string, recoverable bool) {
	if l := global.get(); l != nil {
		l.Log(&Entry{
			Level:     LevelError,
			Component: "system",
			EventType: EventError,
			RequestID: RequestIDFromContext(ctx),
			SessionID: SessionIDFromContext(ctx),
			AgentID:   AgentIDFromContext(ctx),
			Error: &ErrorData{
				ErrorType:   errorType,
				Message:     message,
				Recoverable: recoverable,
			},
		})
	}
}

// LogSystem logs a system event to the global logger.
func LogSystem(ctx context.Context, operation string, details map[string]interface{}) {
	if l := global.get(); l != nil {
		l.Log(&Entry{
			Level:     LevelInfo,
			Component: "system",
			EventType: EventSystem,
			RequestID: RequestIDFromContext(ctx),
			SessionID: SessionIDFromContext(ctx),
			System: &SystemData{
				Operation: operation,
				Details:   details,
			},
		})
	}
}

// GenerateRequestID generates a unique request ID for context propagation.
// This should be called at the entry point of each request.
func GenerateRequestID() string {
	// Simple timestamp-based ID; consider UUID for distributed systems
	return fmt.Sprintf("req-%d", time.Now().UnixNano())
}


