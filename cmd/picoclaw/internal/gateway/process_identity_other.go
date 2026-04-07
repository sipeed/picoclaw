//go:build !linux

package gateway

import "fmt"

func verifyGatewayProcessIdentity(processID int) error {
	return fmt.Errorf(
		"gateway process identity verification is not supported on this platform (PID: %d)",
		processID,
	)
}
