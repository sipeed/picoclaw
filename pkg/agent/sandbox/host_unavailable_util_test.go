package sandbox

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestHostSandbox_StartStopFs(t *testing.T) {
	sb := NewHostSandbox(t.TempDir(), true)
	if err := sb.Start(context.Background()); err != nil {
		t.Fatalf("Start() error: %v", err)
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
	if err == nil || !strings.Contains(err.Error(), "outside the workspace") {
		t.Fatalf("expected working dir restriction error, got: %v", err)
	}

	if err := sb.Fs().WriteFile(context.Background(), "dir/a.txt", []byte("x"), true); err != nil {
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
	sb := NewHostSandbox(root, true)

	got, err := sb.resolvePath("a/b.txt")
	if err != nil {
		t.Fatalf("resolvePath relative error: %v", err)
	}
	want := filepath.Join(root, "a", "b.txt")
	if got != want {
		t.Fatalf("resolvePath relative got %q, want %q", got, want)
	}

	_, err = sb.resolvePath(filepath.Join(root, "..", "outside.txt"))
	if err == nil || !strings.Contains(err.Error(), "outside the workspace") {
		t.Fatalf("expected outside workspace error, got: %v", err)
	}

	target := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatalf("write target file: %v", err)
	}
	link := filepath.Join(root, "link.txt")
	if err := os.Symlink(target, link); err == nil {
		_, err = sb.resolvePath("link.txt")
		if err == nil || !strings.Contains(err.Error(), "symlink resolves outside workspace") {
			t.Fatalf("expected symlink outside error, got: %v", err)
		}
	}
}

func TestUnavailableSandboxAndUtilHelpers(t *testing.T) {
	sb := NewUnavailableSandbox(nil)
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

	if got := durationMs(123).Milliseconds(); got != 123 {
		t.Fatalf("durationMs() got %d, want 123", got)
	}
	if asExitError(errors.New("x"), nil) {
		t.Fatal("asExitError should be false for non-exit errors")
	}
}
