package logger

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/rs/zerolog"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// TraceIDKey is the context key for tracing requests
	TraceIDKey contextKey = "traceID"
)

type ErrorCategory string

const (
	ErrorCategoryModelFailure          ErrorCategory = "Model Failure"
	ErrorCategoryInfrastructureFailure ErrorCategory = "Infrastructure Failure"
	ErrorCategoryLogicFailure          ErrorCategory = "Logic Failure"
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

	currentLevel      = INFO
	currentTimeFormat = "15:04:05"
	logger            zerolog.Logger
	fileLogger        zerolog.Logger
	logFile           *os.File
	once              sync.Once
	mu                sync.RWMutex
)

func init() {
	once.Do(func() {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)

		consoleWriter := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: currentTimeFormat,
		}

		logger = zerolog.New(consoleWriter).With().Timestamp().Logger()
		fileLogger = zerolog.Logger{}
	})
}

func SetTimeFormat(format string) {
	mu.Lock()
	defer mu.Unlock()
	currentTimeFormat = format

	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: currentTimeFormat,
	}

	logger = zerolog.New(consoleWriter).With().Timestamp().Logger()
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

func logMessageCtx(ctx context.Context, level LogLevel, component string, message string, fields map[string]any) {
	if level < currentLevel {
		return
	}

	callerFile, callerLine, callerFunc := getCallerInfo()

	event := getEvent(logger, level)

	if ctx != nil {
		if traceID, ok := ctx.Value(TraceIDKey).(string); ok && traceID != "" {
			event.Str("trace_id", traceID)
		}
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
		fileEvent := getEvent(fileLogger, level)

		if ctx != nil {
			if traceID, ok := ctx.Value(TraceIDKey).(string); ok && traceID != "" {
				fileEvent.Str("trace_id", traceID)
			}
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

func logMessage(level LogLevel, component string, message string, fields map[string]any) {
	logMessageCtx(nil, level, component, message, fields)
}

func DebugCtx(ctx context.Context, message string) {
	logMessageCtx(ctx, DEBUG, "", message, nil)
}

func DebugCCtx(ctx context.Context, component string, message string) {
	logMessageCtx(ctx, DEBUG, component, message, nil)
}

func DebugFCtx(ctx context.Context, message string, fields map[string]any) {
	logMessageCtx(ctx, DEBUG, "", message, fields)
}

func DebugCFCtx(ctx context.Context, component string, message string, fields map[string]any) {
	logMessageCtx(ctx, DEBUG, component, message, fields)
}

func InfoCtx(ctx context.Context, message string) {
	logMessageCtx(ctx, INFO, "", message, nil)
}

func InfoCCtx(ctx context.Context, component string, message string) {
	logMessageCtx(ctx, INFO, component, message, nil)
}

func InfoFCtx(ctx context.Context, message string, fields map[string]any) {
	logMessageCtx(ctx, INFO, "", message, fields)
}

func InfoCFCtx(ctx context.Context, component string, message string, fields map[string]any) {
	logMessageCtx(ctx, INFO, component, message, fields)
}

func WarnCtx(ctx context.Context, message string) {
	logMessageCtx(ctx, WARN, "", message, nil)
}

func WarnCCtx(ctx context.Context, component string, message string) {
	logMessageCtx(ctx, WARN, component, message, nil)
}

func WarnFCtx(ctx context.Context, message string, fields map[string]any) {
	logMessageCtx(ctx, WARN, "", message, fields)
}

func WarnCFCtx(ctx context.Context, component string, message string, fields map[string]any) {
	logMessageCtx(ctx, WARN, component, message, fields)
}

func ErrorCtx(ctx context.Context, message string) {
	logMessageCtx(ctx, ERROR, "", message, nil)
}

func ErrorCCtx(ctx context.Context, component string, message string) {
	logMessageCtx(ctx, ERROR, component, message, nil)
}

func ErrorFCtx(ctx context.Context, message string, fields map[string]any) {
	logMessageCtx(ctx, ERROR, "", message, fields)
}

func ErrorCFCtx(ctx context.Context, component string, message string, fields map[string]any) {
	logMessageCtx(ctx, ERROR, component, message, fields)
}

func FatalCtx(ctx context.Context, message string) {
	logMessageCtx(ctx, FATAL, "", message, nil)
}

func FatalCCtx(ctx context.Context, component string, message string) {
	logMessageCtx(ctx, FATAL, component, message, nil)
}

func FatalfCtx(ctx context.Context, message string, ss ...any) {
	logMessageCtx(ctx, FATAL, "", fmt.Sprintf(message, ss...), nil)
}

func FatalFCtx(ctx context.Context, message string, fields map[string]any) {
	logMessageCtx(ctx, FATAL, "", message, fields)
}

func FatalCFCtx(ctx context.Context, component string, message string, fields map[string]any) {
	logMessageCtx(ctx, FATAL, component, message, fields)
}

func LogErrorWithCategory(ctx context.Context, category ErrorCategory, message string, err error) {
	fields := map[string]any{
		"error_category": string(category),
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	logMessageCtx(ctx, ERROR, "", message, fields)
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
