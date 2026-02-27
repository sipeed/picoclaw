package media

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func createTempFile(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("test content"), 0o644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	return path
}

func TestStoreAndResolve(t *testing.T) {
	dir := t.TempDir()
	store := NewFileMediaStore()

	path := createTempFile(t, dir, "photo.jpg")

	ref, err := store.Store(path, MediaMeta{Filename: "photo.jpg", Source: "telegram"}, "scope1")
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	if !strings.HasPrefix(ref, "media://") {
		t.Errorf("ref should start with media://, got %q", ref)
	}

	resolved, err := store.Resolve(ref)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if resolved != path {
		t.Errorf("Resolve returned %q, want %q", resolved, path)
	}
}

func TestReleaseAll(t *testing.T) {
	dir := t.TempDir()
	store := NewFileMediaStore()

	paths := make([]string, 3)
	refs := make([]string, 3)
	for i := 0; i < 3; i++ {
		paths[i] = createTempFile(t, dir, strings.Repeat("a", i+1)+".jpg")
		var err error
		refs[i], err = store.Store(paths[i], MediaMeta{Source: "test"}, "scope1")
		if err != nil {
			t.Fatalf("Store failed: %v", err)
		}
	}

	if err := store.ReleaseAll("scope1"); err != nil {
		t.Fatalf("ReleaseAll failed: %v", err)
	}

	// Files should be deleted
	for _, p := range paths {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("file %q should have been deleted", p)
		}
	}

	// Refs should be unresolvable
	for _, ref := range refs {
		if _, err := store.Resolve(ref); err == nil {
			t.Errorf("Resolve(%q) should fail after ReleaseAll", ref)
		}
	}
}

func TestMultiScopeIsolation(t *testing.T) {
	dir := t.TempDir()
	store := NewFileMediaStore()

	pathA := createTempFile(t, dir, "fileA.jpg")
	pathB := createTempFile(t, dir, "fileB.jpg")

	refA, _ := store.Store(pathA, MediaMeta{Source: "test"}, "scopeA")
	refB, _ := store.Store(pathB, MediaMeta{Source: "test"}, "scopeB")

	// Release only scopeA
	if err := store.ReleaseAll("scopeA"); err != nil {
		t.Fatalf("ReleaseAll(scopeA) failed: %v", err)
	}

	// scopeA file should be gone
	if _, err := os.Stat(pathA); !os.IsNotExist(err) {
		t.Error("file A should have been deleted")
	}
	if _, err := store.Resolve(refA); err == nil {
		t.Error("refA should be unresolvable after release")
	}

	// scopeB file should still exist
	if _, err := os.Stat(pathB); err != nil {
		t.Error("file B should still exist")
	}
	resolved, err := store.Resolve(refB)
	if err != nil {
		t.Fatalf("refB should still resolve: %v", err)
	}
	if resolved != pathB {
		t.Errorf("resolved %q, want %q", resolved, pathB)
	}
}

func TestReleaseAllIdempotent(t *testing.T) {
	store := NewFileMediaStore()

	// ReleaseAll on non-existent scope should not error
	if err := store.ReleaseAll("nonexistent"); err != nil {
		t.Fatalf("ReleaseAll on empty scope should not error: %v", err)
	}

	// Create and release, then release again
	dir := t.TempDir()
	path := createTempFile(t, dir, "file.jpg")
	_, _ = store.Store(path, MediaMeta{Source: "test"}, "scope1")

	if err := store.ReleaseAll("scope1"); err != nil {
		t.Fatalf("first ReleaseAll failed: %v", err)
	}
	if err := store.ReleaseAll("scope1"); err != nil {
		t.Fatalf("second ReleaseAll should not error: %v", err)
	}
}

func TestReleaseAllCleansMappingsIfRefsMissing(t *testing.T) {
	dir := t.TempDir()
	store := NewFileMediaStore()

	path := createTempFile(t, dir, "file.jpg")
	ref, err := store.Store(path, MediaMeta{Source: "test"}, "scope1")
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Simulate internal inconsistency: scopeToRefs/refToScope contains ref but refs map doesn't.
	store.mu.Lock()
	delete(store.refs, ref)
	store.mu.Unlock()

	if err := store.ReleaseAll("scope1"); err != nil {
		t.Fatalf("ReleaseAll failed: %v", err)
	}

	// ReleaseAll should still clean mappings (even if it can't delete the file without the path).
	store.mu.RLock()
	defer store.mu.RUnlock()
	if _, ok := store.refToScope[ref]; ok {
		t.Error("refToScope should not contain ref after ReleaseAll")
	}
	if _, ok := store.scopeToRefs["scope1"]; ok {
		t.Error("scopeToRefs should not contain scope1 after ReleaseAll")
	}
}

func TestStoreNonexistentFile(t *testing.T) {
	store := NewFileMediaStore()

	_, err := store.Store("/nonexistent/path/file.jpg", MediaMeta{Source: "test"}, "scope1")
	if err == nil {
		t.Error("Store should fail for nonexistent file")
	}
	// Error message should include the underlying os error, not just "file does not exist"
	if !strings.Contains(err.Error(), "no such file or directory") &&
		!strings.Contains(err.Error(), "cannot find") {
		t.Errorf("Error should contain OS error detail, got: %v", err)
	}
}

func TestResolveWithMeta(t *testing.T) {
	dir := t.TempDir()
	store := NewFileMediaStore()

	path := createTempFile(t, dir, "image.png")
	meta := MediaMeta{
		Filename:    "image.png",
		ContentType: "image/png",
		Source:      "telegram",
	}

	ref, err := store.Store(path, meta, "scope1")
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	resolvedPath, resolvedMeta, err := store.ResolveWithMeta(ref)
	if err != nil {
		t.Fatalf("ResolveWithMeta failed: %v", err)
	}
	if resolvedPath != path {
		t.Errorf("ResolveWithMeta path = %q, want %q", resolvedPath, path)
	}
	if resolvedMeta.Filename != meta.Filename {
		t.Errorf("ResolveWithMeta Filename = %q, want %q", resolvedMeta.Filename, meta.Filename)
	}
	if resolvedMeta.ContentType != meta.ContentType {
		t.Errorf("ResolveWithMeta ContentType = %q, want %q", resolvedMeta.ContentType, meta.ContentType)
	}
	if resolvedMeta.Source != meta.Source {
		t.Errorf("ResolveWithMeta Source = %q, want %q", resolvedMeta.Source, meta.Source)
	}

	// Unknown ref should fail
	_, _, err = store.ResolveWithMeta("media://nonexistent")
	if err == nil {
		t.Error("ResolveWithMeta should fail for unknown ref")
	}
}

func TestConcurrentSafety(t *testing.T) {
	dir := t.TempDir()
	store := NewFileMediaStore()

	const goroutines = 20
	const filesPerGoroutine = 5

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func(gIdx int) {
			defer wg.Done()
			scope := strings.Repeat("s", gIdx+1)

			for i := 0; i < filesPerGoroutine; i++ {
				path := createTempFile(t, dir, strings.Repeat("f", gIdx*filesPerGoroutine+i+1)+".tmp")
				ref, err := store.Store(path, MediaMeta{Source: "test"}, scope)
				if err != nil {
					t.Errorf("Store failed: %v", err)
					return
				}

				if _, err := store.Resolve(ref); err != nil {
					t.Errorf("Resolve failed: %v", err)
				}
			}

			if err := store.ReleaseAll(scope); err != nil {
				t.Errorf("ReleaseAll failed: %v", err)
			}
		}(g)
	}

	wg.Wait()
}

// --- TTL cleanup tests ---

func newTestStoreWithCleanup(maxAge time.Duration) *FileMediaStore {
	s := NewFileMediaStoreWithCleanup(MediaCleanerConfig{
		Enabled:  true,
		MaxAge:   maxAge,
		Interval: time.Hour, // won't tick in tests
	})
	return s
}

func TestCleanExpiredRemovesOldEntries(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	store := newTestStoreWithCleanup(10 * time.Minute)
	store.nowFunc = func() time.Time { return now.Add(-20 * time.Minute) }

	path := createTempFile(t, dir, "old.jpg")
	ref, err := store.Store(path, MediaMeta{Source: "test"}, "scope1")
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Advance clock to present
	store.nowFunc = func() time.Time { return now }
	removed := store.CleanExpired()

	if removed != 1 {
		t.Errorf("expected 1 removed, got %d", removed)
	}
	if _, err := store.Resolve(ref); err == nil {
		t.Error("expired ref should be unresolvable")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("expired file should be deleted")
	}
}

func TestCleanExpiredKeepsNonExpired(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	store := newTestStoreWithCleanup(10 * time.Minute)
	store.nowFunc = func() time.Time { return now }

	path := createTempFile(t, dir, "fresh.jpg")
	ref, err := store.Store(path, MediaMeta{Source: "test"}, "scope1")
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	removed := store.CleanExpired()
	if removed != 0 {
		t.Errorf("expected 0 removed, got %d", removed)
	}

	if _, err := store.Resolve(ref); err != nil {
		t.Errorf("fresh ref should still resolve: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Error("fresh file should still exist")
	}
}

func TestCleanExpiredMixedAges(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	store := newTestStoreWithCleanup(10 * time.Minute)

	// Store old entry
	store.nowFunc = func() time.Time { return now.Add(-20 * time.Minute) }
	oldPath := createTempFile(t, dir, "old.jpg")
	oldRef, _ := store.Store(oldPath, MediaMeta{Source: "test"}, "scope1")

	// Store fresh entry
	store.nowFunc = func() time.Time { return now }
	freshPath := createTempFile(t, dir, "fresh.jpg")
	freshRef, _ := store.Store(freshPath, MediaMeta{Source: "test"}, "scope1")

	removed := store.CleanExpired()
	if removed != 1 {
		t.Errorf("expected 1 removed, got %d", removed)
	}

	if _, err := store.Resolve(oldRef); err == nil {
		t.Error("old ref should be gone")
	}
	if _, err := store.Resolve(freshRef); err != nil {
		t.Errorf("fresh ref should still resolve: %v", err)
	}
}

func TestCleanExpiredCleansEmptyScopes(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	store := newTestStoreWithCleanup(10 * time.Minute)

	// Store old entry as the only one in scope
	store.nowFunc = func() time.Time { return now.Add(-20 * time.Minute) }
	path := createTempFile(t, dir, "only.jpg")
	store.Store(path, MediaMeta{Source: "test"}, "lonely_scope")

	store.nowFunc = func() time.Time { return now }
	store.CleanExpired()

	store.mu.RLock()
	defer store.mu.RUnlock()
	if _, ok := store.scopeToRefs["lonely_scope"]; ok {
		t.Error("empty scope should be cleaned up")
	}
}

func TestStartStopLifecycle(t *testing.T) {
	store := NewFileMediaStoreWithCleanup(MediaCleanerConfig{
		Enabled:  true,
		MaxAge:   time.Minute,
		Interval: 50 * time.Millisecond,
	})

	// Start and stop should not panic
	store.Start()
	// Double start should not spawn a second goroutine
	store.Start()
	time.Sleep(100 * time.Millisecond)
	store.Stop()

	// Double stop should not panic
	store.Stop()
}

func TestCleanExpiredZeroMaxAge(t *testing.T) {
	store := NewFileMediaStoreWithCleanup(MediaCleanerConfig{
		Enabled:  true,
		MaxAge:   0,
		Interval: time.Hour,
	})

	dir := t.TempDir()
	path := createTempFile(t, dir, "file.jpg")
	ref, _ := store.Store(path, MediaMeta{Source: "test"}, "scope1")

	// Zero MaxAge should be a no-op
	removed := store.CleanExpired()
	if removed != 0 {
		t.Errorf("expected 0 removed with zero MaxAge, got %d", removed)
	}
	if _, err := store.Resolve(ref); err != nil {
		t.Errorf("ref should still resolve: %v", err)
	}
}

func TestStartDisabledIsNoop(t *testing.T) {
	store := NewFileMediaStoreWithCleanup(MediaCleanerConfig{
		Enabled:  false,
		MaxAge:   time.Minute,
		Interval: time.Minute,
	})
	// Should not start any goroutine or panic
	store.Start()
	store.Stop()
}

func TestStartZeroIntervalNoPanic(t *testing.T) {
	store := NewFileMediaStoreWithCleanup(MediaCleanerConfig{
		Enabled:  true,
		MaxAge:   time.Minute,
		Interval: 0,
	})
	// Zero interval should not panic (time.NewTicker panics on <= 0)
	store.Start()
	store.Stop()
}

func TestStartZeroMaxAgeNoPanic(t *testing.T) {
	store := NewFileMediaStoreWithCleanup(MediaCleanerConfig{
		Enabled:  true,
		MaxAge:   0,
		Interval: time.Minute,
	})
	store.Start()
	store.Stop()
}

func TestConcurrentCleanupSafety(t *testing.T) {
	dir := t.TempDir()
	store := newTestStoreWithCleanup(50 * time.Millisecond)
	store.nowFunc = time.Now

	const workers = 10
	const ops = 20
	var wg sync.WaitGroup
	wg.Add(workers * 4)

	// Store workers
	for w := 0; w < workers; w++ {
		go func(wIdx int) {
			defer wg.Done()
			scope := fmt.Sprintf("scope-%d", wIdx)
			for i := 0; i < ops; i++ {
				p := createTempFile(t, dir, fmt.Sprintf("w%d-f%d.tmp", wIdx, i))
				store.Store(p, MediaMeta{Source: "test"}, scope)
			}
		}(w)
	}

	// Resolve workers
	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			for i := 0; i < ops; i++ {
				store.Resolve("media://nonexistent")
			}
		}()
	}

	// ReleaseAll workers
	for w := 0; w < workers; w++ {
		go func(wIdx int) {
			defer wg.Done()
			for i := 0; i < ops; i++ {
				store.ReleaseAll(fmt.Sprintf("scope-%d", wIdx))
			}
		}(w)
	}

	// CleanExpired workers
	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			for i := 0; i < ops; i++ {
				store.CleanExpired()
			}
		}()
	}

	wg.Wait()
}

func TestRefToScopeConsistency(t *testing.T) {
	dir := t.TempDir()
	store := NewFileMediaStore()

	// Store entries in two scopes
	ref1, _ := store.Store(createTempFile(t, dir, "a.jpg"), MediaMeta{Source: "test"}, "s1")
	ref2, _ := store.Store(createTempFile(t, dir, "b.jpg"), MediaMeta{Source: "test"}, "s1")
	ref3, _ := store.Store(createTempFile(t, dir, "c.jpg"), MediaMeta{Source: "test"}, "s2")

	store.mu.RLock()
	checkRef := func(ref, expectedScope string) {
		t.Helper()
		if scope, ok := store.refToScope[ref]; !ok || scope != expectedScope {
			t.Errorf("refToScope[%s] = %q, want %q", ref, scope, expectedScope)
		}
	}
	checkRef(ref1, "s1")
	checkRef(ref2, "s1")
	checkRef(ref3, "s2")
	store.mu.RUnlock()

	// Release s1 and verify refToScope is cleaned
	store.ReleaseAll("s1")

	store.mu.RLock()
	defer store.mu.RUnlock()
	if _, ok := store.refToScope[ref1]; ok {
		t.Error("refToScope should not contain ref1 after ReleaseAll")
	}
	if _, ok := store.refToScope[ref2]; ok {
		t.Error("refToScope should not contain ref2 after ReleaseAll")
	}
	if _, ok := store.refToScope[ref3]; !ok {
		t.Error("refToScope should still contain ref3")
	}
}
