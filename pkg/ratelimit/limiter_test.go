package ratelimit

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestBucket_TryTake(t *testing.T) {
	b := newBucket(10, 1) // 10 tokens, 1 token/sec refill

	// Should be able to take tokens
	for i := 0; i < 10; i++ {
		if !b.tryTake(1) {
			t.Errorf("Expected to take token %d", i)
		}
	}

	// Should not be able to take more
	if b.tryTake(1) {
		t.Error("Should not be able to take more tokens")
	}
}

func TestBucket_Refill(t *testing.T) {
	b := newBucket(10, 10) // 10 tokens, 10 tokens/sec refill

	// Take all tokens
	for i := 0; i < 10; i++ {
		b.tryTake(1)
	}

	// Wait for refill
	time.Sleep(200 * time.Millisecond)

	// Should have ~2 tokens now
	if !b.tryTake(1) {
		t.Error("Should have refilled at least 1 token")
	}
}

func TestBucket_WaitUntil(t *testing.T) {
	b := newBucket(1, 1) // 1 token, 1 token/sec refill

	// Take the token
	b.tryTake(1)

	// Wait should succeed after refill
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	start := time.Now()
	err := b.waitUntil(ctx, 1)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("waitUntil failed: %v", err)
	}

	// Should have waited approximately 1 second
	if elapsed < 500*time.Millisecond {
		t.Errorf("Waited too short: %v", elapsed)
	}
}

func TestBucket_WaitUntil_Cancel(t *testing.T) {
	b := newBucket(1, 0.1) // 1 token, very slow refill

	// Take the token
	b.tryTake(1)

	// Cancel immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := b.waitUntil(ctx, 1)
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got: %v", err)
	}
}

func TestLimiter_AllowRequest(t *testing.T) {
	config := Config{
		Enabled:           true,
		RequestsPerMinute: 5,
		PerUserLimit:      false,
	}

	l := NewLimiter(config)

	// Should allow first 5 requests
	for i := 0; i < 5; i++ {
		if !l.AllowRequest("user1") {
			t.Errorf("Request %d should be allowed", i)
		}
	}

	// 6th should be denied
	if l.AllowRequest("user1") {
		t.Error("Request 6 should be denied")
	}
}

func TestLimiter_PerUserLimit(t *testing.T) {
	config := Config{
		Enabled:           true,
		RequestsPerMinute: 10,
		PerUserLimit:      true,
	}

	l := NewLimiter(config)

	// User1 uses 5 requests
	for i := 0; i < 5; i++ {
		if !l.AllowRequest("user1") {
			t.Errorf("User1 request %d should be allowed", i)
		}
	}

	// User2 should still have their own limit
	for i := 0; i < 5; i++ {
		if !l.AllowRequest("user2") {
			t.Errorf("User2 request %d should be allowed", i)
		}
	}
}

func TestLimiter_ToolExecution(t *testing.T) {
	config := Config{
		Enabled:                 true,
		RequestsPerMinute:       100,
		ToolExecutionsPerMinute: 3,
		PerUserLimit:            true,
	}

	l := NewLimiter(config)

	// Should allow first 3 tool executions
	for i := 0; i < 3; i++ {
		if !l.AllowToolExecution("user1", "test_tool") {
			t.Errorf("Tool execution %d should be allowed", i)
		}
	}

	// 4th should be denied
	if l.AllowToolExecution("user1", "test_tool") {
		t.Error("Tool execution 4 should be denied")
	}
}

func TestLimiter_Disabled(t *testing.T) {
	config := Config{
		Enabled: false,
	}

	l := NewLimiter(config)

	// Should allow all requests when disabled
	for i := 0; i < 100; i++ {
		if !l.AllowRequest("user1") {
			t.Errorf("Request %d should be allowed when disabled", i)
		}
	}
}

func TestLimiter_Reset(t *testing.T) {
	config := Config{
		Enabled:           true,
		RequestsPerMinute: 2,
		PerUserLimit:      false,
	}

	l := NewLimiter(config)

	// Use all tokens
	l.AllowRequest("user1")
	l.AllowRequest("user1")

	// Should be denied
	if l.AllowRequest("user1") {
		t.Error("Should be denied after using all tokens")
	}

	// Reset
	l.Reset()

	// Should be allowed again
	if !l.AllowRequest("user1") {
		t.Error("Should be allowed after reset")
	}
}

func TestLimiter_GetStatus(t *testing.T) {
	config := Config{
		Enabled:                 true,
		RequestsPerMinute:       10,
		ToolExecutionsPerMinute: 5,
		PerUserLimit:            true,
	}

	l := NewLimiter(config)

	// Use some tokens
	l.AllowRequest("user1")
	l.AllowRequest("user1")
	l.AllowToolExecution("user1", "tool1")

	status := l.GetStatus("user1")

	if status.RequestsUsed != 2 {
		t.Errorf("RequestsUsed = %d, want 2", status.RequestsUsed)
	}

	if status.ToolsUsed != 1 {
		t.Errorf("ToolsUsed = %d, want 1", status.ToolsUsed)
	}

	if status.RequestsLimit != 10 {
		t.Errorf("RequestsLimit = %d, want 10", status.RequestsLimit)
	}

	if status.ToolsLimit != 5 {
		t.Errorf("ToolsLimit = %d, want 5", status.ToolsLimit)
	}
}

func TestLimiter_Concurrent(t *testing.T) {
	config := Config{
		Enabled:           true,
		RequestsPerMinute: 1000,
		PerUserLimit:      false,
	}

	l := NewLimiter(config)

	var wg sync.WaitGroup
	allowed := make(chan bool, 1000)

	// Launch 1000 concurrent requests
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			allowed <- l.AllowRequest("user1")
		}()
	}

	wg.Wait()
	close(allowed)

	// Count allowed requests
	allowedCount := 0
	for a := range allowed {
		if a {
			allowedCount++
		}
	}

	// Should have allowed approximately 1000 (with some tolerance for timing)
	if allowedCount < 950 {
		t.Errorf("Only %d requests allowed, expected ~1000", allowedCount)
	}
}

func TestLimiter_WaitForRequest(t *testing.T) {
	config := Config{
		Enabled:           true,
		RequestsPerMinute: 60, // 1 per second
		PerUserLimit:      false,
	}

	l := NewLimiter(config)

	// Use all tokens
	for i := 0; i < 60; i++ {
		l.AllowRequest("user1")
	}

	// Wait should succeed after refill
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	err := l.WaitForRequest(ctx, "user1")
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("WaitForRequest failed: %v", err)
	}

	// Should have waited at least some time
	if elapsed < 100*time.Millisecond {
		t.Errorf("Waited too short: %v", elapsed)
	}
}

func TestGlobalLimiter(t *testing.T) {
	config := Config{
		Enabled:           true,
		RequestsPerMinute: 5,
		PerUserLimit:      false,
	}

	InitGlobal(config)

	// Should allow first 5 requests
	for i := 0; i < 5; i++ {
		if !Allow("user1") {
			t.Errorf("Global request %d should be allowed", i)
		}
	}

	// Should be denied
	if Allow("user1") {
		t.Error("Global request 6 should be denied")
	}

	// Check status
	status := GetGlobalStatus("user1")
	if status.RequestsLimit != 5 {
		t.Errorf("Global RequestsLimit = %d, want 5", status.RequestsLimit)
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Enabled {
		t.Error("Default config should have rate limiting disabled")
	}

	if config.RequestsPerMinute != 60 {
		t.Errorf("Default RequestsPerMinute = %d, want 60", config.RequestsPerMinute)
	}

	if config.ToolExecutionsPerMinute != 30 {
		t.Errorf("Default ToolExecutionsPerMinute = %d, want 30", config.ToolExecutionsPerMinute)
	}
}

func TestLimiter_Cleanup(t *testing.T) {
	config := Config{
		Enabled:           true,
		RequestsPerMinute: 10,
		PerUserLimit:      true,
	}

	l := NewLimiter(config)

	// Create buckets for multiple users
	l.AllowRequest("user1")
	l.AllowRequest("user2")
	l.AllowRequest("user3")

	// Cleanup immediately (should not remove active buckets)
	l.Cleanup(1 * time.Hour)

	// Buckets should still exist
	if _, ok := l.buckets.Load("user1"); !ok {
		t.Error("user1 bucket should still exist")
	}

	// Wait and cleanup with short max age
	time.Sleep(100 * time.Millisecond)
	l.Cleanup(1 * time.Millisecond)

	// Old buckets should be removed
	if _, ok := l.buckets.Load("user1"); ok {
		t.Error("user1 bucket should be cleaned up")
	}
}

func TestLimiter_SetConfig(t *testing.T) {
	l := NewLimiter(Config{Enabled: false})

	// Should allow when disabled
	if !l.AllowRequest("user1") {
		t.Error("Should allow when disabled")
	}

	// Enable with new config
	l.SetConfig(Config{
		Enabled:           true,
		RequestsPerMinute: 2,
		PerUserLimit:      false,
	})

	// Should now enforce limits
	l.AllowRequest("user1")
	l.AllowRequest("user1")

	if l.AllowRequest("user1") {
		t.Error("Should deny after new config limit")
	}
}
