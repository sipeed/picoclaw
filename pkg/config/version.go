package config

import (
	"fmt"
	"runtime"
)

// Build-time variables injected via ldflags during build process.
// These are set by the Makefile or .goreleaser.yaml using the -X flag:
//
//	-X jane/pkg/config.Version=<version>
//	-X jane/pkg/config.GitCommit=<commit>
//	-X jane/pkg/config.BuildTime=<timestamp>
//	-X jane/pkg/config.GoVersion=<go-version>
var (
	Version   = "dev" // Default value when not built with ldflags
	GitCommit string  // Git commit SHA (short)
	BuildTime string  // Build timestamp in RFC3339 format
	GoVersion string  // Go version used for building
)

// FormatVersion returns the version string with optional git commit
func FormatVersion() string {
	v := Version
	if GitCommit != "" {
		v += fmt.Sprintf(" (git: %s)", GitCommit)
	}
	return v
}

// FormatBuildInfo returns build time and go version info
func FormatBuildInfo() (string, string) {
	build := BuildTime
	goVer := GoVersion
	if goVer == "" {
		goVer = runtime.Version()
	}
	return build, goVer
}

// GetVersion returns the version string
func GetVersion() string {
	return Version
}
