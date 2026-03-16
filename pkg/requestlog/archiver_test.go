package requestlog

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseInterval(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
		hasError bool
	}{
		{"hours format", "24h", 24 * time.Hour, false},
		{"single hour", "1h", 1 * time.Hour, false},
		{"minutes format", "30m", 30 * time.Minute, false},
		{"day format", "1day", 24 * time.Hour, false},
		{"days format", "2days", 48 * time.Hour, false},
		{"empty string", "", 24 * time.Hour, false},
		{"invalid", "invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseInterval(tt.input)
			if tt.hasError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestArchiver_Archive(t *testing.T) {
	tmpDir := t.TempDir()

	oldFile := filepath.Join(tmpDir, "requests-2024-01-01.jsonl")
	content := []byte(`{"timestamp":"2024-01-01T00:00:00Z","request_id":"1","channel":"test"}` + "\n")
	if err := os.WriteFile(oldFile, content, 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	oldTime := time.Now().Add(-40 * 24 * time.Hour)
	if err := os.Chtimes(oldFile, oldTime, oldTime); err != nil {
		t.Fatalf("Chtimes failed: %v", err)
	}

	cfg := Config{
		Enabled:         true,
		RetentionDays:   30,
		CompressArchive: true,
		MaxFiles:        100,
		MaxFileSizeMB:   100,
	}

	archiver := NewArchiver(cfg, tmpDir)
	if err := archiver.Archive(); err != nil {
		t.Fatalf("Archive failed: %v", err)
	}

	compressedFile := oldFile + ".tar.gz"
	if _, err := os.Stat(compressedFile); os.IsNotExist(err) {
		t.Error("expected compressed file to exist")
	}

	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Error("expected original file to be removed after compression")
	}
}

func TestArchiver_ArchiveNoCompress(t *testing.T) {
	tmpDir := t.TempDir()

	oldFile := filepath.Join(tmpDir, "requests-2024-01-01.jsonl")
	content := []byte(`{"timestamp":"2024-01-01T00:00:00Z","request_id":"1","channel":"test"}` + "\n")
	if err := os.WriteFile(oldFile, content, 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	oldTime := time.Now().Add(-40 * 24 * time.Hour)
	if err := os.Chtimes(oldFile, oldTime, oldTime); err != nil {
		t.Fatalf("Chtimes failed: %v", err)
	}

	cfg := Config{
		Enabled:         true,
		RetentionDays:   30,
		CompressArchive: false,
		MaxFiles:        100,
		MaxFileSizeMB:   100,
	}

	archiver := NewArchiver(cfg, tmpDir)
	if err := archiver.Archive(); err != nil {
		t.Fatalf("Archive failed: %v", err)
	}

	compressedFile := oldFile + ".tar.gz"
	if _, err := os.Stat(compressedFile); !os.IsNotExist(err) {
		t.Error("expected compressed file NOT to exist when CompressArchive is false")
	}
}

func TestArchiver_CleanupOldFiles(t *testing.T) {
	tmpDir := t.TempDir()

	for i := range 5 {
		filename := filepath.Join(tmpDir, "requests-2024-01-"+padInt(i)+".jsonl")
		content := make([]byte, 1024*1024)
		if err := os.WriteFile(filename, content, 0o644); err != nil {
			t.Fatalf("WriteFile failed: %v", err)
		}

		oldTime := time.Now().Add(-time.Duration(i+1) * 24 * time.Hour)
		if err := os.Chtimes(filename, oldTime, oldTime); err != nil {
			t.Fatalf("Chtimes failed: %v", err)
		}
	}

	cfg := Config{
		Enabled:         true,
		RetentionDays:   30,
		CompressArchive: false,
		MaxFiles:        2,
		MaxFileSizeMB:   1,
	}

	archiver := NewArchiver(cfg, tmpDir)
	if err := archiver.cleanupOldFiles(); err != nil {
		t.Fatalf("cleanupOldFiles failed: %v", err)
	}

	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}

	if len(files) > 2 {
		t.Errorf("expected at most 2 files after cleanup, got %d", len(files))
	}
}

func TestArchiver_CompressFile(t *testing.T) {
	tmpDir := t.TempDir()

	srcFile := filepath.Join(tmpDir, "test.jsonl")
	content := []byte(`{"test":"data"}` + "\n")
	if err := os.WriteFile(srcFile, content, 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	cfg := Config{CompressArchive: true}
	archiver := NewArchiver(cfg, tmpDir)

	if err := archiver.compressFile(srcFile); err != nil {
		t.Fatalf("compressFile failed: %v", err)
	}

	dstFile := srcFile + ".tar.gz"
	f, err := os.Open(dstFile)
	if err != nil {
		t.Fatalf("Open compressed file failed: %v", err)
	}
	defer f.Close()

	gzReader, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("gzip.NewReader failed: %v", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	header, err := tarReader.Next()
	if err != nil {
		t.Fatalf("tarReader.Next failed: %v", err)
	}

	if header.Name != "test.jsonl" {
		t.Errorf("expected header name 'test.jsonl', got %q", header.Name)
	}
}

func TestArchiver_ArchiveRecentFiles(t *testing.T) {
	tmpDir := t.TempDir()

	recentFile := filepath.Join(tmpDir, "requests-"+time.Now().Format("2006-01-02")+".jsonl")
	content := []byte(`{"timestamp":"2024-01-01T00:00:00Z","request_id":"1","channel":"test"}` + "\n")
	if err := os.WriteFile(recentFile, content, 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	cfg := Config{
		Enabled:         true,
		RetentionDays:   30,
		CompressArchive: true,
		MaxFiles:        100,
		MaxFileSizeMB:   100,
	}

	archiver := NewArchiver(cfg, tmpDir)
	if err := archiver.Archive(); err != nil {
		t.Fatalf("Archive failed: %v", err)
	}

	compressedFile := recentFile + ".tar.gz"
	if _, err := os.Stat(compressedFile); !os.IsNotExist(err) {
		t.Error("expected recent file NOT to be compressed")
	}

	if _, err := os.Stat(recentFile); os.IsNotExist(err) {
		t.Error("expected recent file to still exist")
	}
}

func padInt(i int) string {
	if i < 10 {
		return "0" + string(rune('0'+i))
	}
	return string(rune('0'+i/10)) + string(rune('0'+i%10))
}
