package sandbox

import (
	"path/filepath"
	"testing"
	"time"
)

func TestRegistryFileLock_AcquireRelease(t *testing.T) {
	regPath := filepath.Join(t.TempDir(), "sandbox", "registry.json")
	lock, err := acquireRegistryFileLock(regPath)
	if err != nil {
		t.Fatalf("acquireRegistryFileLock failed: %v", err)
	}
	lock.release()
}

func TestRegistryFileLock_WaitsUntilReleased(t *testing.T) {
	regPath := filepath.Join(t.TempDir(), "sandbox", "registry.json")
	first, err := acquireRegistryFileLock(regPath)
	if err != nil {
		t.Fatalf("first lock failed: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		lock, err := acquireRegistryFileLock(regPath)
		if err == nil && lock != nil {
			lock.release()
		}
		done <- err
	}()

	time.Sleep(80 * time.Millisecond)
	first.release()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("second lock should succeed after release, got: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for second lock acquisition")
	}
}
