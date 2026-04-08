package isolation

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

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
	t.Setenv("USERPROFILE", `C:\Users\tester`)
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

func TestPrepareCommand_AppliesUserEnv(t *testing.T) {
	t.Setenv(config.EnvHome, filepath.Join(t.TempDir(), "home"))
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
