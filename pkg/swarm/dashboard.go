// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// Dashboard provides a text-based UI for monitoring swarm status
type Dashboard struct {
	manager   *Manager
	stopChan  chan struct{}
	mu        sync.RWMutex
	enabled   bool
	refresh   time.Duration
	lastState *DashboardState
}

// DashboardState represents the current state of the swarm
type DashboardState struct {
	Timestamp      int64
	ThisNode       *NodeInfoSnapshot
	Nodes          []*NodeInfoSnapshot
	Connections    *ConnectionStatus
	Stats          *SwarmStats
	LeaderStatus   *LeaderInfo
}

// NodeInfoSnapshot is a serializable snapshot of NodeInfo
type NodeInfoSnapshot struct {
	ID           string
	Role         string
	Status       string
	Capabilities []string
	Model        string
	Load         float64
	TasksRunning int
	MaxTasks     int
	LastSeen     int64
	StartedAt    int64
	Uptime       string
}

// ConnectionStatus shows connection states
type ConnectionStatus struct {
	NATSConnected    bool
	TemporalConnected bool
	EmbeddedNATS     bool
	NATSURL          string
}

// SwarmStats provides aggregate statistics
type SwarmStats struct {
	TotalNodes      int
	OnlineNodes     int
	OfflineNodes    int
	CoordinatorCount int
	WorkerCount      int
	SpecialistCount  int
	TotalCapacity   int
	UsedCapacity    int
}

// LeaderInfo shows election status
type LeaderInfo struct {
	Enabled     bool
	IsLeader    bool
	LeaderID    string
	LeaseExpiry int64
}

// NewDashboard creates a new dashboard
func NewDashboard(manager *Manager) *Dashboard {
	return &Dashboard{
		manager:  manager,
		stopChan: make(chan struct{}),
		refresh:  2 * time.Second,
	}
}

// SetRefreshInterval sets the dashboard refresh interval
func (d *Dashboard) SetRefreshInterval(interval time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.refresh = interval
}

// Start begins the dashboard update loop
func (d *Dashboard) Start(ctx context.Context) error {
	d.mu.Lock()
	d.enabled = true
	d.mu.Unlock()

	// Initial state
	d.updateState()

	logger.InfoC("swarm", "Dashboard started")

	// Start background update loop
	go d.runLoop(ctx)

	return nil
}

// Stop stops the dashboard
func (d *Dashboard) Stop() {
	d.mu.Lock()
	if d.enabled {
		close(d.stopChan)
		d.enabled = false
	}
	d.mu.Unlock()
	logger.InfoC("swarm", "Dashboard stopped")
}

// GetState returns the current dashboard state
func (d *Dashboard) GetState() *DashboardState {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.lastState
}

// runLoop runs the background update loop
func (d *Dashboard) runLoop(ctx context.Context) {
	ticker := time.NewTicker(d.refresh)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			d.updateState()
		case <-d.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// updateState updates the dashboard state from the manager
func (d *Dashboard) updateState() {
	d.mu.Lock()
	defer d.mu.Unlock()

	state := &DashboardState{
		Timestamp: time.Now().UnixMilli(),
	}

	// Get this node info
	if d.manager != nil && d.manager.nodeInfo != nil {
		state.ThisNode = snapshotNodeInfo(d.manager.nodeInfo)
	}

	// Get discovered nodes
	if d.manager != nil && d.manager.discovery != nil {
		nodes := d.manager.discovery.GetAllNodes()
		state.Nodes = make([]*NodeInfoSnapshot, 0, len(nodes))
		for _, node := range nodes {
			state.Nodes = append(state.Nodes, snapshotNodeInfo(node))
		}
	}

	// Get connection status
	if d.manager != nil {
		state.Connections = &ConnectionStatus{
			NATSConnected:     d.manager.IsNATSConnected(),
			TemporalConnected: d.manager.IsTemporalConnected(),
		}
		if d.manager.embeddedNATS != nil {
			state.Connections.EmbeddedNATS = true
			state.Connections.NATSURL = d.manager.embeddedNATS.ClientURL()
		}

		// Get leader status
		state.LeaderStatus = &LeaderInfo{
			Enabled:  d.manager.electionMgr != nil,
			IsLeader: d.manager.IsLeader(),
			LeaderID: d.manager.GetLeaderID(),
		}
	}

	// Calculate statistics
	state.Stats = calculateStats(state.Nodes)

	d.lastState = state
}

// snapshotNodeInfo creates a snapshot of NodeInfo
func snapshotNodeInfo(node *NodeInfo) *NodeInfoSnapshot {
	now := time.Now().UnixMilli()
	uptime := time.Duration(now - node.StartedAt)

	return &NodeInfoSnapshot{
		ID:           node.ID,
		Role:         string(node.Role),
		Status:       string(node.Status),
		Capabilities: node.Capabilities,
		Model:        node.Model,
		Load:         node.Load,
		TasksRunning: node.TasksRunning,
		MaxTasks:     node.MaxTasks,
		LastSeen:     node.LastSeen,
		StartedAt:    node.StartedAt,
		Uptime:       formatUptime(uptime),
	}
}

// calculateStats calculates aggregate statistics
func calculateStats(nodes []*NodeInfoSnapshot) *SwarmStats {
	stats := &SwarmStats{
		TotalNodes: len(nodes),
	}

	for _, node := range nodes {
		switch node.Status {
		case "online", "busy":
			stats.OnlineNodes++
		case "offline":
			stats.OfflineNodes++
		}

		switch node.Role {
		case "coordinator":
			stats.CoordinatorCount++
		case "worker":
			stats.WorkerCount++
		case "specialist":
			stats.SpecialistCount++
		}

		stats.TotalCapacity += node.MaxTasks
		stats.UsedCapacity += node.TasksRunning
	}

	return stats
}

// formatUptime formats a duration as a human-readable uptime
func formatUptime(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
	}
	return fmt.Sprintf("%dd%dh", int(d.Hours()/24), int(d.Hours())%24)
}

// Render returns a formatted string representation of the dashboard
func (d *Dashboard) Render() string {
	d.mu.RLock()
	state := d.lastState
	d.mu.RUnlock()

	if state == nil {
		return "Dashboard not initialized"
	}

	var sb strings.Builder

	// Header
	sb.WriteString("\n")
	sb.WriteString("╔════════════════════════════════════════════════════════════╗\n")
	sb.WriteString("║           PicoClaw Swarm Status Dashboard                  ║\n")
	sb.WriteString("╚════════════════════════════════════════════════════════════╝\n")
	sb.WriteString("\n")

	// This Node
	if state.ThisNode != nil {
		sb.WriteString("【This Node】\n")
		sb.WriteString(fmt.Sprintf("  ID:    %s\n", state.ThisNode.ID))
		sb.WriteString(fmt.Sprintf("  Role:  %s\n", formatRole(state.ThisNode.Role)))
		sb.WriteString(fmt.Sprintf("  Status: %s\n", formatStatus(state.ThisNode.Status)))
		sb.WriteString(fmt.Sprintf("  Load:  %s\n", formatLoadBar(state.ThisNode.Load, state.ThisNode.TasksRunning, state.ThisNode.MaxTasks)))
		sb.WriteString(fmt.Sprintf("  Uptime: %s\n", state.ThisNode.Uptime))
		sb.WriteString("\n")
	}

	// Connections
	if state.Connections != nil {
		sb.WriteString("【Connections】\n")
		sb.WriteString(fmt.Sprintf("  NATS:     %s %s\n", formatBool(state.Connections.NATSConnected), state.Connections.NATSURL))
		sb.WriteString(fmt.Sprintf("  Temporal: %s\n", formatBool(state.Connections.TemporalConnected)))
		if state.Connections.EmbeddedNATS {
			sb.WriteString("  (Embedded NATS Server)\n")
		}
		sb.WriteString("\n")
	}

	// Leader Status
	if state.LeaderStatus != nil && state.LeaderStatus.Enabled {
		sb.WriteString("【Leader Election】\n")
		sb.WriteString(fmt.Sprintf("  Enabled:  Yes\n"))
		sb.WriteString(fmt.Sprintf("  IsLeader: %s\n", formatBool(state.LeaderStatus.IsLeader)))
		sb.WriteString(fmt.Sprintf("  LeaderID: %s\n", state.LeaderStatus.LeaderID))
		if state.LeaderStatus.LeaseExpiry > 0 {
			remaining := time.Until(time.UnixMilli(state.LeaderStatus.LeaseExpiry))
			sb.WriteString(fmt.Sprintf("  Lease:    %s\n", formatDuration(remaining)))
		}
		sb.WriteString("\n")
	}

	// Statistics
	if state.Stats != nil {
		sb.WriteString("【Swarm Statistics】\n")
		sb.WriteString(fmt.Sprintf("  Nodes:      %d total, %d online, %d offline\n",
			state.Stats.TotalNodes, state.Stats.OnlineNodes, state.Stats.OfflineNodes))
		sb.WriteString(fmt.Sprintf("  Roles:      %d coordinator(s), %d worker(s), %d specialist(s)\n",
			state.Stats.CoordinatorCount, state.Stats.WorkerCount, state.Stats.SpecialistCount))
		sb.WriteString(fmt.Sprintf("  Capacity:   %d/%d tasks used\n",
			state.Stats.UsedCapacity, state.Stats.TotalCapacity))
		sb.WriteString("\n")
	}

	// Nodes List
	if len(state.Nodes) > 0 {
		sb.WriteString("【Discovered Nodes】\n")
		for _, node := range state.Nodes {
			statusIcon := getNodeStatusIcon(node.Status)
			roleIcon := getRoleIcon(node.Role)
			sb.WriteString(fmt.Sprintf("  %s %-20s %-2s %-8s %s\n",
				statusIcon,
				truncateID(node.ID),
				roleIcon,
				node.Status,
				formatLoadBar(node.Load, node.TasksRunning, node.MaxTasks),
			))
		}
		sb.WriteString("\n")
	}

	// Legend
	sb.WriteString("【Legend】\n")
	sb.WriteString("  ● = Online  ○ = Offline  ? = Unknown  ◐ = Suspicious\n")
	sb.WriteString("  C = Coordinator  W = Worker  S = Specialist\n")
	sb.WriteString(fmt.Sprintf("  Updated: %s\n", time.UnixMilli(state.Timestamp).Format("15:04:05")))

	return sb.String()
}

// RenderCompact returns a compact one-line status
func (d *Dashboard) RenderCompact() string {
	d.mu.RLock()
	state := d.lastState
	d.mu.RUnlock()

	if state == nil {
		return "Swarm: initializing"
	}

	var parts []string

	if state.ThisNode != nil {
		parts = append(parts, fmt.Sprintf("%s:%s", state.ThisNode.Role, state.ThisNode.Status))
	}

	if state.Stats != nil {
		parts = append(parts, fmt.Sprintf("%d/%d nodes", state.Stats.OnlineNodes, state.Stats.TotalNodes))
	}

	if state.Connections != nil {
		if !state.Connections.NATSConnected {
			parts = append(parts, "NATS:down")
		}
	}

	if state.LeaderStatus != nil && state.LeaderStatus.Enabled {
		if state.LeaderStatus.IsLeader {
			parts = append(parts, "LEADER")
		} else if state.LeaderStatus.LeaderID != "" {
			parts = append(parts, fmt.Sprintf("leader:%s", truncateID(state.LeaderStatus.LeaderID)))
		}
	}

	return fmt.Sprintf("Swarm[%s]", strings.Join(parts, " "))
}

// formatRole formats a role with an icon
func formatRole(role string) string {
	switch role {
	case "coordinator":
		return "📋 " + role
	case "worker":
		return "⚙️ " + role
	case "specialist":
		return "🔧 " + role
	default:
		return role
	}
}

// formatStatus formats a status with an icon
func formatStatus(status string) string {
	switch status {
	case "online":
		return "● " + status
	case "busy":
		return "🔄 " + status
	case "offline":
		return "○ " + status
	case "suspicious":
		return "◐ " + status
	default:
		return "? " + status
	}
}

// formatBool formats a boolean as Yes/No
func formatBool(b bool) string {
	if b {
		return "✓ Yes"
	}
	return "✗ No"
}

// formatLoadBar creates a visual load bar
func formatLoadBar(load float64, tasksRunning, maxTasks int) string {
	width := 10
	filled := int(load * float64(width))
	if filled > width {
		filled = width
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return fmt.Sprintf("[%s] %.0f%% (%d/%d)", bar, load*100, tasksRunning, maxTasks)
}

// formatDuration formats a duration for display
func formatDuration(d time.Duration) string {
	if d < 0 {
		return "expired"
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
}

// getNodeStatusIcon returns an icon for node status
func getNodeStatusIcon(status string) string {
	switch status {
	case "online":
		return "●"
	case "busy":
		return "🔄"
	case "offline":
		return "○"
	case "suspicious":
		return "◐"
	default:
		return "?"
	}
}

// getRoleIcon returns an icon for role
func getRoleIcon(role string) string {
	switch role {
	case "coordinator":
		return "C"
	case "worker":
		return "W"
	case "specialist":
		return "S"
	default:
		return "?"
	}
}

// truncateID truncates an ID for display
func truncateID(id string) string {
	if len(id) <= 20 {
		return id
	}
	return id[:17] + "..."
}

// GetStatusSummary returns a quick status summary for logging
func (d *Dashboard) GetStatusSummary() map[string]interface{} {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.lastState == nil {
		return map[string]interface{}{
			"status": "initializing",
		}
	}

	summary := map[string]interface{}{
		"timestamp":    d.lastState.Timestamp,
		"total_nodes":  d.lastState.Stats.TotalNodes,
		"online_nodes": d.lastState.Stats.OnlineNodes,
		"nats":         d.lastState.Connections.NATSConnected,
		"temporal":     d.lastState.Connections.TemporalConnected,
	}

	if d.lastState.ThisNode != nil {
		summary["node_id"] = d.lastState.ThisNode.ID
		summary["role"] = d.lastState.ThisNode.Role
	}

	if d.lastState.LeaderStatus != nil && d.lastState.LeaderStatus.Enabled {
		summary["is_leader"] = d.lastState.LeaderStatus.IsLeader
	}

	return summary
}
