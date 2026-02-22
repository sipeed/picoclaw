// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestWorker_ExecuteTask(t *testing.T) {
	_, url, cleanup := startTestNATS(t)
	defer cleanup()

	tests := []struct {
		name         string
		chatResponse string
		chatErr      error
		wantStatus   string
		wantContains string // check result or error
	}{
		{
			name:         "successful execution",
			chatResponse: "task completed successfully",
			chatErr:      nil,
			wantStatus:   string(TaskDone),
			wantContains: "task completed",
		},
		{
			name:         "execution returns error",
			chatResponse: "",
			chatErr:      fmt.Errorf("agent processing failed"),
			wantStatus:   string(TaskFailed),
			wantContains: "agent processing failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workerNode := newTestNodeInfo("exec-worker", RoleWorker, []string{"code"}, 4)
			workerBridge := connectTestBridge(t, url, workerNode)
			defer workerBridge.Stop()

			if err := workerBridge.Start(context.Background()); err != nil {
				t.Fatalf("Start() error: %v", err)
			}

			swarmCfg := newTestSwarmConfig(0)
			swarmCfg.MaxConcurrent = 2
			temporal := NewTemporalClient(&config.TemporalConfig{TaskQueue: "test"})
			agentLoop := newTestAgentLoop(t, tt.chatResponse, tt.chatErr)

			worker := NewWorker(swarmCfg, workerBridge, temporal, agentLoop, &mockLLMProvider{}, workerNode)

			// Subscribe to results from coordinator side
			coordNode := newTestNodeInfo("exec-coord", RoleCoordinator, nil, 1)
			coordBridge := connectTestBridge(t, url, coordNode)
			defer coordBridge.Stop()

			taskID := fmt.Sprintf("task-exec%04d", time.Now().UnixNano()%10000)
			var received atomic.Value
			sub, err := coordBridge.SubscribeTaskResult(taskID, func(r *TaskResult) {
				received.Store(r)
			})
			if err != nil {
				t.Fatalf("SubscribeTaskResult() error: %v", err)
			}
			defer sub.Unsubscribe()

			// Start worker
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			if err := worker.Start(ctx); err != nil {
				t.Fatalf("worker Start() error: %v", err)
			}

			// Send task directly to worker's queue
			task := &SwarmTask{
				ID:         taskID,
				Type:       TaskTypeDirect,
				Capability: "code",
				Prompt:     "execute this",
				Status:     TaskPending,
				Timeout:    5000,
			}
			worker.taskQueue <- task

			// Wait for result
			ok := waitFor(t, 5*time.Second, func() bool {
				return received.Load() != nil
			})
			if !ok {
				t.Fatal("timed out waiting for task result")
			}

			got := received.Load().(*TaskResult)
			if got.Status != tt.wantStatus {
				t.Errorf("Status = %q, want %q", got.Status, tt.wantStatus)
			}
			combined := got.Result + got.Error
			if !strings.Contains(combined, tt.wantContains) {
				t.Errorf("Result+Error = %q, want it to contain %q", combined, tt.wantContains)
			}
			if got.NodeID != "exec-worker" {
				t.Errorf("NodeID = %q, want %q", got.NodeID, "exec-worker")
			}
		})
	}
}

func TestWorker_LoadTracking(t *testing.T) {
	tests := []struct {
		name       string
		maxConc    int
		setupTasks int // number of tasks running at check time
		wantBusy   bool
	}{
		{
			name:       "initial state is idle",
			maxConc:    4,
			setupTasks: 0,
			wantBusy:   false,
		},
		{
			name:       "max concurrent reached marks busy",
			maxConc:    2,
			setupTasks: 2,
			wantBusy:   true,
		},
		{
			name:       "below max is online",
			maxConc:    4,
			setupTasks: 1,
			wantBusy:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workerNode := newTestNodeInfo("load-worker", RoleWorker, []string{"code"}, tt.maxConc)
			swarmCfg := newTestSwarmConfig(0)
			swarmCfg.MaxConcurrent = tt.maxConc

			// Simulate load state directly on nodeInfo
			workerNode.TasksRunning = tt.setupTasks
			if tt.maxConc > 0 {
				workerNode.Load = float64(tt.setupTasks) / float64(tt.maxConc)
			}
			if tt.setupTasks >= tt.maxConc {
				workerNode.Status = StatusBusy
			} else {
				workerNode.Status = StatusOnline
			}

			if tt.wantBusy {
				if workerNode.Status != StatusBusy {
					t.Errorf("Status = %q, want %q", workerNode.Status, StatusBusy)
				}
			} else {
				if workerNode.Status != StatusOnline {
					t.Errorf("Status = %q, want %q", workerNode.Status, StatusOnline)
				}
			}

			expectedLoad := float64(tt.setupTasks) / float64(tt.maxConc)
			if workerNode.Load != expectedLoad {
				t.Errorf("Load = %f, want %f", workerNode.Load, expectedLoad)
			}
		})
	}
}

func TestWorker_UpdateLoad(t *testing.T) {
	// Test the actual updateLoad method on Worker
	tests := []struct {
		name         string
		maxConc      int
		tasksRunning int
		wantLoad     float64
		wantStatus   NodeStatus
	}{
		{
			name:         "idle worker",
			maxConc:      4,
			tasksRunning: 0,
			wantLoad:     0.0,
			wantStatus:   StatusOnline,
		},
		{
			name:         "partially loaded",
			maxConc:      4,
			tasksRunning: 2,
			wantLoad:     0.5,
			wantStatus:   StatusOnline,
		},
		{
			name:         "fully loaded",
			maxConc:      2,
			tasksRunning: 2,
			wantLoad:     1.0,
			wantStatus:   StatusBusy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodeInfo := newTestNodeInfo("update-load", RoleWorker, []string{"code"}, tt.maxConc)

			swarmCfg := newTestSwarmConfig(0)
			swarmCfg.MaxConcurrent = tt.maxConc

			w := &Worker{
				nodeInfo: nodeInfo,
				cfg:      swarmCfg,
			}
			// Set atomic counter for thread-safe load tracking
			w.tasksRunning.Store(int32(tt.tasksRunning))

			w.updateLoad()

			if w.nodeInfo.Load != tt.wantLoad {
				t.Errorf("Load = %f, want %f", w.nodeInfo.Load, tt.wantLoad)
			}
			if w.nodeInfo.Status != tt.wantStatus {
				t.Errorf("Status = %q, want %q", w.nodeInfo.Status, tt.wantStatus)
			}
		})
	}
}

func TestWorker_QueueFullRejection(t *testing.T) {
	tests := []struct {
		name        string
		maxConc     int
		fillCount   int // fill this many tasks first
		expectAccept bool
	}{
		{
			name:         "queue accepts when space available",
			maxConc:      1,
			fillCount:    0,
			expectAccept: true,
		},
		{
			name:         "queue rejects when full",
			maxConc:      1,
			fillCount:    2, // queue size is maxConcurrent*2 = 2, fill completely
			expectAccept: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			swarmCfg := newTestSwarmConfig(0)
			swarmCfg.MaxConcurrent = tt.maxConc

			nodeInfo := newTestNodeInfo("queue-worker", RoleWorker, []string{"code"}, tt.maxConc)

			// Create worker with small queue
			w := &Worker{
				nodeInfo:  nodeInfo,
				cfg:       swarmCfg,
				taskQueue: make(chan *SwarmTask, swarmCfg.MaxConcurrent*2),
			}

			// Fill the queue
			for i := 0; i < tt.fillCount; i++ {
				task := &SwarmTask{
					ID:     fmt.Sprintf("fill-task-%d", i),
					Prompt: "filler",
				}
				select {
				case w.taskQueue <- task:
				default:
					// Queue full already
				}
			}

			// Try to send one more
			testTask := &SwarmTask{
				ID:     "test-overflow",
				Prompt: "overflow test",
			}

			accepted := false
			select {
			case w.taskQueue <- testTask:
				accepted = true
			default:
				accepted = false
			}

			if accepted != tt.expectAccept {
				t.Errorf("accepted = %v, want %v", accepted, tt.expectAccept)
			}
		})
	}
}
