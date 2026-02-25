//go:build linux

package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type systemdUserManager struct {
	runner    commandRunner
	exePath   string
	unitName  string
	unitPath  string
	journalID string
}

func newSystemdUserManager(exePath string, runner commandRunner) Manager {
	home, _ := os.UserHomeDir()
	unitName := "picoclaw-gateway.service"
	return &systemdUserManager{
		runner:    runner,
		exePath:   exePath,
		unitName:  unitName,
		unitPath:  filepath.Join(home, ".config", "systemd", "user", unitName),
		journalID: unitName,
	}
}

func (m *systemdUserManager) Backend() string { return BackendSystemdUser }

func (m *systemdUserManager) Install() error {
	if err := os.MkdirAll(filepath.Dir(m.unitPath), 0755); err != nil {
		return err
	}
	pathEnv := buildSystemdPath(os.Getenv("PATH"), m.detectBrewPrefix())
	unit := renderSystemdUnit(m.exePath, pathEnv)
	if err := writeFileIfChanged(m.unitPath, []byte(unit), 0644); err != nil {
		return err
	}
	if out, err := runCommand(m.runner, 8*time.Second, "systemctl", "--user", "daemon-reload"); err != nil {
		return fmt.Errorf("daemon-reload failed: %s", oneLine(string(out)))
	}
	if out, err := runCommand(m.runner, 8*time.Second, "systemctl", "--user", "enable", m.unitName); err != nil {
		return fmt.Errorf("enable failed: %s", oneLine(string(out)))
	}
	return nil
}

func (m *systemdUserManager) Uninstall() error {
	_, _ = runCommand(m.runner, 8*time.Second, "systemctl", "--user", "disable", "--now", m.unitName)
	if err := os.Remove(m.unitPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	_, _ = runCommand(m.runner, 8*time.Second, "systemctl", "--user", "daemon-reload")
	return nil
}

func (m *systemdUserManager) Start() error {
	if _, err := os.Stat(m.unitPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("service is not installed; run `picoclaw service install`")
		}
		return err
	}
	if out, err := runCommand(m.runner, 12*time.Second, "systemctl", "--user", "start", m.unitName); err != nil {
		return fmt.Errorf("start failed: %s", oneLine(string(out)))
	}
	return nil
}

func (m *systemdUserManager) Stop() error {
	if out, err := runCommand(m.runner, 12*time.Second, "systemctl", "--user", "stop", m.unitName); err != nil {
		msg := strings.ToLower(string(out))
		if strings.Contains(msg, "not loaded") || strings.Contains(msg, "could not be found") {
			return nil
		}
		return fmt.Errorf("stop failed: %s", oneLine(string(out)))
	}
	return nil
}

func (m *systemdUserManager) Restart() error {
	if _, err := os.Stat(m.unitPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("service is not installed; run `picoclaw service install`")
		}
		return err
	}
	if out, err := runCommand(m.runner, 12*time.Second, "systemctl", "--user", "restart", m.unitName); err != nil {
		return fmt.Errorf("restart failed: %s", oneLine(string(out)))
	}
	return nil
}

func (m *systemdUserManager) Status() (Status, error) {
	st := Status{Backend: BackendSystemdUser}
	if _, err := os.Stat(m.unitPath); err == nil {
		st.Installed = true
	}

	if out, err := runCommand(m.runner, 5*time.Second, "systemctl", "--user", "is-active", m.unitName); err == nil {
		if strings.TrimSpace(string(out)) == "active" {
			st.Running = true
		}
	}

	if out, err := runCommand(m.runner, 5*time.Second, "systemctl", "--user", "is-enabled", m.unitName); err == nil {
		en := strings.TrimSpace(string(out))
		if strings.HasPrefix(en, "enabled") {
			st.Enabled = true
		}
	}

	if !st.Installed {
		st.Detail = "unit file not installed"
	} else if !st.Running {
		st.Detail = "installed but not running"
	}
	return st, nil
}

func (m *systemdUserManager) Logs(lines int) (string, error) {
	if lines <= 0 {
		lines = 100
	}
	out, err := runCommand(m.runner, 10*time.Second, "journalctl", "--user", "-u", m.journalID, "-n", fmt.Sprintf("%d", lines), "--no-pager")
	if err != nil {
		return "", fmt.Errorf("journalctl failed: %s", oneLine(string(out)))
	}
	return string(out), nil
}

func (m *systemdUserManager) LogsFollow(ctx context.Context, lines int, w io.Writer) error {
	n := lines
	if n <= 0 {
		n = 100
	}
	cmd := exec.CommandContext(ctx, "journalctl", "--user", "-u", m.journalID, "-n", fmt.Sprintf("%d", n), "-f", "--no-pager")
	cmd.Stdout = w
	cmd.Stderr = w
	return cmd.Run()
}

func (m *systemdUserManager) detectBrewPrefix() string {
	if _, err := exec.LookPath("brew"); err != nil {
		return ""
	}
	out, err := runCommand(m.runner, 4*time.Second, "brew", "--prefix")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
