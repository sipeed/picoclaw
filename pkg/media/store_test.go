package media

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
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

func TestStoreNonexistentFile(t *testing.T) {
	store := NewFileMediaStore()

	_, err := store.Store("/nonexistent/path/file.jpg", MediaMeta{Source: "test"}, "scope1")
	if err == nil {
		t.Error("Store should fail for nonexistent file")
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
