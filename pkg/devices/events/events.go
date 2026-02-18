package events

import "context"

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

	msg := actionEmoji + " Device " + actionText + "\n\n"
	msg += "Type: " + string(e.Kind) + "\n"
	msg += "Device: " + e.Vendor + " " + e.Product + "\n"
	if e.Capabilities != "" {
		msg += "Capabilities: " + e.Capabilities + "\n"
	}
	if e.Serial != "" {
		msg += "Serial: " + e.Serial + "\n"
	}
	return msg
}
