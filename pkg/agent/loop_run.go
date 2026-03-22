package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/constants"
	"github.com/sipeed/picoclaw/pkg/git"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/orch"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/session"
	"github.com/sipeed/picoclaw/pkg/tools"
	"github.com/sipeed/picoclaw/pkg/utils"
)

// runAgentLoop is the main message processing loop for a single agent session.
func (al *AgentLoop) runAgentLoop(ctx context.Context, agent *AgentInstance, opts processOptions) (string, error) {
	// -1. Acquire per-session lock to prevent concurrent access on the same session

	if !al.acquireSessionLock(ctx, opts.SessionKey) {
		return "", fmt.Errorf("context canceled while waiting for session lock")
	}

	defer al.releaseSessionLock(opts.SessionKey)

	// Report session lifecycle to canvas.

	al.reporter().ReportSpawn(opts.SessionKey, opts.Channel, opts.UserMessage)

	defer al.reporter().ReportGC(opts.SessionKey, "completed")

	// -0. Create cancelable child context and register active task

	taskCtx, taskCancel := context.WithCancel(ctx)

	defer taskCancel()

	task := &activeTask{
		Description: utils.Truncate(opts.UserMessage, 80),

		MaxIter: agent.MaxIterations,

		StartedAt: time.Now(),

		cancel: taskCancel,

		interrupt: make(chan string, 1),
	}

	// Guarantee heartbeat worktree cleanup on ALL exit paths (error, panic, normal).

	// Wait for spawned subagents first so they aren't killed mid-flight.

	// After auto-commit, attempt to merge the worktree branch into main.

	defer al.cleanupHeartbeatWorktree(agent, opts)

	// For background tasks (cron/heartbeat), generate a TaskID and send notification

	isBackgroundTask := opts.Background && al.state != nil

	if isBackgroundTask && opts.TaskID == "" {
		opts.TaskID = fmt.Sprintf("task-%s-%d", opts.SessionKey, time.Now().UnixMilli())

		// Determine notification channel: use opts.Channel if already a real channel,

		// otherwise resolve from last active channel

		notifyChannel := opts.Channel

		notifyChatID := opts.ChatID

		if constants.IsInternalChannel(notifyChannel) || notifyChannel == "" {
			if lastChannel := al.state.GetLastChannel(); lastChannel != "" {
				if idx := strings.Index(lastChannel, ":"); idx > 0 {
					notifyChannel = lastChannel[:idx]

					notifyChatID = lastChannel[idx+1:]
				}
			}
		}

		if notifyChannel != "" && notifyChatID != "" && !constants.IsInternalChannel(notifyChannel) {
			// Override opts channel/chatID for status updates

			opts.Channel = notifyChannel

			opts.ChatID = notifyChatID

			// Send initial task notification

			_ = al.bus.PublishOutbound(ctx, bus.OutboundMessage{
				Channel: notifyChannel,

				ChatID: notifyChatID,

				Content: fmt.Sprintf("\U0001F916 Background task started\n%s", task.Description),

				IsTaskStatus: true,

				TaskID: opts.TaskID,
			})
		}
	}

	// Shared variable for capturing LLM's final response. The defer below reads it

	// to include the response in the task completion message.

	var finalContent string

	// Use TaskID as key if available (for background tasks), else sessionKey

	taskKey := opts.SessionKey

	if opts.TaskID != "" {
		taskKey = opts.TaskID
	}

	al.activeTasks.Store(taskKey, task)

	defer al.publishTaskCompletion(task, &finalContent, opts, taskKey)

	// Replace ctx with the cancelable child context

	ctx = taskCtx

	// 0. Record last channel for heartbeat notifications (skip internal channels)

	if opts.Channel != "" && opts.ChatID != "" {
		// Don't record internal channels (cli, system, subagent)

		if !constants.IsInternalChannel(opts.Channel) {
			channelKey := fmt.Sprintf("%s:%s", opts.Channel, opts.ChatID)
			if err := al.RecordLastChannel(channelKey); err != nil {
				logger.WarnCF("agent", "Failed to record last channel", map[string]any{"error": err.Error()})
			}

			if err := al.RecordLastHeartbeatTarget(channelKey); err != nil {
				logger.WarnCF("agent", "Failed to record last heartbeat target", map[string]any{"error": err.Error()})
			}
		}
	}

	// 1. Update tool contexts

	al.updateToolContexts(agent, opts.Channel, opts.ChatID)

	// 1-bis. For background tasks that don't send a final response (e.g. heartbeat),

	// redirect the message tool to publish as IsTaskStatus so its output lands in

	// the same bubble as the task status instead of creating a separate message.

	if opts.Background && !opts.SendResponse && opts.TaskID != "" {
		al.redirectMessageToolForTask(agent, task, opts)
	}

	// 1a. Set session-specific working directory for bootstrap file lookup.

	// Prefer the tool-detected project directory (touch_dir) from the session tracker,

	// resolved as an absolute path under workspace. Fall back to worktree or workspace.

	if active := al.sessions.ListActive(); len(active) > 0 && active[0].SessionKey == opts.SessionKey &&

		active[0].TouchDir != "" {
		agent.ContextBuilder.SetWorkDir(filepath.Join(agent.Workspace, active[0].TouchDir))
	} else {
		agent.ContextBuilder.SetWorkDir(agent.EffectiveWorkspace(opts.SessionKey))
	}

	// 1b. Inject peer session awareness into system prompt

	projectPath := agent.ContextBuilder.GetPlanWorkDir()

	if projectPath == "" {
		projectPath = agent.Workspace
	}

	peers := al.sessions.GetPeerPurposes(opts.SessionKey, projectPath)

	if len(peers) > 0 {
		var peerNote strings.Builder

		peerNote.WriteString("Other sessions working on this project:\n")

		for _, p := range peers {
			peerNote.WriteString(fmt.Sprintf("- %s: %s (branch: %s)\n", p.SessionKey, p.Purpose, p.Branch))
		}

		peerNote.WriteString("\nAvoid conflicting changes with these sessions.")

		agent.ContextBuilder.SetPeerNote(peerNote.String())
	} else {
		agent.ContextBuilder.SetPeerNote("")
	}

	// 2. Build messages (skip history for heartbeat)

	var history []providers.Message
	var summary string
	if !opts.NoHistory {
		history = agent.Sessions.GetHistory(opts.SessionKey)
		summary = agent.Sessions.GetSummary(opts.SessionKey)

		// Sanitize history to remove orphaned tool calls (from crashes/session collisions)

		var removedCount int

		history, removedCount = session.SanitizeHistory(history)

		if removedCount > 0 {
			logger.WarnCF("agent", "Sanitized session history: removed orphaned messages",

				map[string]any{
					"session_key": opts.SessionKey,

					"removed_count": removedCount,
				})

			// Persist the sanitized history

			agent.Sessions.SetHistory(opts.SessionKey, history)

			_ = agent.Sessions.Save(opts.SessionKey)
		}
	}
	messages := agent.ContextBuilder.BuildMessages(
		history,
		summary,
		opts.UserMessage,
		opts.Media,
		opts.Channel,
		opts.ChatID,
		opts.SenderID,
		opts.SenderDisplayName,
	)

	// Resolve media:// refs: images→base64 data URLs, non-images→local paths in content
	cfg := al.GetConfig()
	maxMediaSize := cfg.Agents.Defaults.GetMaxMediaSize()
	messages = resolveMediaRefs(messages, al.mediaStore, maxMediaSize)

	// Describe images for text-only main models. In plan pre-execution mode
	// (interviewing/review), the plan model (vision-capable) handles images
	// directly, so skip description generation.
	planStatus := agent.ContextBuilder.GetPlanStatus()
	if len(agent.ImageCandidates) > 0 && !isPlanPreExecution(planStatus) {
		messages = al.describeImagesInMessages(ctx, messages, agent, opts.Channel, opts.ChatID)
	}

	// Process PDFs with OCR when configured
	if ocrCfg := cfg.Agents.Defaults.OCR; ocrCfg != nil && ocrCfg.Command != "" {
		messages = al.processPDFsInMessages(ctx, messages, ocrCfg, opts.Channel, opts.ChatID)
	}

	// 2b. Interview staleness nudge: if MEMORY.md hasn't been updated for

	// several consecutive turns, inject a reminder so the AI writes its findings.

	const interviewStaleThreshold = 2

	if agent.ContextBuilder.GetPlanStatus() == "interviewing" && agent.interviewStaleCount >= interviewStaleThreshold {
		messages = append(messages, providers.Message{
			Role: "user",

			Content: "[System] You have been interviewing for several turns without updating memory/MEMORY.md. Please use edit_file now to save your findings to the ## Context section, or organize the plan into ## Phase sections with `- [ ]` checkbox steps if you have enough information.",
		})
	}

	// 2c. Background plan preamble: append to system prompt (high attention)

	// so the LLM knows from the start that it must mark steps [x].

	// Skip if a chat session is actively working on the plan directory.

	if opts.Background && agent.ContextBuilder.HasActivePlan() && agent.ContextBuilder.GetPlanStatus() == "executing" {
		planDir := agent.ContextBuilder.GetPlanWorkDir()

		skipPreamble := planDir != "" && al.sessions.IsActiveInDir(planDir, "heartbeat")

		if !skipPreamble && len(messages) > 0 && messages[0].Role == "system" {
			var sb strings.Builder

			sb.WriteString(messages[0].Content)

			sb.WriteString("\n\n## Background Execution\n")

			sb.WriteString("You are running as a background heartbeat with no conversation history. ")

			sb.WriteString("MEMORY.md is the only shared state between heartbeats. ")

			sb.WriteString(
				"After completing each plan step, immediately use edit_file to mark it [x] in memory/MEMORY.md.",
			)

			messages[0].Content = sb.String()
		}
	}

	// 2d. Snapshot plan status and MEMORY.md size before LLM iteration.

	preStatus := agent.ContextBuilder.GetPlanStatus()

	var preMemoryLen int

	if preStatus == "interviewing" {
		preMemoryLen = len(agent.ContextBuilder.ReadMemory())
	}

	// 3. Save user message to session (use compact form if available)

	historyMsg := opts.UserMessage

	if opts.HistoryMessage != "" {
		historyMsg = opts.HistoryMessage
	}

	agent.Sessions.AddMessage(opts.SessionKey, "user", historyMsg)

	// 4. Record user prompt for stats

	if al.stats != nil {
		al.stats.RecordPrompt()
	}

	// Capture the finalized system prompt for Mini App inspection

	if len(messages) > 0 {
		al.lastSystemPrompt.Store(messages[0].Content)

		al.promptDirty.Store(false)
	}

	// 5. Run LLM iteration loop (with automatic phase transitions)

	var iteration int

	const maxPhaseTransitions = 10

	for phaseLoop := 0; ; phaseLoop++ {
		// On phase transition: rebuild system prompt with new phase context + nudge

		if phaseLoop > 0 {
			messages = agent.ContextBuilder.BuildMessages(
				agent.Sessions.GetHistory(opts.SessionKey),
				agent.Sessions.GetSummary(opts.SessionKey),
				"", opts.Media, opts.Channel, opts.ChatID,
				opts.SenderID, opts.SenderDisplayName,
			)

			messages = append(messages, providers.Message{
				Role: "user",

				Content: fmt.Sprintf(

					"[System] Phase %d is now active. Continue working on the next steps.",

					agent.ContextBuilder.GetCurrentPhase(),
				),
			})

			if len(messages) > 0 {
				al.lastSystemPrompt.Store(messages[0].Content)
			}
		}

		curPlanStatus := preStatus

		if phaseLoop > 0 {
			curPlanStatus = agent.ContextBuilder.GetPlanStatus()
		}

		var err error

		finalContent, iteration, err = al.runLLMIteration(ctx, agent, messages, opts, task, curPlanStatus)
		if err != nil {
			return "", err
		}

		// 5a. Auto-advance plan phases after LLM iteration

		postStatus := agent.ContextBuilder.GetPlanStatus()

		if !agent.ContextBuilder.HasActivePlan() ||

			!(postStatus == "executing" || postStatus == "review" || postStatus == "completed") {
			break
		}

		// Intercept: if AI changed status to executing or review without user approval

		// (from interviewing or review), validate and hold at "review".

		if preStatus == "interviewing" || (preStatus == "review" && postStatus == "executing") {
			if err := agent.ContextBuilder.ValidatePlanStructure(); err != nil {
				_ = agent.ContextBuilder.SetPlanStatus("interviewing")

				logger.WarnCF("agent", "Reverted plan to interviewing: "+err.Error(),

					map[string]any{"agent_id": agent.ID})

				rejectionMsg := "[System] Plan rejected: " + err.Error() + ". Fix and try again."

				agent.Sessions.AddMessage(opts.SessionKey, "user", rejectionMsg)
			} else {
				_ = agent.ContextBuilder.SetPlanStatus("review")

				al.reporter().ReportStateChange(opts.SessionKey, orch.AgentStatePlanReview, "")

				if !constants.IsInternalChannel(opts.Channel) {
					planDisplay := agent.ContextBuilder.FormatPlanDisplay()

					_ = al.bus.PublishOutbound(ctx, bus.OutboundMessage{
						Channel: opts.Channel,

						ChatID: opts.ChatID,

						Content: planDisplay + "\n\nUse /plan start to approve, or continue chatting to refine.",

						SkipPlaceholder: true,
					})
				}
			}

			break
		}

		if postStatus == "executing" && agent.ContextBuilder.GetTotalPhases() == 0 {
			_ = agent.ContextBuilder.SetPlanStatus("interviewing")

			logger.WarnCF("agent", "Reverted plan to interviewing: no phases defined",

				map[string]any{"agent_id": agent.ID})

			break
		}

		if agent.ContextBuilder.IsPlanComplete() {
			total := agent.ContextBuilder.GetTotalPhases()

			_ = agent.ContextBuilder.SetCurrentPhase(total)

			if preStatus != "completed" {
				_ = agent.ContextBuilder.SetPlanStatus("completed")

				al.reporter().ReportStateChange(opts.SessionKey, orch.AgentStatePlanCompleted, "")

				// Deactivate worktree on plan completion

				commitMsg := "plan: " + agent.ContextBuilder.Memory().GetPlanTaskName()

				wtResult, _ := agent.DeactivateWorktree(opts.SessionKey, commitMsg, false)

				if !constants.IsInternalChannel(opts.Channel) {
					msg := "\u2705 Plan completed!"

					if wtResult != nil && wtResult.CommitsAhead > 0 {
						msg += fmt.Sprintf("\nBranch `%s` retained (%d commits). To merge: `git merge %s`",

							wtResult.Branch, wtResult.CommitsAhead, wtResult.Branch)
					}

					_ = al.bus.PublishOutbound(ctx, bus.OutboundMessage{
						Channel: opts.Channel,

						ChatID: opts.ChatID,

						Content: msg,

						SkipPlaceholder: true,
					})
				}
			}

			break
		}

		if agent.ContextBuilder.IsCurrentPhaseComplete() {
			if phaseLoop >= maxPhaseTransitions {
				logger.WarnCF("agent", "Max phase transitions reached, stopping",

					map[string]any{"agent_id": agent.ID, "transitions": phaseLoop})

				break
			}

			prev := agent.ContextBuilder.GetCurrentPhase()

			_ = agent.ContextBuilder.AdvancePhase()

			next := agent.ContextBuilder.GetCurrentPhase()

			if !constants.IsInternalChannel(opts.Channel) {
				_ = al.bus.PublishOutbound(ctx, bus.OutboundMessage{
					Channel: opts.Channel,

					ChatID: opts.ChatID,

					Content: fmt.Sprintf("Phase %d complete. Moving to Phase %d.", prev, next),

					SkipPlaceholder: true,
				})
			}

			al.notifyStateChange()

			continue
		}

		break
	}

	al.notifyStateChange()

	// 5b. Interview staleness detection: compare MEMORY.md size after iteration.

	if agent.ContextBuilder.GetPlanStatus() == "interviewing" {
		postMemoryLen := len(agent.ContextBuilder.ReadMemory())

		if postMemoryLen == preMemoryLen {
			agent.interviewStaleCount++
		} else {
			agent.interviewStaleCount = 0
		}

		agent.interviewMemoryLen = postMemoryLen
	} else {
		// Reset counter when not interviewing.

		agent.interviewStaleCount = 0
	}

	// 5c. Handle empty response

	if finalContent == "" {
		if iteration >= agent.MaxIterations && agent.MaxIterations > 0 {
			finalContent = toolLimitResponse
		} else {
			finalContent = opts.DefaultResponse
		}
	}

	// 5d. Store result summary for task completion notification
	task.Result = utils.Truncate(finalContent, 280)

	// 6. Save final assistant message to session (deferred write-behind)

	agent.Sessions.AddMessage(opts.SessionKey, "assistant", finalContent)

	agent.Sessions.MarkDirty(opts.SessionKey)

	// 7. Optional: summarization

	if opts.EnableSummary {
		al.maybeSummarize(agent, opts.SessionKey, opts.Channel, opts.ChatID)
	}

	// 8. Optional: send response via bus

	if opts.SendResponse {
		_ = al.bus.PublishOutbound(ctx, bus.OutboundMessage{
			Channel: opts.Channel,

			ChatID: opts.ChatID,

			Content: finalContent,

			SkipPlaceholder: opts.SystemMessage, // suppress Telegram "Thinking..." for system messages

		})
	}

	// 9. Log response

	responsePreview := utils.Truncate(finalContent, 120)
	logger.InfoCF("agent", fmt.Sprintf("Response: %s", responsePreview),
		map[string]any{
			"agent_id": agent.ID,

			"session_key": opts.SessionKey,

			"iterations": iteration,

			"final_length": len(finalContent),
		})

	return finalContent, nil
}

// cleanupHeartbeatWorktree handles worktree cleanup for background tasks.
// Waits for spawned subagents, auto-commits, and attempts fast-forward merge.
func (al *AgentLoop) cleanupHeartbeatWorktree(agent *AgentInstance, opts processOptions) {
	if !opts.Background {
		return
	}

	if agent.SubagentMgr != nil {
		agent.SubagentMgr.WaitAll(35 * time.Minute) // slightly above spawnTimeout
	}

	wt := agent.GetWorktree(opts.SessionKey)

	if wt == nil {
		return
	}

	// 1. Auto-commit uncommitted changes in worktree
	if git.HasUncommittedChanges(wt.Path) {
		_ = git.AutoCommit(wt.Path, "heartbeat: auto-save")
	}

	// 2. Check if there are unique commits worth merging
	repoRoot := git.FindRepoRoot(agent.Workspace)
	ahead := git.CommitsAhead(repoRoot, wt.BaseBranch, wt.Branch)

	if ahead > 0 && repoRoot != "" {
		// 3. Try fast-forward merge into base branch
		mr := git.MergeWorktreeBranch(repoRoot, wt)

		// 4. Notify based on merge result
		if !constants.IsInternalChannel(opts.Channel) {
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)

			if mr.Merged {
				_ = al.bus.PublishOutbound(cleanupCtx, bus.OutboundMessage{
					Channel: opts.Channel,
					ChatID:  opts.ChatID,
					Content: fmt.Sprintf("Heartbeat: merged %d commit(s) to %s.",
						ahead, wt.BaseBranch),
				})
			} else if mr.Conflict {
				_ = al.bus.PublishOutbound(cleanupCtx, bus.OutboundMessage{
					Channel: opts.Channel,
					ChatID:  opts.ChatID,
					Content: fmt.Sprintf("Heartbeat: merge conflict on branch `%s` — manual merge needed.",
						mr.Branch),
				})
			}

			cleanupCancel()
		}
	}

	// 5. Dispose worktree (branch auto-deleted if merged, kept if conflict)
	agent.DeactivateWorktree(opts.SessionKey, "", false)
}

// publishTaskCompletion publishes the final task status on completion for
// background tasks, including the LLM response in the completion bubble.
func (al *AgentLoop) publishTaskCompletion(
	task *activeTask, finalContent *string, opts processOptions, taskKey string,
) {
	al.activeTasks.Delete(taskKey)

	if opts.TaskID == "" {
		return
	}

	elapsed := time.Since(task.StartedAt)

	completionMsg := fmt.Sprintf("\u2705 Task completed (%.1fs)", elapsed.Seconds())

	// Determine the best content to show in the completion bubble.
	// Priority: message tool content > finalContent > task.Result
	task.mu.Lock()
	msgContent := task.messageContent
	task.mu.Unlock()

	var resultContent string

	switch {
	case msgContent != "":
		// The message tool already sent this to the user via the
		// task bubble; re-include it so the completion doesn't erase it.
		resultContent = msgContent

	case *finalContent != "" && *finalContent != defaultResponse && *finalContent != "HEARTBEAT_OK":
		resultContent = *finalContent

	default:
		summary := task.Result
		if summary == "" {
			summary = task.Description
		}
		resultContent = summary
	}

	if resultContent != "" {
		combined := completionMsg + "\n\n" + resultContent

		if len([]rune(combined)) <= 4096 {
			completionMsg = combined
		} else {
			// Too long for one bubble: send header as task status,
			// body as regular message (auto-split by SplitMessage).
			doneCtx, doneCancel := context.WithTimeout(context.Background(), 5*time.Second)

			_ = al.bus.PublishOutbound(doneCtx, bus.OutboundMessage{
				Channel:      opts.Channel,
				ChatID:       opts.ChatID,
				Content:      completionMsg,
				IsTaskStatus: true,
				TaskID:       opts.TaskID,
				Final:        true,
			})

			_ = al.bus.PublishOutbound(doneCtx, bus.OutboundMessage{
				Channel: opts.Channel,
				ChatID:  opts.ChatID,
				Content: resultContent,
			})

			doneCancel()
			return
		}
	}

	doneCtx, doneCancel := context.WithTimeout(context.Background(), 5*time.Second)

	_ = al.bus.PublishOutbound(doneCtx, bus.OutboundMessage{
		Channel:      opts.Channel,
		ChatID:       opts.ChatID,
		Content:      completionMsg,
		IsTaskStatus: true,
		TaskID:       opts.TaskID,
		Final:        true,
	})

	doneCancel()
}

// redirectMessageToolForTask redirects the message tool to publish as IsTaskStatus
// for background tasks that don't send a final response.
func (al *AgentLoop) redirectMessageToolForTask(agent *AgentInstance, task *activeTask, opts processOptions) {
	tool, ok := agent.Tools.Get("message")
	if !ok {
		return
	}
	mt, ok := tool.(*tools.MessageTool)
	if !ok {
		return
	}
	taskID := opts.TaskID

	mt.SetSendCallback(func(channel, chatID, content string) error {
		// Capture the message tool's content so the completion
		// defer can include it instead of losing it to an overwrite.
		if task != nil {
			task.mu.Lock()
			task.messageContent = content
			task.mu.Unlock()
		}

		pubCtx, pubCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer pubCancel()

		return al.bus.PublishOutbound(pubCtx, bus.OutboundMessage{
			Channel:      channel,
			ChatID:       chatID,
			Content:      content,
			IsTaskStatus: true,
			TaskID:       taskID,
		})
	})
}

// selectCandidates resolves the model candidates for a turn. Called once per
// turn (sticky) so the model doesn't change mid-iteration. Plan model hooks
// may further override per-iteration via hooks.SelectModel().
//
// Uses the agent's pre-resolved Candidates/LightCandidates. When a Router is
// configured, it scores the message and may select the light model tier.
func (al *AgentLoop) selectCandidates(
	agent *AgentInstance,
	userMsg string,
	history []providers.Message,
) []providers.FallbackCandidate {
	if agent.Router != nil && len(agent.LightCandidates) > 0 {
		_, isLight, _ := agent.Router.SelectModel(userMsg, history, agent.Model)
		if isLight {
			return agent.LightCandidates
		}
	}
	return agent.Candidates
}

// runLLMIteration executes the LLM call loop with tool handling using hooks.
func (al *AgentLoop) runLLMIteration(
	ctx context.Context,
	agent *AgentInstance,
	messages []providers.Message,
	opts processOptions,
	task *activeTask,
	planSnapshot string,
) (string, int, error) {
	hooks := al.buildHooks(agent, opts, task, planSnapshot)

	// Select candidates once per turn (sticky). Plan model hooks may
	// override per-iteration via hooks.SelectModel().
	turnCandidates := al.selectCandidates(agent, opts.UserMessage, messages)

	iteration := 0
	var finalContent string

	for iteration < agent.MaxIterations {
		iteration++

		if msg := hooks.OnIterationStart(iteration); msg != "" {
			messages = append(messages, providers.Message{Role: "user", Content: msg})
		}

		logger.DebugCF("agent", "LLM iteration",
			map[string]any{
				"agent_id":  agent.ID,
				"iteration": iteration,
				"max":       agent.MaxIterations,
			})

		// Build tool definitions
		providerToolDefs := hooks.FilterTools(agent.Tools.ToProviderDefs())

		// Resolve model and candidates for this call.
		// Default to turn-level candidates; hooks may override per-iteration.
		candidates := turnCandidates
		activeModel := agent.Model
		if m, c := hooks.SelectModel(); m != "" {
			activeModel = m
			candidates = c
		}

		// Log LLM request details
		logger.DebugCF("agent", "LLM request",
			map[string]any{
				"agent_id":          agent.ID,
				"iteration":         iteration,
				"model":             activeModel,
				"messages_count":    len(messages),
				"tools_count":       len(providerToolDefs),
				"max_tokens":        agent.MaxTokens,
				"temperature":       agent.Temperature,
				"system_prompt_len": len(messages[0].Content),
			})
		logger.DebugCF("agent", "Full LLM request",
			map[string]any{
				"iteration":     iteration,
				"messages_json": formatMessagesForLog(messages),
				"tools_json":    formatToolsForLog(providerToolDefs),
			})

		// Streaming setup
		onChunk, streamCleanup := hooks.SetupStreaming()

		hooks.OnPreLLMCall()

		// Call LLM with retry
		response, err := al.callLLMWithRetry(ctx, agent, &messages, opts,
			providerToolDefs, candidates, activeModel, onChunk, iteration)

		// Streaming cleanup
		if streamCleanup != nil {
			onChunk = nil
			streamCleanup()
		}

		if err != nil {
			logger.ErrorCF("agent", "LLM call failed",
				map[string]any{
					"agent_id":  agent.ID,
					"iteration": iteration,
					"model":     activeModel,
					"error":     err.Error(),
				})
			return "", iteration, fmt.Errorf("LLM call failed after retries: %w", err)
		}

		// Record token usage
		if response.Usage != nil && al.stats != nil {
			al.stats.RecordUsage(
				response.Usage.PromptTokens,
				response.Usage.CompletionTokens,
				response.Usage.TotalTokens,
			)
		}

		go al.handleReasoning(ctx, response.Reasoning, opts.Channel, al.targetReasoningChannelID(opts.Channel))

		logger.DebugCF("agent", "LLM response",
			map[string]any{
				"agent_id":       agent.ID,
				"iteration":      iteration,
				"content_chars":  len(response.Content),
				"tool_calls":     len(response.ToolCalls),
				"reasoning":      response.Reasoning,
				"target_channel": al.targetReasoningChannelID(opts.Channel),
				"channel":        opts.Channel,
			})

		// Clean up response content
		response = al.cleanLLMResponse(ctx, response, &messages, agent, iteration,
			providerToolDefs, candidates, activeModel, onChunk)

		// No tool calls — check for plan nudge or return
		if len(response.ToolCalls) == 0 {
			if nudge, cont := hooks.OnNoToolCalls(response.Content, iteration); cont {
				messages = append(messages,
					providers.Message{Role: "assistant", Content: response.Content},
					providers.Message{Role: "user", Content: nudge},
				)
				continue
			}

			finalContent = response.Content
			if finalContent == "" && response.ReasoningContent != "" {
				finalContent = response.ReasoningContent
			}
			logger.InfoCF("agent", "LLM response without tool calls (direct answer)",
				map[string]any{
					"agent_id":      agent.ID,
					"iteration":     iteration,
					"content_chars": len(finalContent),
				})
			break
		}

		// Normalize and filter tool calls
		normalizedToolCalls := make([]providers.ToolCall, 0, len(response.ToolCalls))
		for _, tc := range response.ToolCalls {
			normalizedToolCalls = append(normalizedToolCalls, providers.NormalizeToolCall(tc))
		}

		filtered, rejMsg := hooks.FilterToolCalls(normalizedToolCalls)
		if len(filtered) < len(normalizedToolCalls) && rejMsg != "" {
			messages = append(messages, providers.Message{Role: "user", Content: rejMsg})
		}
		normalizedToolCalls = filtered
		if len(normalizedToolCalls) == 0 {
			continue
		}

		// Log tool calls
		toolNames := make([]string, 0, len(normalizedToolCalls))
		for _, tc := range normalizedToolCalls {
			toolNames = append(toolNames, tc.Name)
		}
		logger.InfoCF("agent", "LLM requested tool calls",
			map[string]any{
				"agent_id":  agent.ID,
				"tools":     toolNames,
				"count":     len(normalizedToolCalls),
				"iteration": iteration,
			})

		hooks.OnToolsProcessed(ctx, iteration, normalizedToolCalls)

		// Build and save assistant message
		assistantMsg := buildAssistantMessage(response, normalizedToolCalls)
		messages = append(messages, assistantMsg)
		agent.Sessions.AddFullMessage(opts.SessionKey, assistantMsg)

		// Execute tool calls and collect results
		lastBlocker := al.executeToolCalls(ctx, agent, normalizedToolCalls, &messages, opts, hooks, iteration)

		// Tick TTL-based tool expiry after execution
		agent.Tools.TickTTL()

		hooks.InjectReminders(iteration, &messages, lastBlocker)
		hooks.RefreshSystemPrompt(messages)
	}

	// Force a final text response if max iterations exhausted
	if finalContent == "" && iteration >= agent.MaxIterations {
		finalContent = al.forceTextResponse(ctx, agent, messages)
	}

	return finalContent, iteration, nil
}

// executeToolCalls runs each tool call sequentially, publishes results,
// and returns the last blocker (error content) for reminder injection.
func (al *AgentLoop) executeToolCalls(
	ctx context.Context,
	agent *AgentInstance,
	toolCalls []providers.ToolCall,
	messages *[]providers.Message,
	opts processOptions,
	hooks iterationHooks,
	iteration int,
) string {
	var lastBlocker string
	for _, tc := range toolCalls {
		argsJSON, _ := json.Marshal(tc.Arguments)
		argsPreview := utils.Truncate(string(argsJSON), 200)
		logger.InfoCF("agent", fmt.Sprintf("Tool call: %s(%s)", tc.Name, argsPreview),
			map[string]any{
				"agent_id":  agent.ID,
				"tool":      tc.Name,
				"iteration": iteration,
			})

		// Heartbeat lazy worktree: create worktree on first write-tool call
		// Always use ai.Workspace (not GetPlanWorkDir) to avoid creating worktrees
		// against stale project paths from previous plans.
		if opts.Background && isWriteTool(tc.Name) && !agent.IsInWorktree(opts.SessionKey) {
			taskName := "heartbeat-" + time.Now().Format("20060102")
			if wt, wtErr := agent.ActivateWorktree(opts.SessionKey, taskName, agent.Workspace); wtErr == nil {
				logger.InfoCF("agent", "Heartbeat worktree created", map[string]any{"branch": wt.Branch})
			}
		}

		asyncCallback := hooks.OnPreToolExec(ctx, tc)

		toolStart := time.Now()
		toolCtx := ctx
		if wt := agent.GetWorktree(opts.SessionKey); wt != nil {
			toolCtx = tools.WithWorkspaceOverride(toolCtx, wt.Path)
			toolCtx = tools.WithWorktreeInfo(toolCtx, wt)
		}

		toolResult := agent.Tools.ExecuteWithContext(
			toolCtx, tc.Name, tc.Arguments,
			opts.Channel, opts.ChatID, asyncCallback,
		)
		toolDuration := time.Since(toolStart)

		hooks.OnToolExecDone(tc, toolResult, toolDuration)

		// Publish results to user
		if !toolResult.Silent && toolResult.ForUser != "" && opts.SendResponse {
			_ = al.bus.PublishOutbound(ctx, bus.OutboundMessage{
				Channel: opts.Channel,
				ChatID:  opts.ChatID,
				Content: toolResult.ForUser,
			})
			logger.DebugCF("agent", "Sent tool result to user",
				map[string]any{"tool": tc.Name, "content_len": len(toolResult.ForUser)})
		}

		if len(toolResult.Media) > 0 && opts.SendResponse {
			al.publishToolMedia(ctx, toolResult, opts)
		}

		// Build tool result message
		contentForLLM := toolResult.ForLLM
		if contentForLLM == "" && toolResult.Err != nil {
			contentForLLM = toolResult.Err.Error()
		}
		if toolResult.IsError || toolResult.Err != nil {
			lastBlocker = contentForLLM
		}

		toolResultMsg := providers.Message{
			Role:       "tool",
			Content:    contentForLLM,
			ToolCallID: tc.ID,
		}
		*messages = append(*messages, toolResultMsg)
		agent.Sessions.AddFullMessage(opts.SessionKey, toolResultMsg)
	}
	return lastBlocker
}

// forceTextResponse makes a final LLM call without tools when max iterations
// are exhausted, forcing a text response.
func (al *AgentLoop) forceTextResponse(ctx context.Context, agent *AgentInstance, messages []providers.Message) string {
	logger.WarnCF("agent", "Max iterations reached, forcing final response without tools",
		map[string]any{"agent_id": agent.ID})

	forceResp, forceErr := agent.Provider.Chat(ctx, messages, nil, agent.Model, map[string]any{
		"max_tokens":       agent.MaxTokens,
		"temperature":      agent.Temperature,
		"prompt_cache_key": agent.ID,
	})
	if forceErr != nil || forceResp.Content == "" {
		return ""
	}
	content := utils.StripThinkBlocks(forceResp.Content)
	if forceResp.Usage != nil && al.stats != nil {
		al.stats.RecordUsage(
			forceResp.Usage.PromptTokens,
			forceResp.Usage.CompletionTokens,
			forceResp.Usage.TotalTokens,
		)
	}
	return content
}

// updateToolContexts updates the context for tools that need channel/chatID info.
func (al *AgentLoop) updateToolContexts(agent *AgentInstance, channel, chatID string) {
	// Use ContextualTool interface instead of type assertions

	if tool, ok := agent.Tools.Get("message"); ok {
		if mt, ok := tool.(tools.ContextualTool); ok {
			mt.SetContext(channel, chatID)
		}
	}

	if tool, ok := agent.Tools.Get("spawn"); ok {
		if st, ok := tool.(tools.ContextualTool); ok {
			st.SetContext(channel, chatID)
		}
	}

	if tool, ok := agent.Tools.Get("subagent"); ok {
		if st, ok := tool.(tools.ContextualTool); ok {
			st.SetContext(channel, chatID)
		}
	}
}
