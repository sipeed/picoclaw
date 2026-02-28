package sandbox

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestHostSandbox_StartStopFs(t *testing.T) {
	root := t.TempDir()
	sb := NewHostSandbox(root, true)
	if err := sb.Start(context.Background()); err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	if got := sb.GetWorkspace(context.Background()); got != root {
		t.Errorf("GetWorkspace() = %q, want %q", got, root)
	}

	if err := sb.Prune(context.Background()); err != nil {
		t.Fatalf("Prune() error: %v", err)
	}
	if sb.Fs() == nil {
		t.Fatal("Fs() returned nil")
	}
}

func TestHostSandbox_ExecAndFs(t *testing.T) {
	root := t.TempDir()
	sb := NewHostSandbox(root, true)

	if _, err := sb.Exec(context.Background(), ExecRequest{Command: "   "}); err == nil {
		t.Fatal("expected empty command error")
	}

	res, err := sb.Exec(context.Background(), ExecRequest{
		Command: "sh",
		Args:    []string{"-c", "printf hello"},
	})
	if err != nil {
		t.Fatalf("Exec() error: %v", err)
	}
	if res.ExitCode != 0 || res.Stdout != "hello" {
		t.Fatalf("unexpected exec result: %#v", res)
	}

	if runtime.GOOS != "windows" {
		_, err = sb.Exec(context.Background(), ExecRequest{
			Command:   "sh",
			Args:      []string{"-c", "sleep 1"},
			TimeoutMs: 10,
		})
		if err == nil {
			t.Fatal("expected timeout-related error")
		}
	}

	_, err = sb.Exec(context.Background(), ExecRequest{
		Command:    "sh",
		Args:       []string{"-c", "echo bad"},
		WorkingDir: "../outside",
	})
	if err == nil || !errors.Is(err, ErrOutsideWorkspace) {
		t.Fatalf("expected working dir restriction error, got: %v", err)
	}

	err = sb.Fs().WriteFile(context.Background(), "dir/a.txt", []byte("x"), true)
	if err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}
	b, err := sb.Fs().ReadFile(context.Background(), "dir/a.txt")
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	if string(b) != "x" {
		t.Fatalf("ReadFile() got %q, want x", string(b))
	}
}

func TestHostSandbox_ResolvePathRestrictions(t *testing.T) {
	root := t.TempDir()

	got, err := ValidatePath("a/b.txt", root, true)
	if err != nil {
		t.Fatalf("resolvePath relative error: %v", err)
	}
	want := filepath.Join(root, "a", "b.txt")
	if got != want {
		t.Fatalf("resolvePath relative got %q, want %q", got, want)
	}

	_, err = ValidatePath(filepath.Join(root, "..", "outside.txt"), root, true)
	if err == nil || !errors.Is(err, ErrOutsideWorkspace) {
		t.Fatalf("expected outside workspace error, got: %v", err)
	}

	target := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatalf("write target file: %v", err)
	}
	link := filepath.Join(root, "link.txt")
	if err := os.Symlink(target, link); err == nil {
		_, err = ValidatePath("link.txt", root, true)
		if err == nil || !errors.Is(err, ErrOutsideWorkspace) {
			t.Fatalf("expected symlink outside error, got: %v", err)
		}
	}
}

func TestUnavailableSandboxAndUtilHelpers(t *testing.T) {
	sb := NewUnavailableSandboxManager(nil)
	if err := sb.Start(context.Background()); err == nil {
		t.Fatal("expected Start() error")
	}
	if err := sb.Prune(context.Background()); err != nil {
		t.Fatalf("Prune() error: %v", err)
	}
	if _, err := sb.Exec(context.Background(), ExecRequest{Command: "echo hi"}); err == nil {
		t.Fatal("expected Exec() error")
	}
	if _, err := sb.Fs().ReadFile(context.Background(), "a.txt"); err == nil {
		t.Fatal("expected Fs().ReadFile error")
	}
	if err := sb.Fs().WriteFile(context.Background(), "a.txt", []byte("x"), true); err == nil {
		t.Fatal("expected Fs().WriteFile error")
	}
}

func TestHostFS_ReadFileWriteFile_Restricted(t *testing.T) {
	root := t.TempDir()
	sb := NewHostSandbox(root, true)
	if err := sb.Start(context.Background()); err != nil {
		t.Fatal(err)
	}

	content := []byte("hello restrict")
	if err := sb.Fs().WriteFile(context.Background(), "a/b/c.txt", content, true); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	readContent, err := sb.Fs().ReadFile(context.Background(), "a/b/c.txt")
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(readContent) != string(content) {
		t.Fatalf("content mismatch")
	}

	// Should not be able to write root path
	if err := sb.Fs().WriteFile(context.Background(), "/etc/passwd_not_exist", []byte("a"), false); err == nil {
		t.Fatalf("expected access denied error writing outside workspace")
	}

	// Should not be able to read root path
	if _, err := sb.Fs().ReadFile(context.Background(), "/etc/passwd"); err == nil {
		t.Fatalf("expected access denied error reading outside workspace")
	}

	if err := sb.Prune(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestHostFS_ReadDir(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "a/b"), 0o755)
	os.WriteFile(filepath.Join(root, "a/f1.txt"), []byte("1"), 0o644)
	os.WriteFile(filepath.Join(root, "a/b/f2.txt"), []byte("2"), 0o644)

	sb := NewHostSandbox(root, true)
	if err := sb.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer sb.Prune(context.Background())

	// Test restricted ReadDir on "a"
	entries, err := sb.Fs().ReadDir(context.Background(), "a")
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}

	foundF1 := false
	foundB := false
	for _, e := range entries {
		if e.Name() == "f1.txt" && !e.IsDir() {
			foundF1 = true
		}
		if e.Name() == "b" && e.IsDir() {
			foundB = true
		}
	}
	if !foundF1 || !foundB {
		t.Errorf("ReadDir result missing expected entries: foundF1=%v, foundB=%v", foundF1, foundB)
	}

	// Test unrestricted ReadDir
	sb2 := NewHostSandbox(root, false)
	entries2, err := sb2.Fs().ReadDir(context.Background(), root)
	if err != nil {
		t.Fatalf("ReadDir unrestricted failed: %v", err)
	}
	foundA := false
	for _, e := range entries2 {
		if e.Name() == "a" {
			foundA = true
			break
		}
	}
	if !foundA {
		t.Errorf("ReadDir unrestricted missing 'a'")
	}
}

func TestHostFS_ReadFileWriteFile_Unrestricted(t *testing.T) {
	root := t.TempDir()
	sb := NewHostSandbox(root, false)

	// Write file directly into workspace since there's no restrictions
	content := []byte("hello unrestricted")
	if err := sb.Fs().WriteFile(context.Background(), "a/b/c_unrestricted.txt", content, true); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	readContent, err := sb.Fs().ReadFile(context.Background(), "a/b/c_unrestricted.txt")
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(readContent) != string(content) {
		t.Fatalf("content mismatch")
	}
}

func TestHostFS_WriteFileMKdir(t *testing.T) {
	root := t.TempDir()
	sb := NewHostSandbox(root, true)
	sb.Start(context.Background())
	defer sb.Prune(context.Background())

	err := sb.Fs().WriteFile(context.Background(), "deep/nested/dir/file.txt", []byte("a"), true)
	if err != nil {
		t.Fatalf("WriteFile with mkdir failed: %v", err)
	}
}

func TestHostFS_WriteFileMKdirFailure(t *testing.T) {
	root := t.TempDir()
	sb := NewHostSandbox(root, true)
	sb.Start(context.Background())
	defer sb.Prune(context.Background())

	// write to a path where parent is a file instead of dir
	err := sb.Fs().WriteFile(context.Background(), "a.txt", []byte("a"), false)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	err = sb.Fs().WriteFile(context.Background(), "a.txt/b.txt", []byte("b"), true)
	if err == nil {
		t.Fatalf("Expected MkdirAll to fail because a.txt is a file")
	}
}

func TestHostFS_ReadFileFailure(t *testing.T) {
	root := t.TempDir()
	sb := NewHostSandbox(root, true)
	sb.Start(context.Background())
	defer sb.Prune(context.Background())

	_, err := sb.Fs().ReadFile(context.Background(), "does_not_exist.txt")
	if err == nil {
		t.Fatalf("Expected ReadFile to fail for non-existent file")
	}
}

func TestHostFS_WriteFileFailure(t *testing.T) {
	root := t.TempDir()
	sb := NewHostSandbox(root, true)
	sb.Start(context.Background())
	defer sb.Prune(context.Background())

	// Create a read-only directory
	roDir := filepath.Join(root, "ro")
	os.Mkdir(roDir, 0o500)

	err := sb.Fs().WriteFile(context.Background(), "ro/failed.txt", []byte("a"), false)
	if err == nil {
		t.Fatalf("Expected WriteFile to fail in read-only dir")
	}
}

func TestHostSandbox_PruneWhenNilFs(t *testing.T) {
	// simulate prune condition for code coverage
	sb := NewHostSandbox(t.TempDir(), true)
	sb.fs.(*hostFS).root = nil
	err := sb.Prune(context.Background())
	if err != nil {
		t.Fatalf("Prune failed when fs root is nil (%v)", err)
	}
}

func TestHostSandbox_StartBadWorkspace(t *testing.T) {
	sb := NewHostSandbox("/this_should_not_exist_normally_12345/abc", true)
	err := sb.Start(context.Background())
	if err == nil {
		t.Fatalf("Start should fail for non-existing workspace root")
	}
}

func TestHostFS_ReadFileWriteFile_WithoutWorkspaceOrRoot(t *testing.T) {
	root := t.TempDir()

	// Test blank workspace
	sb := NewHostSandbox("", true)
	if err := sb.Start(context.Background()); err != nil {
		t.Fatal(err)
	}

	content := []byte("hello empty workspace")
	target := filepath.Join(root, "empty.txt")
	if err := sb.Fs().WriteFile(context.Background(), target, content, true); err == nil {
		t.Fatalf("expected WriteFile to fail due to empty workspace with restrict=true")
	}

	_, err := sb.Fs().ReadFile(context.Background(), target)
	if err == nil {
		t.Fatalf("expected ReadFile to fail due to empty workspace with restrict=true")
	}

	// Test case where root is nil explicitly
	sb2 := NewHostSandbox(root, true)
	sb2.fs.(*hostFS).root = nil
	err = sb2.Fs().WriteFile(context.Background(), "nil_root_test.txt", content, true)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	readContent, err := sb2.Fs().ReadFile(context.Background(), "nil_root_test.txt")
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(readContent) != string(content) {
		t.Fatalf("content mismatch")
	}
}

func TestValidatePathErrors(t *testing.T) {
	_, err := ValidatePath("/a/b/c", "", true)
	if err == nil {
		t.Fatalf("expected err for empty workspace with restrict=true")
	}

	root := t.TempDir()

	// target parent is file, evalSymlinks should fail
	err = os.WriteFile(filepath.Join(root, "a.txt"), []byte("a"), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ValidatePath("a.txt/b.txt", root, true)
	if err == nil {
		t.Fatalf("expected error when ancestor is file")
	}
}
