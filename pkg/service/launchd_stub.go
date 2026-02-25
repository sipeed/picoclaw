//go:build !darwin

package service

func newLaunchdManager(exePath string, runner commandRunner) Manager {
	return newUnsupportedManager("launchd is only available on macOS")
}
