// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package multi

import (
	"context"
	"fmt"
	"sync"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// AgentState represents the current state of an agent in the registry.
type AgentState int

const (
	// AgentIdle means the agent is registered but not currently executing.
	AgentIdle AgentState = iota

	// AgentActive means the agent is currently executing a task.
	AgentActive
)

// agentEntry holds an agent and its runtime state within the registry.
type agentEntry struct {
	agent Agent
	state AgentState
}

// AgentRegistry manages the lifecycle of agents and provides capability-based
// routing for hand-off requests. It is the central coordinator for multi-agent
// collaboration within a session.
//
// Thread-safe: all operations are protected by a read-write mutex.
type AgentRegistry struct {
	mu      sync.RWMutex
	agents  map[string]*agentEntry
	shared  *SharedContext
}

// NewAgentRegistry creates a new AgentRegistry with a fresh SharedContext.
func NewAgentRegistry() *AgentRegistry {
	return &AgentRegistry{
		agents: make(map[string]*agentEntry),
		shared: NewSharedContext(),
	}
}

// Register adds an agent to the registry. Returns an error if an agent
// with the same name is already registered.
func (r *AgentRegistry) Register(agent Agent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := agent.Name()
	if _, exists := r.agents[name]; exists {
		return fmt.Errorf("agent %q already registered", name)
	}

	r.agents[name] = &agentEntry{
		agent: agent,
		state: AgentIdle,
	}

	logger.InfoCF("multi", "Agent registered",
		map[string]interface{}{
			"name":         name,
			"role":         agent.Role(),
			"capabilities": agent.Capabilities(),
		})

	return nil
}

// Unregister removes an agent from the registry.
// Returns an error if the agent is currently active.
func (r *AgentRegistry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, exists := r.agents[name]
	if !exists {
		return fmt.Errorf("agent %q not found", name)
	}

	if entry.state == AgentActive {
		return fmt.Errorf("cannot unregister active agent %q", name)
	}

	delete(r.agents, name)
	return nil
}

// Get returns the agent with the given name, or nil if not found.
func (r *AgentRegistry) Get(name string) Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, ok := r.agents[name]
	if !ok {
		return nil
	}
	return entry.agent
}

// List returns the names of all registered agents.
func (r *AgentRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.agents))
	for name := range r.agents {
		names = append(names, name)
	}
	return names
}

// FindByCapability returns all agents that have the specified capability.
func (r *AgentRegistry) FindByCapability(capability string) []Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matches []Agent
	for _, entry := range r.agents {
		for _, cap := range entry.agent.Capabilities() {
			if cap == capability {
				matches = append(matches, entry.agent)
				break
			}
		}
	}
	return matches
}

// SharedContext returns the registry's shared context instance.
func (r *AgentRegistry) SharedContext() *SharedContext {
	return r.shared
}

// Handoff delegates a task to another agent based on the HandoffRequest.
//
// Routing logic:
//  1. If ToAgent is specified, route directly to that agent.
//  2. If RequiredCapability is specified, find the first idle agent
//     with that capability.
//  3. If no suitable agent is found, return an error.
//
// The hand-off records events in the shared context for traceability.
func (r *AgentRegistry) Handoff(ctx context.Context, req HandoffRequest) *HandoffResult {
	// Inject hand-off context data into shared context
	if req.Context != nil {
		for k, v := range req.Context {
			r.shared.Set(k, v)
		}
	}

	// Record the hand-off request as an event
	r.shared.AddEvent(req.FromAgent, "handoff",
		fmt.Sprintf("delegating task to %s (capability: %s): %s",
			req.ToAgent, req.RequiredCapability, req.Task))

	// Resolve target agent
	target, err := r.resolveTarget(req)
	if err != nil {
		r.shared.AddEvent(req.FromAgent, "error", err.Error())
		return &HandoffResult{Err: err}
	}

	// Mark agent as active
	r.setAgentState(target.Name(), AgentActive)
	defer r.setAgentState(target.Name(), AgentIdle)

	logger.InfoCF("multi", "Executing hand-off",
		map[string]interface{}{
			"from":       req.FromAgent,
			"to":         target.Name(),
			"task_len":   len(req.Task),
			"capability": req.RequiredCapability,
		})

	// Execute the target agent
	content, execErr := target.Execute(ctx, req.Task, r.shared)

	// Record the result
	eventType := "result"
	eventContent := content
	if execErr != nil {
		eventType = "error"
		eventContent = execErr.Error()
	}
	r.shared.AddEvent(target.Name(), eventType, eventContent)

	return &HandoffResult{
		AgentName: target.Name(),
		Content:   content,
		Err:       execErr,
	}
}

// resolveTarget finds the appropriate agent for a hand-off request.
func (r *AgentRegistry) resolveTarget(req HandoffRequest) (Agent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Direct routing by name
	if req.ToAgent != "" {
		entry, ok := r.agents[req.ToAgent]
		if !ok {
			return nil, fmt.Errorf("target agent %q not found", req.ToAgent)
		}
		return entry.agent, nil
	}

	// Capability-based routing: find first idle agent with the capability
	if req.RequiredCapability != "" {
		for _, entry := range r.agents {
			if entry.state != AgentIdle {
				continue
			}
			for _, cap := range entry.agent.Capabilities() {
				if cap == req.RequiredCapability {
					return entry.agent, nil
				}
			}
		}
		return nil, fmt.Errorf("no idle agent found with capability %q", req.RequiredCapability)
	}

	return nil, fmt.Errorf("hand-off request must specify ToAgent or RequiredCapability")
}

// setAgentState updates the state of an agent in the registry.
func (r *AgentRegistry) setAgentState(name string, state AgentState) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if entry, ok := r.agents[name]; ok {
		entry.state = state
	}
}

// GetAgentState returns the current state of an agent.
func (r *AgentRegistry) GetAgentState(name string) (AgentState, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.agents[name]
	if !ok {
		return AgentIdle, false
	}
	return entry.state, true
}
