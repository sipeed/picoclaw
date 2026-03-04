package tools

import (
	"context"
	"fmt"
)

// AskConductorTool allows a subagent to ask the conductor a question.

// The subagent blocks until the conductor answers via AnswerSubagentTool.

type AskConductorTool struct {
	taskID string

	conductorKey string

	subagentKey string

	outCh chan<- ContainerMessage

	inCh <-chan string

	recorder SessionRecorder
}

func NewAskConductorTool(
	taskID, conductorKey, subagentKey string,

	outCh chan<- ContainerMessage,

	inCh <-chan string,

	recorder SessionRecorder,
) *AskConductorTool {
	return &AskConductorTool{
		taskID: taskID,

		conductorKey: conductorKey,

		subagentKey: subagentKey,

		outCh: outCh,

		inCh: inCh,

		recorder: recorder,
	}
}

func (t *AskConductorTool) Name() string { return "ask_conductor" }

func (t *AskConductorTool) Description() string {
	return "Ask the conductor a clarifying question. Blocks until the conductor responds. Use when you need guidance or a decision before proceeding."
}

func (t *AskConductorTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",

		"properties": map[string]any{
			"question": map[string]any{
				"type": "string",

				"description": "The question to ask the conductor",
			},
		},

		"required": []string{"question"},
	}
}

func (t *AskConductorTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	question, ok := args["question"].(string)

	if !ok || question == "" {
		return ErrorResult("required parameter \"question\" (string) is missing")
	}

	// Fire-and-forget: record question in session DAG.

	if t.recorder != nil {
		_ = t.recorder.RecordQuestion(t.conductorKey, t.subagentKey, t.taskID, question)
	}

	// Send question to conductor (blocking with ctx).

	select {
	case t.outCh <- ContainerMessage{Type: "question", Content: question, TaskID: t.taskID}:

	case <-ctx.Done():

		return ErrorResult(fmt.Sprintf("context canceled while sending question: %v", ctx.Err()))
	}

	// Wait for conductor's answer.

	select {
	case answer := <-t.inCh:

		return &ToolResult{
			ForLLM: fmt.Sprintf("Conductor answered: %s", answer),

			ForUser: answer,
		}

	case <-ctx.Done():

		return ErrorResult(fmt.Sprintf("context canceled while waiting for answer: %v", ctx.Err()))
	}
}
