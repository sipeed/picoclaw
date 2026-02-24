package logger

import (
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

	currentLevel = INFO
	logger       *Logger
	once         sync.Once
	mu           sync.RWMutex
)

type Logger struct{}

type LogEntry struct {
	Level     string                 `json:"level"`
	Timestamp string                 `json:"timestamp"`
	Component string                 `json:"component,omitempty"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Caller    string                 `json:"caller,omitempty"`
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

func logMessage(level LogLevel, component string, message string, fields map[string]interface{}) {
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

func formatFields(fields map[string]interface{}) string {
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

func DebugF(message string, fields map[string]interface{}) {
	logMessage(DEBUG, "", message, fields)
}

func DebugCF(component string, message string, fields map[string]interface{}) {
	logMessage(DEBUG, component, message, fields)
}

func Info(message string) {
	logMessage(INFO, "", message, nil)
}

func InfoC(component string, message string) {
	logMessage(INFO, component, message, nil)
}

func InfoF(message string, fields map[string]interface{}) {
	logMessage(INFO, "", message, fields)
}

func InfoCF(component string, message string, fields map[string]interface{}) {
	logMessage(INFO, component, message, fields)
}

func Warn(message string) {
	logMessage(WARN, "", message, nil)
}

func WarnC(component string, message string) {
	logMessage(WARN, component, message, nil)
}

func WarnF(message string, fields map[string]interface{}) {
	logMessage(WARN, "", message, fields)
}

func WarnCF(component string, message string, fields map[string]interface{}) {
	logMessage(WARN, component, message, fields)
}

func Error(message string) {
	logMessage(ERROR, "", message, nil)
}

func ErrorC(component string, message string) {
	logMessage(ERROR, component, message, nil)
}

func ErrorF(message string, fields map[string]interface{}) {
	logMessage(ERROR, "", message, fields)
}

func ErrorCF(component string, message string, fields map[string]interface{}) {
	logMessage(ERROR, component, message, fields)
}

func Fatal(message string) {
	logMessage(FATAL, "", message, nil)
}

func FatalC(component string, message string) {
	logMessage(FATAL, component, message, nil)
}

func FatalF(message string, fields map[string]interface{}) {
	logMessage(FATAL, "", message, fields)
}

func FatalCF(component string, message string, fields map[string]interface{}) {
	logMessage(FATAL, component, message, fields)
}
