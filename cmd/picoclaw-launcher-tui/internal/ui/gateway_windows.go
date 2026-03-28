//go:build windows
// +build windows

package ui

import "os/exec"

var execCommand = exec.Command

func isGatewayProcessRunning() bool {
	cmd := execCommand("cmd", "/C", `tasklist /FI "IMAGENAME eq jane-ai.exe" | findstr /I /C:"jane-ai.exe" >NUL || tasklist /FI "IMAGENAME eq picoclaw.exe" | findstr /I /C:"picoclaw.exe" >NUL`)
	return cmd.Run() == nil
}

func stopGatewayProcess() error {
	cmd := execCommand("cmd", "/C", `taskkill /F /IM jane-ai.exe >NUL 2>&1 || taskkill /F /IM picoclaw.exe >NUL 2>&1`)
	return cmd.Run()
}
