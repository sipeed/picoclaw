package cron

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestAddJobWritesPrivatePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission bits are not reliable on windows")
	}

	storePath := filepath.Join(t.TempDir(), "cron", "jobs.json")
	if err := os.MkdirAll(filepath.Dir(storePath), 0755); err != nil {
		t.Fatalf("failed to create store dir: %v", err)
	}
	if err := os.WriteFile(storePath, []byte(`{"version":1,"jobs":[]}`), 0644); err != nil {
		t.Fatalf("failed to create seed cron store: %v", err)
	}

	cs := NewCronService(storePath, nil)
	everyMS := int64(60000)
	schedule := CronSchedule{
		Kind:    "every",
		EveryMS: &everyMS,
	}

	if _, err := cs.AddJob("perm-test", schedule, "hello", true, "cli", "direct"); err != nil {
		t.Fatalf("AddJob failed: %v", err)
	}

	info, err := os.Stat(storePath)
	if err != nil {
		t.Fatalf("stat cron store: %v", err)
	}

	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("cron store perms = %o, want 600", got)
	}
}
