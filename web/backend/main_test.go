package main

import (
	"testing"

	"github.com/sipeed/picoclaw/web/backend/launcherconfig"
)

func TestShouldEnableLauncherFileLogging(t *testing.T) {
	tests := []struct {
		name          string
		enableConsole bool
		debug         bool
		want          bool
	}{
		{name: "gui mode", enableConsole: false, debug: false, want: true},
		{name: "console mode", enableConsole: true, debug: false, want: false},
		{name: "debug gui mode", enableConsole: false, debug: true, want: true},
		{name: "debug console mode", enableConsole: true, debug: true, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldEnableLauncherFileLogging(tt.enableConsole, tt.debug); got != tt.want {
				t.Fatalf(
					"shouldEnableLauncherFileLogging(%t, %t) = %t, want %t",
					tt.enableConsole,
					tt.debug,
					got,
					tt.want,
				)
			}
		})
	}
}

func TestDashboardTokenConfigHelpPath(t *testing.T) {
	const launcherPath = "/tmp/launcher-config.json"

	tests := []struct {
		name   string
		source launcherconfig.DashboardTokenSource
		want   string
	}{
		{
			name:   "env token does not expose config path",
			source: launcherconfig.DashboardTokenSourceEnv,
			want:   "",
		},
		{
			name:   "config token exposes config path",
			source: launcherconfig.DashboardTokenSourceConfig,
			want:   launcherPath,
		},
		{
			name:   "random token does not expose config path",
			source: launcherconfig.DashboardTokenSourceRandom,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dashboardTokenConfigHelpPath(tt.source, launcherPath); got != tt.want {
				t.Fatalf("dashboardTokenConfigHelpPath(%q, %q) = %q, want %q", tt.source, launcherPath, got, tt.want)
			}
		})
	}
}
