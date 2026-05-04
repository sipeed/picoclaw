package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

var updatePlanStatuses = map[string]struct{}{
	"pending":     {},
	"in_progress": {},
	"completed":   {},
}

type UpdatePlanTool struct{}

type updatePlanStep struct {
	Step   string `json:"step"`
	Status string `json:"status"`
}

type updatePlanResponse struct {
	Status      string           `json:"status"`
	Explanation string           `json:"explanation,omitempty"`
	Plan        []updatePlanStep `json:"plan"`
}

func NewUpdatePlanTool() *UpdatePlanTool {
	return &UpdatePlanTool{}
}

func (t *UpdatePlanTool) Name() string {
	return "update_plan"
}

func (t *UpdatePlanTool) Description() string {
	return "Update the current task plan. Use this only for non-trivial multi-step work. Keep exactly one step in_progress while work is active, use pending and completed for the rest, and avoid repeating the whole plan after each update."
}

func (t *UpdatePlanTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"explanation": map[string]any{
				"type":        "string",
				"description": "Optional short note explaining what changed in the plan.",
			},
			"plan": map[string]any{
				"type":        "array",
				"minItems":    1,
				"description": "Ordered list of plan steps. Keep exactly one step in_progress while work is active.",
				"items": map[string]any{
					"type":                 "object",
					"additionalProperties": true,
					"properties": map[string]any{
						"step": map[string]any{
							"type":        "string",
							"description": "Short plan step.",
						},
						"status": map[string]any{
							"type":        "string",
							"enum":        []string{"pending", "in_progress", "completed"},
							"description": "One of pending, in_progress, or completed.",
						},
					},
					"required": []string{"step", "status"},
				},
			},
		},
		"required": []string{"plan"},
	}
}

func (t *UpdatePlanTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	_ = ctx

	steps, err := readUpdatePlanSteps(args)
	if err != nil {
		return ErrorResult(err.Error())
	}

	explanation, err := optionalStringArg(args, "explanation")
	if err != nil {
		return ErrorResult(err.Error())
	}

	response := updatePlanResponse{
		Status:      "updated",
		Explanation: explanation,
		Plan:        steps,
	}
	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to encode plan update: %v", err))
	}
	return SilentResult(string(data))
}

func readUpdatePlanSteps(args map[string]any) ([]updatePlanStep, error) {
	rawPlan, ok := args["plan"].([]any)
	if !ok || len(rawPlan) == 0 {
		return nil, fmt.Errorf("plan required")
	}

	steps := make([]updatePlanStep, 0, len(rawPlan))
	inProgressCount := 0
	for i, rawEntry := range rawPlan {
		entry, ok := rawEntry.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("plan[%d] must be an object", i)
		}

		step, err := requiredStringArg(entry, "step", fmt.Sprintf("plan[%d].step", i))
		if err != nil {
			return nil, err
		}
		status, err := requiredStringArg(entry, "status", fmt.Sprintf("plan[%d].status", i))
		if err != nil {
			return nil, err
		}
		if _, ok := updatePlanStatuses[status]; !ok {
			return nil, fmt.Errorf("plan[%d].status must be one of pending, in_progress, completed", i)
		}
		if status == "in_progress" {
			inProgressCount++
		}

		steps = append(steps, updatePlanStep{
			Step:   step,
			Status: status,
		})
	}
	if inProgressCount > 1 {
		return nil, fmt.Errorf("plan can contain at most one in_progress step")
	}
	return steps, nil
}

func requiredStringArg(args map[string]any, key, label string) (string, error) {
	value, ok := args[key]
	if !ok || value == nil {
		return "", fmt.Errorf("%s required", label)
	}
	str, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("%s must be a string", label)
	}
	str = strings.TrimSpace(str)
	if str == "" {
		return "", fmt.Errorf("%s required", label)
	}
	return str, nil
}

func optionalStringArg(args map[string]any, key string) (string, error) {
	value, ok := args[key]
	if !ok || value == nil {
		return "", nil
	}
	str, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("%s must be a string", key)
	}
	return strings.TrimSpace(str), nil
}
