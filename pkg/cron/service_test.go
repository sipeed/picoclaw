package cron

import (
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
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

// TestCronServiceAddJob tests basic job addition
func TestCronServiceAddJob(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "cron", "jobs.json")

	var executed atomic.Int32
	handler := func(job *CronJob) (string, error) {
		executed.Add(1)
		return "executed", nil
	}

	cs := NewCronService(storePath, handler)

	// Add one-time job
	job, err := cs.AddJob("test-job", CronSchedule{Kind: "at", AtMS: int64Ptr(time.Now().UnixMilli() + 1000)}, "test message", true, "cli", "direct")
	if err != nil {
		t.Fatalf("AddJob failed: %v", err)
	}

	if job.Name != "test-job" {
		t.Errorf("job name = %q, want %q", job.Name, "test-job")
	}

	if job.Schedule.Kind != "at" {
		t.Errorf("job kind = %q, want %q", job.Schedule.Kind, "at")
	}

	jobs := cs.ListJobs(false)
	if len(jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(jobs))
	}
}

// TestCronServiceAddRecurringJob tests recurring job addition
func TestCronServiceAddRecurringJob(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "cron", "jobs.json")

	cs := NewCronService(storePath, nil)

	// Add recurring job
	job, err := cs.AddJob("recurring", CronSchedule{Kind: "every", EveryMS: int64Ptr(3600000)}, "hourly task", true, "cli", "direct")
	if err != nil {
		t.Fatalf("AddJob failed: %v", err)
	}

	if job.Schedule.Kind != "every" {
		t.Errorf("job kind = %q, want %q", job.Schedule.Kind, "every")
	}

	if job.Schedule.EveryMS == nil || *job.Schedule.EveryMS != 3600000 {
		t.Errorf("job everyMS = %v, want 3600000", job.Schedule.EveryMS)
	}
}

// TestCronServiceAddCronJob tests cron expression job addition
func TestCronServiceAddCronJob(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "cron", "jobs.json")

	cs := NewCronService(storePath, nil)

	// Add cron job
	job, err := cs.AddJob("daily", CronSchedule{Kind: "cron", Expr: "0 9 * * *"}, "daily at 9am", true, "cli", "direct")
	if err != nil {
		t.Fatalf("AddJob failed: %v", err)
	}

	if job.Schedule.Kind != "cron" {
		t.Errorf("job kind = %q, want %q", job.Schedule.Kind, "cron")
	}

	if job.Schedule.Expr != "0 9 * * *" {
		t.Errorf("job expr = %q, want %q", job.Schedule.Expr, "0 9 * * *")
	}
}

// TestCronServiceRemoveJob tests job removal
func TestCronServiceRemoveJob(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "cron", "jobs.json")

	cs := NewCronService(storePath, nil)

	// Add job
	job, _ := cs.AddJob("to-remove", CronSchedule{Kind: "every", EveryMS: int64Ptr(60000)}, "remove me", true, "cli", "direct")

	// Remove job
	removed := cs.RemoveJob(job.ID)
	if !removed {
		t.Error("RemoveJob returned false, want true")
	}

	jobs := cs.ListJobs(false)
	if len(jobs) != 0 {
		t.Errorf("expected 0 jobs after removal, got %d", len(jobs))
	}

	// Remove non-existent job
	removed2 := cs.RemoveJob("nonexistent")
	if removed2 {
		t.Error("RemoveJob should return false for non-existent job")
	}
}

// TestCronServiceEnableDisableJob tests job enable/disable
func TestCronServiceEnableDisableJob(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "cron", "jobs.json")

	cs := NewCronService(storePath, nil)

	// Add job
	job, _ := cs.AddJob("toggle", CronSchedule{Kind: "every", EveryMS: int64Ptr(60000)}, "toggle me", true, "cli", "direct")

	// Disable job
	disabled := cs.EnableJob(job.ID, false)
	if disabled == nil {
		t.Fatal("EnableJob returned nil, want job")
	}
	if disabled.Enabled {
		t.Error("job should be disabled")
	}

	// Enable job
	enabled := cs.EnableJob(job.ID, true)
	if enabled == nil {
		t.Fatal("EnableJob returned nil, want job")
	}
	if !enabled.Enabled {
		t.Error("job should be enabled")
	}

	// Enable non-existent job
	notFound := cs.EnableJob("nonexistent", true)
	if notFound != nil {
		t.Error("EnableJob should return nil for non-existent job")
	}
}

// TestCronServiceUpdateJob tests job update
func TestCronServiceUpdateJob(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "cron", "jobs.json")

	cs := NewCronService(storePath, nil)

	// Add job
	job, _ := cs.AddJob("update-test", CronSchedule{Kind: "every", EveryMS: int64Ptr(60000)}, "original", true, "cli", "direct")

	// Update job
	job.Payload.Message = "updated message"
	err := cs.UpdateJob(job)
	if err != nil {
		t.Fatalf("UpdateJob failed: %v", err)
	}

	// Reload and verify
	cs.Load()
	updatedJob := cs.ListJobs(true)[0]
	if updatedJob.Payload.Message != "updated message" {
		t.Errorf("job message = %q, want %q", updatedJob.Payload.Message, "updated message")
	}

	// Update non-existent job
	job.ID = "nonexistent"
	err = cs.UpdateJob(job)
	if err == nil {
		t.Error("UpdateJob should fail for non-existent job")
	}
}

// TestCronServiceListJobs tests job listing
func TestCronServiceListJobs(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "cron", "jobs.json")

	cs := NewCronService(storePath, nil)

	// Add jobs
	cs.AddJob("job1", CronSchedule{Kind: "every", EveryMS: int64Ptr(60000)}, "msg1", true, "cli", "direct")
	cs.AddJob("job2", CronSchedule{Kind: "every", EveryMS: int64Ptr(120000)}, "msg2", true, "cli", "direct")

	// Disable one job
	jobs := cs.ListJobs(true)
	if len(jobs) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(jobs))
	}
	cs.EnableJob(jobs[0].ID, false)

	// List only enabled jobs
	enabledJobs := cs.ListJobs(false)
	if len(enabledJobs) != 1 {
		t.Errorf("expected 1 enabled job, got %d", len(enabledJobs))
	}

	// List all jobs including disabled
	allJobs := cs.ListJobs(true)
	if len(allJobs) != 2 {
		t.Errorf("expected 2 total jobs, got %d", len(allJobs))
	}
}

// TestCronServiceStartStop tests service start and stop
func TestCronServiceStartStop(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "cron", "jobs.json")

	cs := NewCronService(storePath, nil)

	// Start service
	err := cs.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if !cs.running {
		t.Error("service should be running")
	}

	// Stop service
	cs.Stop()

	if cs.running {
		t.Error("service should be stopped")
	}

	// Start again (should work)
	err = cs.Start()
	if err != nil {
		t.Fatalf("Start second time failed: %v", err)
	}
	cs.Stop()
}

// TestCronServiceStatus tests status reporting
func TestCronServiceStatus(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "cron", "jobs.json")

	cs := NewCronService(storePath, nil)

	// Add jobs
	cs.AddJob("active", CronSchedule{Kind: "every", EveryMS: int64Ptr(60000)}, "msg", true, "cli", "direct")
	cs.AddJob("inactive", CronSchedule{Kind: "every", EveryMS: int64Ptr(120000)}, "msg2", true, "cli", "direct")
	jobs := cs.ListJobs(true)
	if len(jobs) > 0 {
		cs.EnableJob(jobs[0].ID, false)
	}

	status := cs.Status()

	if status["jobs"] != 2 {
		t.Errorf("expected 2 jobs, got %v", status["jobs"])
	}

	// Check if service is running (should be false as we didn't start it)
	isRunning, ok := status["enabled"].(bool)
	if !ok {
		t.Errorf("expected 'enabled' to be bool, got %T", status["enabled"])
	}
	if isRunning {
		t.Error("service should not be running until Start() is called")
	}

	// Verify enabled job count separately
	enabledJobs := cs.ListJobs(false)
	if len(enabledJobs) != 1 {
		t.Errorf("expected 1 enabled job, got %d", len(enabledJobs))
	}
}

// TestCronServiceComputeNextRun tests next run computation
func TestCronServiceComputeNextRun(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "cron", "jobs.json")

	cs := NewCronService(storePath, nil)

	now := time.Now().UnixMilli()

	// Test "at" schedule
	atTime := now + 5000
	nextRun := cs.computeNextRun(&CronSchedule{Kind: "at", AtMS: &atTime}, now)
	if nextRun == nil || *nextRun != atTime {
		t.Errorf("at schedule nextRun = %v, want %v", nextRun, atTime)
	}

	// Test "every" schedule
	everyMS := int64(3600000)
	nextRun = cs.computeNextRun(&CronSchedule{Kind: "every", EveryMS: &everyMS}, now)
	if nextRun == nil || *nextRun != now+everyMS {
		t.Errorf("every schedule nextRun = %v, want %v", nextRun, now+everyMS)
	}
}

// TestCronServicePersistence tests that jobs persist across restarts
func TestCronServicePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "cron", "jobs.json")

	// Create service and add job
	cs1 := NewCronService(storePath, nil)
	cs1.AddJob("persistent", CronSchedule{Kind: "every", EveryMS: int64Ptr(60000)}, "persist", true, "cli", "direct")

	// Create new service instance (simulates restart)
	cs2 := NewCronService(storePath, nil)
	cs2.Load()

	jobs := cs2.ListJobs(true)
	if len(jobs) != 1 {
		t.Errorf("expected 1 persisted job, got %d", len(jobs))
	}

	if jobs[0].Name != "persistent" {
		t.Errorf("job name = %q, want %q", jobs[0].Name, "persistent")
	}
}

// TestCronServiceWithCommand tests job with command payload
func TestCronServiceWithCommand(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "cron", "jobs.json")

	cs := NewCronService(storePath, nil)

	// Add job with command
	job, err := cs.AddJob("cmd-job", CronSchedule{Kind: "at", AtMS: int64Ptr(time.Now().UnixMilli() + 1000)}, "check disk", true, "cli", "direct")
	if err != nil {
		t.Fatalf("AddJob failed: %v", err)
	}

	job.Payload.Command = "df -h"
	cs.UpdateJob(job)

	// Reload and verify
	cs.Load()
	loadedJob := cs.ListJobs(true)[0]
	if loadedJob.Payload.Command != "df -h" {
		t.Errorf("command = %q, want %q", loadedJob.Payload.Command, "df -h")
	}
}

func int64Ptr(v int64) *int64 {
	return &v
}
