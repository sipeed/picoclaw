package tools

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/orch"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// spawnTimeout is the hard upper bound for a single spawn goroutine.
// MaxIterations × HTTP timeout provides the soft limit; this is a safety net.
const spawnTimeout = 30 * time.Minute

type SubagentTask struct {
	ID            string
	Task          string
	Label         string
	AgentID       string
	OriginChannel string
	OriginChatID  string
	Status        string
	Result        string
	Created       int64
	CompletedAt   int64          `json:"-"`
	Iterations    int            `json:"-"`
	ToolCalls     int            `json:"-"`
	ToolStats     map[string]int `json:"-"`
	cancel        context.CancelFunc
}

type SubagentManager struct {
	tasks          map[string]*SubagentTask
	mu             sync.RWMutex
	wg             sync.WaitGroup // tracks running spawn goroutines
	provider       providers.LLMProvider
	defaultModel   string
	bus            *bus.MessageBus
	workspace      string
	tools          *ToolRegistry
	webSearchOpts  WebSearchToolOptions
	maxIterations  int
	maxTokens      int
	temperature    float64
	hasMaxTokens   bool
	hasTemperature bool
	nextID         int
	reporter       orch.AgentReporter
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
		tasks:         make(map[string]*SubagentTask),
		provider:      provider,
		defaultModel:  defaultModel,
		bus:           bus,
		workspace:     workspace,
		tools:         NewToolRegistry(),
		webSearchOpts: webSearchOpts,
		maxIterations: 10,
		nextID:        1,
		reporter:      reporter,
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
		ID:            taskID,
		Task:          task,
		Label:         label,
		AgentID:       agentID,
		OriginChannel: originChannel,
		OriginChatID:  originChatID,
		Status:        "running",
		Created:       time.Now().UnixMilli(),
	}
	sm.tasks[taskID] = subagentTask

	sm.reporter.ReportSpawn(taskID, label, task)

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

	// Build system prompt based on preset type
	systemPrompt := `You are a subagent. Complete the given task independently and report the result.
You have access to tools - use them as needed to complete your task.
After completing the task, provide a clear summary of what was done.`

	// Select prompt based on preset (exploratory vs deliberate)
	p := Preset(preset)
	if IsValidPreset(p) {
		switch p {
		case PresetScout, PresetAnalyst:
			// Exploratory presets
			systemPrompt = `You are an exploratory subagent. Investigate the task and report your findings.
Use your best judgment when encountering ambiguity. Use tools as needed.
Return clear findings and observations.`
		case PresetCoder, PresetWorker, PresetCoordinator:
			// Deliberate presets
			systemPrompt = `You are a deliberate subagent. Complete the task methodically and verify your work.
Before executing significant actions, think through your approach.
After completing, provide a clear summary of what was done and how it was verified.`
		}
	}

	messages := []providers.Message{
		{
			Role:    "system",
			Content: systemPrompt,
		},
		{
			Role:    "user",
			Content: task.Task,
		},
	}

	// Check if context is already canceled before starting
	select {
	case <-ctx.Done():
		sm.mu.Lock()
		task.Status = "canceled"
		task.Result = "Task canceled before execution"
		sm.mu.Unlock()
		return
	default:
	}

	// Run tool loop with access to tools
	sm.mu.RLock()
	// Use preset registry if preset is valid, otherwise use default registry
	tools := sm.tools
	if IsValidPreset(p) {
		tools = sm.buildPresetRegistry(p, sm.workspace)
	}
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

	// Notify conductor that the subagent is starting
	sm.reporter.ReportConversation("conductor", task.ID, task.Task)

	loopResult, err := RunToolLoop(ctx, ToolLoopConfig{
		Provider:      sm.provider,
		Model:         sm.defaultModel,
		Tools:         tools,
		MaxIterations: maxIter,
		LLMOptions:    llmOptions,
		Reporter:      sm.reporter,
		AgentID:       task.ID,
	}, messages, task.OriginChannel, task.OriginChatID)

	sm.mu.Lock()
	var result *ToolResult
	defer func() {
		sm.mu.Unlock()
		// Call callback if provided and result is set
		if callback != nil && result != nil {
			callback(ctx, result)
		}
	}()

	if err != nil {
		task.Status = "failed"
		task.Result = fmt.Sprintf("Error: %v", err)
		// Check if it was canceled
		gcReason := "failed"
		if ctx.Err() != nil {
			task.Status = "canceled"
			task.Result = "Task canceled during execution"
			gcReason = "canceled"
		}
		sm.reporter.ReportGC(task.ID, gcReason)
		result = &ToolResult{
			ForLLM:  task.Result,
			ForUser: "",
			Silent:  false,
			IsError: true,
			Async:   false,
			Err:     err,
		}
	} else {
		task.Status = "completed"
		task.Result = loopResult.Content
		task.CompletedAt = time.Now().UnixMilli()
		task.Iterations = loopResult.Iterations
		task.ToolCalls = loopResult.ToolCalls
		task.ToolStats = loopResult.ToolStats
		// Notify conductor of the result
		sm.reporter.ReportConversation(task.ID, "conductor", loopResult.Content)
		sm.reporter.ReportGC(task.ID, "completed")
		result = &ToolResult{
			ForLLM: fmt.Sprintf(
				"Subagent '%s' completed (iterations: %d, tool calls: %d): %s",
				task.Label,
				loopResult.Iterations,
				loopResult.ToolCalls,
				loopResult.Content,
			),
			ForUser: loopResult.Content,
			Silent:  false,
			IsError: false,
			Async:   false,
		}
	}

	// Send announce message back to main agent
	if sm.bus != nil {
		announceContent := fmt.Sprintf("Task '%s' completed.\n\nResult:\n%s", task.Label, task.Result)
		metadata := map[string]string{
			"duration_ms": strconv.FormatInt(task.CompletedAt-task.Created, 10),
			"iterations":  strconv.Itoa(task.Iterations),
			"tool_calls":  strconv.Itoa(task.ToolCalls),
		}
		if len(task.ToolStats) > 0 {
			metadata["tool_stats"] = formatToolStats(task.ToolStats)
		}
		pubCtx, pubCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer pubCancel()
		sm.bus.PublishInbound(pubCtx, bus.InboundMessage{
			Channel:  "system",
			SenderID: fmt.Sprintf("subagent:%s", task.ID),
			// Format: "original_channel:original_chat_id" for routing back
			ChatID:   fmt.Sprintf("%s:%s", task.OriginChannel, task.OriginChatID),
			Content:  announceContent,
			Metadata: metadata,
		})
	}
}

// buildPresetRegistry constructs a ToolRegistry for the given preset with appropriate restrictions.
func (sm *SubagentManager) buildPresetRegistry(preset Preset, writeRoot string) *ToolRegistry {
	registry := NewToolRegistry()
	config := SandboxConfigForPreset(preset, writeRoot)

	readRoot := writeRoot
	if readRoot == "" {
		readRoot = sm.workspace
	}

	// Register read_file and list_dir with restrict=true
	if config.AllowedTools["read_file"] {
		registry.Register(NewReadFileTool(readRoot, true))
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
		registry.Register(NewWebFetchTool(50000))
	}

	// Register message tool (always available)
	registry.Register(NewMessageTool())

	// Register spawn tool only for coordinator preset
	if config.AllowedTools["spawn"] && preset == PresetCoordinator {
		spawnTool := NewSpawnTool(sm)
		registry.Register(spawnTool)
	}

	return registry
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
	manager       *SubagentManager
	originChannel string
	originChatID  string
}

func NewSubagentTool(manager *SubagentManager) *SubagentTool {
	return &SubagentTool{
		manager:       manager,
		originChannel: "cli",
		originChatID:  "direct",
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
				"type":        "string",
				"description": "The task for subagent to complete",
			},
			"label": map[string]any{
				"type":        "string",
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
			Role:    "system",
			Content: "You are a subagent. Complete the given task independently and provide a clear, concise result.",
		},
		{
			Role:    "user",
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
		ForLLM:  llmContent,
		ForUser: userContent,
		Silent:  false,
		IsError: false,
		Async:   false,
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
