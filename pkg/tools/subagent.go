package tools

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/providers"
)

type SubagentTask struct {
	ID            string
	ParentTaskID  string
	Task          string
	Label         string
	AgentID       string
	OriginChannel string
	OriginChatID  string
	Status        string
	Result        string
	Created       int64
	Depth         int
}

type SubagentExecutionConfig struct {
	Provider providers.LLMProvider
	Model    string
	Tools    *ToolRegistry
}

type SubagentExecutionResolver func(targetAgentID string) (SubagentExecutionConfig, error)

type SubagentTaskEventType string

const (
	SubagentTaskCreated   SubagentTaskEventType = "created"
	SubagentTaskRunning   SubagentTaskEventType = "running"
	SubagentTaskCompleted SubagentTaskEventType = "completed"
	SubagentTaskFailed    SubagentTaskEventType = "failed"
	SubagentTaskCancelled SubagentTaskEventType = "cancelled"
)

type SubagentTaskEvent struct {
	Type      SubagentTaskEventType
	Task      SubagentTask
	Trace     []ToolExecutionTrace
	Err       string
	Timestamp int64
}

type SubagentTaskEventHandler func(event SubagentTaskEvent)

type SubagentManager struct {
	tasks          map[string]*SubagentTask
	mu             sync.RWMutex
	provider       providers.LLMProvider
	defaultModel   string
	bus            *bus.MessageBus
	workspace      string
	tools          *ToolRegistry
	maxIterations  int
	maxTokens      int
	temperature    float64
	hasMaxTokens   bool
	hasTemperature bool
	nextID         int
	resolver       SubagentExecutionResolver
	eventHandler   SubagentTaskEventHandler
	maxConcurrent  int
	maxTasks       int
	maxDepth       int
	taskCancels    map[string]context.CancelFunc

	toolCallsParallelEnabled bool
	maxToolCallConcurrency   int
	parallelToolsMode        string
	toolPolicyOverrides      map[string]string
}

func NewSubagentManager(
	provider providers.LLMProvider,
	defaultModel, workspace string,
	bus *bus.MessageBus,
) *SubagentManager {
	return &SubagentManager{
		tasks:         make(map[string]*SubagentTask),
		provider:      provider,
		defaultModel:  defaultModel,
		bus:           bus,
		workspace:     workspace,
		tools:         NewToolRegistry(),
		maxIterations: 10,
		nextID:        1,
		taskCancels:   make(map[string]context.CancelFunc),
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

// SetExecutionResolver sets a resolver used to pick provider/model/tools for each task.
// If not set, the manager falls back to its default provider/model/tools.
func (sm *SubagentManager) SetExecutionResolver(resolver SubagentExecutionResolver) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.resolver = resolver
}

// SetEventHandler sets a callback that receives task lifecycle events.
func (sm *SubagentManager) SetEventHandler(handler SubagentTaskEventHandler) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.eventHandler = handler
}

// SetLimits sets orchestration limits for this manager.
// maxConcurrent <= 0 means unlimited.
// maxTasks <= 0 means unlimited.
// maxDepth <= 0 means unlimited.
func (sm *SubagentManager) SetLimits(maxConcurrent, maxTasks, maxDepth int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.maxConcurrent = maxConcurrent
	sm.maxTasks = maxTasks
	sm.maxDepth = maxDepth
}

// SetToolCallParallelism configures in-batch parallel tool execution for
// subagent tool loops.
func (sm *SubagentManager) SetToolCallParallelism(
	enabled bool,
	maxConcurrency int,
	mode string,
	toolPolicyOverrides map[string]string,
) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.toolCallsParallelEnabled = enabled
	sm.maxToolCallConcurrency = maxConcurrency
	sm.parallelToolsMode = mode
	sm.toolPolicyOverrides = clonePolicyOverrides(toolPolicyOverrides)
}

func clonePolicyOverrides(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func (sm *SubagentManager) Spawn(
	ctx context.Context,
	task, label, agentID, originChannel, originChatID string,
	callback AsyncCallback,
) (string, error) {
	subagentTask, err := sm.SpawnTask(ctx, task, label, agentID, originChannel, originChatID, callback)
	if err != nil {
		return "", err
	}

	if label != "" {
		return fmt.Sprintf("Spawned subagent '%s' for task: %s (id: %s)", label, task, subagentTask.ID), nil
	}
	return fmt.Sprintf("Spawned subagent for task: %s (id: %s)", task, subagentTask.ID), nil
}

// SpawnTask starts a background subagent task and returns an immutable snapshot of the created task.
func (sm *SubagentManager) SpawnTask(
	ctx context.Context,
	task, label, agentID, originChannel, originChatID string,
	callback AsyncCallback,
) (*SubagentTask, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.maxTasks > 0 {
		active := 0
		for _, item := range sm.tasks {
			if !isTerminalTaskStatus(item.Status) {
				active++
			}
		}
		if active >= sm.maxTasks {
			return nil, fmt.Errorf("max task limit reached (%d)", sm.maxTasks)
		}
	}
	if sm.maxConcurrent > 0 {
		running := 0
		for _, item := range sm.tasks {
			if item.Status == "running" {
				running++
			}
		}
		if running >= sm.maxConcurrent {
			return nil, fmt.Errorf("max concurrent task limit reached (%d)", sm.maxConcurrent)
		}
	}

	depth := 1
	parentTaskID := ""
	senderID := toolExecutionSenderID(ctx)
	if strings.HasPrefix(strings.ToLower(senderID), "subagent:") {
		parentID := strings.TrimSpace(senderID[len("subagent:"):])
		parentTaskID = parentID
		if parent, ok := sm.tasks[parentID]; ok && parent.Depth > 0 {
			depth = parent.Depth + 1
		} else {
			depth = 2
		}
	}
	if sm.maxDepth > 0 && depth > sm.maxDepth {
		return nil, fmt.Errorf("max spawn depth reached (%d)", sm.maxDepth)
	}

	taskID := fmt.Sprintf("subagent-%d", sm.nextID)
	sm.nextID++

	subagentTask := &SubagentTask{
		ID:            taskID,
		ParentTaskID:  parentTaskID,
		Task:          task,
		Label:         label,
		AgentID:       agentID,
		OriginChannel: originChannel,
		OriginChatID:  originChatID,
		Status:        "running",
		Created:       time.Now().UnixMilli(),
		Depth:         depth,
	}
	sm.tasks[taskID] = subagentTask
	taskCtx, cancel := context.WithCancel(ctx)
	sm.taskCancels[taskID] = cancel
	snapshot := *subagentTask

	// Start task in background with context cancellation support.
	go sm.runTask(taskCtx, subagentTask, callback)
	go sm.emitEvent(SubagentTaskEvent{
		Type:      SubagentTaskCreated,
		Task:      snapshot,
		Timestamp: time.Now().UnixMilli(),
	})
	return &snapshot, nil
}

func (sm *SubagentManager) runTask(ctx context.Context, task *SubagentTask, callback AsyncCallback) {
	defer sm.clearTaskCancel(task.ID)

	sm.mu.Lock()
	task.Status = "running"
	task.Created = time.Now().UnixMilli()
	snapshot := *task
	sm.mu.Unlock()
	sm.emitEvent(SubagentTaskEvent{
		Type:      SubagentTaskRunning,
		Task:      snapshot,
		Timestamp: time.Now().UnixMilli(),
	})

	// Build system prompt for subagent
	systemPrompt := `You are a subagent. Complete the given task independently and report the result.
You have access to tools - use them as needed to complete your task.
After completing the task, provide a clear summary of what was done.`

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
		task.Status = "cancelled"
		task.Result = "Task cancelled before execution"
		snapshot = *task
		sm.mu.Unlock()
		sm.emitEvent(SubagentTaskEvent{
			Type:      SubagentTaskCancelled,
			Task:      snapshot,
			Timestamp: time.Now().UnixMilli(),
		})
		sm.cancelTaskTree(task.ID)
		return
	default:
	}

	// Run tool loop with access to tools
	sm.mu.RLock()
	maxIter := sm.maxIterations
	maxTokens := sm.maxTokens
	temperature := sm.temperature
	hasMaxTokens := sm.hasMaxTokens
	hasTemperature := sm.hasTemperature
	resolver := sm.resolver
	defaultTools := sm.tools
	defaultProvider := sm.provider
	defaultModel := sm.defaultModel
	toolCallsParallelEnabled := sm.toolCallsParallelEnabled
	maxToolCallConcurrency := sm.maxToolCallConcurrency
	parallelToolsMode := sm.parallelToolsMode
	toolPolicyOverrides := clonePolicyOverrides(sm.toolPolicyOverrides)
	sm.mu.RUnlock()

	execution := SubagentExecutionConfig{
		Provider: defaultProvider,
		Model:    defaultModel,
		Tools:    defaultTools,
	}
	if resolver != nil {
		resolved, err := resolver(task.AgentID)
		if err != nil {
			sm.mu.Lock()
			task.Status = "failed"
			task.Result = fmt.Sprintf("Error: %v", err)
			snapshot = *task
			sm.mu.Unlock()
			sm.emitEvent(SubagentTaskEvent{
				Type:      SubagentTaskFailed,
				Task:      snapshot,
				Err:       err.Error(),
				Timestamp: time.Now().UnixMilli(),
			})
			if callback != nil {
				callback(ctx, &ToolResult{
					ForLLM:  task.Result,
					IsError: true,
					Err:     err,
				})
			}
			if sm.bus != nil {
				sm.publishTaskAnnouncement(snapshot, task.Status)
			}
			sm.cancelTaskTree(task.ID)
			return
		}
		if resolved.Provider != nil {
			execution.Provider = resolved.Provider
		}
		if resolved.Model != "" {
			execution.Model = resolved.Model
		}
		if resolved.Tools != nil {
			execution.Tools = resolved.Tools
		}
	}
	if execution.Tools == nil {
		execution.Tools = NewToolRegistry()
	}

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
		Provider:                 execution.Provider,
		Model:                    execution.Model,
		Tools:                    execution.Tools,
		MaxIterations:            maxIter,
		LLMOptions:               llmOptions,
		SenderID:                 fmt.Sprintf("subagent:%s", task.ID),
		ToolCallsParallelEnabled: toolCallsParallelEnabled,
		MaxToolCallConcurrency:   maxToolCallConcurrency,
		ParallelToolsMode:        parallelToolsMode,
		ToolPolicyOverrides:      toolPolicyOverrides,
	}, messages, task.OriginChannel, task.OriginChatID)

	var result *ToolResult
	event := SubagentTaskEvent{
		Task:      *task,
		Timestamp: time.Now().UnixMilli(),
	}
	if loopResult != nil {
		event.Trace = loopResult.Trace
	}

	if err != nil {
		sm.mu.Lock()
		task.Status = "failed"
		task.Result = fmt.Sprintf("Error: %v", err)
			// Check if it was cancelled
			if ctx.Err() != nil {
				task.Status = "cancelled"
				task.Result = "Task cancelled during execution"
			}
		snapshot = *task
		sm.mu.Unlock()
		result = &ToolResult{
			ForLLM:  task.Result,
			ForUser: "",
			Silent:  false,
			IsError: true,
			Async:   false,
			Err:     err,
		}
		event.Task = snapshot
		event.Err = err.Error()
		if task.Status == "cancelled" {
			event.Type = SubagentTaskCancelled
		} else {
			event.Type = SubagentTaskFailed
		}
		sm.cancelTaskTree(task.ID)
	} else {
		sm.mu.Lock()
		task.Status = "completed"
		task.Result = loopResult.Content
		snapshot = *task
		sm.mu.Unlock()
		result = &ToolResult{
			ForLLM: fmt.Sprintf(
				"Subagent '%s' completed (iterations: %d): %s",
				task.Label,
				loopResult.Iterations,
				loopResult.Content,
			),
			ForUser: loopResult.Content,
			Silent:  false,
			IsError: false,
			Async:   false,
		}
		event.Type = SubagentTaskCompleted
		event.Task = snapshot
	}
	sm.emitEvent(event)

	// Call callback if provided and result is set.
	if callback != nil && result != nil {
		callback(ctx, result)
	}

	// Send announce message back to main agent
	if sm.bus != nil {
		sm.publishTaskAnnouncement(snapshot, task.Status)
	}
}

func (sm *SubagentManager) GetTask(taskID string) (*SubagentTask, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	task, ok := sm.tasks[taskID]
	if !ok {
		return nil, false
	}
	snapshot := *task
	return &snapshot, true
}

func (sm *SubagentManager) ListTasks() []*SubagentTask {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	tasks := make([]*SubagentTask, 0, len(sm.tasks))
	for _, task := range sm.tasks {
		snapshot := *task
		tasks = append(tasks, &snapshot)
	}
	return tasks
}

func (sm *SubagentManager) emitEvent(event SubagentTaskEvent) {
	sm.mu.RLock()
	handler := sm.eventHandler
	sm.mu.RUnlock()
	if handler == nil {
		return
	}
	handler(event)
}

func (sm *SubagentManager) publishTaskAnnouncement(task SubagentTask, status string) {
	if sm.bus == nil {
		return
	}
	state := "completed"
	if status == "failed" {
		state = "failed"
	} else if status == "cancelled" {
		state = "cancelled"
	}
	announceContent := fmt.Sprintf("Task '%s' %s.\n\nResult:\n%s", task.Label, state, task.Result)
	sm.bus.PublishInbound(bus.InboundMessage{
		Channel:  "system",
		SenderID: fmt.Sprintf("subagent:%s", task.ID),
		// Format: "original_channel:original_chat_id" for routing back
		ChatID:  fmt.Sprintf("%s:%s", task.OriginChannel, task.OriginChatID),
		Content: announceContent,
	})
}

func (sm *SubagentManager) clearTaskCancel(taskID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.taskCancels, taskID)
}

func (sm *SubagentManager) cancelTaskTree(parentTaskID string) {
	queue := []string{strings.TrimSpace(parentTaskID)}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		var children []string
		var cancels []context.CancelFunc

		sm.mu.RLock()
		for taskID, task := range sm.tasks {
			if task.ParentTaskID != current {
				continue
			}
			children = append(children, taskID)
			if cancel, ok := sm.taskCancels[taskID]; ok {
				cancels = append(cancels, cancel)
			}
		}
		sm.mu.RUnlock()

		for _, cancel := range cancels {
			cancel()
		}
		queue = append(queue, children...)
	}
}

func isTerminalTaskStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "completed", "failed", "cancelled":
		return true
	default:
		return false
	}
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
	return "Execute a subagent task synchronously and return the result. Use this for delegating specific tasks to an independent agent instance. Returns execution summary to user and full details to LLM."
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
			"agent_id": map[string]any{
				"type":        "string",
				"description": "Optional target agent ID to delegate the task to",
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
		return ErrorResult("task is required").WithError(fmt.Errorf("task parameter is required"))
	}

	label, _ := args["label"].(string)
	agentID, _ := args["agent_id"].(string)

	if t.manager == nil {
		return ErrorResult("Subagent manager not configured").WithError(fmt.Errorf("manager is nil"))
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
	maxIter := sm.maxIterations
	maxTokens := sm.maxTokens
	temperature := sm.temperature
	hasMaxTokens := sm.hasMaxTokens
	hasTemperature := sm.hasTemperature
	resolver := sm.resolver
	toolCallsParallelEnabled := sm.toolCallsParallelEnabled
	maxToolCallConcurrency := sm.maxToolCallConcurrency
	parallelToolsMode := sm.parallelToolsMode
	toolPolicyOverrides := clonePolicyOverrides(sm.toolPolicyOverrides)
	execution := SubagentExecutionConfig{
		Provider: sm.provider,
		Model:    sm.defaultModel,
		Tools:    sm.tools,
	}
	sm.mu.RUnlock()

	if resolver != nil {
		resolved, resolveErr := resolver(agentID)
		if resolveErr != nil {
			return ErrorResult(fmt.Sprintf("Subagent execution failed: %v", resolveErr)).WithError(resolveErr)
		}
		if resolved.Provider != nil {
			execution.Provider = resolved.Provider
		}
		if resolved.Model != "" {
			execution.Model = resolved.Model
		}
		if resolved.Tools != nil {
			execution.Tools = resolved.Tools
		}
	}
	if execution.Tools == nil {
		execution.Tools = NewToolRegistry()
	}

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
		Provider:                 execution.Provider,
		Model:                    execution.Model,
		Tools:                    execution.Tools,
		MaxIterations:            maxIter,
		LLMOptions:               llmOptions,
		ToolCallsParallelEnabled: toolCallsParallelEnabled,
		MaxToolCallConcurrency:   maxToolCallConcurrency,
		ParallelToolsMode:        parallelToolsMode,
		ToolPolicyOverrides:      toolPolicyOverrides,
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
	llmContent := fmt.Sprintf("Subagent task completed:\nLabel: %s\nIterations: %d\nResult: %s",
		labelStr, loopResult.Iterations, loopResult.Content)

	return &ToolResult{
		ForLLM:  llmContent,
		ForUser: userContent,
		Silent:  false,
		IsError: false,
		Async:   false,
	}
}
