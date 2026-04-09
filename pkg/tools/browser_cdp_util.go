//go:build cdp

package tools

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// FindChromePath attempts to locate a Chrome or Chromium executable on the system.
// It checks (in order): CHROME_PATH env var, platform-specific known paths,
// and finally exec.LookPath for common binary names.
func FindChromePath() (string, error) {
	// 1. Check environment variable
	if p := os.Getenv("CHROME_PATH"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	// 2. Platform-specific known paths
	var candidates []string

	switch runtime.GOOS {
	case "darwin":
		candidates = []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"/Applications/Google Chrome Canary.app/Contents/MacOS/Google Chrome Canary",
			"/opt/homebrew/bin/chromium",
			"/opt/homebrew/bin/chrome",
			"/usr/local/bin/chromium",
			"/usr/local/bin/chrome",
		}
	case "linux":
		candidates = []string{
			"/usr/bin/chromium",
			"/usr/bin/chromium-browser",
			"/usr/bin/google-chrome",
			"/usr/bin/google-chrome-stable",
			"/snap/bin/chromium",
		}
		if home := os.Getenv("HOME"); home != "" {
			candidates = append(candidates, filepath.Join(home, ".local", "bin", "chromium"))
		}
	case "windows":
		pf := os.Getenv("ProgramFiles")
		pfx86 := os.Getenv("ProgramFiles(x86)")
		localAppData := os.Getenv("LOCALAPPDATA")
		candidates = []string{
			filepath.Join(pf, "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(pfx86, "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(pf, "Chromium", "Application", "chrome.exe"),
			filepath.Join(localAppData, "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(localAppData, "Chromium", "Application", "chrome.exe"),
		}
	}

	// 3. Check candidate paths
	for _, p := range candidates {
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	// 4. Fallback to exec.LookPath
	for _, name := range []string{"chromium", "chromium-browser", "google-chrome", "google-chrome-stable", "chrome"} {
		if p, err := exec.LookPath(name); err == nil {
			return p, nil
		}
	}

	return "", fmt.Errorf(
		"Chrome/Chromium not found on this system. " +
			"Install from https://google.com/chrome or set CHROME_PATH environment variable")
}

// LaunchChromeArgs returns the command-line arguments to launch Chrome
// with remote debugging enabled.
func LaunchChromeArgs(debugPort int, headless bool) []string {
	args := []string{
		fmt.Sprintf("--remote-debugging-port=%d", debugPort),
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-background-networking",
		"--disable-extensions",
	}
	if headless {
		args = append(args, "--headless=new")
	}
	return args
}
