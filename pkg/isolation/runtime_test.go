package isolation

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/sipeed/picoclaw/pkg"
	"github.com/sipeed/picoclaw/pkg/config"
)

func TestResolveInstanceRoot_UsesPicoclawHome(t *testing.T) {
	t.Setenv(config.EnvHome, "/custom/picoclaw/home")
	root, err := ResolveInstanceRoot()
	if err != nil {
		t.Fatalf("ResolveInstanceRoot() error = %v", err)
	}
	if root != "/custom/picoclaw/home" {
		t.Fatalf("ResolveInstanceRoot() = %q, want %q", root, "/custom/picoclaw/home")
	}
}

func TestPrepareInstanceRoot_CreatesDirectories(t *testing.T) {
	root := filepath.Join(t.TempDir(), "instance")
	if err := PrepareInstanceRoot(root); err != nil {
		t.Fatalf("PrepareInstanceRoot() error = %v", err)
	}
	for _, dir := range InstanceDirs(root) {
		if info, err := os.Stat(dir); err != nil {
			t.Fatalf("os.Stat(%q): %v", dir, err)
		} else if !info.IsDir() {
			t.Fatalf("%q is not a directory", dir)
		}
	}
}

func TestInstanceDirs_UsesInstanceWorkspaceNotGlobalState(t *testing.T) {
	root := filepath.Join(t.TempDir(), "instance")
	cfg := config.DefaultConfig()
	cfg.Isolation.Enabled = true
	cfg.Agents.Defaults.Workspace = filepath.Join(t.TempDir(), "external-workspace")
	Configure(cfg)
	t.Cleanup(func() { Configure(config.DefaultConfig()) })

	dirs := InstanceDirs(root)
	wantWorkspace := filepath.Join(root, pkg.WorkspaceName)
	found := false
	for _, dir := range dirs {
		if dir == wantWorkspace {
			found = true
		}
		if dir == cfg.WorkspacePath() {
			t.Fatalf("InstanceDirs() should not depend on process-wide workspace state: %q", dir)
		}
	}
	if !found {
		t.Fatalf("InstanceDirs() missing instance workspace dir %q", wantWorkspace)
	}
}

func TestIsSupportedOn(t *testing.T) {
	tests := []struct {
		goos string
		want bool
	}{
		{goos: "linux", want: true},
		{goos: "windows", want: true},
		{goos: "darwin", want: false},
		{goos: "freebsd", want: false},
	}
	for _, tt := range tests {
		if got := isSupportedOn(tt.goos); got != tt.want {
			t.Fatalf("isSupportedOn(%q) = %v, want %v", tt.goos, got, tt.want)
		}
	}
}

func TestValidateExposePaths(t *testing.T) {
	err := ValidateExposePaths([]config.ExposePath{{Source: "/src", Target: "/dst", Mode: "ro"}})
	if err != nil {
		t.Fatalf("ValidateExposePaths() error = %v", err)
	}

	err = ValidateExposePaths([]config.ExposePath{{Source: "/src", Target: "/dst", Mode: "bad"}})
	if err == nil {
		t.Fatal("ValidateExposePaths() expected invalid mode error")
	}

	err = ValidateExposePaths(
		[]config.ExposePath{
			{Source: "/src", Target: "/dst", Mode: "ro"},
			{Source: "/other", Target: "/dst", Mode: "rw"},
		},
	)
	if err == nil {
		t.Fatal("ValidateExposePaths() expected duplicate target error")
	}
}

func TestMergeExposePaths_OverrideByTarget(t *testing.T) {
	merged := MergeExposePaths(
		[]config.ExposePath{{Source: "/src-a", Target: "/dst", Mode: "ro"}},
		[]config.ExposePath{{Source: "/src-b", Target: "/dst", Mode: "rw"}},
	)
	if len(merged) != 1 {
		t.Fatalf("MergeExposePaths len = %d, want 1", len(merged))
	}
	if got := merged[0]; got.Source != "/src-b" || got.Target != "/dst" || got.Mode != "rw" {
		t.Fatalf("merged[0] = %+v, want source=/src-b target=/dst mode=rw", got)
	}
}

func TestBuildLinuxMountPlan(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux-only default mount set")
	}
	plan := BuildLinuxMountPlan("/rootdir", []config.ExposePath{{Source: "/src", Target: "/dst", Mode: "ro"}})
	if len(plan) == 0 {
		t.Fatal("BuildLinuxMountPlan returned empty plan")
	}
	foundRoot := false
	foundOverride := false
	for _, rule := range plan {
		if rule.Source == "/rootdir" && rule.Target == "/rootdir" && rule.Mode == "rw" {
			foundRoot = true
		}
		if rule.Source == "/src" && rule.Target == "/dst" && rule.Mode == "ro" {
			foundOverride = true
		}
	}
	if !foundRoot {
		t.Fatal("BuildLinuxMountPlan missing root mapping")
	}
	if !foundOverride {
		t.Fatal("BuildLinuxMountPlan missing override mapping")
	}
}

func TestBuildWindowsAccessRules(t *testing.T) {
	rules := BuildWindowsAccessRules(
		`C:\picoclaw`,
		[]config.ExposePath{{Source: `D:\data`, Target: `C:\mapped`, Mode: "ro"}},
	)
	if len(rules) == 0 {
		t.Fatal("BuildWindowsAccessRules returned empty rules")
	}
	foundRoot := false
	foundOverride := false
	for _, rule := range rules {
		if rule.Path == `C:\picoclaw` && rule.Mode == "rw" {
			foundRoot = true
		}
		if rule.Path == `D:\data` && rule.Mode == "ro" {
			foundOverride = true
		}
	}
	if !foundRoot {
		t.Fatal("BuildWindowsAccessRules missing root rule")
	}
	if !foundOverride {
		t.Fatal("BuildWindowsAccessRules missing override rule")
	}
}

func TestValidateWindowsExposePaths(t *testing.T) {
	if err := validateWindowsExposePaths(nil); err != nil {
		t.Fatalf("validateWindowsExposePaths(nil) error = %v", err)
	}
	err := validateWindowsExposePaths([]config.ExposePath{{Source: `D:\data`, Target: `D:\data`, Mode: "ro"}})
	if err == nil {
		t.Fatal("validateWindowsExposePaths() expected error for expose_paths")
	}
}

func TestDefaultLinuxSystemExposePaths(t *testing.T) {
	paths := defaultLinuxSystemExposePaths()
	needed := map[string]bool{}
	for _, path := range []string{"/etc/hosts", "/etc/nsswitch.conf", "/etc/ssl", "/usr/share/zoneinfo", "/etc/localtime"} {
		if _, err := os.Stat(path); err == nil {
			needed[path] = false
		}
	}
	for _, item := range paths {
		if _, ok := needed[item.Source]; ok {
			needed[item.Source] = true
		}
	}
	for path, found := range needed {
		if !found {
			t.Fatalf("defaultLinuxSystemExposePaths missing %s", path)
		}
	}
}

func TestExistingExposePaths_SkipsMissingPaths(t *testing.T) {
	existing := filepath.Join(t.TempDir(), "existing")
	if err := os.MkdirAll(existing, 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	filtered := existingExposePaths([]config.ExposePath{
		{Source: existing, Target: existing, Mode: "ro"},
		{Source: filepath.Join(t.TempDir(), "missing"), Target: "/missing", Mode: "ro"},
	})
	if len(filtered) != 1 {
		t.Fatalf("existingExposePaths() len = %d, want 1", len(filtered))
	}
	if got := filtered[0]; got.Source != existing {
		t.Fatalf("existingExposePaths()[0] = %+v, want source=%q", got, existing)
	}
}

func TestPrepareCommand_AppliesUserEnv(t *testing.T) {
	if !isSupportedOn(runtime.GOOS) {
		t.Skipf("isolation not supported on %s", runtime.GOOS)
	}
	t.Setenv(config.EnvHome, filepath.Join(t.TempDir(), "home"))
	if runtime.GOOS == "linux" {
		binDir := filepath.Join(t.TempDir(), "bin")
		if err := os.MkdirAll(binDir, 0o755); err != nil {
			t.Fatalf("os.MkdirAll() error = %v", err)
		}
		fakeBwrap := filepath.Join(binDir, "bwrap")
		if err := os.WriteFile(fakeBwrap, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatalf("os.WriteFile() error = %v", err)
		}
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	}
	cfg := config.DefaultConfig()
	cfg.Isolation.Enabled = true
	Configure(cfg)
	t.Cleanup(func() { Configure(config.DefaultConfig()) })
	cmd := exec.Command("sh", "-c", "true")
	if err := PrepareCommand(cmd); err != nil {
		t.Fatalf("PrepareCommand() error = %v", err)
	}
	hasHome := false
	for _, env := range cmd.Env {
		if len(env) > 5 && env[:5] == "HOME=" {
			hasHome = true
			break
		}
	}
	if runtime.GOOS != "windows" && !hasHome {
		t.Fatal("PrepareCommand() did not inject HOME")
	}
}
