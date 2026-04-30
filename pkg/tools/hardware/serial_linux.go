//go:build linux

package hardwaretools

import "golang.org/x/sys/unix"

func serialGetTermios(fd int) (*unix.Termios, error) {
	return unix.IoctlGetTermios(fd, unix.TCGETS)
}

func serialSetSpeed(tio *unix.Termios, speed uint32) error {
	tio.Ispeed = speed
	tio.Ospeed = speed
	return nil
}

func serialSetTermios(fd int, tio *unix.Termios) error {
	return unix.IoctlSetTermios(fd, unix.TCSETS, tio)
}
