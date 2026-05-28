package events

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestSubscribeOnceClosesAfterFirstEvent(t *testing.T) {
	t.Parallel()

	bus := NewBus()
	defer closeBus(t, bus)

	var handled atomic.Uint64
	sub, err := bus.Channel().SubscribeOnce(
		context.Background(),
		SubscribeOptions{Name: "once", Buffer: 2},
		func(context.Context, Event) error {
			handled.Add(1)
			return nil
		},
	)
	if err != nil {
		t.Fatalf("SubscribeOnce failed: %v", err)
	}

	bus.Publish(context.Background(), Event{Kind: KindAgentTurnStart})
	waitForSubscriptionDone(t, sub)
	bus.Publish(context.Background(), Event{Kind: KindAgentTurnEnd})

	if got := handled.Load(); got != 1 {
		t.Fatalf("handled = %d, want 1", got)
	}
	if got := sub.Stats().Handled; got != 1 {
		t.Fatalf("subscription handled = %d, want 1", got)
	}
}

func TestUnsubscribeClosesChannel(t *testing.T) {
	t.Parallel()

	bus := NewBus()
	defer closeBus(t, bus)

	sub, ch, err := bus.Channel().SubscribeChan(context.Background(), SubscribeOptions{Name: "chan"})
	if err != nil {
		t.Fatalf("SubscribeChan failed: %v", err)
	}
	if err := sub.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("channel is open, want closed")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for channel close")
	}
	waitForSubscriptionDone(t, sub)
}

func TestBlockBackpressureCloseUnblocksPublisher(t *testing.T) {
	t.Parallel()

	bus := NewBus()
	defer closeBus(t, bus)

	sub, _, err := bus.Channel().SubscribeChan(context.Background(), SubscribeOptions{
		Name:         "block-close",
		Buffer:       1,
		Backpressure: Block,
	})
	if err != nil {
		t.Fatalf("SubscribeChan failed: %v", err)
	}

	first := bus.Publish(context.Background(), Event{Kind: Kind("test.first")})
	if first.Delivered != 1 {
		t.Fatalf("first Publish = %+v, want one delivered event", first)
	}

	publishStarted := make(chan struct{})
	publishReturned := make(chan PublishResult, 1)
	go func() {
		close(publishStarted)
		publishReturned <- bus.Publish(context.Background(), Event{Kind: Kind("test.second")})
	}()

	<-publishStarted
	waitForStat(t, func() uint64 {
		return sub.Stats().Received
	}, 2)
	select {
	case result := <-publishReturned:
		t.Fatalf("blocking Publish returned before close: %+v", result)
	default:
	}

	closeReturned := make(chan error, 1)
	go func() {
		closeReturned <- sub.Close()
	}()

	select {
	case err := <-closeReturned:
		if err != nil {
			t.Fatalf("Close failed: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Close to unblock")
	}

	select {
	case <-publishReturned:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for blocking Publish to return after close")
	}
	waitForSubscriptionDone(t, sub)
}

func TestHandlerPanicRecovered(t *testing.T) {
	t.Parallel()

	bus := NewBus()
	defer closeBus(t, bus)

	sub, err := bus.Channel().Subscribe(
		context.Background(),
		SubscribeOptions{Name: "panic", Buffer: 1},
		func(context.Context, Event) error {
			panic("boom")
		},
	)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	bus.Publish(context.Background(), Event{Kind: KindAgentError})
	waitForStat(t, func() uint64 {
		return sub.Stats().Panicked
	}, 1)
}

func TestLockedHandlerProcessesSequentially(t *testing.T) {
	t.Parallel()

	bus := NewBus()
	defer closeBus(t, bus)

	var active atomic.Int64
	var maxActive atomic.Int64
	sub, err := bus.Channel().Subscribe(
		context.Background(),
		SubscribeOptions{Name: "locked", Buffer: 8, Concurrency: Locked},
		func(context.Context, Event) error {
			current := active.Add(1)
			for {
				currentMax := maxActive.Load()
				if current <= currentMax || maxActive.CompareAndSwap(currentMax, current) {
					break
				}
			}
			time.Sleep(10 * time.Millisecond)
			active.Add(-1)
			return nil
		},
	)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	for i := 0; i < 5; i++ {
		bus.Publish(context.Background(), Event{Kind: KindAgentLLMDelta})
	}
	waitForStat(t, func() uint64 {
		return sub.Stats().Handled
	}, 5)

	if got := maxActive.Load(); got != 1 {
		t.Fatalf("max active handlers = %d, want 1", got)
	}
}

func TestHandlerTimeoutDoesNotWedgeLockedSubscription(t *testing.T) {
	t.Parallel()

	bus := NewBus()
	defer closeBus(t, bus)

	releaseFirst := make(chan struct{})
	defer close(releaseFirst)

	var calls atomic.Uint64
	sub, err := bus.Channel().Subscribe(
		context.Background(),
		SubscribeOptions{Name: "timeout", Buffer: 2, Concurrency: Locked, Timeout: 20 * time.Millisecond},
		func(context.Context, Event) error {
			if calls.Add(1) == 1 {
				<-releaseFirst
			}
			return nil
		},
	)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	bus.Publish(context.Background(), Event{Kind: Kind("test.first")})
	waitForStat(t, func() uint64 {
		return sub.Stats().TimedOut
	}, 1)

	bus.Publish(context.Background(), Event{Kind: Kind("test.second")})
	waitForStat(t, func() uint64 {
		return sub.Stats().Handled
	}, 1)

	if got := sub.Stats().Failed; got != 1 {
		t.Fatalf("subscription failed = %d, want timeout failure", got)
	}
}

func waitForSubscriptionDone(t *testing.T, sub Subscription) {
	t.Helper()

	select {
	case <-sub.Done():
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for subscription to stop")
	}
}

func waitForStat(t *testing.T, stat func() uint64, want uint64) {
	t.Helper()

	deadline := time.After(time.Second)
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()

	for {
		if got := stat(); got >= want {
			return
		}
		select {
		case <-ticker.C:
		case <-deadline:
			t.Fatalf("timed out waiting for stat >= %d", want)
		}
	}
}
