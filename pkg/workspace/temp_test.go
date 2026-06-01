package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTempDir(t *testing.T) {
	workspace := filepath.Join("root", "workspace")
	if got := TempDir("  " + workspace + "  "); got != filepath.Join(workspace, "tmp") {
		t.Fatalf("TempDir() = %q, want workspace tmp path", got)
	}
	if got := TempDir(" "); got != "" {
		t.Fatalf("TempDir(empty) = %q, want empty", got)
	}
}

func TestEnsureTempDir(t *testing.T) {
	workspace := t.TempDir()
	got, err := EnsureTempDir(workspace)
	if err != nil {
		t.Fatalf("EnsureTempDir() error = %v", err)
	}
	want := filepath.Join(workspace, "tmp")
	if got != want {
		t.Fatalf("EnsureTempDir() = %q, want %q", got, want)
	}
	info, err := os.Stat(want)
	if err != nil {
		t.Fatalf("tmp dir not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("tmp path is not a directory")
	}
}
