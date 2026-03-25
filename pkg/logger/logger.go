package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

type LogLevel = zerolog.Level

const (
	DEBUG = zerolog.DebugLevel
	INFO  = zerolog.InfoLevel
	WARN  = zerolog.WarnLevel
	ERROR = zerolog.ErrorLevel
	FATAL = zerolog.FatalLevel
)

var (
	logLevelNames = map[LogLevel]string{
		DEBUG: "DEBUG",
		INFO:  "INFO",
		WARN:  "WARN",
		ERROR: "ERROR",
		FATAL: "FATAL",
	}

	levelFromName = map[string]LogLevel{
		"DEBUG": DEBUG,
		"INFO":  INFO,
		"WARN":  WARN,
		"ERROR": ERROR,
		"FATAL": FATAL,
	}

	currentLevel = INFO
	logger       zerolog.Logger
	fileLogger   zerolog.Logger
	logFile      *os.File
	once         sync.Once
	mu           sync.RWMutex

	ringBuf   *logRingBuffer
	logSubs   []*LogSubscriber
	logSubsMu sync.Mutex
)

const ringBufSize = 300

// logRingBuffer is a fixed-size circular buffer for log entries.
type logRingBuffer struct {
	entries []LogEntry
	head    int
	count   int
	seq     uint64
	mu      sync.RWMutex
}

func newLogRingBuffer(size int) *logRingBuffer {
	return &logRingBuffer{
		entries: make([]LogEntry, size),
	}
}

func (rb *logRingBuffer) push(entry LogEntry) {
	rb.mu.Lock()
	rb.entries[rb.head] = entry
	rb.head = (rb.head + 1) % len(rb.entries)
	if rb.count < len(rb.entries) {
		rb.count++
	}
	rb.seq++
	rb.mu.Unlock()
}

func (rb *logRingBuffer) recent(limit int) []LogEntry {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	n := rb.count
	if limit > 0 && limit < n {
		n = limit
	}
	result := make([]LogEntry, n)
	start := (rb.head - n + len(rb.entries)) % len(rb.entries)
	for i := 0; i < n; i++ {
		result[i] = rb.entries[(start+i)%len(rb.entries)]
	}
	return result
}

// visitReverse iterates entries from newest to oldest under the read lock.
// The callback receives a pointer to the internal entry (valid only during
// the call). Return false to stop iteration.
func (rb *logRingBuffer) visitReverse(fn func(*LogEntry) bool) {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	for i := 0; i < rb.count; i++ {
		idx := (rb.head - 1 - i + len(rb.entries)) % len(rb.entries)
		if !fn(&rb.entries[idx]) {
			return
		}
	}
}

// LogSubscriber receives log entries matching its filter.
type LogSubscriber struct {
	Ch     chan LogEntry
	filter func(LogEntry) bool
}

type LogEntry struct {
	Level     string         `json:"level"`
	Timestamp string         `json:"timestamp"`
	Component string         `json:"component,omitempty"`
	Message   string         `json:"message"`
	Fields    map[string]any `json:"fields,omitempty"`
	Caller    string         `json:"caller,omitempty"`
}

var sensitiveKeyPattern = regexp.MustCompile(`(?i)(token|key|secret|password|authorization|credential)`)

// SanitizeFields returns a copy of fields with sensitive values masked.
// Keys matching patterns like token, key, secret, password, authorization,
// or credential (case-insensitive) will have their values replaced with "***".
func SanitizeFields(fields map[string]any) map[string]any {
	if len(fields) == 0 {
		return fields
	}
	sanitized := make(map[string]any, len(fields))
	for k, v := range fields {
		if sensitiveKeyPattern.MatchString(k) {
			sanitized[k] = "***"
		} else {
			sanitized[k] = v
		}
	}
	return sanitized
}

func init() {
	once.Do(func() {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)

		consoleWriter := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "15:04:05", // TODO: make it configurable???
		}

		logger = zerolog.New(consoleWriter).With().Timestamp().Caller().Logger()
		fileLogger = zerolog.Logger{}
		ringBuf = newLogRingBuffer(ringBufSize)
	})
}

func SetLevel(level LogLevel) {
	mu.Lock()
	defer mu.Unlock()
	currentLevel = level
	zerolog.SetGlobalLevel(level)
}

func SetConsoleLevel(level LogLevel) {
	mu.Lock()
	defer mu.Unlock()
	logger = logger.Level(level)
}

func GetLevel() LogLevel {
	mu.RLock()
	defer mu.RUnlock()
	return currentLevel
}

// ParseLevel converts a case-insensitive level name to a LogLevel.
// Returns the level and true if valid, or (INFO, false) if unrecognized.
func ParseLevel(s string) (LogLevel, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return DEBUG, true
	case "info":
		return INFO, true
	case "warn", "warning":
		return WARN, true
	case "error":
		return ERROR, true
	case "fatal":
		return FATAL, true
	default:
		return INFO, false
	}
}

// SetLevelFromString sets the log level from a string value.
// If the string is empty or not a recognized level name, the current level is kept.
func SetLevelFromString(s string) {
	if s == "" {
		return
	}
	if level, ok := ParseLevel(s); ok {
		SetLevel(level)
	}
}

func EnableFileLogging(filePath string) error {
	mu.Lock()
	defer mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	newFile, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	// Close old file if exists
	if logFile != nil {
		logFile.Close()
	}

	logFile = newFile
	fileLogger = zerolog.New(logFile).With().Timestamp().Caller().Logger()
	return nil
}

func DisableFileLogging() {
	mu.Lock()
	defer mu.Unlock()

	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
	fileLogger = zerolog.Logger{}
}

func getCallerSkip() int {
	for i := 2; i < 15; i++ {
		pc, file, _, ok := runtime.Caller(i)
		if !ok {
			continue
		}

		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}

		// bypass common loggers
		if strings.HasSuffix(file, "/logger.go") ||
			strings.HasSuffix(file, "/logger_3rd_party.go") ||
			strings.HasSuffix(file, "/log.go") {
			continue
		}

		funcName := fn.Name()
		if strings.HasPrefix(funcName, "runtime.") {
			continue
		}

		return i - 1
	}

	return 3
}

//nolint:zerologlint
func getEvent(logger zerolog.Logger, level LogLevel) *zerolog.Event {
	switch level {
	case zerolog.DebugLevel:
		return logger.Debug()
	case zerolog.InfoLevel:
		return logger.Info()
	case zerolog.WarnLevel:
		return logger.Warn()
	case zerolog.ErrorLevel:
		return logger.Error()
	case zerolog.FatalLevel:
		return logger.Fatal()
	default:
		return logger.Info()
	}
}

func logMessage(level LogLevel, component string, message string, fields map[string]any) {
	if level < currentLevel {
		return
	}

	// Build LogEntry for ring buffer and subscribers (fork-only)
	entry := LogEntry{
		Level:     logLevelNames[level],
		Timestamp: time.Now().Format(time.RFC3339),
		Component: component,
		Message:   message,
		Fields:    fields,
	}

	// Push to ring buffer and broadcast to subscribers
	ringBuf.push(entry)
	broadcastToSubscribers(entry)

	skip := getCallerSkip()

	event := getEvent(logger, level)

	if component != "" {
		event.Str("component", component)
	}

	appendFields(event, fields)
	event.CallerSkipFrame(skip).Msg(message)

	// Also log to file if enabled
	if fileLogger.GetLevel() != zerolog.NoLevel {
		fileEvent := getEvent(fileLogger, level)

		if component != "" {
			fileEvent.Str("component", component)
		}

		appendFields(fileEvent, fields)
		fileEvent.CallerSkipFrame(skip).Msg(message)
	}

	if level == FATAL {
		os.Exit(1)
	}
}

func appendFields(event *zerolog.Event, fields map[string]any) {
	for k, v := range fields {
		// Type switch to avoid double JSON serialization of strings
		switch val := v.(type) {
		case error:
			event.Str(k, val.Error())
		case string:
			event.Str(k, val)
		case int:
			event.Int(k, val)
		case int64:
			event.Int64(k, val)
		case float64:
			event.Float64(k, val)
		case bool:
			event.Bool(k, val)
		default:
			event.Interface(k, v) // Fallback for struct, slice and maps
		}
	}
}

func Debug(message string) {
	logMessage(DEBUG, "", message, nil)
}

func DebugC(component string, message string) {
	logMessage(DEBUG, component, message, nil)
}

func Debugf(message string, ss ...any) {
	logMessage(DEBUG, "", fmt.Sprintf(message, ss...), nil)
}

func DebugF(message string, fields map[string]any) {
	logMessage(DEBUG, "", message, fields)
}

func DebugCF(component string, message string, fields map[string]any) {
	logMessage(DEBUG, component, message, fields)
}

func Info(message string) {
	logMessage(INFO, "", message, nil)
}

func InfoC(component string, message string) {
	logMessage(INFO, component, message, nil)
}

func InfoF(message string, fields map[string]any) {
	logMessage(INFO, "", message, fields)
}

func Infof(message string, ss ...any) {
	logMessage(INFO, "", fmt.Sprintf(message, ss...), nil)
}

func InfoCF(component string, message string, fields map[string]any) {
	logMessage(INFO, component, message, fields)
}

func Warn(message string) {
	logMessage(WARN, "", message, nil)
}

func WarnC(component string, message string) {
	logMessage(WARN, component, message, nil)
}

func WarnF(message string, fields map[string]any) {
	logMessage(WARN, "", message, fields)
}

func WarnCF(component string, message string, fields map[string]any) {
	logMessage(WARN, component, message, fields)
}

func Error(message string) {
	logMessage(ERROR, "", message, nil)
}

func ErrorC(component string, message string) {
	logMessage(ERROR, component, message, nil)
}

func Errorf(message string, ss ...any) {
	logMessage(ERROR, "", fmt.Sprintf(message, ss...), nil)
}

func ErrorF(message string, fields map[string]any) {
	logMessage(ERROR, "", message, fields)
}

func ErrorCF(component string, message string, fields map[string]any) {
	logMessage(ERROR, component, message, fields)
}

func Fatal(message string) {
	logMessage(FATAL, "", message, nil)
}

func FatalC(component string, message string) {
	logMessage(FATAL, component, message, nil)
}

func Fatalf(message string, ss ...any) {
	logMessage(FATAL, "", fmt.Sprintf(message, ss...), nil)
}

func FatalF(message string, fields map[string]any) {
	logMessage(FATAL, "", message, fields)
}

func FatalCF(component string, message string, fields map[string]any) {
	logMessage(FATAL, component, message, fields)
}

// broadcastToSubscribers sends an entry to all matching subscribers (non-blocking).
func broadcastToSubscribers(entry LogEntry) {
	logSubsMu.Lock()
	subs := make([]*LogSubscriber, len(logSubs))
	copy(subs, logSubs)
	logSubsMu.Unlock()

	for _, sub := range subs {
		if sub.filter != nil && !sub.filter(entry) {
			continue
		}
		select {
		case sub.Ch <- entry:
		default:
			// drop if subscriber channel is full
		}
	}
}

// RecentLogs returns recent log entries from the ring buffer, optionally filtered
// by minimum level and component. The Caller field is stripped for security.
func RecentLogs(minLevel LogLevel, component string, limit int) []LogEntry {
	result := make([]LogEntry, 0, limit)
	ringBuf.visitReverse(func(e *LogEntry) bool {
		if len(result) >= limit {
			return false
		}
		if lvl, ok := levelFromName[e.Level]; ok && lvl < minLevel {
			return true
		}
		if component != "" && e.Component != component {
			return true
		}
		entry := *e
		entry.Caller = ""
		entry.Fields = SanitizeFields(e.Fields)
		result = append(result, entry)
		return true
	})
	// Reverse so oldest first
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return result
}

// Subscribe registers a new log subscriber with an optional filter.
// The returned LogSubscriber's Ch channel has a buffer of 64 entries.
func Subscribe(filter func(LogEntry) bool) *LogSubscriber {
	sub := &LogSubscriber{
		Ch:     make(chan LogEntry, 64),
		filter: filter,
	}
	logSubsMu.Lock()
	logSubs = append(logSubs, sub)
	logSubsMu.Unlock()
	return sub
}

// Unsubscribe removes a subscriber and closes its channel.
func Unsubscribe(sub *LogSubscriber) {
	logSubsMu.Lock()
	for i, s := range logSubs {
		if s == sub {
			logSubs = append(logSubs[:i], logSubs[i+1:]...)
			break
		}
	}
	logSubsMu.Unlock()
	close(sub.Ch)
}
