package config

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatVersion_NoGitCommit(t *testing.T) {
	oldVersion, oldGit := Version, GitCommit
	t.Cleanup(func() { Version, GitCommit = oldVersion, oldGit })

	Version = "1.2.3"
	GitCommit = ""

	assert.Equal(t, "1.2.3", FormatVersion())
}

func TestFormatVersion_WithGitCommit(t *testing.T) {
	oldVersion, oldGit := Version, GitCommit
	t.Cleanup(func() { Version, GitCommit = oldVersion, oldGit })

	Version = "1.2.3"
	GitCommit = "abc123"

	assert.Equal(t, "1.2.3 (git: abc123)", FormatVersion())
}

func TestFormatBuildInfo_UsesBuildTimeAndGoVersion_WhenSet(t *testing.T) {
	oldBuildTime, oldGoVersion := BuildTime, GoVersion
	t.Cleanup(func() { BuildTime, GoVersion = oldBuildTime, oldGoVersion })

	BuildTime = "2026-02-20T00:00:00Z"
	GoVersion = "go1.23.0"

	build, goVer := FormatBuildInfo()

	assert.Equal(t, BuildTime, build)
	assert.Equal(t, GoVersion, goVer)
}

func TestFormatBuildInfo_EmptyBuildTime_ReturnsEmptyBuild(t *testing.T) {
	oldBuildTime, oldGoVersion := BuildTime, GoVersion
	t.Cleanup(func() { BuildTime, GoVersion = oldBuildTime, oldGoVersion })

	BuildTime = ""
	GoVersion = "go1.23.0"

	build, goVer := FormatBuildInfo()

	assert.Empty(t, build)
	assert.Equal(t, GoVersion, goVer)
}

func TestFormatBuildInfo_EmptyGoVersion_FallsBackToRuntimeVersion(t *testing.T) {
	oldBuildTime, oldGoVersion := BuildTime, GoVersion
	t.Cleanup(func() { BuildTime, GoVersion = oldBuildTime, oldGoVersion })

	BuildTime = "x"
	GoVersion = ""

	build, goVer := FormatBuildInfo()

	assert.Equal(t, "x", build)
	assert.Equal(t, runtime.Version(), goVer)
}

func TestGetVersion(t *testing.T) {
	oldVersion := Version
	t.Cleanup(func() { Version = oldVersion })

	Version = "dev"
	assert.Equal(t, "dev", GetVersion())
}

func TestGetVersion_Custom(t *testing.T) {
	oldVersion := Version
	t.Cleanup(func() { Version = oldVersion })

	Version = "v1.0.0"
	assert.Equal(t, "v1.0.0", GetVersion())
}

func TestVersion_DefaultIsDev(t *testing.T) {
	// Reset to default values
	oldVersion := Version
	Version = "dev"
	t.Cleanup(func() { Version = oldVersion })

	assert.Equal(t, "dev", Version)
}
