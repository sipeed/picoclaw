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

func TestMaskSecret(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Long token (>=12 chars): first 3 + 10 stars + last 4
		{"sdhjflsjdflksdf", "sdh**********ksdf"},
		{"abcdefghijklmnopqrstuvwxyz", "abc**********wxyz"},
		// Exactly 12 chars (3+4+5 hidden): suffix shown
		{"abcdefghijkl", "abc**********ijkl"},
		// 8 chars (minimum password length): suffix NOT shown — only prefix+stars
		{"abcdefgh", "abc**********"},
		// 11 chars (one below threshold): suffix NOT shown
		{"abcdefghijk", "abc**********"},
		// 4..3 chars: prefix shown, no suffix
		{"abcdefg", "abc**********"},
		{"abcd", "abc**********"},
		// <=3 chars: fully masked
		{"abc", "**********"},
		{"", "**********"},
	}
	for _, tt := range tests {
		if got := maskSecret(tt.input); got != tt.want {
			t.Errorf("maskSecret(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
