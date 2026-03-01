package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
)

// Logger provides structured audit logging with rotation and filtering.
//
// The logger is safe for concurrent use. All methods are non-blocking;
// entries are queued for async writing to minimize performance impact.
type Logger struct {
	config config.AuditConfig

	// Buffered channel for async writing
	entries chan *Entry

	// Writer handles file I/O and rotation
	writer *rotatingWriter

	// Worker control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Event filtering
	filter *eventFilter

	// Closed flag for safe shutdown
	closed bool
	mu     sync.RWMutex
}

// New creates a new audit logger with the given configuration.
//
// The workspace parameter is used to resolve relative paths in the
// audit configuration. If the audit location is already absolute,
// it is used as-is.
//
// Returns nil if audit logging is disabled in the configuration.
func New(cfg config.AuditConfig, workspace string) (*Logger, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	// Resolve log directory
	logDir := cfg.Location
	if !filepath.IsAbs(logDir) {
		logDir = filepath.Join(workspace, logDir)
	}

	// Ensure directory exists with secure permissions
	if err := os.MkdirAll(logDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create audit log directory: %w", err)
	}

	// Create rotating writer
	rw, err := newRotatingWriter(logDir, cfg.Rotation, cfg.Format)
	if err != nil {
		return nil, fmt.Errorf("failed to create rotating writer: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	l := &Logger{
		config:  cfg,
		entries: make(chan *Entry, 1000), // Buffer up to 1000 entries
		writer:  rw,
		ctx:     ctx,
		cancel:  cancel,
		filter:  newEventFilter(cfg.Events),
	}

	// Start background worker
	l.wg.Add(1)
	go l.worker()

	return l, nil
}

// Log writes a single audit entry.
// This method is non-blocking; the entry is queued for async writing.
// If the logger is closed or the buffer is full, the entry is dropped.
// Safe to call on nil logger (no-op).
func (l *Logger) Log(entry *Entry) {
	if l == nil {
		return
	}

	l.mu.RLock()
	if l.closed {
		l.mu.RUnlock()
		return
	}
	l.mu.RUnlock()

	// Filter by event type
	if !l.filter.Allow(entry.EventType) {
		return
	}

	// Set timestamp if not already set
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	// Try to queue entry without blocking
	select {
	case l.entries <- entry:
	default:
		// Buffer full, drop entry (better than blocking)
		// In production, we might want to increment a metric here
	}
}

// LogToolCall logs a tool execution event.
func (l *Logger) LogToolCall(ctx context.Context, data *ToolCallData, duration time.Duration) {
	if l == nil {
		return
	}

	l.Log(&Entry{
		Level:      LevelInfo,
		Component:  "tool",
		EventType:  EventToolCall,
		RequestID:  RequestIDFromContext(ctx),
		SessionID:  SessionIDFromContext(ctx),
		AgentID:    AgentIDFromContext(ctx),
		ToolCall:   data,
		DurationMs: duration.Milliseconds(),
	})
}

// LogMessage logs a message event (inbound or outbound).
func (l *Logger) LogMessage(ctx context.Context, direction, contentType, content, messageID string) {
	if l == nil {
		return
	}

	// Truncate content to avoid huge log entries
	const maxContentLen = 10000
	if len(content) > maxContentLen {
		content = content[:maxContentLen] + "... [truncated]"
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

// LogError logs an error event.
func (l *Logger) LogError(ctx context.Context, errorType, message string, recoverable bool) {
	if l == nil {
		return
	}

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

// LogSystem logs a system-level event.
func (l *Logger) LogSystem(ctx context.Context, operation string, details map[string]interface{}) {
	if l == nil {
		return
	}

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

// WithChannelContext returns a context enriched with channel information.
// This context can be used with Log* methods to automatically include channel details.
func WithChannelContext(ctx context.Context, channel, chatID, userID string) context.Context {
	ctx = context.WithValue(ctx, channelKey, channel)
	ctx = context.WithValue(ctx, chatIDKey, chatID)
	ctx = context.WithValue(ctx, userIDKey, userID)
	return ctx
}

// LogWithChannel logs an entry with channel context extracted from the context.
func (l *Logger) LogWithChannel(ctx context.Context, entry *Entry) {
	if l == nil {
		return
	}

	// Extract channel context
	if channel, ok := ctx.Value(channelKey).(string); ok {
		entry.Channel = channel
	}
	if chatID, ok := ctx.Value(chatIDKey).(string); ok {
		entry.ChatID = chatID
	}
	if userID, ok := ctx.Value(userIDKey).(string); ok {
		entry.UserID = userID
	}

	l.Log(entry)
}

// Close gracefully shuts down the audit logger.
// It flushes any pending entries and closes the log file.
// This method blocks until all pending entries are written.
// Safe to call on nil logger (returns nil).
func (l *Logger) Close() error {
	if l == nil {
		return nil
	}

	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return nil
	}
	l.closed = true
	l.mu.Unlock()

	// Signal worker to stop
	l.cancel()

	// Wait for worker to finish
	l.wg.Wait()

	// Close the writer
	return l.writer.Close()
}

// worker processes the entry queue in the background.
func (l *Logger) worker() {
	defer l.wg.Done()

	for {
		select {
		case <-l.ctx.Done():
			// Flush remaining entries
			l.flush()
			return
		case entry := <-l.entries:
			if err := l.writeEntry(entry); err != nil {
				// Log to stderr as fallback (can't use logger to avoid recursion)
				fmt.Fprintf(os.Stderr, "[audit] failed to write entry: %v\n", err)
			}
		}
	}
}

// flush writes all pending entries without blocking.
func (l *Logger) flush() {
	for {
		select {
		case entry := <-l.entries:
			if err := l.writeEntry(entry); err != nil {
				fmt.Fprintf(os.Stderr, "[audit] failed to write entry during flush: %v\n", err)
			}
		default:
			// No more entries
			return
		}
	}
}

// writeEntry serializes and writes a single entry.
func (l *Logger) writeEntry(entry *Entry) error {
	var data []byte
	var err error

	if l.config.Format == "json" {
		data, err = json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("failed to marshal entry: %w", err)
		}
		data = append(data, '\n')
	} else {
		// Text format
		data = []byte(l.formatText(entry) + "\n")
	}

	return l.writer.Write(data)
}

// formatText formats an entry as human-readable text.
func (l *Logger) formatText(entry *Entry) string {
	return fmt.Sprintf("[%s] %s %s %s: %s",
		entry.Timestamp.Format(time.RFC3339),
		entry.Level,
		entry.Component,
		entry.EventType,
		l.formatTextDetails(entry),
	)
}

// formatTextDetails formats event-specific details for text output.
func (l *Logger) formatTextDetails(entry *Entry) string {
	switch entry.EventType {
	case EventToolCall:
		if entry.ToolCall != nil {
			return fmt.Sprintf("tool=%s async=%v error=%v",
				entry.ToolCall.Name,
				entry.ToolCall.IsAsync,
				entry.ToolCall.IsError,
			)
		}
	case EventMessage:
		if entry.Message != nil {
			return fmt.Sprintf("direction=%s type=%s",
				entry.Message.Direction,
				entry.Message.ContentType,
			)
		}
	case EventError:
		if entry.Error != nil {
			return fmt.Sprintf("type=%s recoverable=%v msg=%s",
				entry.Error.ErrorType,
				entry.Error.Recoverable,
				entry.Error.Message,
			)
		}
	}
	return ""
}

// channel context keys
type channelContextKey int

const (
	channelKey channelContextKey = iota
	chatIDKey
	userIDKey
)

// eventFilter determines which event types should be logged.
type eventFilter struct {
	events config.AuditEvents
}

func newEventFilter(events config.AuditEvents) *eventFilter {
	return &eventFilter{events: events}
}

func (f *eventFilter) Allow(eventType EventType) bool {
	switch eventType {
	case EventToolCall:
		return f.events.ToolCalls
	case EventMessage:
		return f.events.Messages
	case EventError:
		return f.events.Errors
	case EventSystem:
		return f.events.System
	default:
		return true
	}
}
