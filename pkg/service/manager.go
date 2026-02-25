package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const (
	BackendLaunchd      = "launchd"
	BackendSystemdUser  = "systemd-user"
	BackendUnsupported  = "unsupported"
)

// Status captures installation and runtime state for the background gateway service.
type Status struct {
	Backend   string
	Installed bool
	Running   bool
	Enabled   bool
	Detail    string
}

// Manager controls the background gateway service for the current platform.
type Manager interface {
	Backend() string
	Install() error
	Uninstall() error
	Start() error
	Stop() error
	Restart() error
	Status() (Status, error)
	Logs(lines int) (string, error)
	// LogsFollow streams log output to w until ctx is done (e.g. SIGINT). Follows new lines like tail -f.
	LogsFollow(ctx context.Context, lines int, w io.Writer) error
}

type commandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

type osCommandRunner struct{}

func (osCommandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

func NewManager(exePath string) (Manager, error) {
	exePath = strings.TrimSpace(exePath)
	if exePath == "" {
		return nil, errors.New("executable path is empty")
	}

	runner := osCommandRunner{}
	switch runtime.GOOS {
	case "darwin":
		return newLaunchdManager(exePath, runner), nil
	case "linux":
		if detectWSL() && !isSystemdUserAvailable(runner) {
			return newUnsupportedManager("WSL detected but systemd user manager is not active. Enable systemd in /etc/wsl.conf or run `picoclaw gateway` in a terminal."), nil
		}
		if !isSystemdUserAvailable(runner) {
			return newUnsupportedManager("systemd user manager is not available. Run `picoclaw gateway` in a terminal."), nil
		}
		return newSystemdUserManager(exePath, runner), nil
	default:
		return newUnsupportedManager(fmt.Sprintf("%s is not currently supported for service management", runtime.GOOS)), nil
	}
}

func isSystemdUserAvailable(runner commandRunner) bool {
	if _, err := exec.LookPath("systemctl"); err != nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := runner.Run(ctx, "systemctl", "--user", "show-environment")
	return err == nil
}

func runCommand(runner commandRunner, timeout time.Duration, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	out, err := runner.Run(ctx, name, args...)
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return out, fmt.Errorf("%s %s timed out", name, strings.Join(args, " "))
	}
	if err != nil {
		return out, fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return out, nil
}

func writeFileIfChanged(path string, content []byte, perm os.FileMode) error {
	if existing, err := os.ReadFile(path); err == nil {
		if string(existing) == string(content) {
			return nil
		}
	}
	if err := os.WriteFile(path, content, perm); err != nil {
		return err
	}
	return nil
}

func oneLine(s string) string {
	s = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(s, "\n", " "), "\r", " "))
	if s == "" {
		return ""
	}
	if len(s) > 220 {
		return s[:220] + "..."
	}
	return s
}
