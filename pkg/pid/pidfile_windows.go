//go:build windows

package pid

import (
	"syscall"
	"unsafe"
)

var (
	kernel32                       = syscall.NewLazyDLL("kernel32.dll")
	procOpenProcess                = kernel32.NewProc("OpenProcess")
	procGetExitCodeProcess         = kernel32.NewProc("GetExitCodeProcess")
	procCloseHandle                = kernel32.NewProc("CloseHandle")
	processQueryLimitedInformation = uint32(0x1000)
	stillActive                    = uint32(259)
)

// isProcessRunning checks whether a process with the given PID is alive
// on Windows using OpenProcess + GetExitCodeProcess.
func isProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	handle, _, err := procOpenProcess.Call(
		uintptr(processQueryLimitedInformation),
		0,
		uintptr(pid),
	)
	if handle == 0 || err != nil {
		return false
	}
	defer procCloseHandle.Call(handle)

	var exitCode uint32
	ret, _, err := procGetExitCodeProcess.Call(handle, uintptr(unsafe.Pointer(&exitCode)))
	if ret == 0 || err != nil {
		return false
	}
	return exitCode == stillActive
}
