package pathutil

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveWorkspacePath_WithRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspaces")

	tests := []struct {
		name    string
		path    string
		wantErr string // substring of expected error; empty means success
		wantRel string // expected relative suffix under root (checked on success)
	}{
		// --- valid paths ---
		{
			name:    "relative subdirectory",
			path:    "tenant1",
			wantRel: "tenant1",
		},
		{
			name:    "nested subdirectory",
			path:    "tenant1/project/sub",
			wantRel: filepath.Join("tenant1", "project", "sub"),
		},

		// --- empty / dot ---
		{
			name:    "empty path rejected",
			path:    "",
			wantErr: "subdirectory of workspace_root",
		},
		{
			name:    "dot path rejected",
			path:    ".",
			wantErr: "subdirectory of workspace_root",
		},

		// --- traversal: bare .. ---
		{
			name:    "bare dotdot rejected",
			path:    "..",
			wantErr: "traversal",
		},

		// --- traversal: starts with ../ ---
		{
			name:    "starts with dotdot slash",
			path:    "../escape",
			wantErr: "traversal",
		},
		{
			name:    "starts with dotdot backslash",
			path:    `..\\escape`,
			wantErr: "traversal",
		},

		// --- traversal: ends with /.. ---
		{
			name:    "ends with slash dotdot",
			path:    "foo/..",
			wantErr: "traversal",
		},
		{
			name:    "ends with backslash dotdot",
			path:    `foo\..`,
			wantErr: "traversal",
		},

		// --- traversal: contains /../ ---
		{
			name:    "contains slash dotdot slash",
			path:    "a/../escape",
			wantErr: "traversal",
		},
		{
			name:    "contains backslash dotdot backslash",
			path:    `a\..\escape`,
			wantErr: "traversal",
		},
		{
			name:    "deep traversal",
			path:    "a/b/../../escape",
			wantErr: "traversal",
		},

		// --- absolute paths ---
		{
			name:    "unix absolute path rejected",
			path:    "/etc/passwd",
			wantErr: "must be relative",
		},
	}

	if runtime.GOOS == "windows" {
		tests = append(tests, struct {
			name    string
			path    string
			wantErr string
			wantRel string
		}{
			name:    "windows absolute path rejected",
			path:    `C:\Windows\System32`,
			wantErr: "must be relative",
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveWorkspacePath(root, tt.path)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil (result: %s)", tt.wantErr, got)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got: %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			absRoot, absErr := filepath.Abs(root)
			require.NoError(t, absErr)
			want := filepath.Join(absRoot, tt.wantRel)
			if got != want {
				t.Fatalf("expected %q, got %q", want, got)
			}
		})
	}
}

func TestResolveWorkspacePath_NoRoot(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr string
	}{
		{
			name: "relative path resolved via Abs",
			path: "some/dir",
		},
		{
			name:    "empty path rejected",
			path:    "",
			wantErr: "empty",
		},
		{
			name:    "traversal rejected even without root",
			path:    "../escape",
			wantErr: "traversal",
		},
		{
			name:    "bare dotdot rejected even without root",
			path:    "..",
			wantErr: "traversal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveWorkspacePath("", tt.path)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil (result: %s)", tt.wantErr, got)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got: %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Without root, should return filepath.Abs(path)
			want, absErr := filepath.Abs(tt.path)
			require.NoError(t, absErr)
			if got != want {
				t.Fatalf("expected %q, got %q", want, got)
			}
		})
	}
}

func TestContainsTraversal(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		// positive cases
		{"..", true},
		{"../foo", true},
		{`..\\foo`, true},
		{"foo/..", true},
		{`foo\..`, true},
		{"foo/../bar", true},
		{`foo\..\bar`, true},
		{"a/b/../../c", true},

		// negative cases — dotdot as part of a name is fine
		{"..hidden", false},
		{"foo..bar", false},
		{"tenant1", false},
		{"a/b/c", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := containsTraversal(tt.path)
			if got != tt.want {
				t.Fatalf("containsTraversal(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
