package tools

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestParseFeishuDateTime(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("failed to load location: %v", err)
	}

	tests := []struct {
		name        string
		input       string
		expectHasTZ bool
		expectUnix  int64
	}{
		{
			name:        "unix timestamp",
			input:       "1739935200",
			expectHasTZ: false,
			expectUnix:  1739935200,
		},
		{
			name:        "rfc3339 with timezone",
			input:       "2026-02-25T14:30:00+08:00",
			expectHasTZ: true,
			expectUnix:  1772001000,
		},
		{
			name:        "local datetime",
			input:       "2026-02-25 09:15",
			expectHasTZ: false,
			expectUnix:  time.Date(2026, 2, 25, 9, 15, 0, 0, loc).Unix(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, hasTZ, err := parseFeishuDateTime(tt.input, loc)
			if err != nil {
				t.Fatalf("parseFeishuDateTime() error = %v", err)
			}
			if hasTZ != tt.expectHasTZ {
				t.Fatalf("hasExplicitTZ = %v, want %v", hasTZ, tt.expectHasTZ)
			}
			if got.Unix() != tt.expectUnix {
				t.Fatalf("unix = %d, want %d", got.Unix(), tt.expectUnix)
			}
		})
	}
}

func TestResolveFeishuEventTimezone(t *testing.T) {
	parsed := time.Unix(1772001000, 0) // 2026-02-25T14:30:00+08:00 equivalent

	got, err := resolveFeishuEventTimezone("Asia/Shanghai", parsed, false)
	if err != nil {
		t.Fatalf("resolveFeishuEventTimezone returned error: %v", err)
	}
	if got != "Asia/Shanghai" {
		t.Fatalf("timezone = %q, want %q", got, "Asia/Shanghai")
	}

	withOffset := time.FixedZone("UTC-5", -5*3600)
	got, err = resolveFeishuEventTimezone("", time.Date(2026, 2, 25, 9, 0, 0, 0, withOffset), true)
	if err != nil {
		t.Fatalf("resolveFeishuEventTimezone returned error: %v", err)
	}
	if got != "Etc/GMT+5" {
		t.Fatalf("timezone = %q, want %q", got, "Etc/GMT+5")
	}
}

func TestParseReminderMinutesArg(t *testing.T) {
	got, err := parseReminderMinutesArg(map[string]any{
		"reminder_minutes": []any{30.0, 10.0, 30.0},
	}, "reminder_minutes")
	if err != nil {
		t.Fatalf("parseReminderMinutesArg() error = %v", err)
	}
	if len(got) != 2 || got[0] != 10 || got[1] != 30 {
		t.Fatalf("got %v, want [10 30]", got)
	}

	if _, err := parseReminderMinutesArg(map[string]any{
		"reminder_minutes": []any{"10"},
	}, "reminder_minutes"); err == nil {
		t.Fatalf("expected error for non-integer reminder item")
	}
}

func TestShouldUseFeishuPrimaryCalendar(t *testing.T) {
	tests := []struct {
		name       string
		calendarID string
		want       bool
	}{
		{name: "empty", calendarID: "", want: true},
		{name: "spaces", calendarID: "   ", want: true},
		{name: "primary lower", calendarID: "primary", want: true},
		{name: "primary mixed case", calendarID: "Primary", want: true},
		{name: "primary with spaces", calendarID: "  PRIMARY  ", want: true},
		{name: "real calendar id", calendarID: "feishu.cn_xxx@group.calendar.feishu.cn", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldUseFeishuPrimaryCalendar(tt.calendarID)
			if got != tt.want {
				t.Fatalf("shouldUseFeishuPrimaryCalendar(%q) = %v, want %v", tt.calendarID, got, tt.want)
			}
		})
	}
}

func TestGenerateFeishuIdempotencyKey(t *testing.T) {
	key := generateFeishuIdempotencyKey()
	if key == "" {
		t.Fatalf("generateFeishuIdempotencyKey() returned empty key")
	}
	if _, err := uuid.Parse(key); err != nil {
		t.Fatalf("generateFeishuIdempotencyKey() returned non-uuid value: %q, err=%v", key, err)
	}
}

func TestBuildFeishuInviteeUserIDs(t *testing.T) {
	t.Run("explicit user attendees only", func(t *testing.T) {
		got := buildFeishuInviteeUserIDs([]string{"u1", " u2 ", "u1"}, "feishu", "5bc38gcb")
		if len(got) != 2 {
			t.Fatalf("len(ids) = %d, want 2", len(got))
		}
		if got[0] != "u1" || got[1] != "u2" {
			t.Fatalf("ids = %v, want [u1 u2]", got)
		}
	})

	t.Run("auto include sender for feishu when empty", func(t *testing.T) {
		got := buildFeishuInviteeUserIDs(nil, "feishu", "5bc38gcb")
		if len(got) != 1 {
			t.Fatalf("len(ids) = %d, want 1", len(got))
		}
		if got[0] != "5bc38gcb" {
			t.Fatalf("ids[0] = %q, want 5bc38gcb", got[0])
		}
	})

	t.Run("do not include unknown sender", func(t *testing.T) {
		got := buildFeishuInviteeUserIDs(nil, "feishu", "unknown")
		if len(got) != 0 {
			t.Fatalf("len(ids) = %d, want 0", len(got))
		}
	})

	t.Run("no auto include for non-feishu", func(t *testing.T) {
		got := buildFeishuInviteeUserIDs(nil, "telegram", "123")
		if len(got) != 0 {
			t.Fatalf("len(ids) = %d, want 0", len(got))
		}
	})
}
