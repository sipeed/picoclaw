package health

import (
	"context"
	"runtime"
	"sync"
	"time"

	"jane/pkg/logger"
)

// ResourceTracker tracks and logs basic system resource usage over time.
// This is part of the ETL Ultimate Visibility framework to monitor Go/No-Go signals
// such as Goroutine leaks and memory spikes.
type ResourceTracker struct {
	interval time.Duration
	stopCh   chan struct{}
	stopOnce sync.Once
}

// NewResourceTracker creates a new ResourceTracker that logs metrics every `interval`.
func NewResourceTracker(interval time.Duration) *ResourceTracker {
	if interval == 0 {
		interval = 60 * time.Second // Default to 1 minute
	}
	return &ResourceTracker{
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start begins tracking resources in a background goroutine.
func (rt *ResourceTracker) Start(ctx context.Context) {
	ticker := time.NewTicker(rt.interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-rt.stopCh:
				return
			case <-ticker.C:
				rt.logResources()
			}
		}
	}()
}

// Stop gracefully stops the resource tracker.
func (rt *ResourceTracker) Stop() {
	rt.stopOnce.Do(func() {
		close(rt.stopCh)
	})
}

func (rt *ResourceTracker) logResources() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	goroutines := runtime.NumGoroutine()

	// Convert bytes to megabytes for readability in logs
	allocMB := float64(m.Alloc) / 1024 / 1024
	totalAllocMB := float64(m.TotalAlloc) / 1024 / 1024
	sysMB := float64(m.Sys) / 1024 / 1024

	logger.InfoCF("SystemHealth", "Resource tracking telemetry", map[string]any{
		"goroutines":       goroutines,
		"memory_alloc_mb":  allocMB,
		"memory_total_mb":  totalAllocMB,
		"memory_sys_mb":    sysMB,
		"num_gc":           m.NumGC,
		"gc_pause_ns":      m.PauseNs[(m.NumGC+255)%256], // Latest GC pause time
		"gc_pause_total_ns": m.PauseTotalNs,
	})
}
