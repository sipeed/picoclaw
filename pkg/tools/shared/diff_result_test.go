package toolshared

import (
	"bytes"
	"strings"
	"testing"
)

func TestDiffResult_UserVisibleUnifiedDiff(t *testing.T) {
	result := DiffResult("/tmp/example.txt", []byte("alpha\nbeta\ngamma\n"), []byte("alpha\nbeta 2\ngamma\n"))

	if result == nil {
		t.Fatal("DiffResult() returned nil")
	}
	if result.Silent {
		t.Fatal("expected DiffResult to be user-visible")
	}
	if result.IsError {
		t.Fatal("expected DiffResult to be successful")
	}
	if result.ForLLM == result.ForUser {
		t.Fatal("expected compact model context instead of duplicating the full diff")
	}
	if len(result.ForLLM) >= len(result.ForUser) {
		t.Fatalf("expected ForLLM to stay smaller than ForUser, got %d vs %d", len(result.ForLLM), len(result.ForUser))
	}
	if result.ForLLM != "File edited: /tmp/example.txt" {
		t.Fatalf("expected compact summary in ForLLM, got %q", result.ForLLM)
	}

	for _, want := range []string{
		"File edited: /tmp/example.txt",
		"```diff",
		"--- a/tmp/example.txt",
		"+++ b/tmp/example.txt",
		"@@ -1,3 +1,3 @@",
		" alpha",
		"-beta",
		"+beta 2",
		" gamma",
	} {
		if !strings.Contains(result.ForUser, want) {
			t.Fatalf("DiffResult output missing %q:\n%s", want, result.ForUser)
		}
	}
}

func TestBuildUnifiedDiff_NoContentChange(t *testing.T) {
	diff, err := buildUnifiedDiff("test.txt", []byte("same\n"), []byte("same\n"))
	if err != nil {
		t.Fatalf("buildUnifiedDiff() error = %v", err)
	}
	if diff != noContentChangeDiffMessage {
		t.Fatalf("buildUnifiedDiff() = %q, want %q", diff, noContentChangeDiffMessage)
	}
}

func TestBuildUnifiedDiff_PreservesTrailingNewlineRemoval(t *testing.T) {
	diff, err := buildUnifiedDiff("test.txt", []byte("same\n"), []byte("same"))
	if err != nil {
		t.Fatalf("buildUnifiedDiff() error = %v", err)
	}

	for _, want := range []string{
		"--- a/test.txt",
		"+++ b/test.txt",
		" same",
		"+" + noNewlineAtEOFMarker,
	} {
		if !strings.Contains(diff, want) {
			t.Fatalf("buildUnifiedDiff() missing %q:\n%s", want, diff)
		}
	}
}

func TestBuildUnifiedDiff_PreservesTrailingNewlineAddition(t *testing.T) {
	diff, err := buildUnifiedDiff("test.txt", []byte("same"), []byte("same\n"))
	if err != nil {
		t.Fatalf("buildUnifiedDiff() error = %v", err)
	}

	for _, want := range []string{
		"--- a/test.txt",
		"+++ b/test.txt",
		" same",
		"-" + noNewlineAtEOFMarker,
	} {
		if !strings.Contains(diff, want) {
			t.Fatalf("buildUnifiedDiff() missing %q:\n%s", want, diff)
		}
	}
}

func TestBuildUnifiedDiff_UsesNormalizedDisplayPaths(t *testing.T) {
	diff, err := buildUnifiedDiff("/tmp/nested/example.txt", []byte("before\n"), []byte("after\n"))
	if err != nil {
		t.Fatalf("buildUnifiedDiff() error = %v", err)
	}

	for _, want := range []string{
		"--- a/tmp/nested/example.txt",
		"+++ b/tmp/nested/example.txt",
	} {
		if !strings.Contains(diff, want) {
			t.Fatalf("buildUnifiedDiff() missing %q:\n%s", want, diff)
		}
	}
}

func TestDiffResult_SkipsPreviewForLargeFiles(t *testing.T) {
	before := bytes.Repeat([]byte("a"), maxDiffInputBytes+1)
	after := bytes.Repeat([]byte("b"), maxDiffInputBytes+1)

	result := DiffResult("big.txt", before, after)

	if !result.Silent {
		t.Fatal("expected large diff previews to be skipped silently")
	}
	if result.ForUser != "" {
		t.Fatalf("expected no user-facing preview when skipped, got %q", result.ForUser)
	}
	if !strings.Contains(result.ForLLM, diffPreviewSkippedMessage) {
		t.Fatalf("expected skipped-preview note, got %q", result.ForLLM)
	}
}

func TestDiffResult_TruncatesLargeUserPreview(t *testing.T) {
	after := []byte(strings.Repeat("abcd", maxUserDiffPreviewBytes/4) + "\n")

	result := DiffResult("preview.txt", []byte("before\n"), after)

	if result.Silent {
		t.Fatal("expected preview to remain user-visible below the input caps")
	}
	if !strings.Contains(result.ForUser, diffPreviewTruncatedNote) {
		t.Fatalf("expected truncated preview note, got %q", result.ForUser)
	}
	if !strings.Contains(result.ForLLM, diffPreviewTruncatedNote) {
		t.Fatalf("expected model summary to mention truncation, got %q", result.ForLLM)
	}
	if len(result.ForLLM) >= len(result.ForUser) {
		t.Fatalf("expected ForLLM to remain smaller than ForUser, "+
			"got %d vs %d", len(result.ForLLM), len(result.ForUser))
	}
}

func TestDiffDisplayPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "absolute path",
			path: "/tmp/example.txt",
			want: "tmp/example.txt",
		},
		{
			name: "relative path",
			path: "pkg/tools/fs/edit.go",
			want: "pkg/tools/fs/edit.go",
		},
		{
			name: "empty path",
			path: "",
			want: "file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := diffDisplayPath(tt.path); got != tt.want {
				t.Fatalf("diffDisplayPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}
