package api

import (
	"path/filepath"
	"strings"
	"testing"

	"jane/web/backend/launcherconfig"
)

func TestResolveLaunchCommandUsesConfigFileDefaults(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)

	// Persist non-default launcher options to ensure resolveLaunchCommand does not
	// pin them into autostart args.
	launcherPath := launcherconfig.PathForAppConfig(configPath)
	if err := launcherconfig.Save(launcherPath, launcherconfig.Config{
		Port:   19999,
		Public: true,
	}); err != nil {
		t.Fatalf("launcherconfig.Save() error = %v", err)
	}

	exePath, args, err := h.resolveLaunchCommand()
	if err != nil {
		t.Fatalf("resolveLaunchCommand() error = %v", err)
	}
	if exePath == "" {
		t.Fatal("resolveLaunchCommand() returned empty executable path")
	}
	if len(args) != 2 {
		t.Fatalf("args len = %d, want 2 (got %v)", len(args), args)
	}
	if args[0] != "-no-browser" {
		t.Fatalf("args[0] = %q, want %q", args[0], "-no-browser")
	}
	if args[1] != configPath {
		t.Fatalf("args[1] = %q, want %q", args[1], configPath)
	}
	for _, arg := range args {
		if arg == "-port" || arg == "-public" {
			t.Fatalf("autostart args should not pin network flags, got %v", args)
		}
	}
}

func TestBuildDarwinPlistIncludesRunAtLoad(t *testing.T) {
	plist := buildDarwinPlist("/tmp/picoclaw-web", []string{"-no-browser", "/tmp/config.json"})
	if !strings.Contains(plist, "<key>RunAtLoad</key>") {
		t.Fatalf("plist missing RunAtLoad key:\n%s", plist)
	}
	if !strings.Contains(plist, "<true/>") {
		t.Fatalf("plist missing RunAtLoad true value:\n%s", plist)
	}
}
