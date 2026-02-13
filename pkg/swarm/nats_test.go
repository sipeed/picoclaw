// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNATSBridge_Connect(t *testing.T) {
	_, url, cleanup := startTestNATS(t)
	defer cleanup()

	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "successful connect",
			fn: func(t *testing.T) {
				node := newTestNodeInfo("connect-test", RoleWorker, []string{"general"}, 4)
				bridge := connectTestBridge(t, url, node)
				defer bridge.Stop()

				if !bridge.IsConnected() {
					t.Error("IsConnected() = false, want true")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.fn)
	}
}

func TestNATSBridge_PublishSubscribeTaskAssignment(t *testing.T) {
	_, url, cleanup := startTestNATS(t)
	defer cleanup()

	tests := []struct {
		name       string
		task       *SwarmTask
		workerID   string
		workerCaps []string
	}{
		{
			name: "direct task to specific node",
			task: &SwarmTask{
				ID:         "task-direct01",
				Type:       TaskTypeDirect,
				Capability: "code",
				Prompt:     "write a function",
				Status:     TaskPending,
				Priority:   1,
				Timeout:    5000,
			},
			workerID:   "worker-1",
			workerCaps: []string{"code"},
		},
		{
			name: "task with context data",
			task: &SwarmTask{
				ID:         "task-ctx00001",
				Type:       TaskTypeDirect,
				Capability: "research",
				Prompt:     "find information",
				Status:     TaskPending,
				Context:    map[string]interface{}{"key": "value", "num": float64(42)},
				Timeout:    5000,
			},
			workerID:   "worker-2",
			workerCaps: []string{"research"},
		},
		{
			name: "task with all fields set",
			task: &SwarmTask{
				ID:          "task-full0001",
				WorkflowID:  "wf-1",
				ParentID:    "task-parent",
				Type:        TaskTypeDirect,
				Priority:    3,
				Capability:  "code",
				Prompt:      "complex task",
				Context:     map[string]interface{}{"a": "b"},
				Status:      TaskAssigned,
				CreatedAt:   1000000,
				Timeout:     30000,
			},
			workerID:   "worker-3",
			workerCaps: []string{"code"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create subscriber (worker)
			workerNode := newTestNodeInfo(tt.workerID, RoleWorker, tt.workerCaps, 4)
			workerBridge := connectTestBridge(t, url, workerNode)
			defer workerBridge.Stop()

			var received atomic.Value
			workerBridge.SetOnTaskReceived(func(task *SwarmTask) {
				received.Store(task)
			})

			if err := workerBridge.Start(context.Background()); err != nil {
				t.Fatalf("worker Start() error: %v", err)
			}

			// Create publisher (coordinator)
			coordNode := newTestNodeInfo("coord-pub", RoleCoordinator, nil, 1)
			coordBridge := connectTestBridge(t, url, coordNode)
			defer coordBridge.Stop()

			// Set AssignedTo and publish
			tt.task.AssignedTo = tt.workerID
			if err := coordBridge.PublishTask(tt.task); err != nil {
				t.Fatalf("PublishTask() error: %v", err)
			}

			// Wait for delivery
			ok := waitFor(t, 2*time.Second, func() bool {
				return received.Load() != nil
			})
			if !ok {
				t.Fatal("timed out waiting for task delivery")
			}

			got := received.Load().(*SwarmTask)
			if got.ID != tt.task.ID {
				t.Errorf("received ID = %q, want %q", got.ID, tt.task.ID)
			}
			if got.Prompt != tt.task.Prompt {
				t.Errorf("received Prompt = %q, want %q", got.Prompt, tt.task.Prompt)
			}
			if got.Capability != tt.task.Capability {
				t.Errorf("received Capability = %q, want %q", got.Capability, tt.task.Capability)
			}
			if got.Priority != tt.task.Priority {
				t.Errorf("received Priority = %d, want %d", got.Priority, tt.task.Priority)
			}
		})
	}
}

func TestNATSBridge_PublishSubscribeBroadcast(t *testing.T) {
	_, url, cleanup := startTestNATS(t)
	defer cleanup()

	tests := []struct {
		name           string
		capability     string
		workerACaps    []string
		workerBCaps    []string
		expectReceived bool
	}{
		{
			name:           "broadcast by capability",
			capability:     "code",
			workerACaps:    []string{"code"},
			workerBCaps:    []string{"code"},
			expectReceived: true,
		},
		{
			name:           "broadcast to unmatched capability",
			capability:     "ml",
			workerACaps:    []string{"code"},
			workerBCaps:    []string{"research"},
			expectReceived: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create two workers
			nodeA := newTestNodeInfo("bcast-a", RoleWorker, tt.workerACaps, 4)
			bridgeA := connectTestBridge(t, url, nodeA)
			defer bridgeA.Stop()

			nodeB := newTestNodeInfo("bcast-b", RoleWorker, tt.workerBCaps, 4)
			bridgeB := connectTestBridge(t, url, nodeB)
			defer bridgeB.Stop()

			var receivedCount atomic.Int32
			handler := func(task *SwarmTask) {
				receivedCount.Add(1)
			}
			bridgeA.SetOnTaskReceived(handler)
			bridgeB.SetOnTaskReceived(handler)

			if err := bridgeA.Start(context.Background()); err != nil {
				t.Fatalf("bridgeA Start() error: %v", err)
			}
			if err := bridgeB.Start(context.Background()); err != nil {
				t.Fatalf("bridgeB Start() error: %v", err)
			}

			// Give subscriptions time to propagate
			time.Sleep(50 * time.Millisecond)

			// Publish broadcast task (no AssignedTo)
			coordNode := newTestNodeInfo("bcast-coord", RoleCoordinator, nil, 1)
			coordBridge := connectTestBridge(t, url, coordNode)
			defer coordBridge.Stop()

			task := &SwarmTask{
				ID:         "task-bcast001",
				Type:       TaskTypeBroadcast,
				Capability: tt.capability,
				Prompt:     "broadcast test",
				Timeout:    5000,
			}
			if err := coordBridge.PublishTask(task); err != nil {
				t.Fatalf("PublishTask() error: %v", err)
			}

			if tt.expectReceived {
				// At least one worker should receive it (queue group delivers to one)
				ok := waitFor(t, 2*time.Second, func() bool {
					return receivedCount.Load() >= 1
				})
				if !ok {
					t.Error("expected at least one worker to receive broadcast, got 0")
				}
			} else {
				// No one should receive
				time.Sleep(200 * time.Millisecond)
				if receivedCount.Load() > 0 {
					t.Errorf("expected no workers to receive broadcast, got %d", receivedCount.Load())
				}
			}
		})
	}
}

func TestNATSBridge_DiscoveryRoundTrip(t *testing.T) {
	_, url, cleanup := startTestNATS(t)
	defer cleanup()

	tests := []struct {
		name          string
		responderRole NodeRole
		responderCaps []string
		queryRole     NodeRole
		queryCap      string
		expectFound   bool
	}{
		{
			name:          "query all nodes",
			responderRole: RoleWorker,
			responderCaps: []string{"code"},
			queryRole:     "",
			queryCap:      "",
			expectFound:   true,
		},
		{
			name:          "query by role worker",
			responderRole: RoleWorker,
			responderCaps: []string{"code"},
			queryRole:     RoleWorker,
			queryCap:      "",
			expectFound:   true,
		},
		{
			name:          "query by capability",
			responderRole: RoleWorker,
			responderCaps: []string{"code", "research"},
			queryRole:     "",
			queryCap:      "research",
			expectFound:   true,
		},
		{
			name:          "query with no matches - role mismatch",
			responderRole: RoleWorker,
			responderCaps: []string{"code"},
			queryRole:     RoleSpecialist,
			queryCap:      "",
			expectFound:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Responder bridge
			respNode := newTestNodeInfo("disc-resp", tt.responderRole, tt.responderCaps, 4)
			respBridge := connectTestBridge(t, url, respNode)
			defer respBridge.Stop()

			if err := respBridge.Start(context.Background()); err != nil {
				t.Fatalf("responder Start() error: %v", err)
			}

			// Requester bridge
			reqNode := newTestNodeInfo("disc-req", RoleCoordinator, nil, 1)
			reqBridge := connectTestBridge(t, url, reqNode)
			defer reqBridge.Stop()

			// Give subscriptions time to propagate
			time.Sleep(50 * time.Millisecond)

			query := &DiscoveryQuery{
				RequesterID: "disc-req",
				Role:        tt.queryRole,
				Capability:  tt.queryCap,
			}

			nodes, err := reqBridge.RequestDiscovery(query, 500*time.Millisecond)
			if err != nil {
				t.Fatalf("RequestDiscovery() error: %v", err)
			}

			if tt.expectFound {
				if len(nodes) == 0 {
					t.Error("expected at least 1 node, got 0")
				} else {
					if nodes[0].ID != "disc-resp" {
						t.Errorf("node ID = %q, want %q", nodes[0].ID, "disc-resp")
					}
				}
			} else {
				if len(nodes) != 0 {
					t.Errorf("expected 0 nodes, got %d", len(nodes))
				}
			}
		})
	}
}

func TestNATSBridge_HeartbeatPublishSubscribe(t *testing.T) {
	_, url, cleanup := startTestNATS(t)
	defer cleanup()

	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "heartbeat received",
			fn: func(t *testing.T) {
				pubNode := newTestNodeInfo("hb-pub", RoleWorker, []string{"code"}, 4)
				pubBridge := connectTestBridge(t, url, pubNode)
				defer pubBridge.Stop()

				subNode := newTestNodeInfo("hb-sub", RoleCoordinator, nil, 1)
				subBridge := connectTestBridge(t, url, subNode)
				defer subBridge.Stop()

			var received atomic.Value
			_, err := subBridge.SubscribeHeartbeat("hb-pub", func(hb *Heartbeat) {
				received.Store(hb)
			})
			if err != nil {
				t.Fatalf("SubscribeHeartbeat() error: %v", err)
			}

			// Flush to ensure subscription is registered on server before publishing
			subBridge.conn.Flush()
			time.Sleep(50 * time.Millisecond)

			hb := &Heartbeat{
					NodeID:       "hb-pub",
					Status:       StatusOnline,
					Load:         0.5,
					TasksRunning: 2,
					Timestamp:    time.Now().UnixMilli(),
				}
				if err := pubBridge.PublishHeartbeat(hb); err != nil {
					t.Fatalf("PublishHeartbeat() error: %v", err)
				}

				ok := waitFor(t, 2*time.Second, func() bool {
					return received.Load() != nil
				})
				if !ok {
					t.Fatal("timed out waiting for heartbeat")
				}

				got := received.Load().(*Heartbeat)
				if got.NodeID != hb.NodeID {
					t.Errorf("NodeID = %q, want %q", got.NodeID, hb.NodeID)
				}
				if got.Status != hb.Status {
					t.Errorf("Status = %q, want %q", got.Status, hb.Status)
				}
				if got.Load != hb.Load {
					t.Errorf("Load = %f, want %f", got.Load, hb.Load)
				}
				if got.TasksRunning != hb.TasksRunning {
					t.Errorf("TasksRunning = %d, want %d", got.TasksRunning, hb.TasksRunning)
				}
			},
		},
		{
			name: "wildcard heartbeat subscription",
			fn: func(t *testing.T) {
				subNode := newTestNodeInfo("hb-wild-sub", RoleCoordinator, nil, 1)
				subBridge := connectTestBridge(t, url, subNode)
				defer subBridge.Stop()

				var mu sync.Mutex
				received := make(map[string]*Heartbeat)
				_, err := subBridge.SubscribeAllHeartbeats(func(hb *Heartbeat) {
					mu.Lock()
					received[hb.NodeID] = hb
					mu.Unlock()
				})
			if err != nil {
				t.Fatalf("SubscribeAllHeartbeats() error: %v", err)
			}

			// Flush to ensure subscription is registered on server
			subBridge.conn.Flush()
			time.Sleep(50 * time.Millisecond)

			// Publish from two different nodes
				for _, nodeID := range []string{"hb-wild-a", "hb-wild-b"} {
					node := newTestNodeInfo(nodeID, RoleWorker, []string{"code"}, 4)
					bridge := connectTestBridge(t, url, node)
					defer bridge.Stop()

					hb := &Heartbeat{
						NodeID:    nodeID,
						Status:    StatusOnline,
						Load:      0.1,
						Timestamp: time.Now().UnixMilli(),
					}
					if err := bridge.PublishHeartbeat(hb); err != nil {
						t.Fatalf("PublishHeartbeat(%s) error: %v", nodeID, err)
					}
				}

				ok := waitFor(t, 2*time.Second, func() bool {
					mu.Lock()
					defer mu.Unlock()
					return len(received) >= 2
				})
				if !ok {
					mu.Lock()
					t.Fatalf("timed out waiting for heartbeats, got %d", len(received))
					mu.Unlock()
				}

				mu.Lock()
				defer mu.Unlock()
				if _, ok := received["hb-wild-a"]; !ok {
					t.Error("missing heartbeat from hb-wild-a")
				}
				if _, ok := received["hb-wild-b"]; !ok {
					t.Error("missing heartbeat from hb-wild-b")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.fn)
	}
}

func TestNATSBridge_TaskResultPublishSubscribe(t *testing.T) {
	_, url, cleanup := startTestNATS(t)
	defer cleanup()

	tests := []struct {
		name   string
		result TaskResult
	}{
		{
			name: "success result",
			result: TaskResult{
				TaskID:      "task-res00001",
				NodeID:      "worker-1",
				Status:      "done",
				Result:      "completed successfully",
				CompletedAt: time.Now().UnixMilli(),
			},
		},
		{
			name: "failure result",
			result: TaskResult{
				TaskID:      "task-res00002",
				NodeID:      "worker-2",
				Status:      "failed",
				Error:       "out of memory",
				CompletedAt: time.Now().UnixMilli(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Subscriber
			subNode := newTestNodeInfo("result-sub", RoleCoordinator, nil, 1)
			subBridge := connectTestBridge(t, url, subNode)
			defer subBridge.Stop()

			var received atomic.Value
			_, err := subBridge.SubscribeTaskResult(tt.result.TaskID, func(r *TaskResult) {
				received.Store(r)
			})
			if err != nil {
				t.Fatalf("SubscribeTaskResult() error: %v", err)
			}

			// Flush to ensure subscription is registered on server
			subBridge.conn.Flush()
			time.Sleep(50 * time.Millisecond)

			// Publisher
			pubNode := newTestNodeInfo("result-pub", RoleWorker, []string{"code"}, 4)
			pubBridge := connectTestBridge(t, url, pubNode)
			defer pubBridge.Stop()

			if err := pubBridge.PublishTaskResult(&tt.result); err != nil {
				t.Fatalf("PublishTaskResult() error: %v", err)
			}

			ok := waitFor(t, 2*time.Second, func() bool {
				return received.Load() != nil
			})
			if !ok {
				t.Fatal("timed out waiting for task result")
			}

			got := received.Load().(*TaskResult)
			if got.TaskID != tt.result.TaskID {
				t.Errorf("TaskID = %q, want %q", got.TaskID, tt.result.TaskID)
			}
			if got.Status != tt.result.Status {
				t.Errorf("Status = %q, want %q", got.Status, tt.result.Status)
			}
			if got.Result != tt.result.Result {
				t.Errorf("Result = %q, want %q", got.Result, tt.result.Result)
			}
			if got.Error != tt.result.Error {
				t.Errorf("Error = %q, want %q", got.Error, tt.result.Error)
			}
		})
	}
}

func TestNATSBridge_ShutdownAnnouncement(t *testing.T) {
	_, url, cleanup := startTestNATS(t)
	defer cleanup()

	// Subscriber
	subNode := newTestNodeInfo("shutdown-sub", RoleCoordinator, nil, 1)
	subBridge := connectTestBridge(t, url, subNode)
	defer subBridge.Stop()

	var receivedID atomic.Value
	_, err := subBridge.SubscribeShutdown(func(nodeID string) {
		receivedID.Store(nodeID)
	})
	if err != nil {
		t.Fatalf("SubscribeShutdown() error: %v", err)
	}

	// Flush to ensure subscription is registered on server
	subBridge.conn.Flush()
	time.Sleep(50 * time.Millisecond)

	// Bridge that will shut down
	shutdownNode := newTestNodeInfo("shutdown-node", RoleWorker, []string{"code"}, 4)
	shutdownBridge := connectTestBridge(t, url, shutdownNode)

	if err := shutdownBridge.Start(context.Background()); err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	// Stop the bridge - this should publish shutdown
	if err := shutdownBridge.Stop(); err != nil {
		t.Fatalf("Stop() error: %v", err)
	}

	ok := waitFor(t, 2*time.Second, func() bool {
		return receivedID.Load() != nil
	})
	if !ok {
		t.Fatal("timed out waiting for shutdown announcement")
	}

	got := receivedID.Load().(string)
	if got != "shutdown-node" {
		t.Errorf("received nodeID = %q, want %q", got, "shutdown-node")
	}
}

func TestNATSBridge_Stop(t *testing.T) {
	_, url, cleanup := startTestNATS(t)
	defer cleanup()

	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "graceful stop drains connection",
			fn: func(t *testing.T) {
				node := newTestNodeInfo("stop-test", RoleWorker, []string{"code"}, 4)
				bridge := connectTestBridge(t, url, node)

				if err := bridge.Start(context.Background()); err != nil {
					t.Fatalf("Start() error: %v", err)
				}

				if !bridge.IsConnected() {
					t.Error("IsConnected() = false before Stop")
				}

				if err := bridge.Stop(); err != nil {
					t.Fatalf("Stop() error: %v", err)
				}

				// After drain, connection should eventually close
				ok := waitFor(t, 2*time.Second, func() bool {
					return !bridge.IsConnected()
				})
				if !ok {
					t.Error("bridge still connected after Stop()")
				}
			},
		},
		{
			name: "stop publishes shutdown",
			fn: func(t *testing.T) {
				// Listener bridge
				listenerNode := newTestNodeInfo("stop-listener", RoleCoordinator, nil, 1)
				listenerBridge := connectTestBridge(t, url, listenerNode)
				defer listenerBridge.Stop()

				var gotShutdown atomic.Value
				_, err := listenerBridge.SubscribeShutdown(func(nodeID string) {
					gotShutdown.Store(nodeID)
				})
				if err != nil {
					t.Fatalf("SubscribeShutdown() error: %v", err)
				}

				// Bridge to stop
				stopNode := newTestNodeInfo("stop-sender", RoleWorker, []string{"code"}, 4)
				stopBridge := connectTestBridge(t, url, stopNode)

				if err := stopBridge.Start(context.Background()); err != nil {
					t.Fatalf("Start() error: %v", err)
				}

				if err := stopBridge.Stop(); err != nil {
					t.Fatalf("Stop() error: %v", err)
				}

				ok := waitFor(t, 2*time.Second, func() bool {
					return gotShutdown.Load() != nil
				})
				if !ok {
					t.Fatal("timed out waiting for shutdown from Stop()")
				}

				if gotShutdown.Load().(string) != "stop-sender" {
					t.Errorf("shutdown nodeID = %q, want %q", gotShutdown.Load().(string), "stop-sender")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.fn)
	}
}
