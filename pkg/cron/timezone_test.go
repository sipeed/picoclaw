package cron

import (
	"path/filepath"
	"testing"
	"time"
)

// TestComputeNextRun_CronTimezone verifies that cron expressions respect
// the schedule.TZ field instead of always using UTC.
func TestComputeNextRun_CronTimezone(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "jobs.json")
	cs := NewCronService(storePath, nil)

	now := time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC) // midnight UTC
	nowMS := now.UnixMilli()

	// Cron expr "0 9 * * *" = daily at 9:00
	tests := []struct {
		name     string
		tz       string
		wantHour int // expected hour in UTC of the next run
	}{
		{
			name:     "UTC timezone",
			tz:       "UTC",
			wantHour: 9, // 9:00 UTC
		},
		{
			name:     "Asia/Shanghai timezone",
			tz:       "Asia/Shanghai",
			wantHour: 1, // 9:00 CST = 1:00 UTC
		},
		{
			name:     "empty TZ defaults to UTC",
			tz:       "",
			wantHour: 9, // should default to UTC
		},
		{
			name:     "US/Eastern timezone",
			tz:       "US/Eastern",
			wantHour: 14, // 9:00 EST = 14:00 UTC (or 13:00 during DST)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule := &CronSchedule{
				Kind: "cron",
				Expr: "0 9 * * *",
				TZ:   tt.tz,
			}

			nextMS := cs.computeNextRun(schedule, nowMS)
			if nextMS == nil {
				t.Fatal("computeNextRun returned nil")
			}

			nextTime := time.UnixMilli(*nextMS).UTC()

			if tt.name == "US/Eastern timezone" {
				// Allow for DST variation (13 or 14)
				if nextTime.Hour() != 13 && nextTime.Hour() != 14 {
					t.Errorf("next run hour = %d, want 13 or 14 (UTC)", nextTime.Hour())
				}
			} else {
				if nextTime.Hour() != tt.wantHour {
					t.Errorf("next run hour = %d, want %d (UTC)", nextTime.Hour(), tt.wantHour)
				}
			}
		})
	}
}

// TestComputeNextRun_DefaultTZ_AppliesWhenSet verifies that SetDefaultTimezone
// takes effect when schedule.TZ is empty (service default overrides UTC fallback).
func TestComputeNextRun_DefaultTZ_AppliesWhenSet(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "jobs.json")
	cs := NewCronService(storePath, nil)

	// Set a non-UTC default to verify it takes effect
	cs.SetDefaultTimezone("Asia/Shanghai")

	now := time.Date(2026, 3, 4, 2, 0, 0, 0, time.UTC) // 02:00 UTC = 10:00 CST
	nowMS := now.UnixMilli()

	scheduleEmpty := &CronSchedule{Kind: "cron", Expr: "0 9 * * *", TZ: ""}
	scheduleUTC := &CronSchedule{Kind: "cron", Expr: "0 9 * * *", TZ: "UTC"}

	nextEmpty := cs.computeNextRun(scheduleEmpty, nowMS)
	nextUTC := cs.computeNextRun(scheduleUTC, nowMS)

	if nextEmpty == nil || nextUTC == nil {
		t.Fatal("computeNextRun returned nil")
	}

	// With service default Asia/Shanghai, 9:00 CST already passed (it's 10:00 CST),
	// so next run should be tomorrow 9:00 CST.
	// With explicit UTC, 9:00 UTC hasn't happened yet (it's 02:00 UTC).
	// They must differ, proving SetDefaultTimezone takes effect.
	if *nextEmpty == *nextUTC {
		t.Errorf("service default TZ (Asia/Shanghai) and UTC produced same next run time")
	}
}

// TestComputeNextRun_ServiceDefaultTZ verifies that SetDefaultTimezone()
// takes effect when schedule.TZ is empty (3-tier fallback).
func TestComputeNextRun_ServiceDefaultTZ(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "jobs.json")
	cs := NewCronService(storePath, nil)

	// Set service-level default to US/Eastern
	cs.SetDefaultTimezone("US/Eastern")

	now := time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC)
	nowMS := now.UnixMilli()

	// Schedule with empty TZ should use service default (US/Eastern), not UTC
	scheduleEmpty := &CronSchedule{Kind: "cron", Expr: "0 9 * * *", TZ: ""}
	scheduleCST := &CronSchedule{Kind: "cron", Expr: "0 9 * * *", TZ: "Asia/Shanghai"}

	nextEmpty := cs.computeNextRun(scheduleEmpty, nowMS)
	nextCST := cs.computeNextRun(scheduleCST, nowMS)

	if nextEmpty == nil || nextCST == nil {
		t.Fatal("computeNextRun returned nil")
	}

	// US/Eastern 9:00 != Asia/Shanghai 9:00, so they must differ
	if *nextEmpty == *nextCST {
		t.Errorf("service default TZ (US/Eastern) and Asia/Shanghai produced same time; SetDefaultTimezone not working")
	}

	// Schedule with explicit TZ should override service default
	scheduleExplicit := &CronSchedule{Kind: "cron", Expr: "0 9 * * *", TZ: "UTC"}
	nextExplicit := cs.computeNextRun(scheduleExplicit, nowMS)
	if nextExplicit == nil {
		t.Fatal("computeNextRun returned nil")
	}
	nextUTC := time.UnixMilli(*nextExplicit).UTC()
	if nextUTC.Hour() != 9 {
		t.Errorf("explicit TZ=UTC: next run hour = %d, want 9", nextUTC.Hour())
	}
}
