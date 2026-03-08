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
	once         sync.Once
	mu           sync.RWMutex
)

func init() {
	once.Do(func() {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)

		consoleWriter := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "15:04:05",
			NoColor:    false,
		}

		consoleWriter.FormatLevel = func(i interface{}) string {
			level, ok := i.(string)
			if !ok {
				return fmt.Sprintf("| %-5s |", i)
			}

			switch strings.ToUpper(level) {
			case "DEBUG":
				return "| \x1b[36mDBG\x1b[0m |"
			case "INFO":
				return "| \x1b[32mINF\x1b[0m |"
			case "WARN":
				return "| \x1b[33mWRN\x1b[0m |"
			case "ERROR":
				return "| \x1b[31mERR\x1b[0m |"
			case "FATAL":
				return "| \x1b[35mFTL\x1b[0m |"
			default:
				return fmt.Sprintf("| %-5s |", level)
			}
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

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	fileLogger = zerolog.New(file).With().Timestamp().Caller().Logger()
	return nil
}

func DisableFileLogging() {
	mu.Lock()
	defer mu.Unlock()
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
		//event.Str("component", component)
		event.Str("caller", fmt.Sprintf("%-6s | %s:%d (%s)", component, callerFile, callerLine, callerFunc))
	} else {
		event.Str("caller", fmt.Sprintf("<none> | %s:%d (%s)", callerFile, callerLine, callerFunc))
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
func (b *Logger) Debug(v ...interface{}) {
	logMessage(DEBUG, b.component, fmt.Sprint(v...), nil)
}

// Info logs info messages
func (b *Logger) Info(v ...interface{}) {
	logMessage(INFO, b.component, fmt.Sprint(v...), nil)
}

// Warn logs warning messages
func (b *Logger) Warn(v ...interface{}) {
	logMessage(WARN, b.component, fmt.Sprint(v...), nil)
}

// Error logs error messages
func (b *Logger) Error(v ...interface{}) {
	logMessage(ERROR, b.component, fmt.Sprint(v...), nil)
}

// Debugf logs formatted debug messages
func (b *Logger) Debugf(format string, v ...any) {
	//debugCallerInfo()
	logMessage(DEBUG, b.component, fmt.Sprintf(format, v...), nil)
}

// Infof logs formatted info messages
func (b *Logger) Infof(format string, v ...any) {
	//debugCallerInfo()
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
	//debugCallerInfo()
	logMessage(ERROR, b.component, fmt.Sprintf(format, v...), nil)
}

// Fatalf logs formatted fatal messages and exits
func (b *Logger) Fatalf(format string, v ...any) {
	logMessage(FATAL, b.component, fmt.Sprintf(format, v...), nil)
}

// Log logs a message at a given level with caller information
// msgL: message level (DEBUG, INFO, WARN, ERROR, FATAL)
// caller: unused parameter reserved for compatibility
// format: format string
// a: format arguments
func (b *Logger) Log(msgL, caller int, format string, a ...interface{}) {
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

// for debugging logger only
func debugCallerInfo() {
	for i := 2; i < 15; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			continue
		}

		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}

		fmt.Println(file, line)
	}
}
