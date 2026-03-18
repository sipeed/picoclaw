// this file is for compatible with 3rd party loggers, should not be called in PicoClaw project

package logger

import (
	"fmt"
	"regexp"
)

// botTokenRe matches the bot ID prefix and the secret part of a Telegram bot token.
// Groups: 1 = "bot<id>:", 2 = first 4 chars of secret, 3 = middle, 4 = last 4 chars.
var botTokenRe = regexp.MustCompile(`(bot\d+:)([A-Za-z0-9_-]{4})[A-Za-z0-9_-]{12,}([A-Za-z0-9_-]{4})`)

// maskSecrets replaces any embedded bot tokens in s with a redacted placeholder
// that keeps the first and last 4 characters of the secret for identification.
func maskSecrets(s string) string {
	return botTokenRe.ReplaceAllString(s, "${1}${2}****${3}")
}

// Logger implements common Logger interface
type Logger struct {
	component string
	levels    map[int]LogLevel
}

// Debug logs debug messages
func (b *Logger) Debug(v ...any) {
	logMessage(DEBUG, b.component, maskSecrets(fmt.Sprint(v...)), nil)
}

// Info logs info messages
func (b *Logger) Info(v ...any) {
	logMessage(INFO, b.component, maskSecrets(fmt.Sprint(v...)), nil)
}

// Warn logs warning messages
func (b *Logger) Warn(v ...any) {
	logMessage(WARN, b.component, maskSecrets(fmt.Sprint(v...)), nil)
}

// Error logs error messages
func (b *Logger) Error(v ...any) {
	logMessage(ERROR, b.component, maskSecrets(fmt.Sprint(v...)), nil)
}

// Debugf logs formatted debug messages
func (b *Logger) Debugf(format string, v ...any) {
	logMessage(DEBUG, b.component, maskSecrets(fmt.Sprintf(format, v...)), nil)
}

// Infof logs formatted info messages
func (b *Logger) Infof(format string, v ...any) {
	logMessage(INFO, b.component, maskSecrets(fmt.Sprintf(format, v...)), nil)
}

// Warnf logs formatted warning messages
func (b *Logger) Warnf(format string, v ...any) {
	logMessage(WARN, b.component, maskSecrets(fmt.Sprintf(format, v...)), nil)
}

// Warningf logs formatted warning messages
func (b *Logger) Warningf(format string, v ...any) {
	logMessage(WARN, b.component, maskSecrets(fmt.Sprintf(format, v...)), nil)
}

// Errorf logs formatted error messages
func (b *Logger) Errorf(format string, v ...any) {
	logMessage(ERROR, b.component, maskSecrets(fmt.Sprintf(format, v...)), nil)
}

// Fatalf logs formatted fatal messages and exits
func (b *Logger) Fatalf(format string, v ...any) {
	logMessage(FATAL, b.component, maskSecrets(fmt.Sprintf(format, v...)), nil)
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
	logMessage(level, b.component, maskSecrets(fmt.Sprintf(format, a...)), nil)
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
