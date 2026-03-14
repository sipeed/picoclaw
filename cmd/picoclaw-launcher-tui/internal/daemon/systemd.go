package daemon

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	serviceName     = "picoclaw-launcher.service"
	serviceFileName = serviceName
)

type Status struct {
	Enabled bool
	Active  bool
}

func Supported() bool {
	return runtime.GOOS == "linux"
}

func userUnitPath(home string) string {
	return filepath.Join(home, ".config", "systemd", "user", serviceFileName)
}

func buildExecStart(exePath, configPath string, public bool) string {
	parts := []string{quoteArg(exePath), "-no-browser"}
	if public {
		parts = append(parts, "-public")
	}
	parts = append(parts, quoteArg(configPath))
	return strings.Join(parts, " ")
}

func quoteArg(v string) string {
	if v == "" {
		return "\"\""
	}
	if !strings.ContainsAny(v, " \t\n\"\\") {
		return v
	}
	return fmt.Sprintf("%q", v)
}

func buildUnitContent(exePath, configPath string, public bool) string {
	execStart := buildExecStart(exePath, configPath, public)
	return strings.Join([]string{
		"[Unit]",
		"Description=PicoClaw Launcher (no browser)",
		"After=network.target",
		"",
		"[Service]",
		"Type=simple",
		"ExecStart=" + execStart,
		"Restart=on-failure",
		"RestartSec=2",
		"",
		"[Install]",
		"WantedBy=default.target",
		"",
	}, "\n")
}

func resolveLauncherExecutable() (string, error) {
	path, err := exec.LookPath("picoclaw-launcher")
	if err != nil {
		return "", fmt.Errorf("failed to find picoclaw-launcher in PATH: %w", err)
	}
	return path, nil
}

func writeUnit(configPath string, public bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	unitPath := userUnitPath(home)
	if err := os.MkdirAll(filepath.Dir(unitPath), 0o755); err != nil {
		return err
	}
	exePath, err := resolveLauncherExecutable()
	if err != nil {
		return err
	}
	content := buildUnitContent(exePath, configPath, public)
	return os.WriteFile(unitPath, []byte(content), 0o644)
}

func Enable(configPath string, public bool) error {
	if !Supported() {
		return fmt.Errorf("systemd user service is only supported on linux")
	}
	if err := writeUnit(configPath, public); err != nil {
		return err
	}
	if err := runSystemctl("daemon-reload"); err != nil {
		return err
	}
	return runSystemctl("enable", "--now", serviceName)
}

func Disable() error {
	if !Supported() {
		return fmt.Errorf("systemd user service is only supported on linux")
	}
	return runSystemctl("disable", "--now", serviceName)
}

func Restart(configPath string, public bool) error {
	if !Supported() {
		return fmt.Errorf("systemd user service is only supported on linux")
	}
	if err := writeUnit(configPath, public); err != nil {
		return err
	}
	if err := runSystemctl("daemon-reload"); err != nil {
		return err
	}
	return runSystemctl("restart", serviceName)
}

func GetStatus() (Status, error) {
	if !Supported() {
		return Status{}, fmt.Errorf("systemd user service is only supported on linux")
	}
	enabled, err := systemctlState("is-enabled", serviceName)
	if err != nil {
		return Status{}, err
	}
	active, err := systemctlState("is-active", serviceName)
	if err != nil {
		return Status{}, err
	}
	return Status{Enabled: enabled == "enabled", Active: active == "active"}, nil
}

func systemctlState(args ...string) (string, error) {
	cmd := exec.Command("systemctl", append([]string{"--user"}, args...)...)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		s := strings.TrimSpace(out.String())
		if s == "disabled" || s == "inactive" || s == "failed" || s == "not-found" {
			return s, nil
		}
		se := strings.TrimSpace(stderr.String())
		if se == "" {
			se = err.Error()
		}
		return "", fmt.Errorf("systemctl %s failed: %s", strings.Join(args, " "), se)
	}
	return strings.TrimSpace(out.String()), nil
}

func runSystemctl(args ...string) error {
	cmd := exec.Command("systemctl", append([]string{"--user"}, args...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("systemctl %s failed: %s", strings.Join(args, " "), msg)
	}
	return nil
}
