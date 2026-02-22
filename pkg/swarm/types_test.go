// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNewSwarmTask(t *testing.T) {
	tests := []struct {
		name       string
		taskType   SwarmTaskType
		capability string
		prompt     string
	}{
		{
			name:       "direct task with defaults",
			taskType:   TaskTypeDirect,
			capability: "code",
			prompt:     "write code",
		},
		{
			name:       "broadcast task",
			taskType:   TaskTypeBroadcast,
			capability: "research",
			prompt:     "find info",
		},
		{
			name:       "workflow task",
			taskType:   TaskTypeWorkflow,
			capability: "complex",
			prompt:     "analyze data",
		},
		{
			name:       "empty capability",
			taskType:   TaskTypeDirect,
			capability: "",
			prompt:     "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := time.Now().UnixMilli()
			task := NewSwarmTask(tt.taskType, tt.capability, tt.prompt)
			after := time.Now().UnixMilli()

			if !strings.HasPrefix(task.ID, "task-") {
				t.Errorf("ID = %q, want prefix 'task-'", task.ID)
			}
			// "task-" (5 chars) + 8 hex chars = 13
			if len(task.ID) != 13 {
				t.Errorf("len(ID) = %d, want 13", len(task.ID))
			}
			if task.Type != tt.taskType {
				t.Errorf("Type = %q, want %q", task.Type, tt.taskType)
			}
			if task.Priority != 1 {
				t.Errorf("Priority = %d, want 1", task.Priority)
			}
			if task.Capability != tt.capability {
				t.Errorf("Capability = %q, want %q", task.Capability, tt.capability)
			}
			if task.Prompt != tt.prompt {
				t.Errorf("Prompt = %q, want %q", task.Prompt, tt.prompt)
			}
			if task.Status != TaskPending {
				t.Errorf("Status = %q, want %q", task.Status, TaskPending)
			}
			if task.Context == nil {
				t.Error("Context is nil, want non-nil empty map")
			}
			if len(task.Context) != 0 {
				t.Errorf("len(Context) = %d, want 0", len(task.Context))
			}
			if task.Timeout != 10*60*1000 {
				t.Errorf("Timeout = %d, want %d", task.Timeout, 10*60*1000)
			}
			if task.CreatedAt < before || task.CreatedAt > after {
				t.Errorf("CreatedAt = %d, want between %d and %d", task.CreatedAt, before, after)
			}
		})
	}
}

func TestSwarmTask_JSONRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		task SwarmTask
	}{
		{
			name: "full task",
			task: SwarmTask{
				ID:          "task-abc12345",
				WorkflowID:  "wf-1",
				ParentID:    "task-parent",
				Type:        TaskTypeDirect,
				Priority:    2,
				Capability:  "code",
				Prompt:      "write a function",
				Context:     map[string]interface{}{"key": "value", "num": float64(42)},
				AssignedTo:  "node-1",
				Status:      TaskRunning,
				Result:      "done",
				Error:       "",
				CreatedAt:   1000000,
				CompletedAt: 2000000,
				Timeout:     60000,
			},
		},
		{
			name: "minimal task",
			task: SwarmTask{
				ID:     "task-min00001",
				Type:   TaskTypeBroadcast,
				Prompt: "hello",
				Status: TaskPending,
			},
		},
		{
			name: "task with nested context",
			task: SwarmTask{
				ID:         "task-ctx00001",
				Type:       TaskTypeWorkflow,
				Prompt:     "complex",
				Capability: "analysis",
				Status:     TaskAssigned,
				Context: map[string]interface{}{
					"nested": map[string]interface{}{
						"deep": "value",
					},
					"list": []interface{}{"a", "b"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.task)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}

			var got SwarmTask
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			if got.ID != tt.task.ID {
				t.Errorf("ID = %q, want %q", got.ID, tt.task.ID)
			}
			if got.Type != tt.task.Type {
				t.Errorf("Type = %q, want %q", got.Type, tt.task.Type)
			}
			if got.Priority != tt.task.Priority {
				t.Errorf("Priority = %d, want %d", got.Priority, tt.task.Priority)
			}
			if got.Capability != tt.task.Capability {
				t.Errorf("Capability = %q, want %q", got.Capability, tt.task.Capability)
			}
			if got.Prompt != tt.task.Prompt {
				t.Errorf("Prompt = %q, want %q", got.Prompt, tt.task.Prompt)
			}
			if got.Status != tt.task.Status {
				t.Errorf("Status = %q, want %q", got.Status, tt.task.Status)
			}
			if got.Result != tt.task.Result {
				t.Errorf("Result = %q, want %q", got.Result, tt.task.Result)
			}
			if got.WorkflowID != tt.task.WorkflowID {
				t.Errorf("WorkflowID = %q, want %q", got.WorkflowID, tt.task.WorkflowID)
			}
			if got.AssignedTo != tt.task.AssignedTo {
				t.Errorf("AssignedTo = %q, want %q", got.AssignedTo, tt.task.AssignedTo)
			}
			if got.CreatedAt != tt.task.CreatedAt {
				t.Errorf("CreatedAt = %d, want %d", got.CreatedAt, tt.task.CreatedAt)
			}
			if got.Timeout != tt.task.Timeout {
				t.Errorf("Timeout = %d, want %d", got.Timeout, tt.task.Timeout)
			}
		})
	}
}

func TestNodeInfo_JSONRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		node NodeInfo
	}{
		{
			name: "worker with capabilities",
			node: NodeInfo{
				ID:           "node-1",
				Role:         RoleWorker,
				Capabilities: []string{"code", "research"},
				Model:        "gpt-4",
				Status:       StatusOnline,
				Load:         0.5,
				TasksRunning: 2,
				MaxTasks:     4,
				Metadata:     map[string]string{"region": "us-east"},
				LastSeen:     1000000,
				StartedAt:    900000,
				Address:      "nats://127.0.0.1:4222",
			},
		},
		{
			name: "coordinator",
			node: NodeInfo{
				ID:           "coord-1",
				Role:         RoleCoordinator,
				Capabilities: []string{},
				Status:       StatusBusy,
				MaxTasks:     1,
				Metadata:     map[string]string{},
			},
		},
		{
			name: "specialist offline",
			node: NodeInfo{
				ID:           "spec-1",
				Role:         RoleSpecialist,
				Capabilities: []string{"ml"},
				Status:       StatusOffline,
				Load:         0.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.node)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}

			var got NodeInfo
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			if got.ID != tt.node.ID {
				t.Errorf("ID = %q, want %q", got.ID, tt.node.ID)
			}
			if got.Role != tt.node.Role {
				t.Errorf("Role = %q, want %q", got.Role, tt.node.Role)
			}
			if got.Status != tt.node.Status {
				t.Errorf("Status = %q, want %q", got.Status, tt.node.Status)
			}
			if got.Load != tt.node.Load {
				t.Errorf("Load = %f, want %f", got.Load, tt.node.Load)
			}
			if got.MaxTasks != tt.node.MaxTasks {
				t.Errorf("MaxTasks = %d, want %d", got.MaxTasks, tt.node.MaxTasks)
			}
			if got.Model != tt.node.Model {
				t.Errorf("Model = %q, want %q", got.Model, tt.node.Model)
			}
			if got.Address != tt.node.Address {
				t.Errorf("Address = %q, want %q", got.Address, tt.node.Address)
			}
		})
	}
}

func TestTaskResult_JSONRoundTrip(t *testing.T) {
	tests := []struct {
		name   string
		result TaskResult
	}{
		{
			name: "success result",
			result: TaskResult{
				TaskID:      "task-abc12345",
				NodeID:      "node-1",
				Status:      "done",
				Result:      "completed successfully",
				CompletedAt: 1000000,
			},
		},
		{
			name: "failure result",
			result: TaskResult{
				TaskID:      "task-err00001",
				NodeID:      "node-2",
				Status:      "failed",
				Error:       "execution timeout",
				CompletedAt: 2000000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.result)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}

			var got TaskResult
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			if got.TaskID != tt.result.TaskID {
				t.Errorf("TaskID = %q, want %q", got.TaskID, tt.result.TaskID)
			}
			if got.NodeID != tt.result.NodeID {
				t.Errorf("NodeID = %q, want %q", got.NodeID, tt.result.NodeID)
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
			if got.CompletedAt != tt.result.CompletedAt {
				t.Errorf("CompletedAt = %d, want %d", got.CompletedAt, tt.result.CompletedAt)
			}
		})
	}
}

func TestHeartbeat_JSONRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		hb   Heartbeat
	}{
		{
			name: "online node",
			hb: Heartbeat{
				NodeID:       "node-1",
				Status:       StatusOnline,
				Load:         0.25,
				TasksRunning: 1,
				Timestamp:    1000000,
			},
		},
		{
			name: "busy node",
			hb: Heartbeat{
				NodeID:       "node-2",
				Status:       StatusBusy,
				Load:         1.0,
				TasksRunning: 4,
				Timestamp:    2000000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.hb)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}

			var got Heartbeat
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			if got.NodeID != tt.hb.NodeID {
				t.Errorf("NodeID = %q, want %q", got.NodeID, tt.hb.NodeID)
			}
			if got.Status != tt.hb.Status {
				t.Errorf("Status = %q, want %q", got.Status, tt.hb.Status)
			}
			if got.Load != tt.hb.Load {
				t.Errorf("Load = %f, want %f", got.Load, tt.hb.Load)
			}
			if got.TasksRunning != tt.hb.TasksRunning {
				t.Errorf("TasksRunning = %d, want %d", got.TasksRunning, tt.hb.TasksRunning)
			}
			if got.Timestamp != tt.hb.Timestamp {
				t.Errorf("Timestamp = %d, want %d", got.Timestamp, tt.hb.Timestamp)
			}
		})
	}
}

func TestDiscoveryAnnounce_JSONRoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		announce DiscoveryAnnounce
	}{
		{
			name: "worker announcement",
			announce: DiscoveryAnnounce{
				Node: NodeInfo{
					ID:           "node-1",
					Role:         RoleWorker,
					Capabilities: []string{"code"},
					Status:       StatusOnline,
					MaxTasks:     4,
					Metadata:     map[string]string{},
				},
				Timestamp: 1000000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.announce)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}

			var got DiscoveryAnnounce
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			if got.Node.ID != tt.announce.Node.ID {
				t.Errorf("Node.ID = %q, want %q", got.Node.ID, tt.announce.Node.ID)
			}
			if got.Node.Role != tt.announce.Node.Role {
				t.Errorf("Node.Role = %q, want %q", got.Node.Role, tt.announce.Node.Role)
			}
			if got.Timestamp != tt.announce.Timestamp {
				t.Errorf("Timestamp = %d, want %d", got.Timestamp, tt.announce.Timestamp)
			}
		})
	}
}

func TestDiscoveryQuery_JSONRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		query DiscoveryQuery
	}{
		{
			name: "query all",
			query: DiscoveryQuery{
				RequesterID: "node-0",
			},
		},
		{
			name: "query by role",
			query: DiscoveryQuery{
				RequesterID: "node-0",
				Role:        RoleWorker,
			},
		},
		{
			name: "query by capability",
			query: DiscoveryQuery{
				RequesterID: "node-0",
				Capability:  "code",
			},
		},
		{
			name: "query by both",
			query: DiscoveryQuery{
				RequesterID: "node-0",
				Role:        RoleSpecialist,
				Capability:  "ml",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.query)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}

			var got DiscoveryQuery
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			if got.RequesterID != tt.query.RequesterID {
				t.Errorf("RequesterID = %q, want %q", got.RequesterID, tt.query.RequesterID)
			}
			if got.Role != tt.query.Role {
				t.Errorf("Role = %q, want %q", got.Role, tt.query.Role)
			}
			if got.Capability != tt.query.Capability {
				t.Errorf("Capability = %q, want %q", got.Capability, tt.query.Capability)
			}
		})
	}
}

func TestTaskProgress_JSONRoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		progress TaskProgress
	}{
		{
			name: "half complete",
			progress: TaskProgress{
				TaskID:   "task-abc12345",
				NodeID:   "node-1",
				Progress: 0.5,
				Message:  "processing",
			},
		},
		{
			name: "complete",
			progress: TaskProgress{
				TaskID:   "task-done0001",
				NodeID:   "node-2",
				Progress: 1.0,
				Message:  "finished",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.progress)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}

			var got TaskProgress
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			if got.TaskID != tt.progress.TaskID {
				t.Errorf("TaskID = %q, want %q", got.TaskID, tt.progress.TaskID)
			}
			if got.NodeID != tt.progress.NodeID {
				t.Errorf("NodeID = %q, want %q", got.NodeID, tt.progress.NodeID)
			}
			if got.Progress != tt.progress.Progress {
				t.Errorf("Progress = %f, want %f", got.Progress, tt.progress.Progress)
			}
			if got.Message != tt.progress.Message {
				t.Errorf("Message = %q, want %q", got.Message, tt.progress.Message)
			}
		})
	}
}
