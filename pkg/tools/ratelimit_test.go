package tools

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRateBucket_AllowsUnderLimit(t *testing.T) {
	now := time.Now()
	rb := newRateBucket(5, func() time.Time { return now })

	for i := 0; i < 5; i++ {
		assert.True(t, rb.Allow(), "call %d should be allowed", i+1)
	}
}

func TestRateBucket_BlocksOverLimit(t *testing.T) {
	now := time.Now()
	rb := newRateBucket(3, func() time.Time { return now })

	for i := 0; i < 3; i++ {
		assert.True(t, rb.Allow())
	}
	assert.False(t, rb.Allow(), "4th call should be blocked")
	assert.False(t, rb.Allow(), "5th call should also be blocked")
}

func TestRateBucket_ResetsAfterWindow(t *testing.T) {
	now := time.Now()
	currentTime := now
	rb := newRateBucket(2, func() time.Time { return currentTime })

	// Fill the bucket
	assert.True(t, rb.Allow())
	assert.True(t, rb.Allow())
	assert.False(t, rb.Allow(), "should be blocked at limit")

	// Advance time past the 1-minute window
	currentTime = now.Add(61 * time.Second)

	// Should be allowed again
	assert.True(t, rb.Allow(), "should be allowed after window expires")
	assert.True(t, rb.Allow(), "second call after reset should be allowed")
	assert.False(t, rb.Allow(), "should be blocked again at limit")
}

func TestRateBucket_PartialExpiry(t *testing.T) {
	now := time.Now()
	currentTime := now
	rb := newRateBucket(3, func() time.Time { return currentTime })

	// Make 3 calls
	assert.True(t, rb.Allow())
	currentTime = now.Add(10 * time.Second)
	assert.True(t, rb.Allow())
	currentTime = now.Add(20 * time.Second)
	assert.True(t, rb.Allow())
	assert.False(t, rb.Allow(), "should be blocked at limit")

	// Advance 50s — first call (at t=0) expires, others (t=10, t=20) still valid
	currentTime = now.Add(61 * time.Second)

	// One slot freed up
	assert.True(t, rb.Allow(), "should allow one more after partial expiry")
}

func TestRateBucket_Concurrent(t *testing.T) {
	now := time.Now()
	rb := newRateBucket(100, func() time.Time { return now })

	var wg sync.WaitGroup
	allowed := int32(0)
	var mu sync.Mutex

	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if rb.Allow() {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(100), allowed, "exactly 100 calls should be allowed")
}
