package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

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

	currentLevel = INFO
	logger       zerolog.Logger
	fileLogger   zerolog.Logger
	logFile      *os.File
	once         sync.Once
	mu           sync.RWMutex
)

func init() {
	once.Do(func() {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)

		consoleWriter := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "15:04:05", // TODO: make it configurable???
		}

		logger = zerolog.New(consoleWriter).With().Timestamp().Logger()
		fileLogger = zerolog.Logger{}
	})
}

func SetLevel(level LogLevel) {
	mu.Lock()
	defer mu.Unlock()
	currentLevel = level
	zerolog.SetGlobalLevel(level)
}

func GetLevel() LogLevel {
	mu.RLock()
	defer mu.RUnlock()
	return currentLevel
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

func getCallerInfo() (string, int, string) {
	for i := 2; i < 15; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			continue
		}

		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}

		// bypass common loggers
		if strings.HasSuffix(file, "/logger.go") ||
			strings.HasSuffix(file, "/log.go") {
			continue
		}

		funcName := fn.Name()
		if strings.HasPrefix(funcName, "runtime.") {
			continue
		}

		return filepath.Base(file), line, filepath.Base(funcName)
	}

	return "???", 0, "???"
}

func logMessage(level LogLevel, component string, message string, fields map[string]any) {
	if level < currentLevel {
		return
	}

	callerFile, callerLine, callerFunc := getCallerInfo()

	var event *zerolog.Event
	switch level {
	case zerolog.DebugLevel:
		event = logger.Debug()
	case zerolog.InfoLevel:
		event = logger.Info()
	case zerolog.WarnLevel:
		event = logger.Warn()
	case zerolog.ErrorLevel:
		event = logger.Error()
	case zerolog.FatalLevel:
		event = logger.Error()
	default:
		event = logger.Info()
	}

	// Build combined field with component and caller
	if component != "" {
		event.Str("caller", fmt.Sprintf("%-6s %s:%d (%s)", component, callerFile, callerLine, callerFunc))
	} else {
		event.Str("caller", fmt.Sprintf("<none> %s:%d (%s)", callerFile, callerLine, callerFunc))
	}

	for k, v := range fields {
		event.Interface(k, v)
	}

	event.Msg(message)

	// Also log to file if enabled
	if fileLogger.GetLevel() != zerolog.NoLevel {
		var fileEvent *zerolog.Event
		switch level {
		case zerolog.DebugLevel:
			fileEvent = fileLogger.Debug()
		case zerolog.InfoLevel:
			fileEvent = fileLogger.Info()
		case zerolog.WarnLevel:
			fileEvent = fileLogger.Warn()
		case zerolog.ErrorLevel:
			fileEvent = fileLogger.Error()
		case zerolog.FatalLevel:
			fileEvent = fileLogger.Error()
		default:
			fileEvent = fileLogger.Info()
		}

		if component != "" {
			fileEvent.Str("component", component)
		}
		for k, v := range fields {
			fileEvent.Interface(k, v)
		}
		fileEvent.Msg(message)
	}

	if level == FATAL {
		os.Exit(1)
	}
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

func Fatalf(message string, ss ...any) {
	logMessage(FATAL, "", fmt.Sprintf(message, ss...), nil)
}

func FatalF(message string, fields map[string]any) {
	logMessage(FATAL, "", message, fields)
}

func FatalCF(component string, message string, fields map[string]any) {
	logMessage(FATAL, component, message, fields)
}

// Logger implements common Logger interface
type Logger struct {
	component string
	levels    map[int]LogLevel
}

// Debug logs debug messages
func (b *Logger) Debug(v ...any) {
	logMessage(DEBUG, b.component, fmt.Sprint(v...), nil)
}

// Info logs info messages
func (b *Logger) Info(v ...any) {
	logMessage(INFO, b.component, fmt.Sprint(v...), nil)
}

// Warn logs warning messages
func (b *Logger) Warn(v ...any) {
	logMessage(WARN, b.component, fmt.Sprint(v...), nil)
}

// Error logs error messages
func (b *Logger) Error(v ...any) {
	logMessage(ERROR, b.component, fmt.Sprint(v...), nil)
}

// Debugf logs formatted debug messages
func (b *Logger) Debugf(format string, v ...any) {
	logMessage(DEBUG, b.component, fmt.Sprintf(format, v...), nil)
}

// Infof logs formatted info messages
func (b *Logger) Infof(format string, v ...any) {
	logMessage(INFO, b.component, fmt.Sprintf(format, v...), nil)
}

// Warnf logs formatted warning messages
func (b *Logger) Warnf(format string, v ...any) {
	logMessage(WARN, b.component, fmt.Sprintf(format, v...), nil)
}

// Warningf logs formatted warning messages
func (b *Logger) Warningf(format string, v ...any) {
	logMessage(WARN, b.component, fmt.Sprintf(format, v...), nil)
}

// Errorf logs formatted error messages
func (b *Logger) Errorf(format string, v ...any) {
	logMessage(ERROR, b.component, fmt.Sprintf(format, v...), nil)
}

// Fatalf logs formatted fatal messages and exits
func (b *Logger) Fatalf(format string, v ...any) {
	logMessage(FATAL, b.component, fmt.Sprintf(format, v...), nil)
}

// Log logs a message at a given level with caller information
// the func name must be this because 3rd party loggers expect this
// msgL: message level (DEBUG, INFO, WARN, ERROR, FATAL)
// caller: unused parameter reserved for compatibility
// format: format string
// a: format arguments
//
//nolint:goprintffuncname
func (b *Logger) Log(msgL, caller int, format string, a ...any) {
	level := LogLevel(msgL)
	if b.levels != nil {
		if lvl, ok := b.levels[msgL]; ok {
			level = lvl
		}
	}
	logMessage(level, b.component, fmt.Sprintf(format, a...), nil)
}

// Sync flushes log buffer (no-op for this implementation)
func (b *Logger) Sync() error {
	return nil
}

// WithLevels sets log levels mapping for this logger
func (b *Logger) WithLevels(levels map[int]LogLevel) *Logger {
	b.levels = levels
	return b
}

// NewLogger creates a new logger instance with optional component name
func NewLogger(component string) *Logger {
	return &Logger{component: component}
}
