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

func TestEnableJob_PreservesJobsAddedByAnotherService(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "cron", "jobs.json")

	seed := NewCronService(storePath, nil)
	existingJob, err := seed.AddJob(
		"existing",
		CronSchedule{Kind: "every", EveryMS: int64Ptr(60000)},
		"hello",
		false,
		"cli",
		"direct",
	)
	if err != nil {
		t.Fatalf("seed AddJob failed: %v", err)
	}

	stale := NewCronService(storePath, nil)

	cli := NewCronService(storePath, nil)
	newJob, err := cli.AddJob(
		"new",
		CronSchedule{Kind: "every", EveryMS: int64Ptr(120000)},
		"world",
		false,
		"cli",
		"direct",
	)
	if err != nil {
		t.Fatalf("cli AddJob failed: %v", err)
	}

	updated := stale.EnableJob(existingJob.ID, false)
	if updated == nil {
		t.Fatalf("EnableJob(%q) returned nil", existingJob.ID)
	}

	jobs := NewCronService(storePath, nil).ListJobs(true)
	if len(jobs) != 2 {
		t.Fatalf("ListJobs() returned %d jobs, want 2", len(jobs))
	}

	if !jobExists(jobs, existingJob.ID) {
		t.Fatalf("existing job %q was lost after save", existingJob.ID)
	}
	if !jobExists(jobs, newJob.ID) {
		t.Fatalf("new job %q was lost after save", newJob.ID)
	}
	if jobEnabled(jobs, existingJob.ID) {
		t.Fatalf("existing job %q should be disabled", existingJob.ID)
	}
}

func TestListJobs_RefreshesStoreBeforeReading(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "cron", "jobs.json")

	seed := NewCronService(storePath, nil)
	if _, err := seed.AddJob(
		"existing",
		CronSchedule{Kind: "every", EveryMS: int64Ptr(60000)},
		"hello",
		false,
		"cli",
		"direct",
	); err != nil {
		t.Fatalf("seed AddJob failed: %v", err)
	}

	stale := NewCronService(storePath, nil)

	cli := NewCronService(storePath, nil)
	newJob, err := cli.AddJob(
		"new",
		CronSchedule{Kind: "every", EveryMS: int64Ptr(120000)},
		"world",
		false,
		"cli",
		"direct",
	)
	if err != nil {
		t.Fatalf("cli AddJob failed: %v", err)
	}

	jobs := stale.ListJobs(true)
	if len(jobs) != 2 {
		t.Fatalf("ListJobs() returned %d jobs, want 2", len(jobs))
	}
	if !jobExists(jobs, newJob.ID) {
		t.Fatalf("ListJobs() did not include externally added job %q", newJob.ID)
	}
}

func jobExists(jobs []CronJob, jobID string) bool {
	for _, job := range jobs {
		if job.ID == jobID {
			return true
		}
	}
	return false
}

func jobEnabled(jobs []CronJob, jobID string) bool {
	for _, job := range jobs {
		if job.ID == jobID {
			return job.Enabled
		}
	}
	return false
}

func int64Ptr(v int64) *int64 {
	return &v
}
