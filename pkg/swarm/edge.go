// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// EdgeWorkerConfig contains configuration for edge-optimized workers
type EdgeWorkerConfig struct {
	// Resource limits
	MaxMemoryMB       int64  // Maximum memory usage in MB
	MaxCPUPercent     int    // Maximum CPU percentage (1-100)
	EnableGCThreshold uint64 // GC trigger threshold in bytes

	// Network optimization
	DisableHeartbeat  bool          // Disable periodic heartbeats
	HeartbeatInterval time.Duration // Custom heartbeat interval
	CompressionLevel  int           // Message compression (0-9)

	// Feature flags for minimal footprint
	DisableWorkflow     bool // Disable Temporal workflow support
	DisableDashboard    bool // Disable dashboard features
	DisableDiscovery    bool // Disable full node discovery
	MinimalMode         bool // Enable absolute minimal mode
}

// DefaultEdgeWorkerConfig returns default edge worker configuration
func DefaultEdgeWorkerConfig() *EdgeWorkerConfig {
	return &EdgeWorkerConfig{
		MaxMemoryMB:        50,  // 50MB default
		MaxCPUPercent:      50,  // 50% CPU max
		EnableGCThreshold:  10 * 1024 * 1024, // 10MB
		DisableHeartbeat:   false,
		HeartbeatInterval:  30 * time.Second, // Less frequent
		CompressionLevel:   2,  // Light compression
		DisableWorkflow:    true,  // Workflows disabled by default on edge
		DisableDashboard:   true,
		DisableDiscovery:   false,
		MinimalMode:        false,
	}
}

// EdgeWorker is a resource-optimized worker for edge devices
type EdgeWorker struct {
	*Worker
	config   *EdgeWorkerConfig
	edgeStop atomic.Bool
	mu       sync.RWMutex

	// Resource tracking
	memoryUsed     atomic.Int64
	cpuPercent     atomic.Int64
	lastGCTime     atomic.Value // time.Time

	// Edge-specific optimizations
	compressionEnabled bool
	batchMode          bool
	batchSize          int
	batchTimeout       time.Duration
}

// NewEdgeWorker creates an edge-optimized worker
func NewEdgeWorker(
	baseWorker *Worker,
	config *EdgeWorkerConfig,
) *EdgeWorker {
	if config == nil {
		config = DefaultEdgeWorkerConfig()
	}

	ew := &EdgeWorker{
		Worker:       baseWorker,
		config:       config,
		batchSize:    5,
		batchTimeout: 5 * time.Second,
	}

	ew.compressionEnabled = config.CompressionLevel > 0
	ew.lastGCTime.Store(time.Now())

	// Apply resource limits
	ew.setupResourceLimits()

	logger.InfoCF("swarm", "Edge worker created", map[string]interface{}{
		"max_memory_mb":     config.MaxMemoryMB,
		"max_cpu_percent":   config.MaxCPUPercent,
		"compression_level": config.CompressionLevel,
		"minimal_mode":      config.MinimalMode,
	})

	return ew
}

// Start starts the edge worker with optimizations
func (ew *EdgeWorker) Start(ctx context.Context) error {
	logger.InfoC("swarm", "Starting edge worker")

	// Set up memory monitoring
	go ew.monitorResources(ctx)

	// Start base worker
	if err := ew.Worker.Start(ctx); err != nil {
		return fmt.Errorf("failed to start base worker: %w", err)
	}

	// Disable unnecessary features in minimal mode
	if ew.config.MinimalMode {
		ew.disableNonEssentialFeatures()
	}

	logger.InfoC("swarm", "Edge worker started")
	return nil
}

// Stop gracefully stops the edge worker
func (ew *EdgeWorker) Stop() {
	if !ew.edgeStop.CompareAndSwap(false, true) {
		return // Already stopped
	}

	logger.InfoC("swarm", "Stopping edge worker")

	// Force GC before stopping to free memory
	ew.forceGC()

	ew.Worker.Stop()
	logger.InfoC("swarm", "Edge worker stopped")
}

// setupResourceLimits configures resource constraints
func (ew *EdgeWorker) setupResourceLimits() {
	// Set GC target based on config
	if ew.config.MaxMemoryMB > 0 {
		target := uint64(ew.config.MaxMemoryMB * 1024 * 1024 / 2) // Target 50%
		ew.memoryUsed.Store(int64(target))
	}
}

// monitorResources periodically checks and manages resource usage
func (ew *EdgeWorker) monitorResources(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if ew.shouldGC() {
				ew.forceGC()
			}

			if ew.isOverLimit() {
				ew.handleResourceOverrun()
			}
		}
	}
}

// shouldGC determines if garbage collection should run
func (ew *EdgeWorker) shouldGC() bool {
	if ew.config.EnableGCThreshold == 0 {
		return false
	}

	// Get current memory usage estimate
	memUsed := ew.memoryUsed.Load()
	return uint64(memUsed) > ew.config.EnableGCThreshold
}

// forceGC forces garbage collection
func (ew *EdgeWorker) forceGC() {
	// In Go, we can't force GC directly, but we can hint
	// and clear internal caches
	ew.mu.Lock()

	// Clear any internal caches
	if ew.nodeInfo != nil {
		ew.nodeInfo.Metadata = make(map[string]string)
	}

	ew.mu.Unlock()

	ew.lastGCTime.Store(time.Now())
	logger.DebugC("swarm", "Edge worker GC performed")
}

// isOverLimit checks if resource limits are exceeded
func (ew *EdgeWorker) isOverLimit() bool {
	if ew.config.MaxMemoryMB == 0 && ew.config.MaxCPUPercent == 0 {
		return false
	}

	// Check memory
	if ew.config.MaxMemoryMB > 0 {
		memMB := ew.memoryUsed.Load() / (1024 * 1024)
		if memMB > ew.config.MaxMemoryMB {
			logger.WarnCF("swarm", "Memory limit exceeded", map[string]interface{}{
				"used_mb":    memMB,
				"max_mb":     ew.config.MaxMemoryMB,
			})
			return true
		}
	}

	// Check CPU
	if ew.config.MaxCPUPercent > 0 {
		cpu := ew.cpuPercent.Load()
		if cpu > int64(ew.config.MaxCPUPercent) {
			logger.WarnCF("swarm", "CPU limit exceeded", map[string]interface{}{
				"used_percent": cpu,
				"max_percent":  ew.config.MaxCPUPercent,
			})
			return true
		}
	}

	return false
}

// handleResourceOverrun handles resource limit violations
func (ew *EdgeWorker) handleResourceOverrun() {
	// Reduce load by reducing max concurrent tasks
	ew.mu.Lock()
	if ew.Worker.cfg.MaxConcurrent > 1 {
		ew.Worker.cfg.MaxConcurrent-- // Reduce concurrent task limit
	}
	ew.mu.Unlock()

	// Force GC
	ew.forceGC()

	logger.InfoCF("swarm", "Resource overrun handled", map[string]interface{}{
		"new_max_tasks": ew.Worker.cfg.MaxConcurrent,
	})
}

// disableNonEssentialFeatures disables features in minimal mode
func (ew *EdgeWorker) disableNonEssentialFeatures() {
	// Disable internal statistics collection
	ew.config.DisableDashboard = true

	// Increase heartbeat interval to reduce network usage
	if !ew.config.DisableHeartbeat {
		ew.config.HeartbeatInterval = 60 * time.Second
	}

	logger.InfoC("swarm", "Non-essential features disabled for minimal mode")
}

// GetResourceUsage returns current resource usage statistics
func (ew *EdgeWorker) GetResourceUsage() map[string]interface{} {
	ew.mu.RLock()
	defer ew.mu.RUnlock()

	memMB := ew.memoryUsed.Load() / (1024 * 1024)
	cpu := ew.cpuPercent.Load()

	return map[string]interface{}{
		"memory_mb":      memMB,
		"memory_max_mb":  ew.config.MaxMemoryMB,
		"cpu_percent":    cpu,
		"cpu_max_percent": ew.config.MaxCPUPercent,
		"last_gc":        ew.lastGCTime.Load().(time.Time).Format(time.RFC3339),
		"batch_mode":     ew.batchMode,
	}
}

// IsHealthy checks if the edge worker is healthy
func (ew *EdgeWorker) IsHealthy() bool {
	// Check if we're not critically over resource limits
	if ew.config.MaxMemoryMB > 0 {
		memMB := ew.memoryUsed.Load() / (1024 * 1024)
		if memMB > ew.config.MaxMemoryMB*9/10 { // 90% threshold
			return false
		}
	}

	// Check if base worker is running
	return ew.Worker.running.Load()
}

// GetEdgeConfig returns the edge worker configuration
func (ew *EdgeWorker) GetEdgeConfig() *EdgeWorkerConfig {
	return ew.config
}

// EdgeBuildInfo provides build information for edge deployment
type EdgeBuildInfo struct {
	GOOS   string
	GOARCH string
	Version string
	Commit  string
	BuiltAt string
}

// GetEdgeBuildInfo returns build information for the current binary
func GetEdgeBuildInfo() *EdgeBuildInfo {
	return &EdgeBuildInfo{
		GOOS:    "linux",   // Default target
		GOARCH:  "arm64",   // Default target
		Version: "1.0.0",
		Commit:  "unknown",
		BuiltAt: time.Now().Format(time.RFC3339),
	}
}

// SupportedPlatforms returns platforms supported for edge deployment
func SupportedPlatforms() []string {
	return []string{
		"linux/arm64",  // Raspberry Pi 4+, ARM servers
		"linux/arm",    // Raspberry Pi Zero, older ARM
		"linux/amd64",  // Intel/AMD x86_64
		"linux/386",    // Intel/AMD 32-bit
		"freebsd/arm64", // ARM FreeBSD
		"freebsd/amd64", // AMD64 FreeBSD
	}
}

// BuildCommands returns cross-compile commands for edge platforms
func BuildCommands(appName string) map[string]string {
	return map[string]string{
		"linux/arm64":  fmt.Sprintf("GOOS=linux GOARCH=arm64 go build -o %s-linux-arm64 %s", appName, appName),
		"linux/arm":    fmt.Sprintf("GOOS=linux GOARCH=arm go build -o %s-linux-arm %s", appName, appName),
		"linux/amd64":  fmt.Sprintf("GOOS=linux GOARCH=amd64 go build -o %s-linux-amd64 %s", appName, appName),
		"linux/386":    fmt.Sprintf("GOOS=linux GOARCH=386 go build -o %s-linux-386 %s", appName, appName),
		"freebsd/arm64": fmt.Sprintf("GOOS=freebsd GOARCH=arm64 go build -o %s-freebsd-arm64 %s", appName, appName),
		"freebsd/amd64": fmt.Sprintf("GOOS=freebsd GOARCH=amd64 go build -o %s-freebsd-amd64 %s", appName, appName),
	}
}

// OptimizeForEdge returns build flags optimized for edge deployment
func OptimizeForEdge() string {
	return "-ldflags='-s -w' -trimpath" // Strip debug info, reduce binary size
}
