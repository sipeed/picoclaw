//go:build !windows
// +build !windows

package ui

import "os/exec"

func isGatewayProcessRunning() bool {
	cmd := exec.Command("sh", "-c", "pgrep -f 'picoclaw\\s+gateway' >/dev/null 2>&1")
	return cmd.Run() == nil
}

func stopGatewayProcess() error {
	cmd := exec.Command("sh", "-c", "pkill -f 'picoclaw\\s+gateway' >/dev/null 2>&1")
	return cmd.Run()
}
