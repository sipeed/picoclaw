// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDAG(t *testing.T) {
	dag := NewDAG()
	assert.NotNil(t, dag)
	assert.NotNil(t, dag.nodes)
	assert.NotNil(t, dag.edges)
	assert.Equal(t, 0, dag.NodeCount())
}

func TestDAG_AddNode(t *testing.T) {
	dag := NewDAG()

	node := &DAGNode{
		ID: "node-1",
		Task: &SwarmTask{
			ID:     "task-1",
			Prompt: "Task 1",
		},
		Status: DAGNodePending,
	}

	err := dag.AddNode(node)
	require.NoError(t, err)
	assert.Equal(t, 1, dag.NodeCount())

	// Try adding duplicate node
	err = dag.AddNode(node)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestDAG_AddDependency(t *testing.T) {
	dag := NewDAG()

	node1 := &DAGNode{ID: "node-1", Task: &SwarmTask{ID: "task-1"}, Status: DAGNodePending}
	node2 := &DAGNode{ID: "node-2", Task: &SwarmTask{ID: "task-2"}, Status: DAGNodePending}

	require.NoError(t, dag.AddNode(node1))
	require.NoError(t, dag.AddNode(node2))

	// Add dependency: node-2 depends on node-1
	err := dag.AddDependency("node-1", "node-2")
	require.NoError(t, err)

	// Check that dependency was added
	retrieved, _ := dag.GetNode("node-2")
	assert.Contains(t, retrieved.Dependencies, "node-1")
}

func TestDAG_AddDependency_CycleDetection(t *testing.T) {
	dag := NewDAG()

	node1 := &DAGNode{ID: "node-1", Task: &SwarmTask{ID: "task-1"}, Status: DAGNodePending}
	node2 := &DAGNode{ID: "node-2", Task: &SwarmTask{ID: "task-2"}, Status: DAGNodePending}
	node3 := &DAGNode{ID: "node-3", Task: &SwarmTask{ID: "task-3"}, Status: DAGNodePending}

	require.NoError(t, dag.AddNode(node1))
	require.NoError(t, dag.AddNode(node2))
	require.NoError(t, dag.AddNode(node3))

	// Create dependencies: node-1 -> node-2 -> node-3
	require.NoError(t, dag.AddDependency("node-1", "node-2"))
	require.NoError(t, dag.AddDependency("node-2", "node-3"))

	// Try to create a cycle: node-3 -> node-1
	err := dag.AddDependency("node-3", "node-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cycle")
}

func TestDAG_Validate(t *testing.T) {
	t.Run("valid DAG", func(t *testing.T) {
		dag := NewDAG()

		node1 := &DAGNode{ID: "node-1", Task: &SwarmTask{ID: "task-1"}, Status: DAGNodePending}
		node2 := &DAGNode{ID: "node-2", Task: &SwarmTask{ID: "task-2"}, Status: DAGNodePending}

		require.NoError(t, dag.AddNode(node1))
		require.NoError(t, dag.AddNode(node2))
		require.NoError(t, dag.AddDependency("node-1", "node-2"))

		err := dag.Validate()
		assert.NoError(t, err)
	})

	t.Run("DAG with cycle", func(t *testing.T) {
		dag := NewDAG()

		node1 := &DAGNode{ID: "node-1", Task: &SwarmTask{ID: "task-1"}, Status: DAGNodePending}
		node2 := &DAGNode{ID: "node-2", Task: &SwarmTask{ID: "task-2"}, Status: DAGNodePending}

		require.NoError(t, dag.AddNode(node1))
		require.NoError(t, dag.AddNode(node2))

		// Manually create a cycle
		node1.Dependencies = []string{"node-2"}
		node2.Dependencies = []string{"node-1"}

		err := dag.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cycle")
	})
}

func TestDAG_GetReadyNodes(t *testing.T) {
	dag := NewDAG()

	node1 := &DAGNode{ID: "node-1", Task: &SwarmTask{ID: "task-1"}, Status: DAGNodePending}
	node2 := &DAGNode{ID: "node-2", Task: &SwarmTask{ID: "task-2"}, Status: DAGNodePending, Dependencies: []string{"node-1"}}
	node3 := &DAGNode{ID: "node-3", Task: &SwarmTask{ID: "task-3"}, Status: DAGNodePending, Dependencies: []string{"node-1"}}

	require.NoError(t, dag.AddNode(node1))
	require.NoError(t, dag.AddNode(node2))
	require.NoError(t, dag.AddNode(node3))

	// Initially only node-1 is ready (no dependencies)
	ready := dag.GetReadyNodes()
	assert.Len(t, ready, 1)
	assert.Equal(t, "node-1", ready[0].ID)

	// Mark node-1 as completed
	dag.UpdateNodeStatus("node-1", DAGNodeCompleted, "done", nil)

	// Now node-2 and node-3 should be ready
	ready = dag.GetReadyNodes()
	assert.Len(t, ready, 2)
}

func TestDAG_IsComplete(t *testing.T) {
	dag := NewDAG()

	node1 := &DAGNode{ID: "node-1", Task: &SwarmTask{ID: "task-1"}, Status: DAGNodePending}

	require.NoError(t, dag.AddNode(node1))

	assert.False(t, dag.IsComplete())

	dag.UpdateNodeStatus("node-1", DAGNodeCompleted, "done", nil)
	assert.True(t, dag.IsComplete())
}

func TestDAG_HasFailed(t *testing.T) {
	dag := NewDAG()

	node1 := &DAGNode{ID: "node-1", Task: &SwarmTask{ID: "task-1"}, Status: DAGNodePending}

	require.NoError(t, dag.AddNode(node1))

	assert.False(t, dag.HasFailed())

	dag.UpdateNodeStatus("node-1", DAGNodeFailed, "", assert.AnError)
	assert.True(t, dag.HasFailed())
}

func TestDAG_GetCompletedNodes(t *testing.T) {
	dag := NewDAG()

	node1 := &DAGNode{ID: "node-1", Task: &SwarmTask{ID: "task-1"}, Status: DAGNodePending}
	node2 := &DAGNode{ID: "node-2", Task: &SwarmTask{ID: "task-2"}, Status: DAGNodePending}

	require.NoError(t, dag.AddNode(node1))
	require.NoError(t, dag.AddNode(node2))

	dag.UpdateNodeStatus("node-1", DAGNodeCompleted, "done", nil)
	dag.UpdateNodeStatus("node-2", DAGNodeRunning, "", nil)

	completed := dag.GetCompletedNodes()
	assert.Len(t, completed, 1)
	assert.Equal(t, "node-1", completed[0].ID)
}

func TestDAG_GetFailedNodes(t *testing.T) {
	dag := NewDAG()

	node1 := &DAGNode{ID: "node-1", Task: &SwarmTask{ID: "task-1"}, Status: DAGNodePending}
	node2 := &DAGNode{ID: "node-2", Task: &SwarmTask{ID: "task-2"}, Status: DAGNodePending}

	require.NoError(t, dag.AddNode(node1))
	require.NoError(t, dag.AddNode(node2))

	dag.UpdateNodeStatus("node-1", DAGNodeFailed, "", assert.AnError)
	dag.UpdateNodeStatus("node-2", DAGNodeCompleted, "done", nil)

	failed := dag.GetFailedNodes()
	assert.Len(t, failed, 1)
	assert.Equal(t, "node-1", failed[0].ID)
}

func TestDAGNodeStatus(t *testing.T) {
	statuses := []DAGNodeStatus{
		DAGNodePending,
		DAGNodeReady,
		DAGNodeRunning,
		DAGNodeCompleted,
		DAGNodeFailed,
		DAGNodeSkipped,
	}

	for _, status := range statuses {
		assert.NotEmpty(t, string(status), "Status should not be empty")
	}
}

func TestBuildDAGFromTasks(t *testing.T) {
	tasks := []*DAGNode{
		{
			ID:  "task-1",
			Task: &SwarmTask{ID: "task-1", Prompt: "Task 1"},
		},
		{
			ID:  "task-2",
			Task: &SwarmTask{ID: "task-2", Prompt: "Task 2"},
			Dependencies: []string{"task-1"},
		},
		{
			ID:  "task-3",
			Task: &SwarmTask{ID: "task-3", Prompt: "Task 3"},
			Dependencies: []string{"task-1"},
		},
	}

	dag, err := BuildDAGFromTasks(tasks)
	require.NoError(t, err)
	assert.Equal(t, 3, dag.NodeCount())

	// Verify dependencies
	task2, _ := dag.GetNode("task-2")
	assert.Contains(t, task2.Dependencies, "task-1")

	task3, _ := dag.GetNode("task-3")
	assert.Contains(t, task3.Dependencies, "task-1")
}
