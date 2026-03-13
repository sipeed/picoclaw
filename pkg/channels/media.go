package channels

import (
	"context"

	"jane/pkg/bus"
)

// MediaSender is an optional interface for channels that can send
// media attachments (images, files, audio, video).
// Manager discovers channels implementing this interface via type
// assertion and routes OutboundMediaMessage to them.
type MediaSender interface {
	SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) error
}
