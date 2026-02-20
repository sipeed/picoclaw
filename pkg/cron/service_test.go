package cron

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
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
	if perm != 0600 {
		t.Errorf("cron store has permission %04o, want 0600", perm)
	}
}

func TestComputeNextRunRespectsTimezone(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "cron", "jobs.json")
	cs := NewCronService(storePath, nil)

	// 2024-01-01 10:00:00 UTC
	now := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	schedule := CronSchedule{
		Kind: "cron",
		Expr: "0 0 * * *",
		TZ:   "America/New_York",
	}

	next := cs.computeNextRun(&schedule, now.UnixMilli())
	if next == nil {
		t.Fatalf("expected next run time, got nil")
	}

	got := time.UnixMilli(*next).UTC()
	// Midnight in New York on Jan 2, 2024 is 05:00 UTC (UTC-5)
	want := time.Date(2024, 1, 2, 5, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("next run = %s, want %s", got, want)
	}
}

func int64Ptr(v int64) *int64 {
	return &v
}
