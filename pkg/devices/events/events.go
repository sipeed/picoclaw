package events

import (
	"context"
	"strings"
)

type EventSource interface {
	Kind() Kind
	Start(ctx context.Context) (<-chan *DeviceEvent, error)
	Stop() error
}

type Action string

const (
	ActionAdd    Action = "add"
	ActionRemove Action = "remove"
	ActionChange Action = "change"
)

type Kind string

const (
	KindUSB       Kind = "usb"
	KindBluetooth Kind = "bluetooth"
	KindPCI       Kind = "pci"
	KindGeneric   Kind = "generic"
)

type DeviceEvent struct {
	Action       Action
	Kind         Kind
	DeviceID     string            // e.g. "1-2" for USB bus 1 dev 2
	Vendor       string            // Vendor name or ID
	Product      string            // Product name or ID
	Serial       string            // Serial number if available
	Capabilities string            // Human-readable capability description
	Raw          map[string]string // Raw properties for extensibility
}

func (e *DeviceEvent) FormatMessage() string {
	actionEmoji := "🔌"
	actionText := "Connected"
	if e.Action == ActionRemove {
		actionEmoji = "🔌"
		actionText = "Disconnected"
	}

	var b strings.Builder
	b.WriteString(actionEmoji)
	b.WriteString(" Device ")
	b.WriteString(actionText)
	b.WriteString("\n\nType: ")
	b.WriteString(string(e.Kind))
	b.WriteString("\nDevice: ")
	b.WriteString(e.Vendor)
	b.WriteByte(' ')
	b.WriteString(e.Product)
	b.WriteByte('\n')
	if e.Capabilities != "" {
		b.WriteString("Capabilities: ")
		b.WriteString(e.Capabilities)
		b.WriteByte('\n')
	}
	if e.Serial != "" {
		b.WriteString("Serial: ")
		b.WriteString(e.Serial)
		b.WriteByte('\n')
	}
	return b.String()
}
