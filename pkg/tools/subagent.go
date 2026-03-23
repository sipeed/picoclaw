package tools

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// SubTurnSpawner is an interface for spawning sub-turns.
// This avoids circular dependency between tools and agent packages.
type SubTurnSpawner interface {
	SpawnSubTurn(ctx context.Context, cfg SubTurnConfig) (*ToolResult, error)
}

// SubTurnConfig holds configuration for spawning a sub-turn.
type SubTurnConfig struct {
	Model              string
	Provider           providers.LLMProvider // non-nil overrides the child agent's provider
	Tools              []Tool
	EmptyTools         bool          // true: child agent gets an empty ToolRegistry (overrides Tools)
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

// ModelTag constants define the recognized capability labels for models in config.json.
// These are set via `"tags": ["vision", "code"]` under each model in the model list.
const (
	ModelTagVision      = "vision"       // Supports image/screenshot input (multimodal)
	ModelTagImageGen    = "image-gen"    // Supports image generation output (e.g. DALL-E, Stable Diffusion)
	ModelTagCode        = "code"         // Specialized for code generation and analysis
	ModelTagFast        = "fast"         // Low-latency model, suited for lightweight tasks
	ModelTagLongContext = "long-context" // Supports very long context windows (>100k tokens)
	ModelTagReasoning   = "reasoning"    // Strong logical/math reasoning (e.g., o1, deepseek-r1)
)

// modelTagDescriptions provides LLM-readable explanations of each known tag,
// injected at runtime into the tool description to guide model selection.
var modelTagDescriptions = map[string]string{
	ModelTagVision:      "can analyze images and screenshots (multimodal input)",
	ModelTagImageGen:    "can generate images from text descriptions (e.g. DALL-E, Stable Diffusion)",
	ModelTagCode:        "specialized in code generation and debugging",
	ModelTagFast:        "fast and lightweight, ideal for simple or high-frequency tasks",
	ModelTagLongContext: "handles very long inputs (>100k tokens)",
	ModelTagReasoning:   "excels at logical reasoning, math, and multi-step planning",
}

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
	tasks          map[string]*SubagentTask
	mu             sync.RWMutex
	provider       providers.LLMProvider
	defaultModel   string
	allowedModels  []providers.FallbackCandidate
	bus            *bus.MessageBus
	workspace      string
	tools          *ToolRegistry
	teamConfig     config.TeamToolsConfig
	maxIterations  int
	maxTokens      int
	temperature    float64
	hasMaxTokens   bool
	hasTemperature bool
	nextID         int
	spawner        SpawnSubTurnFunc
}

func NewSubagentManager(
	provider providers.LLMProvider,
	defaultModel string,
	candidates []providers.FallbackCandidate,
	workspace string,
	teamConfig config.TeamToolsConfig,
	bus *bus.MessageBus,
) *SubagentManager {
	return &SubagentManager{
		tasks:         make(map[string]*SubagentTask),
		provider:      provider,
		defaultModel:  defaultModel,
		allowedModels: candidates,
		teamConfig:    teamConfig,
		bus:           bus,
		workspace:     workspace,
		tools:         NewToolRegistry(),
		maxIterations: 10,
		nextID:        1,
	}
}

// IsModelAllowed checks if a specific requested model exists in the permitted candidates list.
func (sm *SubagentManager) IsModelAllowed(model string) bool {
	// If the user requested the default model directly, that's automatically allowed
	if model == sm.defaultModel {
		return true
	}

	// 1. Check against explicitly allowed models in team config
	for _, cand := range sm.teamConfig.AllowedModels {
		if cand.Name == model {
			return true
		}
	}

	// 2. Otherwise, check against the resolved candidates (primary + fallbacks + explicitly configured)
	// If teamConfig.AllowedModels is set, we strictly enforce it and DO NOT fall back to candidates
	// unless the candidate model has tags that overlap with AllowedTags. But since AllowedTags
	// was not implemented yet, just check fallback for backwards compatibility if teamConfig is empty.
	if len(sm.teamConfig.AllowedModels) > 0 {
		return false
	}

	for _, cand := range sm.allowedModels {
		if cand.Model == model {
			return true
		}
	}
	return false
}

// ModelCapabilityHint generates a human-readable summary of allowed models and their tags.
// This is injected into the coordinator's tool descriptions so the LLM can make better routing decisions.
func (sm *SubagentManager) ModelCapabilityHint() string {
	if len(sm.allowedModels) == 0 {
		return ""
	}

	var modelLines []string
	for _, cand := range sm.allowedModels {
		modelLines = append(modelLines, fmt.Sprintf("  - %s (general purpose)", cand.Model))
	}

	hint := "When selecting a 'model' for sub-agents, use ONLY these configured models:\n"
	if len(sm.teamConfig.AllowedModels) > 0 {
		for _, cand := range sm.teamConfig.AllowedModels {
			tagsStr := ""
			if len(cand.Tags) > 0 {
				tagsStr = fmt.Sprintf(" [%s]", strings.Join(cand.Tags, ", "))
			}
			hint += fmt.Sprintf("  - %s%s\n", cand.Name, tagsStr)
		}
	} else {
		hint += strings.Join(modelLines, "\n")
	}

	hint += "\nIf a task requires vision/image analysis, you MUST select a model with the 'vision' tag. If no suitable model is available, omit the 'model' field to use the default."
	return hint
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

// RegisterTool registers a tool for subagent execution.
func (sm *SubagentManager) RegisterTool(tool Tool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.tools.Register(tool)
}

func (sm *SubagentManager) Spawn(
	ctx context.Context,
	task, label, agentID, originChannel, originChatID string,
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

	// Start task in background with context cancellation support
	go sm.runTask(ctx, subagentTask, callback)

	if label != "" {
		return fmt.Sprintf("Spawned subagent '%s' for task: %s", label, task), nil
	}
	return fmt.Sprintf("Spawned subagent for task: %s", task), nil
}

func (sm *SubagentManager) runTask(
	ctx context.Context,
	task *SubagentTask,
	callback AsyncCallback,
) {
	task.Status = "running"
	task.Created = time.Now().UnixMilli()

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

	sm.mu.RLock()
	spawner := sm.spawner
	tools := sm.tools
	maxIter := sm.maxIterations
	maxTokens := sm.maxTokens
	temperature := sm.temperature
	hasMaxTokens := sm.hasMaxTokens
	hasTemperature := sm.hasTemperature
	sm.mu.RUnlock()

	var result *ToolResult
	var err error

	if spawner != nil {
		result, err = spawner(
			ctx,
			task.Task,
			task.Label,
			task.AgentID,
			tools,
			maxTokens,
			temperature,
			hasMaxTokens,
			hasTemperature,
		)
	} else {
		// Fallback to legacy RunToolLoop
		systemPrompt := `You are a subagent. Complete the given task independently and report the result.
You have access to tools - use them as needed to complete your task.
After completing the task, provide a clear summary of what was done.`

		messages := []providers.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: task.Task},
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

		var loopResult *ToolLoopResult
		loopResult, err = RunToolLoop(ctx, ToolLoopConfig{
			Provider:      sm.provider,
			Model:         sm.defaultModel,
			Tools:         tools,
			MaxIterations: maxIter,
			LLMOptions:    llmOptions,
		}, messages, task.OriginChannel, task.OriginChatID)

		if err == nil {
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
		}
	}

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
		if ctx.Err() != nil {
			task.Status = "canceled"
			task.Result = "Task canceled during execution"
		}
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
		task.Result = result.ForLLM
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

// BuildBaseWorkerConfig returns a base ToolLoopConfig that can be customized for isolated workers.
func (sm *SubagentManager) BuildBaseWorkerConfig(ctx context.Context) ToolLoopConfig {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var llmOptions map[string]any
	if sm.hasMaxTokens || sm.hasTemperature {
		llmOptions = map[string]any{}
		if sm.hasMaxTokens {
			llmOptions["max_tokens"] = sm.maxTokens
		}
		if sm.hasTemperature {
			llmOptions["temperature"] = sm.temperature
		}
	}

	return ToolLoopConfig{
		Provider:      sm.provider,
		Model:         sm.defaultModel,
		Tools:         sm.tools,
		MaxIterations: sm.maxIterations,
		LLMOptions:    llmOptions,
	}
}

// SubagentTool executes a subagent task synchronously and returns the result.
// It directly calls SubTurnSpawner with Async=false for synchronous execution.
type SubagentTool struct {
	spawner        SubTurnSpawner
	defaultModel   string
	maxTokens      int
	temperature    float64
	isModelAllowed func(string) bool // nil means no allowlist check
	modelHint      func() string     // nil means no model hint in description
}

func NewSubagentTool(manager *SubagentManager) *SubagentTool {
	if manager == nil {
		return &SubagentTool{}
	}
	return &SubagentTool{
		defaultModel:   manager.defaultModel,
		maxTokens:      manager.maxTokens,
		temperature:    manager.temperature,
		isModelAllowed: manager.IsModelAllowed,
		modelHint:      manager.ModelCapabilityHint,
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
	base := "Execute a subagent task synchronously and return the result. Use this for delegating specific tasks to an independent agent instance with an optional role (system prompt) and model. Returns execution summary to user and full details to LLM."
	if t.modelHint != nil {
		if hint := t.modelHint(); hint != "" {
			return base + "\n\n" + hint
		}
	}
	return base
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
			"role": map[string]any{
				"type":        "string",
				"description": "Optional system prompt / role assignment for the subagent (e.g. 'You are an expert code reviewer'). If omitted, a default subagent prompt is used.",
			},
			"model": map[string]any{
				"type":        "string",
				"description": "Optional specific LLM model ID to route this task to. If omitted, inherits the parent's model.",
			},
		},
		"required": []string{"task"},
	}
}

func (t *SubagentTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	task, ok := args["task"].(string)
	if !ok {
		return ErrorResult("task is required").WithError(fmt.Errorf("task parameter is required"))
	}

	label, _ := args["label"].(string)
	role, _ := args["role"].(string)
	modelParam, _ := args["model"].(string)
	modelParam = strings.TrimSpace(modelParam)

	// Validate model against allowlist if provided
	if modelParam != "" && t.isModelAllowed != nil && !t.isModelAllowed(modelParam) {
		return ErrorResult(fmt.Sprintf("requested model '%s' is not in the allowed models list", modelParam)).
			WithError(fmt.Errorf("model %s not allowed", modelParam))
	}

	// Determine the model to use
	targetModel := t.defaultModel
	if modelParam != "" {
		targetModel = modelParam
	}

	// Build ActualSystemPrompt: prefer explicit role, fall back to auto-generated prompt
	var actualSystemPrompt string
	if role != "" {
		actualSystemPrompt = role
	}

	// Build SystemPrompt (task description, becomes first user message in sub-turn)
	systemPrompt := fmt.Sprintf(
		`You are a subagent. Complete the given task independently and provide a clear, concise result.

Task: %s`,
		task,
	)
	if label != "" {
		systemPrompt = fmt.Sprintf(
			`You are a subagent labeled "%s". Complete the given task independently and provide a clear, concise result.

Task: %s`,
			label,
			task,
		)
	}

	// Use spawner if available (direct SpawnSubTurn call)
	if t.spawner != nil {
		result, err := t.spawner.SpawnSubTurn(ctx, SubTurnConfig{
			Model:              targetModel,
			Tools:              nil, // Will inherit from parent via context
			SystemPrompt:       systemPrompt,
			ActualSystemPrompt: actualSystemPrompt,
			MaxTokens:          t.maxTokens,
			Temperature:        t.temperature,
			Async:              false, // Synchronous execution
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

	// Fallback: spawner not configured
	return ErrorResult("Subagent manager not configured").WithError(fmt.Errorf("spawner not set"))
}
