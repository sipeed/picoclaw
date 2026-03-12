package internal

import (
	"github.com/stretchr/testify/assert"
	"runtime"
	"testing"
)

func TestFormatVersion_NoGitCommit(t *testing.T) {
	oldVersion, oldGit := version, gitCommit
	t.Cleanup(func() { version, gitCommit = oldVersion, oldGit })

	version = "1.2.3"
	gitCommit = ""

	assert.Equal(t, "1.2.3", FormatVersion())
}

func TestFormatVersion_WithGitCommit(t *testing.T) {
	oldVersion, oldGit := version, gitCommit
	t.Cleanup(func() { version, gitCommit = oldVersion, oldGit })

	version = "1.2.3"
	gitCommit = "abc123"

	assert.Equal(t, "1.2.3 (git: abc123)", FormatVersion())
}

func TestFormatBuildInfo_UsesBuildTimeAndGoVersion_WhenSet(t *testing.T) {
	oldBuildTime, oldGoVersion := buildTime, goVersion
	t.Cleanup(func() { buildTime, goVersion = oldBuildTime, oldGoVersion })

	buildTime = "2026-02-20T00:00:00Z"
	goVersion = "go1.23.0"

	build, goVer := FormatBuildInfo()

	assert.Equal(t, buildTime, build)
	assert.Equal(t, goVersion, goVer)
}

func TestFormatBuildInfo_EmptyBuildTime_ReturnsEmptyBuild(t *testing.T) {
	oldBuildTime, oldGoVersion := buildTime, goVersion
	t.Cleanup(func() { buildTime, goVersion = oldBuildTime, oldGoVersion })

	buildTime = ""
	goVersion = "go1.23.0"

	build, goVer := FormatBuildInfo()

	assert.Empty(t, build)
	assert.Equal(t, goVersion, goVer)
}

func TestFormatBuildInfo_EmptyGoVersion_FallsBackToRuntimeVersion(t *testing.T) {
	oldBuildTime, oldGoVersion := buildTime, goVersion
	t.Cleanup(func() { buildTime, goVersion = oldBuildTime, oldGoVersion })

	buildTime = "x"
	goVersion = ""

	build, goVer := FormatBuildInfo()

	assert.Equal(t, "x", build)
	assert.Equal(t, runtime.Version(), goVer)
}

func TestGetVersion(t *testing.T) {
	assert.Equal(t, "dev", GetVersion())
}

func TestGetConfigPath_WithEnv(t *testing.T) {
	t.Setenv("PICOCLAW_CONFIG", "/tmp/custom/config.json")
	t.Setenv("HOME", "/tmp/home")

	got := GetConfigPath()
	want := "/tmp/custom/config.json"

	assert.Equal(t, want, got)
}
