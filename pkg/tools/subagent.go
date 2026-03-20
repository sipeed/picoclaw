package tools

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/providers"
)

// attributedContent prepends "AgentName: " to content when agentID is non-empty.
func attributedContent(agentID, content string) string {
	if strings.TrimSpace(agentID) == "" {
		return content
	}
	name := strings.ToUpper(agentID[:1]) + agentID[1:]
	return "**" + name + ":** " + content
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

type SubagentManager struct {
	tasks          map[string]*SubagentTask
	mu             sync.RWMutex
	provider       providers.LLMProvider
	defaultModel   string
	workspace      string
	tools          *ToolRegistry
	maxIterations  int
	maxTokens      int
	temperature    float64
	hasMaxTokens   bool
	hasTemperature bool
	nextID         int

	// Per-agent dispatch fields. When dispatcher is set, subagent LLM calls
	// are routed through the dispatcher using selfCandidates (for self-spawns)
	// or the target agent's candidates (resolved via candidateResolver).
	dispatcher        *providers.ProviderDispatcher
	fallback          *providers.FallbackChain
	selfCandidates    []providers.FallbackCandidate
	callerAgentID     string
	candidateResolver func(agentID string) ([]providers.FallbackCandidate, bool)
}

// SubagentManagerConfig holds all configuration for constructing a SubagentManager.
type SubagentManagerConfig struct {
	// Provider is the fallback LLM provider used when dispatcher lookup fails.
	Provider providers.LLMProvider
	// DefaultModel is the model name used when no candidates are resolved.
	DefaultModel string
	// Workspace is the agent's working directory.
	Workspace string
	// Dispatcher dispatches LLM calls per-candidate. Optional.
	Dispatcher *providers.ProviderDispatcher
	// Fallback is the chain used when multiple candidates are configured. Optional.
	Fallback *providers.FallbackChain
	// SelfCandidates are the calling agent's model candidates for self-spawns.
	SelfCandidates []providers.FallbackCandidate
	// CallerAgentID is the ID of the agent that owns this manager (for allowlist).
	CallerAgentID string
	// CandidateResolver resolves model candidates for a named target agent.
	// Returns false if the agent is unknown.
	CandidateResolver func(agentID string) ([]providers.FallbackCandidate, bool)
}

func NewSubagentManager(cfg SubagentManagerConfig) *SubagentManager {
	return &SubagentManager{
		tasks:             make(map[string]*SubagentTask),
		provider:          cfg.Provider,
		defaultModel:      cfg.DefaultModel,
		workspace:         cfg.Workspace,
		tools:             NewToolRegistry(),
		maxIterations:     10,
		nextID:            1,
		dispatcher:        cfg.Dispatcher,
		fallback:          cfg.Fallback,
		selfCandidates:    cfg.SelfCandidates,
		callerAgentID:     cfg.CallerAgentID,
		candidateResolver: cfg.CandidateResolver,
	}
}

// resolveLoopConfig builds a ToolLoopConfig for the given target agent ID.
// When agentID is empty it is treated as a self-spawn and uses selfCandidates.
// When agentID is non-empty and a candidateResolver is set, it resolves the
// target agent's candidates. Falls back to provider+defaultModel when no
// dispatch metadata is available.
func (sm *SubagentManager) resolveLoopConfig(agentID string) ToolLoopConfig {
	candidates := sm.selfCandidates
	if agentID != "" && sm.candidateResolver != nil {
		if resolved, ok := sm.candidateResolver(agentID); ok {
			candidates = resolved
		}
	}

	cfg := ToolLoopConfig{
		Provider:      sm.provider,
		Model:         sm.defaultModel,
		MaxIterations: sm.maxIterations,
		Tools:         sm.tools,
		Dispatcher:    sm.dispatcher,
		Fallback:      sm.fallback,
		Candidates:    candidates,
	}

	// Override Model from first candidate when available.
	if len(candidates) > 0 {
		cfg.Model = candidates[0].Model
	}

	if sm.hasMaxTokens || sm.hasTemperature {
		opts := map[string]any{}
		if sm.hasMaxTokens {
			opts["max_tokens"] = sm.maxTokens
		}
		if sm.hasTemperature {
			opts["temperature"] = sm.temperature
		}
		cfg.LLMOptions = opts
	}

	return cfg
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

func (sm *SubagentManager) runTask(ctx context.Context, task *SubagentTask, callback AsyncCallback) {
	task.Status = "running"
	task.Created = time.Now().UnixMilli()

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
		task.Status = "canceled"
		task.Result = "Task canceled before execution"
		sm.mu.Unlock()
		return
	default:
	}

	sm.mu.RLock()
	loopCfg := sm.resolveLoopConfig(task.AgentID)
	sm.mu.RUnlock()

	loopResult, err := RunToolLoop(ctx, loopCfg, messages, task.OriginChannel, task.OriginChatID)

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
		task.Result = loopResult.Content
		result = &ToolResult{
			ForLLM: fmt.Sprintf(
				"Subagent '%s' completed (iterations: %d): %s",
				task.Label,
				loopResult.Iterations,
				loopResult.Content,
			),
			ForUser: attributedContent(task.AgentID, loopResult.Content),
			Silent:  false,
			IsError: false,
			Async:   false,
		}
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
}

func NewSubagentTool(manager *SubagentManager) *SubagentTool {
	return &SubagentTool{
		manager: manager,
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

	// Use RunToolLoop to execute with tools (same as async SpawnTool).
	// SubagentTool always performs a self-spawn (no explicit agent_id).
	sm := t.manager
	sm.mu.RLock()
	loopCfg := sm.resolveLoopConfig("")
	sm.mu.RUnlock()

	// Fall back to "cli"/"direct" for non-conversation callers (e.g., CLI, tests)
	// to preserve the same defaults as the original NewSubagentTool constructor.
	channel := ToolChannel(ctx)
	if channel == "" {
		channel = "cli"
	}
	chatID := ToolChatID(ctx)
	if chatID == "" {
		chatID = "direct"
	}

	loopResult, err := RunToolLoop(ctx, loopCfg, messages, channel, chatID)
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
		ForUser: attributedContent(agentID, userContent),
		Silent:  false,
		IsError: false,
		Async:   false,
	}
}
