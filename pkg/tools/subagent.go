package tools

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sipeed/picoclaw/pkg/providers"
	taskregistry "github.com/sipeed/picoclaw/pkg/tasks"
)

var labeledArtifactPathRe = regexp.MustCompile(`(?im)(?:^|\n)\s*(?:[-*]\s*)?(?:sendable_file_path|file_path|artifact_path|local_path)\s*:\s*` + "`?" + `([^` + "`" + `\n]+)` + "`?")

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
	TargetAgentID      string        // If set, run as this agent (its workspace, model, tools)
	DeliveryMode       AsyncDeliveryMode
}

type SubagentTask struct {
	ID            string
	Task          string
	Label         string
	AgentID       string
	BoardID       string
	ParentTaskID  string
	StepID        string
	StepTitle     string
	Owner         string
	DependsOn     []string
	BlockedBy     []string
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
	workspace      string
	tools          *ToolRegistry
	maxIterations  int
	maxTokens      int
	temperature    float64
	hasMaxTokens   bool
	hasTemperature bool
	nextID         int
	spawner        SpawnSubTurnFunc
	taskRegistry   *taskregistry.Registry

	// mediaResolver resolves media:// refs in tool-loop messages before
	// each LLM call in the legacy RunToolLoop fallback path.
	// This lets subagents reuse the same media handling behavior as the
	// main agent loop without importing pkg/agent and creating a cycle.
	mediaResolver func([]providers.Message) []providers.Message
}

func NewSubagentManager(
	provider providers.LLMProvider,
	defaultModel, workspace string,
) *SubagentManager {
	registry := taskregistry.NewRegistry(taskregistry.WorkspaceStorePath(workspace))
	return NewSubagentManagerWithRegistry(provider, defaultModel, workspace, registry)
}

func NewSubagentManagerWithRegistry(
	provider providers.LLMProvider,
	defaultModel, workspace string,
	registry *taskregistry.Registry,
) *SubagentManager {
	if registry == nil {
		registry = taskregistry.NewRegistry(taskregistry.WorkspaceStorePath(workspace))
	}
	manager := &SubagentManager{
		tasks:         make(map[string]*SubagentTask),
		provider:      provider,
		defaultModel:  defaultModel,
		workspace:     workspace,
		tools:         NewToolRegistry(),
		maxIterations: 10,
		nextID:        registry.MaxNumericSuffix("subagent-") + 1,
		taskRegistry:  registry,
	}
	if manager.nextID <= 0 {
		manager.nextID = 1
	}
	manager.restoreTasksFromRegistry()
	return manager
}

func (sm *SubagentManager) SetSpawner(spawner SpawnSubTurnFunc) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.spawner = spawner
}

// SetMediaResolver injects a message preprocessor that resolves media:// refs
// into LLM-ready content before each tool-loop iteration.
// This is only used by the legacy RunToolLoop fallback path.
func (sm *SubagentManager) SetMediaResolver(
	resolver func([]providers.Message) []providers.Message,
) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.mediaResolver = resolver
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
	boardMeta ...TaskBoardMetadata,
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
	if len(boardMeta) > 0 {
		subagentTask.BoardID = boardMeta[0].BoardID
		subagentTask.ParentTaskID = boardMeta[0].ParentTaskID
		subagentTask.StepID = boardMeta[0].StepID
		subagentTask.StepTitle = boardMeta[0].StepTitle
		subagentTask.Owner = boardMeta[0].Owner
		subagentTask.DependsOn = append([]string(nil), boardMeta[0].DependsOn...)
		subagentTask.BlockedBy = append([]string(nil), boardMeta[0].BlockedBy...)
	}
	sm.tasks[taskID] = subagentTask
	sm.recordTask(subagentTask, taskregistry.StatusRunning, taskregistry.DeliveryPending, "")

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
	// TODO(eventbus): once subagents are modeled as child turns inside
	// pkg/agent, emit SubTurnEnd and SubTurnResultDelivered from the parent
	// AgentLoop instead of this legacy manager.

	// Check if context is already canceled before starting
	select {
	case <-ctx.Done():
		sm.mu.Lock()
		task.Status = "canceled"
		task.Result = "Task canceled before execution"
		sm.mu.Unlock()
		sm.recordTask(task, taskregistry.StatusCancelled, taskregistry.DeliveryNotApplicable, task.Result)
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
	mediaResolver := sm.mediaResolver
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
			MediaResolver: mediaResolver,
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
		// Call callback if provided and result is set
		if callback != nil && result != nil {
			result.WithAsyncTaskID(task.ID)
			callback(ctx, result)
		}
	}()

	if err != nil {
		task.Status = "failed"
		task.Result = fmt.Sprintf("Error: %v", err)
		// Only report cancellation when cancellation is the actual cause.
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			task.Status = "canceled"
			task.Result = "Task canceled during execution"
			sm.recordTask(task, taskregistry.StatusCancelled, taskregistry.DeliveryPending, task.Result)
		} else {
			sm.recordTask(task, taskregistry.StatusFailed, taskregistry.DeliveryPending, task.Result)
		}
		result = &ToolResult{
			ForLLM:  task.Result,
			ForUser: task.Result,
			Silent:  false,
			IsError: true,
			Async:   false,
			Err:     err,
		}
	} else {
		task.Status = "completed"
		result.WithAsyncTaskID(task.ID)
		task.Result = result.ForLLM
		sm.recordTaskResult(task, result)
	}
}

func (sm *SubagentManager) restoreTasksFromRegistry() {
	if sm == nil || sm.taskRegistry == nil {
		return
	}
	for _, rec := range sm.taskRegistry.List() {
		if rec.Runtime != taskregistry.RuntimeSubagent {
			continue
		}
		sm.tasks[rec.TaskID] = subagentTaskFromRecord(rec)
	}
}

func subagentTaskFromRecord(rec taskregistry.Record) *SubagentTask {
	status := "running"
	switch rec.Status {
	case taskregistry.StatusSucceeded:
		status = "completed"
	case taskregistry.StatusFailed:
		status = "failed"
	case taskregistry.StatusCancelled, taskregistry.StatusTimedOut:
		status = "canceled"
	case taskregistry.StatusRunning, taskregistry.StatusQueued:
		status = "running"
	}
	return &SubagentTask{
		ID:            rec.TaskID,
		Task:          rec.Task,
		Label:         rec.Label,
		AgentID:       rec.AgentID,
		BoardID:       rec.BoardID,
		ParentTaskID:  rec.ParentTaskID,
		StepID:        rec.StepID,
		StepTitle:     rec.StepTitle,
		Owner:         rec.Owner,
		DependsOn:     append([]string(nil), rec.DependsOn...),
		BlockedBy:     append([]string(nil), rec.BlockedBy...),
		OriginChannel: rec.Channel,
		OriginChatID:  rec.ChatID,
		Status:        status,
		Result:        rec.TerminalSummary,
		Created:       rec.CreatedAt,
	}
}

func (sm *SubagentManager) recordTask(
	task *SubagentTask,
	status taskregistry.Status,
	delivery taskregistry.DeliveryStatus,
	summary string,
) {
	if sm == nil || sm.taskRegistry == nil || task == nil {
		return
	}
	now := time.Now().UnixMilli()
	rec := taskregistry.Record{
		TaskID:         task.ID,
		Runtime:        taskregistry.RuntimeSubagent,
		TaskKind:       "spawn",
		BoardID:        task.BoardID,
		ParentTaskID:   task.ParentTaskID,
		StepID:         task.StepID,
		StepTitle:      task.StepTitle,
		Owner:          task.Owner,
		DependsOn:      append([]string(nil), task.DependsOn...),
		BlockedBy:      append([]string(nil), task.BlockedBy...),
		Channel:        task.OriginChannel,
		ChatID:         task.OriginChatID,
		AgentID:        task.AgentID,
		Label:          task.Label,
		Task:           task.Task,
		Status:         status,
		DeliveryStatus: delivery,
		NotifyPolicy:   taskregistry.NotifyDoneOnly,
		CreatedAt:      task.Created,
		StartedAt:      task.Created,
		LastEventAt:    now,
	}
	if rec.CreatedAt == 0 {
		rec.CreatedAt = now
	}
	if rec.StartedAt == 0 {
		rec.StartedAt = rec.CreatedAt
	}
	if status == taskregistry.StatusSucceeded || status == taskregistry.StatusFailed || status == taskregistry.StatusCancelled || status == taskregistry.StatusTimedOut {
		rec.EndedAt = now
		rec.TerminalSummary = summary
	}
	if status == taskregistry.StatusFailed {
		rec.Error = summary
	}
	_ = sm.taskRegistry.Upsert(rec)
}

func (sm *SubagentManager) recordTaskResult(task *SubagentTask, result *ToolResult) {
	if sm == nil || sm.taskRegistry == nil || task == nil {
		return
	}
	summary := ""
	if result != nil {
		summary = result.ContentForLLM()
	}
	delivery := taskregistry.DeliveryPending
	if result == nil || (result.Silent && result.AsyncDelivery == AsyncDeliveryParentOnly) {
		delivery = taskregistry.DeliveryNotApplicable
	}
	completion := completionPayloadForTaskRegistry(result)
	deliverable := deliverablePayloadForTaskRegistry(result)
	sm.recordTask(task, taskregistry.StatusSucceeded, delivery, summary)
	if completion != nil || deliverable != nil {
		_ = sm.taskRegistry.Update(task.ID, func(rec *taskregistry.Record) {
			rec.Completion = completion
			rec.Deliverable = deliverable
			rec.LastEventAt = time.Now().UnixMilli()
		})
	}
}

func completionPayloadForTaskRegistry(result *ToolResult) *taskregistry.CompletionPayload {
	if result == nil || result.Completion == nil {
		return nil
	}
	payload := &taskregistry.CompletionPayload{Text: result.Completion.Text}
	for _, item := range result.Completion.Media {
		payload.Media = append(payload.Media, taskregistry.CompletionMedia{
			Ref:         item.Ref,
			Type:        item.Type,
			Filename:    item.Filename,
			ContentType: item.ContentType,
		})
	}
	if payload.Text == "" && len(payload.Media) == 0 {
		return nil
	}
	return payload
}

func deliverablePayloadForTaskRegistry(result *ToolResult) *taskregistry.DeliverablePayload {
	if result == nil {
		return nil
	}
	deliverable := result.Deliverable
	if deliverable == nil && result.Completion != nil {
		deliverable = &DeliverableResult{
			Text: result.Completion.Text,
		}
		for _, item := range result.Completion.Media {
			deliverable.Artifacts = append(deliverable.Artifacts, DeliverableItem{
				Ref:         item.Ref,
				Kind:        item.Type,
				Filename:    item.Filename,
				ContentType: item.ContentType,
			})
		}
	}
	if deliverable == nil {
		return nil
	}
	if result.Completion != nil {
		deliverable.Artifacts = appendMissingDeliverableArtifacts(
			deliverable.Artifacts,
			extractLabeledArtifactItems(result.Completion.Text),
		)
	}
	payload := &taskregistry.DeliverablePayload{
		Text:     deliverable.Text,
		Metadata: copyDeliverableMetadata(deliverable.Metadata),
	}
	for _, item := range deliverable.Artifacts {
		payload.Artifacts = append(payload.Artifacts, taskregistry.DeliverableItem{
			Ref:         item.Ref,
			Kind:        item.Kind,
			Filename:    item.Filename,
			ContentType: item.ContentType,
			Delivered:   item.Delivered,
		})
	}
	if payload.Text == "" && len(payload.Artifacts) == 0 && len(payload.Metadata) == 0 {
		return nil
	}
	return payload
}

func appendMissingDeliverableArtifacts(existing, extra []DeliverableItem) []DeliverableItem {
	if len(extra) == 0 {
		return existing
	}
	seen := make(map[string]struct{}, len(existing)+len(extra))
	for _, item := range existing {
		if ref := strings.TrimSpace(item.Ref); ref != "" {
			seen[ref] = struct{}{}
		}
	}
	out := append([]DeliverableItem(nil), existing...)
	for _, item := range extra {
		ref := strings.TrimSpace(item.Ref)
		if ref == "" {
			continue
		}
		if _, ok := seen[ref]; ok {
			continue
		}
		seen[ref] = struct{}{}
		out = append(out, item)
	}
	return out
}

func extractLabeledArtifactItems(text string) []DeliverableItem {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	matches := labeledArtifactPathRe.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(matches))
	items := make([]DeliverableItem, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		path := normalizeCompletionArtifactPath(match[1])
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		items = append(items, DeliverableItem{
			Ref:         "file:" + path,
			Kind:        artifactKindForPath(path),
			Filename:    filepath.Base(path),
			ContentType: contentTypeForArtifactPath(path),
		})
	}
	return items
}

func normalizeCompletionArtifactPath(raw string) string {
	path := strings.TrimSpace(raw)
	path = strings.Trim(path, "`'\"")
	if idx := strings.IndexAny(path, " \t\r"); idx >= 0 {
		path = path[:idx]
	}
	path = strings.TrimRight(path, ".,;)")
	if !strings.HasPrefix(path, "/") {
		return ""
	}
	return filepath.Clean(path)
}

func artifactKindForPath(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp4", ".mov", ".m4v", ".webm", ".mkv":
		return "video"
	case ".mp3", ".m4a", ".wav", ".ogg", ".flac":
		return "audio"
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".heic":
		return "image"
	case ".md", ".txt", ".json", ".csv", ".pdf":
		return "file"
	default:
		return "file"
	}
}

func contentTypeForArtifactPath(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp4":
		return "video/mp4"
	case ".mov":
		return "video/quicktime"
	case ".webm":
		return "video/webm"
	case ".mp3":
		return "audio/mpeg"
	case ".m4a":
		return "audio/mp4"
	case ".wav":
		return "audio/wav"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".md":
		return "text/markdown"
	case ".txt":
		return "text/plain"
	case ".json":
		return "application/json"
	case ".csv":
		return "text/csv"
	case ".pdf":
		return "application/pdf"
	default:
		return ""
	}
}

func copyDeliverableMetadata(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
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
	spawner      SubTurnSpawner
	defaultModel string
	maxTokens    int
	temperature  float64
}

func NewSubagentTool(manager *SubagentManager) *SubagentTool {
	if manager == nil {
		return &SubagentTool{}
	}
	return &SubagentTool{
		defaultModel: manager.defaultModel,
		maxTokens:    manager.maxTokens,
		temperature:  manager.temperature,
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

	// Build system prompt for subagent
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
			Model:        t.defaultModel,
			Tools:        nil, // Will inherit from parent via context
			SystemPrompt: systemPrompt,
			MaxTokens:    t.maxTokens,
			Temperature:  t.temperature,
			Async:        false, // Synchronous execution
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
