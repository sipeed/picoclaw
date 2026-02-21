package sandbox

import (
	"path/filepath"
	"testing"
	"time"
)

func TestRegistryUpsertAndRemove(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sandbox", "registry.json")
	now := time.Now().UnixMilli()

	err := upsertRegistryEntry(path, registryEntry{
		ContainerName: "c1",
		Image:         "img",
		ConfigHash:    "h1",
		CreatedAtMs:   now,
		LastUsedAtMs:  now,
	})
	if err != nil {
		t.Fatalf("upsertRegistryEntry create failed: %v", err)
	}

	data, err := loadRegistry(path)
	if err != nil {
		t.Fatalf("loadRegistry failed: %v", err)
	}
	if len(data.Entries) != 1 {
		t.Fatalf("entries len = %d, want 1", len(data.Entries))
	}

	err = upsertRegistryEntry(path, registryEntry{
		ContainerName: "c1",
		Image:         "img2",
		ConfigHash:    "h2",
		CreatedAtMs:   now + 1000,
		LastUsedAtMs:  now + 1000,
	})
	if err != nil {
		t.Fatalf("upsertRegistryEntry update failed: %v", err)
	}
	data, err = loadRegistry(path)
	if err != nil {
		t.Fatalf("loadRegistry failed: %v", err)
	}
	if len(data.Entries) != 1 {
		t.Fatalf("entries len after update = %d, want 1", len(data.Entries))
	}
	if data.Entries[0].ConfigHash != "h2" {
		t.Fatalf("config hash = %q, want h2", data.Entries[0].ConfigHash)
	}
	if data.Entries[0].CreatedAtMs != now {
		t.Fatalf("createdAt preserved = %d, want %d", data.Entries[0].CreatedAtMs, now)
	}

	if err := removeRegistryEntry(path, "c1"); err != nil {
		t.Fatalf("removeRegistryEntry failed: %v", err)
	}
	data, err = loadRegistry(path)
	if err != nil {
		t.Fatalf("loadRegistry failed: %v", err)
	}
	if len(data.Entries) != 0 {
		t.Fatalf("entries len after remove = %d, want 0", len(data.Entries))
	}
}

func TestComputeConfigHashDeterministic(t *testing.T) {
	a := computeConfigHash("img", "/workspace")
	b := computeConfigHash("img", "/workspace")
	c := computeConfigHash("img2", "/workspace")

	if a != b {
		t.Fatalf("same input hash mismatch: %q vs %q", a, b)
	}
	if a == c {
		t.Fatalf("different input should produce different hash: %q", a)
	}
}

func TestShouldPruneEntry(t *testing.T) {
	now := time.Now().UnixMilli()
	cfg := ContainerSandboxConfig{
		PruneIdleHours:  1,
		PruneMaxAgeDays: 2,
	}
	oldIdle := registryEntry{
		CreatedAtMs:  now,
		LastUsedAtMs: now - int64(2*time.Hour/time.Millisecond),
	}
	if !shouldPruneEntry(cfg, now, oldIdle) {
		t.Fatal("expected old idle entry to be pruned")
	}

	oldAge := registryEntry{
		CreatedAtMs:  now - int64(3*24*time.Hour/time.Millisecond),
		LastUsedAtMs: now,
	}
	if !shouldPruneEntry(cfg, now, oldAge) {
		t.Fatal("expected old age entry to be pruned")
	}

	fresh := registryEntry{
		CreatedAtMs:  now,
		LastUsedAtMs: now,
	}
	if shouldPruneEntry(cfg, now, fresh) {
		t.Fatal("did not expect fresh entry to be pruned")
	}
}
