package cron

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
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

func TestCronServiceIndexedOperationsRemainCompatible(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "cron", "jobs.json")

	cs := NewCronService(storePath, nil)

	job1, err := cs.AddJob("job-1", CronSchedule{Kind: "every", EveryMS: int64Ptr(60000)}, "hello", false, "cli", "direct")
	if err != nil {
		t.Fatalf("AddJob job1 failed: %v", err)
	}
	job2, err := cs.AddJob("job-2", CronSchedule{Kind: "every", EveryMS: int64Ptr(120000)}, "world", false, "cli", "direct")
	if err != nil {
		t.Fatalf("AddJob job2 failed: %v", err)
	}

	job2.Name = "job-2-updated"
	if err := cs.UpdateJob(job2); err != nil {
		t.Fatalf("UpdateJob failed: %v", err)
	}

	disabled := cs.EnableJob(job1.ID, false)
	if disabled == nil {
		t.Fatal("EnableJob returned nil for existing job")
	}
	if disabled.Enabled {
		t.Fatal("expected job to be disabled")
	}

	if !cs.RemoveJob(job1.ID) {
		t.Fatal("RemoveJob failed for existing job")
	}

	reloaded := NewCronService(storePath, nil)
	jobs := reloaded.ListJobs(true)
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job after reload, got %d", len(jobs))
	}
	if jobs[0].ID != job2.ID {
		t.Fatalf("expected remaining job %s, got %s", job2.ID, jobs[0].ID)
	}
	if jobs[0].Name != "job-2-updated" {
		t.Fatalf("expected updated job name to persist, got %q", jobs[0].Name)
	}

	enabled := reloaded.EnableJob(job2.ID, true)
	if enabled == nil {
		t.Fatal("EnableJob on reloaded service returned nil")
	}
	if !enabled.Enabled {
		t.Fatal("expected reloaded job to be enabled")
	}
}
