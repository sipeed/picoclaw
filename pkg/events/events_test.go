package events

import (
	"context"
	"testing"
	"time"
)

func TestPublishDeliversToMatchingSubscriber(t *testing.T) {
	t.Parallel()

	bus := NewBus()
	defer closeBus(t, bus)

	_, ch, err := bus.Channel().OfKind(KindAgentTurnStart).SubscribeChan(
		context.Background(),
		SubscribeOptions{Name: "turn-starts", Buffer: 1},
	)
	if err != nil {
		t.Fatalf("SubscribeChan failed: %v", err)
	}

	unmatched := bus.Publish(context.Background(), Event{Kind: KindAgentTurnEnd})
	if unmatched.Matched != 0 || unmatched.Delivered != 0 {
		t.Fatalf("unmatched Publish = %+v, want no delivery", unmatched)
	}

	result := bus.Publish(context.Background(), Event{Kind: KindAgentTurnStart})
	if result.Matched != 1 || result.Delivered != 1 || result.Dropped != 0 {
		t.Fatalf("Publish = %+v, want one delivered event", result)
	}

	evt := receiveEvent(t, ch)
	if evt.Kind != KindAgentTurnStart {
		t.Fatalf("event kind = %q, want %q", evt.Kind, KindAgentTurnStart)
	}
	if evt.ID == "" {
		t.Fatal("event ID is empty")
	}
	if evt.Time.IsZero() {
		t.Fatal("event Time is zero")
	}
}

func TestDropNewestIncrementsStats(t *testing.T) {
	t.Parallel()

	bus := NewBus()
	defer closeBus(t, bus)

	sub, _, err := bus.Channel().SubscribeChan(
		context.Background(),
		SubscribeOptions{Name: "drop-newest", Buffer: 1, Backpressure: DropNewest},
	)
	if err != nil {
		t.Fatalf("SubscribeChan failed: %v", err)
	}

	first := bus.Publish(context.Background(), Event{Kind: KindAgentTurnStart})
	if first.Delivered != 1 || first.Dropped != 0 {
		t.Fatalf("first Publish = %+v, want one delivered event", first)
	}

	second := bus.Publish(context.Background(), Event{Kind: KindAgentTurnEnd})
	if second.Delivered != 0 || second.Dropped != 1 {
		t.Fatalf("second Publish = %+v, want one dropped event", second)
	}

	if got := sub.Stats().Dropped; got != 1 {
		t.Fatalf("subscription dropped = %d, want 1", got)
	}
	if got := bus.Stats().Dropped; got != 1 {
		t.Fatalf("bus dropped = %d, want 1", got)
	}
}

func TestDropOldestKeepsNewestEvent(t *testing.T) {
	t.Parallel()

	bus := NewBus()
	defer closeBus(t, bus)

	sub, ch, err := bus.Channel().SubscribeChan(
		context.Background(),
		SubscribeOptions{Name: "drop-oldest", Buffer: 1, Backpressure: DropOldest},
	)
	if err != nil {
		t.Fatalf("SubscribeChan failed: %v", err)
	}

	bus.Publish(context.Background(), Event{Kind: Kind("test.old"), Payload: "old"})
	result := bus.Publish(context.Background(), Event{Kind: Kind("test.new"), Payload: "new"})
	if result.Delivered != 1 || result.Dropped != 1 {
		t.Fatalf("Publish = %+v, want replacement delivery", result)
	}

	evt := receiveEvent(t, ch)
	if evt.Payload != "new" {
		t.Fatalf("payload = %v, want new", evt.Payload)
	}
	if got := sub.Stats().Dropped; got != 1 {
		t.Fatalf("subscription dropped = %d, want 1", got)
	}
}

func TestBlockRespectsContext(t *testing.T) {
	t.Parallel()

	bus := NewBus()
	defer closeBus(t, bus)

	_, _, err := bus.Channel().SubscribeChan(
		context.Background(),
		SubscribeOptions{Name: "block", Buffer: 1, Backpressure: Block},
	)
	if err != nil {
		t.Fatalf("SubscribeChan failed: %v", err)
	}

	first := bus.Publish(context.Background(), Event{Kind: Kind("test.first")})
	if first.Delivered != 1 {
		t.Fatalf("first Publish = %+v, want one delivered event", first)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	second := bus.Publish(ctx, Event{Kind: Kind("test.second")})
	if second.Blocked != 1 || second.Dropped != 1 || second.Delivered != 0 {
		t.Fatalf("second Publish = %+v, want one blocked drop", second)
	}
}

func TestPublishNonBlockingDropsForFullBlockSubscriber(t *testing.T) {
	t.Parallel()

	bus := NewBus()
	defer closeBus(t, bus)

	sub, _, err := bus.Channel().SubscribeChan(
		context.Background(),
		SubscribeOptions{Name: "block", Buffer: 1, Backpressure: Block},
	)
	if err != nil {
		t.Fatalf("SubscribeChan failed: %v", err)
	}

	first := bus.PublishNonBlocking(Event{Kind: Kind("test.first")})
	if first.Delivered != 1 {
		t.Fatalf("first PublishNonBlocking = %+v, want one delivered event", first)
	}

	resultCh := make(chan PublishResult, 1)
	go func() {
		resultCh <- bus.PublishNonBlocking(Event{Kind: Kind("test.second")})
	}()

	select {
	case second := <-resultCh:
		if second.Matched != 1 || second.Delivered != 0 || second.Dropped != 1 || second.Blocked != 0 {
			t.Fatalf("second PublishNonBlocking = %+v, want non-blocking drop", second)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("PublishNonBlocking blocked on full Block subscriber")
	}

	if got := sub.Stats().Dropped; got != 1 {
		t.Fatalf("subscription dropped = %d, want 1", got)
	}
}

func TestStatsSubscribersKeepPriorityOrder(t *testing.T) {
	t.Parallel()

	bus := NewBus()
	defer closeBus(t, bus)

	low, _, err := bus.Channel().SubscribeChan(
		context.Background(),
		SubscribeOptions{Name: "low", Priority: -1},
	)
	if err != nil {
		t.Fatalf("SubscribeChan low failed: %v", err)
	}
	high, _, err := bus.Channel().SubscribeChan(
		context.Background(),
		SubscribeOptions{Name: "high", Priority: 10},
	)
	if err != nil {
		t.Fatalf("SubscribeChan high failed: %v", err)
	}
	peer, _, err := bus.Channel().SubscribeChan(
		context.Background(),
		SubscribeOptions{Name: "peer", Priority: 10},
	)
	if err != nil {
		t.Fatalf("SubscribeChan peer failed: %v", err)
	}

	stats := bus.Stats()
	got := []string{
		stats.SubscriberStats[0].Name,
		stats.SubscriberStats[1].Name,
		stats.SubscriberStats[2].Name,
	}
	want := []string{"high", "peer", "low"}
	if got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("subscriber order = %v, want %v", got, want)
	}

	if err := high.Close(); err != nil {
		t.Fatalf("Close high failed: %v", err)
	}

	stats = bus.Stats()
	got = []string{
		stats.SubscriberStats[0].Name,
		stats.SubscriberStats[1].Name,
	}
	want = []string{"peer", "low"}
	if got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("subscriber order after unsubscribe = %v, want %v", got, want)
	}

	if err := peer.Close(); err != nil {
		t.Fatalf("Close peer failed: %v", err)
	}
	if err := low.Close(); err != nil {
		t.Fatalf("Close low failed: %v", err)
	}
}

func receiveEvent(t *testing.T, ch <-chan Event) Event {
	t.Helper()

	select {
	case evt, ok := <-ch:
		if !ok {
			t.Fatal("event channel closed before receive")
		}
		return evt
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
		return Event{}
	}
}

func closeBus(t *testing.T, bus *EventBus) {
	t.Helper()

	if err := bus.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}
