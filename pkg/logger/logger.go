package logger

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
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
	logger       *Logger
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

// LogSubscriber receives log entries matching its filter.
type LogSubscriber struct {
	Ch     chan LogEntry
	filter func(LogEntry) bool
}

type Logger struct {
	file *os.File
}

type LogEntry struct {
	Level     string         `json:"level"`
	Timestamp string         `json:"timestamp"`
	Component string         `json:"component,omitempty"`
	Message   string         `json:"message"`
	Fields    map[string]any `json:"fields,omitempty"`
	Caller    string         `json:"caller,omitempty"`
}

func init() {
	once.Do(func() {
		logger = &Logger{}
		ringBuf = newLogRingBuffer(ringBufSize)
	})
}

func SetLevel(level LogLevel) {
	mu.Lock()
	defer mu.Unlock()
	currentLevel = level
}

func GetLevel() LogLevel {
	mu.RLock()
	defer mu.RUnlock()
	return currentLevel
}

func EnableFileLogging(filePath string) error {
	mu.Lock()
	defer mu.Unlock()

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	if logger.file != nil {
		logger.file.Close()
	}

	logger.file = file
	log.Println("File logging enabled:", filePath)
	return nil
}

func DisableFileLogging() {
	mu.Lock()
	defer mu.Unlock()

	if logger.file != nil {
		logger.file.Close()
		logger.file = nil
		log.Println("File logging disabled")
	}
}

func logMessage(level LogLevel, component string, message string, fields map[string]any) {
	if level < currentLevel {
		return
	}

	entry := LogEntry{
		Level:     logLevelNames[level],
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Component: component,
		Message:   message,
		Fields:    fields,
	}

	if pc, file, line, ok := runtime.Caller(2); ok {
		fn := runtime.FuncForPC(pc)
		if fn != nil {
			entry.Caller = fmt.Sprintf("%s:%d (%s)", file, line, fn.Name())
		}
	}

	// Push to ring buffer and broadcast to subscribers
	ringBuf.push(entry)
	broadcastToSubscribers(entry)

	if logger.file != nil {
		jsonData, err := json.Marshal(entry)
		if err == nil {
			logger.file.WriteString(string(jsonData) + "\n")
		}
	}

	var fieldStr string
	if len(fields) > 0 {
		fieldStr = " " + formatFields(fields)
	}

	logLine := fmt.Sprintf("[%s] [%s]%s %s%s",
		entry.Timestamp,
		logLevelNames[level],
		formatComponent(component),
		message,
		fieldStr,
	)

	log.Println(logLine)

	if level == FATAL {
		os.Exit(1)
	}
}

func formatComponent(component string) string {
	if component == "" {
		return ""
	}
	return fmt.Sprintf(" %s:", component)
}

func formatFields(fields map[string]any) string {
	var parts []string
	for k, v := range fields {
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}
	return fmt.Sprintf("{%s}", strings.Join(parts, ", "))
}

func Debug(message string) {
	logMessage(DEBUG, "", message, nil)
}

func DebugC(component string, message string) {
	logMessage(DEBUG, component, message, nil)
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
	all := ringBuf.recent(0) // get all
	result := make([]LogEntry, 0, limit)
	for i := len(all) - 1; i >= 0 && len(result) < limit; i-- {
		e := all[i]
		if lvl, ok := levelFromName[e.Level]; ok && lvl < minLevel {
			continue
		}
		if component != "" && e.Component != component {
			continue
		}
		e.Caller = "" // strip for security
		result = append(result, e)
	}
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

// ParseLevel converts a level name string to a LogLevel.
// Returns INFO if the string is not recognized.
func ParseLevel(s string) LogLevel {
	s = strings.ToUpper(strings.TrimSpace(s))
	if lvl, ok := levelFromName[s]; ok {
		return lvl
	}
	return INFO
}
