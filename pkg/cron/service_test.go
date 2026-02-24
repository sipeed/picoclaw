package cron

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/adhocore/gronx"
)

func TestSaveStore_FilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission bits are not enforced on Windows")
	}

	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "cron", "jobs.json")

	cs := NewCronService(storePath, nil)

	_, err := cs.AddJob("test", CronSchedule{Kind: "every", EveryMS: int64Ptr(60000)}, "hello", false, "cli", "direct")
	if err != nil {
		t.Fatalf("AddJob failed: %v", err)
	}

	info, err := os.Stat(storePath)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	perm := info.Mode().Perm()
	if perm != 0o600 {
		t.Errorf("cron store has permission %04o, want 0600", perm)
	}
}

func int64Ptr(v int64) *int64 {
	return &v
}

func TestComputeNextRun_CronUsesScheduleTimezone(t *testing.T) {
	cs := NewCronService(filepath.Join(t.TempDir(), "jobs.json"), nil)

	// Use a fixed UTC reference to make this deterministic across environments.
	nowMS := time.Date(2026, 2, 24, 0, 30, 0, 0, time.UTC).UnixMilli()
	expr := "0 9 * * *"

	utcSchedule := CronSchedule{
		Kind: "cron",
		Expr: expr,
		TZ:   "UTC",
	}
	shSchedule := CronSchedule{
		Kind: "cron",
		Expr: expr,
		TZ:   "Asia/Shanghai",
	}

	utcNextMS := cs.computeNextRun(&utcSchedule, nowMS)
	if utcNextMS == nil {
		t.Fatalf("expected UTC next run, got nil")
	}
	shNextMS := cs.computeNextRun(&shSchedule, nowMS)
	if shNextMS == nil {
		t.Fatalf("expected Asia/Shanghai next run, got nil")
	}

	expectedUTC, err := gronx.NextTickAfter(expr, time.UnixMilli(nowMS).In(time.UTC), false)
	if err != nil {
		t.Fatalf("failed to compute expected UTC tick: %v", err)
	}
	shLoc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("failed to load Asia/Shanghai location: %v", err)
	}
	expectedSH, err := gronx.NextTickAfter(expr, time.UnixMilli(nowMS).In(shLoc), false)
	if err != nil {
		t.Fatalf("failed to compute expected Asia/Shanghai tick: %v", err)
	}

	if got, want := *utcNextMS, expectedUTC.UnixMilli(); got != want {
		t.Fatalf("UTC next run mismatch: got %d, want %d", got, want)
	}
	if got, want := *shNextMS, expectedSH.UnixMilli(); got != want {
		t.Fatalf("Asia/Shanghai next run mismatch: got %d, want %d", got, want)
	}
	if *utcNextMS == *shNextMS {
		t.Fatalf("expected timezone-specific next run to differ, both were %d", *utcNextMS)
	}
}

func TestComputeNextRun_CronInvalidTimezone(t *testing.T) {
	cs := NewCronService(filepath.Join(t.TempDir(), "jobs.json"), nil)

	nowMS := time.Date(2026, 2, 24, 0, 30, 0, 0, time.UTC).UnixMilli()
	schedule := CronSchedule{
		Kind: "cron",
		Expr: "*/5 * * * *",
		TZ:   "Mars/OlympusMons",
	}

	next := cs.computeNextRun(&schedule, nowMS)
	if next != nil {
		t.Fatalf("expected nil for invalid timezone, got %d", *next)
	}
}

func TestAddJob_CronInvalidTimezoneReturnsError(t *testing.T) {
	cs := NewCronService(filepath.Join(t.TempDir(), "jobs.json"), nil)

	_, err := cs.AddJob(
		"bad-tz",
		CronSchedule{
			Kind: "cron",
			Expr: "*/5 * * * *",
			TZ:   "Mars/OlympusMons",
		},
		"hello",
		false,
		"cli",
		"direct",
	)
	if err == nil {
		t.Fatalf("expected error for invalid timezone")
	}
}
