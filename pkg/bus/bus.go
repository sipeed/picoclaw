package bus

import (
	"context"
	"errors"
	"sync"
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
	wg            sync.WaitGroup
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
	mb.wg.Add(1)
	defer mb.wg.Done()

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
	mb.wg.Add(1)
	defer mb.wg.Done()

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
	mb.wg.Add(1)
	defer mb.wg.Done()

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
	if !mb.closed.CompareAndSwap(false, true) {
		return
	}

	// Notify all Pub/Sub that i will be closing down
	close(mb.done)

	// Make sure all Pub was done
	mb.wg.Wait()

	// Batch empty the chan
	drained := 0

	drained += drain(mb.inbound)
	drained += drain(mb.outbound)
	drained += drain(mb.outboundMedia)

	if drained > 0 {
		logger.DebugCF("bus", "Drained buffered messages during close", map[string]any{
			"count": drained,
		})
	}

}

func drain[T any](ch <-chan T) int {
	n := 0
	for {
		select {
		case <-ch:
			n++
		default:
			return n
		}
	}
}
