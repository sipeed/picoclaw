package health

import (
	"context"
	"runtime"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"jane/pkg/logger"
)

// ResourceTracker tracks and logs basic system resource usage over time.
// This is part of the ETL Ultimate Visibility framework to monitor Go/No-Go signals
// such as Goroutine leaks and memory spikes.
type ResourceTracker struct {
	interval time.Duration
	stopCh   chan struct{}
	stopOnce sync.Once

	// OpenTelemetry gauges
	goroutinesGauge metric.Int64ObservableGauge
	allocMBGauge    metric.Float64ObservableGauge
	totalMBGauge    metric.Float64ObservableGauge
	sysMBGauge      metric.Float64ObservableGauge
}

// NewResourceTracker creates a new ResourceTracker that logs metrics every `interval`.
func NewResourceTracker(interval time.Duration) *ResourceTracker {
	if interval == 0 {
		interval = 60 * time.Second // Default to 1 minute
	}

	rt := &ResourceTracker{
		interval: interval,
		stopCh:   make(chan struct{}),
	}

	meter := otel.Meter("jane/pkg/health")

	var err error
	rt.goroutinesGauge, err = meter.Int64ObservableGauge(
		"sys.goroutines",
		metric.WithDescription("Number of active goroutines"),
	)
	if err != nil {
		logger.ErrorCF("SystemHealth", "Failed to create goroutines gauge", map[string]any{"error": err.Error()})
	}

	rt.allocMBGauge, err = meter.Float64ObservableGauge(
		"sys.memory.alloc_mb",
		metric.WithDescription("Memory allocated and still in use, in MB"),
	)
	if err != nil {
		logger.ErrorCF("SystemHealth", "Failed to create memory alloc gauge", map[string]any{"error": err.Error()})
	}

	rt.totalMBGauge, err = meter.Float64ObservableGauge(
		"sys.memory.total_alloc_mb",
		metric.WithDescription("Total memory allocated (even if freed), in MB"),
	)
	if err != nil {
		logger.ErrorCF("SystemHealth", "Failed to create total memory alloc gauge", map[string]any{"error": err.Error()})
	}

	rt.sysMBGauge, err = meter.Float64ObservableGauge(
		"sys.memory.sys_mb",
		metric.WithDescription("Total memory obtained from the OS, in MB"),
	)
	if err != nil {
		logger.ErrorCF("SystemHealth", "Failed to create memory sys gauge", map[string]any{"error": err.Error()})
	}

	if _, err := meter.RegisterCallback(rt.observeMetrics, rt.goroutinesGauge, rt.allocMBGauge, rt.totalMBGauge, rt.sysMBGauge); err != nil {
		logger.ErrorCF("SystemHealth", "Failed to register metrics callback", map[string]any{"error": err.Error()})
	}

	return rt
}

func (rt *ResourceTracker) observeMetrics(_ context.Context, o metric.Observer) error {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	goroutines := runtime.NumGoroutine()
	allocMB := float64(m.Alloc) / 1024 / 1024
	totalAllocMB := float64(m.TotalAlloc) / 1024 / 1024
	sysMB := float64(m.Sys) / 1024 / 1024

	if rt.goroutinesGauge != nil {
		o.ObserveInt64(rt.goroutinesGauge, int64(goroutines))
	}
	if rt.allocMBGauge != nil {
		o.ObserveFloat64(rt.allocMBGauge, allocMB)
	}
	if rt.totalMBGauge != nil {
		o.ObserveFloat64(rt.totalMBGauge, totalAllocMB)
	}
	if rt.sysMBGauge != nil {
		o.ObserveFloat64(rt.sysMBGauge, sysMB)
	}

	return nil
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
