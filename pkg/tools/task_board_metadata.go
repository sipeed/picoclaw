package tools

import (
	"fmt"
	"strings"

	taskregistry "github.com/sipeed/picoclaw/pkg/tasks"
)

// TaskBoardMetadata describes how a delegate/spawn run belongs to a larger
// durable task board.
type TaskBoardMetadata struct {
	BoardID      string
	ParentTaskID string
	StepID       string
	StepTitle    string
	Owner        string
	DependsOn    []string
	BlockedBy    []string
}

// TaskBoardMetadataParameters returns JSON-schema fragments shared by tools
// that can record their runs as task-board steps.
func TaskBoardMetadataParameters() map[string]any {
	return map[string]any{
		"board_id": map[string]any{
			"type":        "string",
			"description": "Optional workflow/task-board ID shared by related delegate/spawn steps.",
		},
		"parent_task_id": map[string]any{
			"type":        "string",
			"description": "Optional parent/root task ID when this step belongs to a larger workflow.",
		},
		"step_id": map[string]any{
			"type":        "string",
			"description": "Optional stable step ID within the board, for example download-media or analyze-caption.",
		},
		"step_title": map[string]any{
			"type":        "string",
			"description": "Optional readable title for this board step.",
		},
		"owner": map[string]any{
			"type":        "string",
			"description": "Optional logical owner for this step. Defaults to the target agent/runtime.",
		},
		"depends_on": map[string]any{
			"type":        "array",
			"description": "Optional list of step/task IDs that this step depends on.",
			"items": map[string]any{
				"type": "string",
			},
		},
		"blocked_by": map[string]any{
			"type":        "array",
			"description": "Optional list of blocker step/task IDs.",
			"items": map[string]any{
				"type": "string",
			},
		},
	}
}

func addTaskBoardMetadataParameters(props map[string]any) {
	for key, value := range TaskBoardMetadataParameters() {
		props[key] = value
	}
}

func parseTaskBoardMetadata(args map[string]any) (TaskBoardMetadata, error) {
	var meta TaskBoardMetadata
	var err error
	meta.BoardID, err = optionalStringArg(args, "board_id")
	if err != nil {
		return meta, err
	}
	meta.ParentTaskID, err = optionalStringArg(args, "parent_task_id")
	if err != nil {
		return meta, err
	}
	meta.StepID, err = optionalStringArg(args, "step_id")
	if err != nil {
		return meta, err
	}
	meta.StepTitle, err = optionalStringArg(args, "step_title")
	if err != nil {
		return meta, err
	}
	meta.Owner, err = optionalStringArg(args, "owner")
	if err != nil {
		return meta, err
	}
	meta.DependsOn, err = optionalStringListArg(args, "depends_on")
	if err != nil {
		return meta, err
	}
	meta.BlockedBy, err = optionalStringListArg(args, "blocked_by")
	if err != nil {
		return meta, err
	}
	return meta, nil
}

func optionalStringListArg(args map[string]any, key string) ([]string, error) {
	raw, ok := args[key]
	if !ok || raw == nil {
		return nil, nil
	}
	switch values := raw.(type) {
	case []string:
		return cleanStringList(values), nil
	case []any:
		out := make([]string, 0, len(values))
		for i, item := range values {
			value, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("%s[%d] must be a string", key, i)
			}
			out = append(out, value)
		}
		return cleanStringList(out), nil
	default:
		return nil, fmt.Errorf("%s must be an array of strings", key)
	}
}

func cleanStringList(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func applyTaskBoardMetadata(rec *taskregistry.Record, meta TaskBoardMetadata) {
	if rec == nil {
		return
	}
	rec.BoardID = meta.BoardID
	rec.ParentTaskID = meta.ParentTaskID
	rec.StepID = meta.StepID
	rec.StepTitle = meta.StepTitle
	rec.Owner = meta.Owner
	rec.DependsOn = append([]string(nil), meta.DependsOn...)
	rec.BlockedBy = append([]string(nil), meta.BlockedBy...)
}
