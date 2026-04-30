//go:build darwin

package hardwaretools

import "golang.org/x/sys/unix"

func serialGetTermios(fd int) (*unix.Termios, error) {
	return unix.IoctlGetTermios(fd, unix.TIOCGETA)
}

func serialSetSpeed(tio *unix.Termios, speed uint32) error {
	tio.Ispeed = uint64(speed)
	tio.Ospeed = uint64(speed)
	return nil
}

func serialSetTermios(fd int, tio *unix.Termios) error {
	return unix.IoctlSetTermios(fd, unix.TIOCSETA, tio)
}
