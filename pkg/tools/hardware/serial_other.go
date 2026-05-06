//go:build !linux && !darwin && !windows

package hardwaretools

import (
	"context"
	"fmt"
	"time"
)

func serialListPorts() ([]serialPortInfo, error) {
	return nil, fmt.Errorf("serial is not supported on this platform")
}

func serialRead(ctx context.Context, cfg serialConfig, length int, timeout time.Duration) ([]byte, error) {
	return nil, fmt.Errorf("serial is not supported on this platform")
}

func serialWrite(ctx context.Context, cfg serialConfig, data []byte, timeout time.Duration) (int, error) {
	return 0, fmt.Errorf("serial is not supported on this platform")
}
