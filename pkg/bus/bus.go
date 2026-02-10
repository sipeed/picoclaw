package bus

import (
	"context"
	"sync"
	"time"
)

type MessageBus struct {
	inbound    chan InboundMessage
	outbound   chan OutboundMessage
	streamChan chan StreamEvent
	interrupt  chan InterruptSignal
	handlers   map[string]MessageHandler
	mu         sync.RWMutex
}

func NewMessageBus() *MessageBus {
	return &MessageBus{
		inbound:    make(chan InboundMessage, 100),
		outbound:   make(chan OutboundMessage, 100),
		streamChan: make(chan StreamEvent, 100),
		interrupt:  make(chan InterruptSignal, 10),
		handlers:   make(map[string]MessageHandler),
	}
}

func (mb *MessageBus) PublishInbound(msg InboundMessage) {
	mb.inbound <- msg
}

func (mb *MessageBus) ConsumeInbound(ctx context.Context) (InboundMessage, bool) {
	select {
	case msg := <-mb.inbound:
		return msg, true
	case <-ctx.Done():
		return InboundMessage{}, false
	}
}

func (mb *MessageBus) PublishOutbound(msg OutboundMessage) {
	mb.outbound <- msg
}

func (mb *MessageBus) SubscribeOutbound(ctx context.Context) (OutboundMessage, bool) {
	select {
	case msg := <-mb.outbound:
		return msg, true
	case <-ctx.Done():
		return OutboundMessage{}, false
	}
}

func (mb *MessageBus) RegisterHandler(channel string, handler MessageHandler) {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	mb.handlers[channel] = handler
}

func (mb *MessageBus) GetHandler(channel string) (MessageHandler, bool) {
	mb.mu.RLock()
	defer mb.mu.RUnlock()
	handler, ok := mb.handlers[channel]
	return handler, ok
}

func (mb *MessageBus) Close() {
	close(mb.inbound)
	close(mb.outbound)
	close(mb.streamChan)
	close(mb.interrupt)
}

// PublishStreamEvent sends a streaming event
func (mb *MessageBus) PublishStreamEvent(event StreamEvent) {
	if event.Timestamp == 0 {
		event.Timestamp = time.Now().Unix()
	}
	select {
	case mb.streamChan <- event:
	default:
		// Drop event if channel is full to avoid blocking
	}
}

// ConsumeStreamEvent receives streaming events
func (mb *MessageBus) ConsumeStreamEvent(ctx context.Context) (StreamEvent, bool) {
	select {
	case event := <-mb.streamChan:
		return event, true
	case <-ctx.Done():
		return StreamEvent{}, false
	}
}

// PublishInterrupt sends an interrupt signal
func (mb *MessageBus) PublishInterrupt(signal InterruptSignal) {
	select {
	case mb.interrupt <- signal:
	default:
		// Drop if channel is full
	}
}

// CheckInterrupt checks if there's an interrupt signal for the given session
func (mb *MessageBus) CheckInterrupt(sessionKey string) bool {
	select {
	case signal := <-mb.interrupt:
		// Put it back if it doesn't match
		if signal.SessionKey != sessionKey {
			mb.interrupt <- signal
			return false
		}
		return true
	default:
		return false
	}
}
