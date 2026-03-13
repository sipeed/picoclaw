package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/orch"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/routing"
)

// spawnTimeout is the hard upper bound for a single spawn goroutine.

// MaxIterations × HTTP timeout provides the soft limit; this is a safety net.

const spawnTimeout = 30 * time.Minute

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

type SubagentTask struct {
	ID string

	Task string

	Label string

	AgentID string

	OriginChannel string

	OriginChatID string

	Status string

	Result string

	Created int64

	CompletedAt int64 `json:"-"`

	Iterations int `json:"-"`

	ToolCalls int `json:"-"`

	ToolStats map[string]int `json:"-"`

	cancel context.CancelFunc

	// Escalation channels for deliberate presets (nil for exploratory).

	inCh chan string // conductor → subagent answers

	outCh chan ContainerMessage // subagent → conductor questions/plan reviews

	// Plan mode state (deliberate presets only).

	PlanState SubagentPlanState

	PlanGoal string

	PlanSteps []string
}

type SubagentManager struct {
	tasks map[string]*SubagentTask

	mu sync.RWMutex

	wg sync.WaitGroup // tracks running spawn goroutines

	provider providers.LLMProvider

	defaultModel string

	bus *bus.MessageBus

	workspace string

	tools *ToolRegistry

	webSearchOpts WebSearchToolOptions

	maxIterations int

	maxTokens int

	temperature float64

	hasMaxTokens bool

	hasTemperature bool

	nextID int

	reporter orch.AgentReporter

	recorder SessionRecorder

	conductorSessionKey string
}

func NewSubagentManager(
	provider providers.LLMProvider,
	defaultModel, workspace string,

	bus *bus.MessageBus,

	reporter orch.AgentReporter,

	webSearchOpts WebSearchToolOptions,
) *SubagentManager {
	if reporter == nil {
		reporter = orch.Noop
	}

	return &SubagentManager{
		tasks: make(map[string]*SubagentTask),

		provider: provider,

		defaultModel: defaultModel,

		bus: bus,

		workspace: workspace,

		tools: NewToolRegistry(),

		webSearchOpts: webSearchOpts,

		maxIterations: 10,

		nextID: 1,

		reporter: reporter,
	}
}

// SetLLMOptions sets max tokens and temperature for subagent LLM calls.
func (sm *SubagentManager) SetLLMOptions(maxTokens int, temperature float64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.maxTokens = maxTokens
	sm.hasMaxTokens = true
	sm.temperature = temperature
	sm.hasTemperature = true
}

// SetTools sets the tool registry for subagent execution.
// If not set, subagent will have access to the provided tools.
func (sm *SubagentManager) SetTools(tools *ToolRegistry) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.tools = tools
}

// SetSessionRecorder configures session recording for DAG persistence.

func (sm *SubagentManager) SetSessionRecorder(r SessionRecorder, conductorSessionKey string) {
	sm.mu.Lock()

	defer sm.mu.Unlock()

	sm.recorder = r

	sm.conductorSessionKey = conductorSessionKey
}

// RegisterTool registers a tool for subagent execution.
func (sm *SubagentManager) RegisterTool(tool Tool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.tools.Register(tool)
}

func (sm *SubagentManager) Spawn(
	ctx context.Context,

	task, label, agentID, originChannel, originChatID, preset string,

	callback AsyncCallback,
) (string, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	taskID := fmt.Sprintf("subagent-%d", sm.nextID)
	sm.nextID++

	subagentTask := &SubagentTask{
		ID: taskID,

		Task: task,

		Label: label,

		AgentID: agentID,

		OriginChannel: originChannel,

		OriginChatID: originChatID,

		Status: "running",

		Created: time.Now().UnixMilli(),
	}

	// Create escalation channels for deliberate presets.

	p := Preset(preset)

	if IsValidPreset(p) && isDeliberatePreset(p) {
		subagentTask.inCh = make(chan string, 1)

		subagentTask.outCh = make(chan ContainerMessage, 4)
	}

	sm.tasks[taskID] = subagentTask

	sm.reporter.ReportSpawn(taskID, label, task)

	// Record fork in session DAG (conductor → subagent).

	if sm.recorder != nil && sm.conductorSessionKey != "" {
		subSessionKey := routing.BuildSubagentSessionKey(taskID)

		_ = sm.recorder.RecordFork(sm.conductorSessionKey, subSessionKey, taskID, label)
	}

	// Start task in background with a detached context that has a hard timeout.

	// The spawned goroutine must outlive the parent (e.g. heartbeat session)

	// which may finish before the subagent completes.

	// The cancel func is stored on the task so CancelTask() can stop it.

	spawnCtx, spawnCancel := context.WithTimeout(context.Background(), spawnTimeout)

	subagentTask.cancel = spawnCancel

	sm.wg.Add(1)

	go func() {
		defer sm.wg.Done()

		sm.runTask(spawnCtx, subagentTask, preset, callback)
	}()

	if label != "" {
		return fmt.Sprintf("Spawned subagent '%s' for task: %s", label, task), nil
	}
	return fmt.Sprintf("Spawned subagent for task: %s", task), nil
}

func (sm *SubagentManager) runTask(ctx context.Context, task *SubagentTask, preset string, callback AsyncCallback) {
	task.Status = "running"

	p := Preset(preset)

	if IsValidPreset(p) && isDeliberatePreset(p) {
		sm.runDeliberateTask(ctx, task, p, callback)
	} else {
		sm.runExploratoryTask(ctx, task, p, callback)
	}
}

// clarifyingSystemPrompt returns the system prompt for the clarifying phase.

func clarifyingSystemPrompt() string {
	return `You are a deliberate subagent in the CLARIFYING phase.

Your job is to understand the task fully before acting. You MUST:

1. Read relevant files and gather context using your tools.

2. If anything is unclear, use ask_conductor to ask the conductor.

3. When you have a clear plan, use submit_plan with a goal and steps.



Do NOT execute any changes yet. Only investigate and plan.

Available escalation tools: ask_conductor, submit_plan.`
}

// executingSystemPrompt returns the system prompt for the executing phase.

func executingSystemPrompt() string {
	return `You are a deliberate subagent in the EXECUTING phase. Your plan was approved.

Execute the plan steps methodically. Use all available tools to complete the work.

After completing, provide a clear summary of what was done and how it was verified.



If you encounter a blocker, use ask_conductor to escalate.`
}

// exploratorySystemPrompt returns the system prompt for exploratory presets.

func exploratorySystemPrompt(p Preset) string {
	switch p {
	case PresetScout, PresetAnalyst:

		return `You are an exploratory subagent. Investigate the task and report your findings.

Use your best judgment when encountering ambiguity. Use tools as needed.

Return clear findings and observations.`

	default:

		return `You are a subagent. Complete the given task independently and report the result.

You have access to tools - use them as needed to complete your task.

After completing the task, provide a clear summary of what was done.`
	}
}

// getLLMOptions returns the LLM options snapshot under read lock.

func (sm *SubagentManager) getLLMOptions() map[string]any {
	sm.mu.RLock()

	defer sm.mu.RUnlock()

	var opts map[string]any

	if sm.hasMaxTokens || sm.hasTemperature {
		opts = map[string]any{}

		if sm.hasMaxTokens {
			opts["max_tokens"] = sm.maxTokens
		}

		if sm.hasTemperature {
			opts["temperature"] = sm.temperature
		}
	}

	return opts
}

// finishTask records completion, sends bus announcement, and invokes callback.

// Must NOT hold sm.mu on entry.

func (sm *SubagentManager) finishTask(
	ctx context.Context,
	task *SubagentTask,
	messages []providers.Message,
	loopResult *ToolLoopResult,
	err error,
	callback AsyncCallback,
) {
	sm.mu.Lock()
	var result *ToolResult
	defer func() {
		sm.mu.Unlock()

		if callback != nil && result != nil {
			callback(ctx, result)
		}
	}()

	if err != nil {
		task.Status = "failed"
		task.Result = fmt.Sprintf("Error: %v", err)

		gcReason := "failed"

		if ctx.Err() != nil {
			task.Status = "canceled"
			task.Result = "Task canceled during execution"

			gcReason = "canceled"
		}

		sm.reporter.ReportGC(task.ID, gcReason)

		if sm.recorder != nil {
			subKey := routing.BuildSubagentSessionKey(task.ID)

			_ = sm.recorder.RecordCompletion(subKey, task.Status, task.Result)
		}

		result = &ToolResult{
			ForLLM: task.Result,

			ForUser: "",

			IsError: true,

			Err: err,
		}
	} else {
		task.Status = "completed"
		task.Result = loopResult.Content

		task.CompletedAt = time.Now().UnixMilli()

		task.Iterations = loopResult.Iterations

		task.ToolCalls = loopResult.ToolCalls

		task.ToolStats = loopResult.ToolStats

		task.PlanState = PlanCompleted

		sm.reporter.ReportConversation(task.ID, "conductor", loopResult.Content)

		sm.reporter.ReportGC(task.ID, "completed")

		if sm.recorder != nil {
			subKey := routing.BuildSubagentSessionKey(task.ID)

			_ = sm.recorder.RecordSubagentTurn(subKey, messages)

			_ = sm.recorder.RecordCompletion(subKey, "completed", loopResult.Content)
		}

		result = &ToolResult{
			ForLLM: fmt.Sprintf(

				"Subagent '%s' completed (iterations: %d, tool calls: %d): %s",

				task.Label,
				loopResult.Iterations,

				loopResult.ToolCalls,

				loopResult.Content,
			),
			ForUser: loopResult.Content,
		}
	}

	// Send announce message back to main agent

	if sm.bus != nil {
		announceContent := fmt.Sprintf("Task '%s' completed.\n\nResult:\n%s", task.Label, task.Result)

		metadata := map[string]string{
			"duration_ms": strconv.FormatInt(task.CompletedAt-task.Created, 10),

			"iterations": strconv.Itoa(task.Iterations),

			"tool_calls": strconv.Itoa(task.ToolCalls),
		}

		if len(task.ToolStats) > 0 {
			metadata["tool_stats"] = formatToolStats(task.ToolStats)
		}

		pubCtx, pubCancel := context.WithTimeout(context.Background(), 5*time.Second)

		defer pubCancel()

		sm.bus.PublishInbound(pubCtx, bus.InboundMessage{
			Channel: "system",

			SenderID: fmt.Sprintf("subagent:%s", task.ID),

			ChatID: fmt.Sprintf("%s:%s", task.OriginChannel, task.OriginChatID),

			Content: announceContent,

			Metadata: metadata,
		})
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

// runExploratoryTask runs a single-phase tool loop for exploratory presets.

func (sm *SubagentManager) runExploratoryTask(
	ctx context.Context,
	task *SubagentTask,
	preset Preset,
	callback AsyncCallback,
) {
	systemPrompt := buildSubagentSystemPrompt(exploratorySystemPrompt(preset), sm.workspace)

	messages := []providers.Message{
		{Role: "system", Content: systemPrompt},

		{Role: "user", Content: task.Task},
	}

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

	reg := sm.tools

	if IsValidPreset(preset) {
		reg = sm.buildPresetRegistry(preset, sm.workspace, task)
	}

	maxIter := sm.maxIterations

	sm.mu.RUnlock()

	sm.reporter.ReportConversation("conductor", task.ID, task.Task)

	loopResult, err := RunToolLoop(ctx, ToolLoopConfig{
		Provider: sm.provider,

		Model: sm.defaultModel,

		Tools: reg,

		MaxIterations: maxIter,

		LLMOptions: sm.getLLMOptions(),

		Reporter: sm.reporter,

		AgentID: task.ID,
	}, messages, task.OriginChannel, task.OriginChatID)

	sm.finishTask(ctx, task, messages, loopResult, err, callback)
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

// buildPresetRegistry constructs a ToolRegistry for the given preset with appropriate restrictions.

// If task is non-nil and has escalation channels, ask_conductor and submit_plan are registered.

func (sm *SubagentManager) buildPresetRegistry(preset Preset, writeRoot string, task ...*SubagentTask) *ToolRegistry {
	registry := NewToolRegistry()

	config := SandboxConfigForPreset(preset, writeRoot)

	readRoot := writeRoot

	if readRoot == "" {
		readRoot = sm.workspace
	}

	// Register read_file and list_dir with restrict=true

	if config.AllowedTools["read_file"] {
		registry.Register(NewReadFileTool(readRoot, true, 0))
	}

	if config.AllowedTools["list_dir"] {
		registry.Register(NewListDirTool(readRoot, true))
	}

	// Register write tools only if allowed and writeRoot is set

	if config.AllowedTools["write_file"] && writeRoot != "" {
		registry.Register(NewWriteFileTool(writeRoot, true))

		registry.Register(NewEditFileTool(writeRoot, true))

		registry.Register(NewAppendFileTool(writeRoot, true))
	}

	// Register exec and bg_monitor if allowed.

	// Each subagent gets its own ExecTool to avoid mutating the shared instance's

	// allowRules (which would leak sandbox restrictions to the conductor).

	if config.AllowedTools["exec"] {
		execWorkDir := writeRoot

		if execWorkDir == "" {
			execWorkDir = sm.workspace
		}

		execTool, err := NewExecTool(execWorkDir, true)
		if err != nil {
			// exec disabled for this subagent; skip registration

			return registry
		}

		if config.ExecPolicy != nil {
			execTool.SetAllowRules(config.ExecPolicy.AllowRules)

			execTool.SetLocalNetOnly(config.ExecPolicy.LocalNetOnly)
		}

		registry.Register(execTool)

		if config.AllowedTools["bg_monitor"] {
			registry.Register(NewBgMonitorTool(execTool))
		}
	}

	// Register git tools (worktree-safe push and PR creation)

	if config.AllowedTools["git_push"] {
		registry.Register(NewGitPushTool())
	}

	if config.AllowedTools["create_pr"] {
		registry.Register(NewCreatePRTool())
	}

	// Register web tools

	if config.AllowedTools["web_search"] {
		webSearchTool, _ := NewWebSearchTool(sm.webSearchOpts)

		if webSearchTool != nil {
			registry.Register(webSearchTool)
		}
	}

	if config.AllowedTools["web_fetch"] {
		if fetchTool, err := NewWebFetchTool(50000); err == nil {
			registry.Register(fetchTool)
		}
	}

	// Register message tool (always available)

	registry.Register(NewMessageTool())

	// Register spawn tool only for coordinator preset

	if config.AllowedTools["spawn"] && preset == PresetCoordinator {
		spawnTool := NewSpawnTool(sm)

		registry.Register(spawnTool)
	}

	// Register escalation tools for deliberate presets with channels.

	if len(task) > 0 && task[0] != nil && task[0].outCh != nil {
		t := task[0]

		subKey := "subagent:" + t.ID

		registry.Register(NewAskConductorTool(

			t.ID, sm.conductorSessionKey, subKey,

			t.outCh, t.inCh, sm.recorder,
		))

		submitPlan := NewSubmitPlanTool(

			t.ID, sm.conductorSessionKey, subKey,

			t.outCh, t.inCh, sm.recorder,
		)

		submitPlan.SetPlanCallback(func(goal string, steps []string) {
			t.PlanGoal = goal

			t.PlanSteps = steps
		})

		registry.Register(submitPlan)
	}

	return registry
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

// WaitAll blocks until all spawned subagent goroutines have finished

// or the timeout expires. Returns true if all goroutines finished,

// false on timeout.

func (sm *SubagentManager) WaitAll(timeout time.Duration) bool {
	done := make(chan struct{})

	go func() {
		sm.wg.Wait()

		close(done)
	}()

	select {
	case <-done:

		return true

	case <-time.After(timeout):

		return false
	}
}

// CancelTask cancels the context for a running subagent task.

func (sm *SubagentManager) CancelTask(taskID string) {
	sm.mu.RLock()

	task, ok := sm.tasks[taskID]

	sm.mu.RUnlock()

	if ok && task.cancel != nil {
		task.cancel()
	}
}

func (sm *SubagentManager) GetTask(taskID string) (*SubagentTask, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	task, ok := sm.tasks[taskID]
	return task, ok
}

func (sm *SubagentManager) ListTasks() []*SubagentTask {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	tasks := make([]*SubagentTask, 0, len(sm.tasks))
	for _, task := range sm.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

// SubagentTool executes a subagent task synchronously and returns the result.
// Unlike SpawnTool which runs tasks asynchronously, SubagentTool waits for completion
// and returns the result directly in the ToolResult.
type SubagentTool struct {
	manager *SubagentManager

	originChannel string

	originChatID string
}

func NewSubagentTool(manager *SubagentManager) *SubagentTool {
	return &SubagentTool{
		manager: manager,

		originChannel: "cli",

		originChatID: "direct",
	}
}

func (t *SubagentTool) Name() string {
	return "subagent"
}

func (t *SubagentTool) Description() string {
	return "Run a task in a subagent and BLOCK until it completes, returning the result directly. Use when you need the answer before deciding your next step. For background/parallel tasks, use spawn instead."
}

func (t *SubagentTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"task": map[string]any{
				"type": "string",

				"description": "The task for subagent to complete",
			},
			"label": map[string]any{
				"type": "string",

				"description": "Optional short label for the task (for display)",
			},
		},
		"required": []string{"task"},
	}
}

func (t *SubagentTool) SetContext(channel, chatID string) {
	t.originChannel = channel

	t.originChatID = chatID
}

func (t *SubagentTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	task, ok := args["task"].(string)
	if !ok {
		return ErrorResult(

			`Required parameter "task" (string) is missing. ` +

				`Example: {"task": "describe what you need done"}`,
		).WithError(fmt.Errorf("task parameter is required"))
	}

	label, _ := args["label"].(string)

	if t.manager == nil {
		return ErrorResult("subagent tool is not available in this session (orchestration may be disabled)").
			WithError(fmt.Errorf("manager is nil"))
	}

	// Build messages for subagent
	messages := []providers.Message{
		{
			Role: "system",

			Content: "You are a subagent. Complete the given task independently and provide a clear, concise result.",
		},
		{
			Role: "user",

			Content: task,
		},
	}

	// Use RunToolLoop to execute with tools (same as async SpawnTool)
	sm := t.manager
	sm.mu.RLock()
	tools := sm.tools
	maxIter := sm.maxIterations
	maxTokens := sm.maxTokens
	temperature := sm.temperature
	hasMaxTokens := sm.hasMaxTokens
	hasTemperature := sm.hasTemperature
	sm.mu.RUnlock()

	var llmOptions map[string]any
	if hasMaxTokens || hasTemperature {
		llmOptions = map[string]any{}
		if hasMaxTokens {
			llmOptions["max_tokens"] = maxTokens
		}
		if hasTemperature {
			llmOptions["temperature"] = temperature
		}
	}

	loopResult, err := RunToolLoop(ctx, ToolLoopConfig{
		Provider: sm.provider,

		Model: sm.defaultModel,

		Tools: tools,

		MaxIterations: maxIter,

		LLMOptions: llmOptions,
	}, messages, t.originChannel, t.originChatID)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Subagent execution failed: %v", err)).WithError(err)
	}

	// ForUser: Brief summary for user (truncated if too long)
	userContent := loopResult.Content
	maxUserLen := 500
	if len(userContent) > maxUserLen {
		userContent = userContent[:maxUserLen] + "..."
	}

	// ForLLM: Full execution details
	labelStr := label
	if labelStr == "" {
		labelStr = "(unnamed)"
	}

	llmContent := fmt.Sprintf("Subagent task completed:\nLabel: %s\nIterations: %d\nTool calls: %d\nResult: %s",

		labelStr, loopResult.Iterations, loopResult.ToolCalls, loopResult.Content)

	return &ToolResult{
		ForLLM: llmContent,

		ForUser: userContent,

		Silent: false,

		IsError: false,

		Async: false,
	}
}

// formatToolStats formats a tool stats map as a compact string: "exec:3,read_file:5".

// Keys are sorted alphabetically for deterministic output.

func formatToolStats(stats map[string]int) string {
	keys := make([]string, 0, len(stats))

	for k := range stats {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	parts := make([]string, 0, len(keys))

	for _, k := range keys {
		parts = append(parts, k+":"+strconv.Itoa(stats[k]))
	}

	return strings.Join(parts, ",")
}

// extractPlanContext reads MEMORY.md from the workspace and extracts relevant

// sections (Task, Context, Commands) to provide as subagent environment.

func extractPlanContext(workspace string) string {
	memPath := filepath.Join(workspace, "memory", "MEMORY.md")

	data, err := os.ReadFile(memPath)
	if err != nil {
		return ""
	}

	content := string(data)

	var sections []string

	// Extract key sections by header.

	for _, header := range []string{"## Context", "## Commands", "## Orchestration"} {
		if section := extractSection(content, header); section != "" {
			sections = append(sections, section)
		}
	}

	// Also extract the task line from the header block.

	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "> Task:") {
			sections = append([]string{strings.TrimSpace(line)}, sections...)

			break
		}
	}

	if len(sections) == 0 {
		return ""
	}

	return strings.Join(sections, "\n\n")
}

// extractSection extracts a markdown section by header (including its content

// until the next section of the same or higher level).

func extractSection(content, header string) string {
	idx := strings.Index(content, header)

	if idx < 0 {
		return ""
	}

	// Determine header level.

	level := 0

	for _, c := range header {
		if c == '#' {
			level++
		} else {
			break
		}
	}

	start := idx

	rest := content[idx+len(header):]

	// Find next section at same or higher level.

	nextHeader := "\n" + strings.Repeat("#", level) + " "

	end := strings.Index(rest, nextHeader)

	if end < 0 {
		return strings.TrimSpace(content[start:])
	}

	return strings.TrimSpace(content[start : start+len(header)+end])
}

// buildSubagentSystemPrompt builds an enriched system prompt for a subagent

// by combining the base prompt with environment context from MEMORY.md.

func buildSubagentSystemPrompt(basePrompt, workspace string) string {
	envContext := extractPlanContext(workspace)

	if envContext == "" {
		return basePrompt
	}

	return basePrompt + "\n\n## Environment Context\n\n" + envContext
}
