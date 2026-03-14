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

func TestScopedCronOperations(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "cron", "jobs.json")
	cs := NewCronService(storePath, nil)

	every := int64(60000)
	j1, err := cs.AddJob("u1", CronSchedule{Kind: "every", EveryMS: &every}, "m1", false, "telegram", "user1")
	if err != nil {
		t.Fatalf("AddJob u1 failed: %v", err)
	}
	j2, err := cs.AddJob("u2", CronSchedule{Kind: "every", EveryMS: &every}, "m2", false, "telegram", "user2")
	if err != nil {
		t.Fatalf("AddJob u2 failed: %v", err)
	}

	jobsU1 := cs.ListJobsForTarget("telegram", "user1", true)
	if len(jobsU1) != 1 || jobsU1[0].ID != j1.ID {
		t.Fatalf("expected only user1 job, got %+v", jobsU1)
	}

	if cs.RemoveJobForTarget(j2.ID, "telegram", "user1") {
		t.Fatalf("expected remove to fail across target boundary")
	}
	if !cs.RemoveJobForTarget(j2.ID, "telegram", "user2") {
		t.Fatalf("expected remove to succeed for owner target")
	}

	if job := cs.EnableJobForTarget(j1.ID, "telegram", "user2", false); job != nil {
		t.Fatalf("expected enable/disable to fail across target boundary")
	}
	if job := cs.EnableJobForTarget(j1.ID, "telegram", "user1", false); job == nil || job.Enabled {
		t.Fatalf("expected owner disable to succeed")
	}
}
