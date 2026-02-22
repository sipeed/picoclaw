// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 Picooclaw contributors

package swarm

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewDashboard(t *testing.T) {
	dash := NewDashboard(nil)
	assert.NotNil(t, dash)
	assert.False(t, dash.enabled)
	assert.Equal(t, 2*time.Second, dash.refresh)
}

func TestDashboard_SetRefreshInterval(t *testing.T) {
	dash := NewDashboard(nil)
	dash.SetRefreshInterval(5 * time.Second)
	assert.Equal(t, 5*time.Second, dash.refresh)
}

func TestDashboard_StopWithoutStart(t *testing.T) {
	dash := NewDashboard(nil)
	// Should not panic
	dash.Stop()
	assert.False(t, dash.enabled)
}

func TestDashboard_RenderEmpty(t *testing.T) {
	dash := NewDashboard(nil)
	output := dash.Render()
	assert.Contains(t, output, "Dashboard not initialized")
}

func TestDashboard_RenderCompactEmpty(t *testing.T) {
	dash := NewDashboard(nil)
	output := dash.RenderCompact()
	assert.Contains(t, output, "initializing")
}

func TestDashboard_SnapshotNodeInfo(t *testing.T) {
	node := &NodeInfo{
		ID:           "test-node-1",
		Role:         RoleWorker,
		Status:       StatusOnline,
		Capabilities: []string{"code", "research"},
		Model:        "test-model",
		Load:         0.5,
		TasksRunning: 2,
		MaxTasks:     5,
		LastSeen:     time.Now().UnixMilli(),
		StartedAt:    time.Now().UnixMilli(), // Started now
	}

	snapshot := snapshotNodeInfo(node)

	assert.Equal(t, "test-node-1", snapshot.ID)
	assert.Equal(t, "worker", snapshot.Role)
	assert.Equal(t, "online", snapshot.Status)
	assert.Equal(t, []string{"code", "research"}, snapshot.Capabilities)
	assert.Equal(t, 0.5, snapshot.Load)
	assert.Equal(t, 2, snapshot.TasksRunning)
	assert.Equal(t, 5, snapshot.MaxTasks)
	// Just started, should show seconds or 0s
	assert.Contains(t, snapshot.Uptime, "s")
}

func TestDashboard_CalculateStats(t *testing.T) {
	nodes := []*NodeInfoSnapshot{
		{Role: "coordinator", Status: "online", MaxTasks: 10, TasksRunning: 2},
		{Role: "worker", Status: "online", MaxTasks: 5, TasksRunning: 3},
		{Role: "worker", Status: "busy", MaxTasks: 5, TasksRunning: 5},
		{Role: "worker", Status: "offline", MaxTasks: 5, TasksRunning: 0},
		{Role: "specialist", Status: "online", MaxTasks: 3, TasksRunning: 1},
	}

	stats := calculateStats(nodes)

	assert.Equal(t, 5, stats.TotalNodes)
	assert.Equal(t, 4, stats.OnlineNodes) // online + busy nodes
	assert.Equal(t, 1, stats.OfflineNodes)
	assert.Equal(t, 1, stats.CoordinatorCount)
	assert.Equal(t, 3, stats.WorkerCount)
	assert.Equal(t, 1, stats.SpecialistCount)
	assert.Equal(t, 28, stats.TotalCapacity)
	assert.Equal(t, 11, stats.UsedCapacity)
}

func TestDashboard_FormatUptime(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{"seconds", 30 * time.Second, "30s"},
		{"minutes", 5 * time.Minute, "5m"},
		{"hours", 3 * time.Hour, "3h0m"},
		{"days", 2*24*time.Hour + 5*time.Hour, "2d5h"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatUptime(tt.duration)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDashboard_FormatLoadBar(t *testing.T) {
	tests := []struct {
		name         string
		load         float64
		tasksRunning int
		maxTasks     int
	}{
		{"empty", 0.0, 0, 10},
		{"half", 0.5, 5, 10},
		{"full", 1.0, 10, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatLoadBar(tt.load, tt.tasksRunning, tt.maxTasks)
			assert.Contains(t, got, "[")
			assert.Contains(t, got, "]")
			assert.Contains(t, got, "%")
			assert.Contains(t, got, "/")
		})
	}
}

func TestDashboard_GetStatusSummary(t *testing.T) {
	dash := NewDashboard(nil)
	summary := dash.GetStatusSummary()
	assert.NotNil(t, summary)
	assert.Contains(t, summary, "status")
	assert.Equal(t, "initializing", summary["status"])
}

func TestDashboard_RenderWithState(t *testing.T) {
	dash := NewDashboard(nil)

	// Manually set state
	dash.lastState = &DashboardState{
		Timestamp: time.Now().UnixMilli(),
		ThisNode: &NodeInfoSnapshot{
			ID:     "test-node",
			Role:   "worker",
			Status: "online",
			Load:   0.3,
		},
		Stats: &SwarmStats{
			TotalNodes:  3,
			OnlineNodes: 3,
		},
		Connections: &ConnectionStatus{
			NATSConnected:     true,
			TemporalConnected: false,
		},
	}

	output := dash.Render()
	assert.Contains(t, output, "PicoClaw Swarm Status")
	assert.Contains(t, output, "test-node")
	assert.Contains(t, output, "worker")
	assert.Contains(t, output, "online")
}

func TestDashboard_TruncateID(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want string
	}{
		{"short", "abc", "abc"},
		{"exact", "12345678901234567890", "12345678901234567890"},
		{"long", "12345678901234567890123", "12345678901234567..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateID(tt.id)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDashboard_RenderCompactWithState(t *testing.T) {
	dash := NewDashboard(nil)

	dash.lastState = &DashboardState{
		Timestamp: time.Now().UnixMilli(),
		ThisNode: &NodeInfoSnapshot{
			ID:     "test-node",
			Role:   "worker",
			Status: "online",
		},
		Stats: &SwarmStats{
			TotalNodes:  5,
			OnlineNodes: 4,
		},
		Connections: &ConnectionStatus{
			NATSConnected: true,
		},
	}

	output := dash.RenderCompact()
	assert.Contains(t, output, "Swarm[")
	assert.Contains(t, output, "worker:online")
	assert.Contains(t, output, "4/5")
}
