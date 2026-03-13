//go:build windows
// +build windows

package ui

import "os/exec"

var execCommand = exec.Command

func isGatewayProcessRunning() bool {
	cmd := execCommand("tasklist", "/FI", "IMAGENAME eq picoclaw.exe")
	return cmd.Run() == nil
}

func stopGatewayProcess() error {
	cmd := execCommand("taskkill", "/F", "/IM", "picoclaw.exe")
	return cmd.Run()
}
