// PicoClaw - Ultra-lightweight personal AI agent
// Swarm mode support for multi-agent coordination
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// MetricsCollector collects and exports metrics for the swarm cluster.
type MetricsCollector struct {
	mu sync.RWMutex

	// Counters (atomic for performance)
	messagesSent      atomic.Int64
	messagesReceived  atomic.Int64
	handoffsInitiated atomic.Int64
	handoffsAccepted  atomic.Int64
	handoffsRejected  atomic.Int64
	handoffsFailed    atomic.Int64
	electionsWon      atomic.Int64

	// Gauges (use atomic.Value for float64)
	currentLoadScore atomic.Value // float64
	activeSessions   atomic.Int64
	memberCount      atomic.Int32

	// Histogram data (simplified)
	latencyBuckets map[string]*LatencyBucket

	startTime time.Time
}

// LatencyBucket tracks latency distribution.
type LatencyBucket struct {
	mu      sync.RWMutex
	count   int64
	sum     int64
	buckets [12]int64 // 0-1ms, 1-2ms, 2-5ms, 5-10ms, 10-20ms, 20-50ms, 50-100ms, 100-200ms, 200-500ms, 500ms-1s, 1-2s, 2s+
}

// NewMetricsCollector creates a new metrics collector.
func NewMetricsCollector() *MetricsCollector {
	mc := &MetricsCollector{
		latencyBuckets: make(map[string]*LatencyBucket),
		startTime:      time.Now(),
	}
	return mc
}

// Counter methods

// MessagesSent increments the sent message counter.
func (m *MetricsCollector) MessagesSent(n int64) {
	m.messagesSent.Add(n)
}

// MessagesReceived increments the received message counter.
func (m *MetricsCollector) MessagesReceived(n int64) {
	m.messagesReceived.Add(n)
}

// HandoffInitiated increments the handoff initiated counter.
func (m *MetricsCollector) HandoffInitiated() {
	m.handoffsInitiated.Add(1)
}

// HandoffAccepted increments the handoff accepted counter.
func (m *MetricsCollector) HandoffAccepted() {
	m.handoffsAccepted.Add(1)
}

// HandoffRejected increments the handoff rejected counter.
func (m *MetricsCollector) HandoffRejected() {
	m.handoffsRejected.Add(1)
}

// HandoffFailed increments the handoff failed counter.
func (m *MetricsCollector) HandoffFailed() {
	m.handoffsFailed.Add(1)
}

// ElectionWon increments the elections won counter.
func (m *MetricsCollector) ElectionWon() {
	m.electionsWon.Add(1)
}

// Gauge methods

// SetLoadScore sets the current load score.
func (m *MetricsCollector) SetLoadScore(score float64) {
	m.currentLoadScore.Store(score)
}

// SetActiveSessions sets the current active session count.
func (m *MetricsCollector) SetActiveSessions(count int64) {
	m.activeSessions.Store(count)
}

// SetMemberCount sets the current cluster member count.
func (m *MetricsCollector) SetMemberCount(count int32) {
	m.memberCount.Store(count)
}

// RecordLatency records a latency observation for the given operation.
func (m *MetricsCollector) RecordLatency(operation string, latency time.Duration) {
	m.mu.Lock()
	if m.latencyBuckets[operation] == nil {
		m.latencyBuckets[operation] = &LatencyBucket{}
	}
	bucket := m.latencyBuckets[operation]
	m.mu.Unlock()

	ms := latency.Milliseconds()

	bucket.mu.Lock()
	bucket.count++
	bucket.sum += ms

	// Bucket the latency
	switch {
	case ms < 1:
		bucket.buckets[0]++
	case ms < 2:
		bucket.buckets[1]++
	case ms < 5:
		bucket.buckets[2]++
	case ms < 10:
		bucket.buckets[3]++
	case ms < 20:
		bucket.buckets[4]++
	case ms < 50:
		bucket.buckets[5]++
	case ms < 100:
		bucket.buckets[6]++
	case ms < 200:
		bucket.buckets[7]++
	case ms < 500:
		bucket.buckets[8]++
	case ms < 1000:
		bucket.buckets[9]++
	case ms < 2000:
		bucket.buckets[10]++
	default:
		bucket.buckets[11]++
	}
	bucket.mu.Unlock()
}

// GetMetrics returns the current metrics as a map.
func (m *MetricsCollector) GetMetrics() map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()

	latency := make(map[string]any)
	for name, bucket := range m.latencyBuckets {
		bucket.mu.RLock()
		latency[name] = map[string]any{
			"count":  bucket.count,
			"avg_ms": float64(bucket.sum) / float64(bucket.count),
			"p50_ms": m.percentile(bucket, 0.50),
			"p95_ms": m.percentile(bucket, 0.95),
			"p99_ms": m.percentile(bucket, 0.99),
		}
		bucket.mu.RUnlock()
	}

	return map[string]any{
		// Counters
		"messages_sent":      m.messagesSent.Load(),
		"messages_received":  m.messagesReceived.Load(),
		"handoffs_initiated": m.handoffsInitiated.Load(),
		"handoffs_accepted":  m.handoffsAccepted.Load(),
		"handoffs_rejected":  m.handoffsRejected.Load(),
		"handoffs_failed":    m.handoffsFailed.Load(),
		"elections_won":      m.electionsWon.Load(),

		// Gauges
		"load_score":      m.currentLoadScore.Load(),
		"active_sessions": m.activeSessions.Load(),
		"member_count":    m.memberCount.Load(),

		// System info
		"uptime_seconds": time.Since(m.startTime).Seconds(),

		// Latency histograms
		"latency_ms": latency,
	}
}

// percentile calculates an approximate percentile from the bucket data.
func (m *MetricsCollector) percentile(bucket *LatencyBucket, p float64) float64 {
	if bucket.count == 0 {
		return 0
	}

	target := int64(float64(bucket.count) * p)
	cumulative := int64(0)

	// Upper bounds for each bucket in ms
	upperBounds := []int64{1, 2, 5, 10, 20, 50, 100, 200, 500, 1000, 2000, 1 << 62}

	for i, count := range bucket.buckets {
		cumulative += count
		if cumulative >= target {
			// Return approximate percentile
			return float64(upperBounds[i])
		}
	}

	return 2000.0 // default max
}

// ExportJSON exports metrics as JSON.
func (m *MetricsCollector) ExportJSON() ([]byte, error) {
	return json.MarshalIndent(m.GetMetrics(), "", "  ")
}

// ExportPrometheus exports metrics in Prometheus text format.
func (m *MetricsCollector) ExportPrometheus() string {
	metrics := m.GetMetrics()
	var out string

	// Counters as Prometheus counters
	out += "# TYPE picoclaw_messages_sent counter\n"
	out += fmt.Sprintf("picoclaw_messages_sent %d\n", metrics["messages_sent"])

	out += "\n# TYPE picoclaw_messages_received counter\n"
	out += fmt.Sprintf("picoclaw_messages_received %d\n", metrics["messages_received"])

	out += "\n# TYPE picoclaw_handoffs_initiated counter\n"
	out += fmt.Sprintf("picoclaw_handoffs_initiated %d\n", metrics["handoffs_initiated"])

	out += "\n# TYPE picoclaw_handoffs_accepted counter\n"
	out += fmt.Sprintf("picoclaw_handoffs_accepted %d\n", metrics["handoffs_accepted"])

	out += "\n# TYPE picoclaw_handoffs_rejected counter\n"
	out += fmt.Sprintf("picoclaw_handoffs_rejected %d\n", metrics["handoffs_rejected"])

	out += "\n# TYPE picoclaw_handoffs_failed counter\n"
	out += fmt.Sprintf("picoclaw_handoffs_failed %d\n", metrics["handoffs_failed"])

	out += "\n# TYPE picoclaw_elections_won counter\n"
	out += fmt.Sprintf("picoclaw_elections_won %d\n", metrics["elections_won"])

	// Gauges as Prometheus gauges
	out += "\n# TYPE picoclaw_load_score gauge\n"
	out += fmt.Sprintf("picoclaw_load_score %.2f\n", metrics["load_score"])

	out += "\n# TYPE picoclaw_active_sessions gauge\n"
	out += fmt.Sprintf("picoclaw_active_sessions %d\n", metrics["active_sessions"])

	out += "\n# TYPE picoclaw_member_count gauge\n"
	out += fmt.Sprintf("picoclaw_member_count %d\n", metrics["member_count"])

	out += "\n# TYPE picoclaw_uptime_seconds gauge\n"
	out += fmt.Sprintf("picoclaw_uptime_seconds %.0f\n", metrics["uptime_seconds"])

	return out
}

// Reset resets all metrics (useful for testing).
func (m *MetricsCollector) Reset() {
	m.messagesSent.Store(0)
	m.messagesReceived.Store(0)
	m.handoffsInitiated.Store(0)
	m.handoffsAccepted.Store(0)
	m.handoffsRejected.Store(0)
	m.handoffsFailed.Store(0)
	m.electionsWon.Store(0)
	m.currentLoadScore.Store(0)
	m.activeSessions.Store(0)
	m.memberCount.Store(0)

	m.mu.Lock()
	m.latencyBuckets = make(map[string]*LatencyBucket)
	m.mu.Unlock()
	m.startTime = time.Now()
}
