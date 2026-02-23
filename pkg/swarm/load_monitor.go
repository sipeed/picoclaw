// PicoClaw - Ultra-lightweight personal AI agent
// Swarm mode support for multi-agent coordination
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"runtime"
	"sync"
	"time"
)

// LoadMonitor monitors system load and calculates a load score.
type LoadMonitor struct {
	config      *LoadMonitorConfig
	samples     []float64
	mu          sync.RWMutex
	sessionCount int
	ticker       *time.Ticker
	stopChan     chan struct{}
	onThreshold  []func(float64)
}

// NewLoadMonitor creates a new load monitor.
func NewLoadMonitor(config *LoadMonitorConfig) *LoadMonitor {
	if config.SampleSize <= 0 {
		config.SampleSize = 60
	}
	if config.Interval.Duration <= 0 {
		config.Interval = Duration{5 * time.Second}
	}

	lm := &LoadMonitor{
		config:     config,
		samples:    make([]float64, 0, config.SampleSize),
		stopChan:   make(chan struct{}),
		onThreshold: make([]func(float64), 0),
	}
	return lm
}

// Start begins monitoring load.
func (lm *LoadMonitor) Start() {
	if lm.ticker != nil {
		return
	}

	lm.ticker = time.NewTicker(lm.config.Interval.Duration)
	go lm.run()
}

// Stop stops monitoring load.
func (lm *LoadMonitor) Stop() {
	if lm.ticker == nil {
		return
	}

	lm.ticker.Stop()
	close(lm.stopChan)
	lm.ticker = nil
}

// run is the main monitoring loop.
func (lm *LoadMonitor) run() {
	for {
		select {
		case <-lm.ticker.C:
			score := lm.calculateScore()
			lm.addSample(score)

			// Check threshold callbacks
			if lm.shouldOffload() {
				lm.mu.RLock()
				callbacks := make([]func(float64), len(lm.onThreshold))
				copy(callbacks, lm.onThreshold)
				lm.mu.RUnlock()

				for _, cb := range callbacks {
					go cb(score)
				}
			}
		case <-lm.stopChan:
			return
		}
	}
}

// LoadMetrics represents current load metrics.
type LoadMetrics struct {
	CPUUsage      float64 `json:"cpu_usage"`
	MemoryUsage   float64 `json:"memory_usage"`
	ActiveSessions int    `json:"active_sessions"`
	Goroutines    int     `json:"goroutines"`
	Score         float64 `json:"score"`
	Timestamp     int64   `json:"timestamp"`
}

// GetCurrentLoad returns the current load metrics.
func (lm *LoadMonitor) GetCurrentLoad() *LoadMetrics {
	metrics := &LoadMetrics{
		ActiveSessions: lm.GetSessionCount(),
		Goroutines:    runtime.NumGoroutine(),
		Timestamp:     time.Now().UnixNano(),
	}

	// Get memory usage
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Normalize using configured thresholds
	maxMem := lm.config.MaxMemoryBytes
	if maxMem == 0 {
		maxMem = 1024 * 1024 * 1024 // Default 1GB
	}
	metrics.MemoryUsage = normalizeMemory(m.Alloc, maxMem)

	maxGoroutines := lm.config.MaxGoroutines
	if maxGoroutines == 0 {
		maxGoroutines = 1000
	}
	metrics.CPUUsage = normalizeCPU(metrics.Goroutines, maxGoroutines)

	maxSessions := lm.config.MaxSessions
	if maxSessions == 0 {
		maxSessions = 100
	}
	sessionUsage := normalizeSessions(metrics.ActiveSessions, maxSessions)

	// Calculate weighted score
	config := lm.config
	metrics.Score = (metrics.CPUUsage * config.CPUWeight) +
		(metrics.MemoryUsage * config.MemoryWeight) +
		(sessionUsage * config.SessionWeight)

	// Clamp score to [0, 1]
	if metrics.Score < 0 {
		metrics.Score = 0
	} else if metrics.Score > 1 {
		metrics.Score = 1
	}

	return metrics
}

// calculateScore calculates the current load score.
func (lm *LoadMonitor) calculateScore() float64 {
	return lm.GetCurrentLoad().Score
}

// addSample adds a load sample to the history.
func (lm *LoadMonitor) addSample(score float64) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	lm.samples = append(lm.samples, score)
	if len(lm.samples) > lm.config.SampleSize {
		lm.samples = lm.samples[1:]
	}
}

// GetAverageScore returns the average load score over the sample window.
func (lm *LoadMonitor) GetAverageScore() float64 {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	if len(lm.samples) == 0 {
		return lm.calculateScore()
	}

	sum := 0.0
	for _, s := range lm.samples {
		sum += s
	}
	return sum / float64(len(lm.samples))
}

// GetSessionCount returns the current number of active sessions.
func (lm *LoadMonitor) GetSessionCount() int {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	return lm.sessionCount
}

// SetSessionCount sets the current number of active sessions.
func (lm *LoadMonitor) SetSessionCount(count int) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.sessionCount = count
}

// IncrementSessions increments the session count.
func (lm *LoadMonitor) IncrementSessions() {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.sessionCount++
}

// DecrementSessions decrements the session count.
func (lm *LoadMonitor) DecrementSessions() {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	if lm.sessionCount > 0 {
		lm.sessionCount--
	}
}

// ShouldOffload returns true if the load is high enough to offload tasks.
func (lm *LoadMonitor) ShouldOffload() bool {
	return lm.shouldOffload()
}

// shouldOffload internal check for offloading.
func (lm *LoadMonitor) shouldOffload() bool {
	avgScore := lm.GetAverageScore()
	currentScore := lm.calculateScore()

	// Use configured offload threshold, or default to 0.8
	threshold := lm.config.OffloadThreshold
	if threshold <= 0 {
		threshold = 0.8
	}

	// Use a combination of current and average for smoother behavior
	combinedScore := (currentScore*0.7 + avgScore*0.3)
	return combinedScore > threshold
}

// OnThreshold registers a callback when the load threshold is exceeded.
func (lm *LoadMonitor) OnThreshold(callback func(float64)) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.onThreshold = append(lm.onThreshold, callback)
}

// GetTrend returns the load trend: "increasing", "decreasing", or "stable".
func (lm *LoadMonitor) GetTrend() string {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	if len(lm.samples) < 3 {
		return "stable"
	}

	// Simple linear regression to detect trend
	n := float64(len(lm.samples))
	sumX := n*(n-1)/2
	sumY := 0.0
	sumXY := 0.0

	for i, s := range lm.samples {
		x := float64(i)
		sumY += s
		sumXY += x * s
	}

	slope := (n*sumXY - sumX*sumY) / (n*(n-1)*(2*n-1)/6)

	if slope > TrendIncreasingThreshold {
		return "increasing"
	} else if slope < TrendDecreasingThreshold {
		return "decreasing"
	}
	return "stable"
}

// Helper functions for normalization

func normalizeMemory(alloc uint64, maxMem uint64) float64 {
	// Use configured max memory threshold
	usage := float64(alloc) / float64(maxMem)
	if usage > 1 {
		return 1
	}
	return usage
}

func normalizeCPU(goroutines int, maxGoroutines int) float64 {
	// Use configured max goroutine threshold
	usage := float64(goroutines) / float64(maxGoroutines)
	if usage > 1 {
		return 1
	}
	return usage
}

func normalizeSessions(sessions int, maxSessions int) float64 {
	// Use configured max sessions threshold
	usage := float64(sessions) / float64(maxSessions)
	if usage > 1 {
		return 1
	}
	return usage
}
