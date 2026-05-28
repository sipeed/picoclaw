package events

import (
	"context"
	"errors"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

const defaultSubscriberBuffer = 16

var (
	// ErrBusClosed is returned when subscribing to a closed event bus.
	ErrBusClosed = errors.New("events: bus is closed")
	// ErrNilHandler is returned when subscribing without a handler.
	ErrNilHandler = errors.New("events: handler is nil")
)

// Handler processes a runtime event delivered to a subscription.
type Handler func(context.Context, Event) error

// SubscribeOptions controls how a subscription receives events.
type SubscribeOptions struct {
	Name         string
	Buffer       int
	Priority     int
	Concurrency  ConcurrencyKind
	Backpressure BackpressurePolicy
	// Timeout bounds how long the subscription worker waits for one handler call.
	// Handlers should still honor ctx cancellation; timed-out calls keep running
	// until their handler returns.
	Timeout     time.Duration
	PanicPolicy PanicPolicy
}

// ConcurrencyKind controls how handler subscriptions process queued events.
type ConcurrencyKind string

const (
	// Concurrent processes each event in its own goroutine.
	Concurrent ConcurrencyKind = "concurrent"
	// Locked processes events sequentially in subscription order.
	Locked ConcurrencyKind = "locked"
	// Keyed is reserved for keyed sequential processing and currently behaves as Locked.
	Keyed ConcurrencyKind = "keyed"
)

// BackpressurePolicy controls delivery when a subscription queue is full.
type BackpressurePolicy string

const (
	// DropNewest drops the event being published when the queue is full.
	DropNewest BackpressurePolicy = "drop_newest"
	// DropOldest drops one queued event and enqueues the event being published.
	DropOldest BackpressurePolicy = "drop_oldest"
	// Block waits for queue capacity until Publish's context is canceled.
	Block BackpressurePolicy = "block"
)

// PanicPolicy controls handler panic behavior.
type PanicPolicy string

const (
	// RecoverAndLog recovers handler panics and records them in subscription stats.
	RecoverAndLog PanicPolicy = "recover_and_log"
	// Crash lets handler panics propagate from the worker goroutine.
	Crash PanicPolicy = "crash"
)

// Subscription represents an active event subscription.
type Subscription interface {
	ID() uint64
	Name() string
	Close() error
	Done() <-chan struct{}
	Stats() SubscriberStats
}

type subscriberCounters struct {
	received atomic.Uint64
	handled  atomic.Uint64
	failed   atomic.Uint64
	dropped  atomic.Uint64
	panicked atomic.Uint64
	timedOut atomic.Uint64
}

type eventSubscription struct {
	bus     *EventBus
	id      uint64
	name    string
	opts    SubscribeOptions
	filters []Filter
	handler Handler
	once    bool

	ch      chan Event
	done    chan struct{}
	closing chan struct{}

	closeOnce sync.Once
	doneOnce  sync.Once
	mu        sync.RWMutex
	closed    bool
	wg        sync.WaitGroup
	blockWG   sync.WaitGroup

	counters subscriberCounters
}

type handlerResult struct {
	err      error
	panicked bool
}

func normalizeSubscribeOptions(opts SubscribeOptions) SubscribeOptions {
	if opts.Buffer <= 0 {
		opts.Buffer = defaultSubscriberBuffer
	}
	if opts.Concurrency == "" {
		opts.Concurrency = Locked
	}
	if opts.Backpressure == "" {
		opts.Backpressure = DropNewest
	}
	if opts.PanicPolicy == "" {
		opts.PanicPolicy = RecoverAndLog
	}
	return opts
}

func newSubscription(
	bus *EventBus,
	id uint64,
	filters []Filter,
	opts SubscribeOptions,
	handler Handler,
	once bool,
) *eventSubscription {
	opts = normalizeSubscribeOptions(opts)
	return &eventSubscription{
		bus:     bus,
		id:      id,
		name:    opts.Name,
		opts:    opts,
		filters: append([]Filter(nil), filters...),
		handler: handler,
		once:    once,
		ch:      make(chan Event, opts.Buffer),
		done:    make(chan struct{}),
		closing: make(chan struct{}),
	}
}

// ID returns the subscription identifier.
func (s *eventSubscription) ID() uint64 {
	if s == nil {
		return 0
	}
	return s.id
}

// Name returns the subscription name.
func (s *eventSubscription) Name() string {
	if s == nil {
		return ""
	}
	return s.name
}

// Close removes the subscription and closes its delivery channel.
func (s *eventSubscription) Close() error {
	if s == nil || s.bus == nil {
		return nil
	}
	s.bus.unsubscribe(s.id)
	return nil
}

// Done returns a channel closed after the subscription has stopped processing.
func (s *eventSubscription) Done() <-chan struct{} {
	if s == nil {
		ch := make(chan struct{})
		close(ch)
		return ch
	}
	return s.done
}

// Stats returns a snapshot of the subscription counters.
func (s *eventSubscription) Stats() SubscriberStats {
	if s == nil {
		return SubscriberStats{}
	}
	return SubscriberStats{
		ID:       s.id,
		Name:     s.name,
		Received: s.counters.received.Load(),
		Handled:  s.counters.handled.Load(),
		Failed:   s.counters.failed.Load(),
		Dropped:  s.counters.dropped.Load(),
		Panicked: s.counters.panicked.Load(),
		TimedOut: s.counters.timedOut.Load(),
	}
}

func (s *eventSubscription) run(ctx context.Context) {
	defer func() {
		s.wg.Wait()
		s.closeDone()
	}()

	for evt := range s.ch {
		s.dispatch(ctx, evt)
		if s.once {
			_ = s.Close()
			return
		}
	}
}

func (s *eventSubscription) dispatch(ctx context.Context, evt Event) {
	switch s.opts.Concurrency {
	case Concurrent:
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.handle(ctx, evt)
		}()
	case Keyed:
		// TODO: replace this with keyed executors when runtime events need
		// per-scope ordering with cross-scope concurrency.
		s.handle(ctx, evt)
	default:
		s.handle(ctx, evt)
	}
}

func (s *eventSubscription) handle(ctx context.Context, evt Event) {
	if ctx == nil {
		ctx = context.Background()
	}

	if s.opts.Timeout <= 0 {
		s.recordHandlerResult(ctx, s.invokeHandler(ctx, evt))
		return
	}

	ctx, cancel := context.WithTimeout(ctx, s.opts.Timeout)
	defer cancel()

	done := make(chan handlerResult, 1)
	go func() {
		done <- s.invokeHandler(ctx, evt)
	}()

	select {
	case result := <-done:
		s.recordHandlerResult(ctx, result)
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			s.counters.timedOut.Add(1)
		}
		s.counters.failed.Add(1)
	}
}

func (s *eventSubscription) invokeHandler(ctx context.Context, evt Event) (result handlerResult) {
	if s.opts.PanicPolicy != Crash {
		defer func() {
			if recovered := recover(); recovered != nil {
				s.counters.panicked.Add(1)
				result.panicked = true
				log.Printf("events: subscriber %q recovered panic: %v", s.name, recovered)
			}
		}()
	}

	result.err = s.handler(ctx, evt)
	return result
}

func (s *eventSubscription) recordHandlerResult(ctx context.Context, result handlerResult) {
	if result.panicked {
		return
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		s.counters.timedOut.Add(1)
	}
	if result.err != nil {
		s.counters.failed.Add(1)
		return
	}
	s.counters.handled.Add(1)
}

func (s *eventSubscription) watchContext(ctx context.Context) {
	if ctx == nil {
		return
	}

	go func() {
		select {
		case <-ctx.Done():
			_ = s.Close()
		case <-s.done:
		}
	}()
}

func (s *eventSubscription) closeInput() {
	s.closeOnce.Do(func() {
		close(s.closing)
		s.mu.Lock()
		s.closed = true
		s.mu.Unlock()
		s.blockWG.Wait()
		s.mu.Lock()
		close(s.ch)
		s.mu.Unlock()
		if s.handler == nil {
			s.closeDone()
		}
	})
}

func (s *eventSubscription) closeDone() {
	s.doneOnce.Do(func() {
		close(s.done)
	})
}

type deliveryResult struct {
	delivered int
	dropped   int
	blocked   int
	closed    bool
}

func (s *eventSubscription) enqueue(ctx context.Context, evt Event, nonBlocking bool) deliveryResult {
	if ctx == nil {
		ctx = context.Background()
	}

	if nonBlocking {
		return s.enqueueNonBlocking(evt)
	}

	if s.opts.Backpressure == Block {
		return s.enqueueBlocking(ctx, evt)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return deliveryResult{closed: true}
	}

	s.counters.received.Add(1)

	switch s.opts.Backpressure {
	case DropOldest:
		return s.enqueueDropOldest(evt)
	default:
		return s.enqueueDropNewest(evt)
	}
}

func (s *eventSubscription) enqueueBlocking(ctx context.Context, evt Event) deliveryResult {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return deliveryResult{closed: true}
	}
	s.blockWG.Add(1)
	s.counters.received.Add(1)
	s.mu.Unlock()

	defer s.blockWG.Done()
	return s.enqueueBlock(ctx, evt)
}

func (s *eventSubscription) enqueueNonBlocking(evt Event) deliveryResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return deliveryResult{closed: true}
	}

	s.counters.received.Add(1)
	if s.opts.Backpressure == DropOldest {
		return s.enqueueDropOldest(evt)
	}
	return s.enqueueDropNewest(evt)
}

func (s *eventSubscription) enqueueDropNewest(evt Event) deliveryResult {
	select {
	case <-s.closing:
		return deliveryResult{closed: true}
	default:
	}

	select {
	case s.ch <- evt:
		return deliveryResult{delivered: 1}
	default:
		s.counters.dropped.Add(1)
		return deliveryResult{dropped: 1}
	}
}

func (s *eventSubscription) enqueueDropOldest(evt Event) deliveryResult {
	select {
	case <-s.closing:
		return deliveryResult{closed: true}
	default:
	}

	select {
	case s.ch <- evt:
		return deliveryResult{delivered: 1}
	default:
	}

	dropped := 0
	select {
	case <-s.ch:
		s.counters.dropped.Add(1)
		dropped = 1
	default:
	}

	select {
	case <-s.closing:
		return deliveryResult{dropped: dropped, closed: true}
	case s.ch <- evt:
		return deliveryResult{delivered: 1, dropped: dropped}
	default:
		s.counters.dropped.Add(1)
		return deliveryResult{dropped: dropped + 1}
	}
}

func (s *eventSubscription) enqueueBlock(ctx context.Context, evt Event) deliveryResult {
	select {
	case <-s.closing:
		return deliveryResult{closed: true}
	case s.ch <- evt:
		return deliveryResult{delivered: 1}
	case <-ctx.Done():
		s.counters.dropped.Add(1)
		return deliveryResult{dropped: 1, blocked: 1}
	}
}
