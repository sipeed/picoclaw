package logger

import (
	"testing"
	"time"
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
	Info("This should log")
	Warn("This should log")
	Error("This should log")

	InfoC("test", "Component message")
	InfoF("Fields message", map[string]any{"key": "value"})

	WarnC("test", "Warning with component")
	ErrorF("Error with fields", map[string]any{"error": "test"})

	SetLevel(DEBUG)
	DebugC("test", "Debug with component")
	WarnF("Warning with fields", map[string]any{"key": "value"})
}

// ── Ring buffer tests ──

func TestRingBuffer_PushAndRecent(t *testing.T) {
	rb := newLogRingBuffer(5)

	for i := 0; i < 3; i++ {
		rb.push(LogEntry{Message: "msg" + string(rune('A'+i))})
	}

	got := rb.recent(0)
	if len(got) != 3 {
		t.Fatalf("expected 3, got %d", len(got))
	}
	if got[0].Message != "msgA" || got[2].Message != "msgC" {
		t.Errorf("unexpected order: %v", got)
	}
}

func TestRingBuffer_Wrap(t *testing.T) {
	rb := newLogRingBuffer(3)
	for i := 0; i < 5; i++ {
		rb.push(LogEntry{Message: string(rune('A' + i))})
	}

	got := rb.recent(0)
	if len(got) != 3 {
		t.Fatalf("expected 3, got %d", len(got))
	}
	// Should have C, D, E (oldest two dropped)
	if got[0].Message != "C" || got[1].Message != "D" || got[2].Message != "E" {
		t.Errorf("expected [C,D,E], got [%s,%s,%s]", got[0].Message, got[1].Message, got[2].Message)
	}
}

func TestRingBuffer_RecentLimit(t *testing.T) {
	rb := newLogRingBuffer(10)
	for i := 0; i < 8; i++ {
		rb.push(LogEntry{Message: string(rune('A' + i))})
	}

	got := rb.recent(3)
	if len(got) != 3 {
		t.Fatalf("expected 3, got %d", len(got))
	}
	if got[0].Message != "F" || got[2].Message != "H" {
		t.Errorf("expected last 3 entries, got %v", got)
	}
}

func TestRecentLogs_FilterByLevel(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(DEBUG)

	// Log messages at different levels
	DebugC("test", "debug msg")
	InfoC("test", "info msg")
	WarnC("test", "warn msg")
	ErrorC("test", "error msg")

	got := RecentLogs(WARN, "", 100)
	for _, e := range got {
		if e.Level == "DEBUG" || e.Level == "INFO" {
			t.Errorf("unexpected level %s in result with minLevel=WARN", e.Level)
		}
	}
}

func TestRecentLogs_FilterByComponent(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(DEBUG)

	InfoC("alpha", "from alpha")
	InfoC("beta", "from beta")
	InfoC("alpha", "another from alpha")

	got := RecentLogs(DEBUG, "alpha", 100)
	for _, e := range got {
		if e.Component != "alpha" {
			t.Errorf("unexpected component %s in result with component=alpha", e.Component)
		}
	}
}

func TestRecentLogs_CallerStripped(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(DEBUG)

	InfoC("test", "caller test")

	got := RecentLogs(DEBUG, "", 100)
	for _, e := range got {
		if e.Caller != "" {
			t.Errorf("Caller should be stripped, got %q", e.Caller)
		}
	}
}

func TestSubscribe_ReceivesEntries(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(DEBUG)

	sub := Subscribe(nil)
	defer Unsubscribe(sub)

	InfoC("sub-test", "hello subscriber")

	select {
	case entry := <-sub.Ch:
		if entry.Message != "hello subscriber" {
			t.Errorf("expected 'hello subscriber', got %q", entry.Message)
		}
	case <-time.After(time.Second):
		t.Error("timed out waiting for log entry")
	}
}

func TestSubscribe_FilterApplied(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(DEBUG)

	sub := Subscribe(func(e LogEntry) bool {
		return e.Component == "target"
	})
	defer Unsubscribe(sub)

	InfoC("other", "should be filtered out")
	InfoC("target", "should arrive")

	select {
	case entry := <-sub.Ch:
		if entry.Component != "target" {
			t.Errorf("expected component=target, got %q", entry.Component)
		}
	case <-time.After(time.Second):
		t.Error("timed out waiting for filtered entry")
	}
}

func TestUnsubscribe_ClosesChannel(t *testing.T) {
	sub := Subscribe(nil)
	Unsubscribe(sub)

	_, ok := <-sub.Ch
	if ok {
		t.Error("expected channel to be closed after Unsubscribe")
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input string
		want  LogLevel
	}{
		{"debug", DEBUG},
		{"DEBUG", DEBUG},
		{"info", INFO},
		{"WARN", WARN},
		{"error", ERROR},
		{"fatal", FATAL},
		{"unknown", INFO},
		{"", INFO},
	}
	for _, tt := range tests {
		got := ParseLevel(tt.input)
		if got != tt.want {
			t.Errorf("ParseLevel(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
