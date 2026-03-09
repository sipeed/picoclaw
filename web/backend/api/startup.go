package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	autoStartEntryName = "PicoClawLauncher"
	launchAgentLabel   = "io.picoclaw.launcher"
)

type autoStartRequest struct {
	Enabled bool `json:"enabled"`
}

type autoStartResponse struct {
	Enabled   bool   `json:"enabled"`
	Supported bool   `json:"supported"`
	Platform  string `json:"platform"`
	Message   string `json:"message,omitempty"`
}

var errAutoStartUnsupported = errors.New("autostart is not supported on this platform")

func (h *Handler) registerStartupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/system/autostart", h.handleGetAutoStart)
	mux.HandleFunc("PUT /api/system/autostart", h.handleSetAutoStart)
}

func (h *Handler) handleGetAutoStart(w http.ResponseWriter, r *http.Request) {
	enabled, supported, message, err := h.getAutoStartStatus()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read startup setting: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(autoStartResponse{
		Enabled:   enabled,
		Supported: supported,
		Platform:  runtime.GOOS,
		Message:   message,
	})
}

func (h *Handler) handleSetAutoStart(w http.ResponseWriter, r *http.Request) {
	var req autoStartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	if err := h.setAutoStart(req.Enabled); err != nil {
		if errors.Is(err, errAutoStartUnsupported) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to update startup setting: %v", err), http.StatusInternalServerError)
		return
	}

	enabled, supported, message, err := h.getAutoStartStatus()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to verify startup setting: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(autoStartResponse{
		Enabled:   enabled,
		Supported: supported,
		Platform:  runtime.GOOS,
		Message:   message,
	})
}

func (h *Handler) resolveLaunchCommand() (string, []string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", nil, err
	}

	args := []string{"-no-browser"}
	if h.configPath != "" {
		args = append(args, h.configPath)
	}

	return exePath, args, nil
}

func (h *Handler) getAutoStartStatus() (enabled bool, supported bool, message string, err error) {
	switch runtime.GOOS {
	case "darwin":
		exists, err := fileExists(macLaunchAgentPath())
		return exists, true, "Changes apply on next login.", err
	case "linux":
		exists, err := fileExists(linuxAutoStartPath())
		return exists, true, "Changes apply on next login.", err
	case "windows":
		exists, err := windowsRunKeyExists()
		return exists, true, "Changes apply on next login.", err
	default:
		return false, false, "Current platform does not support launch at login.", nil
	}
}

func (h *Handler) setAutoStart(enabled bool) error {
	exePath, args, err := h.resolveLaunchCommand()
	if err != nil {
		return err
	}

	switch runtime.GOOS {
	case "darwin":
		return setDarwinAutoStart(enabled, exePath, args)
	case "linux":
		return setLinuxAutoStart(enabled, exePath, args)
	case "windows":
		return setWindowsAutoStart(enabled, exePath, args)
	default:
		return errAutoStartUnsupported
	}
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func macLaunchAgentPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", launchAgentLabel+".plist")
}

func setDarwinAutoStart(enabled bool, exePath string, args []string) error {
	plistPath := macLaunchAgentPath()
	if enabled {
		if err := os.MkdirAll(filepath.Dir(plistPath), 0o755); err != nil {
			return err
		}
		content := buildDarwinPlist(exePath, args)
		return os.WriteFile(plistPath, []byte(content), 0o644)
	}

	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func xmlEscape(s string) string {
	var b bytes.Buffer
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		case '\'':
			b.WriteString("&apos;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func buildDarwinPlist(exePath string, args []string) string {
	programArgs := make([]string, 0, len(args)+1)
	programArgs = append(programArgs, exePath)
	programArgs = append(programArgs, args...)

	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(
		`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">` + "\n",
	)
	b.WriteString(`<plist version="1.0">` + "\n")
	b.WriteString(`<dict>` + "\n")
	b.WriteString(`  <key>Label</key>` + "\n")
	b.WriteString(`  <string>` + launchAgentLabel + `</string>` + "\n")
	b.WriteString(`  <key>ProgramArguments</key>` + "\n")
	b.WriteString(`  <array>` + "\n")
	for _, arg := range programArgs {
		b.WriteString(`    <string>` + xmlEscape(arg) + `</string>` + "\n")
	}
	b.WriteString(`  </array>` + "\n")
	b.WriteString(`  <key>RunAtLoad</key>` + "\n")
	b.WriteString(`  <true/>` + "\n")
	b.WriteString(`  <key>ProcessType</key>` + "\n")
	b.WriteString(`  <string>Background</string>` + "\n")
	b.WriteString(`</dict>` + "\n")
	b.WriteString(`</plist>` + "\n")
	return b.String()
}

func linuxAutoStartPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "autostart", "picoclaw-web.desktop")
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if !strings.ContainsAny(s, " \t\n'\"\\$`") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

func buildLinuxExecLine(exePath string, args []string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, shellQuote(exePath))
	for _, arg := range args {
		parts = append(parts, shellQuote(arg))
	}
	return strings.Join(parts, " ")
}

func setLinuxAutoStart(enabled bool, exePath string, args []string) error {
	desktopPath := linuxAutoStartPath()
	if enabled {
		if err := os.MkdirAll(filepath.Dir(desktopPath), 0o755); err != nil {
			return err
		}
		content := strings.Join([]string{
			"[Desktop Entry]",
			"Type=Application",
			"Version=1.0",
			"Name=PicoClaw Web",
			"Comment=Start PicoClaw Web on login",
			"Exec=" + buildLinuxExecLine(exePath, args),
			"Terminal=false",
			"X-GNOME-Autostart-enabled=true",
			"NoDisplay=true",
			"",
		}, "\n")
		return os.WriteFile(desktopPath, []byte(content), 0o644)
	}

	if err := os.Remove(desktopPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func windowsCommandLine(exePath string, args []string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, fmt.Sprintf("%q", exePath))
	for _, arg := range args {
		parts = append(parts, fmt.Sprintf("%q", arg))
	}
	return strings.Join(parts, " ")
}

func windowsRunKeyExists() (bool, error) {
	cmd := exec.Command("reg", "query", `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, "/v", autoStartEntryName)
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func setWindowsAutoStart(enabled bool, exePath string, args []string) error {
	key := `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`
	if enabled {
		commandLine := windowsCommandLine(exePath, args)
		cmd := exec.Command("reg", "add", key, "/v", autoStartEntryName, "/t", "REG_SZ", "/d", commandLine, "/f")
		return cmd.Run()
	}

	cmd := exec.Command("reg", "delete", key, "/v", autoStartEntryName, "/f")
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil
		}
		return err
	}
	return nil
}
