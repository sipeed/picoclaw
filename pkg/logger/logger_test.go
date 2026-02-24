package logger

import (
	"testing"
)

func TestLogLevelFiltering(t *testing.T) {
	initialLevel := currentLevel
	defer SetLevel(initialLevel)

	SetLevel(WARN)

	tests := []struct {
		name      string
		level     LogLevel
		shouldLog bool
	}{
		{"DEBUG message", DEBUG, false},
		{"INFO message", INFO, false},
		{"WARN message", WARN, true},
		{"ERROR message", ERROR, true},
		{"FATAL message", FATAL, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch tt.level {
			case DEBUG:
				Debug(tt.name)
			case INFO:
				Info(tt.name)
			case WARN:
				Warn(tt.name)
			case ERROR:
				Error(tt.name)
			case FATAL:
				if tt.shouldLog {
					t.Logf("FATAL test skipped to prevent program exit")
				}
			}
		})
	}

	SetLevel(INFO)
}

func TestLoggerWithComponent(t *testing.T) {
	initialLevel := currentLevel
	defer SetLevel(initialLevel)

	SetLevel(DEBUG)

	tests := []struct {
		name      string
		component string
		message   string
		fields    map[string]interface{}
	}{
		{"Simple message", "test", "Hello, world!", nil},
		{"Message with component", "discord", "Discord message", nil},
		{"Message with fields", "telegram", "Telegram message", map[string]interface{}{
			"user_id": "12345",
			"count":   42,
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch {
			case tt.fields == nil && tt.component != "":
				InfoC(tt.component, tt.message)
			case tt.fields != nil:
				InfoF(tt.message, tt.fields)
			default:
				Info(tt.message)
			}
		})
	}

	SetLevel(INFO)
}

func TestLogLevels(t *testing.T) {
	tests := []struct {
		name  string
		level LogLevel
		want  string
	}{
		{"DEBUG level", DEBUG, "DEBUG"},
		{"INFO level", INFO, "INFO"},
		{"WARN level", WARN, "WARN"},
		{"ERROR level", ERROR, "ERROR"},
		{"FATAL level", FATAL, "FATAL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if logLevelNames[tt.level] != tt.want {
				t.Errorf("logLevelNames[%d] = %s, want %s", tt.level, logLevelNames[tt.level], tt.want)
			}
		})
	}
}

func TestLoggerHelperFunctions(t *testing.T) {
	initialLevel := currentLevel
	defer SetLevel(initialLevel)

	SetLevel(INFO)

	Debug("This should not log")
	Info("This should log")
	Warn("This should log")
	Error("This should log")

	InfoC("test", "Component message")
	InfoF("Fields message", map[string]interface{}{"key": "value"})

	WarnC("test", "Warning with component")
	ErrorF("Error with fields", map[string]interface{}{"error": "test"})

	SetLevel(DEBUG)
	DebugC("test", "Debug with component")
	WarnF("Warning with fields", map[string]interface{}{"key": "value"})
}
