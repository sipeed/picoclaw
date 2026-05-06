package events

import "context"

// EventChannel is a filtered view over an EventBus.
type EventChannel interface {
	Filter(filter Filter) EventChannel
	OfKind(kinds ...Kind) EventChannel
	KindPrefix(prefix string) EventChannel
	Source(component string, names ...string) EventChannel
	Scope(scope ScopeFilter) EventChannel

	Subscribe(ctx context.Context, opts SubscribeOptions, handler Handler) (Subscription, error)
	SubscribeChan(ctx context.Context, opts SubscribeOptions) (Subscription, <-chan Event, error)
	SubscribeOnce(ctx context.Context, opts SubscribeOptions, handler Handler) (Subscription, error)
}

type eventChannel struct {
	bus     *EventBus
	filters []Filter
}

// Filter returns a new EventChannel with filter appended.
func (c eventChannel) Filter(filter Filter) EventChannel {
	filters := append([]Filter(nil), c.filters...)
	if filter != nil {
		filters = append(filters, filter)
	}
	return eventChannel{bus: c.bus, filters: filters}
}

// OfKind returns a new EventChannel matching any of kinds.
func (c eventChannel) OfKind(kinds ...Kind) EventChannel {
	return c.Filter(MatchKind(kinds...))
}

// KindPrefix returns a new EventChannel matching events with the kind prefix.
func (c eventChannel) KindPrefix(prefix string) EventChannel {
	return c.Filter(MatchKindPrefix(prefix))
}

// Source returns a new EventChannel matching source component and optional names.
func (c eventChannel) Source(component string, names ...string) EventChannel {
	return c.Filter(MatchSource(component, names...))
}

// Scope returns a new EventChannel matching non-empty scope fields.
func (c eventChannel) Scope(scope ScopeFilter) EventChannel {
	return c.Filter(MatchScope(scope))
}

// Subscribe registers handler for events matching this channel.
func (c eventChannel) Subscribe(ctx context.Context, opts SubscribeOptions, handler Handler) (Subscription, error) {
	if handler == nil {
		return nil, ErrNilHandler
	}
	return c.bus.subscribe(ctx, c.filters, opts, handler, false)
}

// SubscribeChan registers a channel subscription for events matching this channel.
func (c eventChannel) SubscribeChan(ctx context.Context, opts SubscribeOptions) (Subscription, <-chan Event, error) {
	sub, err := c.bus.subscribe(ctx, c.filters, opts, nil, false)
	if err != nil {
		return nil, nil, err
	}
	return sub, sub.(*eventSubscription).ch, nil
}

// SubscribeOnce registers handler and closes the subscription after the first event.
func (c eventChannel) SubscribeOnce(ctx context.Context, opts SubscribeOptions, handler Handler) (Subscription, error) {
	if handler == nil {
		return nil, ErrNilHandler
	}
	return c.bus.subscribe(ctx, c.filters, opts, handler, true)
}
