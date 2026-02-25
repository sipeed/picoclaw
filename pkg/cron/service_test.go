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

func TestSaveStoreFixesExistingFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission bits are not enforced on Windows")
	}

	storePath := filepath.Join(t.TempDir(), "cron", "jobs.json")
	if err := os.MkdirAll(filepath.Dir(storePath), 0o755); err != nil {
		t.Fatalf("failed to create store dir: %v", err)
	}
	if err := os.WriteFile(storePath, []byte(`{"version":1,"jobs":[]}`), 0o644); err != nil {
		t.Fatalf("failed to create seed cron store: %v", err)
	}

	cs := NewCronService(storePath, nil)
	if _, err := cs.AddJob(
		"perm-test",
		CronSchedule{Kind: "every", EveryMS: int64Ptr(60000)},
		"hello",
		true,
		"cli",
		"direct",
	); err != nil {
		t.Fatalf("AddJob failed: %v", err)
	}

	info, err := os.Stat(storePath)
	if err != nil {
		t.Fatalf("stat cron store: %v", err)
	}

	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("cron store perms = %o, want 600", got)
	}
}

func int64Ptr(v int64) *int64 {
	return &v
}
