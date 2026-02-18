// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

// Package multi provides the foundation for multi-agent collaboration.
// It defines the Agent interface, shared context, and agent registry
// that enable multiple specialized agents to work together within
// a single PicoClaw session.
//
// This package is designed to be non-invasive: it introduces new
// abstractions without modifying the existing AgentLoop or SubagentManager.
// The existing subagent system can be gradually migrated to use these
// interfaces.
package multi

import (
	"context"

	"github.com/sipeed/picoclaw/pkg/tools"
)

// Agent defines the interface that all agents must implement.
// Each agent has a unique name, a role description, a system prompt,
// and a set of capabilities that determine which tasks it can handle.
type Agent interface {
	// Name returns the unique identifier for this agent.
	Name() string

	// Role returns a human-readable description of what this agent does.
	Role() string

	// SystemPrompt returns the system prompt used when this agent
	// interacts with the LLM.
	SystemPrompt() string

	// Capabilities returns the list of capability tags this agent supports.
	// These are used by the registry to match agents to tasks.
	// Example: ["code", "search", "file_operations"]
	Capabilities() []string

	// Tools returns the tool registry available to this agent.
	// Each agent can have a different set of tools.
	Tools() *tools.ToolRegistry

	// Execute runs the agent on the given task within the provided context.
	// The shared context allows reading/writing data visible to other agents.
	// Returns the agent's response content and any error.
	Execute(ctx context.Context, task string, shared *SharedContext) (string, error)
}

// AgentConfig holds configuration for creating a BaseAgent.
type AgentConfig struct {
	// Name is the unique identifier for the agent.
	Name string

	// Role describes what this agent specializes in.
	Role string

	// SystemPrompt is the prompt sent to the LLM.
	SystemPrompt string

	// Capabilities lists the capability tags.
	Capabilities []string
}

// BaseAgent provides a minimal Agent implementation that can be embedded
// in concrete agent types. It handles the common fields (name, role, prompt,
// capabilities) and leaves Execute to be implemented by the concrete type.
type BaseAgent struct {
	config AgentConfig
	tools  *tools.ToolRegistry
}

// NewBaseAgent creates a new BaseAgent with the given configuration.
func NewBaseAgent(cfg AgentConfig, registry *tools.ToolRegistry) *BaseAgent {
	if registry == nil {
		registry = tools.NewToolRegistry()
	}
	return &BaseAgent{
		config: cfg,
		tools:  registry,
	}
}

func (a *BaseAgent) Name() string             { return a.config.Name }
func (a *BaseAgent) Role() string             { return a.config.Role }
func (a *BaseAgent) SystemPrompt() string     { return a.config.SystemPrompt }
func (a *BaseAgent) Capabilities() []string   { return a.config.Capabilities }
func (a *BaseAgent) Tools() *tools.ToolRegistry { return a.tools }

// HandoffRequest represents a request to delegate a task from one agent
// to another. It carries the task description and optional metadata
// for routing.
type HandoffRequest struct {
	// FromAgent is the name of the agent delegating the task.
	FromAgent string

	// ToAgent is the name of the target agent. If empty, the registry
	// will select the best agent based on RequiredCapability.
	ToAgent string

	// RequiredCapability is used for capability-based routing when
	// ToAgent is not specified.
	RequiredCapability string

	// Task is the description of what needs to be done.
	Task string

	// Context carries additional key-value data for the target agent.
	Context map[string]interface{}
}

// HandoffResult contains the outcome of a hand-off operation.
type HandoffResult struct {
	// AgentName is the name of the agent that handled the task.
	AgentName string

	// Content is the response produced by the agent.
	Content string

	// Err is set if the hand-off or execution failed.
	Err error
}
