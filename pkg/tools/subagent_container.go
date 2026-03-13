package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/routing"
)

// ContainerMessage is sent from a subagent to the conductor via outCh.

type ContainerMessage struct {
	Type string // "question" or "plan_review"

	Content string

	TaskID string
}

// isDeliberatePreset returns true for presets that use the deliberate

// (clarifying → review → executing) workflow with escalation channels.

func isDeliberatePreset(p Preset) bool {
	switch p {
	case PresetCoder, PresetWorker, PresetCoordinator:

		return true
	}

	return false
}

// SubagentPlanState represents the deliberate workflow phase.

type SubagentPlanState int

const (
	PlanNone SubagentPlanState = iota // Not a deliberate preset

	PlanClarifying // Gathering info, asking questions

	PlanReview // Plan submitted, awaiting approval

	PlanExecuting // Plan approved, executing

	PlanCompleted // Done

)

// String returns a human-readable label for the plan state.

func (s SubagentPlanState) String() string {
	switch s {
	case PlanClarifying:

		return "clarifying"

	case PlanReview:

		return "review"

	case PlanExecuting:

		return "executing"

	case PlanCompleted:

		return "completed"

	default:

		return "none"
	}
}

// setPlanState updates task's plan state in memory and records status in session DAG.

func (sm *SubagentManager) setPlanState(task *SubagentTask, state SubagentPlanState) {
	task.PlanState = state

	if sm.recorder != nil {
		subKey := routing.BuildSubagentSessionKey(task.ID)

		_ = sm.recorder.RecordCompletion(subKey, state.String(), "")
	}
}

// runDeliberateTask runs the clarifying → review → executing workflow.

func (sm *SubagentManager) runDeliberateTask(
	ctx context.Context,
	task *SubagentTask,
	preset Preset,
	callback AsyncCallback,
) {
	select {
	case <-ctx.Done():

		sm.mu.Lock()

		task.Status = "canceled"

		task.Result = "Task canceled before execution"

		sm.mu.Unlock()

		return

	default:
	}

	sm.mu.RLock()

	reg := sm.buildPresetRegistry(preset, sm.workspace, task)

	maxIter := sm.maxIterations

	sm.mu.RUnlock()

	sm.reporter.ReportConversation("conductor", task.ID, task.Task)

	sm.setPlanState(task, PlanClarifying)

	// Phase 1: Clarifying — subagent gathers info and submits a plan.

	clarifyMsgs := []providers.Message{
		{Role: "system", Content: buildSubagentSystemPrompt(clarifyingSystemPrompt(), sm.workspace)},

		{Role: "user", Content: task.Task},
	}

	clarifyResult, err := RunToolLoop(ctx, ToolLoopConfig{
		Provider: sm.provider,

		Model: sm.defaultModel,

		Tools: reg,

		MaxIterations: maxIter,

		LLMOptions: sm.getLLMOptions(),

		Reporter: sm.reporter,

		AgentID: task.ID,
	}, clarifyMsgs, task.OriginChannel, task.OriginChatID)
	if err != nil {
		sm.finishTask(ctx, task, clarifyMsgs, nil, err, callback)

		return
	}

	// After clarifying, the subagent should have used submit_plan.

	// If it didn't produce a plan, treat the clarifying result as direct completion.

	if task.PlanGoal == "" {
		sm.finishTask(ctx, task, clarifyMsgs, clarifyResult, nil, callback)

		return
	}

	// Phase 2: Executing — plan was approved, now execute it.

	sm.setPlanState(task, PlanExecuting)

	executeMsgs := []providers.Message{
		{Role: "system", Content: buildSubagentSystemPrompt(executingSystemPrompt(), sm.workspace)},

		{Role: "user", Content: fmt.Sprintf("Execute the approved plan:\nGoal: %s\nSteps:\n%s",

			task.PlanGoal, formatPlanSteps(task.PlanSteps))},
	}

	execResult, err := RunToolLoop(ctx, ToolLoopConfig{
		Provider: sm.provider,

		Model: sm.defaultModel,

		Tools: reg,

		MaxIterations: maxIter * 2, // Executing gets more iterations

		LLMOptions: sm.getLLMOptions(),

		Reporter: sm.reporter,

		AgentID: task.ID,
	}, executeMsgs, task.OriginChannel, task.OriginChatID)

	sm.finishTask(ctx, task, executeMsgs, execResult, err, callback)
}

// formatPlanSteps formats plan steps as a numbered list.

func formatPlanSteps(steps []string) string {
	var b strings.Builder

	for i, step := range steps {
		fmt.Fprintf(&b, "%d. %s\n", i+1, step)
	}

	return b.String()
}

// PendingQuestions drains all outCh channels and returns pending container messages.

// Non-blocking: reads all available messages without waiting.

func (sm *SubagentManager) PendingQuestions() []ContainerMessage {
	sm.mu.RLock()

	defer sm.mu.RUnlock()

	var msgs []ContainerMessage

	for _, task := range sm.tasks {
		if task.outCh == nil {
			continue
		}

		for {
			select {
			case msg := <-task.outCh:

				msgs = append(msgs, msg)

			default:

				goto nextTask
			}
		}

	nextTask:
	}

	return msgs
}

// AnswerQuestion sends an answer to a subagent's inCh (non-blocking).

func (sm *SubagentManager) AnswerQuestion(taskID, answer string) error {
	sm.mu.RLock()

	task, ok := sm.tasks[taskID]

	sm.mu.RUnlock()

	if !ok {
		return fmt.Errorf("task %q not found", taskID)
	}

	if task.inCh == nil {
		return fmt.Errorf("task %q has no escalation channel", taskID)
	}

	select {
	case task.inCh <- answer:

		return nil

	default:

		return fmt.Errorf("task %q answer channel full", taskID)
	}
}
