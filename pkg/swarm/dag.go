// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// DAG represents a directed acyclic graph for workflow execution
type DAG struct {
	nodes map[string]*DAGNode
	edges map[string][]string // adjacency list: node -> dependents
	mu    sync.RWMutex
}

// NewDAG creates a new empty DAG
func NewDAG() *DAG {
	return &DAG{
		nodes: make(map[string]*DAGNode),
		edges: make(map[string][]string),
	}
}

// AddNode adds a node to the DAG
func (d *DAG) AddNode(node *DAGNode) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if _, exists := d.nodes[node.ID]; exists {
		return fmt.Errorf("node %s already exists", node.ID)
	}

	node.Status = DAGNodePending
	d.nodes[node.ID] = node
	d.edges[node.ID] = []string{}

	return nil
}

// AddDependency adds a dependency edge between nodes
func (d *DAG) AddDependency(from, to string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if _, exists := d.nodes[from]; !exists {
		return fmt.Errorf("node %s does not exist", from)
	}
	if _, exists := d.nodes[to]; !exists {
		return fmt.Errorf("node %s does not exist", to)
	}

	// Check for cycles
	if d.wouldCreateCycle(from, to) {
		return fmt.Errorf("adding edge %s -> %s would create a cycle", from, to)
	}

	// Add edge and update node dependencies
	d.edges[from] = append(d.edges[from], to)
	d.nodes[to].Dependencies = append(d.nodes[to].Dependencies, from)

	return nil
}

// wouldCreateCycle checks if adding an edge would create a cycle
func (d *DAG) wouldCreateCycle(from, to string) bool {
	visited := make(map[string]bool)
	return d.hasPathDFS(to, from, visited)
}

// hasPathDFS performs DFS to check if a path exists
func (d *DAG) hasPathDFS(start, end string, visited map[string]bool) bool {
	if start == end {
		return true
	}
	visited[start] = true

	for _, neighbor := range d.edges[start] {
		if !visited[neighbor] {
			if d.hasPathDFS(neighbor, end, visited) {
				return true
			}
		}
	}

	return false
}

// Validate checks if the DAG is valid (no cycles, all dependencies exist)
func (d *DAG) Validate() error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Build adjacency list from both edges and Dependencies field
	adjacency := make(map[string][]string)
	for from, toList := range d.edges {
		adjacency[from] = append(adjacency[from], toList...)
	}
	// Also check Dependencies fields for any additional edges
	for nodeID, node := range d.nodes {
		for _, depID := range node.Dependencies {
			// Check if this creates a duplicate edge
			exists := false
			for _, existing := range adjacency[depID] {
				if existing == nodeID {
					exists = true
					break
				}
			}
			if !exists {
				adjacency[depID] = append(adjacency[depID], nodeID)
			}
		}
	}

	// Check for cycles using DFS
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var checkCycleDFS func(string) bool
	checkCycleDFS = func(nodeID string) bool {
		visited[nodeID] = true
		recStack[nodeID] = true

		for _, neighbor := range adjacency[nodeID] {
			if !visited[neighbor] {
				if checkCycleDFS(neighbor) {
					return true
				}
			} else if recStack[neighbor] {
				return true
			}
		}

		recStack[nodeID] = false
		return false
	}

	for nodeID := range d.nodes {
		if !visited[nodeID] {
			if checkCycleDFS(nodeID) {
				return fmt.Errorf("cycle detected in DAG")
			}
		}
	}

	// Validate all dependencies exist
	for nodeID, node := range d.nodes {
		for _, depID := range node.Dependencies {
			if _, exists := d.nodes[depID]; !exists {
				return fmt.Errorf("node %s depends on non-existent node %s", nodeID, depID)
			}
		}
	}

	return nil
}

// detectCycleDFS uses DFS with recursion stack to detect cycles
func (d *DAG) detectCycleDFS(nodeID string, visited, recStack map[string]bool) bool {
	visited[nodeID] = true
	recStack[nodeID] = true

	for _, neighbor := range d.edges[nodeID] {
		if !visited[neighbor] {
			if d.detectCycleDFS(neighbor, visited, recStack) {
				return true
			}
		} else if recStack[neighbor] {
			return true
		}
	}

	recStack[nodeID] = false
	return false
}

// GetReadyNodes returns all nodes that are ready to execute (dependencies satisfied)
func (d *DAG) GetReadyNodes() []*DAGNode {
	d.mu.RLock()
	defer d.mu.RUnlock()

	ready := make([]*DAGNode, 0)

	for _, node := range d.nodes {
		if node.Status == DAGNodePending {
			// Check if all dependencies are completed
			allDepsComplete := true
			for _, depID := range node.Dependencies {
				if depNode, exists := d.nodes[depID]; !exists || depNode.Status != DAGNodeCompleted {
					allDepsComplete = false
					break
				}
			}

			if allDepsComplete && len(node.Dependencies) > 0 {
				// Has dependencies and they're all complete
				ready = append(ready, node)
			} else if len(node.Dependencies) == 0 {
				// No dependencies - root node
				ready = append(ready, node)
			}
		}
	}

	return ready
}

// GetNode returns a node by ID
func (d *DAG) GetNode(id string) (*DAGNode, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	node, exists := d.nodes[id]
	return node, exists
}

// UpdateNodeStatus updates the status of a node
func (d *DAG) UpdateNodeStatus(id string, status DAGNodeStatus, result string, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if node, exists := d.nodes[id]; exists {
		node.Status = status
		node.Result = result
		if err != nil {
			node.Error = err.Error()
		}

		if status == DAGNodeRunning {
			node.StartedAt = time.Now().UnixMilli()
		} else if status == DAGNodeCompleted || status == DAGNodeFailed {
			node.CompletedAt = time.Now().UnixMilli()
		}
	}
}

// IsComplete returns true if all nodes are completed, failed, or skipped
func (d *DAG) IsComplete() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	for _, node := range d.nodes {
		if node.Status == DAGNodePending || node.Status == DAGNodeReady || node.Status == DAGNodeRunning {
			return false
		}
	}

	return true
}

// HasFailed returns true if any node failed
func (d *DAG) HasFailed() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	for _, node := range d.nodes {
		if node.Status == DAGNodeFailed {
			return true
		}
	}

	return false
}

// GetCompletedNodes returns all completed nodes
func (d *DAG) GetCompletedNodes() []*DAGNode {
	d.mu.RLock()
	defer d.mu.RUnlock()

	completed := make([]*DAGNode, 0)
	for _, node := range d.nodes {
		if node.Status == DAGNodeCompleted {
			completed = append(completed, node)
		}
	}

	return completed
}

// GetFailedNodes returns all failed nodes
func (d *DAG) GetFailedNodes() []*DAGNode {
	d.mu.RLock()
	defer d.mu.RUnlock()

	failed := make([]*DAGNode, 0)
	for _, node := range d.nodes {
		if node.Status == DAGNodeFailed {
			failed = append(failed, node)
		}
	}

	return failed
}

// NodeCount returns the total number of nodes
func (d *DAG) NodeCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return len(d.nodes)
}

// DAGExecutor executes DAG workflows with parallel execution support
type DAGExecutor struct {
	dag         *DAG
	coordinator *Coordinator
	maxParallel int
	nodeResults map[string]string
	mu          sync.Mutex
}

// NewDAGExecutor creates a new DAG executor
func NewDAGExecutor(dag *DAG, coordinator *Coordinator, maxParallel int) *DAGExecutor {
	if maxParallel <= 0 {
		maxParallel = 5 // Default parallelism
	}

	return &DAGExecutor{
		dag:         dag,
		coordinator: coordinator,
		maxParallel: maxParallel,
		nodeResults: make(map[string]string),
	}
}

// Execute runs the DAG workflow
func (e *DAGExecutor) Execute(ctx context.Context) (map[string]string, error) {
	logger.InfoCF("swarm", "Starting DAG execution", map[string]interface{}{
		"nodes":        e.dag.NodeCount(),
		"max_parallel": e.maxParallel,
	})

	// Validate DAG before execution
	if err := e.dag.Validate(); err != nil {
		return nil, fmt.Errorf("DAG validation failed: %w", err)
	}

	// Track running nodes with semaphore
	semaphore := make(chan struct{}, e.maxParallel)
	var wg sync.WaitGroup

	// Continue until DAG is complete
	for !e.dag.IsComplete() {
		// Check for failures
		if e.dag.HasFailed() {
			return e.nodeResults, fmt.Errorf("DAG execution failed: some nodes failed")
		}

		// Get ready nodes
		readyNodes := e.dag.GetReadyNodes()
		if len(readyNodes) == 0 {
			// No ready nodes but DAG not complete - might be waiting for running nodes
			if e.hasRunningNodes() {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			// No running nodes and no ready nodes - should be complete
			break
		}

		// Execute ready nodes in parallel
		for _, node := range readyNodes {
			// Mark as running
			e.dag.UpdateNodeStatus(node.ID, DAGNodeRunning, "", nil)

			wg.Add(1)
			go func(n *DAGNode) {
				defer wg.Done()

				// Acquire semaphore
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				// Execute the node's task
				result, err := e.executeNode(ctx, n)

				e.mu.Lock()
				e.nodeResults[n.ID] = result
				e.mu.Unlock()

				if err != nil {
					e.dag.UpdateNodeStatus(n.ID, DAGNodeFailed, "", err)
					logger.WarnCF("swarm", "DAG node failed", map[string]interface{}{
						"node_id": n.ID,
						"error":   err.Error(),
					})
				} else {
					e.dag.UpdateNodeStatus(n.ID, DAGNodeCompleted, result, nil)
					logger.DebugCF("swarm", "DAG node completed", map[string]interface{}{
						"node_id": n.ID,
					})
				}
			}(node)
		}

		// Wait a bit before checking for more ready nodes
		time.Sleep(50 * time.Millisecond)
	}

	// Wait for all running nodes to complete
	wg.Wait()

	if e.dag.HasFailed() {
		return e.nodeResults, fmt.Errorf("DAG execution completed with failures")
	}

	logger.InfoCF("swarm", "DAG execution completed successfully", map[string]interface{}{
		"nodes": len(e.nodeResults),
	})

	return e.nodeResults, nil
}

// executeNode executes a single DAG node
func (e *DAGExecutor) executeNode(ctx context.Context, node *DAGNode) (string, error) {
	// For now, execute through coordinator
	// In a full implementation, this could dispatch to specialist workers
	result, err := e.coordinator.executeLocally(ctx, node.Task)
	if err != nil {
		return "", err
	}
	return result.Result, nil
}

// hasRunningNodes checks if any nodes are currently running
func (e *DAGExecutor) hasRunningNodes() bool {
	_ = e.dag.GetReadyNodes() // Check for any ready nodes

	// Check actual node status in DAG
	e.dag.mu.RLock()
	defer e.dag.mu.RUnlock()

	for _, node := range e.dag.nodes {
		if node.Status == DAGNodeRunning {
			return true
		}
	}

	return false
}

// BuildDAGFromTasks creates a DAG from a list of tasks with dependencies
func BuildDAGFromTasks(tasks []*DAGNode) (*DAG, error) {
	dag := NewDAG()

	// Add all nodes first
	for _, task := range tasks {
		if err := dag.AddNode(task); err != nil {
			return nil, fmt.Errorf("failed to add node %s: %w", task.ID, err)
		}
	}

	// Add dependencies
	for _, task := range tasks {
		for _, dep := range task.Dependencies {
			if err := dag.AddDependency(dep, task.ID); err != nil {
				return nil, fmt.Errorf("failed to add dependency %s -> %s: %w", dep, task.ID, err)
			}
		}
	}

	return dag, nil
}
