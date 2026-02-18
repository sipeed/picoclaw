package bus

import (
	"context"
	"sync"
	"sync/atomic"
)

type interceptorEntry struct {
	id uint64
	fn InboundInterceptor
}

type MessageBus struct {
	inbound      chan InboundMessage
	outbound     chan OutboundMessage
	handlers     map[string]MessageHandler
	interceptors []*interceptorEntry
	nextID       uint64
	closed       bool
	mu           sync.RWMutex
}

func NewMessageBus() *MessageBus {
	return &MessageBus{
		inbound:  make(chan InboundMessage, 100),
		outbound: make(chan OutboundMessage, 100),
		handlers: make(map[string]MessageHandler),
	}
}

// AddInterceptor registers an interceptor that inspects inbound messages before
// they reach the main consumer queue. Returns a removal function.
func (mb *MessageBus) AddInterceptor(fn InboundInterceptor) func() {
	id := atomic.AddUint64(&mb.nextID, 1)
	entry := &interceptorEntry{id: id, fn: fn}

	mb.mu.Lock()
	mb.interceptors = append(mb.interceptors, entry)
	mb.mu.Unlock()

	return func() {
		mb.mu.Lock()
		defer mb.mu.Unlock()
		for i, e := range mb.interceptors {
			if e.id == id {
				mb.interceptors = append(mb.interceptors[:i], mb.interceptors[i+1:]...)
				break
			}
		}
	}
}

func (mb *MessageBus) PublishInbound(msg InboundMessage) {
	mb.mu.RLock()
	if mb.closed {
		mb.mu.RUnlock()
		return
	}
	interceptors := make([]*interceptorEntry, len(mb.interceptors))
	copy(interceptors, mb.interceptors)
	mb.mu.RUnlock()

	for _, entry := range interceptors {
		if entry.fn(msg) {
			return
		}
	}
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
	mb.mu.RLock()
	defer mb.mu.RUnlock()
	if mb.closed {
		return
	}
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
	mb.mu.Lock()
	defer mb.mu.Unlock()
	if mb.closed {
		return
	}
	mb.closed = true
	close(mb.inbound)
	close(mb.outbound)
}
