package runtime

import (
	"context"
	"fmt"

	"github.com/sipeed/picoclaw/pkg/swarm/core"
)

type MemorySaveTool struct {
	memory  core.SharedMemory
	swarmID core.SwarmID
}

func NewMemorySaveTool(mem core.SharedMemory, swarmID core.SwarmID) *MemorySaveTool {
	return &MemorySaveTool{
		memory:  mem,
		swarmID: swarmID,
	}
}

func (t *MemorySaveTool) Name() string {
	return "save_memory"
}

func (t *MemorySaveTool) Description() string {
	return "Save a fact or finding to the swarm's long-term memory for other agents to use."
}

func (t *MemorySaveTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"content": map[string]interface{}{
				"type":        "string",
				"description": "The fact or information to save",
			},
			"tags": map[string]interface{}{
				"type":        "string",
				"description": "Comma-separated tags for organization",
			},
		},
		"required": []string{"content"},
	}
}

func (t *MemorySaveTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	content, ok := args["content"].(string)
	if !ok {
		return "", fmt.Errorf("content is required")
	}
	
	tagsStr, _ := args["tags"].(string)
	metadata := map[string]any{
		"tags": tagsStr,
	}

	fact := core.Fact{
		SwarmID:    t.swarmID,
		Content:    content,
		Confidence: 1.0, // Self-reported facts are trusted
		Source:     "agent_tool",
		Metadata:   metadata,
	}

	if err := t.memory.SaveFact(ctx, fact); err != nil {
		return "", fmt.Errorf("failed to save memory: %w", err)
	}

	return "Fact saved to memory.", nil
}

type MemorySearchTool struct {
	memory  core.SharedMemory
	swarmID core.SwarmID
	global  bool
}

func NewMemorySearchTool(mem core.SharedMemory, swarmID core.SwarmID, global bool) *MemorySearchTool {
	return &MemorySearchTool{
		memory:  mem,
		swarmID: swarmID,
		global:  global,
	}
}

func (t *MemorySearchTool) Name() string {
	return "search_memory"
}

func (t *MemorySearchTool) Description() string {
	return "Search the swarm's long-term memory for relevant facts."
}

func (t *MemorySearchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "The search query",
			},
		},
		"required": []string{"query"},
	}
}

func (t *MemorySearchTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	query, ok := args["query"].(string)
	if !ok {
		return "", fmt.Errorf("query is required")
	}

	results, err := t.memory.SearchFacts(ctx, t.swarmID, query, 5, t.global)
	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}

	if len(results) == 0 {
		return "No relevant facts found.", nil
	}

	out := "Found facts:\n"
	for _, r := range results {
		out += fmt.Sprintf("- %s (Score: %.2f)\n", r.Content, r.Score)
	}
	return out, nil
}
