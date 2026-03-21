package qq

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestRateLimiter_Wait(t *testing.T) {
	limiter := newRateLimiter(50 * time.Millisecond)

	// First call should not wait
	start := time.Now()
	limiter.wait("chat-1")
	elapsed := time.Since(start)
	if elapsed > 10*time.Millisecond {
		t.Errorf("first wait should be immediate, got %v", elapsed)
	}

	// Second call to same chat should wait
	start = time.Now()
	limiter.wait("chat-1")
	elapsed = time.Since(start)
	if elapsed < 40*time.Millisecond {
		t.Errorf("second wait should be at least 40ms, got %v", elapsed)
	}

	// Different chat should not wait
	start = time.Now()
	limiter.wait("chat-2")
	elapsed = time.Since(start)
	if elapsed > 10*time.Millisecond {
		t.Errorf("different chat should not wait, got %v", elapsed)
	}
}

func TestRateLimiter_Clear(t *testing.T) {
	limiter := newRateLimiter(100 * time.Millisecond)

	limiter.wait("chat-1")

	limiter.clear("chat-1")

	start := time.Now()
	limiter.wait("chat-1")
	elapsed := time.Since(start)
	if elapsed > 10*time.Millisecond {
		t.Errorf("after clear, wait should be immediate, got %v", elapsed)
	}
}

func TestRateLimiter_ClearAll(t *testing.T) {
	limiter := newRateLimiter(100 * time.Millisecond)

	limiter.wait("chat-1")
	limiter.wait("chat-2")

	limiter.clearAll()

	start := time.Now()
	limiter.wait("chat-1")
	elapsed := time.Since(start)
	if elapsed > 10*time.Millisecond {
		t.Errorf("after clearAll, wait should be immediate, got %v", elapsed)
	}
}

func TestRateLimiter_WaitWithContext_Canceled(t *testing.T) {
	limiter := newRateLimiter(100 * time.Millisecond)

	// First call to record a send time
	limiter.wait("chat-1")

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	cancel()

	start := time.Now()
	err := limiter.waitWithContext("chat-1", ctx)
	elapsed := time.Since(start)

	if err == nil {
		t.Error("expected context cancellation error")
	}
	if elapsed >= 50*time.Millisecond {
		t.Errorf("should have returned early due to cancellation, got %v", elapsed)
	}
}

func TestRateLimiter_WaitWithContext_Success(t *testing.T) {
	limiter := newRateLimiter(50 * time.Millisecond)

	ctx := context.Background()

	start := time.Now()
	err := limiter.waitWithContext("chat-1", ctx)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if elapsed > 10*time.Millisecond {
		t.Errorf("first wait should be immediate, got %v", elapsed)
	}
}

func TestRateLimiter_Concurrent(t *testing.T) {
	limiter := newRateLimiter(10 * time.Millisecond)

	var wg sync.WaitGroup
	chatIDs := []string{"chat-1", "chat-2", "chat-3"}

	// Run 10 iterations for each chat concurrently
	for i := 0; i < 10; i++ {
		for _, chatID := range chatIDs {
			wg.Add(1)
			go func(id string) {
				defer wg.Done()
				limiter.wait(id)
			}(chatID)
		}
	}

	// All goroutines should complete within reasonable time
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Error("concurrent rate limiter test timed out")
	}
}
