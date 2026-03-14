package logger

import (
	"testing"
)

func TestLogLevelFiltering(t *testing.T) {
	initialLevel := GetLevel()
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
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)

	SetLevel(DEBUG)

	tests := []struct {
		name      string
		component string
		message   string
		fields    map[string]any
	}{
		{"Simple message", "test", "Hello, world!", nil},
		{"Message with component", "discord", "Discord message", nil},
		{"Message with fields", "telegram", "Telegram message", map[string]any{
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

func TestSetGetLevel(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)

	tests := []LogLevel{DEBUG, INFO, WARN, ERROR, FATAL}

	for _, level := range tests {
		SetLevel(level)
		if GetLevel() != level {
			t.Errorf("SetLevel(%v) -> GetLevel() = %v, want %v", level, GetLevel(), level)
		}
	}
}

func TestLoggerHelperFunctions(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)

	SetLevel(INFO)

	Debug("This should not log")
	Debugf("this should not log")
	Info("This should log")
	Warn("This should log")
	Error("This should log")

	InfoC("test", "Component message")
	InfoF("Fields message", map[string]any{"key": "value"})
	Infof("test from %v", "Infof")

	WarnC("test", "Warning with component")
	ErrorF("Error with fields", map[string]any{"error": "test"})
	Errorf("test from %v", "Errorf")

	SetLevel(DEBUG)
	DebugC("test", "Debug with component")
	Debugf("test from %v", "Debugf")
	WarnF("Warning with fields", map[string]any{"key": "value"})
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantLevel LogLevel
		wantOk    bool
	}{
		// Valid cases - uppercase
		{"DEBUG uppercase", "DEBUG", DEBUG, true},
		{"INFO uppercase", "INFO", INFO, true},
		{"WARN uppercase", "WARN", WARN, true},
		{"WARNING uppercase", "WARNING", WARN, true},
		{"ERROR uppercase", "ERROR", ERROR, true},
		{"FATAL uppercase", "FATAL", FATAL, true},

		// Valid cases - lowercase
		{"DEBUG lowercase", "debug", DEBUG, true},
		{"INFO lowercase", "info", INFO, true},
		{"WARN lowercase", "warn", WARN, true},
		{"WARNING lowercase", "warning", WARN, true},
		{"ERROR lowercase", "error", ERROR, true},
		{"FATAL lowercase", "fatal", FATAL, true},

		// Valid cases - mixed case
		{"Debug mixed case", "Debug", DEBUG, true},
		{"Info mixed case", "InFo", INFO, true},
		{"Warn mixed case", "WaRn", WARN, true},
		{"Error mixed case", "ErRoR", ERROR, true},

		// Invalid cases
		{"empty string", "", INFO, false},
		{"unknown value", "TRACE", INFO, false},
		{"unknown value 2", "VERBOSE", INFO, false},
		{"invalid value", "invalid", INFO, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level, ok := parseLevel(tt.input)
			if ok != tt.wantOk {
				t.Errorf("parseLevel(%q) ok = %v, want %v", tt.input, ok, tt.wantOk)
			}
			if level != tt.wantLevel {
				t.Errorf("parseLevel(%q) level = %v, want %v", tt.input, level, tt.wantLevel)
			}
		})
	}
}
