package fstools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSearchFilesTool_ContentSearch(t *testing.T) {
	tmpDir := t.TempDir()
	mustWriteSearchFile(t, tmpDir, "main.go", "package main\n\nfunc main() {\n\tprintln(\"needle\")\n}\n")
	mustWriteSearchFile(t, tmpDir, "README.md", "needle in docs\n")

	tool := NewSearchFilesTool(tmpDir, true, 0)
	result := tool.Execute(context.Background(), map[string]any{
		"pattern":   "needle",
		"path":      ".",
		"file_glob": "*.go",
	})

	if result.IsError {
		t.Fatalf("search_files failed: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "main.go:") {
		t.Fatalf("expected Go match, got:\n%s", result.ForLLM)
	}
	if strings.Contains(result.ForLLM, "README.md") {
		t.Fatalf("file_glob should exclude README.md, got:\n%s", result.ForLLM)
	}
}

func TestSearchFilesTool_FilesSearch(t *testing.T) {
	tmpDir := t.TempDir()
	mustWriteSearchFile(t, tmpDir, "cmd/app/main.go", "package main\n")
	mustWriteSearchFile(t, tmpDir, "pkg/app/app_test.go", "package app\n")
	mustWriteSearchFile(t, tmpDir, "notes.txt", "ignore\n")

	tool := NewSearchFilesTool(tmpDir, true, 0)
	result := tool.Execute(context.Background(), map[string]any{
		"pattern": "*.go",
		"target":  "files",
		"path":    ".",
	})

	if result.IsError {
		t.Fatalf("search_files failed: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "cmd/app/main.go") ||
		!strings.Contains(result.ForLLM, "pkg/app/app_test.go") {
		t.Fatalf("expected Go files, got:\n%s", result.ForLLM)
	}
	if strings.Contains(result.ForLLM, "notes.txt") {
		t.Fatalf("unexpected notes.txt match:\n%s", result.ForLLM)
	}
}

func TestSearchFilesTool_CountMode(t *testing.T) {
	tmpDir := t.TempDir()
	mustWriteSearchFile(t, tmpDir, "a.txt", "needle\nneedle\n")
	mustWriteSearchFile(t, tmpDir, "b.txt", "needle\n")

	tool := NewSearchFilesTool(tmpDir, true, 0)
	result := tool.Execute(context.Background(), map[string]any{
		"pattern":     "needle",
		"output_mode": "count",
	})

	if result.IsError {
		t.Fatalf("search_files failed: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "a.txt: 2") ||
		!strings.Contains(result.ForLLM, "b.txt: 1") {
		t.Fatalf("expected count output, got:\n%s", result.ForLLM)
	}
}

func TestSearchFilesTool_RespectsGitignoreByDefault(t *testing.T) {
	tmpDir := t.TempDir()
	mustWriteSearchFile(t, tmpDir, ".gitignore", ".env\nignored/\n*.log\n")
	mustWriteSearchFile(t, tmpDir, ".env", "secret needle\n")
	mustWriteSearchFile(t, tmpDir, "ignored/file.txt", "ignored needle\n")
	mustWriteSearchFile(t, tmpDir, "debug.log", "logged needle\n")
	mustWriteSearchFile(t, tmpDir, "visible.txt", "visible needle\n")

	tool := NewSearchFilesTool(tmpDir, true, 0)
	result := tool.Execute(context.Background(), map[string]any{
		"pattern": "needle",
		"path":    ".",
	})

	if result.IsError {
		t.Fatalf("search_files failed: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "visible.txt") {
		t.Fatalf("expected visible match, got:\n%s", result.ForLLM)
	}
	for _, ignored := range []string{".env", "ignored/file.txt", "debug.log"} {
		if strings.Contains(result.ForLLM, ignored) {
			t.Fatalf("expected %s to be ignored, got:\n%s", ignored, result.ForLLM)
		}
	}
}

func TestSearchFilesTool_IncludeIgnoredFindsGitignoredFiles(t *testing.T) {
	tmpDir := t.TempDir()
	mustWriteSearchFile(t, tmpDir, ".gitignore", ".env\nignored/\n*.log\n")
	mustWriteSearchFile(t, tmpDir, ".env", "secret needle\n")
	mustWriteSearchFile(t, tmpDir, "ignored/file.txt", "ignored needle\n")
	mustWriteSearchFile(t, tmpDir, "debug.log", "logged needle\n")
	mustWriteSearchFile(t, tmpDir, "visible.txt", "visible needle\n")

	tool := NewSearchFilesTool(tmpDir, true, 0)
	result := tool.Execute(context.Background(), map[string]any{
		"pattern":         "needle",
		"path":            ".",
		"include_ignored": true,
	})

	if result.IsError {
		t.Fatalf("search_files failed: %s", result.ForLLM)
	}
	for _, expected := range []string{".env", "ignored/file.txt", "debug.log", "visible.txt"} {
		if !strings.Contains(result.ForLLM, expected) {
			t.Fatalf("expected %s with include_ignored, got:\n%s", expected, result.ForLLM)
		}
	}
}

func TestSearchFilesTool_ExplicitIgnoredFilePathStillRespectsGitignore(t *testing.T) {
	tmpDir := t.TempDir()
	mustWriteSearchFile(t, tmpDir, ".gitignore", ".env\n")
	mustWriteSearchFile(t, tmpDir, ".env", "secret needle\n")

	tool := NewSearchFilesTool(tmpDir, true, 0)
	result := tool.Execute(context.Background(), map[string]any{
		"pattern": "needle",
		"path":    ".env",
	})

	if result.IsError {
		t.Fatalf("search_files failed: %s", result.ForLLM)
	}
	if strings.Contains(result.ForLLM, ".env") {
		t.Fatalf("expected explicit ignored file to be skipped without include_ignored, got:\n%s", result.ForLLM)
	}

	result = tool.Execute(context.Background(), map[string]any{
		"pattern":         "needle",
		"path":            ".env",
		"include_ignored": true,
	})
	if result.IsError {
		t.Fatalf("search_files with include_ignored failed: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, ".env") {
		t.Fatalf("expected explicit ignored file with include_ignored, got:\n%s", result.ForLLM)
	}
}

func TestSearchFilesTool_RespectsNestedGitignore(t *testing.T) {
	tmpDir := t.TempDir()
	mustWriteSearchFile(t, tmpDir, "sub/.gitignore", "secret.txt\n")
	mustWriteSearchFile(t, tmpDir, "sub/secret.txt", "hidden needle\n")
	mustWriteSearchFile(t, tmpDir, "sub/public.txt", "visible needle\n")

	tool := NewSearchFilesTool(tmpDir, true, 0)
	result := tool.Execute(context.Background(), map[string]any{
		"pattern": "needle",
		"path":    ".",
	})

	if result.IsError {
		t.Fatalf("search_files failed: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "sub/public.txt") {
		t.Fatalf("expected public match, got:\n%s", result.ForLLM)
	}
	if strings.Contains(result.ForLLM, "sub/secret.txt") {
		t.Fatalf("expected nested ignored file to be skipped, got:\n%s", result.ForLLM)
	}
}

func TestSearchFilesTool_RestrictsOutsideWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("needle\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	tool := NewSearchFilesTool(tmpDir, true, 0)
	result := tool.Execute(context.Background(), map[string]any{
		"pattern": "needle",
		"path":    outside,
	})

	if !result.IsError {
		t.Fatalf("expected outside workspace search to fail")
	}
	if !strings.Contains(result.ForLLM, "outside the workspace") &&
		!strings.Contains(result.ForLLM, "access denied") &&
		!strings.Contains(result.ForLLM, "escapes workspace") {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
}

func mustWriteSearchFile(t *testing.T, root string, rel string, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
