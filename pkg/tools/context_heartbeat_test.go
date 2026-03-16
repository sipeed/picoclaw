package tools

import (
	"context"
	"sync"
	"testing"
)

func TestWebSearchQuotaBasic(t *testing.T) {
	ctx := context.Background()

	// No quota → nil
	if q := GetWebSearchQuota(ctx); q != nil {
		t.Error("expected nil quota on bare context")
	}

	ctx = WithWebSearchQuota(ctx, 3)
	q := GetWebSearchQuota(ctx)
	if q == nil {
		t.Fatal("expected non-nil quota")
	}
	if q.Max() != 3 {
		t.Errorf("max = %d, want 3", q.Max())
	}
	if q.Remaining() != 3 {
		t.Errorf("remaining = %d, want 3", q.Remaining())
	}

	// Consume 3
	for i := range 3 {
		if !q.TryConsume() {
			t.Errorf("consume %d should succeed", i+1)
		}
	}

	// 4th should fail
	if q.TryConsume() {
		t.Error("consume after exhaustion should fail")
	}
	if q.Remaining() != 0 {
		t.Errorf("remaining = %d, want 0", q.Remaining())
	}
}

func TestWebSearchQuotaConcurrent(t *testing.T) {
	ctx := WithWebSearchQuota(context.Background(), 100)
	q := GetWebSearchQuota(ctx)

	var wg sync.WaitGroup
	consumed := make(chan bool, 200)

	for range 200 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			consumed <- q.TryConsume()
		}()
	}
	wg.Wait()
	close(consumed)

	ok := 0
	for c := range consumed {
		if c {
			ok++
		}
	}
	if ok != 100 {
		t.Errorf("consumed = %d, want exactly 100", ok)
	}
}

func TestHeartbeatContext(t *testing.T) {
	ctx := context.Background()
	if IsHeartbeatContext(ctx) {
		t.Error("bare context should not be heartbeat")
	}
	ctx = WithHeartbeatContext(ctx)
	if !IsHeartbeatContext(ctx) {
		t.Error("should be heartbeat after WithHeartbeatContext")
	}
}
