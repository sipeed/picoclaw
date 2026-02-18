package multiagent

import (
	"context"
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/tools"
)

// BlackboardTool exposes the Blackboard to an LLM agent via the tool interface.
// Each instance is bound to a specific agent ID for authorship tracking.
type BlackboardTool struct {
	board   *Blackboard
	agentID string
}

// NewBlackboardTool creates a blackboard tool bound to a specific agent.
func NewBlackboardTool(board *Blackboard, agentID string) *BlackboardTool {
	return &BlackboardTool{
		board:   board,
		agentID: agentID,
	}
}

func (t *BlackboardTool) Name() string { return "blackboard" }

func (t *BlackboardTool) Description() string {
	return "Read, write, list, or delete entries in the shared context blackboard. " +
		"Use this to share information between agents in a multi-agent session."
}

func (t *BlackboardTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"read", "write", "list", "delete"},
				"description": "The action to perform on the blackboard",
			},
			"key": map[string]interface{}{
				"type":        "string",
				"description": "The key to read, write, or delete (not required for list)",
			},
			"value": map[string]interface{}{
				"type":        "string",
				"description": "The value to write (only required for write action)",
			},
		},
		"required": []string{"action"},
	}
}

func (t *BlackboardTool) Execute(_ context.Context, args map[string]interface{}) *tools.ToolResult {
	action, _ := args["action"].(string)
	key, _ := args["key"].(string)
	value, _ := args["value"].(string)

	switch strings.ToLower(action) {
	case "read":
		if key == "" {
			return tools.ErrorResult("key is required for read action")
		}
		entry := t.board.GetEntry(key)
		if entry == nil {
			return tools.NewToolResult(fmt.Sprintf("No entry found for key %q", key))
		}
		return tools.NewToolResult(fmt.Sprintf("Key: %s\nValue: %s\nAuthor: %s\nScope: %s",
			entry.Key, entry.Value, entry.Author, entry.Scope))

	case "write":
		if key == "" {
			return tools.ErrorResult("key is required for write action")
		}
		if value == "" {
			return tools.ErrorResult("value is required for write action")
		}
		t.board.Set(key, value, t.agentID)
		return tools.NewToolResult(fmt.Sprintf("Written key %q to blackboard", key))

	case "list":
		keys := t.board.List()
		if len(keys) == 0 {
			return tools.NewToolResult("Blackboard is empty")
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Blackboard entries (%d):\n", len(keys)))
		for _, k := range keys {
			entry := t.board.GetEntry(k)
			if entry != nil {
				sb.WriteString(fmt.Sprintf("- %s (by %s): %s\n", k, entry.Author, entry.Value))
			}
		}
		return tools.NewToolResult(sb.String())

	case "delete":
		if key == "" {
			return tools.ErrorResult("key is required for delete action")
		}
		if t.board.Delete(key) {
			return tools.NewToolResult(fmt.Sprintf("Deleted key %q from blackboard", key))
		}
		return tools.NewToolResult(fmt.Sprintf("Key %q not found on blackboard", key))

	default:
		return tools.ErrorResult(fmt.Sprintf("unknown action %q; use read, write, list, or delete", action))
	}
}
