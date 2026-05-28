//go:build linux

package isolation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestBuildLinuxBwrapArgs_IncludesNamespaceFlagsAndExec(t *testing.T) {
	root := t.TempDir()
	binaryDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binaryDir, 0o755); err != nil {
		t.Fatal(err)
	}
	binaryPath := filepath.Join(binaryDir, "tool")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	plan := BuildLinuxMountPlan(root, []config.ExposePath{{Source: binaryDir, Target: binaryDir, Mode: "ro"}})
	args, err := buildLinuxBwrapArgs(binaryPath, binaryPath, []string{binaryPath, "--flag"}, root, plan)
	if err != nil {
		t.Fatalf("buildLinuxBwrapArgs() error = %v", err)
	}
	hasNet := false
	hasIPC := false
	hasExec := false
	for i := range args {
		switch args[i] {
		case "--unshare-net":
			hasNet = true
		case "--unshare-ipc":
			hasIPC = true
		case "--":
			if i+1 < len(args) && args[i+1] == binaryPath {
				hasExec = true
			}
		}
	}
	if hasNet {
		t.Fatalf("bwrap args should not unshare net by default: %v", args)
	}
	if !hasIPC || !hasExec {
		t.Fatalf("bwrap args missing required items: %v", args)
	}
}

func TestResolveLinuxWorkingDir_ResolvesRelativeDir(t *testing.T) {
	cwd := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if chdirErr := os.Chdir(previous); chdirErr != nil {
			t.Fatalf("restore cwd: %v", chdirErr)
		}
	}()
	if chdirErr := os.Chdir(cwd); chdirErr != nil {
		t.Fatal(chdirErr)
	}

	resolvedDir, execDir, err := resolveLinuxWorkingDir("./hooks", "./hook.sh")
	if err != nil {
		t.Fatalf("resolveLinuxWorkingDir() error = %v", err)
	}
	want := filepath.Join(cwd, "hooks")
	if resolvedDir != want || execDir != want {
		t.Fatalf("resolveLinuxWorkingDir() = (%q, %q), want (%q, %q)", resolvedDir, execDir, want, want)
	}
}

func TestResolveLinuxCommandPath_UsesExecDirForRelativeCommand(t *testing.T) {
	execDir := filepath.Join(t.TempDir(), "hooks")
	got, err := resolveLinuxCommandPath("./hook.sh", execDir)
	if err != nil {
		t.Fatalf("resolveLinuxCommandPath() error = %v", err)
	}
	want := filepath.Join(execDir, "hook.sh")
	if got != want {
		t.Fatalf("resolveLinuxCommandPath() = %q, want %q", got, want)
	}
}

func TestBuildLinuxBwrapArgs_UsesResolvedPathForRelativeCommand(t *testing.T) {
	root := t.TempDir()
	execDir := filepath.Join(root, "hooks")
	if err := os.MkdirAll(execDir, 0o755); err != nil {
		t.Fatal(err)
	}
	resolvedPath := filepath.Join(execDir, "hook.sh")
	if err := os.WriteFile(resolvedPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	plan := []MountRule{
		{Source: execDir, Target: execDir, Mode: "rw"},
		{Source: resolvedPath, Target: resolvedPath, Mode: "ro"},
	}
	args, err := buildLinuxBwrapArgs("./hook.sh", resolvedPath, []string{"./hook.sh"}, execDir, plan)
	if err != nil {
		t.Fatalf("buildLinuxBwrapArgs() error = %v", err)
	}
	hasExecDir := false
	for _, arg := range args {
		if arg == execDir {
			hasExecDir = true
			break
		}
	}
	if !hasExecDir {
		t.Fatalf("buildLinuxBwrapArgs() missing resolved chdir: %v", args)
	}
	for i := range args {
		if args[i] == "--" {
			if i+1 >= len(args) || args[i+1] != resolvedPath {
				t.Fatalf("buildLinuxBwrapArgs() exec path = %v, want %q after --", args, resolvedPath)
			}
			return
		}
	}
	t.Fatalf("buildLinuxBwrapArgs() missing exec delimiter: %v", args)
}

func TestAppendLinuxArgumentMounts_AddsAbsoluteArgumentPaths(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "input.txt")
	if err := os.WriteFile(input, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(root, "out", "result.txt")
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		t.Fatal(err)
	}

	plan := appendLinuxArgumentMounts(nil, []string{input, "--output=" + output})
	if len(plan) != 2 {
		t.Fatalf("appendLinuxArgumentMounts() len = %d, want 2", len(plan))
	}
	if plan[0].Source != input || plan[0].Mode != "ro" {
		t.Fatalf("appendLinuxArgumentMounts()[0] = %+v, want source=%q mode=ro", plan[0], input)
	}
	if plan[1].Source != filepath.Dir(output) || plan[1].Mode != "rw" {
		t.Fatalf("appendLinuxArgumentMounts()[1] = %+v, want source=%q mode=rw", plan[1], filepath.Dir(output))
	}
}
