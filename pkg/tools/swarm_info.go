// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package tools

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
)

// SwarmInfoTool provides information about the PicoClaw swarm environment
type SwarmInfoTool struct {
	nodeID       string
	nodeRole     string
	swarmConfig string
	hid          string
	sid          string
	workers      map[string]WorkerInfo
	mu           sync.RWMutex
}

type WorkerInfo struct {
	ID           string   `json:"id"`
	Role         string   `json:"role"`
	Capabilities []string `json:"capabilities"`
	WorkDir      string   `json:"work_dir"`
	Status       string   `json:"status"`
	Host         string   `json:"host"`
	Port         int      `json:"port"`
}

// NewSwarmInfoTool creates a new swarm info tool
func NewSwarmInfoTool() *SwarmInfoTool {
	return &SwarmInfoTool{
		workers: make(map[string]WorkerInfo),
	}
}

func (t *SwarmInfoTool) Name() string {
	return "swarm_info"
}

func (t *SwarmInfoTool) Description() string {
	return "CRITICAL: Call this FIRST when asked about workers, nodes, or their directories. Returns complete information about all swarm nodes including: node IDs (e.g., claw-xxx), roles (coordinator/worker), work directories (e.g., /Users/dev/service/worker-a), and capabilities. ALWAYS use this before accessing worker directories to understand the swarm layout."
}

func (t *SwarmInfoTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Optional: specific query like 'workdirs', 'nodes', 'current', 'all'",
			},
		},
	}
}

func (t *SwarmInfoTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	t.mu.Lock()
	defer t.mu.Unlock()

	query := ""
	if q, ok := args["query"].(string); ok {
		query = q
	}

	// Try to load worker info from known locations
	t.discoverWorkers()

	var result strings.Builder

	if query == "" || query == "all" {
		result.WriteString("🦀 PicoClaw Swarm Environment\n\n")
		result.WriteString(t.formatAllInfo())
	} else if query == "workdirs" {
		result.WriteString("📁 Worker Directories:\n\n")
		result.WriteString(t.formatWorkDirs())
	} else if query == "nodes" {
		result.WriteString("🔗 Swarm Nodes:\n\n")
		result.WriteString(t.formatNodes())
	} else if query == "current" {
		result.WriteString("📍 Current Node:\n\n")
		result.WriteString(t.formatCurrentNode())
	} else {
		result.WriteString("🦀 PicoClaw Swarm Environment\n\n")
		result.WriteString(t.formatAllInfo())
	}

	return &ToolResult{
		ForLLM:  result.String(),
		ForUser: result.String(),
		IsError: false,
	}
}

func (t *SwarmInfoTool) discoverWorkers() {
	// Add known workers from config
	workers := []WorkerInfo{
		{
			ID:           "coordinator",
			Role:         "coordinator",
			Capabilities: []string{"orchestration", "scheduling"},
			WorkDir:      "/Users/dev/service/coordinator",
			Status:       "online",
			Host:         "localhost",
		},
		{
			ID:           "worker-a",
			Role:         "worker",
			Capabilities: []string{"code", "macos"},
			WorkDir:      "/Users/dev/service/worker-a",
			Status:       "online",
			Host:         "localhost",
		},
		{
			ID:           "worker-b",
			Role:         "worker",
			Capabilities: []string{"search", "windows"},
			WorkDir:      "/Users/dev/service/worker-b",
			Status:       "online",
			Host:         "localhost",
		},
	}

	for _, w := range workers {
		t.workers[w.ID] = w
	}
}

func (t *SwarmInfoTool) formatAllInfo() string {
	var s strings.Builder

	s.WriteString("📋 SWARM NODES DIRECTORY MAP\n\n")

	s.WriteString("When asked about worker directories, use these exact paths:\n\n")

	for _, w := range t.workers {
		s.WriteString(fmt.Sprintf("【%s】%s\n", strings.ToUpper(w.ID), w.ID))
		s.WriteString(fmt.Sprintf("  Role: %s\n", w.Role))
		s.WriteString(fmt.Sprintf("  WorkDir: %s\n", w.WorkDir))
		s.WriteString(fmt.Sprintf("  Capabilities: %s\n", strings.Join(w.Capabilities, ", ")))
		s.WriteString(fmt.Sprintf("  → To list files: ls %s\n\n", w.WorkDir))
	}

	s.WriteString("⚠️ IMPORTANT:\n")
	s.WriteString("  - Each worker has its own work directory\n")
	s.WriteString("  - Use the full path (e.g., /Users/dev/service/worker-a) when accessing files\n")
	s.WriteString("  - Do not use relative paths or 'worker' subdirectory\n")

	return s.String()
}

func (t *SwarmInfoTool) formatWorkDirs() string {
	var s strings.Builder
	for _, w := range t.workers {
		s.WriteString(fmt.Sprintf("%s: %s\n", w.ID, w.WorkDir))
		s.WriteString(fmt.Sprintf("  → ls %s\n", w.WorkDir))
	}
	return s.String()
}

func (t *SwarmInfoTool) formatNodes() string {
	var s strings.Builder
	for _, w := range t.workers {
		s.WriteString(fmt.Sprintf("- %s: %s\n", w.ID, w.Role))
		s.WriteString(fmt.Sprintf("  Capabilities: %v\n", w.Capabilities))
		s.WriteString(fmt.Sprintf("  WorkDir: %s\n", w.WorkDir))
	}
	return s.String()
}

func (t *SwarmInfoTool) formatCurrentNode() string {
	// Try to detect current node by checking working directory
	cwd, _ := os.Getwd()
	var s strings.Builder

	s.WriteString(fmt.Sprintf("Working Directory: %s\n", cwd))

	// Determine which node this is based on path
	if strings.Contains(cwd, "coordinator") {
		s.WriteString("\nDetected: coordinator node\n")
		s.WriteString("WorkDir: /Users/dev/service/coordinator\n")
	} else if strings.Contains(cwd, "worker-a") {
		s.WriteString("\nDetected: worker-a node\n")
		s.WriteString("WorkDir: /Users/dev/service/worker-a\n")
	} else if strings.Contains(cwd, "worker-b") {
		s.WriteString("\nDetected: worker-b node\n")
		s.WriteString("WorkDir: /Users/dev/service/worker-b\n")
	} else {
		s.WriteString("\nDetected: gateway/workspace node\n")
		s.WriteString("WorkDir: /Users/dev/workspace\n")
	}

	return s.String()
}

// SetNodeInfo sets the current node information
func (t *SwarmInfoTool) SetNodeInfo(nodeID, nodeRole, hid, sid string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.nodeID = nodeID
	t.nodeRole = nodeRole
	t.hid = hid
	t.sid = sid
}

// AddWorker registers a worker's information
func (t *SwarmInfoTool) AddWorker(id, role string, capabilities []string, workDir string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.workers[id] = WorkerInfo{
		ID:           id,
		Role:         role,
		Capabilities: capabilities,
		WorkDir:      workDir,
		Status:       "online",
		Host:         "localhost",
	}
}
