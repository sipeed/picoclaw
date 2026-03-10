package daemon

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestBuildExecStart_DefaultIncludesNoBrowser(t *testing.T) {
	execPath := "/usr/local/bin/picoclaw-launcher"
	configPath := "/home/user/.picoclaw/config.json"

	line := buildExecStart(execPath, configPath, false)

	if !strings.Contains(line, "-no-browser") {
		t.Fatalf("expected -no-browser in ExecStart, got %q", line)
	}
	if strings.Contains(line, " -public") {
		t.Fatalf("did not expect -public in ExecStart when public=false, got %q", line)
	}
	if !strings.Contains(line, configPath) {
		t.Fatalf("expected config path in ExecStart, got %q", line)
	}
}

func TestBuildExecStart_PublicIncludesPublicFlag(t *testing.T) {
	execPath := "/usr/local/bin/picoclaw-launcher"
	configPath := "/home/user/.picoclaw/config.json"

	line := buildExecStart(execPath, configPath, true)

	if !strings.Contains(line, " -public") {
		t.Fatalf("expected -public in ExecStart, got %q", line)
	}
}

func TestBuildUnitContent(t *testing.T) {
	unit := buildUnitContent("/usr/local/bin/picoclaw-launcher", "/tmp/config.json", true)

	checks := []string{
		"[Unit]",
		"[Service]",
		"[Install]",
		"Restart=on-failure",
		"ExecStart=",
		"-no-browser",
		"-public",
	}
	for _, check := range checks {
		if !strings.Contains(unit, check) {
			t.Fatalf("expected unit content to include %q, got:\n%s", check, unit)
		}
	}
}

func TestUserUnitPath(t *testing.T) {
	home := "/home/example"
	if runtime.GOOS == "windows" {
		home = `C:\\Users\\example`
	}

	got := userUnitPath(home)
	want := filepath.Join(home, ".config", "systemd", "user", serviceFileName)
	if got != want {
		t.Fatalf("unexpected unit path, want %q, got %q", want, got)
	}
}
