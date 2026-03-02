//go:build windows
// +build windows

package ui

import "os/exec"

func isGatewayProcessRunning() bool {
	cmd := exec.Command("tasklist", "/FI", "IMAGENAME eq picoclaw.exe")
	return cmd.Run() == nil
}

func stopGatewayProcess() error {
	cmd := exec.Command("taskkill", "/F", "/IM", "picoclaw.exe")
	return cmd.Run()
}
