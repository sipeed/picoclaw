//go:build windows

package hardwaretools

import "testing"

func TestSanitizeWindowsSerialFlags(t *testing.T) {
	flags := uint32(
		dcbFlagBinary |
			dcbFlagParity |
			dcbFlagOutxCtsFlow |
			dcbFlagOutxDsrFlow |
			dcbFlagDtrControlMask |
			dcbFlagDsrSensitivity |
			dcbFlagTXContinueOnXoff |
			dcbFlagOutX |
			dcbFlagInX |
			dcbFlagRtsControlMask,
	)

	got := sanitizeWindowsSerialFlags(flags)

	if got&dcbFlagBinary == 0 {
		t.Fatal("sanitizeWindowsSerialFlags() should preserve fBinary")
	}
	if got&dcbFlagParity == 0 {
		t.Fatal("sanitizeWindowsSerialFlags() should preserve fParity")
	}
	if got&(dcbFlagOutxCtsFlow|
		dcbFlagOutxDsrFlow|
		dcbFlagDtrControlMask|
		dcbFlagDsrSensitivity|
		dcbFlagTXContinueOnXoff|
		dcbFlagOutX|
		dcbFlagInX|
		dcbFlagRtsControlMask) != 0 {
		t.Fatalf("sanitizeWindowsSerialFlags() = %#x, want flow-control bits cleared", got)
	}
}
