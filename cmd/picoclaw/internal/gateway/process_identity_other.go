//go:build !linux

package gateway

func verifyGatewayProcessIdentity(processID int) error {
	// Best-effort: non-Linux platforms don't have a portable, dependency-free way
	// to validate /proc-style executable + argv identity. Don't block status/stop.
	return nil
}
