//go:build !windows

package main

import "os/exec"

// hideProcessWindow is a no-op on non-Windows platforms.
func hideProcessWindow(cmd *exec.Cmd) {}
