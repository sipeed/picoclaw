// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultEdgeWorkerConfig(t *testing.T) {
	cfg := DefaultEdgeWorkerConfig()

	assert.Equal(t, int64(50), cfg.MaxMemoryMB)
	assert.Equal(t, 50, cfg.MaxCPUPercent)
	assert.Equal(t, uint64(10*1024*1024), cfg.EnableGCThreshold)
	assert.False(t, cfg.DisableHeartbeat)
	assert.Equal(t, 30*time.Second, cfg.HeartbeatInterval)
	assert.Equal(t, 2, cfg.CompressionLevel)
	assert.True(t, cfg.DisableWorkflow)
	assert.True(t, cfg.DisableDashboard)
	assert.False(t, cfg.MinimalMode)
}

func TestEdgeWorker_GetResourceUsage(t *testing.T) {
	cfg := &EdgeWorkerConfig{
		MaxMemoryMB:       100,
		MaxCPUPercent:     75,
		EnableGCThreshold: 20 * 1024 * 1024,
	}

	ew := &EdgeWorker{
		config: cfg,
	}
	ew.memoryUsed.Store(50 * 1024 * 1024) // 50MB
	ew.cpuPercent.Store(25)
	ew.lastGCTime.Store(time.Now())

	usage := ew.GetResourceUsage()

	assert.Equal(t, int64(50), usage["memory_mb"])
	assert.Equal(t, int64(100), usage["memory_max_mb"])
	assert.Equal(t, int64(25), usage["cpu_percent"])
	assert.Equal(t, 75, usage["cpu_max_percent"])
}

func TestEdgeWorker_IsHealthy(t *testing.T) {
	cfg := &EdgeWorkerConfig{
		MaxMemoryMB: 100,
	}

	ew := &EdgeWorker{
		config: cfg,
		Worker: &Worker{
			running: atomic.Bool{},
		},
	}
	ew.Worker.running.Store(true)

	// Healthy when under limit
	ew.memoryUsed.Store(50 * 1024 * 1024) // 50MB < 100MB
	assert.True(t, ew.IsHealthy())

	// Unhealthy when over 90% limit
	ew.memoryUsed.Store(95 * 1024 * 1024) // 95MB > 90MB threshold
	assert.False(t, ew.IsHealthy())
}

func TestEdgeWorker_shouldGC(t *testing.T) {
	tests := []struct {
		name           string
		threshold      uint64
		memoryUsed     int64
		expectedResult bool
	}{
		{
			name:           "under threshold",
			threshold:      10 * 1024 * 1024,
			memoryUsed:     5 * 1024 * 1024,
			expectedResult: false,
		},
		{
			name:           "over threshold",
			threshold:      10 * 1024 * 1024,
			memoryUsed:     15 * 1024 * 1024,
			expectedResult: true,
		},
		{
			name:           "no threshold set",
			threshold:      0,
			memoryUsed:     100 * 1024 * 1024,
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ew := &EdgeWorker{
				config: &EdgeWorkerConfig{
					EnableGCThreshold: tt.threshold,
				},
			}
			ew.memoryUsed.Store(tt.memoryUsed)

			result := ew.shouldGC()
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestEdgeWorker_isOverLimit(t *testing.T) {
	tests := []struct {
		name           string
		maxMemoryMB     int64
		memoryUsed     int64
		maxCPUPercent   int
		cpuPercent     int64
		expectedResult bool
	}{
		{
			name:           "under all limits",
			maxMemoryMB:    100,
			memoryUsed:     50 * 1024 * 1024,
			maxCPUPercent:  80,
			cpuPercent:     40,
			expectedResult: false,
		},
		{
			name:           "over memory limit",
			maxMemoryMB:    100,
			memoryUsed:     150 * 1024 * 1024,
			maxCPUPercent:  80,
			cpuPercent:     40,
			expectedResult: true,
		},
		{
			name:           "over CPU limit",
			maxMemoryMB:    100,
			memoryUsed:     50 * 1024 * 1024,
			maxCPUPercent:  80,
			cpuPercent:     90,
			expectedResult: true,
		},
		{
			name:           "no limits set",
			maxMemoryMB:    0,
			memoryUsed:     1000 * 1024 * 1024,
			maxCPUPercent:  0,
			cpuPercent:     100,
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ew := &EdgeWorker{
				config: &EdgeWorkerConfig{
					MaxMemoryMB:   tt.maxMemoryMB,
					MaxCPUPercent: tt.maxCPUPercent,
				},
			}
			ew.memoryUsed.Store(tt.memoryUsed)
			ew.cpuPercent.Store(tt.cpuPercent)

			result := ew.isOverLimit()
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestEdgeWorker_GetEdgeConfig(t *testing.T) {
	cfg := &EdgeWorkerConfig{
		MaxMemoryMB:   200,
		MinimalMode:   true,
	}

	ew := &EdgeWorker{
		config: cfg,
	}

	assert.Equal(t, cfg, ew.GetEdgeConfig())
	assert.Same(t, cfg, ew.GetEdgeConfig())
}

func TestGetEdgeBuildInfo(t *testing.T) {
	info := GetEdgeBuildInfo()

	assert.Equal(t, "linux", info.GOOS)
	assert.Equal(t, "arm64", info.GOARCH)
	assert.Equal(t, "1.0.0", info.Version)
	assert.NotEmpty(t, info.BuiltAt)
}

func TestSupportedPlatforms(t *testing.T) {
	platforms := SupportedPlatforms()

	assert.Contains(t, platforms, "linux/arm64")
	assert.Contains(t, platforms, "linux/arm")
	assert.Contains(t, platforms, "linux/amd64")
	assert.Contains(t, platforms, "linux/386")
	assert.Contains(t, platforms, "freebsd/arm64")
	assert.Contains(t, platforms, "freebsd/amd64")
}

func TestBuildCommands(t *testing.T) {
	commands := BuildCommands("picoclaw")

	// Check that key platforms have build commands
	assert.Contains(t, commands, "linux/arm64")
	assert.Contains(t, commands, "linux/amd64")
	assert.Contains(t, commands, "freebsd/arm64")

	// Verify command format
	arm64Cmd := commands["linux/arm64"]
	assert.Contains(t, arm64Cmd, "GOOS=linux")
	assert.Contains(t, arm64Cmd, "GOARCH=arm64")
	assert.Contains(t, arm64Cmd, "go build")
	assert.Contains(t, arm64Cmd, "picoclaw-linux-arm64")
}

func TestOptimizeForEdge(t *testing.T) {
	flags := OptimizeForEdge()

	assert.Contains(t, flags, "-ldflags")
	assert.Contains(t, flags, "-s -w")
	assert.Contains(t, flags, "-trimpath")
}

func TestEdgeWorker_StopIdempotent(t *testing.T) {
	// Create a minimal Worker that won't panic on Stop
	worker := &Worker{
		taskQueue: make(chan *SwarmTask),
	}
	worker.running.Store(true)

	ew := &EdgeWorker{
		Worker: worker,
		config: DefaultEdgeWorkerConfig(),
	}

	// First stop should work
	ew.Stop()

	// Second stop should be no-op
	ew.Stop()

	assert.True(t, ew.edgeStop.Load())
}

func TestDefaultEdgeWorkerConfig_CompressionLevels(t *testing.T) {
	cfg := DefaultEdgeWorkerConfig()

	// Verify compression is enabled but not maximum
	assert.Greater(t, cfg.CompressionLevel, 0)
	assert.Less(t, cfg.CompressionLevel, 9)
}

func TestEdgeWorkerConfig_DefaultsAreReasonable(t *testing.T) {
	cfg := DefaultEdgeWorkerConfig()

	// Memory limit should be reasonable for edge devices (10-100MB)
	assert.GreaterOrEqual(t, cfg.MaxMemoryMB, int64(10))
	assert.LessOrEqual(t, cfg.MaxMemoryMB, int64(100))

	// CPU limit should be reasonable (10-100%)
	assert.GreaterOrEqual(t, cfg.MaxCPUPercent, 10)
	assert.LessOrEqual(t, cfg.MaxCPUPercent, 100)

	// Heartbeat should be less frequent than default
	assert.GreaterOrEqual(t, cfg.HeartbeatInterval, 30*time.Second)
}
