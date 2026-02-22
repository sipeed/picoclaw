// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSpecialistNode_extractSkillMetadata(t *testing.T) {
	specialist := &SpecialistNode{}

	t.Run("metadata with description and version", func(t *testing.T) {
		content := `Description: This is a test skill
Version: 1.0.0
Author: Test Author

# Skill Name

Some content here`

		metadata := specialist.extractSkillMetadata(content)
		assert.Equal(t, "This is a test skill", metadata["description"])
		assert.Equal(t, "1.0.0", metadata["version"])
		assert.Equal(t, "Test Author", metadata["author"])
	})

	t.Run("metadata from heading", func(t *testing.T) {
		content := `# Test Skill Heading

Some content here`

		metadata := specialist.extractSkillMetadata(content)
		assert.Equal(t, "Test Skill Heading", metadata["description"])
	})

	t.Run("empty content", func(t *testing.T) {
		content := ``

		metadata := specialist.extractSkillMetadata(content)
		assert.Empty(t, metadata["description"])
	})
}

func TestCapability(t *testing.T) {
	cap := &Capability{
		Name:        "test-capability",
		Description: "A test capability",
		Version:     "1.0.0",
		NodeID:      "node-1",
		Metadata: map[string]interface{}{
			"key": "value",
		},
	}

	assert.Equal(t, "test-capability", cap.Name)
	assert.Equal(t, "A test capability", cap.Description)
	assert.Equal(t, "1.0.0", cap.Version)
	assert.Equal(t, "node-1", cap.NodeID)
	assert.NotNil(t, cap.Metadata)
}

func TestCapabilityRequest(t *testing.T) {
	req := CapabilityRequest{
		RequesterID: "node-1",
		Capability:  "test-capability",
		Version:     "1.0.0",
	}

	assert.Equal(t, "node-1", req.RequesterID)
	assert.Equal(t, "test-capability", req.Capability)
	assert.Equal(t, "1.0.0", req.Version)
}

func TestCapabilityResponse(t *testing.T) {
	caps := []Capability{
		{
			Name:        "cap-1",
			Description: "Capability 1",
			Version:     "1.0.0",
			NodeID:      "node-1",
		},
	}

	resp := CapabilityResponse{
		Capabilities: caps,
		RequestID:    "req-1",
		Timestamp:    time.Now().UnixMilli(),
	}

	assert.Equal(t, caps, resp.Capabilities)
	assert.Equal(t, "req-1", resp.RequestID)
	assert.NotZero(t, resp.Timestamp)
}

func TestDAGNodeCreation(t *testing.T) {
	node := &DAGNode{
		ID: "node-1",
		Task: &SwarmTask{
			ID:     "task-1",
			Prompt: "Test task",
		},
		Status: DAGNodePending,
	}

	assert.Equal(t, "node-1", node.ID)
	assert.Equal(t, "task-1", node.Task.ID)
	assert.Equal(t, DAGNodePending, node.Status)
}

func TestDAGNodeStatuses(t *testing.T) {
	statuses := []DAGNodeStatus{
		DAGNodePending,
		DAGNodeReady,
		DAGNodeRunning,
		DAGNodeCompleted,
		DAGNodeFailed,
		DAGNodeSkipped,
	}

	for _, status := range statuses {
		assert.NotEmpty(t, string(status))
	}
}
