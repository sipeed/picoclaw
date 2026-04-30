//go:build !linux && !darwin && !windows

package hardwaretools

import (
	"strings"
	"testing"
)

func TestSerialListPortsUnsupportedPlatform(t *testing.T) {
	_, err := serialListPorts()
	if err == nil {
		t.Fatal("expected unsupported platform error")
	}
	if !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("serialListPorts() error = %v, want unsupported platform message", err)
	}
}
