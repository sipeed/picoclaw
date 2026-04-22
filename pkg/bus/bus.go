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

var (
	ErrMissingInboundContext       = errors.New("inbound message context is required")
	ErrMissingOutboundContext      = errors.New("outbound message context is required")
	ErrMissingOutboundMediaContext = errors.New("outbound media context is required")
)

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
	inbound       chan InboundMessage
	outbound      chan OutboundMessage
	outboundMedia chan OutboundMediaMessage
	audioChunks   chan AudioChunk
	voiceControls chan VoiceControl

	closeOnce      sync.Once
	done           chan struct{}
	closed         atomic.Bool
	wg             sync.WaitGroup
	streamDelegate atomic.Value // stores StreamDelegate
}

func NewMessageBus() *MessageBus {
	return &MessageBus{
		inbound:       make(chan InboundMessage, defaultBusBufferSize),
		outbound:      make(chan OutboundMessage, defaultBusBufferSize),
		outboundMedia: make(chan OutboundMediaMessage, defaultBusBufferSize),
		audioChunks:   make(chan AudioChunk, defaultBusBufferSize*4), // Audio chunks need more buffer.
		voiceControls: make(chan VoiceControl, defaultBusBufferSize),
		done:          make(chan struct{}),
	}
}

func publish[T any](ctx context.Context, mb *MessageBus, ch chan T, msg T) error {
	// check bus closed before acquiring wg, to avoid unnecessary wg.Add and potential deadlock
	if mb.closed.Load() {
		return ErrBusClosed
	}

	// check again,before sending message, to avoid sending to closed channel
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-mb.done:
		return ErrBusClosed
	default:
	}

	mb.wg.Add(1)
	defer mb.wg.Done()

	select {
	case ch <- msg:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-mb.done:
		return ErrBusClosed
	}
}

func (mb *MessageBus) PublishInbound(ctx context.Context, msg InboundMessage) error {
	msg = NormalizeInboundMessage(msg)
	if msg.Context.isZero() {
		return ErrMissingInboundContext
	}
	return publish(ctx, mb, mb.inbound, msg)
}

func (mb *MessageBus) InboundChan() <-chan InboundMessage {
	return mb.inbound
}

func (mb *MessageBus) PublishOutbound(ctx context.Context, msg OutboundMessage) error {
	msg = NormalizeOutboundMessage(msg)
	if msg.Context.isZero() {
		return ErrMissingOutboundContext
	}
	return publish(ctx, mb, mb.outbound, msg)
}

func (mb *MessageBus) OutboundChan() <-chan OutboundMessage {
	return mb.outbound
}

func (mb *MessageBus) PublishOutboundMedia(ctx context.Context, msg OutboundMediaMessage) error {
	msg = NormalizeOutboundMediaMessage(msg)
	if msg.Context.isZero() {
		return ErrMissingOutboundMediaContext
	}
	return publish(ctx, mb, mb.outboundMedia, msg)
}

func (mb *MessageBus) OutboundMediaChan() <-chan OutboundMediaMessage {
	return mb.outboundMedia
}

func (mb *MessageBus) PublishAudioChunk(ctx context.Context, chunk AudioChunk) error {
	return publish(ctx, mb, mb.audioChunks, chunk)
}

func (mb *MessageBus) AudioChunksChan() <-chan AudioChunk {
	return mb.audioChunks
}

func (mb *MessageBus) PublishVoiceControl(ctx context.Context, ctrl VoiceControl) error {
	return publish(ctx, mb, mb.voiceControls, ctrl)
}

func (mb *MessageBus) VoiceControlsChan() <-chan VoiceControl {
	return mb.voiceControls
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
	mb.closeOnce.Do(func() {
		// notify all blocked publishers to exit
		close(mb.done)

		// because every publisher will check mb.closed before acquiring wg
		// so we can be sure that new publishers will not be added new messages after this point
		mb.closed.Store(true)

		// wait for all ongoing Publish calls to finish, ensuring all messages have been sent to channels or exited
		mb.wg.Wait()

		// close channels safely
		close(mb.inbound)
		close(mb.outbound)
		close(mb.outboundMedia)
		close(mb.audioChunks)
		close(mb.voiceControls)

		// clean up any remaining messages in channels
		drained := 0
		for range mb.inbound {
			drained++
		}
		for range mb.outbound {
			drained++
		}
		for range mb.outboundMedia {
			drained++
		}
		for range mb.audioChunks {
			drained++
		}
		for range mb.voiceControls {
			drained++
		}

		if drained > 0 {
			logger.DebugCF("bus", "Drained buffered messages during close", map[string]any{
				"count": drained,
			})
		}
	})
}
