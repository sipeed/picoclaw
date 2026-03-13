package internal

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestFormatVersion_NoGitCommit(t *testing.T) {
	oldVersion, oldGit := config.Version, config.GitCommit
	t.Cleanup(func() { config.Version, config.GitCommit = oldVersion, oldGit })

	config.Version = "1.2.3"
	config.GitCommit = ""

	assert.Equal(t, "1.2.3", FormatVersion())
}

func TestFormatVersion_WithGitCommit(t *testing.T) {
	oldVersion, oldGit := config.Version, config.GitCommit
	t.Cleanup(func() { config.Version, config.GitCommit = oldVersion, oldGit })

	config.Version = "1.2.3"
	config.GitCommit = "abc123"

	assert.Equal(t, "1.2.3 (git: abc123)", FormatVersion())
}

func TestFormatBuildInfo_UsesBuildTimeAndGoVersion_WhenSet(t *testing.T) {
	oldBuildTime, oldGoVersion := config.BuildTime, config.GoVersion
	t.Cleanup(func() { config.BuildTime, config.GoVersion = oldBuildTime, oldGoVersion })

	config.BuildTime = "2026-02-20T00:00:00Z"
	config.GoVersion = "go1.23.0"

	build, goVer := FormatBuildInfo()

	assert.Equal(t, config.BuildTime, build)
	assert.Equal(t, config.GoVersion, goVer)
}

func TestFormatBuildInfo_EmptyBuildTime_ReturnsEmptyBuild(t *testing.T) {
	oldBuildTime, oldGoVersion := config.BuildTime, config.GoVersion
	t.Cleanup(func() { config.BuildTime, config.GoVersion = oldBuildTime, oldGoVersion })

	config.BuildTime = ""
	config.GoVersion = "go1.23.0"

	build, goVer := FormatBuildInfo()

	assert.Empty(t, build)
	assert.Equal(t, config.GoVersion, goVer)
}

func TestFormatBuildInfo_EmptyGoVersion_FallsBackToRuntimeVersion(t *testing.T) {
	oldBuildTime, oldGoVersion := config.BuildTime, config.GoVersion
	t.Cleanup(func() { config.BuildTime, config.GoVersion = oldBuildTime, oldGoVersion })

	config.BuildTime = "x"
	config.GoVersion = ""

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
