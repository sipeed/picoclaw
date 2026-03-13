package agent

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/constants"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/orch"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/tools"
	"github.com/sipeed/picoclaw/pkg/utils"
)

// iterationHooks contains optional callbacks that extend the core LLM
// iteration loop.  Each hook is nil when the corresponding fork feature
// is inactive, keeping the core loop close to upstream's structure.
type iterationHooks struct {
	// OnIterationStart is called at the top of each iteration.
	// Returns an optional user-role message to inject (e.g. user intervention).
	OnIterationStart func(iteration int) (interventionMsg string)

	// FilterTools is called after building provider tool definitions,
	// before the LLM call.  Returns a (possibly filtered) slice.
	FilterTools func(defs []providers.ToolDefinition) []providers.ToolDefinition

	// SetupStreaming is called before each LLM call to set up streaming
	// preview.  Returns an onChunk callback and a cleanup function.
	// Both may be nil if streaming is not applicable.
	SetupStreaming func() (onChunk func(accumulated, reasoning string), cleanup func())

	// SelectModel overrides the model and candidates for this call.
	// Returns empty string to use defaults.
	SelectModel func() (model string, candidates []providers.FallbackCandidate)

	// OnPreLLMCall is called just before the LLM call (e.g. orch state reporting).
	OnPreLLMCall func()

	// OnNoToolCalls is called when the LLM returns no tool calls.
	// Returns an optional nudge message and whether to continue the loop.
	OnNoToolCalls func(content string, iteration int) (nudge string, continueLoop bool)

	// FilterToolCalls is called after normalizing tool calls, before execution.
	// Returns the filtered calls and an optional rejection message.
	// If all calls are filtered out, the loop continues with the rejection message.
	FilterToolCalls func(calls []providers.ToolCall) (filtered []providers.ToolCall, rejectionMsg string)

	// OnPreToolExec is called before each tool execution.
	// Returns an async callback (may be nil).
	OnPreToolExec func(ctx context.Context, tc providers.ToolCall) tools.AsyncCallback

	// OnToolExecDone is called after each tool execution with the result.
	OnToolExecDone func(tc providers.ToolCall, result *tools.ToolResult, duration time.Duration)

	// OnToolsProcessed is called after all tool calls in an iteration
	// have been logged and their results built.  Receives the tool call
	// list for status publishing and session-touch recording.
	OnToolsProcessed func(ctx context.Context, iteration int, toolCalls []providers.ToolCall)

	// InjectReminders is called at the end of each iteration to append
	// fork-specific reminder messages (task, plan, orch, subagent questions).
	InjectReminders func(iteration int, messages *[]providers.Message, lastBlocker string)

	// RefreshSystemPrompt is called at the end of each iteration to
	// rebuild the system prompt after tool execution may have changed state.
	RefreshSystemPrompt func(messages []providers.Message)
}

// buildHooks constructs the hook set based on the current agent state.
// All fork-specific logic is wired here; the core loop only calls hooks.
func (al *AgentLoop) buildHooks(
	agent *AgentInstance,
	opts processOptions,
	task *activeTask,
	planSnapshot string,
) iterationHooks {
	h := iterationHooks{}
	isBackground := opts.TaskID != ""

	// ── Task tracking ──
	if task != nil {
		h.OnIterationStart = func(iteration int) string {
			task.mu.Lock()
			task.Iteration = iteration
			task.mu.Unlock()

			select {
			case msg := <-task.interrupt:
				logger.InfoCF("agent", "User intervention injected",
					map[string]any{"agent_id": agent.ID, "iteration": iteration})
				return "[User Intervention] " + msg
			default:
				return ""
			}
		}

		h.OnToolExecDone = func(tc providers.ToolCall, result *tools.ToolResult, duration time.Duration) {
			updateToolLogResult(task, tc, result, duration)
		}
	}

	// ── Plan mode ──
	if planSnapshot != "" {
		preUnchecked := -1
		if planSnapshot == "executing" {
			preUnchecked = strings.Count(agent.ContextBuilder.ReadMemory(), "- [ ]")
		}
		planMarkNudged := false

		if isPlanPreExecution(planSnapshot) {
			h.FilterTools = func(defs []providers.ToolDefinition) []providers.ToolDefinition {
				return filterInterviewTools(defs)
			}

			h.FilterToolCalls = func(calls []providers.ToolCall) ([]providers.ToolCall, string) {
				allowed := calls[:0]
				var rejected []string
				for _, tc := range calls {
					if isToolAllowedDuringInterview(tc.Name, tc.Arguments) {
						allowed = append(allowed, tc)
					} else {
						rejected = append(rejected, tc.Name)
					}
				}
				if len(rejected) > 0 {
					logger.InfoCF("agent", "Interview mode: rejected tool calls",
						map[string]any{"agent_id": agent.ID, "rejected": rejected})
				}
				return allowed, interviewRejectMessage
			}
		}

		h.OnNoToolCalls = func(content string, iteration int) (string, bool) {
			if preUnchecked <= 0 || planMarkNudged || planSnapshot != "executing" {
				return "", false
			}
			curUnchecked := strings.Count(agent.ContextBuilder.ReadMemory(), "- [ ]")
			if curUnchecked <= 0 {
				return "", false
			}
			planMarkNudged = true

			var nudge string
			if curUnchecked == preUnchecked {
				nudge = fmt.Sprintf("[System] %d unchecked steps remain in MEMORY.md and "+
					"none were marked [x] during this session. "+
					"If you completed any steps, use edit_file to mark them [x] now. "+
					"If steps are still in progress, continue working on them.", curUnchecked)
			} else {
				nudge = fmt.Sprintf("[System] Progress recorded. %d unchecked steps remain. "+
					"Continue working on the next step.", curUnchecked)
			}
			logger.InfoCF("agent", "Nudging plan execution: continue plan steps",
				map[string]any{"agent_id": agent.ID, "iteration": iteration, "unchecked": curUnchecked})
			return nudge, true
		}

		// Plan model selection
		if isPlanPreExecution(planSnapshot) && agent.PlanModel != "" {
			h.SelectModel = func() (string, []providers.FallbackCandidate) {
				logger.InfoCF("agent", "Using plan model",
					map[string]any{"agent_id": agent.ID, "plan_model": agent.PlanModel})
				return agent.PlanModel, agent.PlanCandidates
			}
		}
	}

	// ── Streaming ──
	if !constants.IsInternalChannel(opts.Channel) {
		h.SetupStreaming = func() (func(string, string), func()) {
			return al.setupStreamingHook(opts, task)
		}
	}

	// ── Orchestration ──
	if al.orchReporter != orch.Noop {
		h.OnPreLLMCall = func() {
			al.reporter().ReportStateChange(opts.SessionKey, orch.AgentStateWaiting, "")
		}

		// Wrap OnPreToolExec to add orch state reporting
		h.OnPreToolExec = func(ctx context.Context, tc providers.ToolCall) tools.AsyncCallback {
			al.reporter().ReportStateChange(opts.SessionKey, orch.AgentStateToolCall, tc.Name)
			return al.buildAsyncCallback(opts, tc.Name)
		}
	} else {
		// Even without orch, we still need async callback
		h.OnPreToolExec = func(ctx context.Context, tc providers.ToolCall) tools.AsyncCallback {
			return al.buildAsyncCallback(opts, tc.Name)
		}
	}

	// ── Tool status + session touch ──
	if !constants.IsInternalChannel(opts.Channel) && task != nil {
		h.OnToolsProcessed = func(ctx context.Context, iteration int, toolCalls []providers.ToolCall) {
			al.publishToolStatus(ctx, agent, opts, task, iteration, isBackground, toolCalls)
			al.recordSessionTouches(agent, opts, toolCalls)
		}
	} else {
		// Session touch without status publishing
		h.OnToolsProcessed = func(ctx context.Context, iteration int, toolCalls []providers.ToolCall) {
			al.recordSessionTouches(agent, opts, toolCalls)
		}
	}

	// ── Reminder injection (task + plan + orch + subagent questions) ──
	h.InjectReminders = al.buildReminderInjector(agent, opts, task, planSnapshot)

	// ── System prompt refresh ──
	h.RefreshSystemPrompt = func(messages []providers.Message) {
		if touchDir := al.sessions.GetTouchDir(opts.SessionKey); touchDir != "" {
			agent.ContextBuilder.SetWorkDir(filepath.Join(agent.Workspace, touchDir))
		}
		if newPrompt := agent.ContextBuilder.BuildSystemPrompt(); len(messages) > 0 &&
			messages[0].Content != newPrompt {
			messages[0].Content = newPrompt
			al.lastSystemPrompt.Store(newPrompt)
			al.promptDirty.Store(false)
		}
	}

	return h
}

// ── Hook helper implementations ──

// setupStreamingHook creates a streaming display goroutine and returns
// the onChunk callback and cleanup function.
func (al *AgentLoop) setupStreamingHook(opts processOptions, task *activeTask) (func(string, string), func()) {
	type streamUpdate struct{ accumulated, reasoning string }

	streamCh := make(chan streamUpdate, 1)
	streamDone := make(chan struct{})

	ctx := context.Background() // outlive the caller's context for flush

	go func() {
		defer close(streamDone)
		for up := range streamCh {
			display := buildStreamingDisplay(up.accumulated, up.reasoning)
			outMsg := bus.OutboundMessage{
				Channel: opts.Channel,
				ChatID:  opts.ChatID,
				Content: display,
			}
			if opts.Background && opts.TaskID != "" {
				outMsg.IsTaskStatus = true
				outMsg.TaskID = opts.TaskID
			} else {
				outMsg.IsStatus = true
			}
			_ = al.bus.PublishOutbound(ctx, outMsg)
		}
	}()

	onChunk := func(accumulated, reasoning string) {
		if task != nil {
			task.streamedChunks = true
		}
		up := streamUpdate{accumulated, reasoning}
		select {
		case streamCh <- up:
		default:
			select {
			case <-streamCh:
			default:
			}
			select {
			case streamCh <- up:
			default:
			}
		}
	}

	cleanup := func() {
		close(streamCh)
		<-streamDone
	}

	return onChunk, cleanup
}

// buildAsyncCallback creates the async tool callback that publishes
// results as system inbound messages.
func (al *AgentLoop) buildAsyncCallback(opts processOptions, toolName string) tools.AsyncCallback {
	return func(_ context.Context, result *tools.ToolResult) {
		content := result.ForLLM
		if content == "" {
			content = result.ForUser
		}
		if content == "" {
			return
		}

		logger.InfoCF("agent", "Async tool completed, publishing to conductor",
			map[string]any{"tool": toolName, "content_len": len(content), "is_error": result.IsError})

		pubCtx, pubCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer pubCancel()

		_ = al.bus.PublishInbound(pubCtx, bus.InboundMessage{
			Channel:  "system",
			SenderID: fmt.Sprintf("async:%s", toolName),
			ChatID:   fmt.Sprintf("%s:%s", opts.Channel, opts.ChatID),
			Content:  fmt.Sprintf("Async tool '%s' completed.\n\nResult:\n%s", toolName, content),
		})
	}
}

// updateToolLogResult updates the task's tool log entry with execution result.
func updateToolLogResult(task *activeTask, tc providers.ToolCall, result *tools.ToolResult, duration time.Duration) {
	task.mu.Lock()
	defer task.mu.Unlock()

	// Walk backward to find the matching pending entry
	for i := len(task.toolLog) - 1; i >= 0; i-- {
		if task.toolLog[i].Result == "\u23F3" {
			if result.IsError || result.Err != nil {
				task.toolLog[i].Result = fmt.Sprintf("\u2717 %.1fs", duration.Seconds())
				if result.Err != nil {
					task.toolLog[i].ErrDetail = utils.Truncate(result.Err.Error(), 300)
				} else if result.ForLLM != "" {
					lines := strings.Split(strings.TrimSpace(result.ForLLM), "\n")
					start := len(lines) - 3
					if start < 0 {
						start = 0
					}
					task.toolLog[i].ErrDetail = utils.Truncate(
						strings.Join(lines[start:], "\n"), 300)
				}
				entry := task.toolLog[i]
				task.lastError = &entry
			} else {
				task.toolLog[i].Result = fmt.Sprintf("\u2713 %.1fs", duration.Seconds())
			}
			break
		}
	}
}

// publishToolStatus adds pending entries to the tool log and publishes
// a rich status update via the message bus.
func (al *AgentLoop) publishToolStatus(
	ctx context.Context,
	agent *AgentInstance,
	opts processOptions,
	task *activeTask,
	iteration int,
	isBackground bool,
	toolCalls []providers.ToolCall,
) {
	task.mu.Lock()
	for _, tc := range toolCalls {
		task.toolLog = append(task.toolLog, toolLogEntry{
			Name:     fmt.Sprintf("[%d] %s", iteration, tc.Name),
			ArgsSnip: buildArgsSnippet(tc.Name, tc.Arguments, agent.Workspace),
			Result:   "\u23F3",
		})
		if task.projectDir == "" && tc.Name == "exec" {
			task.projectDir = extractExecProjectDir(tc.Arguments)
		}
		switch tc.Name {
		case "read_file", "write_file", "edit_file", "append_file", "list_dir":
			if p, _ := tc.Arguments["path"].(string); p != "" {
				if rel := fileParentRelDir(p, agent.Workspace); rel != "" {
					if task.fileCommonDir == "" {
						task.fileCommonDir = rel
					} else {
						task.fileCommonDir = commonDirPrefix(task.fileCommonDir, rel)
					}
				}
			}
		}
	}
	task.mu.Unlock()

	statusContent := buildRichStatus(task, isBackground, agent.Workspace)
	if isBackground {
		_ = al.bus.PublishOutbound(ctx, bus.OutboundMessage{
			Channel:      opts.Channel,
			ChatID:       opts.ChatID,
			Content:      statusContent,
			IsTaskStatus: true,
			TaskID:       opts.TaskID,
		})
	} else {
		_ = al.bus.PublishOutbound(ctx, bus.OutboundMessage{
			Channel:  opts.Channel,
			ChatID:   opts.ChatID,
			Content:  statusContent,
			IsStatus: true,
		})
	}
}

// recordSessionTouches records session activity for heartbeat/plan coordination.
func (al *AgentLoop) recordSessionTouches(
	agent *AgentInstance,
	opts processOptions,
	toolCalls []providers.ToolCall,
) {
	for _, tc := range toolCalls {
		var detectedDir string
		if tc.Name == "exec" {
			detectedDir = extractExecProjectDir(tc.Arguments)
		}
		if detectedDir == "" {
			switch tc.Name {
			case "read_file", "write_file", "edit_file", "append_file", "list_dir":
				if p, _ := tc.Arguments["path"].(string); p != "" {
					detectedDir = fileParentRelDir(p, agent.Workspace)
				}
			}
		}
		if detectedDir != "" {
			meta := &TouchMeta{
				ProjectPath: agent.ContextBuilder.GetPlanWorkDir(),
				Purpose:     utils.Truncate(opts.UserMessage, 80),
				Branch:      agent.GetWorktreeBranch(opts.SessionKey),
			}
			if meta.ProjectPath == "" {
				meta.ProjectPath = agent.Workspace
			}
			al.sessions.Touch(opts.SessionKey, opts.Channel, opts.ChatID, detectedDir, meta)
		}
	}
}

// buildReminderInjector returns a function that injects all end-of-iteration
// reminder messages: task reminders, plan reminders, orch nudges, and
// pending subagent questions.
func (al *AgentLoop) buildReminderInjector(
	agent *AgentInstance,
	opts processOptions,
	task *activeTask,
	planSnapshot string,
) func(int, *[]providers.Message, string) {
	lastReminderIdx := -1

	return func(iteration int, messages *[]providers.Message, lastBlocker string) {
		// Task reminder
		if shouldInjectReminder(iteration, agent.TaskReminderInterval) && !opts.NoHistory {
			if lastReminderIdx >= 0 && lastReminderIdx < len(*messages) {
				*messages = append((*messages)[:lastReminderIdx], (*messages)[lastReminderIdx+1:]...)
			}
			reminderMsg := buildTaskReminder(opts.UserMessage, lastBlocker)
			*messages = append(*messages, reminderMsg)
			lastReminderIdx = len(*messages) - 1
			logger.DebugCF("agent", "Injected task reminder",
				map[string]any{"agent_id": agent.ID, "iteration": iteration, "has_blocker": lastBlocker != ""})
		}

		// Plan reminder
		if iteration > 1 && isPlanPreExecution(planSnapshot) {
			if reminder, ok := buildPlanReminder(planSnapshot); ok {
				*messages = append(*messages, reminder)
				logger.DebugCF("agent", "Injected plan reminder",
					map[string]any{"agent_id": agent.ID, "iteration": iteration, "plan_status": planSnapshot})
			}
		}

		// Orch nudge
		if planSnapshot == "executing" && agent.Subagents != nil && agent.Subagents.Enabled {
			if reminder, ok := buildOrchReminder(iteration); ok {
				*messages = append(*messages, reminder)
				logger.DebugCF("agent", "Injected orchestration nudge",
					map[string]any{"agent_id": agent.ID, "iteration": iteration})
			}
		}

		// Subagent questions/plan reviews
		if agent.SubagentMgr != nil {
			for _, q := range agent.SubagentMgr.PendingQuestions() {
				var content string
				switch q.Type {
				case "plan_review":
					content = fmt.Sprintf(
						"[Subagent %s submitted a plan for review]:\n%s\nRespond using the review_subagent_plan tool with task_id=%q.",
						q.TaskID, q.Content, q.TaskID,
					)
				default:
					content = fmt.Sprintf(
						"[Subagent %s asks]: %s\nRespond using the answer_subagent tool with task_id=%q.",
						q.TaskID, q.Content, q.TaskID,
					)
				}
				*messages = append(*messages, providers.Message{Role: "user", Content: content})
			}
		}

		// Tool log trim
		if task != nil {
			task.mu.Lock()
			if len(task.toolLog) > maxToolLogEntries {
				task.toolLog = task.toolLog[len(task.toolLog)-maxToolLogEntries:]
			}
			task.mu.Unlock()
		}
	}
}
