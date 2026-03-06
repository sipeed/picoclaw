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

// TestRecurringJobsPreserveSchedule verifies that recurring jobs maintain their schedule after execution.
func TestRecurringJobsPreserveSchedule(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "cron", "jobs.json")

	handler := func(job *CronJob) (string, error) {
		return "done", nil
	}

	cs := NewCronService(storePath, handler)

	// Add an 'every' type recurring job
	everyMS := int64(1000)  // 1 second intervals
	job, err := cs.AddJob("recurring-test", CronSchedule{Kind: "every", EveryMS: &everyMS}, "hello recurring", false, "cli", "direct")
	if err != nil {
		t.Fatalf("AddJob failed: %v", err)
	}

	// Initially, the job should be enabled and have a next run time
	if !job.Enabled {
		t.Errorf("Job should be enabled initially");
	}
	if job.State.NextRunAtMS == nil {
		t.Errorf("Job should have a next run time initially");
	}

	// Execute the job manually once
	cs.executeJobByID(job.ID)

	// Verify the job still exists, is enabled, and has next run time
	foundJob := false
	for _, j := range cs.ListJobs(true) { // include disabled jobs
		if j.ID == job.ID {
			foundJob = true
			// Check that it's still enabled and has NextRunAtMS
			if !j.Enabled {
				t.Errorf("Recurring 'every' job was disabled after execution - this indicates a bug where recurring jobs become one-time.")
			}
			if j.State.NextRunAtMS == nil {
				 t.Errorf("Recurring 'every' job lost its next run time after execution");
			}
			break
		}
	}

	if !foundJob {
		t.Errorf("Job not found after execution - it was incorrectly removed.")
	}
}

// TestInvalidCronExpressionIsHandledGracefully checks that cron jobs with invalid expressions are handled appropriately
func TestInvalidCronExpressionIsHandledGracefully(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "cron", "jobs.json")

	cs := NewCronService(storePath, func(job *CronJob) (string, error) {
		return "ok", nil
	})

	// Add a cron job with invalid expression
	job, err := cs.AddJob("invalid-cron-test", CronSchedule{Kind: "cron", Expr: "not-a-valid-cron-expr"}, "invalid cron", false, "cli", "direct")
	if err != nil {
		t.Fatalf("AddJob for invalid cron job failed: %v", err)
	}

	// Execute the job - this should not crash and should handle gracefully
	cs.executeJobByID(job.ID)

	// For invalid cron expressions, the job should possibly be disabled after execution based on our fix
	var jobFound *CronJob
	for _, j := range cs.ListJobs(true) {
		if j.ID == job.ID {
			jobFound = &j
			break
		}
	}

	if jobFound == nil {
		t.Fatalf("Job not found after execution")
	}
}

func int64Ptr(v int64) *int64 {
	return &v
}
