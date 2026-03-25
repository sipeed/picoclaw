package logger

import (
	"testing"
	"time"
)

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

func TestSanitizeFields(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]any
		maskedK []string // keys that should be "***"
		safeK   []string // keys that should keep original value
	}{
		{
			name:    "nil fields",
			input:   nil,
			maskedK: nil,
		},
		{
			name:    "empty fields",
			input:   map[string]any{},
			maskedK: nil,
		},
		{
			name: "sensitive keys masked",
			input: map[string]any{
				"token":         "abc123",
				"api_key":       "sk-xxx",
				"secret":        "s3cr3t",
				"password":      "pass",
				"authorization": "Bearer tok",
			},
			maskedK: []string{"token", "api_key", "secret", "password", "authorization"},
		},
		{
			name: "case insensitive",
			input: map[string]any{
				"Token":         "abc",
				"API_KEY":       "xyz",
				"Secret":        "s",
				"PASSWORD":      "p",
				"Authorization": "a",
				"Credential":    "c",
			},
			maskedK: []string{"Token", "API_KEY", "Secret", "PASSWORD", "Authorization", "Credential"},
		},
		{
			name:  "safe keys preserved",
			input: map[string]any{"error": "something failed", "count": 42, "user_id": "12345", "component": "test"},
			safeK: []string{"error", "count", "user_id", "component"},
		},
		{
			name: "mixed keys",
			input: map[string]any{
				"token":         "sensitive",
				"msg_signature": "safe",
				"corp_secret":   "sensitive2",
				"nonce":         "safe2",
			},
			maskedK: []string{"token", "corp_secret"},
			safeK:   []string{"msg_signature", "nonce"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeFields(tt.input)
			for _, k := range tt.maskedK {
				if v, ok := result[k]; !ok || v != "***" {
					t.Errorf("expected key %q to be masked, got %v", k, v)
				}
			}
			for _, k := range tt.safeK {
				if result[k] != tt.input[k] {
					t.Errorf("expected key %q to be preserved as %v, got %v", k, tt.input[k], result[k])
				}
			}
		})
	}
}

func TestSanitizeFieldsDoesNotMutateOriginal(t *testing.T) {
	original := map[string]any{"token": "secret_value", "name": "test"}
	_ = SanitizeFields(original)
	if original["token"] != "secret_value" {
		t.Error("SanitizeFields should not mutate the original map")
	}
}

func TestRecentLogsSanitizesFields(t *testing.T) {
	initialLevel := GetLevel()
	defer SetLevel(initialLevel)
	SetLevel(DEBUG)

	InfoCF("sanitize-test", "log with sensitive fields", map[string]any{
		"token":   "my-secret-token",
		"api_key": "sk-12345",
		"user_id": "safe-value",
	})

	got := RecentLogs(DEBUG, "sanitize-test", 100)
	if len(got) == 0 {
		t.Fatal("expected at least one log entry")
	}

	last := got[len(got)-1]
	if last.Fields["token"] != "***" {
		t.Errorf("expected token to be masked, got %v", last.Fields["token"])
	}
	if last.Fields["api_key"] != "***" {
		t.Errorf("expected api_key to be masked, got %v", last.Fields["api_key"])
	}
	if last.Fields["user_id"] != "safe-value" {
		t.Errorf("expected user_id to be preserved, got %v", last.Fields["user_id"])
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
		got, _ := ParseLevel(tt.input)
		if got != tt.want {
			t.Errorf("ParseLevel(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
