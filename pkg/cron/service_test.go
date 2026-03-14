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
	if perm != 0o600 {
		t.Errorf("cron store has permission %04o, want 0600", perm)
	}
}

func TestComputeNextRun_UsesScheduleTimeZone(t *testing.T) {
	cs := NewCronService(filepath.Join(t.TempDir(), "cron", "jobs.json"), nil)
	now := time.Date(2026, time.March, 13, 12, 30, 0, 0, time.UTC).UnixMilli()
	baseline := cs.computeNextRun(&CronSchedule{Kind: "cron", Expr: "0 9 * * *"}, now)
	if baseline == nil {
		t.Fatal("baseline computeNextRun() returned nil")
	}

	t.Run("uses explicit timezone", func(t *testing.T) {
		next := cs.computeNextRun(&CronSchedule{
			Kind: "cron",
			Expr: "0 9 * * *",
			TZ:   "America/New_York",
		}, now)
		if next == nil {
			t.Fatal("computeNextRun() returned nil")
		}

		wantNext := time.Date(2026, time.March, 13, 13, 0, 0, 0, time.UTC).UnixMilli()
		if *next != wantNext {
			t.Fatalf("computeNextRun() = %d, want %d", *next, wantNext)
		}
		if *next == *baseline {
			t.Fatal("explicit timezone should change the computed next run for this fixture")
		}
	})

	t.Run("falls back to baseline on invalid timezone", func(t *testing.T) {
		next := cs.computeNextRun(&CronSchedule{
			Kind: "cron",
			Expr: "0 9 * * *",
			TZ:   "Mars/OlympusMons",
		}, now)
		if next == nil {
			t.Fatal("computeNextRun() returned nil")
		}
		if *next != *baseline {
			t.Fatalf("computeNextRun() = %d, want baseline %d", *next, *baseline)
		}
	})
}

func int64Ptr(v int64) *int64 {
	return &v
}
