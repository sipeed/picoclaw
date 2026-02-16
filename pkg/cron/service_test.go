package cron

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCronStorePermissions(t *testing.T) {
	store := filepath.Join(t.TempDir(), "cron", "jobs.json")
	cs := NewCronService(store, nil)
	_, err := cs.AddJob("test", CronSchedule{Kind: "every", EveryMS: int64Ptr(1000)}, "hello", true, "cli", "direct")
	if err != nil {
		t.Fatalf("add job: %v", err)
	}
	info, err := os.Stat(store)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("expected 0600, got %o", info.Mode().Perm())
	}
}

func int64Ptr(v int64) *int64 { return &v }
