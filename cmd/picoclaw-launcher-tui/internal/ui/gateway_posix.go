//go:build !windows
// +build !windows

package ui

import "os/exec"

var execCommand = exec.Command

func isGatewayProcessRunning() bool {
	cmd := execCommand("sh", "-c", "pgrep -f 'picoclaw\\s+gateway' >/dev/null 2>&1")
	return cmd.Run() == nil
}

func stopGatewayProcess() error {
	cmd := execCommand("sh", "-c", "pkill -f 'picoclaw\\s+gateway' >/dev/null 2>&1")
	return cmd.Run()
}
