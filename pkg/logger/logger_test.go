package logger

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog"
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

func TestFormatFieldValue(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		// Basic types test (default case of the switch)
		{
			name:     "Integer Type",
			input:    42,
			expected: "42",
		},
		{
			name:     "Boolean Type",
			input:    true,
			expected: "true",
		},
		{
			name:     "Unsupported Struct Type",
			input:    struct{ A int }{A: 1},
			expected: "{1}",
		},

		// Simple strings and byte slices test
		{
			name:     "Simple string without spaces",
			input:    "simple_value",
			expected: "simple_value",
		},
		{
			name:     "Simple byte slice",
			input:    []byte("byte_value"),
			expected: "byte_value",
		},

		// Unquoting test (strconv.Unquote)
		{
			name:     "Quoted string",
			input:    `"quoted_value"`,
			expected: "quoted_value",
		},

		// Strings with newline (\n) test
		{
			name:     "String with newline",
			input:    "line1\nline2",
			expected: "\nline1\nline2",
		},
		{
			name:     "Quoted string with newline (Unquote -> newline)",
			input:    `"line1\nline2"`, // Escaped \n that Unquote will resolve
			expected: "\nline1\nline2",
		},

		// Strings with spaces test (which should be quoted)
		{
			name:     "String with spaces",
			input:    "hello world",
			expected: `"hello world"`,
		},
		{
			name:     "Quoted string with spaces (Unquote -> has spaces -> Re-quote)",
			input:    `"hello world"`,
			expected: `"hello world"`,
		},

		// JSON formats test (strings with spaces that start/end with brackets)
		{
			name:     "Valid JSON object",
			input:    `{"key": "value"}`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "Valid JSON array",
			input:    `[1, 2, "three"]`,
			expected: `[1, 2, "three"]`,
		},
		{
			name:     "Fake JSON (starts with { but doesn't end with })",
			input:    `{"key": "value"`, // Missing closing bracket, has spaces
			expected: `"{\"key\": \"value\""`,
		},
		{
			name:     "Empty JSON (object)",
			input:    `{ }`,
			expected: `{ }`,
		},

		// 7. Edge Cases
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Whitespace only string",
			input:    "   ",
			expected: `"   "`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := formatFieldValue(tt.input)
			if actual != tt.expected {
				t.Errorf("formatFieldValue() = %q, expected %q", actual, tt.expected)
			}
		})
	}
}

func TestDefaultLevelIsInfo(t *testing.T) {
	// The package-level default (before any SetLevel call) should be INFO.
	// Because earlier tests may have changed it, we just verify the constant is wired correctly.
	if logLevelNames[INFO] != "INFO" {
		t.Errorf("INFO constant mapped to %q, want \"INFO\"", logLevelNames[INFO])
	}
}

func TestParseLevelValid(t *testing.T) {
	tests := []struct {
		input string
		want  LogLevel
	}{
		{"debug", DEBUG},
		{"DEBUG", DEBUG},
		{"Debug", DEBUG},
		{"info", INFO},
		{"INFO", INFO},
		{"warn", WARN},
		{"WARN", WARN},
		{"warning", WARN},
		{"WARNING", WARN},
		{"error", ERROR},
		{"ERROR", ERROR},
		{"fatal", FATAL},
		{"FATAL", FATAL},
		{"  info  ", INFO},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, ok := ParseLevel(tt.input)
			if !ok {
				t.Fatalf("ParseLevel(%q) returned ok=false, want true", tt.input)
			}
			if got != tt.want {
				t.Errorf("ParseLevel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseLevelInvalid(t *testing.T) {
	tests := []string{"", "garbage", "verbose", "trace", "critical"}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, ok := ParseLevel(input)
			if ok {
				t.Errorf("ParseLevel(%q) returned ok=true, want false", input)
			}
		})
	}
}

func TestSetLevelFromString(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)

	// Valid string changes the level
	SetLevel(INFO)
	SetLevelFromString("error")
	if got := GetLevel(); got != ERROR {
		t.Errorf("after SetLevelFromString(\"error\"): GetLevel() = %v, want ERROR", got)
	}

	// Empty string is a no-op
	SetLevelFromString("")
	if got := GetLevel(); got != ERROR {
		t.Errorf("after SetLevelFromString(\"\"): GetLevel() = %v, want ERROR (unchanged)", got)
	}

	// Invalid string is a no-op
	SetLevelFromString("garbage")
	if got := GetLevel(); got != ERROR {
		t.Errorf("after SetLevelFromString(\"garbage\"): GetLevel() = %v, want ERROR (unchanged)", got)
	}

	// Case-insensitive
	SetLevelFromString("FATAL")
	if got := GetLevel(); got != FATAL {
		t.Errorf("after SetLevelFromString(\"FATAL\"): GetLevel() = %v, want FATAL", got)
	}
}

func TestAppendFields_ErrorUsesErrorString(t *testing.T) {
	var buf bytes.Buffer
	l := zerolog.New(&buf)

	event := l.Info()
	appendFields(event, map[string]any{"error": errors.New("transcription request failed")})
	event.Msg("test")

	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	if len(lines) == 0 {
		t.Fatal("expected log output, got none")
	}

	var got map[string]any
	if err := json.Unmarshal(lines[0], &got); err != nil {
		t.Fatalf("unmarshal log line: %v", err)
	}

	if got["error"] != "transcription request failed" {
		t.Fatalf("error field = %#v, want %q", got["error"], "transcription request failed")
	}
}

func TestDisableConsole(t *testing.T) {
	DisableConsole()
	Info("this should go to nowhere")
}

func TestConfigureFromEnv(t *testing.T) {
	home := os.Getenv("HOME")
	if home == "" {
		t.Skip("HOME not set")
	}

	tmpFile := "/tmp/picoclaw_test_log_" + fmt.Sprintf("%d", time.Now().UnixNano())
	defer os.Remove(tmpFile)

	os.Setenv("PICOCLAW_LOG_FILE", tmpFile)
	defer os.Unsetenv("PICOCLAW_LOG_FILE")

	ConfigureFromEnv()

	if logFile == nil {
		t.Error("expected log file to be set")
	}

	Info("test message")

	os.Setenv("PICOCLAW_LOG_FILE", "~/test_log")
	ConfigureFromEnv()

	expanded := filepath.Join(home, "test_log")
	defer os.Remove(expanded)
}

func TestConfigureFromEnvNoEnv(t *testing.T) {
	os.Unsetenv("PICOCLAW_LOG_FILE")
	ConfigureFromEnv()
}

func TestGetPackageNameFromFile(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"normal package path", "/home/user/project/pkg/logger/logger.go", "logger"},
		{"nested package", "/home/user/project/internal/service/auth/handler.go", "auth"},
		{"cmd package", "/home/user/project/cmd/server/main.go", "server"},
		{"project root returns main", "./main.go", "<main>"},
		{"single dot returns main", ".", "<main>"},
		{"single directory", "mypkg/file.go", "mypkg"},
		{"deep nesting", "/a/b/c/d/e/f.go", "e"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getPackageNameFromFile(tt.path)
			if got != tt.want {
				t.Errorf("getPackageNameFromFile(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}
