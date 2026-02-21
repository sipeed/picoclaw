package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestPermissionStore_ApproveAndCheck(t *testing.T) {
	store := NewPermissionStore()
	store.Approve("/home/user/projects")

	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "exact dir", path: "/home/user/projects", want: true},
		{name: "child file", path: "/home/user/projects/main.go", want: true},
		{name: "nested child", path: "/home/user/projects/pkg/tools/foo.go", want: true},
		{name: "parent dir", path: "/home/user", want: false},
		{name: "sibling dir", path: "/home/user/documents", want: false},
		{name: "prefix overlap", path: "/home/user/projects-other/foo.go", want: false},
		{name: "unrelated path", path: "/tmp/foo", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := store.IsApproved(tt.path)
			if got != tt.want {
				t.Errorf("IsApproved(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestPermissionStore_MultipleApprovals(t *testing.T) {
	store := NewPermissionStore()
	store.Approve("/home/user/projects")
	store.Approve("/tmp/data")

	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "first approval child", path: "/home/user/projects/main.go", want: true},
		{name: "second approval child", path: "/tmp/data/file.csv", want: true},
		{name: "neither approved", path: "/var/log/syslog", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := store.IsApproved(tt.path)
			if got != tt.want {
				t.Errorf("IsApproved(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestPermissionStore_ConcurrentAccess(t *testing.T) {
	store := NewPermissionStore()
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			store.Approve(fmt.Sprintf("/dir/%d", n))
		}(i)
	}
	wg.Wait()

	// Concurrent reads
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			got := store.IsApproved(fmt.Sprintf("/dir/%d/file.txt", n))
			if !got {
				t.Errorf("expected /dir/%d/file.txt to be approved", n)
			}
		}(i)
	}
	wg.Wait()
}

func TestValidatePath_WithPermission(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	if err := os.MkdirAll(workspace, 0755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	outside := filepath.Join(root, "outside")
	if err := os.MkdirAll(outside, 0755); err != nil {
		t.Fatalf("failed to create outside dir: %v", err)
	}
	outsideFile := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(outsideFile, []byte("secret"), 0644); err != nil {
		t.Fatalf("failed to write secret file: %v", err)
	}

	ctx := context.Background()

	t.Run("approved by permFn", func(t *testing.T) {
		store := NewPermissionStore()
		approver := func(_ context.Context, _ string) (bool, error) {
			return true, nil
		}

		path, err := validatePathWithPermission(ctx, outsideFile, workspace, true, store, approver)
		if err != nil {
			t.Fatalf("expected success, got error: %v", err)
		}
		if path != outsideFile {
			t.Errorf("expected path %s, got %s", outsideFile, path)
		}
		// Verify it was cached
		if !store.IsApproved(outside) {
			t.Errorf("expected directory %s to be approved in store", outside)
		}
	})

	t.Run("denied by permFn", func(t *testing.T) {
		store := NewPermissionStore()
		denier := func(_ context.Context, _ string) (bool, error) {
			return false, nil
		}

		_, err := validatePathWithPermission(ctx, outsideFile, workspace, true, store, denier)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "denied permission") {
			t.Errorf("expected denied message, got: %v", err)
		}
	})

	t.Run("nil permFn returns descriptive error", func(t *testing.T) {
		store := NewPermissionStore()

		_, err := validatePathWithPermission(ctx, outsideFile, workspace, true, store, nil)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "outside the workspace") {
			t.Errorf("expected 'outside the workspace' message, got: %v", err)
		}
		if !strings.Contains(err.Error(), "Ask the user") {
			t.Errorf("expected 'Ask the user' hint, got: %v", err)
		}
	})

	t.Run("inside workspace still works", func(t *testing.T) {
		insideFile := filepath.Join(workspace, "hello.txt")
		if err := os.WriteFile(insideFile, []byte("hello"), 0644); err != nil {
			t.Fatalf("failed to write inside file: %v", err)
		}

		path, err := validatePathWithPermission(ctx, insideFile, workspace, true, nil, nil)
		if err != nil {
			t.Fatalf("expected success for inside-workspace path, got error: %v", err)
		}
		if path != insideFile {
			t.Errorf("expected path %s, got %s", insideFile, path)
		}
	})

	t.Run("permFn error propagates", func(t *testing.T) {
		store := NewPermissionStore()
		errFn := func(_ context.Context, _ string) (bool, error) {
			return false, fmt.Errorf("connection lost")
		}

		_, err := validatePathWithPermission(ctx, outsideFile, workspace, true, store, errFn)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "connection lost") {
			t.Errorf("expected 'connection lost' in error, got: %v", err)
		}
	})
}

func TestValidatePath_WithPermission_CachedApproval(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	if err := os.MkdirAll(workspace, 0755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	outside := filepath.Join(root, "outside")
	if err := os.MkdirAll(outside, 0755); err != nil {
		t.Fatalf("failed to create outside dir: %v", err)
	}
	outsideFile := filepath.Join(outside, "data.txt")
	if err := os.WriteFile(outsideFile, []byte("data"), 0644); err != nil {
		t.Fatalf("failed to write data file: %v", err)
	}

	ctx := context.Background()
	store := NewPermissionStore()
	callCount := 0
	permFn := func(_ context.Context, _ string) (bool, error) {
		callCount++
		return true, nil
	}

	// First call — should invoke permFn
	_, err := validatePathWithPermission(ctx, outsideFile, workspace, true, store, permFn)
	if err != nil {
		t.Fatalf("first call: unexpected error: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected permFn called once, got %d", callCount)
	}

	// Second call — should use cached approval, not invoke permFn again
	_, err = validatePathWithPermission(ctx, outsideFile, workspace, true, store, permFn)
	if err != nil {
		t.Fatalf("second call: unexpected error: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected permFn still called once (cached), got %d", callCount)
	}

	// Different file in same directory — should also use cache
	otherFile := filepath.Join(outside, "other.txt")
	_, err = validatePathWithPermission(ctx, otherFile, workspace, true, store, permFn)
	if err != nil {
		t.Fatalf("third call: unexpected error: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected permFn still called once (same dir cached), got %d", callCount)
	}
}
