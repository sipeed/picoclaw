//go:build !linux

package sources

import (
	"context"

	"github.com/sipeed/picoclaw/pkg/devices/events"
)

type USBMonitor struct{}

func NewUSBMonitor() *USBMonitor {
	return &USBMonitor{}
}

func (m *USBMonitor) Kind() events.Kind {
	return events.KindUSB
}

func (m *USBMonitor) Start(ctx context.Context) (<-chan *events.DeviceEvent, error) {
	ch := make(chan *events.DeviceEvent)
	close(ch) // Immediately close, no events
	return ch, nil
}

func (m *USBMonitor) Stop() error {
	return nil
}
