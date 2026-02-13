// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

func TestCoordinator_DispatchDirect(t *testing.T) {
	_, url, cleanup := startTestNATS(t)
	defer cleanup()

	tests := []struct {
		name     string
		workerID string
		task     *SwarmTask
	}{
		{
			name:     "task dispatched to discovered worker",
			workerID: "coord-worker-1",
			task: &SwarmTask{
				ID:         "task-coord001",
				Type:       TaskTypeDirect,
				Capability: "code",
				Prompt:     "test task",
				Status:     TaskPending,
				Timeout:    5000,
			},
		},
		{
			name:     "task with specific AssignedTo",
			workerID: "coord-worker-2",
			task: &SwarmTask{
				ID:         "task-coord002",
				Type:       TaskTypeDirect,
				Capability: "code",
				Prompt:     "specific assignment",
				Status:     TaskPending,
				AssignedTo: "coord-worker-2",
				Timeout:    5000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up worker bridge that will receive and respond
			workerNode := newTestNodeInfo(tt.workerID, RoleWorker, []string{"code"}, 4)
			workerBridge := connectTestBridge(t, url, workerNode)
			defer workerBridge.Stop()

			workerBridge.SetOnTaskReceived(func(task *SwarmTask) {
				// Simulate worker execution: publish result
				result := &TaskResult{
					TaskID:      task.ID,
					NodeID:      tt.workerID,
					Status:      string(TaskDone),
					Result:      "executed: " + task.Prompt,
					CompletedAt: time.Now().UnixMilli(),
				}
				workerBridge.PublishTaskResult(result)
			})

			if err := workerBridge.Start(context.Background()); err != nil {
				t.Fatalf("worker Start() error: %v", err)
			}

			// Set up coordinator
			coordNode := newTestNodeInfo("coord-main", RoleCoordinator, nil, 1)
			coordBridge := connectTestBridge(t, url, coordNode)
			defer coordBridge.Stop()

			if err := coordBridge.Start(context.Background()); err != nil {
				t.Fatalf("coordinator bridge Start() error: %v", err)
			}

			swarmCfg := newTestSwarmConfig(0)
			discovery := NewDiscovery(coordBridge, coordNode, swarmCfg)
			// Register the worker in discovery
			discovery.handleNodeJoin(workerNode)

			temporal := NewTemporalClient(&config.TemporalConfig{TaskQueue: "test"})
			agentLoop := newTestAgentLoop(t, "local result", nil)
			localBus := bus.NewMessageBus()

			coordinator := NewCoordinator(swarmCfg, coordBridge, temporal, discovery, agentLoop, &mockLLMProvider{}, localBus)

			// Give subscriptions time to propagate
			time.Sleep(50 * time.Millisecond)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			result, err := coordinator.DispatchTask(ctx, tt.task)
			if err != nil {
				t.Fatalf("DispatchTask() error: %v", err)
			}
			if result == nil {
				t.Fatal("DispatchTask() returned nil result")
			}
			if result.Status != string(TaskDone) {
				t.Errorf("Status = %q, want %q", result.Status, string(TaskDone))
			}
			if !strings.Contains(result.Result, tt.task.Prompt) {
				t.Errorf("Result = %q, want it to contain %q", result.Result, tt.task.Prompt)
			}
		})
	}
}

func TestCoordinator_DispatchNoWorkers(t *testing.T) {
	_, url, cleanup := startTestNATS(t)
	defer cleanup()

	tests := []struct {
		name         string
		chatResponse string
		chatErr      error
		wantStatus   string
		wantContains string // check result or error contains this
	}{
		{
			name:         "local fallback success",
			chatResponse: "local execution result",
			chatErr:      nil,
			wantStatus:   string(TaskDone),
			wantContains: "local execution result",
		},
		{
			name:         "local fallback on error",
			chatResponse: "",
			chatErr:      fmt.Errorf("LLM unavailable"),
			wantStatus:   string(TaskFailed),
			wantContains: "LLM unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coordNode := newTestNodeInfo("coord-noworker", RoleCoordinator, nil, 1)
			coordBridge := connectTestBridge(t, url, coordNode)
			defer coordBridge.Stop()

			if err := coordBridge.Start(context.Background()); err != nil {
				t.Fatalf("Start() error: %v", err)
			}

			swarmCfg := newTestSwarmConfig(0)
			discovery := NewDiscovery(coordBridge, coordNode, swarmCfg)
			// No workers registered -- empty discovery

			temporal := NewTemporalClient(&config.TemporalConfig{TaskQueue: "test"})
			agentLoop := newTestAgentLoop(t, tt.chatResponse, tt.chatErr)
			localBus := bus.NewMessageBus()

			coordinator := NewCoordinator(swarmCfg, coordBridge, temporal, discovery, agentLoop, &mockLLMProvider{}, localBus)

			task := &SwarmTask{
				ID:         "task-local001",
				Type:       TaskTypeDirect,
				Capability: "code",
				Prompt:     "test prompt",
				Status:     TaskPending,
				Timeout:    5000,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

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
			// Check either Result or Error field
			combined := result.Result + result.Error
			if !strings.Contains(combined, tt.wantContains) {
				t.Errorf("Result+Error = %q, want it to contain %q", combined, tt.wantContains)
			}
		})
	}
}

func TestCoordinator_TaskTimeout(t *testing.T) {
	_, url, cleanup := startTestNATS(t)
	defer cleanup()

	// Set up a worker that never responds
	workerNode := newTestNodeInfo("timeout-worker", RoleWorker, []string{"code"}, 4)
	workerBridge := connectTestBridge(t, url, workerNode)
	defer workerBridge.Stop()

	// Intentionally do NOT set onTaskReceived - worker never processes
	if err := workerBridge.Start(context.Background()); err != nil {
		t.Fatalf("worker Start() error: %v", err)
	}

	coordNode := newTestNodeInfo("timeout-coord", RoleCoordinator, nil, 1)
	coordBridge := connectTestBridge(t, url, coordNode)
	defer coordBridge.Stop()

	if err := coordBridge.Start(context.Background()); err != nil {
		t.Fatalf("coord Start() error: %v", err)
	}

	swarmCfg := newTestSwarmConfig(0)
	discovery := NewDiscovery(coordBridge, coordNode, swarmCfg)
	discovery.handleNodeJoin(workerNode)

	temporal := NewTemporalClient(&config.TemporalConfig{TaskQueue: "test"})
	agentLoop := newTestAgentLoop(t, "unused", nil)
	localBus := bus.NewMessageBus()

	coordinator := NewCoordinator(swarmCfg, coordBridge, temporal, discovery, agentLoop, &mockLLMProvider{}, localBus)

	time.Sleep(50 * time.Millisecond)

	task := &SwarmTask{
		ID:         "task-timeout1",
		Type:       TaskTypeDirect,
		Capability: "code",
		Prompt:     "will timeout",
		Status:     TaskPending,
		Timeout:    100, // 100ms -- very short
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := coordinator.DispatchTask(ctx, task)
	if err != nil {
		t.Fatalf("DispatchTask() error: %v", err)
	}
	if result == nil {
		t.Fatal("DispatchTask() returned nil result")
	}
	if result.Status != string(TaskFailed) {
		t.Errorf("Status = %q, want %q", result.Status, string(TaskFailed))
	}
	if !strings.Contains(result.Error, "timeout") {
		t.Errorf("Error = %q, want it to contain 'timeout'", result.Error)
	}
}

func TestCoordinator_UnknownTaskType(t *testing.T) {
	_, url, cleanup := startTestNATS(t)
	defer cleanup()

	coordNode := newTestNodeInfo("unknown-coord", RoleCoordinator, nil, 1)
	coordBridge := connectTestBridge(t, url, coordNode)
	defer coordBridge.Stop()

	swarmCfg := newTestSwarmConfig(0)
	discovery := NewDiscovery(coordBridge, coordNode, swarmCfg)
	temporal := NewTemporalClient(&config.TemporalConfig{TaskQueue: "test"})
	agentLoop := newTestAgentLoop(t, "unused", nil)
	localBus := bus.NewMessageBus()

	coordinator := NewCoordinator(swarmCfg, coordBridge, temporal, discovery, agentLoop, &mockLLMProvider{}, localBus)

	task := &SwarmTask{
		ID:     "task-unknown01",
		Type:   SwarmTaskType("invalid"),
		Prompt: "should fail",
	}

	ctx := context.Background()
	_, err := coordinator.DispatchTask(ctx, task)
	if err == nil {
		t.Fatal("DispatchTask() expected error for unknown task type, got nil")
	}
	if !strings.Contains(err.Error(), "unknown task type") {
		t.Errorf("error = %q, want it to contain 'unknown task type'", err.Error())
	}
}
