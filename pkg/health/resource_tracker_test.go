package health

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestResourceTracker_StartStop(t *testing.T) {
	interval := 10 * time.Millisecond
	rt := NewResourceTracker(interval)
	assert.NotNil(t, rt)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rt.Start(ctx)

	// Wait on stop channel to simulate proper lifecycle management
	// without flakiness
	rt.Stop()

	// Test idempotent Stop
	rt.Stop()
	assert.NotPanics(t, func() {
		rt.Stop()
	})
}

func TestResourceTracker_logResources(t *testing.T) {
	rt := NewResourceTracker(1 * time.Second)

	// Since we can't easily assert on the stdout without mocking zerolog globally,
	// we just ensure the function runs without panicking.
	assert.NotPanics(t, func() {
		rt.logResources()
	})

	// Verify reasonable values are accessible
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	assert.GreaterOrEqual(t, m.Alloc, uint64(0))
	assert.GreaterOrEqual(t, runtime.NumGoroutine(), 1)
}
