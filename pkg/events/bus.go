package events

import (
	"context"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

var globalEventSeq atomic.Uint64

// Bus publishes runtime events and creates filtered channels.
type Bus interface {
	Publish(ctx context.Context, evt Event) PublishResult
	PublishNonBlocking(evt Event) PublishResult
	Channel() EventChannel
	Close() error
	Stats() Stats
}

// PublishResult reports per-publish delivery outcomes.
type PublishResult struct {
	Matched   int
	Delivered int
	Dropped   int
	Blocked   int
	Closed    bool
}

// EventBus is an in-process runtime event broadcaster.
type EventBus struct {
	mu          sync.RWMutex
	subs        map[uint64]*eventSubscription
	orderedSubs []*eventSubscription
	closed      bool

	nextSubID atomic.Uint64
	published atomic.Uint64
	matched   atomic.Uint64
	delivered atomic.Uint64
	dropped   atomic.Uint64
	blocked   atomic.Uint64
}

var _ Bus = (*EventBus)(nil)

// NewBus creates an in-process runtime event bus.
func NewBus() *EventBus {
	return &EventBus{
		subs: make(map[uint64]*eventSubscription),
	}
}

// Publish broadcasts evt to subscriptions whose filters match it.
func (b *EventBus) Publish(ctx context.Context, evt Event) PublishResult {
	return b.publish(ctx, evt, false)
}

// PublishNonBlocking broadcasts evt without waiting for subscriber queue capacity.
func (b *EventBus) PublishNonBlocking(evt Event) PublishResult {
	return b.publish(context.Background(), evt, true)
}

func (b *EventBus) publish(ctx context.Context, evt Event, nonBlocking bool) PublishResult {
	if b == nil {
		return PublishResult{Closed: true}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if evt.Time.IsZero() {
		evt.Time = time.Now()
	}
	if evt.ID == "" {
		evt.ID = nextEventID()
	}

	subs, closed := b.snapshotSubscribers()
	if closed {
		return PublishResult{Closed: true}
	}

	b.published.Add(1)
	result := PublishResult{}

	for _, sub := range subs {
		if !matchesFilters(sub.filters, evt) {
			continue
		}

		result.Matched++
		b.matched.Add(1)

		delivery := sub.enqueue(ctx, evt, nonBlocking)
		if delivery.closed {
			continue
		}
		result.Delivered += delivery.delivered
		result.Dropped += delivery.dropped
		result.Blocked += delivery.blocked
		b.delivered.Add(uint64(delivery.delivered))
		b.dropped.Add(uint64(delivery.dropped))
		b.blocked.Add(uint64(delivery.blocked))
	}

	return result
}

// Channel returns the root event channel for this bus.
func (b *EventBus) Channel() EventChannel {
	return eventChannel{bus: b}
}

// Close closes the bus and all active subscriptions.
func (b *EventBus) Close() error {
	if b == nil {
		return nil
	}

	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true
	subs := b.orderedSubs
	b.subs = nil
	b.orderedSubs = nil
	b.mu.Unlock()

	for _, sub := range subs {
		sub.closeInput()
	}
	return nil
}

// Stats returns a snapshot of bus and subscription counters.
func (b *EventBus) Stats() Stats {
	if b == nil {
		return Stats{Closed: true}
	}

	b.mu.RLock()
	closed := b.closed
	subs := b.orderedSubs
	b.mu.RUnlock()

	stats := Stats{
		Published:       b.published.Load(),
		Matched:         b.matched.Load(),
		Delivered:       b.delivered.Load(),
		Dropped:         b.dropped.Load(),
		Blocked:         b.blocked.Load(),
		Closed:          closed,
		Subscribers:     len(subs),
		SubscriberStats: make([]SubscriberStats, 0, len(subs)),
	}
	for _, sub := range subs {
		stats.SubscriberStats = append(stats.SubscriberStats, sub.Stats())
	}
	return stats
}

func (b *EventBus) subscribe(
	ctx context.Context,
	filters []Filter,
	opts SubscribeOptions,
	handler Handler,
	once bool,
) (Subscription, error) {
	if b == nil {
		return nil, ErrBusClosed
	}

	id := b.nextSubID.Add(1)
	sub := newSubscription(b, id, filters, opts, handler, once)

	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		sub.closeInput()
		return nil, ErrBusClosed
	}
	b.subs[id] = sub
	b.rebuildOrderedSubscribersLocked()
	b.mu.Unlock()

	if handler != nil {
		go sub.run(ctx)
	}
	sub.watchContext(ctx)
	return sub, nil
}

func (b *EventBus) unsubscribe(id uint64) {
	b.mu.Lock()
	sub, ok := b.subs[id]
	if ok {
		delete(b.subs, id)
		b.rebuildOrderedSubscribersLocked()
	}
	b.mu.Unlock()

	if ok {
		sub.closeInput()
	}
}

func (b *EventBus) snapshotSubscribers() ([]*eventSubscription, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return nil, true
	}

	return b.orderedSubs, false
}

func (b *EventBus) rebuildOrderedSubscribersLocked() {
	subs := make([]*eventSubscription, 0, len(b.subs))
	for _, sub := range b.subs {
		subs = append(subs, sub)
	}
	sortSubscriptions(subs)
	b.orderedSubs = subs
}

func sortSubscriptions(subs []*eventSubscription) {
	sort.Slice(subs, func(i, j int) bool {
		if subs[i].opts.Priority == subs[j].opts.Priority {
			return subs[i].id < subs[j].id
		}
		return subs[i].opts.Priority > subs[j].opts.Priority
	})
}

func nextEventID() string {
	id := globalEventSeq.Add(1)
	return "evt-" + strconv.FormatUint(id, 10)
}
