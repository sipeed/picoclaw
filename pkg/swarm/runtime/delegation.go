package runtime

import (
	"context"
	"fmt"

	"github.com/sipeed/picoclaw/pkg/swarm/core"
)

// DelegateTool allows a node to spawn a child worker and wait for the result
type DelegateTool struct {
	orchestrator *Orchestrator
	parentNodeID core.NodeID
	swarmID      core.SwarmID
}

func NewDelegateTool(orch *Orchestrator, swarmID core.SwarmID, parentID core.NodeID) *DelegateTool {
	return &DelegateTool{
		orchestrator: orch,
		swarmID:      swarmID,
		parentNodeID: parentID,
	}
}

func (t *DelegateTool) Name() string {
	return "delegate_task"
}

func (t *DelegateTool) Description() string {
	return "Delegate a sub-task to a specialized worker (Researcher, Writer, Analyst, etc.). Returns the worker's output."
}

func (t *DelegateTool) Parameters() map[string]interface{} {
	roles := make([]string, 0)
	for roleName := range t.orchestrator.config.Roles {
		roles = append(roles, roleName)
	}
	if len(roles) == 0 {
		roles = []string{"Researcher", "Analyst", "Writer", "Critic"} // Fallback
	}

	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"role": map[string]interface{}{
				"type":        "string",
				"description": "The role of the worker to spawn",
				"enum":        roles,
			},
			"task": map[string]interface{}{
				"type":        "string",
				"description": "Detailed instruction for the worker",
			},
		},
		"required": []string{"role", "task"},
	}
}

func (t *DelegateTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	roleName, ok := args["role"].(string)
	if !ok {
		return "", fmt.Errorf("role is required")
	}
	task, ok := args["task"].(string)
	if !ok {
		return "", fmt.Errorf("task is required")
	}

	// Request Orchestrator to spawn a worker
	// This is a blocking call that waits for the worker to finish
	result, err := t.orchestrator.RunSubTask(ctx, t.swarmID, t.parentNodeID, roleName, task)
	if err != nil {
		return "", fmt.Errorf("delegation failed: %w", err)
	}

	return result, nil
}
