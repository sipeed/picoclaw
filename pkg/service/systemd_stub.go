//go:build !linux

package service

func newSystemdUserManager(exePath string, runner commandRunner) Manager {
	return newUnsupportedManager("systemd user services are only available on Linux")
}
