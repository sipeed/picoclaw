package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sipeed/picoclaw/pkg/logger"
)

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected logger.LogLevel
	}{
		{"debug", logger.DEBUG},
		{"DEBUG", logger.DEBUG},
		{"warn", logger.WARN},
		{"error", logger.ERROR},
		{"info", logger.INFO},
		{"unknown", logger.INFO}, // Default fallback
		{"", logger.INFO},        // Empty string fallback
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := parseLogLevel(tt.input); got != tt.expected {
				t.Errorf("parseLogLevel(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Skipping test: unable to get user home directory")
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Home directory root", "~", home},
		{"Home directory subpath", "~/logs/audit.log", filepath.Join(home, "/logs/audit.log")},
		{"Absolute path", "/var/log/app.log", "/var/log/app.log"},
		{"Relative path", "./logs/app.log", "./logs/app.log"},
		{"Other user home (should not expand)", "~user/logs.txt", "~user/logs.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := expandPath(tt.input); got != tt.expected {
				t.Errorf("expandPath(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
