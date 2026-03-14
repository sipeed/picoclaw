package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
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

			// Custom formatter to handle multiline strings and JSON objects
			FormatFieldValue: formatFieldValue,
		}

		logger = zerolog.New(consoleWriter).With().Timestamp().Logger()
		fileLogger = zerolog.Logger{}
	})
}

func formatFieldValue(i any) string {
	var s string

	switch val := i.(type) {
	case string:
		s = val
	case []byte:
		s = string(val)
	default:
		return fmt.Sprintf("%v", i)
	}

	if unquoted, err := strconv.Unquote(s); err == nil {
		s = unquoted
	}

	if strings.Contains(s, "\n") {
		return fmt.Sprintf("\n%s", s)
	}

	if strings.Contains(s, " ") {
		if (strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}")) ||
			(strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]")) {
			return s
		}
		return fmt.Sprintf("%q", s)
	}

	return s
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
			strings.HasSuffix(file, "/logger_3rd_party.go") ||
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

	callerFile, callerLine, callerFunc := getCallerInfo()

	event := getEvent(logger, level)

	// Build combined field with component and caller
	if component != "" {
		event.Str("caller", fmt.Sprintf("%-6s %s:%d (%s)", component, callerFile, callerLine, callerFunc))
	} else {
		event.Str("caller", fmt.Sprintf("<none> %s:%d (%s)", callerFile, callerLine, callerFunc))
	}

	appendFields(event, fields)
	event.Msg(message)

	// Also log to file if enabled
	if fileLogger.GetLevel() != zerolog.NoLevel {
		fileEvent := getEvent(fileLogger, level)

		if component != "" {
			fileEvent.Str("component", component)
		}

		appendFields(event, fields)
		fileEvent.Msg(message)
	}

	if level == FATAL {
		os.Exit(1)
	}
}

func appendFields(event *zerolog.Event, fields map[string]any) {
	for k, v := range fields {
		// Type switch to avoid double JSON serialization of strings
		switch val := v.(type) {
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
