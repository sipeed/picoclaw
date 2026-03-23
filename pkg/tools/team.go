package tools

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
)

type TeamTool struct {
	manager       *SubagentManager
	spawner       SubTurnSpawner
	cfg           *config.Config
	originChannel string
	originChatID  string
}

type TeamMember struct {
	ID        string
	Role      string
	Task      string
	Model     string   // Heterogeneous Agents: Optional specific model for this task
	DependsOn []string // List of member IDs this member depends on
	Produces  string   // Auto-reviewer: declares artifact type ("code", "data", "document")
}

func NewTeamTool(manager *SubagentManager, cfg *config.Config) *TeamTool {
	return &TeamTool{
		manager:       manager,
		cfg:           cfg,
		originChannel: "cli",
		originChatID:  "direct",
	}
}

// SetSpawner sets the SubTurnSpawner used to execute team members as sub-turns.
func (t *TeamTool) SetSpawner(spawner SubTurnSpawner) {
	t.spawner = spawner
}

func (t *TeamTool) Name() string {
	return "team"
}

func (t *TeamTool) Description() string {
	base := `Compose and execute a team of specialized sub-agents to accomplish a complex task.

WHEN TO USE THIS TOOL (use proactively — do not attempt to handle these alone):
- The task involves 2 or more distinct areas of concern (e.g. research + writing, coding + testing, data gathering + analysis).
- The task would require more than 5 consecutive tool calls if done alone.
- Any part of the task can be done in parallel to save time.
- The task is large enough that a single agent would likely lose context or quality midway.
- The user asks you to "build", "create", "generate", "analyze", or "convert" something non-trivial.
When in doubt, prefer delegation over doing everything yourself.

CRITICAL RULES FOR TASK PLANNING:
1. Think like a project manager: analyze the full task first, then design the team structure before spawning anyone.
2. Decompose the task into the smallest independently-ownable units of work. A member should own exactly ONE distinct concern — not a broad compound goal.
3. Identify dependencies between units: if one member's output is required by another, declare it via 'depends_on'. Independent units should run concurrently.
4. Each member's 'task' must be precise and self-contained. Include relevant context (e.g. reference to outputs from dependencies) directly in the task description.
5. Sub-agents are full agents with access to the same tools, including this 'team' tool. If a member's sub-task is itself complex, it may recursively form its own team.

Strategy guide:
- sequential: each step depends on the full output of the previous step in a strict chain.
- parallel: all tasks are fully independent with no shared inputs or outputs.
- dag: most real-world tasks — some tasks depend on others, some can run concurrently.
- evaluator_optimizer: the output needs iterative critique and revision cycles.`

	if t.manager != nil {
		if hint := t.manager.ModelCapabilityHint(); hint != "" {
			return base + "\n\n" + hint
		}
	}
	return base
}

func (t *TeamTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"strategy": map[string]any{
				"type":        "string",
				"enum":        []string{"sequential", "parallel", "dag", "evaluator_optimizer"},
				"description": "How to run the team members. 'sequential': one after another. 'parallel': all at once. 'dag': execute based on declared dependencies. 'evaluator_optimizer': EXACTLY two members (worker & evaluator). The evaluator will check the worker's output; if it fails, the worker is revived with its FULL stateful memory intact and asked to fix it. Use this for complex generation tasks (like coding) requiring deep reasoning.",
			},
			"max_team_tokens": map[string]any{
				"type":        "integer",
				"description": "The maximum combined LLM tokens (prompt + completion) this entire team is allowed to consume. Once exceeded, the team is instantly killed.",
			},
			"members": map[string]any{
				"type":        "array",
				"description": "The list of sub-agents in the team.",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"id": map[string]any{
							"type":        "string",
							"description": "Unique identifier for this member, used for dependencies in 'dag' strategy.",
						},
						"role": map[string]any{
							"type":        "string",
							"description": "The system prompt/role assignment for the member.",
						},
						"task": map[string]any{
							"type":        "string",
							"description": "The specific task this member needs to accomplish.",
						},
						"model": map[string]any{
							"type":        "string",
							"description": "Optional specific LLM model ID to route this task to (e.g., 'gpt-4o' for vision, 'claude-3-5-sonnet' for logic). If omitted, inherits the parent's model.",
						},
						"depends_on": map[string]any{
							"type":        "array",
							"description": "List of 'id' strings this member depends on. Only applicable for 'dag' strategy.",
							"items":       map[string]any{"type": "string"},
						},
						"produces": map[string]any{
							"type":        "string",
							"description": "Declares the type of artifact this member produces. Use 'code' for source code files, 'data' for structured data/JSON/CSV, 'document' for prose documents/reports. When set, the framework automatically appends a QA reviewer step after all workers finish to validate output correctness. Omit if no verification is needed.",
						},
					},
					"required": []string{"role", "task"},
				},
			},
		},
		"required": []string{"strategy", "members"},
	}
}

func (t *TeamTool) SetContext(channel, chatID string) {
	t.originChannel = channel
	t.originChatID = chatID
}

// reviewerTaskTemplates maps a `produces` artifact type to the task prompt
// that the auto-injected QA reviewer will receive.
var reviewerTaskTemplates = map[string]string{
	"code":     "You are a code quality reviewer. Read all code files in the workspace that were just written by your predecessors. Check for: syntax errors, incorrect or missing imports, broken logic, type mismatches, and any issues that would cause compilation or runtime failures. List every issue found with the filename and line number if possible. If everything looks correct, respond with 'REVIEW PASSED'.",
	"data":     "You are a data validation reviewer. Read all output data files (JSON, CSV, YAML, etc.) in the workspace. Check for: invalid format, missing required fields, schema inconsistencies, and malformed values. List every issue found. If everything is valid, respond with 'REVIEW PASSED'.",
	"document": "You are a document quality reviewer. Read all output documents in the workspace. Check for: logical inconsistencies, incomplete sections, factual contradictions, and poor structure. List every issue found. If the documents are complete and correct, respond with 'REVIEW PASSED'.",
}

// maybeRunAutoReviewer inspects TeamMembers for `produces` declarations.
// If any member produced a verifiable artifact type, it runs an automatic
// QA reviewer agent after all workers have completed.
func (t *TeamTool) maybeRunAutoReviewer(
	ctx context.Context,
	members []TeamMember,
	baseConfig ToolLoopConfig,
	workerSummary string,
) string {
	// Collect unique produces types from all members
	producedTypes := make(map[string]bool)
	for _, m := range members {
		if m.Produces != "" {
			producedTypes[m.Produces] = true
		}
	}
	if len(producedTypes) == 0 {
		return "" // No verifiable artifacts declared, skip review
	}

	// Build reviewer task: combine templates for all declared artifact types
	var taskParts []string
	for artifactType := range producedTypes {
		if tmpl, ok := reviewerTaskTemplates[artifactType]; ok {
			taskParts = append(taskParts, tmpl)
		}
	}
	if len(taskParts) == 0 {
		return "" // Unknown produces types, skip
	}

	sm := t.manager
	sm.mu.RLock()
	teamConfig := sm.teamConfig
	sm.mu.RUnlock()

	if teamConfig.DisableAutoReviewer {
		return ""
	}

	reviewerTask := strings.Join(taskParts, "\n\n") +
		"\n\nContext from the workers that produced these artifacts:\n" + workerSummary

	reviewerMessages := []providers.Message{
		{Role: "user", Content: reviewerTask},
	}

	// Use a dedicated reviewer model if configured — typically a cheaper/faster model
	// is sufficient for QA review, saving tokens compared to the main worker model.
	reviewerConfig := baseConfig
	if teamConfig.ReviewerModel != "" && sm.IsModelAllowed(teamConfig.ReviewerModel) {
		reviewerConfig.Model = teamConfig.ReviewerModel
	}

	cnf, err := t.cfg.GetModelConfig(reviewerConfig.Model)

	if err == nil {
		provider, model, err := providers.CreateProviderFromConfig(cnf)

		if err == nil {
			reviewerConfig.Model = model
			reviewerConfig.Provider = provider
		}
	}

	providerName := "unknown"
	if reviewerConfig.Provider != nil {
		providerName = reviewerConfig.Provider.GetDefaultModel()
	}

	logger.InfoCF("team", fmt.Sprintf("reviewer use provider: [%s] and model: [%s]", providerName, reviewerConfig.Model), map[string]any{
		"model": teamConfig.ReviewerModel,
	})

	loopContent, _, err := t.spawnWorker(ctx, reviewerConfig, reviewerMessages, nil)
	if err != nil {
		return fmt.Sprintf("[Auto-Reviewer] Failed to run: %v", err)
	}
	return "[Auto-Reviewer Result]\n" + loopContent
}

func (t *TeamTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	strategy, ok := args["strategy"].(string)
	if !ok {
		return ErrorResult("strategy is required")
	}

	if t.manager == nil {
		return ErrorResult("Subagent manager not configured").WithError(fmt.Errorf("manager is nil"))
	}

	sm := t.manager
	sm.mu.RLock()
	teamConfig := sm.teamConfig
	sm.mu.RUnlock()

	// 1. Validate Strategy
	validStrategy := false
	if len(teamConfig.AllowedStrategies) > 0 {
		for _, s := range teamConfig.AllowedStrategies {
			if strategy == s {
				validStrategy = true
				break
			}
		}
	} else {
		// Default allowed strategies if not configured
		if strategy == "sequential" || strategy == "parallel" || strategy == "dag" || strategy == "evaluator_optimizer" {
			validStrategy = true
		}
	}

	if !validStrategy {
		return ErrorResult(fmt.Sprintf("strategy '%s' is not allowed by configuration", strategy))
	}

	membersRaw, ok := args["members"].([]any)
	if !ok || len(membersRaw) == 0 {
		return ErrorResult("members map array is required and must not be empty")
	}

	// 2. Validate Max Members
	if teamConfig.MaxMembers > 0 && len(membersRaw) > teamConfig.MaxMembers {
		return ErrorResult(fmt.Sprintf("Team exceeds maximum allowed members (%d). You requested %d members.", teamConfig.MaxMembers, len(membersRaw)))
	}

	maxTokensFloat, ok := args["max_team_tokens"].(float64)

	// Enforce hard budget from config as the ceiling.
	effectiveMaxTokens := int64(0)
	if teamConfig.MaxTeamTokens > 0 {
		effectiveMaxTokens = int64(teamConfig.MaxTeamTokens)
	}

	if ok && maxTokensFloat > 0 {
		requestedTokens := int64(maxTokensFloat)
		if effectiveMaxTokens > 0 && requestedTokens > effectiveMaxTokens {
			// LLM requested more than config allows: clamp to the hard ceiling.
			// effectiveMaxTokens already holds the correct ceiling, no change needed.
		} else if effectiveMaxTokens == 0 || requestedTokens < effectiveMaxTokens {
			// LLM asked for less, or there is no hard limit: honour the requested budget.
			effectiveMaxTokens = requestedTokens
		}
	}

	var budget *atomic.Int64
	if effectiveMaxTokens > 0 {
		budget = &atomic.Int64{}
		budget.Store(effectiveMaxTokens)
	}

	var members []TeamMember
	for i, mRaw := range membersRaw {
		mMap, ok := mRaw.(map[string]any)
		if !ok {
			return ErrorResult(fmt.Sprintf("member at index %d is invalid", i))
		}

		id, iOk := mMap["id"].(string)
		role, rOk := mMap["role"].(string)
		task, tOk := mMap["task"].(string)

		if !rOk || !tOk || strings.TrimSpace(role) == "" || strings.TrimSpace(task) == "" {
			return ErrorResult(fmt.Sprintf("member at index %d is missing required 'role' or 'task'", i))
		}

		// ID is highly recommended, generate one if missing for backwards compatibility
		if !iOk || strings.TrimSpace(id) == "" {
			id = fmt.Sprintf("member_%d", i)
		}

		modelStr, _ := mMap["model"].(string)
		modelStr = strings.TrimSpace(modelStr)

		var dependsOn []string
		if depRaw, dOk := mMap["depends_on"].([]any); dOk {
			for _, d := range depRaw {
				if dStr, dsOk := d.(string); dsOk {
					dependsOn = append(dependsOn, dStr)
				}
			}
		}

		producesStr, _ := mMap["produces"].(string)
		producesStr = strings.TrimSpace(producesStr)

		members = append(members, TeamMember{
			ID:        id,
			Role:      role,
			Task:      task,
			Model:     modelStr,
			DependsOn: dependsOn,
			Produces:  producesStr,
		})
	}

	// Base struct setup
	baseConfig := t.manager.BuildBaseWorkerConfig(ctx)
	if budget != nil {
		baseConfig.RemainingTokenBudget = budget
	}

	// Create a cancellable context for team bounding.
	// cancel() is always deferred so any spawned goroutines are cleaned up on return.
	timeoutDur := 15 * time.Minute
	if teamConfig.MaxTimeoutMinutes > 0 {
		timeoutDur = time.Duration(teamConfig.MaxTimeoutMinutes) * time.Minute
	}
	teamCtx, cancel := context.WithTimeout(ctx, timeoutDur)
	defer cancel()

	// If strategy is parallel or dag, we must upgrade the file tools to be concurrent-safe (locking)
	if strategy == "parallel" || strategy == "dag" {
		baseConfig.Tools = upgradeRegistryForConcurrency(baseConfig.Tools)
	}

	// Resolve max context runes for dependency injection into downstream prompts.
	contextLimit := 8000 // default
	if teamConfig.MaxContextRunes > 0 {
		contextLimit = teamConfig.MaxContextRunes
	}

	switch strategy {
	case "sequential":
		result := t.executeSequential(teamCtx, baseConfig, members, contextLimit)
		if reviewNote := t.maybeRunAutoReviewer(teamCtx, members, baseConfig, result.ForLLM); reviewNote != "" {
			result.ForLLM += "\n\n" + reviewNote
			result.ForUser += "\n\n" + reviewNote
		}
		return result
	case "dag":
		result := t.executeDAG(teamCtx, cancel, baseConfig, members, contextLimit)
		if reviewNote := t.maybeRunAutoReviewer(teamCtx, members, baseConfig, result.ForLLM); reviewNote != "" {
			result.ForLLM += "\n\n" + reviewNote
			result.ForUser += "\n\n" + reviewNote
		}
		return result
	case "evaluator_optimizer":
		return t.executeEvaluatorOptimizer(teamCtx, baseConfig, members, contextLimit)
	}
	// parallel
	result := t.executeParallel(teamCtx, baseConfig, members)
	if reviewNote := t.maybeRunAutoReviewer(teamCtx, members, baseConfig, result.ForLLM); reviewNote != "" {
		result.ForLLM += "\n\n" + reviewNote
		result.ForUser += "\n\n" + reviewNote
	}
	return result
}

// upgradeRegistryForConcurrency takes an existing ToolRegistry, clones it,
// and upgrades any tools that implement ConcurrencyUpgradeable to their locking counterparts.
func upgradeRegistryForConcurrency(original *ToolRegistry) *ToolRegistry {
	if original == nil {
		return nil
	}

	upgraded := NewToolRegistry()
	for _, name := range original.ListTools() {
		tool, ok := original.Get(name)
		if !ok {
			continue
		}

		if upgradeable, isUpgradeable := tool.(ConcurrencyUpgradeable); isUpgradeable {
			upgraded.Register(upgradeable.UpgradeToConcurrent())
		} else {
			upgraded.Register(tool)
		}
	}
	return upgraded
}

// spawnWorker executes a single team member's turn, routing through SubTurnSpawner when available.
// Returns (content, messages, error). The messages slice is non-nil only for stateful workers
// (evaluator_optimizer) and can be passed as InitialMessages for the next iteration.
func (t *TeamTool) spawnWorker(ctx context.Context, cfg ToolLoopConfig, messages []providers.Message, budget *atomic.Int64) (string, []providers.Message, error) {
	if t.spawner == nil {
		// Fallback: direct RunToolLoop (no turnState integration)
		res, err := RunToolLoop(ctx, cfg, messages, t.originChannel, t.originChatID)
		if err != nil {
			return "", nil, err
		}
		return res.Content, res.Messages, nil
	}

	// Convert ToolLoopConfig + messages into SubTurnConfig for SubTurnSpawner.
	var toolSlice []Tool
	if cfg.Tools != nil {
		for _, name := range cfg.Tools.ListTools() {
			if tool, ok := cfg.Tools.Get(name); ok {
				toolSlice = append(toolSlice, tool)
			}
		}
	}

	// Extract system prompt and non-system messages
	var actualSystemPrompt string
	var initialMessages []providers.Message
	for _, msg := range messages {
		if msg.Role == "system" {
			actualSystemPrompt = msg.Content
		} else {
			initialMessages = append(initialMessages, msg)
		}
	}

	maxTokens, temperature := getLLMOptionsFromConfig(cfg)

	subCfg := SubTurnConfig{
		Model:              cfg.Model,
		Provider:           cfg.Provider,
		Tools:              toolSlice,
		ActualSystemPrompt: actualSystemPrompt,
		InitialMessages:    initialMessages,
		MaxTokens:          maxTokens,
		Temperature:        temperature,
		Async:              false,
		InitialTokenBudget: budget,
	}

	res, err := t.spawner.SpawnSubTurn(ctx, subCfg)
	if err != nil {
		return "", nil, err
	}
	return res.ForLLM, res.Messages, nil
}

// spawnWorkerEmptyTools is like spawnWorker but forces an empty tool registry on the sub-turn.
// Used for the evaluator in evaluator_optimizer to prevent side effects.
func (t *TeamTool) spawnWorkerEmptyTools(ctx context.Context, cfg ToolLoopConfig, messages []providers.Message) (string, error) {
	if t.spawner == nil {
		// Fallback: direct RunToolLoop with empty registry
		emptyConfig := cfg
		emptyConfig.Tools = NewToolRegistry()
		res, err := RunToolLoop(ctx, emptyConfig, messages, t.originChannel, t.originChatID)
		if err != nil {
			return "", err
		}
		return res.Content, nil
	}

	var actualSystemPrompt string
	var initialMessages []providers.Message
	for _, msg := range messages {
		if msg.Role == "system" {
			actualSystemPrompt = msg.Content
		} else {
			initialMessages = append(initialMessages, msg)
		}
	}

	maxTokens, temperature := getLLMOptionsFromConfig(cfg)

	subCfg := SubTurnConfig{
		Model:              cfg.Model,
		Provider:           cfg.Provider,
		EmptyTools:         true,
		ActualSystemPrompt: actualSystemPrompt,
		InitialMessages:    initialMessages,
		MaxTokens:          maxTokens,
		Temperature:        temperature,
		Async:              false,
	}

	res, err := t.spawner.SpawnSubTurn(ctx, subCfg)
	if err != nil {
		return "", err
	}
	return res.ForLLM, nil
}

// getLLMOptionsFromConfig extracts MaxTokens and Temperature from a ToolLoopConfig's LLMOptions map.
func getLLMOptionsFromConfig(cfg ToolLoopConfig) (int, float64) {
	var maxTokens int
	var temperature float64
	if cfg.LLMOptions != nil {
		if v, ok := cfg.LLMOptions["max_tokens"].(int); ok {
			maxTokens = v
		}
		if v, ok := cfg.LLMOptions["temperature"].(float64); ok {
			temperature = v
		}
	}
	return maxTokens, temperature
}


// potentially overriding the model based on the member's definition.
func (t *TeamTool) buildWorkerConfig(baseConfig ToolLoopConfig, registry *ToolRegistry, m TeamMember) (ToolLoopConfig, error) {
	cfg := baseConfig
	cfg.Tools = registry
	// Heterogeneous Agents: Override model if this team member requested a specific one
	if m.Model != "" {
		if !t.manager.IsModelAllowed(m.Model) {
			return cfg, fmt.Errorf("requested model '%s' is not in the allowed fallback candidates list for this agent workspace", m.Model)
		}
		// Resolve model name from model_list if it's an alias
		//resolvedModel := m.Model
		//if t.cfg != nil {
		//	for _, mc := range t.cfg.ModelList {
		//		if mc.ModelName == m.Model && mc.Model != "" {
		//			resolvedModel = mc.Model
		//			break
		//		}
		//	}
		//}

		cnf, err := t.cfg.GetModelConfig(m.Model)

		if err != nil {
			return cfg, err
		}

		provider, model, err := providers.CreateProviderFromConfig(cnf)

		if err != nil {
			return ToolLoopConfig{}, err
		}

		cfg.Model = model
		cfg.Provider = provider
	}

	providerName := "unknown"
	if cfg.Provider != nil {
		providerName = cfg.Provider.GetDefaultModel()
	}

	logger.InfoCF("team", fmt.Sprintf("[%s] use provider: [%s] and model: [%s]", m.Role, providerName, cfg.Model), map[string]any{
		"member_index": m.ID,
		"model":        m.Model,
	})

	return cfg, nil
}

func (t *TeamTool) executeSequential(ctx context.Context, baseConfig ToolLoopConfig, members []TeamMember, contextLimit int) *ToolResult {
	var finalOutput strings.Builder
	finalOutput.WriteString("Team Execution Summary (Sequential):\n\n")

	var previousResult string

	for i, m := range members {
		// If there is a previous result, we append it to the task so the new agent sees it.
		actualTask := m.Task
		if i > 0 && previousResult != "" {
			actualTask = fmt.Sprintf("%s\n\n--- Context from previous phase ---\n%s", m.Task, truncateContextN(previousResult, contextLimit))
		}

		messages := []providers.Message{
			{Role: "system", Content: m.Role},
			{Role: "user", Content: actualTask},
		}

		workerConfig, err := t.buildWorkerConfig(baseConfig, baseConfig.Tools, m)
		if err != nil {
			errStr := fmt.Sprintf("Phase %d (Role: %s) configuration failed: %v", i+1, m.Role, err)
			finalOutput.WriteString(errStr + "\n")
			return ErrorResult(errStr).WithError(err)
		}

		content, _, err := t.spawnWorker(ctx, workerConfig, messages, baseConfig.RemainingTokenBudget)
		if err != nil {
			errStr := fmt.Sprintf("Phase %d (Role: %s) failed: %v", i+1, m.Role, err)
			finalOutput.WriteString(errStr + "\n")
			return ErrorResult(errStr).WithError(err) // Fail fast
		}

		previousResult = content

		finalOutput.WriteString(fmt.Sprintf("### Phase %d completed by Role: [%s]\n%s\n\n", i+1, m.Role, previousResult))
	}

	return &ToolResult{
		ForLLM:  finalOutput.String(),
		ForUser: buildUserSummary("Sequential", members, nil),
	}
}

func (t *TeamTool) executeParallel(ctx context.Context, baseConfig ToolLoopConfig, members []TeamMember) *ToolResult {
	var wg sync.WaitGroup
	type workResult struct {
		index int
		role  string
		res   string
		err   error
	}

	resultsChan := make(chan workResult, len(members))

	for i, m := range members {
		wg.Add(1)
		go func(index int, member TeamMember) {
			defer wg.Done()

			logger.InfoCF("team", fmt.Sprintf("[%s] Parallel worker starting", member.Role), map[string]any{
				"member_index": index,
				"model":        member.Model,
			})

			messages := []providers.Message{
				{Role: "system", Content: member.Role},
				{Role: "user", Content: member.Task},
			}

			workerConfig, err := t.buildWorkerConfig(baseConfig, baseConfig.Tools, member)
			if err != nil {
				resultsChan <- workResult{index: index, role: member.Role, err: err}
				return
			}

			content, _, err := t.spawnWorker(ctx, workerConfig, messages, baseConfig.RemainingTokenBudget)

			if err != nil {
				resultsChan <- workResult{index: index, role: member.Role, err: err}
				return
			}
			resultsChan <- workResult{index: index, role: member.Role, res: content}
			logger.InfoCF("team", fmt.Sprintf("[%s] Parallel worker finished", member.Role), map[string]any{
				"member_index": index,
			})
		}(i, m)
	}

	// Wait for all goroutines to finish
	wg.Wait()
	close(resultsChan)

	// Pre-allocate to maintain order since channels don't guarantee arrival order
	orderedResults := make([]workResult, len(members))
	for res := range resultsChan {
		orderedResults[res.index] = res
	}

	var successOutput strings.Builder
	var failureOutput strings.Builder
	successCount, failureCount := 0, 0

	successOutput.WriteString("Team Execution Summary (Parallel):\n\n")

	for _, res := range orderedResults {
		if res.err != nil {
			failureCount++
			failureOutput.WriteString(fmt.Sprintf("### Worker [%s] FAILED:\n%v\n\n", res.role, res.err))
		} else {
			successCount++
			successOutput.WriteString(fmt.Sprintf("### Worker [%s] Output:\n%s\n\n", res.role, res.res))
		}
	}

	if failureCount == 0 {
		// All workers succeeded
		return &ToolResult{
			ForLLM:  successOutput.String(),
			ForUser: buildUserSummary("Parallel", members, nil),
		}
	}

	// Partial failure: preserve successful results and append failure summary.
	// This lets the coordinator decide how to handle the partial outcome.
	fullOutput := successOutput.String()
	if failureCount > 0 {
		fullOutput += "---\n## ⚠️ Partial Failures\n\n" + failureOutput.String() +
			fmt.Sprintf("\n%d/%d workers succeeded. %d worker(s) failed. The successful results above may still be usable.",
				successCount, len(members), failureCount)
	}

	return &ToolResult{
		ForLLM:  fullOutput,
		ForUser: fmt.Sprintf("⚠️ Parallel execution: %d/%d workers succeeded. %d failed.", successCount, len(members), failureCount),
		IsError: failureCount == len(members),
	}
}

func (t *TeamTool) executeEvaluatorOptimizer(ctx context.Context, baseConfig ToolLoopConfig, members []TeamMember, contextLimit int) *ToolResult {
	if len(members) != 2 {
		return ErrorResult("The evaluator_optimizer strategy requires exactly two members: [0] Worker, [1] Evaluator.")
	}

	worker := members[0]
	evaluator := members[1]

	var finalOutput strings.Builder
	finalOutput.WriteString("Team Execution Summary (Evaluator-Optimizer):\n\n")

	// 1. Initialize the stateful memory for the worker
	workerMessages := []providers.Message{
		{Role: "system", Content: worker.Role},
		{Role: "user", Content: worker.Task},
	}

	sm := t.manager
	sm.mu.RLock()
	teamConfig := sm.teamConfig
	sm.mu.RUnlock()

	maxLoops := 5
	if teamConfig.MaxEvaluatorLoops > 0 {
		maxLoops = teamConfig.MaxEvaluatorLoops
	}

	// Pre-compute both configs once — they don't change between loop iterations.
	workerConfig, err := t.buildWorkerConfig(baseConfig, baseConfig.Tools, worker)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Worker configuration failed: %v", err)).WithError(err)
	}
	evalConfig, err := t.buildWorkerConfig(baseConfig, NewToolRegistry(), evaluator)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Evaluator configuration failed: %v", err)).WithError(err)
	}

	logger.InfoCF("team", "Evaluator-Optimizer starting", map[string]any{
		"worker":    worker.Role,
		"evaluator": evaluator.Role,
		"max_loops": maxLoops,
	})

	for attempt := 1; attempt <= maxLoops; attempt++ {
		finalOutput.WriteString(fmt.Sprintf("## Attempt %d\n", attempt))
		logger.InfoCF("team", fmt.Sprintf("Evaluator-Optimizer attempt %d/%d", attempt, maxLoops), map[string]any{})

		// 2. Trigger Worker (resumes from its exact previous state!)
		workerContent, workerMsgs, err := t.spawnWorker(ctx, workerConfig, workerMessages, baseConfig.RemainingTokenBudget)
		if err != nil {
			errStr := fmt.Sprintf("Worker failed on attempt %d: %v", attempt, err)
			finalOutput.WriteString(errStr + "\n")
			return ErrorResult(errStr).WithError(err)
		}

		// Save the worker's cognitive state so it remembers its thought process for the next loop
		if workerMsgs != nil {
			workerMessages = workerMsgs
		}

		finalOutput.WriteString(fmt.Sprintf("### Worker Output:\n%s\n\n", workerContent))

		// 3. Trigger Evaluator (Ephemeral, stateless evaluation)
		// The evaluator only needs to reason about text — give it no tools to avoid
		// unnecessary tool calls, wasted tokens, and potential side effects.
		evalContext := fmt.Sprintf("%s\n\n--- Worker's Output to Evaluate ---\n%s\n\nIf the output is completely correct and fulfills the task, you MUST reply starting with strictly '[PASS]'. Otherwise, explain the issues in detail.", evaluator.Task, truncateContextN(workerContent, contextLimit))

		evalMessages := []providers.Message{
			{Role: "system", Content: evaluator.Role},
			{Role: "user", Content: evalContext},
		}

		evalContent, err := t.spawnWorkerEmptyTools(ctx, evalConfig, evalMessages)
		if err != nil {
			errStr := fmt.Sprintf("Evaluator failed on attempt %d: %v", attempt, err)
			finalOutput.WriteString(errStr + "\n")
			return ErrorResult(errStr).WithError(err)
		}

		finalOutput.WriteString(fmt.Sprintf("### Evaluator Feedback:\n%s\n\n", evalContent))

		// 4. Check for PASS condition
		if strings.HasPrefix(strings.TrimSpace(evalContent), "[PASS]") {
			finalOutput.WriteString("✅ Evaluation Passed! Loop finished successfully.\n")
			logger.InfoCF("team", "Evaluator-Optimizer passed", map[string]any{"attempt": attempt})
			return &ToolResult{
				ForLLM:  finalOutput.String(),
				ForUser: fmt.Sprintf("✅ Evaluator-Optimizer passed on attempt %d/%d (worker: %s).", attempt, maxLoops, worker.Role),
			}
		}

		logger.InfoCF("team", "Evaluator-Optimizer did not pass, retrying", map[string]any{"attempt": attempt, "max_loops": maxLoops})

		// 5. If not passed, and not the last attempt, inject feedback into Worker's stateful memory
		if attempt < maxLoops {
			injection := fmt.Sprintf("The evaluator rejected your previous attempt. Please fix the issues based on this feedback:\n\n%s", evalContent)
			workerMessages = append(workerMessages, providers.Message{
				Role:    "user",
				Content: injection,
			})
		}
	}

	finalOutput.WriteString("❌ Maximum evaluation loops reached without a [PASS]. Returning current state.\n")
	logger.WarnCF("team", "Evaluator-Optimizer exhausted max loops", map[string]any{"max_loops": maxLoops})
	return &ToolResult{
		ForLLM:  finalOutput.String(),
		ForUser: fmt.Sprintf("❌ Evaluator-Optimizer exhausted %d attempts without a [PASS].", maxLoops),
	}
}

func (t *TeamTool) executeDAG(ctx context.Context, cancel context.CancelFunc, baseConfig ToolLoopConfig, members []TeamMember, contextLimit int) *ToolResult {
	logger.InfoCF("team", "DAG execution starting", map[string]any{"member_count": len(members)})
	// 1. Build and VALIDATE dependency graph
	memberMap := make(map[string]TeamMember)
	inDegree := make(map[string]int)
	graph := make(map[string][]string) // node -> nodes that depend on it

	// Register all valid members first
	for _, m := range members {
		memberMap[m.ID] = m
		inDegree[m.ID] = 0
		graph[m.ID] = []string{}
	}

	// Build edges and check for ghost nodes
	for _, m := range members {
		for _, dep := range m.DependsOn {
			if _, exists := memberMap[dep]; !exists {
				return ErrorResult(fmt.Sprintf("DAG Validation Error: Member [%s] depends on undefined member [%s]", m.ID, dep))
			}
			graph[dep] = append(graph[dep], m.ID)
			inDegree[m.ID]++
		}
	}

	// 1.5. Cycle Detection using Kahn's Algorithm
	var kahnQueue []string
	kahnInDegree := make(map[string]int)
	for k, v := range inDegree {
		kahnInDegree[k] = v
		if v == 0 {
			kahnQueue = append(kahnQueue, k)
		}
	}

	processedCount := 0
	for len(kahnQueue) > 0 {
		curr := kahnQueue[0]
		kahnQueue = kahnQueue[1:]
		processedCount++

		for _, dependent := range graph[curr] {
			kahnInDegree[dependent]--
			if kahnInDegree[dependent] == 0 {
				kahnQueue = append(kahnQueue, dependent)
			}
		}
	}

	if processedCount != len(members) {
		return ErrorResult("DAG Validation Error: Circular dependency (cycle) detected in the team layout. Please fix your 'depends_on' definitions.")
	}

	// 2. Channels for coordination
	type nodeResult struct {
		id  string
		res string
		err error
	}
	readyChan := make(chan string, len(members))
	resultChan := make(chan nodeResult, len(members))

	// Channels specifically for passing context from dependencies to dependants
	contextMap := make(map[string]*strings.Builder)
	var contextMu sync.Mutex

	// 3. Initialize queue with nodes having 0 in-degree
	nodesToProcess := len(members)
	for id, deg := range inDegree {
		if deg == 0 {
			readyChan <- id
		}
	}

	var wg sync.WaitGroup
	var masterErr error
	var masterErrMu sync.Mutex

	// Shared results store for the final output
	finalResults := make(map[string]string)
	var finalResultsMu sync.Mutex

	// 4. DAG Execution Loop
	for i := 0; i < nodesToProcess; i++ {
		select {
		case <-ctx.Done():
			return ErrorResult("DAG execution timed out or cancelled")

		case memberID := <-readyChan:
			wg.Add(1)
			go func(id string) {
				defer wg.Done()

				m := memberMap[id]

				// Construct the task with context from all dependencies
				actualTask := m.Task
				contextMu.Lock()
				b := contextMap[id]
				depsContext := ""
				if b != nil {
					depsContext = b.String()
				}
				contextMu.Unlock()

				if depsContext != "" {
					actualTask = fmt.Sprintf("%s\n\n--- Context from dependencies ---\n%s", m.Task, truncateContextN(depsContext, contextLimit))
				}

				messages := []providers.Message{
					{Role: "system", Content: m.Role},
					{Role: "user", Content: actualTask},
				}

				workerConfig, err := t.buildWorkerConfig(baseConfig, baseConfig.Tools, m)
				if err != nil {
					masterErrMu.Lock()
					if masterErr == nil {
						masterErr = err
					}
					masterErrMu.Unlock()
					resultChan <- nodeResult{id: id, err: err}
					return
				}

				content, _, err := t.spawnWorker(ctx, workerConfig, messages, baseConfig.RemainingTokenBudget)

				if err != nil {
					masterErrMu.Lock()
					if masterErr == nil {
						masterErr = fmt.Errorf("worker [%s] failed: %v", m.ID, err)
					}
					masterErrMu.Unlock()
					resultChan <- nodeResult{id: id, err: err}
					return
				}

				// Store result for final output
				finalResultsMu.Lock()
				finalResults[id] = content
				finalResultsMu.Unlock()

				// Pass result to dependents
				resultChan <- nodeResult{id: id, res: content}
			}(memberID)

		case res := <-resultChan:
			if res.err != nil {
				// Fast fail on first error.
				// Cancel the team context first so that all in-flight goroutines
				// receive the cancellation signal and terminate cleanly.
				cancel()
				wg.Wait()
				return ErrorResult(res.err.Error())
			}

			// Update dependents
			for _, dependentID := range graph[res.id] {
				contextMu.Lock()
				b := contextMap[dependentID]
				if b == nil {
					b = &strings.Builder{}
				}
				b.WriteString(fmt.Sprintf("--- Result from [%s] ---\n%s\n\n", res.id, res.res))
				contextMap[dependentID] = b
				contextMu.Unlock()

				inDegree[dependentID]--
				if inDegree[dependentID] == 0 {
					readyChan <- dependentID
				}
			}
		}
	}

	// Wait for any remaining goroutines (though the select loop handles the exact count)
	wg.Wait()

	if masterErr != nil {
		return ErrorResult(masterErr.Error())
	}

	// 5. Format final output
	var finalOutput strings.Builder
	finalOutput.WriteString("Team Execution Summary (DAG):\n\n")

	// Preserve original member order for final output readability
	for _, m := range members {
		if res, ok := finalResults[m.ID]; ok {
			finalOutput.WriteString(fmt.Sprintf("### Worker [%s] (Role: %s) Output:\n%s\n\n", m.ID, m.Role, res))
		}
	}

	return &ToolResult{
		ForLLM:  finalOutput.String(),
		ForUser: buildUserSummary("DAG", members, nil),
	}
}

// truncateContextN limits the number of runes in ctx to maxRunes.
// It prevents Context Window Explosion (Token Bombs) when passing upstream
// worker results into downstream agent prompts.
func truncateContextN(ctx string, maxRunes int) string {
	runes := []rune(ctx)
	if len(runes) > maxRunes {
		return string(runes[:maxRunes]) + "\n...[Context truncated due to length]..."
	}
	return ctx
}

// truncateContext is the default wrapper using 8000 runes (≈6000 words).
// Call truncateContextN directly when a configurable limit is needed.
func truncateContext(ctx string) string {
	return truncateContextN(ctx, 8000)
}

// buildUserSummary produces a concise human-readable summary for the ForUser field,
// listing each member's role. errors (if any) are appended as a separate section.
func buildUserSummary(strategy string, members []TeamMember, errors []string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Team (%s) completed — %d member(s):\n", strategy, len(members)))
	for i, m := range members {
		sb.WriteString(fmt.Sprintf("  [%d] %s", i+1, m.Role))
		if m.Model != "" {
			sb.WriteString(fmt.Sprintf(" (model: %s)", m.Model))
		}
		sb.WriteString("\n")
	}
	if len(errors) > 0 {
		sb.WriteString("\n⚠️ Failures:\n")
		for _, e := range errors {
			sb.WriteString("  • " + e + "\n")
		}
	}
	return strings.TrimRight(sb.String(), "\n")
}
