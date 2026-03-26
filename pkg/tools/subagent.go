package tools

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/orch"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/routing"
)

// spawnTimeout is the hard upper bound for a single spawn goroutine.
// MaxIterations × HTTP timeout provides the soft limit; this is a safety net.
const spawnTimeout = 30 * time.Minute

// SubTurnSpawner is an interface for spawning sub-turns.
// This avoids circular dependency between tools and agent packages.
type SubTurnSpawner interface {
	SpawnSubTurn(ctx context.Context, cfg SubTurnConfig) (*ToolResult, error)
}

// SubTurnConfig holds configuration for spawning a sub-turn.
type SubTurnConfig struct {
	Model              string
	Tools              []Tool
	SystemPrompt       string
	MaxTokens          int
	Temperature        float64
	Async              bool          // true for async (spawn), false for sync (subagent)
	Critical           bool          // continue running after parent finishes gracefully
	Timeout            time.Duration // 0 = use default (5 minutes)
	MaxContextRunes    int           // 0 = auto, -1 = no limit, >0 = explicit limit
	ActualSystemPrompt string
	InitialMessages    []providers.Message
	InitialTokenBudget *atomic.Int64 // Shared token budget for team members; nil if no budget
}

type SubagentTask struct {
	ID    string
	Task  string
	Label string

	AgentID       string
	OriginChannel string
	OriginChatID  string
	Status        string
	Result        string
	Created       int64

	CompletedAt int64          `json:"-"`
	Iterations  int            `json:"-"`
	ToolCalls   int            `json:"-"`
	ToolStats   map[string]int `json:"-"`

	cancel context.CancelFunc

	// Escalation channels for deliberate presets (nil for exploratory).
	inCh  chan string           // conductor → subagent answers
	outCh chan ContainerMessage // subagent → conductor questions/plan reviews

	// Plan mode state (deliberate presets only).
	PlanState SubagentPlanState
	PlanGoal  string
	PlanSteps []string
}

type SpawnSubTurnFunc func(
	ctx context.Context,
	task, label, agentID string,
	tools *ToolRegistry,
	maxTokens int,
	temperature float64,
	hasMaxTokens, hasTemperature bool,
) (*ToolResult, error)

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

	nextID  int
	spawner SpawnSubTurnFunc

	reporter            orch.AgentReporter
	recorder            SessionRecorder
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

func (sm *SubagentManager) SetSpawner(spawner SpawnSubTurnFunc) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.spawner = spawner
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
	var result *ToolResult
	sm.mu.Lock()
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

// GetTaskCopy returns a copy of the task with the given ID, taken under the
// read lock, so the caller receives a consistent snapshot with no data race.
func (sm *SubagentManager) GetTaskCopy(taskID string) (SubagentTask, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	task, ok := sm.tasks[taskID]
	if !ok {
		return SubagentTask{}, false
	}
	return *task, true
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

// ListTaskCopies returns value copies of all tasks, taken under the read lock,
// so callers receive consistent snapshots with no data race.
func (sm *SubagentManager) ListTaskCopies() []SubagentTask {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	copies := make([]SubagentTask, 0, len(sm.tasks))
	for _, task := range sm.tasks {
		copies = append(copies, *task)
	}
	return copies
}

// SubagentTool executes a subagent task synchronously and returns the result.
// It directly calls SubTurnSpawner with Async=false for synchronous execution.
type SubagentTool struct {
	manager *SubagentManager

	spawner      SubTurnSpawner
	defaultModel string
	maxTokens    int
	temperature  float64

	originChannel string
	originChatID  string
}

func NewSubagentTool(manager *SubagentManager) *SubagentTool {
	if manager == nil {
		return &SubagentTool{}
	}
	return &SubagentTool{
		manager:       manager,
		defaultModel:  manager.defaultModel,
		maxTokens:     manager.maxTokens,
		temperature:   manager.temperature,
		originChannel: "cli",
		originChatID:  "direct",
	}
}

// SetSpawner sets the SubTurnSpawner for direct sub-turn execution.
func (t *SubagentTool) SetSpawner(spawner SubTurnSpawner) {
	t.spawner = spawner
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

	// Use spawner if available (direct SpawnSubTurn call)
	if t.spawner != nil {
		systemPrompt := fmt.Sprintf(
			"You are a subagent. Complete the given task independently and provide a clear, concise result.\n\nTask: %s",
			task,
		)
		if label != "" {
			systemPrompt = fmt.Sprintf(
				"You are a subagent labeled %q. Complete the given task independently and provide a clear, concise result.\n\nTask: %s",
				label, task,
			)
		}
		result, err := t.spawner.SpawnSubTurn(ctx, SubTurnConfig{
			Model:        t.defaultModel,
			Tools:        nil,
			SystemPrompt: systemPrompt,
			MaxTokens:    t.maxTokens,
			Temperature:  t.temperature,
			Async:        false,
		})
		if err != nil {
			return ErrorResult(fmt.Sprintf("Subagent execution failed: %v", err)).WithError(err)
		}

		// Format result for display
		userContent := result.ForLLM
		if result.ForUser != "" {
			userContent = result.ForUser
		}
		maxUserLen := 500
		if len(userContent) > maxUserLen {
			userContent = userContent[:maxUserLen] + "..."
		}

		labelStr := label
		if labelStr == "" {
			labelStr = "(unnamed)"
		}
		llmContent := fmt.Sprintf("Subagent task completed:\nLabel: %s\nResult: %s",
			labelStr, result.ForLLM)

		return &ToolResult{
			ForLLM:  llmContent,
			ForUser: userContent,
			Silent:  false,
			IsError: result.IsError,
			Async:   false,
		}
	}

	if t.manager == nil {
		return ErrorResult("subagent tool is not available in this session (orchestration may be disabled)").
			WithError(fmt.Errorf("manager is nil"))
	}

	// Fallback: use RunToolLoop with the manager
	sm := t.manager
	sm.mu.RLock()
	tools := sm.tools
	maxIter := sm.maxIterations
	sm.mu.RUnlock()

	messages := []providers.Message{
		{Role: "system", Content: "You are a subagent. Complete the given task independently and provide a clear, concise result."},
		{Role: "user", Content: task},
	}

	llmOptions := sm.getLLMOptions()

	loopResult, err := RunToolLoop(ctx, ToolLoopConfig{
		Provider:      sm.provider,
		Model:         sm.defaultModel,
		Tools:         tools,
		MaxIterations: maxIter,
		LLMOptions:    llmOptions,
	}, messages, t.originChannel, t.originChatID)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Subagent execution failed: %v", err)).WithError(err)
	}

	// ForUser: Brief summary for user (truncated if too long)
	userContent2 := loopResult.Content
	maxUserLen2 := 500
	if len(userContent2) > maxUserLen2 {
		userContent2 = userContent2[:maxUserLen2] + "..."
	}

	// ForLLM: Full execution details
	labelStr2 := label
	if labelStr2 == "" {
		labelStr2 = "(unnamed)"
	}

	llmContent2 := fmt.Sprintf("Subagent task completed:\nLabel: %s\nIterations: %d\nTool calls: %d\nResult: %s",
		labelStr2, loopResult.Iterations, loopResult.ToolCalls, loopResult.Content)

	return &ToolResult{
		ForLLM:  llmContent2,
		ForUser: userContent2,
		Silent:  false,
		IsError: false,
		Async:   false,
	}
}
