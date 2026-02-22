// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"testing"
	"time"
)

func TestDiscovery_NodeRegistration(t *testing.T) {
	tests := []struct {
		name    string
		selfID  string
		nodes   []*NodeInfo
		wantCnt int
	}{
		{
			name:   "register single node",
			selfID: "self-1",
			nodes: []*NodeInfo{
				newTestNodeInfo("other-1", RoleWorker, []string{"code"}, 4),
			},
			wantCnt: 1,
		},
		{
			name:   "register multiple nodes",
			selfID: "self-2",
			nodes: []*NodeInfo{
				newTestNodeInfo("other-a", RoleWorker, []string{"code"}, 4),
				newTestNodeInfo("other-b", RoleSpecialist, []string{"ml"}, 2),
				newTestNodeInfo("other-c", RoleWorker, []string{"research"}, 4),
			},
			wantCnt: 3,
		},
		{
			name:   "skip self registration",
			selfID: "self-3",
			nodes: []*NodeInfo{
				newTestNodeInfo("self-3", RoleWorker, []string{"code"}, 4), // same as self
				newTestNodeInfo("other-d", RoleWorker, []string{"code"}, 4),
			},
			wantCnt: 1, // self-3 should be skipped
		},
		{
			name:   "duplicate registration overwrites",
			selfID: "self-4",
			nodes: []*NodeInfo{
				{ID: "dup-1", Role: RoleWorker, Capabilities: []string{"code"}, Status: StatusOnline, Load: 0.1, MaxTasks: 4},
				{ID: "dup-1", Role: RoleWorker, Capabilities: []string{"code"}, Status: StatusBusy, Load: 0.9, MaxTasks: 4},
			},
			wantCnt: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selfNode := newTestNodeInfo(tt.selfID, RoleCoordinator, nil, 1)
			cfg := newTestSwarmConfig(0)
			d := NewDiscovery(nil, selfNode, cfg) // bridge not needed for direct handleNodeJoin

			for _, node := range tt.nodes {
				d.handleNodeJoin(node)
			}

			if got := d.NodeCount(); got != tt.wantCnt {
				t.Errorf("NodeCount() = %d, want %d", got, tt.wantCnt)
			}

			// For duplicate test, check the final state
			if tt.name == "duplicate registration overwrites" {
				node, ok := d.GetNode("dup-1")
				if !ok {
					t.Fatal("GetNode('dup-1') returned false")
				}
				if node.Status != StatusBusy {
					t.Errorf("Status = %q, want %q (overwritten)", node.Status, StatusBusy)
				}
			}
		})
	}
}

func TestDiscovery_HeartbeatUpdatesNode(t *testing.T) {
	tests := []struct {
		name         string
		selfID       string
		registerNode *NodeInfo
		heartbeat    Heartbeat
		expectUpdate bool
		wantStatus   NodeStatus
		wantLoad     float64
		wantTasks    int
	}{
		{
			name:         "update load",
			selfID:       "hb-self-1",
			registerNode: &NodeInfo{ID: "hb-node-1", Role: RoleWorker, Status: StatusOnline, Load: 0.1, MaxTasks: 4},
			heartbeat:    Heartbeat{NodeID: "hb-node-1", Status: StatusOnline, Load: 0.7, TasksRunning: 3, Timestamp: time.Now().UnixMilli()},
			expectUpdate: true,
			wantStatus:   StatusOnline,
			wantLoad:     0.7,
			wantTasks:    3,
		},
		{
			name:         "update status to busy",
			selfID:       "hb-self-2",
			registerNode: &NodeInfo{ID: "hb-node-2", Role: RoleWorker, Status: StatusOnline, Load: 0.5, MaxTasks: 4},
			heartbeat:    Heartbeat{NodeID: "hb-node-2", Status: StatusBusy, Load: 1.0, TasksRunning: 4, Timestamp: time.Now().UnixMilli()},
			expectUpdate: true,
			wantStatus:   StatusBusy,
			wantLoad:     1.0,
			wantTasks:    4,
		},
		{
			name:         "skip own heartbeat",
			selfID:       "hb-self-3",
			registerNode: nil, // don't register anything
			heartbeat:    Heartbeat{NodeID: "hb-self-3", Status: StatusOnline, Load: 0.5, Timestamp: time.Now().UnixMilli()},
			expectUpdate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selfNode := newTestNodeInfo(tt.selfID, RoleCoordinator, nil, 1)
			cfg := newTestSwarmConfig(0)
			d := NewDiscovery(nil, selfNode, cfg)

			if tt.registerNode != nil {
				d.handleNodeJoin(tt.registerNode)
			}

			d.handleHeartbeat(&tt.heartbeat)

			if tt.expectUpdate {
				node, ok := d.GetNode(tt.heartbeat.NodeID)
				if !ok {
					t.Fatal("GetNode() returned false after heartbeat")
				}
				if node.Status != tt.wantStatus {
					t.Errorf("Status = %q, want %q", node.Status, tt.wantStatus)
				}
				if node.Load != tt.wantLoad {
					t.Errorf("Load = %f, want %f", node.Load, tt.wantLoad)
				}
				if node.TasksRunning != tt.wantTasks {
					t.Errorf("TasksRunning = %d, want %d", node.TasksRunning, tt.wantTasks)
				}
			}
		})
	}
}

func TestDiscovery_SelectWorker(t *testing.T) {
	tests := []struct {
		name       string
		nodes      []*NodeInfo
		capability string
		wantID     string // empty means nil expected
	}{
		{
			name: "picks lowest load",
			nodes: []*NodeInfo{
				{ID: "w-a", Role: RoleWorker, Capabilities: []string{"code"}, Status: StatusOnline, Load: 0.3, TasksRunning: 1, MaxTasks: 4},
				{ID: "w-b", Role: RoleWorker, Capabilities: []string{"code"}, Status: StatusOnline, Load: 0.1, TasksRunning: 0, MaxTasks: 4},
			},
			capability: "code",
			wantID:     "w-b",
		},
		{
			name: "skips full worker",
			nodes: []*NodeInfo{
				{ID: "w-full", Role: RoleWorker, Capabilities: []string{"code"}, Status: StatusOnline, Load: 1.0, TasksRunning: 2, MaxTasks: 2},
				{ID: "w-avail", Role: RoleWorker, Capabilities: []string{"code"}, Status: StatusOnline, Load: 0.5, TasksRunning: 2, MaxTasks: 4},
			},
			capability: "code",
			wantID:     "w-avail",
		},
		{
			name: "no workers with capability",
			nodes: []*NodeInfo{
				{ID: "w-research", Role: RoleWorker, Capabilities: []string{"research"}, Status: StatusOnline, Load: 0.1, MaxTasks: 4},
			},
			capability: "code",
			wantID:     "",
		},
		{
			name: "falls back to specialist",
			nodes: []*NodeInfo{
				{ID: "s-code", Role: RoleSpecialist, Capabilities: []string{"code"}, Status: StatusOnline, Load: 0.2, TasksRunning: 0, MaxTasks: 2},
			},
			capability: "code",
			wantID:     "s-code",
		},
		{
			name: "all workers full",
			nodes: []*NodeInfo{
				{ID: "w-f1", Role: RoleWorker, Capabilities: []string{"code"}, Status: StatusBusy, Load: 1.0, TasksRunning: 1, MaxTasks: 1},
				{ID: "w-f2", Role: RoleWorker, Capabilities: []string{"code"}, Status: StatusBusy, Load: 1.0, TasksRunning: 1, MaxTasks: 1},
			},
			capability: "code",
			wantID:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selfNode := newTestNodeInfo("select-self", RoleCoordinator, nil, 1)
			cfg := newTestSwarmConfig(0)
			d := NewDiscovery(nil, selfNode, cfg)

			for _, node := range tt.nodes {
				d.handleNodeJoin(node)
			}

			got := d.SelectWorker(tt.capability)

			if tt.wantID == "" {
				if got != nil {
					t.Errorf("SelectWorker() = %q, want nil", got.ID)
				}
			} else {
				if got == nil {
					t.Fatalf("SelectWorker() = nil, want %q", tt.wantID)
				}
				if got.ID != tt.wantID {
					t.Errorf("SelectWorker().ID = %q, want %q", got.ID, tt.wantID)
				}
			}
		})
	}
}

func TestDiscovery_StaleNodeCleanup(t *testing.T) {
	tests := []struct {
		name       string
		lastSeen   int64 // milliseconds ago
		initStatus NodeStatus
		wantStatus NodeStatus
		wantExists bool  // whether node should still exist after cleanup
	}{
		{
			name:       "stale node marked offline",
			lastSeen:   300, // 300ms ago, > timeout (200ms) but < GC threshold (2s)
			initStatus: StatusOnline,
			wantStatus: StatusOffline,
			wantExists: true,
		},
		{
			name:       "fresh node untouched",
			lastSeen:   10, // 10ms ago, well within timeout
			initStatus: StatusOnline,
			wantStatus: StatusOnline,
			wantExists: true,
		},
		{
			name:       "already offline node unchanged",
			lastSeen:   300, // 300ms ago, > timeout but < GC threshold
			initStatus: StatusOffline,
			wantStatus: StatusOffline,
			wantExists: true,
		},
		{
			name:       "long dead node GC'd",
			lastSeen:   5000, // 5s ago, > GC threshold (2s)
			initStatus: StatusOffline,
			wantExists: false, // should be removed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selfNode := newTestNodeInfo("cleanup-self", RoleCoordinator, nil, 1)
			cfg := newTestSwarmConfig(0) // NodeTimeout is 200ms
			d := NewDiscovery(nil, selfNode, cfg)

			node := &NodeInfo{
				ID:       "cleanup-node",
				Role:     RoleWorker,
				Status:   tt.initStatus,
				MaxTasks: 4,
				LastSeen: time.Now().UnixMilli() - tt.lastSeen,
			}
			// Directly insert into registry (bypass handleNodeJoin which sets LastSeen)
			d.mu.Lock()
			d.registry["cleanup-node"] = node
			d.mu.Unlock()

			d.cleanupStaleNodes()

			got, ok := d.GetNode("cleanup-node")
			if tt.wantExists {
				if !ok {
					t.Fatal("GetNode() returned false, expected node to exist")
				}
				if got.Status != tt.wantStatus {
					t.Errorf("Status = %q, want %q", got.Status, tt.wantStatus)
				}
			} else {
				if ok {
					t.Errorf("GetNode() returned true, expected node to be GC'd")
				}
			}
		})
	}
}

func TestDiscovery_GetNodesFiltering(t *testing.T) {
	// Set up a discovery with mixed nodes
	selfNode := newTestNodeInfo("filter-self", RoleCoordinator, nil, 1)
	cfg := newTestSwarmConfig(0)
	d := NewDiscovery(nil, selfNode, cfg)

	nodes := []*NodeInfo{
		{ID: "f-w1", Role: RoleWorker, Capabilities: []string{"code", "research"}, Status: StatusOnline, MaxTasks: 4},
		{ID: "f-w2", Role: RoleWorker, Capabilities: []string{"code"}, Status: StatusOnline, MaxTasks: 4},
		{ID: "f-s1", Role: RoleSpecialist, Capabilities: []string{"ml"}, Status: StatusOnline, MaxTasks: 2},
		{ID: "f-off", Role: RoleWorker, Capabilities: []string{"code"}, Status: StatusOffline, MaxTasks: 4},
	}
	for _, n := range nodes {
		d.mu.Lock()
		d.registry[n.ID] = n
		d.mu.Unlock()
	}

	tests := []struct {
		name       string
		role       NodeRole
		capability string
		wantCount  int
	}{
		{
			name:       "no filter returns all online",
			role:       "",
			capability: "",
			wantCount:  3, // f-w1, f-w2, f-s1 (f-off excluded)
		},
		{
			name:       "filter by role worker",
			role:       RoleWorker,
			capability: "",
			wantCount:  2, // f-w1, f-w2 (f-off is offline)
		},
		{
			name:       "filter by capability code",
			role:       "",
			capability: "code",
			wantCount:  2, // f-w1, f-w2
		},
		{
			name:       "filter by role and capability",
			role:       RoleWorker,
			capability: "research",
			wantCount:  1, // f-w1
		},
		{
			name:       "offline nodes excluded",
			role:       RoleWorker,
			capability: "code",
			wantCount:  2, // f-w1, f-w2 (not f-off)
		},
		{
			name:       "filter by specialist role",
			role:       RoleSpecialist,
			capability: "",
			wantCount:  1, // f-s1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := d.GetNodes(tt.role, tt.capability)
			if len(got) != tt.wantCount {
				ids := make([]string, len(got))
				for i, n := range got {
					ids[i] = n.ID
				}
				t.Errorf("GetNodes(%q, %q) returned %d nodes %v, want %d", tt.role, tt.capability, len(got), ids, tt.wantCount)
			}
		})
	}
}

func TestDiscovery_NodeLeave(t *testing.T) {
	tests := []struct {
		name         string
		registerNode bool
		leaveID      string
		wantStatus   NodeStatus
	}{
		{
			name:         "known node leaves",
			registerNode: true,
			leaveID:      "leave-node",
			wantStatus:   StatusOffline,
		},
		{
			name:         "unknown node leave no panic",
			registerNode: false,
			leaveID:      "unknown-node",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selfNode := newTestNodeInfo("leave-self", RoleCoordinator, nil, 1)
			cfg := newTestSwarmConfig(0)
			d := NewDiscovery(nil, selfNode, cfg)

			if tt.registerNode {
				node := newTestNodeInfo("leave-node", RoleWorker, []string{"code"}, 4)
				d.handleNodeJoin(node)
			}

			// Should not panic
			d.handleNodeLeave(tt.leaveID)

			if tt.registerNode {
				node, ok := d.GetNode(tt.leaveID)
				if !ok {
					t.Fatal("GetNode() returned false for left node")
				}
				if node.Status != tt.wantStatus {
					t.Errorf("Status = %q, want %q", node.Status, tt.wantStatus)
				}
			}
		})
	}
}
