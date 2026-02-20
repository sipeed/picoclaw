// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// SpecialistNode is a worker that specializes in specific capabilities
type SpecialistNode struct {
	*Worker
	registry *CapabilityRegistry
	skillsDir string
}

// NewSpecialistNode creates a new specialist node
func NewSpecialistNode(
	cfg *config.SwarmConfig,
	bridge *NATSBridge,
	temporal *TemporalClient,
	agentLoop *agent.AgentLoop,
	provider providers.LLMProvider,
	nodeInfo *NodeInfo,
	js nats.JetStreamContext,
	nc *nats.Conn,
	skillsDir string,
) *SpecialistNode {
	registry := NewCapabilityRegistry(nodeInfo, js, nc)

	// Create base worker
	worker := NewWorker(cfg, bridge, temporal, agentLoop, provider, nodeInfo)

	return &SpecialistNode{
		Worker:    worker,
		registry:  registry,
		skillsDir: skillsDir,
	}
}

// Start initializes and starts the specialist node
func (s *SpecialistNode) Start(ctx context.Context) error {
	// Initialize capability registry
	if err := s.registry.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize capability registry: %w", err)
	}

	// Perform dynamic capability discovery
	if err := s.DynamicCapabilityDiscovery(); err != nil {
		logger.WarnCF("swarm", "Dynamic capability discovery failed", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Start base worker
	if err := s.Worker.Start(ctx); err != nil {
		return fmt.Errorf("failed to start worker: %w", err)
	}

	logger.InfoCF("swarm", "Specialist node started", map[string]interface{}{
		"node_id":      s.nodeInfo.ID,
		"capabilities": fmt.Sprintf("%v", s.nodeInfo.Capabilities),
	})

	return nil
}

// DynamicCapabilityDiscovery scans for skills and registers them as capabilities
func (s *SpecialistNode) DynamicCapabilityDiscovery() error {
	if s.skillsDir == "" {
		// Use default skills directory
		s.skillsDir = filepath.Join(os.Getenv("HOME"), ".picoclaw", "skills")
	}

	// Check if directory exists
	if _, err := os.Stat(s.skillsDir); os.IsNotExist(err) {
		logger.DebugC("swarm", "Skills directory does not exist, skipping discovery")
		return nil
	}

	logger.InfoCF("swarm", "Starting dynamic capability discovery", map[string]interface{}{
		"directory": s.skillsDir,
	})

	// Walk the skills directory
	err := filepath.Walk(s.skillsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue on error
		}

		// Look for .md files which typically define skills
		if !info.IsDir() && strings.HasSuffix(path, ".md") {
			if err := s.discoverSkillFromFile(path); err != nil {
				logger.DebugCF("swarm", "Failed to discover skill", map[string]interface{}{
					"path":  path,
					"error": err.Error(),
				})
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk skills directory: %w", err)
	}

	return nil
}

// discoverSkillFromFile parses a skill file and registers it as a capability
func (s *SpecialistNode) discoverSkillFromFile(path string) error {
	// Read file content
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Extract skill name from filename
	skillName := strings.TrimSuffix(filepath.Base(path), ".md")

	// Parse metadata from content
	// Skills typically have frontmatter or specific patterns
	metadata := s.extractSkillMetadata(string(content))

	// Register as capability
	description := metadata["description"]
	if description == "" {
		description = fmt.Sprintf("Skill: %s", skillName)
	}

	version := metadata["version"]
	if version == "" {
		version = "1.0.0"
	}

	// Build metadata map
	metaMap := make(map[string]interface{})
	metaMap["file"] = path
	for k, v := range metadata {
		metaMap[k] = v
	}

	if err := s.registry.Register(skillName, description, version, metaMap); err != nil {
		return fmt.Errorf("failed to register skill %s: %w", skillName, err)
	}

	logger.InfoCF("swarm", "Discovered and registered skill", map[string]interface{}{
		"skill":      skillName,
		"description": description,
		"version":     version,
	})

	return nil
}

// extractSkillMetadata extracts metadata from skill content
func (s *SpecialistNode) extractSkillMetadata(content string) map[string]string {
	metadata := make(map[string]string)

	lines := strings.Split(content, "\n")

	// Look for metadata patterns
	// Pattern 1: Key: Value at the start
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, ":") && !strings.HasPrefix(line, "#") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(strings.ToLower(parts[0]))
				value := strings.TrimSpace(parts[1])

				// Recognize common metadata keys
				switch key {
				case "description", "desc", "summary":
					metadata["description"] = value
				case "version", "ver":
					metadata["version"] = value
				case "category", "type":
					metadata["category"] = value
				case "author":
					metadata["author"] = value
				case "tags":
					metadata["tags"] = value
				}
			}
		}

		// Only check first 20 lines for metadata
		if strings.HasPrefix(line, "#") && len(metadata) > 0 {
			// Found a heading, metadata section likely ended
			break
		}
	}

	// Extract description from first heading if not found
	if metadata["description"] == "" {
		for _, line := range lines {
			if strings.HasPrefix(line, "#") {
				metadata["description"] = strings.TrimSpace(strings.TrimPrefix(line, "#"))
				break
			}
		}
	}

	return metadata
}

// RegisterCapability manually registers a capability
func (s *SpecialistNode) RegisterCapability(name, description, version string, metadata map[string]interface{}) error {
	return s.registry.Register(name, description, version, metadata)
}

// GetCapabilities returns all registered capabilities
func (s *SpecialistNode) GetCapabilities() []Capability {
	return s.registry.List()
}

// HasCapability checks if the specialist has a specific capability
func (s *SpecialistNode) HasCapability(name string) bool {
	_, ok := s.registry.Get(name)
	return ok
}

// DiscoverSwarmCapabilities finds capabilities across the swarm
func (s *SpecialistNode) DiscoverSwarmCapabilities(ctx context.Context, name, version string) ([]Capability, error) {
	return s.registry.Discover(ctx, name, version)
}

// ExecuteSpecializedTask executes a task that requires this specialist's capabilities
func (s *SpecialistNode) ExecuteSpecializedTask(ctx context.Context, task *SwarmTask) (string, error) {
	logger.InfoCF("swarm", "Executing specialized task", map[string]interface{}{
		"task_id":    task.ID,
		"capability": task.Capability,
	})

	// Check if we have this capability
	if !s.HasCapability(task.Capability) {
		return "", fmt.Errorf("specialist does not have capability: %s", task.Capability)
	}

	// Execute the task using the agent loop
	result, err := s.agentLoop.ProcessDirect(ctx, task.Prompt, "swarm:specialist:"+task.ID)
	if err != nil {
		return "", fmt.Errorf("specialized task execution failed: %w", err)
	}

	return result, nil
}

// HeartbeatWithCapabilities sends a heartbeat with capability information
func (s *SpecialistNode) HeartbeatWithCapabilities() error {
	// This would be called periodically to announce capabilities
	caps := s.GetCapabilities()

	capabilityNames := make([]string, len(caps))
	for i, cap := range caps {
		capabilityNames[i] = cap.Name
	}

	// Update node info with current capabilities
	s.nodeInfo.Capabilities = capabilityNames

	logger.DebugCF("swarm", "Heartbeat with capabilities", map[string]interface{}{
		"node_id":      s.nodeInfo.ID,
		"capabilities": capabilityNames,
	})

	return nil
}

// CapabilityHeartbeatLoop sends periodic capability announcements
func (s *SpecialistNode) CapabilityHeartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.HeartbeatWithCapabilities(); err != nil {
				logger.WarnCF("swarm", "Failed to send capability heartbeat", map[string]interface{}{
					"error": err.Error(),
				})
			}
		}
	}
}
