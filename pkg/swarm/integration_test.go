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
	"sync/atomic"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

func TestIntegration_CoordinatorWorkerRoundTrip(t *testing.T) {
	_, url, cleanup := startTestNATS(t)
	defer cleanup()

	tests := []struct {
		name         string
		chatResponse string
		chatErr      error
		wantStatus   string
		wantContains string
	}{
		{
			name:         "full round-trip success",
			chatResponse: "analysis complete",
			chatErr:      nil,
			wantStatus:   string(TaskDone),
			wantContains: "analysis complete",
		},
		{
			name:         "error round-trip",
			chatResponse: "",
			chatErr:      fmt.Errorf("agent crashed"),
			wantStatus:   string(TaskFailed),
			wantContains: "agent crashed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// --- Worker side ---
			workerNode := newTestNodeInfo("integ-worker", RoleWorker, []string{"code"}, 4)
			workerBridge := connectTestBridge(t, url, workerNode)
			defer workerBridge.Stop()

			if err := workerBridge.Start(context.Background()); err != nil {
				t.Fatalf("worker bridge Start() error: %v", err)
			}

			workerCfg := newTestSwarmConfig(0)
			workerCfg.MaxConcurrent = 2
			workerTemporal := NewTemporalClient(&config.TemporalConfig{TaskQueue: "test"})
			workerAgentLoop := newTestAgentLoop(t, tt.chatResponse, tt.chatErr)

			worker := NewWorker(workerCfg, workerBridge, workerTemporal, workerAgentLoop, &mockLLMProvider{}, workerNode)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			if err := worker.Start(ctx); err != nil {
				t.Fatalf("worker Start() error: %v", err)
			}

			// --- Coordinator side ---
			coordNode := newTestNodeInfo("integ-coord", RoleCoordinator, nil, 1)
			coordBridge := connectTestBridge(t, url, coordNode)
			defer coordBridge.Stop()

			if err := coordBridge.Start(context.Background()); err != nil {
				t.Fatalf("coord bridge Start() error: %v", err)
			}

			coordCfg := newTestSwarmConfig(0)
			discovery := NewDiscovery(coordBridge, coordNode, coordCfg)
			// Register the worker in coordinator's discovery
			discovery.handleNodeJoin(workerNode)

			coordTemporal := NewTemporalClient(&config.TemporalConfig{TaskQueue: "test"})
			coordAgentLoop := newTestAgentLoop(t, "local fallback", nil)
			localBus := bus.NewMessageBus()

			coordinator := NewCoordinator(coordCfg, coordBridge, coordTemporal, discovery, coordAgentLoop, &mockLLMProvider{}, localBus)

			// Give all subscriptions time to propagate
			time.Sleep(100 * time.Millisecond)

			// --- Dispatch ---
			task := &SwarmTask{
				ID:         fmt.Sprintf("integ-task-%d", time.Now().UnixNano()%100000),
				Type:       TaskTypeDirect,
				Capability: "code",
				Prompt:     "integration test prompt",
				Status:     TaskPending,
				Timeout:    5000,
			}

			result, err := coordinator.DispatchTask(ctx, task)
			if err != nil {
				t.Fatalf("DispatchTask() error: %v", err)
			}
			if result == nil {
				t.Fatal("DispatchTask() returned nil result")
			}
			if result.Status != tt.wantStatus {
				t.Errorf("Status = %q, want %q", result.Status, tt.wantStatus)
			}
			combined := result.Result + result.Error
			if !strings.Contains(combined, tt.wantContains) {
				t.Errorf("Result+Error = %q, want it to contain %q", combined, tt.wantContains)
			}
		})
	}
}

func TestIntegration_MultiNodeDiscovery(t *testing.T) {
	_, url, cleanup := startTestNATS(t)
	defer cleanup()

	type nodeSetup struct {
		id         string
		role       NodeRole
		caps       []string
		bridge     *NATSBridge
		discovery  *Discovery
	}

	nodes := []struct {
		id   string
		role NodeRole
		caps []string
	}{
		{"disc-node-a", RoleWorker, []string{"code"}},
		{"disc-node-b", RoleWorker, []string{"research"}},
		{"disc-node-c", RoleSpecialist, []string{"ml"}},
	}

	setups := make([]*nodeSetup, len(nodes))

	// Create all nodes
	for i, n := range nodes {
		nodeInfo := newTestNodeInfo(n.id, n.role, n.caps, 4)
		bridge := connectTestBridge(t, url, nodeInfo)
		defer bridge.Stop()

		cfg := newTestSwarmConfig(0)
		cfg.NATS.HeartbeatInterval = "50ms"
		cfg.NATS.NodeTimeout = "5s"

		disc := NewDiscovery(bridge, nodeInfo, cfg)

		setups[i] = &nodeSetup{
			id:        n.id,
			role:      n.role,
			caps:      n.caps,
			bridge:    bridge,
			discovery: disc,
		}
	}

	// Start all bridges and discoveries
	ctx := context.Background()
	for _, s := range setups {
		if err := s.bridge.Start(ctx); err != nil {
			t.Fatalf("bridge Start(%s) error: %v", s.id, err)
		}
	}

	// Stagger discovery starts slightly to let announce messages propagate
	for _, s := range setups {
		if err := s.discovery.Start(ctx); err != nil {
			t.Fatalf("discovery Start(%s) error: %v", s.id, err)
		}
		time.Sleep(50 * time.Millisecond)
	}
	defer func() {
		for _, s := range setups {
			s.discovery.Stop()
		}
	}()

	// Wait for all nodes to discover each other
	ok := waitFor(t, 5*time.Second, func() bool {
		for _, s := range setups {
			if s.discovery.NodeCount() < 2 {
				return false
			}
		}
		return true
	})

	if !ok {
		for _, s := range setups {
			t.Errorf("node %s discovered %d other nodes, want 2", s.id, s.discovery.NodeCount())
		}
		t.Fatal("timed out waiting for multi-node discovery")
	}

	// Verify each node knows about the other two
	for _, s := range setups {
		count := s.discovery.NodeCount()
		if count != 2 {
			t.Errorf("node %s: NodeCount() = %d, want 2", s.id, count)
		}
		allNodes := s.discovery.GetNodes("", "")
		for _, other := range setups {
			if other.id == s.id {
				continue
			}
			found := false
			for _, n := range allNodes {
				if n.ID == other.id {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("node %s: missing discovery of node %s", s.id, other.id)
			}
		}
	}
}

func TestIntegration_CapabilityRouting(t *testing.T) {
	_, url, cleanup := startTestNATS(t)
	defer cleanup()

	tests := []struct {
		name           string
		taskCapability string
		expectWorkerA  bool // Worker A has "code"
		expectWorkerB  bool // Worker B has "research"
	}{
		{
			name:           "code task to code worker",
			taskCapability: "code",
			expectWorkerA:  true,
			expectWorkerB:  false,
		},
		{
			name:           "research task to research worker",
			taskCapability: "research",
			expectWorkerA:  false,
			expectWorkerB:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Worker A: capability "code"
			nodeA := newTestNodeInfo("cap-worker-a", RoleWorker, []string{"code"}, 4)
			bridgeA := connectTestBridge(t, url, nodeA)
			defer bridgeA.Stop()

			var receivedA atomic.Int32
			var mu sync.Mutex
			var resultsA []*SwarmTask
			bridgeA.SetOnTaskReceived(func(task *SwarmTask) {
				receivedA.Add(1)
				mu.Lock()
				resultsA = append(resultsA, task)
				mu.Unlock()
			})

			if err := bridgeA.Start(context.Background()); err != nil {
				t.Fatalf("bridgeA Start() error: %v", err)
			}

			// Worker B: capability "research"
			nodeB := newTestNodeInfo("cap-worker-b", RoleWorker, []string{"research"}, 4)
			bridgeB := connectTestBridge(t, url, nodeB)
			defer bridgeB.Stop()

			var receivedB atomic.Int32
			bridgeB.SetOnTaskReceived(func(task *SwarmTask) {
				receivedB.Add(1)
			})

			if err := bridgeB.Start(context.Background()); err != nil {
				t.Fatalf("bridgeB Start() error: %v", err)
			}

			// Give subscriptions time to propagate
			time.Sleep(100 * time.Millisecond)

			// Coordinator publishes broadcast task
			coordNode := newTestNodeInfo("cap-coord", RoleCoordinator, nil, 1)
			coordBridge := connectTestBridge(t, url, coordNode)
			defer coordBridge.Stop()

			task := &SwarmTask{
				ID:         fmt.Sprintf("cap-task-%s", tt.taskCapability),
				Type:       TaskTypeBroadcast,
				Capability: tt.taskCapability,
				Prompt:     "capability routing test",
				Timeout:    5000,
			}
			// Broadcast: no AssignedTo, publish to capability subject
			if err := coordBridge.PublishTask(task); err != nil {
				t.Fatalf("PublishTask() error: %v", err)
			}

			// Wait for delivery
			time.Sleep(500 * time.Millisecond)

			gotA := receivedA.Load() > 0
			gotB := receivedB.Load() > 0

			if gotA != tt.expectWorkerA {
				t.Errorf("Worker A (code) received = %v, want %v", gotA, tt.expectWorkerA)
			}
			if gotB != tt.expectWorkerB {
				t.Errorf("Worker B (research) received = %v, want %v", gotB, tt.expectWorkerB)
			}
		})
	}
}
