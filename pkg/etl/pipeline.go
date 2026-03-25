package etl

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"jane/pkg/logger"
)

// Pipeline manages the ETL process for Ultimate Visibility metrics
type Pipeline struct {
	workspacePath string
	interval      time.Duration
	stopCh        chan struct{}
	stopOnce      sync.Once
}

// SystemMetrics represents the extracted system KPIs
type SystemMetrics struct {
	Timestamp       time.Time `json:"timestamp"`
	Goroutines      int       `json:"goroutines"`
	MemoryAllocMB   float64   `json:"memory_alloc_mb"`
	MemoryTotalMB   float64   `json:"memory_total_mb"`
	MemorySysMB     float64   `json:"memory_sys_mb"`
	NumGC           uint32    `json:"num_gc"`
}

// NewPipeline creates a new ETL pipeline
func NewPipeline(workspacePath string, interval time.Duration) *Pipeline {
	if interval == 0 {
		interval = 1 * time.Minute
	}
	return &Pipeline{
		workspacePath: workspacePath,
		interval:      interval,
		stopCh:        make(chan struct{}),
	}
}

// Start begins the ETL extraction loop
func (p *Pipeline) Start(ctx context.Context) {
	ticker := time.NewTicker(p.interval)

	// Ensure log directory exists
	logDir := filepath.Join(p.workspacePath, "logs", "etl")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		logger.ErrorCF("ETL", "Failed to create ETL log directory", map[string]any{"error": err.Error()})
		return
	}

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-p.stopCh:
				return
			case <-ticker.C:
				p.extractAndLoad()
			}
		}
	}()
	logger.InfoCF("ETL", "Pipeline started", map[string]any{"interval": p.interval.String()})
}

// Stop halts the ETL pipeline
func (p *Pipeline) Stop() {
	p.stopOnce.Do(func() {
		close(p.stopCh)
	})
}

func (p *Pipeline) extractAndLoad() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	metrics := SystemMetrics{
		Timestamp:     time.Now().UTC(),
		Goroutines:    runtime.NumGoroutine(),
		MemoryAllocMB: float64(m.Alloc) / 1024 / 1024,
		MemoryTotalMB: float64(m.TotalAlloc) / 1024 / 1024,
		MemorySysMB:   float64(m.Sys) / 1024 / 1024,
		NumGC:         m.NumGC,
	}

	// Transform (JSON serialization)
	data, err := json.Marshal(metrics)
	if err != nil {
		logger.ErrorCF("ETL", "Failed to marshal metrics", map[string]any{"error": err.Error()})
		return
	}

	// Load (Write to JSONL file)
	logFile := filepath.Join(p.workspacePath, "logs", "etl", "system_metrics.jsonl")
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.ErrorCF("ETL", "Failed to open metrics file", map[string]any{"error": err.Error()})
		return
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		logger.ErrorCF("ETL", "Failed to write metrics", map[string]any{"error": err.Error()})
	}
}
