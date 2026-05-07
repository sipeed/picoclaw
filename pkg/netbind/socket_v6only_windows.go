//go:build windows

package netbind

import (
	"syscall"

	"golang.org/x/sys/windows"
)

func applyIPv6OnlyControl(enabled bool) func(string, string, syscall.RawConn) error {
	return func(_, _ string, rawConn syscall.RawConn) error {
		var controlErr error
		if err := rawConn.Control(func(fd uintptr) {
			value := 0
			if enabled {
				value = 1
			}
			controlErr = windows.SetsockoptInt(windows.Handle(fd), windows.IPPROTO_IPV6, windows.IPV6_V6ONLY, value)
		}); err != nil {
			return err
		}
		return controlErr
	}
}
