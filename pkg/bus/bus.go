package bus

import (
	"context"
	"errors"
	"sync/atomic"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// ErrBusClosed is returned when publishing to a closed MessageBus.
var ErrBusClosed = errors.New("message bus closed")

const defaultBusBufferSize = 64

type MessageBus struct {
	inbound       chan InboundMessage
	outbound      chan OutboundMessage
	outboundMedia chan OutboundMediaMessage
	done          chan struct{}
	closed        atomic.Bool
}

func NewMessageBus() *MessageBus {
	return &MessageBus{
		inbound:       make(chan InboundMessage, defaultBusBufferSize),
		outbound:      make(chan OutboundMessage, defaultBusBufferSize),
		outboundMedia: make(chan OutboundMediaMessage, defaultBusBufferSize),
		done:          make(chan struct{}),
	}
}

func (mb *MessageBus) PublishInbound(ctx context.Context, msg InboundMessage) error {
	if mb.closed.Load() {
		return ErrBusClosed
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case mb.inbound <- msg:
		return nil
	case <-mb.done:
		return ErrBusClosed
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (mb *MessageBus) ConsumeInbound(ctx context.Context) (InboundMessage, bool) {
	select {
	case msg, ok := <-mb.inbound:
		return msg, ok
	case <-mb.done:
		return InboundMessage{}, false
	case <-ctx.Done():
		return InboundMessage{}, false
	}
}

func (mb *MessageBus) PublishOutbound(ctx context.Context, msg OutboundMessage) error {
	if mb.closed.Load() {
		return ErrBusClosed
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case mb.outbound <- msg:
		return nil
	case <-mb.done:
		return ErrBusClosed
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (mb *MessageBus) SubscribeOutbound(ctx context.Context) (OutboundMessage, bool) {
	select {
	case msg, ok := <-mb.outbound:
		return msg, ok
	case <-mb.done:
		return OutboundMessage{}, false
	case <-ctx.Done():
		return OutboundMessage{}, false
	}
}

func (mb *MessageBus) PublishOutboundMedia(ctx context.Context, msg OutboundMediaMessage) error {
	if mb.closed.Load() {
		return ErrBusClosed
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case mb.outboundMedia <- msg:
		return nil
	case <-mb.done:
		return ErrBusClosed
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (mb *MessageBus) SubscribeOutboundMedia(ctx context.Context) (OutboundMediaMessage, bool) {
	select {
	case msg, ok := <-mb.outboundMedia:
		return msg, ok
	case <-mb.done:
		return OutboundMediaMessage{}, false
	case <-ctx.Done():
		return OutboundMediaMessage{}, false
	}
}

func (mb *MessageBus) Close() {
	if mb.closed.CompareAndSwap(false, true) {
		close(mb.done)

		// Drain buffered channels so messages aren't silently lost.
		// Channels are NOT closed to avoid send-on-closed panics from concurrent publishers.
		drained := 0
		for {
			select {
			case <-mb.inbound:
				drained++
			default:
				goto doneInbound
			}
		}
	doneInbound:
		for {
			select {
			case <-mb.outbound:
				drained++
			default:
				goto doneOutbound
			}
		}
	doneOutbound:
		for {
			select {
			case <-mb.outboundMedia:
				drained++
			default:
				goto doneMedia
			}
		}
	doneMedia:
		if drained > 0 {
			logger.DebugCF("bus", "Drained buffered messages during close", map[string]any{
				"count": drained,
			})
		}
	}
}
