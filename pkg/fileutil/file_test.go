package fileutil

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestWriteFileAtomic_Basic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	data := []byte("hello picoclaw")

	err := WriteFileAtomic(path, data, 0o644)
	if err != nil {
		t.Fatalf("WriteFileAtomic failed: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(got) != string(data) {
		t.Errorf("got %q, want %q", got, data)
	}
}

func TestWriteFileAtomic_Permissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secret.txt")

	err := WriteFileAtomic(path, []byte("secret"), 0o600)
	if err != nil {
		t.Fatalf("WriteFileAtomic failed: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	// On Unix, check file mode (ignoring directory bits)
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("permissions = %o, want %o", got, 0o600)
	}
}

func TestWriteFileAtomic_Overwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "overwrite.txt")

	// Write initial content
	if err := WriteFileAtomic(path, []byte("old"), 0o644); err != nil {
		t.Fatalf("first write failed: %v", err)
	}

	// Overwrite
	if err := WriteFileAtomic(path, []byte("new"), 0o644); err != nil {
		t.Fatalf("second write failed: %v", err)
	}

	got, _ := os.ReadFile(path)
	if string(got) != "new" {
		t.Errorf("got %q after overwrite, want %q", got, "new")
	}
}

func TestWriteFileAtomic_EmptyData(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.txt")

	err := WriteFileAtomic(path, []byte{}, 0o644)
	if err != nil {
		t.Fatalf("WriteFileAtomic with empty data failed: %v", err)
	}

	got, _ := os.ReadFile(path)
	if len(got) != 0 {
		t.Errorf("expected empty file, got %d bytes", len(got))
	}
}

func TestWriteFileAtomic_CreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a", "b", "c", "deep.txt")

	err := WriteFileAtomic(path, []byte("deep"), 0o644)
	if err != nil {
		t.Fatalf("WriteFileAtomic with nested dirs failed: %v", err)
	}

	got, _ := os.ReadFile(path)
	if string(got) != "deep" {
		t.Errorf("got %q, want %q", got, "deep")
	}
}

func TestWriteFileAtomic_NoTempFileOnSuccess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "clean.txt")

	if err := WriteFileAtomic(path, []byte("data"), 0o644); err != nil {
		t.Fatalf("WriteFileAtomic failed: %v", err)
	}

	// Verify no temp files remain
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.Name() != "clean.txt" {
			t.Errorf("unexpected file remaining: %s", e.Name())
		}
	}
}

func TestWriteFileAtomic_LargeFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "large.bin")

	// 1MB of data
	data := make([]byte, 1<<20)
	for i := range data {
		data[i] = byte(i % 256)
	}

	if err := WriteFileAtomic(path, data, 0o644); err != nil {
		t.Fatalf("WriteFileAtomic with large file failed: %v", err)
	}

	got, _ := os.ReadFile(path)
	if len(got) != len(data) {
		t.Errorf("file size = %d, want %d", len(got), len(data))
	}
}

func TestWriteFileAtomic_Concurrent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "concurrent.txt")

	var wg sync.WaitGroup
	errs := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			data := []byte(string(rune('A' + n)))
			if err := WriteFileAtomic(path, data, 0o644); err != nil {
				errs <- err
			}
		}(i)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent write error: %v", err)
	}

	// File should exist and contain exactly 1 byte (last writer wins)
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile after concurrent writes failed: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("expected 1 byte after concurrent writes, got %d", len(got))
	}
}

func TestWriteFileAtomic_InvalidPath(t *testing.T) {
	// /dev/null/impossible is not a valid path on any OS
	err := WriteFileAtomic("/dev/null/impossible/file.txt", []byte("data"), 0o644)
	if err == nil {
		t.Error("expected error for invalid path, got nil")
	}
}
