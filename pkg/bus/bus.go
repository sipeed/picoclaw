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

// StreamDelegate is implemented by the channel Manager to provide streaming
// capabilities to the agent loop without tight coupling.
type StreamDelegate interface {
	// GetStreamer returns a Streamer for the given channel+chatID if the channel
	// supports streaming. Returns nil, false if streaming is unavailable.
	GetStreamer(ctx context.Context, channel, chatID string) (Streamer, bool)
}

// Streamer pushes incremental content to a streaming-capable channel.
// Defined here so the agent loop can use it without importing pkg/channels.
type Streamer interface {
	Update(ctx context.Context, content string) error
	Finalize(ctx context.Context, content string) error
	Cancel(ctx context.Context)
}

type MessageBus struct {
	inbound        chan InboundMessage
	outbound       chan OutboundMessage
	outboundMedia  chan OutboundMediaMessage
	done           chan struct{}
	closed         atomic.Bool
	streamDelegate atomic.Value // stores StreamDelegate
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

// SetStreamDelegate registers a StreamDelegate (typically the channel Manager).
func (mb *MessageBus) SetStreamDelegate(d StreamDelegate) {
	mb.streamDelegate.Store(d)
}

// GetStreamer returns a Streamer for the given channel+chatID via the delegate.
func (mb *MessageBus) GetStreamer(ctx context.Context, channel, chatID string) (Streamer, bool) {
	if d, ok := mb.streamDelegate.Load().(StreamDelegate); ok && d != nil {
		return d.GetStreamer(ctx, channel, chatID)
	}
	return nil, false
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
