package providers

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestRateLimiter_AllowsUpToRPM verifies that up to RPM requests pass immediately
// (burst capacity) and the (RPM+1)-th request is delayed.
func TestRateLimiter_AllowsUpToRPM(t *testing.T) {
	rpm := 5
	rl := newRateLimiter(rpm)

	// All rpm tokens should be available immediately (bucket starts full).
	for i := 0; i < rpm; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		if err := rl.Wait(ctx); err != nil {
			t.Fatalf("request %d should pass immediately, got: %v", i+1, err)
		}
		cancel()
	}

	// The next request must wait; cancel it to confirm it blocks.
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	err := rl.Wait(ctx)
	if err == nil {
		t.Fatal("expected request beyond RPM to block, but it passed immediately")
	}
}

// TestRateLimiter_ContextCancellation verifies that a blocked Wait respects cancellation.
func TestRateLimiter_ContextCancellation(t *testing.T) {
	rl := newRateLimiter(1)

	// Drain the one token.
	ctx := context.Background()
	if err := rl.Wait(ctx); err != nil {
		t.Fatalf("first request failed: %v", err)
	}

	// Second request should block; cancel it.
	cancelCtx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	err := rl.Wait(cancelCtx)
	if err == nil {
		t.Fatal("expected cancellation error, got nil")
	}
}

// TestRateLimiter_TokenRefill verifies that tokens refill over time.
func TestRateLimiter_TokenRefill(t *testing.T) {
	rpm := 60 // 1 token per second
	rl := newRateLimiter(rpm)

	// Drain all tokens.
	for i := 0; i < rpm; i++ {
		rl.Wait(context.Background()) //nolint:errcheck
	}

	// Advance time via nowFunc: simulate 2 seconds passing (should give 2 tokens).
	start := time.Now()
	rl.nowFunc = func() time.Time { return start.Add(2 * time.Second) }

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := rl.Wait(ctx); err != nil {
		t.Fatalf("expected refilled token to be available: %v", err)
	}
}

// TestRateLimiterRegistry_NoLimiter verifies that keys without a registered limiter pass freely.
func TestRateLimiterRegistry_NoLimiter(t *testing.T) {
	r := NewRateLimiterRegistry()
	ctx := context.Background()
	for i := 0; i < 100; i++ {
		if err := r.Wait(ctx, "unregistered/key"); err != nil {
			t.Fatalf("unregistered key should not block: %v", err)
		}
	}
}

// TestRateLimiterRegistry_ZeroRPM verifies that RPM=0 means no limiter is registered.
func TestRateLimiterRegistry_ZeroRPM(t *testing.T) {
	r := NewRateLimiterRegistry()
	r.Register("some/key", 0)
	ctx := context.Background()
	for i := 0; i < 50; i++ {
		if err := r.Wait(ctx, "some/key"); err != nil {
			t.Fatalf("zero-RPM key should not block: %v", err)
		}
	}
}

// TestRateLimiterRegistry_Enforcement verifies the registry enforces RPM per key.
func TestRateLimiterRegistry_Enforcement(t *testing.T) {
	r := NewRateLimiterRegistry()
	r.Register("openai/gpt-4o", 3)

	// First 3 calls should pass (burst = RPM).
	for i := 0; i < 3; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		if err := r.Wait(ctx, "openai/gpt-4o"); err != nil {
			t.Fatalf("call %d should pass: %v", i+1, err)
		}
		cancel()
	}

	// 4th call should block.
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := r.Wait(ctx, "openai/gpt-4o"); err == nil {
		t.Fatal("4th call should have been rate-limited")
	}
}

// TestRateLimiterRegistry_RegisterCandidates verifies that RegisterCandidates
// correctly picks up RPM from FallbackCandidate.
func TestRateLimiterRegistry_RegisterCandidates(t *testing.T) {
	r := NewRateLimiterRegistry()
	candidates := []FallbackCandidate{
		{Provider: "openai", Model: "gpt-4o", RPM: 2},
		{Provider: "anthropic", Model: "claude-3", RPM: 0}, // no limit
	}
	r.RegisterCandidates(candidates)

	// openai/gpt-4o: 2 tokens burst, 3rd should block.
	for i := 0; i < 2; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		if err := r.Wait(ctx, "openai/gpt-4o"); err != nil {
			t.Fatalf("openai call %d should pass: %v", i+1, err)
		}
		cancel()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := r.Wait(ctx, "openai/gpt-4o"); err == nil {
		t.Fatal("openai 3rd call should have been limited")
	}

	// anthropic/claude-3: no limit, should always pass.
	for i := 0; i < 10; i++ {
		if err := r.Wait(context.Background(), "anthropic/claude-3"); err != nil {
			t.Fatalf("anthropic call should not be limited: %v", err)
		}
	}
}

func TestRateLimiterRegistry_RegisterCandidatesUsesStableIdentity(t *testing.T) {
	r := NewRateLimiterRegistry()
	candidates := []FallbackCandidate{
		{Provider: "openai", Model: "gpt-4o", RPM: 1, IdentityKey: "model_name:primary"},
		{Provider: "openai", Model: "gpt-4o", RPM: 2, IdentityKey: "model_name:fallback"},
	}
	r.RegisterCandidates(candidates)

	if err := r.Wait(context.Background(), "model_name:primary"); err != nil {
		t.Fatalf("primary first call should pass: %v", err)
	}
	if err := r.Wait(context.Background(), "model_name:fallback"); err != nil {
		t.Fatalf("fallback first call should pass: %v", err)
	}
	if err := r.Wait(context.Background(), "model_name:fallback"); err != nil {
		t.Fatalf("fallback second call should pass: %v", err)
	}

	ctxPrimary, cancelPrimary := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancelPrimary()
	if err := r.Wait(ctxPrimary, "model_name:primary"); err == nil {
		t.Fatal("primary second call should have been limited")
	}

	ctxFallback, cancelFallback := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancelFallback()
	if err := r.Wait(ctxFallback, "model_name:fallback"); err == nil {
		t.Fatal("fallback third call should have been limited")
	}
}

// TestRateLimiter_Concurrency verifies thread safety under concurrent access.
func TestRateLimiter_Concurrency(t *testing.T) {
	rpm := 20
	rl := newRateLimiter(rpm)
	var passed atomic.Int64
	var wg sync.WaitGroup

	// Launch 30 goroutines; only ~20 should pass immediately.
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
			defer cancel()
			if rl.Wait(ctx) == nil {
				passed.Add(1)
			}
		}()
	}
	wg.Wait()

	got := passed.Load()
	// Allow small timing slack: between rpm-2 and rpm+2.
	if got < int64(rpm-2) || got > int64(rpm+2) {
		t.Fatalf("expected ~%d immediate passes, got %d", rpm, got)
	}
}
