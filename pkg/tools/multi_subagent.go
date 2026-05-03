package tools

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// MultiSubagentTool executes several subagent tasks synchronously and collects
// their results. Each call may optionally include an allowed target agent ID for
// routing metadata/reporting, while execution otherwise follows the same direct
// SubTurnSpawner path as the existing synchronous subagent tool.
type MultiSubagentTool struct {
	spawner        SubTurnSpawner
	defaultModel   string
	maxTokens      int
	temperature    float64
	allowlistCheck func(targetAgentID string) bool
}

func NewMultiSubagentTool(manager *SubagentManager) *MultiSubagentTool {
	if manager == nil {
		return &MultiSubagentTool{}
	}
	return &MultiSubagentTool{
		defaultModel: manager.defaultModel,
		maxTokens:    manager.maxTokens,
		temperature:  manager.temperature,
	}
}

func (t *MultiSubagentTool) SetSpawner(spawner SubTurnSpawner) {
	t.spawner = spawner
}

func (t *MultiSubagentTool) SetAllowlistChecker(check func(targetAgentID string) bool) {
	t.allowlistCheck = check
}

func (t *MultiSubagentTool) Name() string {
	return "multi_subagent"
}

func (t *MultiSubagentTool) Description() string {
	return "Execute several subagent tasks synchronously and return grouped results. Use this when parallel delegation materially helps the current turn. Optional agent_id values are validated against the allowlist and reported per call."
}

func (t *MultiSubagentTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"calls": map[string]any{
				"type":        "array",
				"description": "The subagent calls to execute in parallel",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"task": map[string]any{
							"type":        "string",
							"description": "The task for the subagent to complete",
						},
						"label": map[string]any{
							"type":        "string",
							"description": "Optional short label for the task",
						},
						"agent_id": map[string]any{
							"type":        "string",
							"description": "Optional target agent ID metadata for this call, such as mini or code",
						},
					},
					"required": []string{"task"},
				},
			},
		},
		"required": []string{"calls"},
	}
}

type multiSubagentCall struct {
	Task    string
	Label   string
	AgentID string
}

type multiSubagentCallResult struct {
	Index   int
	Label   string
	AgentID string
	Result  *ToolResult
	Err     error
}

func (t *MultiSubagentTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	if t.spawner == nil {
		return ErrorResult("Subagent manager not configured").WithError(fmt.Errorf("spawner not set"))
	}

	rawCalls, ok := args["calls"].([]any)
	if !ok || len(rawCalls) == 0 {
		return ErrorResult("calls is required and must be a non-empty array")
	}

	calls := make([]multiSubagentCall, 0, len(rawCalls))
	for i, raw := range rawCalls {
		callMap, ok := raw.(map[string]any)
		if !ok {
			return ErrorResult(fmt.Sprintf("calls[%d] must be an object", i))
		}
		task, ok := callMap["task"].(string)
		if !ok || strings.TrimSpace(task) == "" {
			return ErrorResult(fmt.Sprintf("calls[%d].task is required and must be a non-empty string", i))
		}
		label, _ := callMap["label"].(string)
		agentID, _ := callMap["agent_id"].(string)
		if agentID != "" && t.allowlistCheck != nil && !t.allowlistCheck(agentID) {
			return ErrorResult(fmt.Sprintf("calls[%d] is not allowed to target agent '%s'", i, agentID))
		}
		calls = append(calls, multiSubagentCall{Task: task, Label: label, AgentID: agentID})
	}

	results := make([]multiSubagentCallResult, len(calls))
	var wg sync.WaitGroup
	for i, call := range calls {
		wg.Add(1)
		go func(i int, call multiSubagentCall) {
			defer wg.Done()

			systemPrompt := fmt.Sprintf(
				`You are a subagent. Complete the given task independently and provide a clear, concise result.

Task: %s`,
				call.Task,
			)
			if call.Label != "" {
				systemPrompt = fmt.Sprintf(
					`You are a subagent labeled "%s". Complete the given task independently and provide a clear, concise result.

Task: %s`,
					call.Label,
					call.Task,
				)
			}

			res, err := t.spawner.SpawnSubTurn(ctx, SubTurnConfig{
				Model:              t.defaultModel,
				Tools:              nil,
				SystemPrompt:       systemPrompt,
				ActualSystemPrompt: buildMultiSubagentActualSystemPrompt(call.Label),
				MaxTokens:          t.maxTokens,
				Temperature:        t.temperature,
				Async:              false,
			})
			results[i] = multiSubagentCallResult{
				Index:   i,
				Label:   call.Label,
				AgentID: call.AgentID,
				Result:  res,
				Err:     err,
			}
		}(i, call)
	}
	wg.Wait()

	var llmParts []string
	var userParts []string
	hadError := false
	for _, item := range results {
		label := item.Label
		if label == "" {
			label = fmt.Sprintf("call-%d", item.Index+1)
		}
		agentSegment := "default"
		if item.AgentID != "" {
			agentSegment = item.AgentID
		}

		if item.Err != nil {
			hadError = true
			llmParts = append(llmParts, fmt.Sprintf("[%s | agent=%s] ERROR: %v", label, agentSegment, item.Err))
			userParts = append(userParts, fmt.Sprintf("%s: error: %v", label, item.Err))
			continue
		}
		if item.Result == nil {
			hadError = true
			llmParts = append(llmParts, fmt.Sprintf("[%s | agent=%s] ERROR: no result returned", label, agentSegment))
			userParts = append(userParts, fmt.Sprintf("%s: error: no result returned", label))
			continue
		}
		if item.Result.IsError {
			hadError = true
		}
		llmParts = append(llmParts, fmt.Sprintf("[%s | agent=%s]\n%s", label, agentSegment, item.Result.ForLLM))
		userText := item.Result.ForUser
		if strings.TrimSpace(userText) == "" {
			userText = item.Result.ForLLM
		}
		userParts = append(userParts, fmt.Sprintf("%s: %s", label, userText))
	}

	return &ToolResult{
		ForLLM:  "Parallel subagent results:\n" + strings.Join(llmParts, "\n\n"),
		ForUser: strings.Join(userParts, "\n"),
		Silent:  false,
		IsError: hadError,
		Async:   false,
	}
}

func buildMultiSubagentActualSystemPrompt(label string) string {
	if strings.TrimSpace(label) == "" {
		return "You are a subagent. Complete the given task independently and provide a clear, concise result."
	}
	return fmt.Sprintf("You are a subagent labeled %q. Complete the given task independently and provide a clear, concise result.", label)
}
