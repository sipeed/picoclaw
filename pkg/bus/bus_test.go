package bus

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestMessageBus_InterceptorConsumesMessage(t *testing.T) {
	mb := NewMessageBus()

	consumed := make(chan bool, 1)
	mb.AddInterceptor(func(msg InboundMessage) bool {
		if msg.Content == "intercept-me" {
			consumed <- true
			return true
		}
		return false
	})

	mb.PublishInbound(InboundMessage{Content: "intercept-me"})

	select {
	case <-consumed:
		// ok
	case <-time.After(time.Second):
		t.Fatal("interceptor did not consume the message")
	}

	// Message should not have reached the main queue
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, ok := mb.ConsumeInbound(ctx)
	if ok {
		t.Error("intercepted message should not reach main consumer")
	}
}

func TestMessageBus_InterceptorPassesThrough(t *testing.T) {
	mb := NewMessageBus()

	mb.AddInterceptor(func(msg InboundMessage) bool {
		return false // never consume
	})

	mb.PublishInbound(InboundMessage{Content: "pass-through"})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	msg, ok := mb.ConsumeInbound(ctx)
	if !ok {
		t.Fatal("message should have passed through to main consumer")
	}
	if msg.Content != "pass-through" {
		t.Errorf("expected 'pass-through', got %q", msg.Content)
	}
}

func TestMessageBus_InterceptorRemoval(t *testing.T) {
	mb := NewMessageBus()
	count := 0

	remove := mb.AddInterceptor(func(msg InboundMessage) bool {
		count++
		return true
	})

	mb.PublishInbound(InboundMessage{Content: "first"})
	if count != 1 {
		t.Fatalf("expected count=1, got %d", count)
	}

	remove() // unregister

	mb.PublishInbound(InboundMessage{Content: "second"})

	// Message should now reach main queue
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	msg, ok := mb.ConsumeInbound(ctx)
	if !ok || msg.Content != "second" {
		t.Error("after removal, message should reach main consumer")
	}
	if count != 1 {
		t.Errorf("interceptor should not have been called after removal, count=%d", count)
	}
}

func TestMessageBus_MultipleInterceptors(t *testing.T) {
	mb := NewMessageBus()
	order := []string{}
	var mu sync.Mutex

	mb.AddInterceptor(func(msg InboundMessage) bool {
		mu.Lock()
		order = append(order, "first")
		mu.Unlock()
		return false
	})

	mb.AddInterceptor(func(msg InboundMessage) bool {
		mu.Lock()
		order = append(order, "second")
		mu.Unlock()
		return msg.Content == "stop-at-second"
	})

	// This message passes through both interceptors
	mb.PublishInbound(InboundMessage{Content: "pass"})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, ok := mb.ConsumeInbound(ctx)
	if !ok {
		t.Error("message should reach main consumer")
	}

	mu.Lock()
	if len(order) != 2 || order[0] != "first" || order[1] != "second" {
		t.Errorf("expected [first, second], got %v", order)
	}
	order = nil
	mu.Unlock()

	// This message is consumed by second interceptor
	mb.PublishInbound(InboundMessage{Content: "stop-at-second"})

	ctx2, cancel2 := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel2()
	_, ok = mb.ConsumeInbound(ctx2)
	if ok {
		t.Error("message consumed by second interceptor should not reach main consumer")
	}
}

func TestMessageBus_InterceptorConcurrency(t *testing.T) {
	mb := NewMessageBus()
	var wg sync.WaitGroup
	intercepted := int32(0)

	remove := mb.AddInterceptor(func(msg InboundMessage) bool {
		if msg.Content == "catch" {
			return true
		}
		return false
	})

	// Publish many messages concurrently
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				mb.PublishInbound(InboundMessage{Content: "catch"})
			} else {
				mb.PublishInbound(InboundMessage{Content: "pass"})
			}
		}(i)
	}

	wg.Wait()
	remove()

	// Drain remaining messages
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		msg, ok := mb.ConsumeInbound(ctx)
		cancel()
		if !ok {
			break
		}
		if msg.Content == "catch" {
			t.Error("intercepted message should not reach consumer")
		}
		intercepted++
	}
	_ = intercepted // just ensure no panics
}
