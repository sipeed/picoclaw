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

	"github.com/sipeed/picoclaw/pkg/redaction"
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

	currentLevel = INFO
	logger       *Logger
	once         sync.Once
	mu           sync.RWMutex

	// redactionEnabled controls whether log messages are redacted for privacy
	redactionEnabled = true
)

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

	// Apply redaction to message and fields for privacy
	if redactionEnabled {
		message = redaction.Redact(message)
		if fields != nil {
			fields = redaction.RedactFields(fields)
		}
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

	if logger.file != nil {
		jsonData, err := json.Marshal(entry)
		if err == nil {
			logger.file.Write(append(jsonData, '\n'))
		}
	}

	var fieldStr string
	if len(fields) > 0 {
		fieldStr = " " + formatFields(fields)
	} else {
		fieldStr = ""
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
	parts := make([]string, 0, len(fields))
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

// SetRedactionEnabled enables or disables log redaction for privacy.
func SetRedactionEnabled(enabled bool) {
	mu.Lock()
	defer mu.Unlock()
	redactionEnabled = enabled
}

// IsRedactionEnabled returns whether log redaction is enabled.
func IsRedactionEnabled() bool {
	mu.RLock()
	defer mu.RUnlock()
	return redactionEnabled
}

// ConfigureRedaction sets up the global redaction configuration.
func ConfigureRedaction(config redaction.Config) {
	redaction.SetGlobalConfig(config)
}
