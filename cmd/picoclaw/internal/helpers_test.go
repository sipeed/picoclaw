package internal

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGetConfigPath(t *testing.T) {
	t.Setenv("HOME", "/tmp/home")

	got := GetConfigPath()
	want := filepath.Join("/tmp/home", ".picoclaw", "config.json")

	if got != want {
		t.Fatalf("GetConfigPath() = %q, want %q", got, want)
	}
}

func TestFormatVersion_NoGitCommit(t *testing.T) {
	oldVersion, oldGit := version, gitCommit

	t.Cleanup(func() {
		version, gitCommit = oldVersion, oldGit
	})

	version = "1.2.3"
	gitCommit = ""

	got := FormatVersion()
	want := "1.2.3"

	if got != want {
		t.Fatalf("FormatVersion() = %q, want %q", got, want)
	}
}

func TestFormatVersion_WithGitCommit(t *testing.T) {
	oldVersion, oldGit := version, gitCommit

	t.Cleanup(func() {
		version, gitCommit = oldVersion, oldGit
	})

	version = "1.2.3"
	gitCommit = "abc123"

	got := FormatVersion()
	want := "1.2.3 (git: abc123)"

	if got != want {
		t.Fatalf("FormatVersion() = %q, want %q", got, want)
	}
}

func TestFormatBuildInfo_UsesBuildTimeAndGoVersion_WhenSet(t *testing.T) {
	oldBuildTime, oldGoVersion := buildTime, goVersion

	t.Cleanup(func() {
		buildTime, goVersion = oldBuildTime, oldGoVersion
	})

	buildTime = "2026-02-20T00:00:00Z"
	goVersion = "go1.23.0"

	build, goVer := FormatBuildInfo()

	if build != buildTime {
		t.Fatalf("FormatBuildInfo().build = %q, want %q", build, buildTime)
	}

	if goVer != goVersion {
		t.Fatalf("FormatBuildInfo().goVer = %q, want %q", goVer, goVersion)
	}
}

func TestFormatBuildInfo_EmptyBuildTime_ReturnsEmptyBuild(t *testing.T) {
	oldBuildTime, oldGoVersion := buildTime, goVersion

	t.Cleanup(func() {
		buildTime, goVersion = oldBuildTime, oldGoVersion
	})

	buildTime = ""
	goVersion = "go1.23.0"

	build, goVer := FormatBuildInfo()

	if build != "" {
		t.Fatalf("FormatBuildInfo().build = %q, want empty", build)
	}

	if goVer != goVersion {
		t.Fatalf("FormatBuildInfo().goVer = %q, want %q", goVer, goVersion)
	}
}

func TestFormatBuildInfo_EmptyGoVersion_FallsBackToRuntimeVersion(t *testing.T) {
	oldBuildTime, oldGoVersion := buildTime, goVersion

	t.Cleanup(func() {
		buildTime, goVersion = oldBuildTime, oldGoVersion
	})

	buildTime = "x"
	goVersion = ""

	build, goVer := FormatBuildInfo()
	if build != "x" {
		t.Fatalf("FormatBuildInfo().build = %q, want %q", build, "x")
	}

	if goVer != runtime.Version() {
		t.Fatalf("FormatBuildInfo().goVer = %q, want runtime.Version()=%q", goVer, runtime.Version())
	}
}

func TestGetConfigPath_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-specific HOME behavior varies; run on windows")
	}

	t.Setenv("USERPROFILE", `C:\Users\Test`)

	got := GetConfigPath()
	want := filepath.Join(`C:\Users\Test`, ".picoclaw", "config.json")

	if !strings.EqualFold(got, want) {
		t.Fatalf("GetConfigPath() = %q, want %q", got, want)
	}
}
