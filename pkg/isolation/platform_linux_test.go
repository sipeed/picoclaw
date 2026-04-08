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
	args, err := buildLinuxBwrapArgs(binaryPath, []string{binaryPath, "--flag"}, root, plan)
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
